import assert from 'node:assert/strict';
import test from 'node:test';

import {
  addPendingCollection,
  buildPendingCollectionMarker,
  buildPrivateCollectionPayload,
  describeCollectionRecovery,
  mergeCollectionRecoveryHistory,
  parsePrivateCollectionResponse,
  queryPrivateCollectionRequest,
  reconcilePrivateCollectionRequest,
  removePendingCollection,
  submitPrivateCollection,
  submitPrivateCaptureFailure,
	DuplicatePrivateCollectionError,
	duplicateComparisonLines,
} from '../build/shared/private-collection.js';
import { getApiBaseUrl } from '../build/shared/auth.js';

test('builds Owner private collection payload from the visible 1688 page', () => {
  const page = {
	schema_version: 'sourcing1688.private.v1', offer_id_url: '123', offer_id_page: '123',
    source_url: 'https://detail.1688.com/offer/123.html?spm=tracking',
    collected_at: '2026-07-12T10:00:00.000Z',
    driver: 'plugin',
    parser_version: '1688-detail-v1',
    title: '页面标题',
    price_1688: 3.5,
	price_model: 'fixed',
    min_order_qty: 2,
    currency: 'CNY',
    images: ['https://cbu01.alicdn.com/a.jpg'],
    supplier_name: '供应商',
    supplier_id_1688: 'seller-1',
    supplier_business_id: 'seller-1',
	field_statuses: { title: 'observed', price: 'observed', moq: 'observed', supplier: 'observed', images: 'observed', sku: 'no_sku' },
  };

  assert.deepEqual(buildPrivateCollectionPayload(page, 'collect_test_001', '0.2.0'), {
	schema_version: page.schema_version,
	page_offer_id: page.offer_id_page,
	price_model: page.price_model,
    request_id: 'collect_test_001',
    source_url: page.source_url,
    observed_at: page.collected_at,
    parser_version: page.parser_version,
    extension_version: '0.2.0',
    raw_payload: page,
    title: page.title,
    price: page.price_1688,
    moq: page.min_order_qty,
    supplier_name: page.supplier_name,
    supplier_business_id: page.supplier_business_id,
    images: page.images,
	field_statuses: page.field_statuses,
  });
});

test('saved requires a server record id and request id', () => {
  assert.deepEqual(parsePrivateCollectionResponse({
    code: 0,
    data: { status: 'saved', record_id: 42, snapshot_id: 7, request_id: 'collect_test_001' },
  }), {
    status: 'saved', recordId: 42, snapshotId: 7, requestId: 'collect_test_001',
    idempotentReplay: false, newObservation: false,
  });

  assert.throws(
    () => parsePrivateCollectionResponse({ code: 0, data: { status: 'saved', record_id: 0 } }),
    /没有返回有效记录编号/,
  );

  assert.throws(
    () => parsePrivateCollectionResponse({
      code: 0,
      data: { status: 'saved', record_id: 42, request_id: 'collect_other_tab' },
    }, 'collect_test_001'),
    /其他采集请求/,
  );
});

test('builds HTTP API origin from configured WebSocket server', () => {
  assert.equal(getApiBaseUrl('ws://localhost:8080'), 'http://localhost:8080/api/v1');
  assert.equal(getApiBaseUrl('wss://owner.lingmirror.com/ws/extension'), 'https://owner.lingmirror.com/api/v1');
});

test('private collection uses only the extension-scoped sourcing API', async () => {
  const calls = [];
  const fetcher = async (url, init) => {
    calls.push({ url, init });
    return new Response(JSON.stringify({
      code: 0,
      data: {
        status: 'saved', record_id: 42, snapshot_id: 7,
        request_id: 'collect_test_001', new_observation: true,
      },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } });
  };

  const page = {
    source_url: 'https://detail.1688.com/offer/123.html',
    collected_at: '2026-07-12T10:00:00.000Z',
    driver: 'plugin', parser_version: '1688-detail-v1', title: '页面标题',
    price_1688: 3.5, min_order_qty: 2, currency: 'CNY', images: [],
  };
  const saved = await submitPrivateCollection(
    page, 'collect_test_001', '0.2.0', 'extension-token',
    'wss://owner.lingmirror.com', fetcher,
  );

  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, 'https://owner.lingmirror.com/api/v1/extension/sourcing-1688/private-collections');
  assert.equal(calls[0].init.headers.Authorization, 'Bearer extension-token');
  assert.equal(saved.recordId, 42);
});

