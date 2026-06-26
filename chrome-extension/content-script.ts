/**
 * Content script injected into 1688 product detail pages (detail.1688.com/offer/*).
 *
 * Extracts structured product data on page load and on demand.
 * Communicates with the background service worker via chrome.runtime.
 */

// ─── Type definitions (self-contained — no module imports for content scripts) ─

interface PageData {
  source_url: string;
  collected_at: string;
  driver: string;
  title: string;
  price_1688: number;
  price_min?: number | null;
  price_max?: number | null;
  currency: string;
  min_order_qty: number;
  images: string[];
  spec_variants?: SpecVariant[];
  supplier_name: string;
  supplier_id_1688: string;
  supplier_score?: number | null;
  description?: string;
  attributes?: Record<string, string>;
  package_weight_kg?: number | null;
  package_length_cm?: number | null;
  package_width_cm?: number | null;
  package_height_cm?: number | null;
  freight_cny?: number | null;
}

interface SpecVariant {
  spec: string;
  price: number;
  stock: number;
  image_url?: string;
}

interface ContentScriptFetchRequest {
  type: "fetch_product_from_page";
  requestId: string;
}

interface ContentScriptFetchResult {
  type: "fetch_product_from_page_result";
  requestId: string;
  payload:
    | { status: "ok"; data: PageData }
    | { code: string; message: string };
}

// ─── Extraction strategies ─────────────────────────────────────────────────

/**
 * Attempt to extract structured data from embedded JSON in script tags.
 * 1688 often embeds product data in window.__NUXT__, __INITIAL_STATE__, or
 * <script type="application/json"> tags.
 */
function tryEmbeddedJSON(): Partial<PageData> | null {
  const scripts = document.querySelectorAll("script");
  for (const script of scripts) {
    const text = script.textContent || "";

    // Try application/json script tags
    if (
      script.getAttribute("type") === "application/json" ||
      script.getAttribute("type") === "application/ld+json"
    ) {
      try {
        const parsed = JSON.parse(text);
        if (parsed?.name || parsed?.productName) {
          return extractFromLDJSON(parsed);
        }
      } catch {
        // continue
      }
    }

    // Try window.__NUXT__ state
    if (text.includes("__NUXT__")) {
      try {
        const match = text.match(/window\.__NUXT__\s*=\s*({.+?});/s);
        if (match) {
          const nuxt = JSON.parse(match[1]);
          const data = extractFromNuxt(nuxt);
          if (data?.title) return data;
        }
      } catch {
        // continue
      }
    }

    // Try window.__INITIAL_STATE__ or window.data
    if (text.includes("__INITIAL_STATE__") || text.includes("skuMap") || text.includes("offerId")) {
      try {
        const match = text.match(/window\.__INITIAL_STATE__\s*=\s*({.+?});/s);
        if (match) {
          const state = JSON.parse(match[1]);
          const data = extractFromInitialState(state);
          if (data?.title) return data;
        }
      } catch {
        // continue
      }
    }
  }
  return null;
}

function extractFromLDJSON(parsed: Record<string, unknown>): Partial<PageData> {
  const data: Partial<PageData> = {};
  if (typeof parsed.name === "string") data.title = parsed.name;
  if (typeof parsed.productName === "string")
    data.title = parsed.productName as string;
  if (parsed.offers && typeof parsed.offers === "object") {
    const offers = parsed.offers as Record<string, unknown>;
    if (typeof offers.price === "number") data.price_1688 = offers.price as number;
    if (typeof offers.price === "string")
      data.price_1688 = parseFloat(offers.price as string) || 0;
  }
  if (parsed.image) {
    const img = parsed.image;
    if (typeof img === "string") data.images = [img];
    if (Array.isArray(img)) data.images = img.filter((i): i is string => typeof i === "string");
  }
  if (typeof parsed.description === "string") data.description = parsed.description as string;
  return data;
}

