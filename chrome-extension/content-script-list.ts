/**
 * Content script injected into 1688 list/search/shop pages.
 * Extracts visible product cards — title, price range, detail URL, image.
 * Sends results to background service worker for forwarding to backend.
 */

function extractListItems(): Array<{title: string; price_range: string; detail_url: string; image_url?: string}> {
  const items: Array<{title: string; price_range: string; detail_url: string; image_url?: string}> = [];
  const seen = new Set<string>();

  const selectors = [
    ".offer-list-item", ".sm-offer-item", ".offer-item-row",
    "[class*='offer-card']", "[class*='product-item']",
    ".list-item", ".mo-items .mo-item", "li[class*='offer']",
  ].join(",");

  const cards = document.querySelectorAll(selectors);
  cards.forEach((card) => {
    try {
      const titleEl = card.querySelector<HTMLElement>('[class*="title"] a, [class*="name"] a, h3 a');
      const title = titleEl?.textContent?.trim() || "";

      const priceEl = card.querySelector<HTMLElement>("[class*='price']");
      const priceText = priceEl?.textContent?.trim() || "";
      const range = priceText.match(/(\d+\.?\d*)\s*[-~]\s*(\d+\.?\d*)/);
      const single = priceText.match(/(\d+\.?\d*)/);
      const priceRange = range ? `${range[1]}-${range[2]}` : single ? single[1] : "";

      const linkEl = card.querySelector<HTMLAnchorElement>('a[href*="offer"], a[href*="detail"], [class*="title"] a');
      let detailUrl = linkEl?.href || "";
      if (detailUrl && !detailUrl.startsWith("http")) detailUrl = "https:" + detailUrl;

      const imgEl = card.querySelector<HTMLImageElement>('img[src*=".jpg"], img[src*=".png"], img[src*=".webp"]');
      const src = imgEl?.src || imgEl?.getAttribute("data-src") || "";
      const imageUrl = src.startsWith("//") ? "https:" + src : src || undefined;

      if (title && detailUrl && !seen.has(detailUrl)) {
        seen.add(detailUrl);
        items.push({ title, price_range: priceRange, detail_url: detailUrl, image_url: imageUrl });
      }
    } catch { /* skip malformed card */ }
  });

  return items;
}

function extractListPageData() {
  return { page_url: location.href, collected_at: new Date().toISOString(), items: extractListItems() };
}

// On-demand fetch via chrome.runtime
chrome.runtime.onMessage.addListener((msg: any, _sender: chrome.runtime.MessageSender, sendResponse: (r: any) => void) => {
  if (msg.type === "fetch_list_page") {
    sendResponse({ type: "list_page_data", data: extractListItems() });
    return true;
  }
});

// Auto-extract on page load
function autoExtract() {
  setTimeout(() => {
    try {
      const data = extractListPageData();
      if (data.items.length > 0) {
        chrome.runtime.sendMessage({ type: "list_page_result", payload: { status: "ok", data } }).catch(() => {});
      }
    } catch { /* silent */ }
  }, 2000);
}
if (document.readyState === "complete") autoExtract();
else window.addEventListener("load", autoExtract);
