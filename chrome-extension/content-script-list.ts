/** Owner-triggered collection for visible 1688 home/search/list cards. */

type ListFieldStatus = "observed" | "unknown" | "parse_failed" | "no_sku";

interface ListPageData {
  schema_version: "sourcing1688.private.v1";
  offer_id_url: string;
  offer_id_page: string;
  source_url: string;
  collected_at: string;
  driver: string;
  parser_version: string;
  title: string;
  price_1688: number;
  price_model: "fixed" | "range" | "unknown";
  price_min?: number;
  price_max?: number;
  currency: "CNY";
  min_order_qty: number;
  images: string[];
  spec_variants?: never[];
  supplier_name: string;
  supplier_id_1688: string;
  supplier_business_id: string;
  attributes?: Record<string, string>;
  field_statuses: Record<string, ListFieldStatus>;
}

interface VisibleOffer {
  offerId: string;
  card: HTMLElement;
  pageData: ListPageData;
}

const LIST_UI_HOST_ID = "lingmirror-list-collector-host";
const LIST_CHECKBOX_ATTR = "data-lingmirror-offer-selector";
const selectedOfferIDs = new Set<string>();
const selectedOfferSnapshots = new Map<string, VisibleOffer>();
let currentOffers = new Map<string, VisibleOffer>();
let scanning = false;
let scanQueued = false;
let collectingBatch = false;
let cancelBatch = false;
let panelStatus: HTMLElement | null = null;
let panelResults: HTMLElement | null = null;
let collectSelectedButton: HTMLButtonElement | null = null;

function reliableOfferURL(raw: string): { offerId: string; sourceURL: string } | null {
  try {
    const url = new URL(raw, location.href);
    if (url.protocol !== "https:" || url.hostname !== "detail.1688.com") return null;
    const offerId = url.pathname.match(/^\/offer\/(\d+)\.html$/i)?.[1];
    if (!offerId) return null;
    return { offerId, sourceURL: `https://detail.1688.com/offer/${offerId}.html` };
  } catch {
    return null;
  }
}

function isVisibleCard(element: HTMLElement): boolean {
  if (element.hidden || element.getAttribute("aria-hidden") === "true") return false;
  for (let node: HTMLElement | null = element; node; node = node.parentElement) {
    const style = node.getAttribute("style") || "";
    if (/display\s*:\s*none|visibility\s*:\s*hidden/i.test(style)) return false;
  }
	const computed = getComputedStyle(element);
	if (computed.display === "none" || computed.visibility === "hidden" || computed.visibility === "collapse" || computed.opacity === "0") return false;
	if (typeof element.getBoundingClientRect === "function") {
		const rect = element.getBoundingClientRect();
		if (rect.width <= 0 || rect.height <= 0) return false;
		if (rect.bottom <= 0 || rect.right <= 0 || rect.top >= window.innerHeight || rect.left >= window.innerWidth) return false;
	}
  return true;
}

function closestProductCard(anchor: HTMLAnchorElement): HTMLElement {
  return (anchor.closest(
    "[data-offer-id], [data-tracelog*='offer'], .offer-item, .product-item, .item, li",
  ) as HTMLElement | null) || anchor;
}

function safeText(value: string | null | undefined): string {
  return (value || "").replace(/\s+/g, " ").trim();
}

function extractReliableTitle(anchor: HTMLAnchorElement, card: HTMLElement): string {
  const candidates = [
    anchor.getAttribute("title"),
    card.querySelector<HTMLElement>("[class*='title']")?.getAttribute("title"),
    card.querySelector<HTMLElement>("[class*='title']")?.textContent,
    anchor.querySelector<HTMLImageElement>("img")?.alt,
    anchor.textContent,
  ];
  for (const candidate of candidates) {
    const title = safeText(candidate);
    if (title.length >= 2 && title.length <= 500 && !/^¥|^￥/.test(title)) return title;
  }
  return "";
}

