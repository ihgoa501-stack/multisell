/**
 * Background Service Worker (Manifest V3).
 *
 * Maintains a persistent WebSocket connection to the LingMirror backend,
 * relays fetch_product requests to content scripts, and forwards
 * extraction results back to the server.
 */

import type {
  FetchProductMessage,
  FetchListPageMessage,
  FetchProductResult,
  FetchProductError,
  WSOutgoingMessage,
  ExtensionMessage,
  ContentScriptFetchRequest,
  PopupMessage,
  StatusResponse,
  CollectPrivateProductRequest,
  CollectPrivateProductResponse,
} from "./shared/protocol.js";
import { getApiBaseUrl, getJWT, getLoginUrl, getServerUrl, getWsUrl, setDeviceCredential, setJWT } from "./shared/auth.js";
import {
  addPendingCollection,
  buildPendingCollectionMarker,
  mergeCollectionRecoveryHistory,
	isTrustedPrivateCollectionSource,
  reconcilePrivateCollectionRequest,
  removePendingCollection,
  submitPrivateCaptureFailure,
  submitPrivateCollection,
	DuplicatePrivateCollectionError,
  type CollectionReconciliationResult,
  type PendingCollectionMap,
  type PendingCollectionMarker,
} from "./shared/private-collection.js";

// ─── State ─────────────────────────────────────────────────────────────────

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempt = 0;
let pingInterval: ReturnType<typeof setInterval> | null = null;
let authenticated = false;
let errored = false;
let connectionStatus: "connected" | "disconnected" | "no_token" | "error" = "disconnected";
const PENDING_COLLECTIONS_KEY = "pendingCollectionRequests";
const LEGACY_PENDING_COLLECTION_KEY = "pendingCollectionRequest";
const LAST_COLLECTION_RECOVERY_KEY = "lastCollectionRecovery";
const COLLECTION_RECOVERY_HISTORY_KEY = "collectionRecoveryHistory";
const COLLECTION_RECONCILIATION_ALARM = "reconcile-private-collections";
let pendingStorageMutation = Promise.resolve();
let recoveryStorageMutation = Promise.resolve();
let reconciliationRun: Promise<void> | null = null;

async function readPendingCollections(): Promise<PendingCollectionMap> {
  const stored = await chrome.storage.local.get([PENDING_COLLECTIONS_KEY, LEGACY_PENDING_COLLECTION_KEY]);
  const current = (stored[PENDING_COLLECTIONS_KEY] || {}) as PendingCollectionMap;
  const legacy = stored[LEGACY_PENDING_COLLECTION_KEY] as PendingCollectionMarker | undefined;
  if (!legacy?.requestId || current[legacy.requestId]) return current;
  const migrated = addPendingCollection(current, legacy);
  await chrome.storage.local.set({ [PENDING_COLLECTIONS_KEY]: migrated });
  await chrome.storage.local.remove(LEGACY_PENDING_COLLECTION_KEY);
  return migrated;
}

function mutatePendingCollections(
  mutation: (pending: PendingCollectionMap) => PendingCollectionMap,
): Promise<void> {
  pendingStorageMutation = pendingStorageMutation.catch(() => undefined).then(async () => {
    const pending = await readPendingCollections();
    await chrome.storage.local.set({ [PENDING_COLLECTIONS_KEY]: mutation(pending) });
  });
  return pendingStorageMutation;
}

async function storeRecovery(
  result: CollectionReconciliationResult,
  marker?: PendingCollectionMarker,
): Promise<void> {
  recoveryStorageMutation = recoveryStorageMutation.catch(() => undefined).then(async () => {
    const stored = await chrome.storage.local.get([COLLECTION_RECOVERY_HISTORY_KEY]);
	const prior = Array.isArray(stored[COLLECTION_RECOVERY_HISTORY_KEY])
	  ? stored[COLLECTION_RECOVERY_HISTORY_KEY]
      : [];
	const history = mergeCollectionRecoveryHistory(prior, result, marker, new Date().toISOString());
	const recovered = history[0];
    await chrome.storage.local.set({
      [LAST_COLLECTION_RECOVERY_KEY]: recovered,
	  [COLLECTION_RECOVERY_HISTORY_KEY]: history,
    });
	chrome.runtime.sendMessage({ type: "collection_recovery_update", payload: recovered }).catch(() => {
	  // Popup is normally closed; persisted history remains authoritative.
	});
  });
  return recoveryStorageMutation;
}

