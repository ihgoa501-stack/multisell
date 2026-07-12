import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import { fileURLToPath } from 'node:url';

import { parseHTML } from 'linkedom';

const here = dirname(fileURLToPath(import.meta.url));
const extensionRoot = resolve(here, '..');
const compiledScript = await readFile(resolve(extensionRoot, 'build/content-script.js'), 'utf8');

async function loadFixture(name, url, messageResponder) {
  const html = await readFile(resolve(here, 'fixtures', name), 'utf8');
  const { window } = parseHTML(html);
  const sentMessages = [];
  const listeners = [];
  const chrome = {
    runtime: {
      getManifest: () => ({ version: '0.2.0' }),
      onMessage: { addListener: (listener) => listeners.push(listener) },
      sendMessage: async (message) => { sentMessages.push(message); return messageResponder ? messageResponder(message) : undefined; },
    },
  };
  Object.defineProperty(window, 'location', { value: new URL(url), configurable: true });
  const context = vm.createContext({
    ...window,
    window,
    document: window.document,
    location: window.location,
    chrome,
    crypto: globalThis.crypto,
    console,
    setTimeout,
    clearTimeout,
    URL,
    HTMLMetaElement: window.HTMLMetaElement,
    HTMLLinkElement: window.HTMLLinkElement,
    HTMLElement: window.HTMLElement,
    HTMLImageElement: window.HTMLImageElement,
    HTMLAnchorElement: window.HTMLAnchorElement,
    HTMLDivElement: window.HTMLDivElement,
    HTMLButtonElement: window.HTMLButtonElement,
  });
  vm.runInContext(compiledScript, context, { filename: 'content-script.js' });
  let data;
  try {
    data = vm.runInContext('extractPageData()', context);
  } catch {
    // Blocked fixtures intentionally do not need to be parseable product DOM.
    data = undefined;
  }
  return {
    data,
    collect: () => vm.runInContext('collectCurrentPage()', context),
    submit: () => vm.runInContext('submitPendingPreview()', context),
    sentMessages,
    listeners,
    document: window.document,
  };
}

test('realistic detail DOM extracts identity, tiered price, MOQ, supplier, images and complete SKU fields', async () => {
  const { data } = await loadFixture(
    '1688-detail-complete.html',
    'https://detail.1688.com/offer/692570310190.html?spm=a260k.test',
  );

  assert.equal(data.offer_id_url, '692570310190');
  assert.equal(data.offer_id_page, '692570310190');
  assert.equal(data.title, '加厚猫毛清洁滚筒');
  assert.equal(data.price_1688, 3.8);
  assert.equal(data.price_model, 'tiered');
  assert.equal(data.price_min, 3.5);
  assert.equal(data.price_max, 3.8);
  assert.equal(data.min_order_qty, 10);
  assert.equal(data.supplier_name, '义乌市清洁用品厂');
  assert.equal(data.supplier_business_id, '887766');
  assert.deepEqual(Array.from(data.images), [
    'https://cbu01.alicdn.com/img/offer-main.jpg',
    'https://cbu01.alicdn.com/img/offer-side.jpg',
  ]);
  assert.deepEqual(JSON.parse(JSON.stringify(data.spec_variants)), [
    { spec: '红色', price: 3.8, stock: 120, image_url: 'https://cbu01.alicdn.com/img/red.jpg' },
    { spec: '蓝色', price: 3.5, stock: 80 },
  ]);
  assert.deepEqual(JSON.parse(JSON.stringify(data.field_statuses)), {
    title: 'observed', price: 'observed', moq: 'observed', supplier: 'observed', images: 'observed', sku: 'observed',
  });
  assert.equal('raw_html' in data, false, 'private collection must not clone and upload the full page DOM');
});

test('missing detail fields stay explicit and visible SKU controls with incomplete data are parse_failed', async () => {
  const { data } = await loadFixture(
    '1688-detail-missing-fields.html',
    'https://detail.1688.com/offer/123456789012.html',
  );

  assert.equal(data.offer_id_url, '123456789012');
  assert.equal(data.offer_id_page, '123456789012');
  assert.equal(data.price_model, 'unknown');
  assert.equal(data.price_1688, 0);
  assert.equal(data.min_order_qty, 0);
  assert.deepEqual(Array.from(data.images), []);
  assert.deepEqual(JSON.parse(JSON.stringify(data.field_statuses)), {
    title: 'parse_failed', price: 'unknown', moq: 'unknown', supplier: 'unknown', images: 'unknown', sku: 'parse_failed',
  });
});

test('loading the content script installs local UI and listener but never uploads automatically', async () => {
  const loaded = await loadFixture(
    '1688-detail-complete.html',
    'https://detail.1688.com/offer/692570310190.html',
  );

  assert.equal(loaded.listeners.length, 1);
  assert.ok(loaded.document.getElementById('lingmirror-private-collector'));
  assert.deepEqual(loaded.sentMessages, []);
});

