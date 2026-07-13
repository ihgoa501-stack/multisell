/**
 * Popup script for the LingMirror Sourcing Agent.
 *
 * Displays connection status, provides device-pairing controls, and
 * allows manual triggering of product data extraction on the current tab.
 */

import { getJWT, getServerUrl, setServerUrl, getLoginUrl } from "./shared/auth.js";
import type { CollectionRecoveryUpdate, ContentScriptFetchRequest, ContentScriptFetchResult, StatusResponse } from "./shared/protocol.js";
import { describeCollectionRecovery, type CollectionReconciliationResult } from "./shared/private-collection.js";

// ─── DOM references ────────────────────────────────────────────────────────

const statusDot = document.getElementById("statusDot") as HTMLSpanElement;
const statusLabel = document.getElementById("statusLabel") as HTMLSpanElement;
const fetchBtn = document.getElementById("fetchBtn") as HTMLButtonElement;
const loginBtn = document.getElementById("loginBtn") as HTMLButtonElement;
const resultCard = document.getElementById("resultCard") as HTMLDivElement;
const resultContent = document.getElementById("resultContent") as HTMLPreElement;
const settingsLink = document.getElementById("settingsLink") as HTMLAnchorElement;
const boxBtn = document.getElementById("boxBtn") as HTMLButtonElement;

// ─── State ─────────────────────────────────────────────────────────────────

let currentStatus: "connected" | "disconnected" | "no_token" | "error" = "disconnected";
let isFetching = false;

// ─── Status UI ─────────────────────────────────────────────────────────────

function updateStatus(
  status: "connected" | "disconnected" | "no_token" | "error"
): void {
  currentStatus = status;
  statusDot.className = "status-dot " + status;

  switch (status) {
    case "connected":
      statusLabel.textContent = "已连接";
      fetchBtn.disabled = false;
      loginBtn.textContent = "打开凌镜";
      loginBtn.title = "打开凌镜插件设置页";
      break;
    case "no_token":
      statusLabel.textContent = "未连接";
      fetchBtn.disabled = true;
      loginBtn.textContent = "连接凌镜";
      loginBtn.title = "";
      break;
    case "disconnected":
      statusLabel.textContent = "已断开";
      fetchBtn.disabled = true;
      loginBtn.textContent = "重新连接";
      loginBtn.title = "";
      break;
    case "error":
      statusLabel.textContent = "连接异常";
      fetchBtn.disabled = true;
      loginBtn.textContent = "重新连接";
      loginBtn.title = "";
      break;
  }

  boxBtn.style.display = status === "connected" ? "flex" : "none";
}

function showResult(payload: Record<string, unknown>): void {
  resultCard.classList.add("visible");

  if (payload.status === "saved" && payload.recordId) {
    resultContent.className = "";
    resultContent.textContent =
      `已保存到凌镜私人采集箱\n` +
      `记录编号：#${payload.recordId}\n` +
      `可以关闭插件并继续浏览`;
    return;
  }
  if (payload.status === "not_saved") {
    resultContent.className = "error-text";
    resultContent.textContent = describeCollectionRecovery(payload as unknown as CollectionReconciliationResult);
    return;
  }
  if (payload.status === "reconcile_required") {
    resultContent.className = "error-text";
    resultContent.textContent = describeCollectionRecovery(payload as unknown as CollectionReconciliationResult);
    return;
  }
  if (payload.status === "ok" && payload.data) {
    const data = payload.data as Record<string, unknown>;
    // Show a compact summary
    resultContent.className = "";
    resultContent.textContent =
      `Title:   ${data.title || "N/A"}\n` +
      `Price:   ¥${data.price_1688 ?? "N/A"}\n` +
      `Images:  ${(data.images as any[])?.length || 0}\n` +
      `Specs:   ${(data.spec_variants as any[])?.length || 0}\n` +
      `Seller:  ${data.supplier_name || "N/A"}`;
  } else if (payload.code) {
    // Error
    resultContent.className = "error-text";
    resultContent.textContent = `Error: ${payload.code}\n${payload.message || ""}`;
  } else {
    resultContent.className = "error-text";
    resultContent.textContent = `Unexpected response: ${JSON.stringify(payload)}`;
  }
}

function showFetching(): void {
  resultCard.classList.add("visible");
  resultContent.className = "";
  resultContent.innerHTML = '<span class="spinner"></span> Extracting product data...';
}

// ─── Actions ───────────────────────────────────────────────────────────────

