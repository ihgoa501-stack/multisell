/**
 * 凌镜 AI 选品助手 — Content Script
 *
 * Runs on 1688 product detail pages and Taobao item pages.
 * Extracts structured product data and sends it to the background script.
 *
 * Extraction strategies (in priority order):
 *   1. Embedded JSON (<script type="application/ld+json">, __NUXT__, __INITIAL_STATE__)
 *   2. DOM selectors (multiple fallback chains per field)
 *   3. Meta tags (og:title, og:image, meta description)
 */

// ─── Extraction Result Shape ──────────────────────────────────────────────

/**
 * @typedef {Object} PageData
 * @property {string} source_url
 * @property {string} collected_at  - ISO 8601 timestamp
 * @property {string} driver - "extension"
 * @property {string} title
 * @property {number} price_1688
 * @property {number} [price_min]
 * @property {number} [price_max]
 * @property {string} currency
 * @property {number} min_order_qty
 * @property {string[]} images
 * @property {SpecVariant[]} [spec_variants]
 * @property {string} supplier_name
 * @property {string} supplier_id_1688
 * @property {number|null} [supplier_score]
 * @property {string} [description]
 * @property {Object<string,string>} [attributes]
 * @property {number|null} [package_weight_kg]
 * @property {number|null} [package_length_cm]
 * @property {number|null} [package_width_cm]
 * @property {number|null} [package_height_cm]
 * @property {number|null} [freight_cny]
 */

/**
 * @typedef {Object} SpecVariant
 * @property {string} spec
 * @property {number} price
 * @property {number} stock
 * @property {string} [image_url]
 */

// ─── Embedded JSON Extraction ─────────────────────────────────────────────

/**
 * Try to extract product data from embedded JSON in script tags.
 * 1688 often embeds data in application/ld+json, __NUXT__, or __INITIAL_STATE__.
 * @returns {Partial<PageData>|null}
 */
function tryEmbeddedJSON() {
  var scripts = document.querySelectorAll('script');
  for (var i = 0; i < scripts.length; i++) {
    var script = scripts[i];
    var text = script.textContent || '';

    // Try application/ld+json or application/json
    var type = script.getAttribute('type') || '';
    if (type === 'application/ld+json' || type === 'application/json') {
      try {
        var parsed = JSON.parse(text);
        if (parsed && (parsed.name || parsed.productName)) {
          return extractFromLDJSON(parsed);
        }
      } catch (_) {
        // continue
      }
    }

    // Try window.__NUXT__
    if (text.indexOf('__NUXT__') !== -1) {
      try {
        var match = text.match(/window\.__NUXT__\s*=\s*({.+?});/s);
        if (match) {
          var data = extractFromNuxt(JSON.parse(match[1]));
          if (data && data.title) return data;
        }
      } catch (_) {
        // continue
      }
    }

    // Try window.__INITIAL_STATE__
    if (text.indexOf('__INITIAL_STATE__') !== -1 || text.indexOf('skuMap') !== -1) {
      try {
        var match = text.match(/window\.__INITIAL_STATE__\s*=\s*({.+?});/s);
        if (match) {
          var data = extractFromInitialState(JSON.parse(match[1]));
          if (data && data.title) return data;
        }
      } catch (_) {
        // continue
      }
    }
  }
  return null;
}

/**
 * Extract data from an LD+JSON structured data block.
 * @param {Object} parsed
 * @returns {Partial<PageData>}
 */
function extractFromLDJSON(parsed) {
  var data = {};
  if (typeof parsed.name === 'string') data.title = parsed.name;
  if (typeof parsed.productName === 'string') data.title = parsed.productName;
  if (parsed.offers && typeof parsed.offers === 'object') {
    var offers = parsed.offers;
    if (typeof offers.price === 'number') data.price_1688 = offers.price;
    if (typeof offers.price === 'string') data.price_1688 = parseFloat(offers.price) || 0;
  }
  if (parsed.image) {
    if (typeof parsed.image === 'string') data.images = [parsed.image];
    if (Array.isArray(parsed.image)) data.images = parsed.image.filter(function (i) { return typeof i === 'string'; });
  }
  if (typeof parsed.description === 'string') data.description = parsed.description;
  return data;
}

/**
 * Navigate Nuxt state tree to find product data.
 * @param {Object} nuxt
 * @returns {Partial<PageData>}
 */