async function runPendingCollectionReconciliation(): Promise<void> {
  const token = await getJWT();
  if (!token) return;
  const serverUrl = await getServerUrl();
  const apiOrigin = new URL(getApiBaseUrl(serverUrl)).origin;
  const pending = await readPendingCollections();
  for (const marker of Object.values(pending)) {
    if (marker.apiOrigin !== apiOrigin) continue;
    const result = await reconcilePrivateCollectionRequest(marker.requestId, token, serverUrl);
    await storeRecovery(result, marker);
    if (result.status !== "reconcile_required") {
      await mutatePendingCollections((items) => removePendingCollection(items, marker.requestId));
    }
  }
}

async function keepReconciliationAlarmInSync(): Promise<void> {
  const serverUrl = await getServerUrl();
  const apiOrigin = new URL(getApiBaseUrl(serverUrl)).origin;
  const pending = await readPendingCollections();
  const hasCurrentServerWork = Object.values(pending).some((marker) => marker.apiOrigin === apiOrigin);
  if (hasCurrentServerWork) {
    // MV3 workers can be suspended and extension credentials do not open a
    // WebSocket. A one-shot alarm is therefore the reliable network-recovery
    // trigger. It is recreated only while unresolved markers remain.
    chrome.alarms.create(COLLECTION_RECONCILIATION_ALARM, { delayInMinutes: 1 });
  } else {
    await chrome.alarms.clear(COLLECTION_RECONCILIATION_ALARM);
  }
}

function reconcilePendingCollections(): Promise<void> {
  if (reconciliationRun) return reconciliationRun;
	reconciliationRun = runPendingCollectionReconciliation().finally(async () => {
	  await keepReconciliationAlarmInSync().catch(() => undefined);
    reconciliationRun = null;
  });
  return reconciliationRun;
}

const MAX_RECONNECT_DELAY = 30_000; // 30 seconds
const INITIAL_RECONNECT_DELAY = 1_000; // 1 second
const PING_INTERVAL = 15_000; // 15 seconds

// ─── Connection status broadcast ───────────────────────────────────────────

function setConnectionStatus(
  status: "connected" | "disconnected" | "no_token" | "error"
) {
  connectionStatus = status;
  const msg: StatusResponse = { type: "connection_status", status };
  chrome.runtime.sendMessage(msg).catch(() => {
    // No listeners (popup closed) — that's fine
  });
}

// ─── WebSocket management ─────────────────────────────────────────────────

async function connect(): Promise<void> {
  // Avoid duplicate connections
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }

  // Reset auth state for fresh connection
  authenticated = false;
  errored = false;

  const token = await getJWT();
  if (!token) {
    setConnectionStatus("no_token");
    return;
  }
	try {
		const encoded = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
		const payload = JSON.parse(atob(encoded));
		if (payload?.type === "extension_access") {
			setConnectionStatus("connected");
			return;
		}
	} catch { /* the HTTPS API remains the authority for token validation */ }

  const serverUrl = await getServerUrl();
  const wsUrl = getWsUrl(serverUrl, token);

  try {
    ws = new WebSocket(wsUrl);
  } catch (err) {
    console.error("[LingMirror] WebSocket creation failed:", err);
    setConnectionStatus("error");
    scheduleReconnect();
    return;
  }

  ws.onopen = () => {
    console.log("[LingMirror] WebSocket connected, sending auth...");
    reconnectAttempt = 0;
    ws?.send(JSON.stringify({ type: "auth", token }));
  };

  ws.onmessage = (event: MessageEvent) => {
    try {
      const msg = JSON.parse(event.data);

      // Handle auth response first — before any other message processing
      if (msg.type === "auth") {
        if (msg.data === "ok") {
          console.log("[LingMirror] WebSocket authenticated");
          authenticated = true;
          setConnectionStatus("connected");
          startPing();
        } else {
          console.error(
            "[LingMirror] WebSocket auth failed:",
            msg.message || "invalid token"
          );
          setConnectionStatus("error");
          ws?.close();
        }
        return;
      }

      // Ignore messages received before authentication
      if (!authenticated) return;

      switch (msg.type) {
        case "pong":
          // Heartbeat acknowledged; nothing else to do
          break;

        case "fetch_product":
          handleFetchProduct(msg as FetchProductMessage);
          break;

        case "fetch_list_page":
          handleFetchListPage(msg as FetchListPageMessage);
          break;

        default:
          console.warn("[LingMirror] Unknown WS message type:", msg.type);
      }
    } catch (err) {
      console.error("[LingMirror] Failed to parse WS message:", err);
    }
  };

  ws.onclose = (event: CloseEvent) => {
    console.log("[LingMirror] WebSocket closed:", event.code, event.reason);
    if (authenticated && !errored) {
      setConnectionStatus("disconnected");
    }
    authenticated = false;
    errored = false;
    cleanup();
    scheduleReconnect();
  };

  ws.onerror = () => {
    console.error("[LingMirror] WebSocket error");
    errored = true;
    setConnectionStatus("error");
  };
}

