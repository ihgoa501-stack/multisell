type LingMirrorAuthMessage = {
  source?: unknown;
  type?: unknown;
  accessToken?: unknown;
  nonce?: unknown;
  environment?: unknown;
};

function trustedLoginOrigin(origin: string): boolean {
  try {
    const url = new URL(origin);
    if (url.protocol !== "http:" && url.protocol !== "https:") return false;
    if ((url.hostname === "localhost" || url.hostname === "127.0.0.1") && url.port === "3000") return true;
    return url.protocol === "https:" && (url.hostname === "lingmirror.com" || url.hostname === "owner.lingmirror.com" || url.hostname === "118.196.42.156");
  } catch {
    return false;
  }
}

window.addEventListener("message", (event: MessageEvent<LingMirrorAuthMessage>) => {
  if (event.source !== window || !trustedLoginOrigin(event.origin)) return;
  const data = event.data;
  if (data?.source !== "lingmirror-web") return;
  const nonce = typeof data.nonce === "string" ? data.nonce : "";
  if (!nonce) return;
  const type = data.type === "LINGMIRROR_EXTENSION_PAIRING" ? "begin_extension_pairing"
    : data.type === "LINGMIRROR_EXTENSION_PAIRING_CONFIRMED" ? "finish_extension_pairing" : "";
  if (!type) return;
  chrome.runtime.sendMessage({ type, nonce, environment: data.environment, origin: event.origin }, (response) => {
    window.postMessage(
      {
        source: "lingmirror-extension",
		type: type === "begin_extension_pairing" ? "LINGMIRROR_EXTENSION_PAIRING_RESULT" : "LINGMIRROR_EXTENSION_PAIRING_FINISHED",
        ok: Boolean(response?.ok),
		deviceId: response?.deviceId,
		extensionId: response?.extensionId,
		error: response?.error,
      },
      event.origin
    );
  });
});