function extractFromNuxt(nuxt) {
  var data = {};
  try {
    var state = nuxt.state || nuxt;
    var detail = state.detail || state.productDetail || state.offerDetail || state;
    var offer = detail.offer || detail.product || detail;

    if (offer.subject || offer.title) data.title = offer.subject || offer.title;
    if (offer.price) data.price_1688 = parseFloat(offer.price) || 0;
    if (offer.priceRange && Array.isArray(offer.priceRange)) {
      var prices = offer.priceRange
        .map(function (p) { return parseFloat(p.price || p); })
        .filter(function (p) { return !isNaN(p); });
      if (prices.length > 0) {
        data.price_min = Math.min.apply(null, prices);
        data.price_max = Math.max.apply(null, prices);
        if (!data.price_1688) data.price_1688 = prices[0];
      }
    }
    if (offer.images && Array.isArray(offer.images)) {
      data.images = offer.images.map(function (img) {
        return typeof img === 'string' ? img : img.url || img.imageUrl || '';
      }).filter(Boolean);
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
  } catch (_) {
    // silent
  }
  return data;
}

/**
 * Extract data from __INITIAL_STATE__ embedded data.
 * @param {Object} state
 * @returns {Partial<PageData>}
 */
function extractFromInitialState(state) {
  var data = {};
  try {
    var s = state;
    var detail = s.detail || s.offerDetail || s.productDetail || s;
    var offer = detail.offer || detail.product || detail;

    if (offer.subject || offer.title) data.title = offer.subject || offer.title;
    if (offer.price) data.price_1688 = parseFloat(offer.price) || 0;
    if (offer.images && Array.isArray(offer.images)) {
      data.images = offer.images.map(function (img) {
        return typeof img === 'string' ? img : img.url;
      }).filter(Boolean);
    }
    if (offer.skuMap || offer.skuList) {
      data.spec_variants = extractSKUData(offer.skuMap || offer.skuList);
    }
  } catch (_) {
    // silent
  }
  return data;
}

/**
 * Parse SKU variant data from either an array or a key-value map.
 * @param {Object|Array} skuData
 * @returns {SpecVariant[]}
 */
function extractSKUData(skuData) {
  var variants = [];
  try {
    if (Array.isArray(skuData)) {
      for (var i = 0; i < skuData.length; i++) {
        var item = skuData[i];
        variants.push({
          spec: item.spec || item.name || item.skuName || '',
          price: parseFloat(item.price || item.skuPrice || '0') || 0,
          stock: parseInt(item.stock || item.quantity || '0', 10) || 0,
          image_url: item.image || item.imageUrl || undefined
        });
      }
    } else if (typeof skuData === 'object') {
      for (var key in skuData) {
        if (skuData.hasOwnProperty(key)) {
          var val = skuData[key];
          variants.push({
            spec: key,
            price: parseFloat(val.price || val.skuPrice || '0') || 0,
            stock: parseInt(val.stock || val.quantity || '0', 10) || 0,
            image_url: val.image || val.imageUrl || undefined
          });
        }
      }
    }
  } catch (_) {
    // silent
  }
  return variants;
}

// ─── DOM-Based Extraction (Fallback) ──────────────────────────────────────

/**
 * Extract product title from the DOM using multiple selectors.
 * @returns {string}
 */
function extractTitleFromDOM() {
  var selectors = [
    'h1[data-title]',
    '.detail-title h1',
    '.mod-detail h1',
    '.product-title',
    'h1.title',
    '.mod-detail-title',
    '.detail-title',
    '[data-component-title] h1'
  ];

  for (var i = 0; i < selectors.length; i++) {
    var el = document.querySelector(selectors[i]);
    if (el) {
      var text = el.innerText || el.textContent || '';
      text = text.trim();
      if (text) return text;
    }
  }

  // Meta fallback
  var og = document.querySelector('meta[property="og:title"]');
  if (og && og.content) return og.content.trim();

  var desc = document.querySelector('meta[name="description"]');
  if (desc && desc.content) {
    return desc.content.split(/[。.，,\n]/)[0].trim();
  }

  return (document.title || '').replace(/ - .+$/, '').trim();
}

/**
 * Extract the main price from the DOM.
 * @returns {number}
 */
function extractPriceFromDOM() {
  var selectors = [
    '[data-price]',
    '.price-con .price',
    '.detail-price .price',
    '.price',
    '.mod-price .price',
    '#mod-detail-price .price',
    '.offer-price',
    '.product-price',
    '[class*="price"]'
  ];

  for (var i = 0; i < selectors.length; i++) {
    var el = document.querySelector(selectors[i]);
    if (el) {
      var text = el.getAttribute('data-price') || el.textContent || '';
      text = text.trim();
      var match = text.match(/(\d+\.?\d*)/);
      if (match) return parseFloat(match[1]);
    }
  }

  // Look for ¥ pattern in body text
  var bodyText = document.body.innerText;
  var priceMatch = bodyText.match(/[¥￥]\s*(\d+\.?\d*)/);
  if (priceMatch) return parseFloat(priceMatch[1]);

  return 0;
}

/**
 * Extract main image URLs from the DOM.
 * @returns {string[]}
 */
function extractImagesFromDOM() {
  var images = [];
  var seen = {};

  function addUrl(src) {
    if (!src) return;
    var url = src.indexOf('//') === 0 ? 'https:' + src : src;
    if (url.indexOf('http') === 0 && url.indexOf('data:image') === -1 && !seen[url]) {
      seen[url] = true;
      images.push(url);
    }
  }

  var selectors = [
    '.image-item img',
    '#dt-tab img',
    '.detail-gallery img',
    '.gallery img',
    '.mod-detail-gallery img',
    '[class*="gallery"] img',
    '[class*="preview"] img',
    '.main-img img'
  ];

  for (var i = 0; i < selectors.length; i++) {
    var els = document.querySelectorAll(selectors[i]);
    if (els.length > 0) {
      for (var j = 0; j < els.length; j++) {
        var img = els[j];
        addUrl(img.src || img.getAttribute('data-src') || img.getAttribute('data-lazy-src'));
      }
      if (images.length > 0) break;
    }
  }

  // Meta fallback
  var og = document.querySelector('meta[property="og:image"]');
  if (og && og.content) addUrl(og.content);

  return images;
}

/**
 * Extract SKU specification variants from the DOM.
 * @returns {SpecVariant[]}
 */
function extractSpecVariantsFromDOM() {
  var variants = [];

  // Try data attributes
  var skuItems = document.querySelectorAll('[data-sku], [data-skuid]');
  if (skuItems.length > 0) {
    for (var i = 0; i < skuItems.length; i++) {
      var el = skuItems[i];
      var spec = (el.innerText || el.textContent || '').trim();
      var priceStr = el.getAttribute('data-price') || '';
      var price = parseFloat(priceStr) || 0;
      if (spec && !variants.some(function (v) { return v.spec === spec; })) {
        variants.push({ spec: spec, price: price, stock: 0 });
      }
    }
  }

  // Fallback: look for spec selector items
  if (variants.length === 0) {
    var specSelectors = [
      '.sku-item',
      '.prop-item',
      '.sku-name',
      '[class*="sku"] li',
      '.attr-item'
    ];

    for (var s = 0; s < specSelectors.length; s++) {
      var items = document.querySelectorAll(specSelectors[s]);
      if (items.length > 0) {
        for (var k = 0; k < items.length; k++) {
          var specText = (items[k].innerText || items[k].textContent || '').trim();
          if (specText && !variants.some(function (v) { return v.spec === specText; })) {
            variants.push({ spec: specText, price: 0, stock: 0 });
          }
        }
        break;
      }
    }
  }

  return variants;
}

/**
 * Extract supplier name and ID from the DOM.
 * @returns {{name: string, id: string, score: number|null}}
 */
function extractSupplierFromDOM() {
  var name = '';
  var id = '';
  var score = null;

  var nameSelectors = [
    '.company-name',
    '.mod-supplier__name',
    '.supplier-name',
    '.shop-name',
    '[data-supplier]',
    '.seller-info .name',
    '.shop-info .name',
    '.store-name',
    '[class*="supplier"]',
    '[class*="shop"]'
  ];

  for (var i = 0; i < nameSelectors.length; i++) {
    var el = document.querySelector(nameSelectors[i]);
    if (el) {
      name = (el.innerText || el.textContent || '').trim();
      if (name) break;
    }
  }

  // Try to extract supplier ID from links
  var allLinks = document.querySelectorAll('a[href*="1688.com"]');
  for (var l = 0; l < allLinks.length; l++) {
    var href = allLinks[l].href || '';
    var match = href.match(/companyid=(\d+)/) || href.match(/company\/(\d+)/);
    if (match) {
      id = match[1];
      break;
    }
  }

  return { name: name, id: id, score: score };
}

/**
 * Extract product description text.
 * @returns {string}
 */
function extractDescriptionFromDOM() {
  var meta = document.querySelector('meta[name="description"]');
  if (meta && meta.content) {
    return meta.content.trim().substring(0, 2000);
  }

  var selectors = [
    '.desc-content',
    '.detail-desc',
    '#description',
    '.mod-detail-description',
    '.attributes',
    '[class*="description"]',
    '[class*="detail"]'
  ];

  for (var i = 0; i < selectors.length; i++) {
    var el = document.querySelector(selectors[i]);
    if (el) {
      return (el.innerText || el.textContent || '').trim().substring(0, 2000);
    }
  }

  return '';
}

// ─── Main Orchestrator ────────────────────────────────────────────────────

/**
 * Extract all structured product data from the current page.
 * Uses embedded JSON first, then DOM selectors as fallback.
 * @returns {PageData}
 */
function extractPageData() {
  var supplier = extractSupplierFromDOM();
  var images = extractImagesFromDOM();
  var variants = extractSpecVariantsFromDOM();
  var price = extractPriceFromDOM();

  // Resolve price range from variants
  var variantPrices = [];
  for (var i = 0; i < variants.length; i++) {
    if (variants[i].price > 0) variantPrices.push(variants[i].price);
  }
  var priceMin = variantPrices.length > 0 ? Math.min.apply(null, variantPrices) : null;
  var priceMax = variantPrices.length > 0 ? Math.max.apply(null, variantPrices) : null;

  // Build the data object
  var data = {
    source_url: window.location.href,
    collected_at: new Date().toISOString(),
    driver: 'extension',
    title: extractTitleFromDOM(),
    price_1688: price,
    price_min: priceMin,
    price_max: priceMax,
    currency: 'CNY',
    min_order_qty: 1,
    images: images,
    spec_variants: variants.length > 0 ? variants : undefined,
    supplier_name: supplier.name,
    supplier_id_1688: supplier.id,
    supplier_score: supplier.score,
    description: extractDescriptionFromDOM(),
    attributes: undefined,
    package_weight_kg: null,
    package_length_cm: null,
    package_width_cm: null,
    package_height_cm: null,
    freight_cny: null
  };

  // Try to enhance with embedded JSON data
  try {
    var embedded = tryEmbeddedJSON();
    if (embedded) {
      if (embedded.title) data.title = embedded.title;
      if (embedded.price_1688 && embedded.price_1688 > 0) data.price_1688 = embedded.price_1688;
      if (embedded.images && embedded.images.length > 0) data.images = embedded.images;
      if (embedded.spec_variants && embedded.spec_variants.length > 0) {
        data.spec_variants = embedded.spec_variants;
        var ep = [];
        for (var j = 0; j < embedded.spec_variants.length; j++) {
          if (embedded.spec_variants[j].price > 0) ep.push(embedded.spec_variants[j].price);
        }
        if (ep.length > 0) {
          data.price_min = Math.min.apply(null, ep);
          data.price_max = Math.max.apply(null, ep);
        }
      }
      if (embedded.supplier_name) data.supplier_name = embedded.supplier_name;
      if (embedded.supplier_id_1688) data.supplier_id_1688 = embedded.supplier_id_1688;
      if (embedded.description) data.description = embedded.description;
      if (embedded.price_min !== undefined) data.price_min = embedded.price_min;
      if (embedded.price_max !== undefined) data.price_max = embedded.price_max;
    }
  } catch (_) {
    // Embedded JSON is best-effort enhancement
  }

  return data;
}

/**
 * Send extracted data to the background script.
 * @param {PageData} data
 * @param {string} [requestId] - Optional request correlation ID
 */
function sendResult(data, requestId) {
  var msg = {
    type: 'fetch_product_from_page_result',
    requestId: requestId || 'auto_' + Date.now() + '_' + Math.random().toString(36).slice(2, 8),
    payload: { status: 'ok', data: data }
  };
  chrome.runtime.sendMessage(msg).catch(function () {
    // Background may not be ready yet
  });
}

// ─── Message Handlers ─────────────────────────────────────────────────────

/**
 * Listen for on-demand fetch requests from the background service worker.
 * Sent when the user clicks "Fetch" in the popup or when the backend
 * sends a fetch_product command via WebSocket.
 */
chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
  if (message.type === 'fetch_product_from_page') {
    try {
      var data = extractPageData();
      sendResponse({
        type: 'fetch_product_from_page_result',
        requestId: message.requestId,
        payload: { status: 'ok', data: data }
      });
    } catch (err) {
      sendResponse({
        type: 'fetch_product_from_page_result',
        requestId: message.requestId,
        payload: {
          code: 'EXTRACTION_FAILED',
          message: err instanceof Error ? err.message : 'Unknown extraction error'
        }
      });
    }
    return true; // Keep message channel open for async response
  }
});

// ─── Auto-Extraction on Page Load ─────────────────────────────────────────

/**
 * Automatically extract product data when the page finishes loading.
 * This enables proactive data collection as the user browses 1688.
 */
function autoExtract() {
  setTimeout(function () {
    try {
      var data = extractPageData();
      if (data.title) {
        sendResult(data);
      }
    } catch (_) {
      // Auto-extraction is best-effort
    }
  }, 1500);
}

if (document.readyState === 'complete') {
  autoExtract();
} else {
  window.addEventListener('load', autoExtract);
}
