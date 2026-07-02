/**
 * Background Service Worker (Manifest V3).
 *
 * Maintains a persistent WebSocket connection to the LingMirror backend,
 * relays fetch_product requests to content scripts, and forwards
 * extraction results back to the server.
 */

import type {
  FetchProductMessage,
  FetchProductResult,
  FetchProductError,
  WSOutgoingMessage,
  ExtensionMessage,
  ContentScriptFetchRequest,
  PopupMessage,
  StatusResponse,
} from "./shared/protocol.js";
import { getJWT, getServerUrl, getWsUrl } from "./shared/auth.js";

// ─── State ─────────────────────────────────────────────────────────────────

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempt = 0;
let pingInterval: ReturnType<typeof setInterval> | null = null;
let authenticated = false;
let errored = false;
let connectionStatus: "connected" | "disconnected" | "no_token" | "error" = "disconnected";

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
    connect();
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
    _sender: chrome.runtime.MessageSender,
    sendResponse: (response?: any) => void
  ) => {
    // List page auto-extraction result from content script
    if (message.type === "list_page_result") {
      sendToServer(message as any);
      return;
    }

    // Popup status query
    if (message.type === "get_status") {
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
connect();

// Reconnect on browser startup
chrome.runtime.onStartup.addListener(() => connect());

// Reconnect on extension install/update
chrome.runtime.onInstalled.addListener(() => connect());
