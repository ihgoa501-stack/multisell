// JWT token storage and retrieval for LingMirror authentication.

const JWT_KEY = "lingmirror_jwt";
const SERVER_URL_KEY = "lingmirror_server_url";

/** Retrieve stored JWT token from chrome.storage.local. */
export async function getJWT(): Promise<string | null> {
  const result = await chrome.storage.local.get([JWT_KEY]);
  return result[JWT_KEY] || null;
}

/** Persist JWT token to chrome.storage.local. */
export async function setJWT(token: string): Promise<void> {
  await chrome.storage.local.set({ [JWT_KEY]: token });
}

/** Remove JWT token from chrome.storage.local (logout). */
export async function clearJWT(): Promise<void> {
  await chrome.storage.local.remove(JWT_KEY);
}

/** Get the configured server WebSocket URL. */
export async function getServerUrl(): Promise<string> {
  const result = await chrome.storage.local.get([SERVER_URL_KEY]);
  return result[SERVER_URL_KEY] || "ws://localhost:8080";
}

/** Persist the server WebSocket URL. */
export async function setServerUrl(url: string): Promise<void> {
  await chrome.storage.local.set({ [SERVER_URL_KEY]: url });
}

/** Build the HTTP login URL from a WebSocket server URL. */
export function getLoginUrl(serverUrl: string): string {
  const baseUrl = serverUrl
    .replace(/^ws:\/\//, "http://")
    .replace(/^wss:\/\//, "https://");
  return `${baseUrl}/login`;
}

/** Build the WebSocket endpoint URL including JWT token. */
export function getWsUrl(serverUrl: string, token: string): string {
  const wsBase = serverUrl.endsWith("/ws/extension")
    ? serverUrl
    : `${serverUrl}/ws/extension`;
  return `${wsBase}?token=${encodeURIComponent(token)}`;
}