function extractReliablePrice(card: HTMLElement): { price: number; min?: number; max?: number; model: "fixed" | "range" | "unknown" } {
  const priceNode = card.querySelector<HTMLElement>("[data-price], [class*='price']");
  const dataPrice = safeText(priceNode?.getAttribute("data-price"));
  const text = dataPrice || safeText(priceNode?.textContent);
  if (!text || (!dataPrice && !/[¥￥]/.test(text))) return { price: 0, model: "unknown" };
  const numbers = text.match(/\d+(?:\.\d+)?/g)?.map(Number).filter((value) => Number.isFinite(value) && value > 0) || [];
  if (numbers.length === 0) return { price: 0, model: "unknown" };
  if (numbers.length >= 2 && /[-~–—至]/.test(text)) {
    const min = Math.min(numbers[0], numbers[1]);
    const max = Math.max(numbers[0], numbers[1]);
    return { price: min, min, max, model: "range" };
  }
  return { price: numbers[0], model: "fixed" };
}

function extractReliableMOQ(card: HTMLElement): number {
  const candidates = card.querySelectorAll<HTMLElement>("[data-moq], [class*='moq'], [class*='quantity'], span, div");
  for (const candidate of Array.from(candidates)) {
    const dataMOQ = safeText(candidate.getAttribute("data-moq"));
    if (dataMOQ && /^\d+$/.test(dataMOQ)) return Number(dataMOQ);
    const text = safeText(candidate.textContent);
    const match = text.match(/(?:^|[^\d.])(\d+)\s*(?:件|个|套|盒|包)\s*(?:起批|起订)(?:$|\D)/);
    if (match) return Number(match[1]);
  }
  return 0;
}

function extractReliableImage(card: HTMLElement): string[] {
  const image = card.querySelector<HTMLImageElement>("img");
  const raw = image?.currentSrc || image?.getAttribute("src") || image?.getAttribute("data-src") || "";
  if (!raw) return [];
  try {
    const url = new URL(raw, location.href);
    const approved = url.protocol === "https:" && (url.hostname.endsWith(".alicdn.com") || url.hostname.endsWith(".1688.com"));
    return approved ? [url.toString()] : [];
  } catch {
    return [];
  }
}

function extractReliableSupplier(card: HTMLElement): { name: string; id: string } {
  const selectors = [
    ".company-name", ".company", ".shop-name", ".shopname", ".seller-name",
    "[class*='company']", "[class*='shop']", "[data-company]",
  ];
  let name = "";
  for (const sel of selectors) {
    const el = card.querySelector<HTMLElement>(sel);
    if (el) {
      name = safeText(el.textContent);
      if (name && name.length <= 160 && !/^(进店|进入店铺|联系商家|收藏店铺|查看全部|实力商家|金牌制造)$/.test(name)) {
        break;
      }
    }
  }
  let id = "";
  const anchors = card.querySelectorAll<HTMLAnchorElement>("a[href]");
  for (const anchor of Array.from(anchors)) {
    const href = anchor.href;
    if (!href || href.includes("/offer/")) continue;
    const match = href.match(/^https?:\/\/([a-zA-Z0-9_-]+)\.1688\.com(?:\/|$)/);
    if (match) {
      id = match[1].trim();
      if (!name) {
        name = safeText(anchor.getAttribute("title") || anchor.textContent);
      }
      break;
    }
  }
  return { name, id };
}

function pageDataFromCard(anchor: HTMLAnchorElement, card: HTMLElement, identity: { offerId: string; sourceURL: string }): ListPageData | null {
  const title = extractReliableTitle(anchor, card);
  if (!title) return null;
  const price = extractReliablePrice(card);
  const moq = extractReliableMOQ(card);
  const images = extractReliableImage(card);
  const supplier = extractReliableSupplier(card);
  const pageData: ListPageData = {
    schema_version: "sourcing1688.private.v1",
    offer_id_url: identity.offerId,
    offer_id_page: identity.offerId,
    source_url: identity.sourceURL,
    collected_at: new Date().toISOString(),
    driver: "chrome_extension_list_visible",
    parser_version: "1688-list-visible-v1",
    title,
    price_1688: price.price,
    price_model: price.model,
    currency: "CNY",
    min_order_qty: moq,
    images,
    supplier_name: supplier.name,
    supplier_id_1688: supplier.id,
    supplier_business_id: supplier.id,
    field_statuses: {
      title: "observed",
      price: price.price > 0 ? "observed" : "unknown",
      moq: moq > 0 ? "observed" : "unknown",
      supplier: supplier.name ? "observed" : "unknown",
      images: images.length > 0 ? "observed" : "unknown",
      sku: "unknown",
    },
  };
  if (price.min) pageData.price_min = price.min;
  if (price.max) pageData.price_max = price.max;
  return pageData;
}

