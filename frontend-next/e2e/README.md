# LingMirror E2E 测试

> Playwright 主链路 E2E 测试，覆盖：登录 → dashboard → /ai 运行 agent → trace 回放 → action 审批 → 执行 → AgentOS 驾驶舱

## 安装

```bash
cd frontend-next/e2e
npx playwright install chromium
```

## 运行

### 前置条件
1. 后端运行：`cd backend-go && go run cmd/server/main.go`（默认 :8080）
2. 前端运行：`cd frontend-next && npm run dev`（默认 :3000）
3. 数据库已 migrate（000001 + 000002 applied）
4. 如果没有测试用户，spec 会自动尝试 `POST /api/v1/auth/register` 创建一个

### 跑测试

```bash
# 自动启动 Next dev server
cd frontend-next/e2e
npm run e2e

# 带浏览器界面（本地调试）
npm run e2e:headed

# UI 模式（交互式调试）
npm run e2e:ui

# 单测
npx playwright test tests/main-chain.spec.ts -g "login"
```

### 后端未启动时

如果后端不可达，主链路测试会 **skip**（不 fail），便于 CI 在纯前端构建时不红。设置 `E2E_SKIP_WEB_SERVER=1` 可跳过自动启动 dev server（假设你已经手动起了）。

### CI 配置

```yaml
# GitHub Actions 示例
- name: E2E
  run: |
    cd backend-go && go run cmd/server/main.go &
    cd frontend-next && npm run dev &
    sleep 5
    cd frontend-next/e2e && npx playwright install chromium && npm run e2e
```

## 测试覆盖

| Spec | 验证点 |
|---|---|
| main-chain.spec.ts | 登录 → dashboard 6 个 stat card → /ai 命令栏 → POST /ai/run 拿 trace_id → /agents/[id]/trace/[traceId] 渲染 timeline + events → /actions/[id] 显示风险 → 批准 → 执行 → 状态变 executed → /agentos 显示 work queue |
| 同上 | /orders 列表页 table 渲染 |
| 同上 | /search 全局搜索 |
| 同上 | /agentos autonomy 控件 |

## 故障排查

| 现象 | 原因 | 处理 |
|---|---|---|
| `Backend not reachable` skip | 后端没起或端口不对 | 确认 :8080 health，或设 `E2E_API_BASE` |
| login 失败 | 没有 /auth/register 路由 | 见 parity report，Go 端缺 register，需补或手动 seed 用户 |
| trace 页面无 events | orchestrator stub 未写 event | 检查 DB ai_trace_event 表 |
| action 批准按钮不可见 | action 状态不是 suggested | 检查 unified_action.status |
