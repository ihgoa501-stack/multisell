import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import { fileURLToPath } from 'node:url';
import { parseHTML } from 'linkedom';

const here = dirname(fileURLToPath(import.meta.url));
const compiled = await readFile(resolve(here, '../build/content-script-list.js'), 'utf8');

function productCard(id, options = {}) {
  const title = options.title || `商品${id}`;
  const price = options.price === false ? '' : `<span class="price">¥${options.price || '8.80'}</span>`;
  const moq = options.moq ? `<span>${options.moq}件起批</span>` : '';
  const image = options.image === false ? '' : '<img src="https://cbu01.alicdn.com/img/card.jpg">';
  const company = options.company ? `<div class="company-name">${options.company}</div>` : '';
  const shopLink = options.shopId ? `<a href="https://${options.shopId}.1688.com">进入店铺</a>` : '';
  return `<li class="offer-item" ${options.hidden ? 'style="display:none"' : ''}>
    <a href="https://detail.1688.com/offer/${id}.html" title="${title}">${image}<span class="title">${title}</span></a>${price}${moq}${company}${shopLink}
  </li>`;
}

function loadList(html, responder, pageURL = 'https://s.1688.com/selloffer/offer_search.htm?keywords=test') {
  const { window } = parseHTML(`<html><body><ul id="results">${html}</ul></body></html>`);
  window.HTMLElement.prototype.getBoundingClientRect = () => ({ width: 200, height: 120, top: 20, left: 20, right: 220, bottom: 140 });
  Object.defineProperty(window, 'location', { value: new URL(pageURL), configurable: true });
  const messages = [];
  const messageListeners = [];
  const chrome = {
    runtime: {
      sendMessage: async (message) => {
        messages.push(message);
        return responder ? responder(message) : { type: 'private_collection_result', requestId: message.requestId,
          payload: { status: 'saved', recordId: messages.length, snapshotId: messages.length, idempotentReplay: false, newObservation: true } };
      },
      onMessage: {
        addListener: (listener) => {
          messageListeners.push(listener);
        }
      }
    }
  };
  const timeoutsCalled = [];
  const context = vm.createContext({
    ...window, window, document: window.document, location: window.location, chrome,
    crypto: globalThis.crypto, console, URL, MutationObserver: window.MutationObserver,
    HTMLElement: window.HTMLElement, HTMLAnchorElement: window.HTMLAnchorElement,
    HTMLButtonElement: window.HTMLButtonElement, HTMLImageElement: window.HTMLImageElement,
    getComputedStyle: () => ({ position: 'static', display: 'block', visibility: 'visible', opacity: '1' }),
    setTimeout: (fn, delay, ...args) => {
      timeoutsCalled.push(delay);
      const testDelay = delay >= 300 ? 1 : delay;
      return setTimeout(fn, testDelay, ...args);
    },
    clearTimeout,
    timeoutsCalled,
  });
  vm.runInContext(compiled, context, { filename: 'content-script-list.js' });
  return { context, window, document: window.document, messages, timeoutsCalled, messageListeners };
}

test('visible list extraction keeps only exact offer links and explicit fields', () => {
  const loaded = loadList([
    productCard('1001', { title: '可靠商品', price: '3.50–4.20', moq: 10 }),
    productCard('1002', { title: '缺失字段商品', price: false, image: false }),
    productCard('1003', { hidden: true }),
    '<li><a href="https://detail.1688.com/offer/not-a-number.html">错误链接</a></li>',
    '<li><a href="https://evil.example/offer/1004.html">站外链接</a></li>',
    '<li><a href="https://detail.1688.com/offer/1001.html">重复链接</a></li>',
  ].join(''));
  const offers = JSON.parse(vm.runInContext('JSON.stringify(extractVisibleOffers().map(({offerId,pageData}) => ({offerId,pageData})))', loaded.context));
  assert.equal(offers.length, 2);
  assert.equal(offers[0].offerId, '1001');
  assert.equal(offers[0].pageData.price_model, 'range');
  assert.equal(offers[0].pageData.price_min, 3.5);
  assert.equal(offers[0].pageData.price_max, 4.2);
  assert.equal(offers[0].pageData.min_order_qty, 10);
  assert.equal(offers[0].pageData.field_statuses.supplier, 'unknown');
  assert.equal(offers[0].pageData.field_statuses.sku, 'unknown');
  assert.equal(offers[1].pageData.price_1688, 0);
  assert.equal(offers[1].pageData.field_statuses.price, 'unknown');
  assert.equal(offers[1].pageData.field_statuses.images, 'unknown');
  assert.equal('raw_html' in offers[0].pageData, false);
});