function extractFromNuxt(nuxt: Record<string, unknown>): Partial<PageData> {
  const data: Partial<PageData> = {};
  try {
    // Navigate through Nuxt state tree to find product data
    const state = (nuxt as any).state || nuxt;
    const detail = state.detail || state.productDetail || state.offerDetail || state;
    const offer = detail.offer || detail.product || detail;

    if (offer.subject || offer.title) {
      data.title = offer.subject || offer.title;
    }
    if (offer.price) {
      data.price_1688 = parseFloat(offer.price) || 0;
    }
    if (offer.priceRange && Array.isArray(offer.priceRange)) {
      const prices = offer.priceRange
        .map((p: any) => parseFloat(p.price || p))
        .filter((p: number) => !isNaN(p));
      if (prices.length > 0) {
        data.price_min = Math.min(...prices);
        data.price_max = Math.max(...prices);
        if (!data.price_1688) data.price_1688 = prices[0];
      }
    }
    if (offer.images && Array.isArray(offer.images)) {
      data.images = offer.images.map((img: any) =>
        typeof img === "string" ? img : img.url || img.imageUrl || ""
      ).filter(Boolean);
    }
    if (offer.skuMap || offer.skuList) {
      data.spec_variants = extractSKUData(offer.skuMap || offer.skuList);
    }
    if (offer.companyName || offer.shopName) {
      data.supplier_name = offer.companyName || offer.shopName;
    }
    if (offer.companyId || offer.shopId) {
      data.supplier_id_1688 = String(offer.companyId || offer.shopId);
    }
  } catch {
    // silent
  }
  return data;
}

function extractFromInitialState(state: Record<string, unknown>): Partial<PageData> {
  const data: Partial<PageData> = {};
  try {
    const s = state as any;
    const detail = s.detail || s.offerDetail || s.productDetail || s;
    const offer = detail.offer || detail.product || detail;

    if (offer.subject || offer.title) data.title = offer.subject || offer.title;
    if (offer.price) data.price_1688 = parseFloat(offer.price) || 0;
    if (offer.images && Array.isArray(offer.images)) {
      data.images = offer.images
        .map((img: any) => (typeof img === "string" ? img : img.url))
        .filter(Boolean);
    }
    if (offer.skuMap || offer.skuList) {
      data.spec_variants = extractSKUData(offer.skuMap || offer.skuList);
    }
  } catch {
    // silent
  }
  return data;
}

function extractSKUData(skuData: any): SpecVariant[] {
  const variants: SpecVariant[] = [];
  try {
    if (Array.isArray(skuData)) {
      for (const item of skuData) {
        variants.push({
          spec: item.spec || item.name || item.skuName || "",
          price: parseFloat(item.price || item.skuPrice || "0") || 0,
          stock: parseInt(item.stock || item.quantity || "0", 10) || 0,
          image_url: item.image || item.imageUrl || undefined,
        });
      }
    } else if (typeof skuData === "object") {
      for (const [key, val] of Object.entries(skuData as Record<string, any>)) {
        variants.push({
          spec: key,
          price: parseFloat(val.price || val.skuPrice || "0") || 0,
          stock: parseInt(val.stock || val.quantity || "0", 10) || 0,
          image_url: val.image || val.imageUrl || undefined,
        });
      }
    }
  } catch {
    // silent
  }
  return variants;
}

// ─── DOM-based extraction (fallback) ───────────────────────────────────────

function extractTitleFromDOM(): string {
  const selectors = [
    "h1[data-title]",
    ".detail-title h1",
    ".mod-detail h1",
    ".product-title",
    "h1.title",
    ".mod-detail-title",
    ".detail-title",
    "[data-component-title] h1",
  ];

  for (const sel of selectors) {
    const el = document.querySelector(sel);
    if (el) {
      const text = (el as HTMLElement).innerText?.trim() || el.textContent?.trim() || "";
      if (text) return text;
    }
  }

  // Meta fallback
  const og = document.querySelector('meta[property="og:title"]');
  if (og instanceof HTMLMetaElement && og.content) return og.content.trim();

  const desc = document.querySelector('meta[name="description"]');
  if (desc instanceof HTMLMetaElement && desc.content) {
    // Often the first part of description is the title
    return desc.content.split(/[。.，,\n]/)[0].trim();
  }

  return document.title?.replace(/ - .+$/, "").trim() || "";
}