test('duplicate offer requires an explicit new-observation intent and preserves the new request id', async () => {
	const page = {
		schema_version: 'sourcing1688.private.v1', offer_id_url: '123', offer_id_page: '123',
		source_url: 'https://detail.1688.com/offer/123.html', collected_at: '2026-07-12T10:00:00Z',
		driver: 'plugin', parser_version: '1688-detail-v1', title: '重复商品', price_1688: 0,
		price_model: 'unknown', min_order_qty: 0, currency: 'CNY', images: [],
		field_statuses: { title: 'observed', price: 'unknown', moq: 'unknown', supplier: 'unknown', images: 'unknown', sku: 'no_sku' },
	};
	await assert.rejects(
		() => submitPrivateCollection(page, 'collect_duplicate_probe', '0.2.0', 'token', 'https://owner.test', async () =>
			new Response(JSON.stringify({ code: 409, message: 'choice required', data: { status: 'duplicate_requires_choice', record_id: 42, snapshot_id: 7,
				existing: { title: '旧标题', price: 8, moq: 3, supplier_name: '旧供应商', sku_count: 2, image_count: 1, observed_at: '2026-07-11T09:00:00Z' },
			} }), { status: 409, headers: { 'Content-Type': 'application/json' } })),
		(err) => err instanceof DuplicatePrivateCollectionError && err.recordId === 42 && err.snapshotId === 7 && err.existing.title === '旧标题',
	);
	let submitted;
	await submitPrivateCollection(page, 'collect_new_observation', '0.2.0', 'token', 'https://owner.test', async (_url, init) => {
		submitted = JSON.parse(init.body);
		return new Response(JSON.stringify({ code: 0, data: { status: 'saved', record_id: 42, snapshot_id: 8, request_id: 'collect_new_observation', new_observation: true } }), { status: 200, headers: { 'Content-Type': 'application/json' } });
	}, 'save_new_observation');
	assert.equal(submitted.request_id, 'collect_new_observation');
	assert.equal(submitted.observation_intent, 'save_new_observation');
});

test('duplicate comparison shows every safe current-vs-existing field without raw page data', () => {
	const page = {
		title: '新标题', price_1688: 9, min_order_qty: 3, supplier_name: '新供应商',
		spec_variants: [{ spec: '红色' }], images: ['https://cbu01.alicdn.com/a.jpg'],
	};
	const lines = duplicateComparisonLines(page, {
		title: '旧标题', price: 8, moq: 3, supplier_name: '旧供应商', sku_count: 2, image_count: 1,
		observed_at: '2026-07-11T09:00:00Z',
	});
	assert.equal(lines.length, 8);
	for (const label of ['标题', '价格', '起订量', '供应商', 'SKU数', '图片数', '已有观察时间']) {
		assert.ok(lines.some((line) => line.includes(label)), `missing ${label}`);
	}
	assert.ok(lines.some((line) => line.includes('本次 新标题｜已有 旧标题（有变化）')));
	assert.ok(lines.some((line) => line.includes('本次 3件｜已有 3件（相同）')));
	assert.equal(lines.join('\n').includes('raw_payload'), false);
	assert.equal(lines.join('\n').includes('raw_html'), false);
});

