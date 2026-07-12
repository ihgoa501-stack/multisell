/**
 * Content script injected into 1688 product detail pages (detail.1688.com/offer/*).
 *
 * Extracts structured product data on page load and on demand.
 * Communicates with the background service worker via chrome.runtime.
 */

// ─── Type definitions (self-contained — no module imports for content scripts) ─

interface PageData {
	schema_version: "sourcing1688.private.v1";
	offer_id_url: string;
	offer_id_page: string;
  source_url: string;
  collected_at: string;
  driver: string;
  parser_version: string;
  title: string;
  price_1688: number;
	price_model: "fixed" | "range" | "tiered" | "sku" | "unknown";
  price_min?: number | null;
  price_max?: number | null;
  currency: string;
  min_order_qty: number;
  images: string[];
  spec_variants?: SpecVariant[];
  supplier_name: string;
  supplier_id_1688: string;
  supplier_business_id: string;
  supplier_score?: number | null;
  description?: string;
  attributes?: Record<string, string>;
  package_weight_kg?: number | null;
  package_length_cm?: number | null;
  package_width_cm?: number | null;
  package_height_cm?: number | null;
  freight_cny?: number | null;
	field_statuses: Record<string, "observed" | "unknown" | "parse_failed" | "no_sku">;
}

interface ExistingPrivateCollectionSummary {
  title: string | null;
  price: number | null;
  moq: number | null;
  supplier_name: string;
  sku_count: number;
  image_count: number;
  observed_at: string;
}