function extractPriceFromDOM(): number {
  const selectors = [
    '[data-price]',
    '.price-con .price',
    '.detail-price .price',
    '.price',
    '.mod-price .price',
    '#mod-detail-price .price',
    '.offer-price',
    '.product-price',
    '[class*="price"]',
  ];

  for (const sel of selectors) {
    const el = document.querySelector(sel);
    if (el) {
      const text = el.getAttribute("data-price") || el.textContent?.trim() || "";
      const match = text.match(/(\d+\.?\d*)/);
      if (match) return parseFloat(match[1]);
    }
  }

  // Look for ¥ price pattern in body text
  const bodyText = document.body.innerText;
  const priceMatch = bodyText.match(/[¥￥]\s*(\d+\.?\d*)/);
  if (priceMatch) return parseFloat(priceMatch[1]);

  return 0;
}

function extractImagesFromDOM(): string[] {
  const images = new Set<string>();

  const addUrl = (src: string | null | undefined) => {
    if (!src) return;
    const url = src.startsWith("//") ? "https:" + src : src;
    if (url.startsWith("http") && !url.includes("data:image")) {
      images.add(url);
    }
  };

  const selectors = [
    ".image-item img",
    "#dt-tab img",
    ".detail-gallery img",
    ".gallery img",
    ".mod-detail-gallery img",
    "ul.spec-items img",
    ".tab-content img",
    ".image-list img",
    ".main-img img",
    '[class*="gallery"] img',
    '[class*="preview"] img',
  ];

  for (const sel of selectors) {
    const els = document.querySelectorAll(sel);
    if (els.length > 0) {
      els.forEach((img) => {
        const el = img as HTMLImageElement;
        addUrl(el.src || el.getAttribute("data-src") || el.getAttribute("data-lazy-src"));
      });
      if (images.size > 0) break;
    }
  }

  // Meta fallback
  const og = document.querySelector('meta[property="og:image"]');
  if (og instanceof HTMLMetaElement && og.content) {
    addUrl(og.content);
  }

  return Array.from(images);
}

function extractSpecVariantsFromDOM(): SpecVariant[] {
  const variants: SpecVariant[] = [];

  // Try to parse SKU data embedded in data attributes
  const skuContainers = document.querySelectorAll("[data-sku], [data-skuid]");
  if (skuContainers.length > 0) {
    skuContainers.forEach((el) => {
      const spec = (el as HTMLElement).innerText?.trim() || "";
      const priceStr = el.getAttribute("data-price") || "";
      const price = parseFloat(priceStr) || 0;
      if (spec) {
        // Check if already added
        if (!variants.some((v) => v.spec === spec)) {
          variants.push({ spec, price, stock: 0 });
        }
      }
    });
  }

  // Fallback: look for spec selectors
  if (variants.length === 0) {
    const specSelectors = [
      ".sku-item",
      ".prop-item",
      ".sku-name",
      "[class*='sku'] li",
      ".attr-item",
    ];

    for (const sel of specSelectors) {
      const items = document.querySelectorAll(sel);
      if (items.length > 0) {
        items.forEach((item) => {
          const spec = (item as HTMLElement).innerText?.trim() || "";
          if (spec && !variants.some((v) => v.spec === spec)) {
            variants.push({ spec, price: 0, stock: 0 });
          }
        });
        break;
      }
    }
  }

  return variants;
}

function extractSupplierFromDOM(): { name: string; id: string; score: number | null } {
  let name = "";
  let id = "";
  let score: number | null = null;

  const nameSelectors = [
    ".company-name",
    ".mod-supplier__name",
    ".supplier-name",
    ".shop-name",
    "[data-supplier]",
    ".seller-info .name",
    ".shop-info .name",
    ".store-name",
    '[class*="supplier"]',
    '[class*="shop"]',
    '[class*="store"]',
  ];

  for (const sel of nameSelectors) {
    const el = document.querySelector(sel);
    if (el) {
      name = (el as HTMLElement).innerText?.trim() || "";
      if (name) break;
    }
  }

  // Try to extract supplier ID from links
  const allLinks = document.querySelectorAll('a[href*="1688.com"]');
  for (const link of allLinks) {
    const href = (link as HTMLAnchorElement).href || "";
    const match = href.match(/companyid=(\d+)/) || href.match(/company\/(\d+)/);
    if (match) {
      id = match[1];
      break;
    }
  }

  // Supplier score
  const scoreSelectors = [
    ".supplier-score",
    ".company-score",
    "[data-score]",
    ".mod-supplier__score",
    ".seller-rating",
    '[class*="score"]',
    '[class*="rating"]',
  ];

  for (const sel of scoreSelectors) {
    const el = document.querySelector(sel);
    if (el) {
      const text = (el as HTMLElement).innerText?.trim() || el.getAttribute("data-score") || "";
      const match = text.match(/(\d+\.?\d*)/);
      if (match) {
        score = parseInt(match[1], 10) || null;
        break;
      }
    }
  }

  return { name, id, score };
}