/** Fetch product data from the current active tab. */
async function handleFetch(): Promise<void> {
  if (isFetching) return;
  isFetching = true;
  fetchBtn.disabled = true;
  showFetching();

  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

    if (!tab?.id || !tab.url) {
      showResult({ code: "NO_TAB", message: "Could not determine active tab" } as any);
      return;
    }

    // Verify we're on a 1688 product page
    if (!tab.url.includes("detail.1688.com")) {
      showResult({
        code: "WRONG_PAGE",
        message: "This extension works on 1688 product detail pages.\nOpen a product at detail.1688.com/offer/...",
      } as any);
      return;
    }

    const response = await chrome.tabs.sendMessage(tab.id, { type: "collect_private_product_from_page" });
    const result = response as { type?: string; payload?: Record<string, unknown> };

    if (result.type === "private_collection_result" && result.payload) {
      showResult(result.payload);
    } else {
      showResult({ code: "UNEXPECTED", message: "页面没有返回保存结果，本次不能视为已保存" } as any);
    }
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown error";
    showResult({ code: "POPUP_ERROR", message } as any);
  } finally {
    isFetching = false;
    if (currentStatus === "connected") {
      fetchBtn.disabled = false;
    }
  }
}

/** Open the Owner-confirmed device-pairing page. */
async function handleLogin(): Promise<void> {
  const token = await getJWT();

  if (token && currentStatus === "connected") {
    // Already paired — open the pairing/device page.
    const serverUrl = await getServerUrl();
    const httpUrl = getLoginUrl(serverUrl).replace("/login", "");
    chrome.tabs.create({ url: httpUrl });
    return;
  }

  // The trusted settings page supplies its own origin to the extension bridge.
  const loginUrl = getLoginUrl(await getServerUrl());
  chrome.tabs.create({ url: loginUrl });
}

/** Show settings prompt. */
async function handleSettings(): Promise<void> {
  const serverUrl = await getServerUrl();
  const httpUrl = serverUrl
    .replace(/^ws:\/\//, "http://")
    .replace(/^wss:\/\//, "https://");
  const input = prompt(
    "凌镜服务器地址：",
    httpUrl.replace(/\/ws\/extension$/, "")
  );
  if (input && input.trim()) {
    const wsUrl = input
      .trim()
      .replace(/^http:\/\//, "ws://")
      .replace(/^https:\/\//, "wss://");
    await setServerUrl(wsUrl);
    updateStatus("disconnected");
  }
}

/** Open my sourcing box page. */
async function handleOpenBox(): Promise<void> {
  const serverUrl = await getServerUrl();
  const loginUrl = getLoginUrl(serverUrl);
  try {
    const url = new URL(loginUrl);
    url.pathname = "/sourcing1688";
    chrome.tabs.create({ url: url.toString() });
  } catch {
    const httpUrl = loginUrl.replace("/settings/plugin", "/sourcing1688");
    chrome.tabs.create({ url: httpUrl });
  }
}

// ─── Listen for status updates from background ────────────────────────────

chrome.runtime.onMessage.addListener((message: StatusResponse | CollectionRecoveryUpdate) => {
  if (message.type === "connection_status") {
    updateStatus(message.status);
  }
  if (message.type === "collection_recovery_update") showResult(message.payload);
});

// ─── Initial state check ───────────────────────────────────────────────────

async function init(): Promise<void> {
  // Query background for current status
  chrome.runtime.sendMessage({ type: "get_status" }, (response: StatusResponse) => {
    if (response?.type === "connection_status") {
      updateStatus(response.status);
    }
  });

  // If no JWT stored, show no_token state immediately
  const token = await getJWT();
  if (!token) {
    updateStatus("no_token");
	} else {
	  // getJWT may have restored the device token after the background's first
	  // startup attempt; explicitly resume every uncertain request now.
	  void chrome.runtime.sendMessage({ type: "reconcile_pending_collections" });
  }

  const recovery = (await chrome.storage.local.get(["lastCollectionRecovery"])).lastCollectionRecovery as
    | Record<string, unknown>
    | undefined;
  if (recovery?.status) showResult(recovery);
}

// ─── Wire up event listeners ───────────────────────────────────────────────

fetchBtn.addEventListener("click", handleFetch);
loginBtn.addEventListener("click", handleLogin);
settingsLink.addEventListener("click", handleSettings);
boxBtn.addEventListener("click", handleOpenBox);

// Run init
init();