function cleanup(): void {
  if (pingInterval) {
    clearInterval(pingInterval);
    pingInterval = null;
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  ws = null;
}

function startPing(): void {
  if (pingInterval) {
    clearInterval(pingInterval);
  }
  pingInterval = setInterval(() => {
    if (ws?.readyState === WebSocket.OPEN && authenticated) {
      ws.send(JSON.stringify({ type: "ping" }));
    }
  }, PING_INTERVAL);
}

function scheduleReconnect(): void {
  if (reconnectTimer) return; // Already scheduled
  const delay = Math.min(
    INITIAL_RECONNECT_DELAY * Math.pow(2, reconnectAttempt),
    MAX_RECONNECT_DELAY
  );
  reconnectAttempt++;
  console.log(`[LingMirror] Reconnecting in ${delay}ms (attempt #${reconnectAttempt})`);
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    void connect().then(reconcilePendingCollections);
  }, delay);
}

// ─── Message routing ───────────────────────────────────────────────────────

/**
 * Handle a fetch_product request from the server.
 * Finds the tab with the requested URL and asks its content script to extract data.
 */
async function handleFetchProduct(msg: FetchProductMessage): Promise<void> {
  const url = msg.payload.url;

  try {
    const tabs = await chrome.tabs.query({ url });
    const targetTab = tabs[0]; // Use first matching tab

    if (!targetTab?.id) {
      // Tab not open — try all tabs by URL partial match
      const allTabs = await chrome.tabs.query({});
      const fallbackTab = allTabs.find(
        (t) => t.url && (t.url === url || t.url.includes(url))
      );

      if (!fallbackTab?.id) {
        sendToServer({
          type: "fetch_product_error",
          id: msg.id,
          payload: {
            code: "TAB_NOT_FOUND",
            message: `No open tab found for URL: ${url}`,
          },
        } satisfies FetchProductError);
        return;
      }

      // Send to content script in found tab
      const fetchReq: ContentScriptFetchRequest = {
        type: "fetch_product_from_page",
        requestId: msg.id,
      };
      const response = await chrome.tabs.sendMessage(fallbackTab.id, fetchReq);
      forwardResult(msg.id, response);
      return;
    }

    // Send fetch request to the content script
    const fetchReq: ContentScriptFetchRequest = {
      type: "fetch_product_from_page",
      requestId: msg.id,
    };
    const response = await chrome.tabs.sendMessage(targetTab.id, fetchReq);
    forwardResult(msg.id, response);
  } catch (err) {
    sendToServer({
      type: "fetch_product_error",
      id: msg.id,
      payload: {
        code: "CONTENT_SCRIPT_ERROR",
        message:
          err instanceof Error
            ? err.message
            : "Failed to communicate with content script",
      },
    } satisfies FetchProductError);
  }
}

async function waitForTabComplete(tabId: number, timeoutMs = 30_000): Promise<void> {
  const current = await chrome.tabs.get(tabId);
  if (current.status === "complete") return;

  await new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => {
      chrome.tabs.onUpdated.removeListener(listener);
      reject(new Error("Timed out waiting for list page to load"));
    }, timeoutMs);
    const listener = (updatedTabId: number, changeInfo: chrome.tabs.TabChangeInfo) => {
      if (updatedTabId !== tabId || changeInfo.status !== "complete") return;
      clearTimeout(timer);
      chrome.tabs.onUpdated.removeListener(listener);
      resolve();
    };
    chrome.tabs.onUpdated.addListener(listener);
  });
}

