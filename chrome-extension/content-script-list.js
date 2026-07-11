"use strict";
/**
 * Content script injected into Ozon and supported supplier list/search pages.
 * Extracts visible product cards — title, price range, detail URL, image.
 * Sends results to background service worker for forwarding to backend.
 */
function extractListItems() {
    const items = [];
    const seen = new Set();
    const selectors = [
        ".offer-list-item", ".sm-offer-item", ".offer-item-row",
        "[class*='offer-card']", "[class*='product-item']",
        ".list-item", ".mo-items .mo-item", "li[class*='offer']",
        "[data-widget='searchResultsV2'] [data-index]", "[class*='tile-root']",
    ].join(",");
    const cards = document.querySelectorAll(selectors);
    cards.forEach((card) => {
        try {
            const productLinks = Array.from(card.querySelectorAll('a[href*="offer"], a[href*="detail"], [class*="title"] a, a[href*="/product/"]'));
            const linkEl = productLinks.sort((left, right) => (right.textContent || "").trim().length - (left.textContent || "").trim().length)[0];
            const titleEl = card.querySelector('[class*="title"] a, [class*="name"] a, h3 a, a[href*="/product/"] span');
            const title = titleEl?.textContent?.trim() || linkEl?.textContent?.trim() || linkEl?.getAttribute("title")?.trim() || "";
            const priceEl = card.querySelector("[class*='price']") ||
                Array.from(card.querySelectorAll("span")).find((el) => /^[\s\d.,\u00a0\u2009\u202f]+₽\s*$/.test((el.textContent || "").trim()));
            const priceText = priceEl?.textContent?.trim() || "";
            const normalizedPrice = priceText.replace(/[\s\u00a0\u202f]+/g, "");
            const range = normalizedPrice.match(/(\d+(?:[.,]\d+)?)\s*[-~]\s*(\d+(?:[.,]\d+)?)/);
            const single = normalizedPrice.match(/(\d+(?:[.,]\d+)?)/);
            const priceRange = range ? `${range[1]}-${range[2]}` : single ? single[1] : "";
            let detailUrl = linkEl?.href || "";
            if (detailUrl && !detailUrl.startsWith("http"))
                detailUrl = "https:" + detailUrl;
            const imgEl = card.querySelector('img[src*=".jpg"], img[src*=".png"], img[src*=".webp"]');
            const src = imgEl?.src || imgEl?.getAttribute("data-src") || "";
            const imageUrl = src.startsWith("//") ? "https:" + src : src || undefined;
            if (title && detailUrl && !seen.has(detailUrl)) {
                seen.add(detailUrl);
                items.push({
                    title,
                    price_range: priceRange,
                    detail_url: detailUrl,
                    image_url: imageUrl,
                    raw_text: (card.textContent || "").trim().slice(0, 8000),
                    raw_html: card.outerHTML.slice(0, 16000),
                });
            }
        }
        catch { /* skip malformed card */ }
    });
    return items;
}
function extractListPageData() {
    return { page_url: location.href, collected_at: new Date().toISOString(), items: extractListItems() };
}
// On-demand fetch via chrome.runtime
chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
    if (msg.type === "fetch_list_page") {
        void waitForListItems().then((items) => sendResponse({ type: "list_page_data", data: items }));
        return true;
    }
});
async function waitForListItems() {
    for (let attempt = 0; attempt < 10; attempt++) {
        const items = extractListItems();
        if (items.length > 0)
            return items;
        if (attempt === 3)
            window.scrollTo({ top: document.body.scrollHeight / 2, behavior: "instant" });
        await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    return [];
}
// Auto-extract on page load
function autoExtractListPage() {
    setTimeout(() => {
        try {
            const data = extractListPageData();
            if (data.items.length > 0) {
                chrome.runtime.sendMessage({ type: "list_page_result", payload: { status: "ok", data } }).catch(() => { });
            }
        }
        catch { /* silent */ }
    }, 2000);
}
if (document.readyState === "complete")
    autoExtractListPage();
else
    window.addEventListener("load", autoExtractListPage);
//# sourceMappingURL=content-script-list.js.map
