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
    dashboard_load: {
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
  },
};

export default function () {
  const res = http.get(`${BASE}/api/v1/dashboard/overview`, authHeaders);

  const ok = check(res, {
    'status is 200': (r) => r.status === 200,
    'has overview body': (r) => r.body && r.body.length > 0,
  });

  errorRate.add(!ok);

  sleep(1);
}