function extractDescriptionFromDOM(): string {
  // Meta description first
  const meta = document.querySelector('meta[name="description"]');
  if (meta instanceof HTMLMetaElement && meta.content) {
    return meta.content.trim().substring(0, 2000);
  }

  const descSelectors = [
    ".desc-content",
    ".detail-desc",
    "#description",
    ".mod-detail-description",
    ".attributes",
    '[class*="description"]',
    '[class*="detail"]',
  ];

  for (const sel of descSelectors) {
    const el = document.querySelector(sel);
    if (el) {
      return (el as HTMLElement).innerText?.trim().substring(0, 2000) || "";
    }
  }

  return "";
}

function extractAttributesFromDOM(): Record<string, string> {
  const attrs: Record<string, string> = {};

  const attrSelectors = [
    ".attributes-table tr",
    ".mod-detail-attributes tr",
    ".parameter-table tr",
    "[data-attr]",
    ".attr-item",
    ".prop-item",
    "table.attributes tr",
  ];

  for (const sel of attrSelectors) {
    const rows = document.querySelectorAll(sel);
    if (rows.length > 0) {
      rows.forEach((row) => {
        const cells = row.querySelectorAll("td, th");
        if (cells.length >= 2) {
          const key = cells[0].textContent?.trim().replace(/[：:]\s*$/, "") || "";
          const value = cells[1].textContent?.trim() || "";
          if (key && value) attrs[key] = value;
        }
      });
      break;
    }
  }

  return attrs;
}

function extractMOQFromDOM(): number {
  const bodyText = document.body.innerText;

  const moqMatch = bodyText.match(
    /(?:起订量|最小起订|MOQ|min\s*order)[：:]\s*(\d+)/i
  );
  if (moqMatch) return parseInt(moqMatch[1], 10);

  const batchMatch = bodyText.match(/(?:≥|>=|起批)\s*(\d+)/);
  if (batchMatch) return parseInt(batchMatch[1], 10);

  return 1;
}

function extractPackageFromDOM(): {
  weight: number | null;
  length: number | null;
  width: number | null;
  height: number | null;
} {
  const result = {
    weight: null as number | null,
    length: null as number | null,
    width: null as number | null,
    height: null as number | null,
  };

  const bodyText = document.body.innerText;

  // Weight: 毛重, 净重, package weight
  const wMatch = bodyText.match(
    /(?:毛重|重量|净重|package\s*weight)[：:]\s*(\d+\.?\d*)\s*(kg|公斤|g)/i
  );
  if (wMatch) {
    result.weight = parseFloat(wMatch[1]);
    if (wMatch[2].toLowerCase() === "g") result.weight /= 1000;
  }

  // Dimensions: 包装尺寸, 外箱尺寸, volume, package size
  const dMatch = bodyText.match(
    /(?:包装尺寸|外箱尺寸|体积|package\s*size|dimensions?)[：:]\s*(\d+\.?\d*)\s*[*×xX]\s*(\d+\.?\d*)\s*[*×xX]\s*(\d+\.?\d*)/i
  );
  if (dMatch) {
    result.length = parseFloat(dMatch[1]);
    result.width = parseFloat(dMatch[2]);
    result.height = parseFloat(dMatch[3]);
  }

  return result;
}

// ─── Main extraction orchestrator ──────────────────────────────────────────

