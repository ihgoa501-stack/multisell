/** Owner-triggered collection for visible 1688 home/search/list cards. */

type ListItemData = import("./shared/protocol.js").ListItemData;

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
let selectAllButton: HTMLButtonElement | null = null;
let collectPageButton: HTMLButtonElement | null = null;
let listPanel: HTMLElement | null = null;
let listPanelContent: HTMLElement | null = null;
let listPanelCollapseButton: HTMLButtonElement | null = null;
type ListPanelSide = "left" | "right";
type ListPanelPosition = { side: ListPanelSide; top: number; collapsed: boolean };
const LIST_PANEL_POSITION_KEY = "lingmirror_list_collector_position_v1";
const LIST_PANEL_EDGE = 18;
let listPanelPosition: ListPanelPosition = { side: "right", top: 80, collapsed: false };

function clampListPanelTop(top: number): number {
  const height = Math.max(44, listPanel?.getBoundingClientRect().height || 360);
  return Math.max(12, Math.min(Math.round(top), Math.max(12, window.innerHeight - Math.min(height, window.innerHeight - 24) - 12)));
}

function applyListPanelPosition(): void {
  const host = document.getElementById(LIST_UI_HOST_ID);
  if (!host || !listPanel || !listPanelContent || !listPanelCollapseButton) return;
  listPanelPosition.top = clampListPanelTop(listPanelPosition.top);
  const isLeft = listPanelPosition.side === "left";
  Object.assign(host.style, {
    top: `${listPanelPosition.top}px`, bottom: "auto",
    left: isLeft ? `${LIST_PANEL_EDGE}px` : "auto",
    right: isLeft ? "auto" : `${LIST_PANEL_EDGE}px`,
  });
  listPanelContent.style.display = listPanelPosition.collapsed ? "none" : "block";
  listPanel.style.width = listPanelPosition.collapsed ? "auto" : "330px";
  listPanelCollapseButton.textContent = listPanelPosition.collapsed ? "展开" : "收起";
  listPanelCollapseButton.setAttribute("aria-expanded", listPanelPosition.collapsed ? "false" : "true");
}

async function persistListPanelPosition(): Promise<void> {
  try {
    await chrome.storage?.local?.set({ [LIST_PANEL_POSITION_KEY]: listPanelPosition });
  } catch {
    // Position persistence is convenience only; collection must keep working.
  }
}

async function restoreListPanelPosition(): Promise<void> {
  try {
    const stored = (await chrome.storage?.local?.get(LIST_PANEL_POSITION_KEY))?.[LIST_PANEL_POSITION_KEY] as Partial<ListPanelPosition> | undefined;
    if (stored && (stored.side === "left" || stored.side === "right") && Number.isFinite(stored.top)) {
      listPanelPosition = { side: stored.side, top: Number(stored.top), collapsed: stored.collapsed === true };
    } else {
      listPanelPosition.top = clampListPanelTop(window.innerHeight - (listPanel?.getBoundingClientRect().height || 360) - LIST_PANEL_EDGE);
    }
  } catch {
    // Use the safe default when extension storage is unavailable.
  }
  applyListPanelPosition();
}

function setListPanelCollapsed(collapsed: boolean): void {
  listPanelPosition.collapsed = collapsed;
  applyListPanelPosition();
  void persistListPanelPosition();
}

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