test('capture failure reports only safe metadata and never page contents', async () => {
  let call;
  await submitPrivateCaptureFailure({
    requestId: 'collect_failed_001', sourceUrl: 'https://detail.1688.com/offer/123.html',
    errorCode: 'sku_parse_failed', schemaVersion: 'sourcing1688.private.v1',
    extensionVersion: '0.2.0', parserVersion: '1688-detail-v1', occurredAt: '2026-07-12T10:00:00Z',
  }, 'extension-token', 'wss://owner.lingmirror.com', async (url, init) => {
    call = { url, init };
    return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } });
  });
  assert.equal(call.url, 'https://owner.lingmirror.com/api/v1/extension/sourcing-1688/private-collections/failures');
  const body = JSON.parse(call.init.body);
  assert.equal(body.error_code, 'sku_parse_failed');
  assert.equal(body.request_id, 'collect_failed_001');
  assert.equal('raw_html' in body, false);
  assert.equal('raw_payload' in body, false);
});

test('restart marker stores only request identity and API origin, never page data', () => {
  const marker = buildPendingCollectionMarker(
    'collect_test_001',
    'wss://owner.lingmirror.com/ws/extension',
    '2026-07-12T10:00:00.000Z',
    { tabId: 17, sourceUrl: 'https://detail.1688.com/offer/123.html?spm=private-tracking' },
  );
  assert.deepEqual(marker, {
    requestId: 'collect_test_001',
    apiOrigin: 'https://owner.lingmirror.com',
    createdAt: '2026-07-12T10:00:00.000Z',
    tabId: 17,
    sourceSummary: 'detail.1688.com/offer/123',
  });
  assert.equal('pageData' in marker, false);
  assert.equal('raw_payload' in marker, false);
});

test('pending markers are keyed by request id so simultaneous tabs cannot overwrite each other', () => {
  const first = buildPendingCollectionMarker('collect_a', 'wss://owner.lingmirror.com', '2026-07-12T10:00:00Z', { tabId: 11 });
  const second = buildPendingCollectionMarker('collect_b', 'wss://owner.lingmirror.com', '2026-07-12T10:00:01Z', { tabId: 22 });
  const pending = addPendingCollection(addPendingCollection({}, first), second);
  assert.deepEqual(Object.keys(pending).sort(), ['collect_a', 'collect_b']);
  assert.equal(pending.collect_a.tabId, 11);
  assert.deepEqual(Object.keys(removePendingCollection(pending, 'collect_a')), ['collect_b']);
  assert.equal(pending.collect_a.requestId, 'collect_a', 'removal must not mutate the caller map');
});

