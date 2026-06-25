import { check, sleep } from 'k6';
import ws from 'k6/ws';
import { Counter, Rate } from 'k6/metrics';

const BASE = __ENV.API_BASE || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN || '';

// 把 http(s):// 转成 ws(s)://
const WS_BASE = BASE.replace(/^http/, 'ws');
const WS_URL = `${WS_BASE}/ws${TOKEN ? `?token=${TOKEN}` : ''}`;

const messagesReceived = new Counter('ws_messages_received');
const errorRate = new Rate('errors');

export const options = {
  scenarios: {
    websocket_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '10s', target: 100 },
        { duration: '10s', target: 0 },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    ws_messages_received: ['count>0'],
    errors: ['rate<0.05'],
  },
};

export default function () {
  const res = ws.connect(WS_URL, {}, (socket) => {
    socket.on('open', () => {
      socket.send(JSON.stringify({ type: 'ping' }));
      // 保持连接一段时间后自动关闭
      socket.setTimeout(() => {
        socket.close();
      }, 10000); // 10秒后关闭
    });

    socket.on('message', (data) => {
      messagesReceived.add(1);
      // 回 ping 维持心跳
      try {
        const msg = JSON.parse(data);
        if (msg.type === 'ping' || msg.type === 'ping_req') {
          socket.send(JSON.stringify({ type: 'pong' }));
        }
      } catch (e) {
        // 非 JSON 消息，忽略
      }
    });

    socket.on('error', () => errorRate.add(true));
    socket.on('close', () => {});
  });

  check(res, {
    'status is 101': (r) => r && r.status === 101,
  });
}
