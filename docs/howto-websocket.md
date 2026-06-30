# How to 使用 WebSocket 流式更新

> 连接凌镜 WebSocket 端点，接收 AI 流式输出和实时事件通知。

---

## 前置条件

- 有效的 JWT Token（见 [API 快速参考](reference-api-quick.md#认证方式)）
- WebSocket 客户端（浏览器原生 `WebSocket`、`wscat`、Postman 等）

## 步骤

### 1. 连接 WebSocket

```javascript
// 前端 JavaScript
const token = "your-jwt-token";
const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

ws.onopen = () => console.log("已连接");
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log("收到:", msg);
};
ws.onerror = (err) => console.error("错误:", err);
ws.onclose = () => console.log("连接关闭");
```

**注意:** Token 通过查询参数 `?token=` 传递（WebSocket 握手不支持自定义 Header）。

### 2. 消息格式

所有消息为 JSON：

```json
{
  "type": "stream_chunk",
  "payload": {
    "content": "正在分析利润数据...",
    "agent_id": "A6",
    "trace_id": "abc-123"
  }
}
```

### 3. 消息类型

| `type` | payload | 说明 |
|--------|---------|------|
| `stream_chunk` | `{content, agent_id, trace_id}` | AI Agent 流式输出片段 |
| `stream_end` | `{agent_id, trace_id}` | 流结束 |
| `agent_decision` | `{agent_id, decision_point, action, confidence, risk_level}` | Agent 完成决策 |
| `notification` | `{title, message, level}` | 系统通知 |
| `dashboard_update` | `{overview, ...}` | 仪表盘自动刷新数据 |

### 4. 前端完整示例

```typescript
// frontend-next/src/lib/realtime.ts 已有封装
import { createRealtimeConnection } from "@/lib/realtime";

const conn = createRealtimeConnection({
  token: "your-jwt-token",
  onChunk: (content) => console.log("AI 输出:", content),
  onDecision: (decision) => console.log("Agent 决策:", decision),
  onNotification: (notif) => console.log("通知:", notif),
});
```

### 5. 后端接入点

后端 WebSocket 实现在 `internal/realtime/`：

- `hub.go`: 连接池管理、广播
- `handler.go`: HTTP → WebSocket 升级、消息路由
- 通过 `realtime.NewHandler(hub, logger, jwtSecret)` 创建处理器

AI 对话流式输出通过 `WithAIChat(ai.NewAIChatHandler(aiOrch))` 挂载，Client → Server 发送 JSON 格式 AI 对话请求。

## 验证

1. 启动后端和前端
2. 打开浏览器 DevTools → Network → WS
3. 在 AI 对话页面发送消息，观察 WebSocket 消息面板中的 `stream_chunk` 事件

## 故障排查

| 问题 | 原因与解决 |
|------|-----------|
| `401 Unauthorized` | Token 无效或已过期。刷新 Token 后重连。 |
| 连接立即关闭 | Token 验证失败。检查 `JWT_SECRET` 配置是否一致。 |
| 收不到消息 | 确认事件发布到了正确的 topic。用 `wscat` 测试裸连接。 |
| 前端无法连接 wss | 生产环境使用 `wss://` 协议，配置 Nginx 支持 WebSocket 升级。 |

---

## 相关文档

- [参考 - API 快速参考](reference-api-quick.md)
- [后端实时模块](../../backend-go/internal/realtime/)
