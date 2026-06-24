import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

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

export const options = {
  scenarios: {
    action_approve_load: {
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
    http_req_failed: ['rate<0.60'],
    errors: ['rate<0.60'],
  },
};

export default function () {
  // 步骤 1：拉取 suggested 状态的 action 列表
  const listRes = http.get(
    `${BASE}/api/v1/ai/actions?status=suggested`,
    authHeaders
  );

  const listOk = check(listRes, {
    'list status is 200': (r) => r.status === 200,
  });
  errorRate.add(!listOk);

  if (!listOk) {
    sleep(1);
    return;
  }

  // 解析 action 列表，取第一个 id
  let actions = [];
  try {
    const body = JSON.parse(listRes.body);
    actions = Array.isArray(body) ? body : (body.actions || body.data || []);
  } catch (e) {
    errorRate.add(true);
    sleep(1);
    return;
  }

  if (actions.length === 0) {
    sleep(1);
    return;
  }

  const actionId = actions[0].id || actions[0].action_id;
  if (!actionId) {
    sleep(1);
    return;
  }

  // 步骤 2：approve 指定 action
  const approveRes = http.post(
    `${BASE}/api/v1/ai/actions/${actionId}/approve`,
    JSON.stringify({ operator: 'loadtest_user', reason: 'load test approval' }),
    authHeaders
  );

  const approveOk = check(approveRes, {
    'approve status is expected': (r) => r.status === 200 || r.status === 201 || (r.status === 500 && r.body && r.body.indexOf("invalid action transition") !== -1),
  });
  errorRate.add(!approveOk);

  sleep(1);
}