test('1688 home cards are recognized while unrelated ERP-injected links remain untouched', () => {
  const loaded = loadList(
    `${productCard('1101')}<aside id="other-erp"><input class="erp-checkbox" checked><a href="https://erp.example/product/9">ERP商品</a></aside>`,
    undefined,
    'https://www.1688.com/',
  );
  assert.equal(vm.runInContext('extractVisibleOffers().length', loaded.context), 1);
  const otherERP = loaded.document.getElementById('other-erp');
  assert.equal(otherERP.querySelectorAll('[data-lingmirror-offer-selector]').length, 0);
  assert.equal(otherERP.querySelector('.erp-checkbox').hasAttribute('checked'), true);
});

test('list collector has isolated selection UI and both Owner batch actions', () => {
  const loaded = loadList(productCard('2001') + productCard('2002'));
  const host = loaded.document.getElementById('lingmirror-list-collector-host');
  assert.ok(host?.shadowRoot, 'panel must be isolated in a shadow root');
  const labels = Array.from(host.shadowRoot.querySelectorAll('button')).map((button) => button.textContent);
  assert.ok(labels.includes('采集选中'));
  assert.ok(labels.includes('采集本页'));
  assert.ok(labels.includes('停止批量'));
  for (const card of loaded.document.querySelectorAll('.offer-item')) {
    const selectorHost = card.querySelector('[data-lingmirror-offer-selector]');
    assert.ok(selectorHost?.shadowRoot, 'card selector must be isolated from marketplace/ERP CSS');
  }
});

test('collect current page submits every currently visible offer without a fixed quota and keeps per-item results', async () => {
  const loaded = loadList(productCard('3001') + productCard('3002'));
  await vm.runInContext('collectOffers(extractVisibleOffers())', loaded.context);
  const collectionMessages = loaded.messages.filter((message) => message.type === 'collect_private_product');
  assert.equal(collectionMessages.length, 2);
  assert.deepEqual(collectionMessages.map((message) => message.pageData.offer_id_page), ['3001', '3002']);
  assert.ok(collectionMessages.every((message) => message.pageData.driver === 'chrome_extension_list_visible'));
  const panelText = loaded.document.getElementById('lingmirror-list-collector-host').shadowRoot.innerHTML;
  assert.match(panelText, /商品3001：已保存/);
  assert.match(panelText, /商品3002：已保存/);
});

