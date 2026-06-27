/**
 * 凌镜 AI 选品助手 — Background Service Worker (MV3)
 *
 * Maintains a WebSocket connection to the LingMirror backend.
 * Relays fetch_product commands to content scripts and forwards
 * extraction results back to the server.
 */

// ─── Configuration Constants ──────────────────────────────────────────────

var WS_RECONNECT_MAX_RETRIES = 5;
var WS_RECONNECT_BASE_DELAY = 1000; // 1 second
var WS_HEARTBEAT_INTERVAL = 15000;  // 15 seconds
var WS_URL_KEY = 'lingmirror_ws_url';
var JWT_KEY = 'lingmirror_jwt';

// ─── State ────────────────────────────────────────────────────────────────

/** @type {WebSocket|null} */
var ws = null;

/** @type {number} */
var reconnectAttempt = 0;

/** @type {number|null} */
var reconnectTimer = null;

/** @type {number|null} */
var heartbeatTimer = null;

/** @type {boolean} */
var isConnected = false;

/** @type {boolean} */
var isDestroyed = false; // Set to true on service worker shutdown

// ─── Connection Status Broadcasting ───────────────────────────────────────

/**
 * Notify popup and other listeners about connection status changes.
 * @param {'connected'|'disconnected'|'no_token'|'error'} status
 */
function broadcastStatus(status) {
  chrome.runtime.sendMessage({ type: 'connection_status', status: status }).catch(function () {
    // No listeners (popup closed) — ok
  });
}

// ─── WebSocket Core ───────────────────────────────────────────────────────

/**
 * Calculate the reconnect delay with exponential backoff.
 * Caps at ~30 seconds.
 * @param {number} attempt - Current retry attempt (0-based)
 * @returns {number} Delay in milliseconds
 */
function getReconnectDelay(attempt) {
  var delay = WS_RECONNECT_BASE_DELAY * Math.pow(2, attempt);
  return Math.min(delay, 30000);
}

/**
 * Connect to the LingMirror backend WebSocket.
 * Reads JWT token and server URL from chrome.storage.
 */
function connect() {
  if (isDestroyed) return;

  // Avoid duplicate connections
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }

  chrome.storage.local.get([JWT_KEY, WS_URL_KEY], function (result) {
    var token = result[JWT_KEY];
    var serverUrl = result[WS_URL_KEY] || 'ws://localhost:8080';

    if (!token) {
      broadcastStatus('no_token');
      return;
    }

    // Build WebSocket URL with token for initial auth
    var wsUrl = serverUrl + '/ws/extension';
    try {
      ws = new WebSocket(wsUrl);
    } catch (err) {
      console.error('[凌镜] WebSocket creation failed:', err);
      broadcastStatus('error');
      scheduleReconnect();
      return;
    }

    ws.onopen = function () {
      console.log('[凌镜] WebSocket connected');

      // Send auth message with token
      ws.send(JSON.stringify({
        type: 'auth',
        token: token
      }));

      isConnected = true;
      reconnectAttempt = 0;
      broadcastStatus('connected');

      // Start heartbeat
      if (heartbeatTimer) clearInterval(heartbeatTimer);
      heartbeatTimer = setInterval(function () {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }));
        }
      }, WS_HEARTBEAT_INTERVAL);
    };

    ws.onmessage = function (event) {
      try {
        var msg = JSON.parse(event.data);

        switch (msg.type) {
          case 'pong':
            // Heartbeat acknowledged
            break;

          case 'fetch_product':
            handleFetchProduct(msg);
            break;

          case 'auth_ok':
            // Auth acknowledged by server
            console.log('[凌镜] Auth OK');
            break;

          case 'auth_error':
            console.error('[凌镜] Auth error:', msg.message);
            broadcastStatus('error');
            break;

          default:
            console.warn('[凌镜] Unknown WS message type:', msg.type);
        }
      } catch (err) {
        console.error('[凌镜] Failed to parse WS message:', err);
      }
    };

    ws.onclose = function (event) {
      console.log('[凌镜] WebSocket closed:', event.code, event.reason);
      isConnected = false;
      broadcastStatus('disconnected');
      cleanup();
      if (!isDestroyed) scheduleReconnect();
    };

    ws.onerror = function () {
      console.error('[凌镜] WebSocket error');
      broadcastStatus('error');
    };
  });
}

/**
 * Clean up timers and WebSocket reference.
 */
function cleanup() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer);
    heartbeatTimer = null;
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  ws = null;
}

/**
 * Schedule a reconnection attempt with exponential backoff.
 * Stops after max retries.
 */
