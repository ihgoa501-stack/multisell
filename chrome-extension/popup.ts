/**
 * Popup script for the LingMirror Sourcing Agent.
 *
 * Displays connection status, provides login/logout controls, and
 * allows manual triggering of product data extraction on the current tab.
 */

import { getJWT, clearJWT, getServerUrl, setServerUrl, getLoginUrl } from "./shared/auth.js";
import type { ContentScriptFetchRequest, ContentScriptFetchResult, StatusResponse } from "./shared/protocol.js";

// ─── DOM references ────────────────────────────────────────────────────────

const statusDot = document.getElementById("statusDot") as HTMLSpanElement;
const statusLabel = document.getElementById("statusLabel") as HTMLSpanElement;
const fetchBtn = document.getElementById("fetchBtn") as HTMLButtonElement;
const loginBtn = document.getElementById("loginBtn") as HTMLButtonElement;
const resultCard = document.getElementById("resultCard") as HTMLDivElement;
const resultContent = document.getElementById("resultContent") as HTMLPreElement;
const settingsLink = document.getElementById("settingsLink") as HTMLAnchorElement;

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
      statusLabel.textContent = "Connected";
      fetchBtn.disabled = false;
      loginBtn.textContent = "Connected";
      loginBtn.title = "Click to open LingMirror dashboard";
      break;
    case "no_token":
      statusLabel.textContent = "Not logged in";
      fetchBtn.disabled = true;
      loginBtn.textContent = "Login to LingMirror";
      loginBtn.title = "";
      break;
    case "disconnected":
      statusLabel.textContent = "Disconnected";
      fetchBtn.disabled = true;
      loginBtn.textContent = "Login to LingMirror";
      loginBtn.title = "";
      break;
    case "error":
      statusLabel.textContent = "Connection error";
      fetchBtn.disabled = true;
      loginBtn.textContent = "Reconnect";
      loginBtn.title = "";
      break;
  }
}

function showResult(payload: Record<string, unknown>): void {
  resultCard.classList.add("visible");

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

    // Send fetch request to content script
    const fetchReq: ContentScriptFetchRequest = {
      type: "fetch_product_from_page",
      requestId: "popup_" + Date.now(),
    };

    const response = await chrome.tabs.sendMessage(tab.id, fetchReq);
    const result = response as ContentScriptFetchResult;

    if (result.type === "fetch_product_from_page_result") {
      showResult(result.payload as any);
    } else {
      showResult({ code: "UNEXPECTED", message: "Unexpected response from content script" } as any);
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

/** Handle login/logout. */
async function handleLogin(): Promise<void> {
  const token = await getJWT();

  if (token && currentStatus !== "no_token") {
    // Already logged in and connected — open dashboard
    const serverUrl = await getServerUrl();
    const httpUrl = getLoginUrl(serverUrl).replace("/login", "");
    chrome.tabs.create({ url: httpUrl });
    return;
  }

  // Not logged in or token missing — guide user to login
  const serverUrl = await getServerUrl();

  // Ask user for server URL (first-time setup or when not connected)
  if (!token) {
    const input = prompt(
      "Enter your LingMirror server URL:",
      serverUrl.replace(/^ws:\/\//, "http://").replace(/^wss:\/\//, "https://")
    );
    if (input) {
      const wsUrl = input.replace(/^http:\/\//, "ws://").replace(/^https:\/\//, "wss://");
      await setServerUrl(wsUrl);
    }
  }

  // Open login page — user will copy JWT from there
  const loginUrl = getLoginUrl(await getServerUrl());
  chrome.tabs.create({ url: loginUrl }, () => {
    // After opening login, listen for JWT from the page
    // (In production, use chrome.identity.launchWebAuthFlow or
    //  a content script on lingmirror.com to intercept the JWT)
  });
}

/** Show settings prompt. */
async function handleSettings(): Promise<void> {
  const serverUrl = await getServerUrl();
  const httpUrl = serverUrl
    .replace(/^ws:\/\//, "http://")
    .replace(/^wss:\/\//, "https://");
  const input = prompt(
    "LingMirror Server URL:",
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

// ─── Listen for status updates from background ────────────────────────────

chrome.runtime.onMessage.addListener((message: StatusResponse) => {
  if (message.type === "connection_status") {
    updateStatus(message.status);
  }
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
  }
}

// ─── Wire up event listeners ───────────────────────────────────────────────

fetchBtn.addEventListener("click", handleFetch);
loginBtn.addEventListener("click", handleLogin);
settingsLink.addEventListener("click", handleSettings);

// Run init
init();
