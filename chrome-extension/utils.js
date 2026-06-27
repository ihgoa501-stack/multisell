/**
 * 凌镜 AI 选品助手 — Shared Utilities
 *
 * Chrome storage wrappers and UUID generation.
 * Loaded before content.js and popup.js.
 */

// ─── Chrome Storage Wrappers ──────────────────────────────────────────────

/**
 * Retrieve a value from chrome.storage.local.
 * @param {string} key
 * @returns {Promise<any>}
 */
function getStore(key) {
  return chrome.storage.local.get([key]).then(function (result) {
    return result[key] !== undefined ? result[key] : null;
  });
}

/**
 * Persist a value in chrome.storage.local.
 * @param {string} key
 * @param {*} value
 * @returns {Promise<void>}
 */
function setStore(key, value) {
  return chrome.storage.local.set({ [key]: value });
}

/**
 * Remove a key from chrome.storage.local.
 * @param {string} key
 * @returns {Promise<void>}
 */
function removeStore(key) {
  return chrome.storage.local.remove(key);
}

// ─── UUID v4 Generator ───────────────────────────────────────────────────

/**
 * Generate a RFC 4122 v4 UUID.
 * Uses crypto.randomUUID when available, falls back to Math.random.
 * @returns {string}
 */
function generateUUID() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  // Fallback for environments without crypto.randomUUID
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    var r = (Math.random() * 16) | 0;
    var v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