function extractVisibleOffers(): VisibleOffer[] {
  const offers = new Map<string, VisibleOffer>();
  for (const anchor of Array.from(document.querySelectorAll<HTMLAnchorElement>("a[href]"))) {
    const identity = reliableOfferURL(anchor.href);
    if (!identity) continue;
    const card = closestProductCard(anchor);
    if (!isVisibleCard(card) || offers.has(identity.offerId)) continue;
    const pageData = pageDataFromCard(anchor, card, identity);
    if (pageData) offers.set(identity.offerId, { offerId: identity.offerId, card, pageData });
  }
  return Array.from(offers.values());
}

function addIsolatedSelector(offer: VisibleOffer): void {
  const selector = `span[${LIST_CHECKBOX_ATTR}="${offer.offerId}"]`;
  if (offer.card.querySelector(selector)) return;
  const host = document.createElement("span");
  host.setAttribute(LIST_CHECKBOX_ATTR, offer.offerId);
  Object.assign(host.style, { position: "absolute", zIndex: "2147483000", left: "6px", top: "6px" });
  if (getComputedStyle(offer.card).position === "static") offer.card.style.position = "relative";
  const shadow = host.attachShadow({ mode: "open" });
  const label = document.createElement("label");
  Object.assign(label.style, { display: "flex", gap: "5px", alignItems: "center", background: "white", color: "#111827", border: "1px solid #c7d2fe", borderRadius: "7px", padding: "5px 7px", font: "12px system-ui", boxShadow: "0 2px 8px #0002", cursor: "pointer" });
  const checkbox = document.createElement("input");
  checkbox.type = "checkbox";
  checkbox.checked = selectedOfferIDs.has(offer.offerId);
  checkbox.addEventListener("change", () => {
    if (checkbox.checked) {
      selectedOfferIDs.add(offer.offerId);
      selectedOfferSnapshots.set(offer.offerId, offer);
    } else {
      selectedOfferIDs.delete(offer.offerId);
      selectedOfferSnapshots.delete(offer.offerId);
    }
    updatePanelSummary();
  });
  label.append(checkbox, document.createTextNode("选择"));
  shadow.append(label);
  offer.card.append(host);
}

function updatePanelSummary(message?: string): void {
  if (panelStatus) panelStatus.textContent = message || `本页当前可见 ${currentOffers.size} 个可靠商品链接，已选 ${selectedOfferIDs.size} 个`;
  if (collectSelectedButton) collectSelectedButton.disabled = collectingBatch || selectedOfferIDs.size === 0;
}

function scanVisibleOffers(): void {
  if (scanning) { scanQueued = true; return; }
  scanning = true;
  try {
    currentOffers = new Map(extractVisibleOffers().map((offer) => [offer.offerId, offer]));
    for (const offer of currentOffers.values()) {
      if (selectedOfferIDs.has(offer.offerId)) selectedOfferSnapshots.set(offer.offerId, offer);
      addIsolatedSelector(offer);
    }
    updatePanelSummary();
  } finally {
    scanning = false;
    if (scanQueued) { scanQueued = false; setTimeout(scanVisibleOffers, 100); }
  }
}

function queueScan(): void {
  if (scanQueued) return;
  scanQueued = true;
  setTimeout(() => { scanQueued = false; scanVisibleOffers(); }, 180);
}

function collectionRequestID(): string {
  return `collect_${crypto.randomUUID().replace(/-/g, "")}`;
}

function appendBatchResult(text: string, tone: "ok" | "warn" | "error", actions?: HTMLElement[]): void {
  if (!panelResults) return;
  const row = document.createElement("div");
  row.textContent = text;
  Object.assign(row.style, { marginTop: "7px", padding: "7px", borderRadius: "6px", background: tone === "ok" ? "#ecfdf3" : tone === "warn" ? "#fffbeb" : "#fef2f2", color: "#111827" });
  if (actions) for (const action of actions) row.append(action);
  panelResults.append(row);
}