function extractPageData(): PageData {
  const supplier = extractSupplierFromDOM();
  const dimensions = extractPackageFromDOM();
  const images = extractImagesFromDOM();
  const variants = extractSpecVariantsFromDOM();
  const price = extractPriceFromDOM();
  const attrs = extractAttributesFromDOM();

  // Resolve price range from variants
  let priceMin: number | null = null;
  let priceMax: number | null = null;
  const variantPrices = variants.filter((v) => v.price > 0).map((v) => v.price);
  if (variantPrices.length > 0) {
    priceMin = Math.min(...variantPrices);
    priceMax = Math.max(...variantPrices);
  }

  // Build final PageData
  const data: PageData = {
    source_url: window.location.href,
    collected_at: new Date().toISOString(),
    driver: "plugin",
    title: extractTitleFromDOM(),
    price_1688: price,
    price_min: priceMin,
    price_max: priceMax,
    currency: "CNY",
    min_order_qty: extractMOQFromDOM(),
    images,
    spec_variants: variants.length > 0 ? variants : undefined,
    supplier_name: supplier.name,
    supplier_id_1688: supplier.id,
    supplier_score: supplier.score,
    description: extractDescriptionFromDOM(),
    attributes: Object.keys(attrs).length > 0 ? attrs : undefined,
    package_weight_kg: dimensions.weight,
    package_length_cm: dimensions.length,
    package_width_cm: dimensions.width,
    package_height_cm: dimensions.height,
  };

  // Try to improve with embedded JSON
  try {
    const embedded = tryEmbeddedJSON();
    if (embedded) {
      // Merge: embedded overrides DOM values when present
      if (embedded.title) data.title = embedded.title;
      if (embedded.price_1688 && embedded.price_1688 > 0) data.price_1688 = embedded.price_1688;
      if (embedded.images && embedded.images.length > 0) data.images = embedded.images;
      if (embedded.spec_variants && embedded.spec_variants.length > 0) {
        data.spec_variants = embedded.spec_variants;
        const ep = embedded.spec_variants
          .filter((v) => v.price > 0)
          .map((v) => v.price);
        if (ep.length > 0) {
          data.price_min = Math.min(...ep);
          data.price_max = Math.max(...ep);
        }
      }
      if (embedded.supplier_name) data.supplier_name = embedded.supplier_name;
      if (embedded.supplier_id_1688) data.supplier_id_1688 = embedded.supplier_id_1688;
      if (embedded.description) data.description = embedded.description;
      if (embedded.price_min !== undefined) data.price_min = embedded.price_min;
      if (embedded.price_max !== undefined) data.price_max = embedded.price_max;
    }
  } catch {
    // Embedded JSON is optional enhancement
  }

  return data;
}

// ─── Messaging handlers ────────────────────────────────────────────────────

/** Handle on-demand fetch requests from the background service worker. */
chrome.runtime.onMessage.addListener(
  (
    message: ContentScriptFetchRequest,
    _sender: chrome.runtime.MessageSender,
    sendResponse: (response: ContentScriptFetchResult) => void
  ) => {
    if (message.type === "fetch_product_from_page") {
      try {
        const data = extractPageData();
        sendResponse({
          type: "fetch_product_from_page_result",
          requestId: message.requestId,
          payload: { status: "ok", data },
        });
      } catch (err) {
        sendResponse({
          type: "fetch_product_from_page_result",
          requestId: message.requestId,
          payload: {
            code: "EXTRACTION_FAILED",
            message: err instanceof Error ? err.message : "Unknown extraction error",
          },
        });
      }
      return true; // Keep message channel open for async response
    }
  }
);

// ─── Auto-extraction on page load ──────────────────────────────────────────

/**
 * On page load, auto-extract product data and send to background.
 * This enables proactive collection when the user browses 1688.
 */
function autoExtract() {
  // Small delay to ensure dynamic content has loaded
  setTimeout(() => {
    try {
      const data = extractPageData();
      // Only send if we got meaningful data
      if (data.title) {
        chrome.runtime.sendMessage({
          type: "fetch_product_from_page_result",
          requestId: "auto_" + Date.now() + "_" + Math.random().toString(36).slice(2, 8),
          payload: { status: "ok", data },
        } satisfies ContentScriptFetchResult).catch(() => {
          // Background may not be ready yet — that's OK
        });
      }
    } catch {
      // Silent — auto-extraction is best-effort
    }
  }, 1500);
}

// Trigger auto-extraction
if (document.readyState === "complete") {
  autoExtract();
} else {
  window.addEventListener("load", autoExtract);
}