test('selection action submits only checked items and batch results preserve each failure independently', async () => {
  let call = 0;
  const loaded = loadList(productCard('3101') + productCard('3102'), async (message) => {
    call += 1;
    if (call === 1) return { type: 'private_collection_result', requestId: message.requestId,
      payload: { status: 'not_saved', saved: false, code: 'NOT_SAVED', message: '服务器确认未保存' } };
    return { type: 'private_collection_result', requestId: message.requestId,
      payload: { status: 'saved', recordId: 22, snapshotId: 22, idempotentReplay: false, newObservation: true } };
  });
  vm.runInContext('selectedOfferIDs.add("3101"); selectedOfferIDs.add("3102"); scanVisibleOffers()', loaded.context);
  const panel = loaded.document.getElementById('lingmirror-list-collector-host').shadowRoot;
  const collectSelected = Array.from(panel.querySelectorAll('button')).find((button) => button.textContent === '采集选中');
  collectSelected.click();
  while (vm.runInContext('collectingBatch', loaded.context)) await new Promise((resolve) => setTimeout(resolve, 20));
  assert.equal(loaded.messages.filter((message) => message.type === 'collect_private_product').length, 2);
  assert.match(panel.innerHTML, /商品3101：服务器确认未保存/);
  assert.match(panel.innerHTML, /商品3102：已保存 #22/);
});

test('visible extraction has no hard-coded product count', () => {
  const cards = Array.from({ length: 37 }, (_, index) => productCard(String(4000 + index))).join('');
  const loaded = loadList(cards);
  assert.equal(vm.runInContext('extractVisibleOffers().length', loaded.context), 37);
});

test('SPA and virtual-list mutations are rescanned, and an in-flight batch can stop before the next item', async () => {
  const loaded = loadList(productCard('5001') + productCard('5002'), async (message) => {
    await new Promise((resolve) => setTimeout(resolve, 20));
    return { type: 'private_collection_result', requestId: message.requestId,
      payload: { status: 'saved', recordId: 1, snapshotId: 1, idempotentReplay: false, newObservation: true } };
  });
  loaded.document.getElementById('results').insertAdjacentHTML('beforeend', productCard('5003'));
  await new Promise((resolve) => setTimeout(resolve, 240));
  assert.equal(vm.runInContext('currentOffers.has("5003")', loaded.context), true);

  const promise = vm.runInContext('collectOffers(extractVisibleOffers())', loaded.context);
  setTimeout(() => vm.runInContext('cancelBatch = true', loaded.context), 5);
  await promise;
  assert.equal(loaded.messages.filter((message) => message.type === 'collect_private_product').length, 1);
  const panelText = loaded.document.getElementById('lingmirror-list-collector-host').shadowRoot.innerHTML;
  assert.match(panelText, /已停止/);
  assert.match(panelText, /剩余 2 个未提交/);
});

test('visible list extraction parses supplier name and id if present in card', () => {
  const loaded = loadList(productCard('7001', { company: '杭州智造有限公司', shopId: 'hzsourcing' }));
  const offers = JSON.parse(vm.runInContext('JSON.stringify(extractVisibleOffers().map(({offerId,pageData}) => ({offerId,pageData})))', loaded.context));
  assert.equal(offers.length, 1);
  assert.equal(offers[0].pageData.supplier_name, '杭州智造有限公司');
  assert.equal(offers[0].pageData.supplier_id_1688, 'hzsourcing');
  assert.equal(offers[0].pageData.supplier_business_id, 'hzsourcing');
  assert.equal(offers[0].pageData.field_statuses.supplier, 'observed');
});

test('visible list extraction ignores generic boilerplate link text as supplier name', () => {
  const loaded = loadList(productCard('7002', { shopId: 'hzsourcing', company: '' }) + `<a href="https://hzsourcing.1688.com">进入店铺</a>`);
  const offers = JSON.parse(vm.runInContext('JSON.stringify(extractVisibleOffers().map(({offerId,pageData}) => ({offerId,pageData})))', loaded.context));
  assert.equal(offers.length, 1);
  assert.equal(offers[0].pageData.supplier_id_1688, 'hzsourcing');
  assert.equal(offers[0].pageData.supplier_name, ''); // Should be empty, not '进入店铺'
  assert.equal(offers[0].pageData.field_statuses.supplier, 'unknown');
});

test('custom batch delay and jitter is read from the UI input and respects the minimum boundary', async () => {
  // Test default delay logic (value = 2.0s -> targetDelayMs = 2000 -> jitter in [1400, 2600])
  const loadedDefault = loadList(productCard('8001') + productCard('8002'));
  await vm.runInContext('collectOffers(extractVisibleOffers())', loadedDefault.context);
  const defaultDelays = loadedDefault.timeoutsCalled.filter(t => t >= 300); // filter out short layout timeouts if any
  assert.equal(defaultDelays.length, 1);
  assert.ok(defaultDelays[0] >= 1400 && defaultDelays[0] <= 2600, `Default delay ${defaultDelays[0]} should be between 1400 and 2600 ms`);

  // Test custom delay logic (value = 1.0s -> targetDelayMs = 1000 -> jitter in [700, 1300])
  const loadedCustom = loadList(productCard('8101') + productCard('8102'));
  const inputEl = loadedCustom.document.getElementById('lingmirror-list-collector-host').shadowRoot.getElementById('lingmirror-batch-delay-input');
  assert.ok(inputEl, 'delay input element must exist in shadow root');
  assert.equal(inputEl.value, '2.0', 'default delay input value must be 2.0');
  inputEl.value = '1.0';
  await vm.runInContext('collectOffers(extractVisibleOffers())', loadedCustom.context);
  const customDelays = loadedCustom.timeoutsCalled.filter(t => t >= 300);
  assert.equal(customDelays.length, 1);
  assert.ok(customDelays[0] >= 700 && customDelays[0] <= 1300, `Custom 1.0s delay ${customDelays[0]} should be between 700 and 1300 ms`);

  // Test minimum boundary logic (value = 0.2s -> enforced to 0.5s -> targetDelayMs = 500 -> jitter in [350, 650])
  const loadedMin = loadList(productCard('8201') + productCard('8202'));
  const inputMin = loadedMin.document.getElementById('lingmirror-list-collector-host').shadowRoot.getElementById('lingmirror-batch-delay-input');
  inputMin.value = '0.2';
  await vm.runInContext('collectOffers(extractVisibleOffers())', loadedMin.context);
  const minDelays = loadedMin.timeoutsCalled.filter(t => t >= 300);
  assert.equal(minDelays.length, 1);
  assert.ok(minDelays[0] >= 350 && minDelays[0] <= 650, `Enforced minimum 0.5s delay ${minDelays[0]} should be between 350 and 650 ms`);
});

test('fetch_list_page message listener extracts list items and responds', async () => {
  const loaded = loadList(productCard('9001', { title: '测试商品9001', price: '12.50' }));
  assert.equal(loaded.messageListeners.length, 1);
  const listener = loaded.messageListeners[0];
  let responseData;
  const sendResponse = (response) => {
    responseData = response;
  };
  const isAsync = listener({ type: 'fetch_list_page' }, {}, sendResponse);
  assert.ok(isAsync);
  assert.ok(responseData);
  assert.ok(responseData.data);
  assert.equal(responseData.data.length, 1);
  assert.equal(responseData.data[0].title, '测试商品9001');
  assert.equal(responseData.data[0].price_range, '12.5');
  assert.equal(responseData.data[0].detail_url, 'https://detail.1688.com/offer/9001.html');
});
