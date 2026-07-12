type LingMirrorAuthMessage = {
  source?: unknown;
  type?: unknown;
  accessToken?: unknown;
};

function trustedLoginOrigin(origin: string): boolean {
  try {
    const url = new URL(origin);
    if (url.protocol !== "http:" && url.protocol !== "https:") return false;
    if (url.hostname === "localhost" && url.port === "3000") return true;
    return url.protocol === "https:" && (url.hostname === "lingmirror.com" || url.hostname.endsWith(".lingmirror.com"));
  } catch {
    return false;
  }
}

window.addEventListener("message", (event: MessageEvent<LingMirrorAuthMessage>) => {
  if (event.source !== window || !trustedLoginOrigin(event.origin)) return;
  const data = event.data;
  if (data?.source !== "lingmirror-web" || data.type !== "LINGMIRROR_EXTENSION_AUTH") return;
  const token = typeof data.accessToken === "string" ? data.accessToken.trim() : "";
  if (token.length < 32 || token.split(".").length !== 3) return;

  chrome.runtime.sendMessage({ type: "set_token", token }, (response) => {
    window.postMessage(
      {
        source: "lingmirror-extension",
        type: "LINGMIRROR_EXTENSION_AUTH_RESULT",
        ok: Boolean(response?.ok),
      },
      event.origin
    );
  });
});