/** Open a marketplace search page in the background and extract its cards. */
async function handleFetchListPage(msg: FetchListPageMessage): Promise<void> {
  let createdTabId: number | undefined;
  try {
    const existing = await chrome.tabs.query({ url: msg.payload.url });
    let tab = existing[0];
    if (!tab?.id) {
      tab = await chrome.tabs.create({ url: msg.payload.url, active: false });
      createdTabId = tab.id;
    }
    if (!tab.id) throw new Error("Browser did not create a list page tab");
    await waitForTabComplete(tab.id);
    const response = await chrome.tabs.sendMessage(tab.id, { type: "fetch_list_page" });
    const items = Array.isArray(response?.data) ? response.data : [];
    sendToServer({
      type: "list_page_result",
      id: msg.id,
      payload: {
        status: "ok",
        data: { page_url: msg.payload.url, collected_at: new Date().toISOString(), items },
      },
    });
  } catch (err) {
    sendToServer({
      type: "list_page_result",
      id: msg.id,
      payload: {
        status: "error",
        data: { page_url: msg.payload.url, collected_at: new Date().toISOString(), items: [] },
        error: {
          code: "LIST_COLLECTION_FAILED",
          message: err instanceof Error ? err.message : "Unknown browser collection error",
        },
      },
    });
  } finally {
    if (createdTabId !== undefined) chrome.tabs.remove(createdTabId).catch(() => {});
  }
}

/**
 * Forward a content script result to the server.
 */
function forwardResult(
  requestId: string,
  response: ExtensionMessage
): void {
  if (response.type !== "fetch_product_from_page_result") return;

  if ("status" in response.payload && response.payload.status === "ok") {
    sendToServer({
      type: "fetch_product_result",
      id: requestId,
      payload: { status: "ok", data: (response.payload as { status: "ok"; data: any }).data },
    } satisfies FetchProductResult);
  } else {
    sendToServer({
      type: "fetch_product_error",
      id: requestId,
      payload: {
        code: (response.payload as { code: string }).code || "EXTRACTION_FAILED",
        message: (response.payload as { message: string }).message || "Extraction failed",
      },
    } satisfies FetchProductError);
  }
}

/**
 * Send a message to the backend WebSocket server.
 */
function sendToServer(msg: WSOutgoingMessage): void {
  if (ws?.readyState === WebSocket.OPEN && authenticated) {
    ws.send(JSON.stringify(msg));
  }
}

// ─── Internal message handlers (content script & popup) ────────────────────

/**
 * Handle messages from:
 * 1. Content scripts (auto-extracted product data)
 * 2. Popup (status queries)
 */
