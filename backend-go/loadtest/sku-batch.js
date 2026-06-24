import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const BASE = __ENV.API_BASE || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN || '';

const headers = {
  'Content-Type': 'application/json',
};
if (TOKEN) {
  headers['Authorization'] = `Bearer ${TOKEN}`;
}
const authHeaders = {
  headers: headers,
};

const errorRate = new Rate('errors');
const skuProcessed = new Counter('sku_processed');

export const options = {
  scenarios: {
    sku_batch_run: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 5 },
        { duration: '10s', target: 5 },
        { duration: '10s', target: 0 },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.05'],
    errors: ['rate<0.05'],
    sku_processed: ['count>0'],
  },
};

// SKU id 循环计数器，从 1 递增到 1000 后回到 1
const skuIds = Array.from({ length: 1000 }, (_, i) => i + 1);

export default function () {
  // k6 的 exec.vuInTestInstance 可在分片场景下作模运算；
  // 这里直接用迭代上下文的 __VU/__ITER 让多个 VU 分摊 SKU 范围
  const idx = ((__VU - 1) * 1000 + __ITER) % 1000;
  const skuId = skuIds[idx];

  const payload = JSON.stringify({
    agent_id: 'A6',
    decision_point: 'profit_check',
    context: {
      sku_id: skuId,
    },
  });

  const res = http.post(`${BASE}/api/v1/ai/run`, payload, authHeaders);

  const ok = check(res, {
    'status is 200': (r) => r.status === 200,
    'has response body': (r) => r.body && r.body.length > 0,
  });

  errorRate.add(!ok);

  if (ok) {
    skuProcessed.add(1);
  }

  sleep(0.1);
}