async function submitVisibleOffer(offer: VisibleOffer, observationIntent?: "save_new_observation"): Promise<void> {
  const requestId = collectionRequestID();
  const response = await chrome.runtime.sendMessage({
    type: "collect_private_product", requestId, pageData: offer.pageData, observationIntent,
  }) as any;
  const payload = response?.payload;
  if (payload?.status === "saved") {
    appendBatchResult(`${offer.pageData.title}：已保存 #${payload.recordId}`, "ok");
    return;
  }
  if (payload?.status === "duplicate_requires_choice") {
    const view = document.createElement("button");
    view.textContent = "查看已有";
    view.addEventListener("click", () => void chrome.runtime.sendMessage({ type: "open_private_collection", recordId: payload.recordId }));
    const save = document.createElement("button");
    save.textContent = "保存新观察";
    save.style.marginLeft = "6px";
    save.addEventListener("click", () => void submitVisibleOffer(offer, "save_new_observation"));
    appendBatchResult(`${offer.pageData.title}：已有记录，需Owner选择`, "warn", [view, save]);
    return;
  }
  appendBatchResult(`${offer.pageData.title}：${payload?.message || "未确认保存"}`, "error");
}

async function collectOffers(offers: VisibleOffer[]): Promise<void> {
  if (collectingBatch || offers.length === 0) return;
  collectingBatch = true;
  cancelBatch = false;
  if (panelResults) panelResults.replaceChildren();
  updatePanelSummary(`准备采集 ${offers.length} 个当前可见商品，可随时停止`);
  try {
    for (let index = 0; index < offers.length; index += 1) {
      if (cancelBatch) {
        appendBatchResult(`已停止；剩余 ${offers.length - index} 个未提交`, "warn");
        break;
      }
      updatePanelSummary(`正在采集 ${index + 1}/${offers.length}：${offers[index].pageData.title}`);
      await submitVisibleOffer(offers[index]);
      if (index + 1 < offers.length) await new Promise((resolve) => setTimeout(resolve, 250));
    }
  } finally {
    collectingBatch = false;
    updatePanelSummary(cancelBatch ? "批量采集已停止" : "本批逐项处理完成；请核对下方每一项结果");
  }
}

function makePanelButton(text: string): HTMLButtonElement {
  const button = document.createElement("button");
  button.type = "button";
  button.textContent = text;
  Object.assign(button.style, { border: "1px solid #c7d2fe", borderRadius: "7px", padding: "7px 9px", background: "white", color: "#312e81", cursor: "pointer", margin: "5px 5px 0 0" });
  return button;
}

function installListCollectorUI(): void {
  if (document.getElementById(LIST_UI_HOST_ID)) return;
  const host = document.createElement("div");
  host.id = LIST_UI_HOST_ID;
  Object.assign(host.style, { position: "fixed", right: "18px", bottom: "18px", zIndex: "2147483647" });
  const shadow = host.attachShadow({ mode: "open" });
  const panel = document.createElement("section");
  Object.assign(panel.style, { width: "330px", maxHeight: "55vh", overflow: "auto", background: "#fff", color: "#111827", border: "1px solid #c7d2fe", borderRadius: "12px", padding: "12px", boxShadow: "0 10px 30px #0003", font: "13px/1.45 system-ui" });
  const title = document.createElement("strong");
  title.textContent = "凌镜 · 当前可见商品";
  panelStatus = document.createElement("div");
  panelStatus.style.marginTop = "6px";
  const selectAll = makePanelButton("勾选当前可见");
  selectAll.addEventListener("click", () => {
    for (const [id, offer] of currentOffers) {
      selectedOfferIDs.add(id);
      selectedOfferSnapshots.set(id, offer);
    }
    scanVisibleOffers();
  });
  collectSelectedButton = makePanelButton("采集选中");
  collectSelectedButton.addEventListener("click", () => void collectOffers(Array.from(selectedOfferIDs)
    .map((id) => currentOffers.get(id) || selectedOfferSnapshots.get(id))
    .filter((offer): offer is VisibleOffer => Boolean(offer))));
  const collectPage = makePanelButton("采集本页");
  collectPage.addEventListener("click", () => { scanVisibleOffers(); void collectOffers(Array.from(currentOffers.values())); });
  const cancel = makePanelButton("停止批量");
  cancel.addEventListener("click", () => { cancelBatch = true; });
  panelResults = document.createElement("div");
  panel.append(title, panelStatus, selectAll, collectSelectedButton, collectPage, cancel, panelResults);
  shadow.append(panel);
  host.setAttribute("aria-label", "凌镜1688列表采集");
  document.documentElement.append(host);
}

installListCollectorUI();
scanVisibleOffers();
new MutationObserver(queueScan).observe(document.documentElement, { childList: true, subtree: true });
window.addEventListener("scroll", queueScan, { passive: true });
window.addEventListener("popstate", queueScan);