const blockedPages = [
  {
    name: 'non-product page', fixture: '1688-not-product.html', url: 'https://detail.1688.com/index.html',
    code: 'NOT_PRODUCT_PAGE', happened: '不是1688商品详情页', next: 'detail.1688.com/offer/数字.html',
  },
  {
    name: 'login required', fixture: '1688-login-required.html', url: 'https://detail.1688.com/offer/10001.html',
    code: 'LOGIN_REQUIRED', happened: '要求登录', next: '先在1688页面完成登录',
  },
  {
    name: 'captcha or risk control', fixture: '1688-risk-challenge.html', url: 'https://detail.1688.com/offer/10002.html',
    code: 'RISK_CHALLENGE', happened: '验证码或访问风控', next: '插件不会自动重试',
  },
  {
    name: 'unavailable offer placeholder', fixture: '1688-offer-unavailable.html', url: 'https://detail.1688.com/offer/10003.html',
    code: 'OFFER_UNAVAILABLE', happened: '商品下架、失效或占位页面', next: '选择一个仍可正常查看的商品',
  },
  {
    name: 'page still loading', fixture: '1688-page-loading.html', url: 'https://detail.1688.com/offer/10004.html',
    code: 'PAGE_LOADING', happened: '页面仍在加载', next: '等待页面停止转圈',
  },
  {
    name: 'SKU still unstable', fixture: '1688-sku-unstable.html', url: 'https://detail.1688.com/offer/10005.html',
    code: 'SKU_UNSTABLE', happened: 'SKU还没有稳定', next: '规格区停止加载',
  },
];

for (const scenario of blockedPages) {
  test(`${scenario.name} is locally blocked with an actionable Chinese explanation`, async () => {
    const loaded = await loadFixture(scenario.fixture, scenario.url);
    const result = await loaded.collect();
    const panelText = loaded.document.getElementById('lingmirror-private-collector')?.textContent || '';

    assert.equal(result.payload.status, 'failed');
    assert.equal(result.payload.code, scenario.code);
    assert.equal(result.payload.saved, false);
    assert.match(panelText, /发生了什么：/);
    assert.match(panelText, new RegExp(scenario.happened));
    assert.match(panelText, /是否保存：没有保存，也没有上传失败页面内容。/);
    assert.match(panelText, /下一步：/);
    assert.match(panelText, new RegExp(scenario.next));
    assert.deepEqual(loaded.sentMessages, [], 'blocked page content must never be sent to the background worker');
  });
}

test('a page becoming blocked after preview is stopped before confirmation upload', async () => {
  const loaded = await loadFixture(
    '1688-detail-complete.html',
    'https://detail.1688.com/offer/692570310190.html',
  );
  const preview = await loaded.collect();
  assert.equal(preview.payload.code, 'PREVIEW_REQUIRED');
  assert.deepEqual(loaded.sentMessages, []);

  const challenge = loaded.document.createElement('div');
  challenge.setAttribute('data-page-state', 'captcha');
  challenge.textContent = '安全验证';
  loaded.document.body.appendChild(challenge);
  await loaded.submit();

  const panelText = loaded.document.getElementById('lingmirror-private-collector')?.textContent || '';
  assert.match(panelText, /验证码或访问风控/);
  assert.match(panelText, /是否保存：没有保存/);
  assert.deepEqual(loaded.sentMessages, [], 'confirmation re-check must not upload a newly blocked page');
});

test('duplicate sidebar compares all safe fields and offers both explicit Owner actions', async () => {
  const loaded = await loadFixture(
    '1688-detail-complete.html',
    'https://detail.1688.com/offer/692570310190.html',
    async (message) => message.type === 'collect_private_product' ? {
      type: 'private_collection_result', requestId: message.requestId,
      payload: {
        status: 'duplicate_requires_choice', saved: false, recordId: 42, snapshotId: 7,
        message: 'choice required',
        existing: {
          title: '旧款猫毛滚筒', price: 3.5, moq: 10, supplier_name: '旧供应商',
          sku_count: 1, image_count: 2, observed_at: '2026-07-11T09:00:00Z',
        },
      },
    } : undefined,
  );
  await loaded.collect();
  await loaded.submit();

  const panel = loaded.document.getElementById('lingmirror-private-collector');
  const panelText = panel?.textContent || '';
  for (const label of ['本次页面 vs 已有观察', '标题', '价格', '起订量', '供应商', 'SKU数', '图片数', '已有观察时间']) {
    assert.match(panelText, new RegExp(label));
  }
  assert.match(panelText, /本次 加厚猫毛清洁滚筒｜已有 旧款猫毛滚筒（有变化）/);
  assert.match(panelText, /本次 10件｜已有 10件（相同）/);
  const buttons = Array.from(panel?.querySelectorAll('button') || []).map((button) => button.textContent);
  assert.ok(buttons.includes('查看已有记录'));
  assert.ok(buttons.includes('保存为新观察'));
  assert.equal(panelText.includes('raw_payload'), false);
  assert.equal(panelText.includes('raw_html'), false);
});