function extractReliableImage(card: HTMLElement, anchor?: HTMLAnchorElement): string[] {
  const image = anchor?.querySelector<HTMLImageElement>("img") || card.querySelector<HTMLImageElement>("img");
  const raw = image?.currentSrc || image?.getAttribute("src") || image?.getAttribute("data-src") || "";
  if (!raw) return [];
  try {
    const url = new URL(raw, location.href);
    const approved = url.protocol === "https:" && (
      url.hostname === "alicdn.com" || url.hostname.endsWith(".alicdn.com") ||
      url.hostname === "1688.com" || url.hostname.endsWith(".1688.com")
    );
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
  const isValid = (val: string) => val && val.length <= 160 && !/^(进店|进入店铺|联系商家|收藏店铺|查看全部|实力商家|金牌制造)$/.test(val);

  let name = "";
  for (const sel of selectors) {
    const el = card.querySelector<HTMLElement>(sel);
    if (el) {
      const candidate = safeText(el.textContent);
      if (isValid(candidate)) {
        name = candidate;
        break;
      }
    }
  }
  let id = "";
  const anchors = card.querySelectorAll<HTMLAnchorElement>("a[href]");
  const systemSubdomains = /^(s|search|detail|member|login|cbu|page|work|info|spm|show|m|club|dianpu|winport)$/i;
  for (const anchor of Array.from(anchors)) {
    const href = anchor.href;
    if (!href || href.includes("/offer/")) continue;
    const match = href.match(/^https?:\/\/([a-zA-Z0-9_-]+)\.1688\.com(?:\/|\?|#|$)/);
    if (match && !systemSubdomains.test(match[1])) {
      id = match[1].trim();
      if (!name) {
        const candidate = safeText(anchor.getAttribute("title") || anchor.textContent);
        if (isValid(candidate)) {
          name = candidate;
        }
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
  const images = extractReliableImage(card, anchor);
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
  const existing = offer.card.querySelector(selector);
  if (existing) {
    const checkbox = existing.shadowRoot?.querySelector('input[type="checkbox"]') as HTMLInputElement | null;
    if (checkbox) checkbox.checked = selectedOfferIDs.has(offer.offerId);
    return;
  }
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
  if (selectAllButton) selectAllButton.disabled = collectingBatch;
  if (collectPageButton) collectPageButton.disabled = collectingBatch;
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

async function submitVisibleOffer(offer: VisibleOffer, observationIntent?: "save_new_observation"): Promise<boolean> {
  const requestId = collectionRequestID();
  try {
    const response = await chrome.runtime.sendMessage({
      type: "collect_private_product", requestId, pageData: offer.pageData, observationIntent,
    }) as any;
    const payload = response?.payload;
    if (payload?.status === "saved") {
      appendBatchResult(`${offer.pageData.title}：已保存 #${payload.recordId}`, "ok");
      selectedOfferIDs.delete(offer.offerId);
      selectedOfferSnapshots.delete(offer.offerId);
      const host = offer.card.querySelector(`span[${LIST_CHECKBOX_ATTR}="${offer.offerId}"]`);
      const checkbox = host?.shadowRoot?.querySelector('input[type="checkbox"]') as HTMLInputElement | null;
      if (checkbox) checkbox.checked = false;
      return true;
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
      return false;
    }
    const actions: HTMLElement[] = [];
    if (payload?.status === "not_saved") {
      const retry = makePanelButton("重试此项");
      retry.addEventListener("click", async () => {
        retry.disabled = true;
        retry.textContent = "重试中…";
        await submitVisibleOffer(offer);
        retry.textContent = "已重试";
      });
      actions.push(retry);
    }
    appendBatchResult(`${offer.pageData.title}：${payload?.message || "未确认保存"}`, "error", actions);
    return false;
  } catch (err: any) {
    appendBatchResult(`${offer.pageData.title}：发送失败 (${err?.message || err})`, "error");
    return false;
  }
}

function confirmBatch(offers: VisibleOffer[], scope: "selected" | "page"): void {
  if (collectingBatch || !panelResults) return;
  panelResults.replaceChildren();
  if (offers.length === 0) {
    appendBatchResult(scope === "selected" ? "没有已选商品" : "本页当前没有可采集的可靠商品", "warn");
    return;
  }
  const confirm = makePanelButton(`确认采集 ${offers.length} 个`);
  confirm.addEventListener("click", () => void collectOffers(offers));
  const cancel = makePanelButton("取消");
  cancel.addEventListener("click", () => {
    panelResults?.replaceChildren();
    updatePanelSummary();
  });
  const scopeText = scope === "selected" ? "已选商品" : "本页当前可见商品";
  appendBatchResult(`即将采集 ${offers.length} 个${scopeText}；不自动翻页。请确认数量后提交。`, "warn", [confirm, cancel]);
}

async function collectOffers(offers: VisibleOffer[]): Promise<void> {
  if (collectingBatch || offers.length === 0) return;
  collectingBatch = true;
  cancelBatch = false;
  if (panelResults) panelResults.replaceChildren();
  updatePanelSummary(`准备采集 ${offers.length} 个当前可见商品，可随时停止`);
  try {
    const host = document.getElementById(LIST_UI_HOST_ID);
    const delayInput = host?.shadowRoot?.getElementById("lingmirror-batch-delay-input") as HTMLInputElement | null;

    for (let index = 0; index < offers.length; index += 1) {
      if (cancelBatch) {
        appendBatchResult("已停止；剩余 " + (offers.length - index) + " 个未提交", "warn");
        break;
      }
      updatePanelSummary("正在采集 " + (index + 1) + "/" + offers.length + "：" + offers[index].pageData.title);
      await submitVisibleOffer(offers[index]);
      if (index + 1 < offers.length) {
        const parsed = parseFloat(delayInput?.value || "");
        const currentInputSeconds = isNaN(parsed) ? 2.0 : parsed;
        const currentDelayMs = Math.max(0.5, currentInputSeconds) * 1000;
        const jitter = (currentDelayMs * 0.7) + Math.random() * (currentDelayMs * 0.6);
        await new Promise((resolve) => setTimeout(resolve, jitter));
      }
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
  Object.assign(panel.style, { width: "330px", maxWidth: "calc(100vw - 36px)", maxHeight: "calc(100vh - 24px)", overflow: "auto", background: "#fff", color: "#111827", border: "1px solid #c7d2fe", borderRadius: "12px", padding: "12px", boxShadow: "0 10px 30px #0003", font: "13px/1.45 system-ui" });
  listPanel = panel;
  const header = document.createElement("div");
  header.id = "lingmirror-list-collector-drag-handle";
  header.tabIndex = 0;
  header.setAttribute("role", "toolbar");
  header.setAttribute("aria-label", "拖动凌镜列表采集；方向键调整位置");
  Object.assign(header.style, { fontWeight: "700", cursor: "grab", userSelect: "none", touchAction: "none", minHeight: "26px" });
  const title = document.createElement("strong");
  title.textContent = "凌镜 · 当前可见商品";
  const collapse = makePanelButton("收起");
  collapse.setAttribute("aria-label", "收起或展开凌镜列表采集");
  Object.assign(collapse.style, { float: "right", margin: "0", padding: "2px 7px" });
  collapse.addEventListener("click", (event) => { event.stopPropagation(); setListPanelCollapsed(!listPanelPosition.collapsed); });
  listPanelCollapseButton = collapse;
  header.append(title, collapse);
  const content = document.createElement("div");
  listPanelContent = content;
  const scopeHint = document.createElement("div");
  scopeHint.textContent = "只处理当前已加载且可见的商品，不自动翻页。";
  Object.assign(scopeHint.style, { marginTop: "4px", color: "#4b5563" });
  panelStatus = document.createElement("div");
  panelStatus.style.marginTop = "6px";

  const delayContainer = document.createElement("div");
  Object.assign(delayContainer.style, { marginTop: "8px", marginBottom: "8px", display: "flex", gap: "6px", alignItems: "center" });
  const delayLabel = document.createElement("label");
  delayLabel.textContent = "采集间隔 (秒):";
  const delayInput = document.createElement("input");
  delayInput.type = "number";
  delayInput.id = "lingmirror-batch-delay-input";
  delayInput.min = "0.5";
  delayInput.step = "0.5";
  delayInput.value = "2.0";
  Object.assign(delayInput.style, { width: "55px", border: "1px solid #c7d2fe", borderRadius: "5px", padding: "3px 5px", font: "12px system-ui" });
  delayContainer.append(delayLabel, delayInput);

  selectAllButton = makePanelButton("勾选当前可见");
  selectAllButton.addEventListener("click", () => {
    for (const [id, offer] of currentOffers) {
      selectedOfferIDs.add(id);
      selectedOfferSnapshots.set(id, offer);
    }
    scanVisibleOffers();
  });
  collectSelectedButton = makePanelButton("采集选中");
  collectSelectedButton.addEventListener("click", () => confirmBatch(Array.from(selectedOfferIDs)
    .map((id) => currentOffers.get(id) || selectedOfferSnapshots.get(id))
    .filter((offer): offer is VisibleOffer => Boolean(offer)), "selected"));
  collectPageButton = makePanelButton("采集本页当前可见");
  collectPageButton.addEventListener("click", () => { scanVisibleOffers(); confirmBatch(Array.from(currentOffers.values()), "page"); });
  const cancel = makePanelButton("停止批量");
  cancel.addEventListener("click", () => { cancelBatch = true; });
  panelResults = document.createElement("div");
  content.append(scopeHint, panelStatus, delayContainer, selectAllButton, collectSelectedButton, collectPageButton, cancel, panelResults);
  panel.append(header, content);
  shadow.append(panel);
  host.setAttribute("aria-label", "凌镜1688列表采集");
  document.documentElement.append(host);

  let dragStart: { pointerId: number; clientX: number; clientY: number; left: number; top: number } | null = null;
  header.addEventListener("pointerdown", (event) => {
    if ((event.target as HTMLElement).closest("button")) return;
    const rect = host.getBoundingClientRect();
    dragStart = { pointerId: event.pointerId, clientX: event.clientX, clientY: event.clientY, left: rect.left, top: rect.top };
    header.setPointerCapture?.(event.pointerId);
    header.style.cursor = "grabbing";
    event.preventDefault();
  });
  header.addEventListener("pointermove", (event) => {
    if (!dragStart || dragStart.pointerId !== event.pointerId) return;
    const width = Math.max(180, host.getBoundingClientRect().width || 354);
    const nextLeft = Math.max(12, Math.min(dragStart.left + event.clientX - dragStart.clientX, Math.max(12, window.innerWidth - width - 12)));
    listPanelPosition.top = clampListPanelTop(dragStart.top + event.clientY - dragStart.clientY);
    Object.assign(host.style, { left: `${nextLeft}px`, right: "auto", top: `${listPanelPosition.top}px`, bottom: "auto" });
  });
  const finishDrag = (event: PointerEvent) => {
    if (!dragStart || dragStart.pointerId !== event.pointerId) return;
    const rect = host.getBoundingClientRect();
    listPanelPosition.side = rect.left + rect.width / 2 < window.innerWidth / 2 ? "left" : "right";
    listPanelPosition.top = clampListPanelTop(rect.top);
    dragStart = null;
    header.releasePointerCapture?.(event.pointerId);
    header.style.cursor = "grab";
    applyListPanelPosition();
    void persistListPanelPosition();
  };
  header.addEventListener("pointerup", finishDrag);
  header.addEventListener("pointercancel", finishDrag);
  header.addEventListener("keydown", (event) => {
    if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
      listPanelPosition.side = event.key === "ArrowLeft" ? "left" : "right";
    } else if (event.key === "ArrowUp" || event.key === "ArrowDown") {
      listPanelPosition.top = clampListPanelTop(listPanelPosition.top + (event.key === "ArrowUp" ? -16 : 16));
    } else if (event.key === "Escape") {
      setListPanelCollapsed(true);
      event.preventDefault();
      return;
    } else {
      return;
    }
    event.preventDefault();
    applyListPanelPosition();
    void persistListPanelPosition();
  });
  window.addEventListener("resize", () => {
    applyListPanelPosition();
    void persistListPanelPosition();
  });
  void restoreListPanelPosition();
}

function extractListItemsData(): ListItemData[] {
  const offers = extractVisibleOffers();
  return offers.map((offer) => {
    const data = offer.pageData;
    const priceRange = data.price_min && data.price_max
      ? `${data.price_min}-${data.price_max}`
      : data.price_1688 > 0
      ? `${data.price_1688}`
      : "";
    return {
      title: data.title,
      price_range: priceRange,
      detail_url: data.source_url,
      image_url: data.images[0] || "",
      raw_text: offer.card.textContent || "",
    };
  });
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === "fetch_list_page") {
    try {
      const items = extractListItemsData();
      sendResponse({ data: items });
    } catch (err: any) {
      sendResponse({ error: err?.message || String(err) });
    }
    return true;
  }
});

if (window.location.hostname !== "detail.1688.com") {
  installListCollectorUI();
  scanVisibleOffers();
  new MutationObserver((mutations) => {
    const isExtensionMutation = mutations.every((m) => {
      const target = m.target as HTMLElement | null;
      if (target?.id === LIST_UI_HOST_ID || target?.closest(`#${LIST_UI_HOST_ID}`)) return true;
      if (target?.hasAttribute(LIST_CHECKBOX_ATTR) || target?.closest(`[${LIST_CHECKBOX_ATTR}]`)) return true;

      const isExtensionNode = (node: Node) => {
        if (node instanceof HTMLElement) {
          return node.id === LIST_UI_HOST_ID || node.hasAttribute(LIST_CHECKBOX_ATTR) || node.closest(`[${LIST_CHECKBOX_ATTR}]`) !== null;
        }
        return false;
      };

      const addedAllExtension = Array.from(m.addedNodes).every(isExtensionNode);
      const removedAllExtension = Array.from(m.removedNodes).every(isExtensionNode);
      return addedAllExtension && removedAllExtension;
    });
    if (!isExtensionMutation) {
      queueScan();
    }
  }).observe(document.documentElement, { childList: true, subtree: true });
  window.addEventListener("scroll", queueScan, { passive: true });
  window.addEventListener("popstate", queueScan);
}
