// Device-bound extension authentication. Web login JWTs are never stored.

const JWT_KEY = "lingmirror_jwt";
const SERVER_URL_KEY = "lingmirror_server_url";
const DEVICE_KEY = "lingmirror_extension_device";

export interface ExtensionDeviceCredential {
  deviceId: string;
  deviceSecret: string;
  environment: "development" | "acceptance" | "production";
	apiOrigin: string;
}

function tokenExpiresAfter(token: string, minimumRemainingSeconds = 30): boolean {
	try {
		const part = token.split(".")[1];
		if (!part) return false;
		const encoded = part.replace(/-/g, "+").replace(/_/g, "/");
		const payload = JSON.parse(atob(encoded));
		return payload?.type === "extension_access" && typeof payload?.exp === "number" && payload.exp > Math.floor(Date.now() / 1000) + minimumRemainingSeconds;
	} catch {
		return false;
	}
}

/** Retrieve the short-lived JWT from session storage. Legacy local tokens are
 * migrated once and removed so credentials do not survive a browser restart. */
export async function getJWT(): Promise<string | null> {
	const session = await chrome.storage.session.get([JWT_KEY]);
	if (typeof session[JWT_KEY] === "string" && tokenExpiresAfter(session[JWT_KEY])) return session[JWT_KEY];
	if (session[JWT_KEY]) await chrome.storage.session.remove(JWT_KEY);
	await chrome.storage.local.remove(JWT_KEY);
	const device = await getDeviceCredential();
	if (!device) return null;
	try {
		const serverUrl = await getServerUrl();
		if (device.apiOrigin !== new URL(getApiBaseUrl(serverUrl)).origin) {
			await clearDeviceCredential(); return null;
		}
		const response = await fetch(`${getApiBaseUrl(serverUrl)}/auth/extension-devices/refresh`, {
			method: "POST", headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ device_id: device.deviceId, device_secret: device.deviceSecret, environment: device.environment }),
		});
		const body = await response.json();
		const token = body?.data?.access_token;
		if (!response.ok || typeof token !== "string") throw new Error("device refresh rejected");
		await setJWT(token); return token;
	} catch {
		await clearJWT(); return null;
	}
}

/** Persist JWT only for the current browser session. */
export async function setJWT(token: string): Promise<void> {
	await chrome.storage.session.set({ [JWT_KEY]: token });
	await chrome.storage.local.remove(JWT_KEY);
}

/** Remove JWT token from chrome.storage.local (logout). */
export async function clearJWT(): Promise<void> {
	await Promise.all([
		chrome.storage.session.remove(JWT_KEY),
		chrome.storage.local.remove(JWT_KEY),
	]);
}

export async function getDeviceCredential(): Promise<ExtensionDeviceCredential | null> {
  const result = await chrome.storage.local.get([DEVICE_KEY]);
  const value = result[DEVICE_KEY] as ExtensionDeviceCredential | undefined;
	return value?.deviceId && value?.deviceSecret && value?.environment && value?.apiOrigin ? value : null;
}

export async function setDeviceCredential(value: ExtensionDeviceCredential): Promise<void> {
  await chrome.storage.local.set({ [DEVICE_KEY]: value });
}

export async function clearDeviceCredential(): Promise<void> {
  await Promise.all([clearJWT(), chrome.storage.local.remove(DEVICE_KEY)]);
}

/** Get the configured server WebSocket URL. */
export async function getServerUrl(): Promise<string> {
  const result = await chrome.storage.local.get([SERVER_URL_KEY]);
  return result[SERVER_URL_KEY] || "ws://localhost:8080";
}

/** Persist the server WebSocket URL. */
export async function setServerUrl(url: string): Promise<void> {
	const current = await getServerUrl();
	let changed = false;
	try { changed = new URL(getApiBaseUrl(current)).origin !== new URL(getApiBaseUrl(url)).origin; } catch { changed = true; }
	if (changed) await clearDeviceCredential();
  await chrome.storage.local.set({ [SERVER_URL_KEY]: url });
}

/** Build the HTTP login URL from a WebSocket server URL. */
export function getLoginUrl(serverUrl: string): string {
  const baseUrl = serverUrl
    .replace(/^ws:\/\//, "http://")
    .replace(/^wss:\/\//, "https://");
  const url = new URL(baseUrl);
  if ((url.hostname === "localhost" || url.hostname === "127.0.0.1") && url.port === "8080") {
    url.port = "3000";
  }
  url.pathname = "/settings/plugin";
  return url.toString();
}

/** Build the WebSocket endpoint URL (token sent as first message, not in URL). */
export function getWsUrl(serverUrl: string, _token: string): string {
  const wsBase = serverUrl.endsWith("/ws/extension")
    ? serverUrl
    : `${serverUrl}/ws/extension`;
  return wsBase;
}

/** Build the authenticated HTTP API base from the configured server URL. */
export function getApiBaseUrl(serverUrl: string): string {
  const httpBase = serverUrl
    .replace(/^ws:\/\//, "http://")
    .replace(/^wss:\/\//, "https://")
    .replace(/\/ws\/extension\/?$/, "")
    .replace(/\/$/, "");
  return `${httpBase}/api/v1`;
}