interface SpecVariant {
  spec: string;
  price?: number;
  stock?: number;
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

interface CollectPrivateProductResponse {
  type: "private_collection_result";
  requestId: string;
  payload:
		| { status: "saved"; recordId: number; snapshotId: number; idempotentReplay: boolean; newObservation: boolean }
			| { status: "duplicate_requires_choice"; recordId: number; snapshotId: number; message: string; saved: false; existing: ExistingPrivateCollectionSummary }
    | { status: "not_saved"; code: string; message: string; saved: false }
    | { status: "reconcile_required"; code: string; message: string; saved: false }
    | { status: "failed"; code: string; message: string; saved: false };
}

type PageBlockCode =
  | "NOT_PRODUCT_PAGE"
  | "LOGIN_REQUIRED"
  | "RISK_CHALLENGE"
  | "OFFER_UNAVAILABLE"
  | "PAGE_LOADING"
  | "SKU_UNSTABLE";

interface PageBlockReason {
  code: PageBlockCode;
  happened: string;
  nextStep: string;
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
		const price = Number.parseFloat(item.price ?? item.skuPrice);
		const stock = Number.parseInt(item.stock ?? item.quantity, 10);
        variants.push({
          spec: item.spec || item.name || item.skuName || "",
		  ...(Number.isFinite(price) ? { price } : {}),
		  ...(Number.isFinite(stock) ? { stock } : {}),
          image_url: item.image || item.imageUrl || undefined,
        });
      }
    } else if (typeof skuData === "object") {
      for (const [key, val] of Object.entries(skuData as Record<string, any>)) {
		const price = Number.parseFloat(val.price ?? val.skuPrice);
		const stock = Number.parseInt(val.stock ?? val.quantity, 10);
		variants.push({
          spec: key,
		  ...(Number.isFinite(price) ? { price } : {}),
		  ...(Number.isFinite(stock) ? { stock } : {}),
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
      const stockStr = el.getAttribute("data-stock") || el.getAttribute("data-quantity") || "";
      const parsedPrice = parseFloat(priceStr);
      const parsedStock = Number.parseInt(stockStr, 10);
      const imageURL = el.getAttribute("data-image") || el.getAttribute("data-image-url") || undefined;
      if (spec) {
        // Check if already added
        if (!variants.some((v) => v.spec === spec)) {
          variants.push({
            spec,
            ...(Number.isFinite(parsedPrice) ? { price: parsedPrice } : {}),
            ...(Number.isFinite(parsedStock) ? { stock: parsedStock } : {}),
            ...(imageURL ? { image_url: imageURL } : {}),
          });
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
            variants.push({ spec });
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

  return 0;
}

function extractOfferIdentity(): { url: string; page: string } {
	const url = window.location.href.match(/\/offer\/(\d+)\.html/i)?.[1] || "";
	const declared = document.querySelector<HTMLElement>("[data-offer-id], [data-offerid]");
	const attribute = declared?.getAttribute("data-offer-id") || declared?.getAttribute("data-offerid") || "";
	if (/^\d+$/.test(attribute)) return { url, page: attribute };
	const canonical = document.querySelector<HTMLLinkElement>('link[rel="canonical"]')?.href ||
		document.querySelector<HTMLMetaElement>('meta[property="og:url"]')?.content || "";
	const canonicalID = canonical.match(/\/offer\/(\d+)\.html/i)?.[1] || "";
	if (canonicalID) return { url, page: canonicalID };
	for (const script of Array.from(document.scripts)) {
		const match = (script.textContent || "").match(/["'](?:offerId|offer_id)["']\s*:\s*["']?(\d+)/i);
		if (match) return { url, page: match[1] };
	}
	return { url, page: "" };
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
  const variantPrices = variants.map((v) => v.price).filter((v): v is number => typeof v === "number" && v > 0);
  if (variantPrices.length > 0) {
    priceMin = Math.min(...variantPrices);
    priceMax = Math.max(...variantPrices);
  }

  // Build final PageData
	const offerIdentity = extractOfferIdentity();
  const data: PageData = {
	schema_version: "sourcing1688.private.v1",
	offer_id_url: offerIdentity.url,
	offer_id_page: offerIdentity.page,
    source_url: window.location.href,
    collected_at: new Date().toISOString(),
    driver: "plugin",
    parser_version: "lingmirror-extension@0.2.0",
    title: extractTitleFromDOM(),
    price_1688: price,
	price_model: "unknown",
    price_min: priceMin,
    price_max: priceMax,
    currency: "CNY",
    min_order_qty: extractMOQFromDOM(),
    images,
    spec_variants: variants.length > 0 ? variants : undefined,
    supplier_name: supplier.name,
    supplier_id_1688: supplier.id,
    supplier_business_id: supplier.id,
    supplier_score: supplier.score,
    description: extractDescriptionFromDOM(),
    attributes: Object.keys(attrs).length > 0 ? attrs : undefined,
    package_weight_kg: dimensions.weight,
    package_length_cm: dimensions.length,
    package_width_cm: dimensions.width,
    package_height_cm: dimensions.height,
	field_statuses: {
		title: extractTitleFromDOM() ? "observed" : "parse_failed",
		price: price > 0 ? "observed" : "unknown",
		moq: extractMOQFromDOM() > 0 ? "observed" : "unknown",
		supplier: supplier.name || supplier.id ? "observed" : "unknown",
		sku: variants.length === 0 ? "no_sku" : variants.some((v) => v.price === undefined || v.stock === undefined) ? "parse_failed" : "observed",
	},
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
          .map((v) => v.price)
          .filter((v): v is number => typeof v === "number" && v > 0);
        if (ep.length > 0) {
          data.price_min = Math.min(...ep);
          data.price_max = Math.max(...ep);
        }
      }
      if (embedded.supplier_name) data.supplier_name = embedded.supplier_name;
		if (embedded.supplier_id_1688) {
			data.supplier_id_1688 = embedded.supplier_id_1688;
			data.supplier_business_id = embedded.supplier_id_1688;
		}
      if (embedded.description) data.description = embedded.description;
      if (embedded.price_min !== undefined) data.price_min = embedded.price_min;
      if (embedded.price_max !== undefined) data.price_max = embedded.price_max;
    }
  } catch {
    // Embedded JSON is optional enhancement
  }
	const visibleSKUControls = Boolean(document.querySelector("[data-sku], [data-skuid], .sku-item, [class*='sku'] li")) || /规格|颜色分类|尺码/.test(document.body.innerText);
	const visibleTierPrices = Boolean(document.querySelector("[class*='ladder-price'], [class*='price-range'], [class*='volume-price']")) || /\d+\s*(?:件|个|套)\s*(?:起|以上).*?[¥￥]\s*\d/.test(document.body.innerText);
	data.price_model = visibleTierPrices ? "tiered" : data.spec_variants?.some((variant) => typeof variant.price === "number") ? "sku"
		: (data.price_min ?? 0) > 0 && (data.price_max ?? 0) > 0 && data.price_min !== data.price_max ? "range"
		: data.price_1688 > 0 ? "fixed" : "unknown";
	data.field_statuses = {
		title: data.title ? "observed" : "parse_failed",
		price: data.price_1688 > 0 || (data.price_min ?? 0) > 0 || (data.price_max ?? 0) > 0 ? "observed" : "unknown",
		moq: data.min_order_qty > 0 ? "observed" : "unknown",
		supplier: data.supplier_name || data.supplier_business_id ? "observed" : "unknown",
		images: data.images.length > 0 ? "observed" : "unknown",
		sku: !data.spec_variants?.length ? (visibleSKUControls ? "parse_failed" : "no_sku")
			: data.spec_variants.some((variant) => variant.price === undefined || variant.stock === undefined) ? "parse_failed" : "observed",
	};

  return data;
}

/**
 * Fail closed before parsing. These signals deliberately use only the current
 * DOM and URL: a blocked page is never sent to the background worker.
 */
function detectPageBlock(): PageBlockReason | null {
  const url = new URL(window.location.href);
  if (url.hostname !== "detail.1688.com" || !/^\/offer\/\d+\.html$/i.test(url.pathname)) {
    return {
      code: "NOT_PRODUCT_PAGE",
      happened: "当前页面不是1688商品详情页。",
      nextStep: "请打开网址形如 detail.1688.com/offer/数字.html 的商品页后再采集。",
    };
  }

  const bodyText = (document.body?.innerText || document.body?.textContent || "").replace(/\s+/g, " ");
  const has = (selector: string): boolean => Boolean(document.querySelector(selector));

  const riskSelector = has(
    "iframe[src*='captcha' i], iframe[src*='verify' i], [class*='captcha' i], [id*='captcha' i], " +
    "[class*='risk-control' i], [data-page-state='risk'], [data-page-state='captcha']",
  );
  if (riskSelector || /请完成验证|滑动验证|安全验证|访问过于频繁|账号存在风险|访问被拒绝/.test(bodyText)) {
    return {
      code: "RISK_CHALLENGE",
      happened: "1688正在显示验证码或访问风控，本次读取已停止。",
      nextStep: "请你在页面上人工完成验证；插件不会自动重试，确认恢复为商品详情页后再点击采集。",
    };
  }

  const loginSelector = has(
    "iframe[src*='login' i], [data-page-state='login'], .login-dialog, .login-modal, [class*='login-mask' i]",
  );
  if (loginSelector || /请先登录(?:1688)?|登录后查看|登录后可见|账号登录后/.test(bodyText)) {
    return {
      code: "LOGIN_REQUIRED",
      happened: "当前1688页面要求登录，商品关键内容还不能读取。",
      nextStep: "请先在1688页面完成登录，看到正常商品标题、价格和规格后再点击采集。",
    };
  }

  const unavailableSelector = has(
    "[data-page-state='offsale'], [data-page-state='unavailable'], .offer-offline, .offer-not-found, .error-placeholder",
  );
  if (unavailableSelector || /商品已下架|该商品不存在|商品已失效|页面不存在|宝贝已删除|暂无法查看该商品/.test(bodyText)) {
    return {
      code: "OFFER_UNAVAILABLE",
      happened: "当前是商品下架、失效或占位页面，不是可采集的商品详情。",
      nextStep: "请返回1688选择一个仍可正常查看的商品；当前页面不能保存到采集箱。",
    };
  }

  const pageLoading = document.readyState === "loading" || has(
    "[data-page-loading='true'], [data-page-state='loading'], main[aria-busy='true'], .page-loading, .detail-skeleton",
  );
  if (pageLoading) {
    return {
      code: "PAGE_LOADING",
      happened: "商品页面仍在加载，标题、价格或供应商信息还没有完成。",
      nextStep: "请等待页面停止转圈并显示完整商品内容，然后重新点击采集。",
    };
  }

  const skuUnstable = has(
    "[data-sku-loading='true'], [data-sku-stable='false'], [data-page-state='sku-loading'], .sku-loading, .sku-skeleton, [aria-label='规格'][aria-busy='true']",
  );
  if (skuUnstable) {
    return {
      code: "SKU_UNSTABLE",
      happened: "商品规格或规格价格仍在变化，当前SKU还没有稳定。",
      nextStep: "请等规格区停止加载，确认所选规格和价格不再变化后重新点击采集。",
    };
  }

  return null;
}

// ─── Messaging handlers ────────────────────────────────────────────────────

/** Handle on-demand fetch requests from the background service worker. */
chrome.runtime.onMessage.addListener(
  (
    message: ContentScriptFetchRequest,
    sender: chrome.runtime.MessageSender,
    sendResponse: (response: ContentScriptFetchResult) => void
  ) => {
    if (sender.id !== chrome.runtime.id) return;
    if ((message as unknown as { type?: string }).type === "collect_private_product_from_page") {
      void collectCurrentPage().then(sendResponse as unknown as (response: CollectPrivateProductResponse) => void);
      return true;
    }
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

// ─── Owner-triggered private collection UI ─────────────────────────────────

let collecting = false;
let collectorPanel: HTMLDivElement | null = null;
let collectorBody: HTMLDivElement | null = null;
let collectorButton: HTMLButtonElement | null = null;
let pendingPreview: { requestId: string; pageData: PageData; observationIntent?: "save_new_observation" } | null = null;

function newCollectionRequestId(): string {
  const id = typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : `${Date.now()}_${Math.random().toString(36).slice(2)}`;
  return `collect_${id}`;
}

function setCollectorBody(lines: string[], tone: "normal" | "success" | "error" = "normal"): void {
  if (!collectorBody) return;
  collectorBody.replaceChildren();
  collectorBody.style.color = tone === "error" ? "#b42318" : tone === "success" ? "#067647" : "#344054";
  for (const line of lines) {
    const row = document.createElement("div");
    row.textContent = line;
    row.style.marginTop = "8px";
    collectorBody.appendChild(row);
  }
}

function showPageBlock(reason: PageBlockReason): void {
  pendingPreview = null;
  setCollectorBody([
    `发生了什么：${reason.happened}`,
    "是否保存：没有保存，也没有上传失败页面内容。",
    `下一步：${reason.nextStep}`,
  ], "error");
}

function openCollectorPanel(): void {
  if (collectorPanel) collectorPanel.style.display = "block";
}

function collectionIdentity(data: PageData): string {
  const offer = data.source_url.match(/\/offer\/(\d+)\.html/i)?.[1] || "";
  const sku = (data.spec_variants || []).map((item) => ({ spec: item.spec, price: item.price, stock: item.stock, image: item.image_url || "" }));
  const selectedSKU = Array.from(document.querySelectorAll("[aria-checked='true'], [aria-selected='true'], [data-selected='true']"))
    .slice(0, 20)
    .map((node) => ({ id: node.getAttribute("data-skuid") || node.getAttribute("data-sku-id") || "", text: (node.textContent || "").trim().slice(0, 120) }));
  return JSON.stringify({ offer, title: data.title, supplier: data.supplier_business_id || data.supplier_name,
    price: data.price_1688, priceMin: data.price_min, priceMax: data.price_max, moq: data.min_order_qty, sku, selectedSKU });
}

function previewLines(pageData: PageData): string[] {
  const statusLabels: Record<string, string> = { price: "价格", moq: "起订量", supplier: "供应商身份", images: "主图", sku: "SKU" };
  const missing = Object.entries(pageData.field_statuses)
    .filter(([field, status]) => field !== "title" && (status === "unknown" || status === "parse_failed"))
    .map(([field, status]) => `${statusLabels[field] || field}${status === "parse_failed" ? "解析失败" : "未取得"}`);
  return [
    "请确认后再保存：",
    `商品：${pageData.title}`,
    `价格：${pageData.price_min && pageData.price_max ? `¥${pageData.price_min}–¥${pageData.price_max}` : pageData.price_1688 > 0 ? `¥${pageData.price_1688}` : "未取得"}`,
    `起订量：${pageData.min_order_qty > 0 ? `${pageData.min_order_qty}件` : "未取得"}`,
    `供应商：${pageData.supplier_name || "未取得稳定身份"}`,
    `SKU：${pageData.spec_variants?.length || 0}　图片：${pageData.images.length}`,
    `缺失/异常：${missing.length > 0 ? missing.join("、") : "无"}`,
  ];
}

function appendPreviewActions(): void {
  if (!collectorBody || !pendingPreview) return;
  const actions = document.createElement("div");
  actions.style.marginTop = "14px";
  const confirm = document.createElement("button");
  confirm.type = "button";
  confirm.textContent = "确认保存";
  Object.assign(confirm.style, { border: "0", borderRadius: "8px", padding: "8px 12px", cursor: "pointer", background: "#4f46e5", color: "white" });
  const cancel = document.createElement("button");
  cancel.type = "button";
  cancel.textContent = "取消";
  Object.assign(cancel.style, { marginLeft: "8px", border: "1px solid #d0d5dd", borderRadius: "8px", padding: "7px 12px", cursor: "pointer", background: "white" });
  confirm.addEventListener("click", () => void submitPendingPreview());
  cancel.addEventListener("click", () => {
    pendingPreview = null;
    setCollectorBody(["已取消，本次没有上传或保存"]);
  });
  actions.append(confirm, cancel);
  collectorBody.appendChild(actions);
}

function duplicateComparisonLines(page: PageData, existing: ExistingPrivateCollectionSummary): string[] {
	const same = (current: unknown, previous: unknown) => current === previous ? "相同" : "有变化";
	const text = (value: string | null | undefined) => value?.trim() || "未取得";
	const number = (value: number | null | undefined, suffix = "") =>
		Number.isFinite(value) && (value as number) > 0 ? `${value}${suffix}` : "未取得";
	const currentPrice = page.price_1688 > 0 ? page.price_1688 : null;
	const currentMOQ = page.min_order_qty > 0 ? page.min_order_qty : null;
	const currentSupplier = page.supplier_name?.trim() || "";
	const rows: Array<[string, unknown, unknown, string, string]> = [
		["标题", page.title.trim(), existing.title?.trim() || "", text(page.title), text(existing.title)],
		["价格", currentPrice, existing.price, number(currentPrice, "元"), number(existing.price, "元")],
		["起订量", currentMOQ, existing.moq, number(currentMOQ, "件"), number(existing.moq, "件")],
		["供应商", currentSupplier, existing.supplier_name?.trim() || "", text(currentSupplier), text(existing.supplier_name)],
		["SKU数", page.spec_variants?.length || 0, existing.sku_count, String(page.spec_variants?.length || 0), String(existing.sku_count)],
		["图片数", page.images?.length || 0, existing.image_count, String(page.images?.length || 0), String(existing.image_count)],
	];
	return ["本次页面 vs 已有观察：", ...rows.map(([label, current, previous, currentText, previousText]) =>
		`${label}：本次 ${currentText}｜已有 ${previousText}（${same(current, previous)}）`),
		`已有观察时间：${existing.observed_at || "未取得"}`];
}

function showDuplicateChoice(recordId: number, pageData: PageData, existing: ExistingPrivateCollectionSummary): void {
	setCollectorBody(["这个1688商品已经在私人采集箱中。", ...duplicateComparisonLines(pageData, existing), "请选择查看已有记录，或把当前页面保存为一条新的观察。"]);
	if (!collectorBody) return;
	const view = document.createElement("button");
	view.type = "button";
	view.textContent = "查看已有记录";
	Object.assign(view.style, { border: "1px solid #d0d5dd", borderRadius: "8px", padding: "7px 12px", cursor: "pointer", background: "white", marginTop: "12px" });
	view.addEventListener("click", () => void chrome.runtime.sendMessage({ type: "open_private_collection", recordId }));
	const save = document.createElement("button");
	save.type = "button";
	save.textContent = "保存为新观察";
	Object.assign(save.style, { border: "0", borderRadius: "8px", padding: "8px 12px", cursor: "pointer", background: "#4f46e5", color: "white", margin: "12px 0 0 8px" });
	save.addEventListener("click", () => {
		pendingPreview = { requestId: newCollectionRequestId(), pageData, observationIntent: "save_new_observation" };
		void submitPendingPreview();
	});
	collectorBody.append(view, save);
}

async function submitPendingPreview(): Promise<void> {
  if (!pendingPreview || collecting) return;
  collecting = true;
  if (collectorButton) collectorButton.disabled = true;
  const preview = pendingPreview;
  try {
    const pageBlock = detectPageBlock();
    if (pageBlock) {
      showPageBlock(pageBlock);
      return;
    }
    const latest = extractPageData();
    if (collectionIdentity(latest) !== collectionIdentity(preview.pageData)) {
      pendingPreview = { requestId: newCollectionRequestId(), pageData: latest };
      setCollectorBody(["页面关键内容已经变化，请重新确认。", ...previewLines(latest)], "error");
      appendPreviewActions();
      return;
    }
    setCollectorBody(["正在保存到凌镜私人采集箱……", `商品：${latest.title}`]);
		const response = await chrome.runtime.sendMessage({ type: "collect_private_product", requestId: preview.requestId, pageData: latest, observationIntent: preview.observationIntent }) as CollectPrivateProductResponse;
		if (response?.payload?.status === "duplicate_requires_choice") {
			pendingPreview = null;
			showDuplicateChoice(response.payload.recordId, latest, response.payload.existing);
			return;
		}
    if (response?.payload?.status !== "saved") {
      const message = response?.payload && "message" in response.payload
        ? response.payload.message
        : "凌镜没有返回保存结果；当前商品未确认保存";
      throw new Error(message);
    }
    pendingPreview = null;
    setCollectorBody(["已保存到凌镜私人采集箱", `记录编号：#${response.payload.recordId}`,
      response.payload.idempotentReplay ? "该请求已保存，本次返回原记录" : "你可以继续浏览，稍后到凌镜统一整理"], "success");
  } catch (err) {
    const message = err instanceof Error ? err.message : "采集失败；当前商品未保存";
    setCollectorBody([message, message.includes("待确认") ? "请勿重复点击，恢复连接后按本次请求核对" : "修复提示的问题后重新读取页面"], "error");
  } finally {
    collecting = false;
    if (collectorButton) collectorButton.disabled = false;
  }
}

async function collectCurrentPage(): Promise<CollectPrivateProductResponse> {
  if (collecting) {
    return {
      type: "private_collection_result",
      requestId: "",
      payload: { status: "failed", code: "ALREADY_RUNNING", message: "当前商品正在采集，请等待结果", saved: false },
    };
  }
  openCollectorPanel();
  setCollectorBody(["正在读取当前1688商品……"]);
  const requestId = newCollectionRequestId();
  try {
    const pageBlock = detectPageBlock();
    if (pageBlock) {
      showPageBlock(pageBlock);
      return {
        type: "private_collection_result",
        requestId,
        payload: { status: "failed", code: pageBlock.code, message: pageBlock.happened, saved: false },
      };
    }
    const pageData = extractPageData();
	if (!pageData.title) {
      void reportPrivateFailure(requestId, pageData, "title_parse_failed");
      throw new Error("没有读取到商品标题，本次未保存。请刷新商品详情页后重试");
    }
	if (!pageData.offer_id_url || !pageData.offer_id_page || pageData.offer_id_url !== pageData.offer_id_page) {
      void reportPrivateFailure(requestId, pageData, "invalid_source_url");
      throw new Error("无法确认当前1688商品身份，本次未保存。请刷新商品详情页后重试");
    }
    if (pageData.field_statuses.sku === "parse_failed") {
      void reportPrivateFailure(requestId, pageData, "sku_parse_failed");
    }
    pendingPreview = { requestId, pageData };
    setCollectorBody(previewLines(pageData));
    appendPreviewActions();
    return { type: "private_collection_result", requestId,
      payload: { status: "failed", code: "PREVIEW_REQUIRED", message: "请在页面预览中确认保存", saved: false } };
  } catch (err) {
    const message = err instanceof Error ? err.message : "采集失败；当前商品未保存";
    setCollectorBody([message, "如果已经登录，请刷新页面后重试；不需要打开控制台"], "error");
    return {
      type: "private_collection_result",
      requestId,
      payload: { status: "failed", code: "COLLECTION_FAILED", message, saved: false },
    };
  }
}

async function reportPrivateFailure(
  requestId: string,
  pageData: PageData,
  errorCode: "invalid_source_url" | "title_parse_failed" | "sku_parse_failed" | "invalid_payload",
): Promise<void> {
  await chrome.runtime.sendMessage({
    type: "record_private_capture_failure",
    failure: {
      requestId,
      sourceUrl: pageData.source_url || location.href,
      errorCode,
      schemaVersion: pageData.schema_version,
      extensionVersion: chrome.runtime.getManifest().version,
      parserVersion: pageData.parser_version,
      occurredAt: new Date().toISOString(),
    },
  }).catch(() => undefined);
}

function installCollectorUI(): void {
  if (document.getElementById("lingmirror-private-collector")) return;
  const host = document.createElement("div");
  host.id = "lingmirror-private-collector";

  const button = document.createElement("button");
  button.type = "button";
  button.textContent = "采集到凌镜";
  button.setAttribute("aria-label", "将当前1688商品采集到凌镜私人采集箱");
  Object.assign(button.style, {
    position: "fixed", right: "20px", bottom: "24px", zIndex: "2147483646",
    border: "0", borderRadius: "10px", padding: "11px 16px", cursor: "pointer",
    background: "#4f46e5", color: "white", fontSize: "14px", fontWeight: "600",
    boxShadow: "0 6px 18px rgba(16,24,40,.22)",
  });
  button.addEventListener("click", () => void collectCurrentPage());
  collectorButton = button;

  const panel = document.createElement("div");
  Object.assign(panel.style, {
    display: "none", position: "fixed", right: "20px", bottom: "78px", zIndex: "2147483646",
    width: "340px", maxWidth: "calc(100vw - 40px)", maxHeight: "65vh", overflow: "auto",
    background: "white", border: "1px solid #e4e7ec", borderRadius: "12px", padding: "16px",
    boxShadow: "0 16px 40px rgba(16,24,40,.22)", color: "#101828", fontFamily: "system-ui, sans-serif",
  });
  const header = document.createElement("div");
  header.textContent = "凌镜采集助手";
  header.style.fontWeight = "700";
  const close = document.createElement("button");
  close.type = "button";
  close.textContent = "收起";
  Object.assign(close.style, { float: "right", border: "0", background: "transparent", cursor: "pointer", color: "#475467" });
  close.addEventListener("click", () => { panel.style.display = "none"; });
  header.appendChild(close);
  const body = document.createElement("div");
  body.style.fontSize = "13px";
  panel.append(header, body);
  collectorPanel = panel;
  collectorBody = body;

  host.append(panel, button);
  document.documentElement.appendChild(host);
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", installCollectorUI, { once: true });
} else {
  installCollectorUI();
}