chrome.runtime.onMessage.addListener(
  (
    message: ExtensionMessage | PopupMessage,
    sender: chrome.runtime.MessageSender,
    sendResponse: (response?: any) => void
  ) => {
    if (sender.id !== chrome.runtime.id) {
      sendResponse({ ok: false, error: "消息来源不是当前凌镜插件" });
      return;
    }
    if ((message as any).type === "begin_extension_pairing") {
      void (async () => {
        const input = message as any;
		if (sender.id !== chrome.runtime.id || !sender.url) throw new Error("无效的配对消息来源");
		const senderURL = new URL(sender.url);
		if (senderURL.pathname !== "/settings/plugin") throw new Error("配对只能从凌镜插件设置页发起");
        const deviceStored = await chrome.storage.local.get(["lingmirror_device_id"]);
        const deviceId = deviceStored.lingmirror_device_id || crypto.randomUUID();
        await chrome.storage.local.set({ lingmirror_device_id: deviceId });
        const claimSecret = crypto.randomUUID() + crypto.randomUUID();
        const serverUrl = await getServerUrl();
		const apiOrigin = new URL(getApiBaseUrl(serverUrl)).origin;
		if (senderURL.origin !== new URL(getLoginUrl(serverUrl)).origin) throw new Error("配对页面与目标凌镜服务器不一致");
        const response = await fetch(`${getApiBaseUrl(serverUrl)}/auth/extension-pairings/claim`, {
          method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({
            nonce: input.nonce, claim_secret: claimSecret, device_id: deviceId,
            extension_id: chrome.runtime.id, environment: input.environment,
            browser_label: `${navigator.userAgent.includes("Edg/") ? "Edge" : "Chrome"} · ${navigator.platform || "browser"}`,
          }),
        });
        const body = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(body?.message || "浏览器配对声明失败");
		await chrome.storage.session.set({ pendingExtensionPairing: { nonce: input.nonce, claimSecret, environment: input.environment, apiOrigin, senderOrigin: senderURL.origin } });
        sendResponse({ ok: true, deviceId, extensionId: chrome.runtime.id });
      })().catch((err) => sendResponse({ ok: false, error: err instanceof Error ? err.message : String(err) }));
      return true;
    }

    if ((message as any).type === "finish_extension_pairing") {
      void (async () => {
		const pending = (await chrome.storage.session.get(["pendingExtensionPairing"])).pendingExtensionPairing;
		if (!pending || pending.nonce !== (message as any).nonce) throw new Error("配对会话已失效，请重新开始");
		if (sender.id !== chrome.runtime.id || !sender.url || new URL(sender.url).origin !== pending.senderOrigin) throw new Error("确认消息不是来自原配对页面");
        const serverUrl = await getServerUrl();
		if (new URL(getApiBaseUrl(serverUrl)).origin !== pending.apiOrigin) throw new Error("目标服务器已变化，请重新配对");
        const response = await fetch(`${getApiBaseUrl(serverUrl)}/auth/extension-pairings/exchange`, {
          method: "POST", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ nonce: pending.nonce, claim_secret: pending.claimSecret }),
        });
        const body = await response.json().catch(() => ({})); const data = body?.data;
        if (!response.ok || !data?.access_token || !data?.device_secret) throw new Error(body?.message || "配对凭证签发失败");
        await setJWT(data.access_token);
		await setDeviceCredential({ deviceId: data.device_id, deviceSecret: data.device_secret, environment: pending.environment, apiOrigin: pending.apiOrigin });
        await chrome.storage.session.remove("pendingExtensionPairing");
		await connect();
		void reconcilePendingCollections();
		sendResponse({ ok: true, deviceId: data.device_id });
      })().catch((err) => sendResponse({ ok: false, error: err instanceof Error ? err.message : String(err) }));
      return true;
    }

    if ((message as any).type === "record_private_capture_failure") {
	  if (!sender.tab?.url?.startsWith("https://detail.1688.com/offer/")) {
		sendResponse({ ok: false, error: "采集失败记录只能来自当前1688商品页" });
		return;
	  }
      void (async () => {
        const token = await getJWT();
        if (!token) throw new Error("插件尚未连接凌镜");
        const serverUrl = await getServerUrl();
        await submitPrivateCaptureFailure((message as any).failure, token, serverUrl);
        sendResponse({ ok: true });
      })().catch((err) => sendResponse({ ok: false, error: err instanceof Error ? err.message : String(err) }));
      return true;
    }

	if (message.type === "open_private_collection") {
		void getServerUrl().then((serverUrl) => {
			const appURL = new URL("/sourcing1688", getApiBaseUrl(serverUrl));
			appURL.searchParams.set("record_id", String(message.recordId));
			return chrome.tabs.create({ url: appURL.toString() });
		}).then(() => sendResponse({ ok: true })).catch((err) => sendResponse({ ok: false, error: String(err) }));
		return true;
	}

	if (message.type === "collect_private_product") {
      const collect = message as CollectPrivateProductRequest;
	  const senderURL = sender.tab?.url || "";
	  if (!isTrustedPrivateCollectionSource(senderURL, collect.pageData) || !collect.requestId.startsWith("collect_")) {
		sendResponse({ type: "private_collection_result", requestId: collect.requestId, payload: {
		  status: "not_saved", code: "INVALID_MESSAGE_SOURCE", message: "当前标签页与采集商品不一致，本次没有保存", saved: false,
		} } satisfies CollectPrivateProductResponse);
		return;
	  }
      void (async () => {
        const token = await getJWT();
        if (!token) {
          sendResponse({
            type: "private_collection_result",
            requestId: collect.requestId,
            payload: { status: "failed", code: "AUTH_REQUIRED", message: "请先登录凌镜并连接插件；当前商品未保存", saved: false },
          } satisfies CollectPrivateProductResponse);
          return;
        }
        let marker: PendingCollectionMarker | undefined;
        try {
		  const serverUrl = await getServerUrl();
		  const markerForRequest = buildPendingCollectionMarker(
			collect.requestId,
			serverUrl,
			new Date().toISOString(),
			{ tabId: sender.tab?.id, sourceUrl: collect.pageData.source_url },
		  );
		  marker = markerForRequest;
		  await mutatePendingCollections((items) => addPendingCollection(items, markerForRequest));
          const saved = await submitPrivateCollection(
            collect.pageData,
            collect.requestId,
            chrome.runtime.getManifest().version,
            token,
            serverUrl,
			fetch,
			collect.observationIntent,
          );
		  // A stale marker is harmless (startup reconciliation is idempotent), so
		  // storage cleanup must never downgrade a server-confirmed save.
		  await mutatePendingCollections((items) => removePendingCollection(items, collect.requestId)).catch(() => undefined);
		  sendResponse({
            type: "private_collection_result",
            requestId: collect.requestId,
            payload: {
              status: "saved",
              recordId: saved.recordId,
              snapshotId: saved.snapshotId,
              idempotentReplay: saved.idempotentReplay,
              newObservation: saved.newObservation,
            },
		  } satisfies CollectPrivateProductResponse);
        } catch (err) {
		  if (err instanceof DuplicatePrivateCollectionError) {
			await mutatePendingCollections((items) => removePendingCollection(items, collect.requestId)).catch(() => undefined);
			sendResponse({ type: "private_collection_result", requestId: collect.requestId, payload: {
			  status: "duplicate_requires_choice", recordId: err.recordId, snapshotId: err.snapshotId,
			  message: "该商品已有记录，请明确选择查看已有记录或保存为新观察", saved: false, existing: err.existing,
			} } satisfies CollectPrivateProductResponse);
			return;
		  }
		  const serverUrl = await getServerUrl();
		  const reconciled = await reconcilePrivateCollectionRequest(collect.requestId, token, serverUrl);
		  await storeRecovery(reconciled, marker).catch(() => undefined);
		  if (reconciled.status === "saved") {
			await mutatePendingCollections((items) => removePendingCollection(items, collect.requestId));
			sendResponse({ type: "private_collection_result", requestId: collect.requestId, payload: {
			  status: "saved", recordId: reconciled.recordId, snapshotId: reconciled.snapshotId,
			  idempotentReplay: true, newObservation: false,
			} } satisfies CollectPrivateProductResponse);
			return;
		  }
		  const message = err instanceof Error ? err.message : "凌镜保存失败";
		  if (reconciled.status === "not_saved") {
			await mutatePendingCollections((items) => removePendingCollection(items, collect.requestId));
			sendResponse({ type: "private_collection_result", requestId: collect.requestId,
			  payload: { status: "not_saved", code: "NOT_SAVED", message: `${message}；服务器确认本次未保存，可以修正后重新采集`, saved: false } } satisfies CollectPrivateProductResponse);
			return;
		  }
          sendResponse({
            type: "private_collection_result",
            requestId: collect.requestId,
			payload: { status: "reconcile_required", code: "RECONCILE_REQUIRED", message: `${message}；结果待确认，请勿重复点击`, saved: false },
          } satisfies CollectPrivateProductResponse);
        }
      })().catch((err) => {
        sendResponse({
          type: "private_collection_result",
          requestId: collect.requestId,
		  payload: { status: "reconcile_required", code: "RECONCILE_REQUIRED", message: `${String(err)}；结果待确认，请勿重复点击`, saved: false },
        } satisfies CollectPrivateProductResponse);
      });
      return true;
    }

    // List page auto-extraction result from content script
    if (message.type === "list_page_result") {
      sendToServer(message as any);
      return;
    }

    // Popup status query
	if (message.type === "reconcile_pending_collections") {
	  void reconcilePendingCollections();
	  sendResponse({ ok: true });
	  return;
	}

    if (message.type === "get_status") {
	  // Opening the popup is an explicit opportunity to resume requests that
	  // could not be reconciled while the browser was offline or unpaired.
	  void reconcilePendingCollections();
      sendResponse({ type: "connection_status", status: connectionStatus });
      return;
    }

    // Auto-extraction result from content script
    if (message.type === "fetch_product_from_page_result") {
      if ("status" in message.payload && message.payload.status === "ok") {
        // Forward to server as anonymous product data
        sendToServer({
          type: "fetch_product_result",
          id: message.requestId,
          payload: {
            status: "ok",
            data: (message.payload as { status: "ok"; data: any }).data,
          },
        } satisfies FetchProductResult);
      }
      return;
    }

  }
);

// ─── Initialization ───────────────────────────────────────────────────────

// Connect when service worker starts
void connect().then(reconcilePendingCollections);

// Reconnect on browser startup
chrome.runtime.onStartup.addListener(() => { void connect().then(reconcilePendingCollections); });

// Reconnect on extension install/update
chrome.runtime.onInstalled.addListener(() => { void connect().then(reconcilePendingCollections); });

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === COLLECTION_RECONCILIATION_ALARM) void reconcilePendingCollections();
});