function scheduleReconnect() {
  if (reconnectTimer) return; // Already scheduled

  if (reconnectAttempt >= WS_RECONNECT_MAX_RETRIES) {
    console.error('[凌镜] Max reconnection attempts reached (' + WS_RECONNECT_MAX_RETRIES + ')');
    broadcastStatus('error');
    return;
  }

  var delay = getReconnectDelay(reconnectAttempt);
  reconnectAttempt++;
  console.log('[凌镜] Reconnecting in ' + delay + 'ms (attempt #' + reconnectAttempt + ')');

  reconnectTimer = setTimeout(function () {
    reconnectTimer = null;
    connect();
  }, delay);
}

// ─── Message Routing ──────────────────────────────────────────────────────

/**
 * Handle a fetch_product request from the backend.
 * Finds the tab with the target URL and asks its content script to extract data.
 * @param {Object} msg - { type: "fetch_product", id: "req_uuid", payload: { url: "..." } }
 */
function handleFetchProduct(msg) {
  var url = msg.payload.url;

  chrome.tabs.query({ url: url }, function (tabs) {
    var targetTab = tabs && tabs[0];

    if (!targetTab || !targetTab.id) {
      // Tab not found by exact URL — try partial match
      chrome.tabs.query({}, function (allTabs) {
        var fallbackTab = null;
        for (var i = 0; i < allTabs.length; i++) {
          if (allTabs[i].url && (allTabs[i].url === url || allTabs[i].url.indexOf(url) !== -1)) {
            fallbackTab = allTabs[i];
            break;
          }
        }

        if (!fallbackTab || !fallbackTab.id) {
          sendToServer({
            type: 'fetch_product_error',
            id: msg.id,
            payload: {
              code: 'TAB_NOT_FOUND',
              message: 'No open tab found for URL: ' + url
            }
          });
          return;
        }

        // Send fetch request to content script in found tab
        chrome.tabs.sendMessage(fallbackTab.id, {
          type: 'fetch_product_from_page',
          requestId: msg.id
        }, function (response) {
          forwardResult(msg.id, response);
        });
      });
      return;
    }

    // Send fetch request to the content script
    chrome.tabs.sendMessage(targetTab.id, {
      type: 'fetch_product_from_page',
      requestId: msg.id
    }, function (response) {
      forwardResult(msg.id, response);
    });
  });
}

/**
 * Forward a content script result to the server.
 * @param {string} requestId
 * @param {Object} response - The response from the content script
 */
function forwardResult(requestId, response) {
  if (!response || response.type !== 'fetch_product_from_page_result') return;

  if (response.payload && response.payload.status === 'ok') {
    sendToServer({
      type: 'fetch_product_result',
      id: requestId,
      payload: { status: 'ok', data: response.payload.data }
    });
  } else if (response.payload) {
    sendToServer({
      type: 'fetch_product_error',
      id: requestId,
      payload: {
        code: response.payload.code || 'EXTRACTION_FAILED',
        message: response.payload.message || 'Extraction failed'
      }
    });
  }
}

/**
 * Send a JSON message to the backend via WebSocket.
 * @param {Object} msg
 */
function sendToServer(msg) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(msg));
  }
}

// ─── Internal Message Handlers ────────────────────────────────────────────

/**
 * Handle messages from content scripts and popup.
 */
chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
  // Popup status query
  if (message.type === 'get_status') {
    var status = isConnected ? 'connected' : (ws ? 'disconnected' : 'no_token');
    sendResponse({ type: 'connection_status', status: status });
    return;
  }

  // Content script auto-extraction result
  if (message.type === 'fetch_product_from_page_result') {
    if (message.payload && message.payload.status === 'ok') {
      sendToServer({
        type: 'fetch_product_result',
        id: message.requestId,
        payload: { status: 'ok', data: message.payload.data }
      });
    }
    return;
  }

  // Popup: update server URL
  if (message.type === 'set_server_url') {
    chrome.storage.local.set({ lingmirror_ws_url: message.url }, function () {
      // Disconnect and reconnect with new URL
      if (ws) {
        ws.close();
        cleanup();
      }
      connect();
    });
    return;
  }

  // Popup: set JWT token
  if (message.type === 'set_token') {
    chrome.storage.local.set({ [JWT_KEY]: message.token }, function () {
      if (ws) {
        ws.close();
        cleanup();
      }
      connect();
    });
    return;
  }

  // Popup: clear JWT (logout)
  if (message.type === 'clear_token') {
    chrome.storage.local.remove(JWT_KEY, function () {
      if (ws) {
        ws.close();
        cleanup();
      }
      broadcastStatus('no_token');
    });
    return;
  }
});

// ─── Service Worker Lifecycle ─────────────────────────────────────────────

// Connect when service worker starts
connect();

// Reconnect on browser startup
chrome.runtime.onStartup.addListener(function () {
  isDestroyed = false;
  connect();
});

// Reconnect on extension install/update
chrome.runtime.onInstalled.addListener(function () {
  isDestroyed = false;
  connect();
});