test('reconciliation distinguishes saved, confirmed missing, and unreachable', async () => {
  const saved = await queryPrivateCollectionRequest(
    'collect_test_001', 'extension-token', 'wss://owner.lingmirror.com',
    async (url) => {
      assert.equal(url, 'https://owner.lingmirror.com/api/v1/extension/sourcing-1688/private-collections/requests/collect_test_001');
      return new Response(JSON.stringify({
        code: 0,
        data: { status: 'saved', record_id: 91, snapshot_id: 17, request_id: 'collect_test_001' },
      }), { status: 200, headers: { 'Content-Type': 'application/json' } });
    },
  );
  assert.deepEqual(saved, { status: 'saved', requestId: 'collect_test_001', recordId: 91, snapshotId: 17 });

  const notSaved = await queryPrivateCollectionRequest(
    'collect_duplicate', 'extension-token', 'wss://owner.lingmirror.com',
    async () => new Response(JSON.stringify({
      code: 0,
      data: { status: 'not_saved', request_id: 'collect_duplicate', failure_code: 'duplicate_requires_choice', safe_message: '本次未保存' },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
  );
  assert.deepEqual(notSaved, { status: 'not_saved', requestId: 'collect_duplicate', message: '本次未保存' });

  const uncertain = await queryPrivateCollectionRequest(
    'collect_uncertain', 'extension-token', 'wss://owner.lingmirror.com',
    async () => new Response(JSON.stringify({
      code: 0,
      data: { status: 'reconcile_required', request_id: 'collect_uncertain', safe_message: '服务器正在核对' },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
  );
  assert.deepEqual(uncertain, { status: 'reconcile_required', requestId: 'collect_uncertain', message: '服务器正在核对' });

  const missing = await queryPrivateCollectionRequest(
    'collect_test_002', 'extension-token', 'wss://owner.lingmirror.com',
    async () => new Response('', { status: 404 }),
  );
  assert.equal(missing, null);

  await assert.rejects(
    queryPrivateCollectionRequest(
      'collect_test_003', 'extension-token', 'wss://owner.lingmirror.com',
      async () => { throw new TypeError('network unavailable'); },
    ),
    /network unavailable/,
  );
});

test('reconciliation retries transient 404 before declaring not saved', async () => {
  let calls = 0;
  const waits = [];
  const result = await reconcilePrivateCollectionRequest(
    'collect_race', 'extension-token', 'wss://owner.lingmirror.com', {
      delaysMs: [10, 20],
      wait: async (milliseconds) => { waits.push(milliseconds); },
      fetcher: async () => {
        calls += 1;
        if (calls < 3) return new Response('', { status: 404 });
        return new Response(JSON.stringify({
          code: 0,
          data: { status: 'saved', record_id: 8, snapshot_id: 9, request_id: 'collect_race' },
        }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      },
    },
  );
  assert.deepEqual(waits, [10, 20]);
  assert.deepEqual(result, { status: 'saved', requestId: 'collect_race', recordId: 8, snapshotId: 9 });
});

test('reconciliation keeps missing receipt and unreachable distinct from server-confirmed not saved', async () => {
  const missing = await reconcilePrivateCollectionRequest(
    'collect_missing', 'extension-token', 'wss://owner.lingmirror.com', {
      delaysMs: [0], wait: async () => {},
      fetcher: async () => new Response('', { status: 404 }),
    },
  );
  assert.equal(missing.status, 'reconcile_required');
  assert.match(missing.message, /不能据此确认未保存/);

  const unknown = await reconcilePrivateCollectionRequest(
    'collect_unknown', 'extension-token', 'wss://owner.lingmirror.com', {
      delaysMs: [],
      fetcher: async () => { throw new TypeError('offline'); },
    },
  );
  assert.deepEqual(unknown, {
    status: 'reconcile_required', requestId: 'collect_unknown', message: 'offline',
  });
});

test('popup recovery copy keeps saved, not saved, and uncertain states explicit', () => {
  assert.match(describeCollectionRecovery({ status: 'saved', requestId: 'a', recordId: 4, snapshotId: 5 }), /已保存.*#4/s);
  assert.match(describeCollectionRecovery({ status: 'not_saved', requestId: 'b' }), /确认未保存/);
  assert.match(describeCollectionRecovery({ status: 'reconcile_required', requestId: 'c', message: 'offline' }), /仍待确认/);
  assert.match(describeCollectionRecovery({
    status: 'not_saved', requestId: 'b', sourceSummary: 'detail.1688.com/offer/123',
  }), /offer\/123.*确认未保存/s);
});

test('recovery history keeps multiple tab results identifiable and deduplicates retries', () => {
  const markerA = buildPendingCollectionMarker('collect_a', 'wss://owner.lingmirror.com', '2026-07-12T10:00:00Z', {
    tabId: 11, sourceUrl: 'https://detail.1688.com/offer/111.html?token=secret',
  });
  const markerB = buildPendingCollectionMarker('collect_b', 'wss://owner.lingmirror.com', '2026-07-12T10:00:01Z', {
    tabId: 22, sourceUrl: 'https://detail.1688.com/offer/222.html?token=secret',
  });
  let history = mergeCollectionRecoveryHistory([], { status: 'not_saved', requestId: 'collect_a' }, markerA, '2026-07-12T10:01:00Z');
  history = mergeCollectionRecoveryHistory(history, { status: 'reconcile_required', requestId: 'collect_b', message: 'offline' }, markerB, '2026-07-12T10:01:01Z');
  history = mergeCollectionRecoveryHistory(history, { status: 'saved', requestId: 'collect_a', recordId: 7, snapshotId: 8 }, markerA, '2026-07-12T10:01:02Z');
  assert.equal(history.length, 2);
  assert.deepEqual(history.map((item) => item.requestId), ['collect_a', 'collect_b']);
  assert.equal(history[0].sourceSummary, 'detail.1688.com/offer/111');
  assert.equal(JSON.stringify(history).includes('secret'), false);
});
