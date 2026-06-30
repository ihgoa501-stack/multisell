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
npx playwright test tests/login.spec.ts
npx playwright test tests/products.spec.ts
npx playwright test tests/owner-approval.spec.ts
npx playwright test tests/main-chain.spec.ts -g "login"
```

### 后端未启动时

如果后端不可达，主链路测试会 **skip**（不 fail），便于 CI 在纯前端构建时不红。
设置 `E2E_SKIP_WEB_SERVER=1` 可跳过自动启动 dev server（假设你已经手动起了）。

Mock-based 测试（`login.spec.ts`、`products.spec.ts`、`owner-approval.spec.ts`、`sourcing.spec.ts`）
**无需后端**——它们使用 Playwright route interception 拦截所有 API 调用，并注入了 fake JWT token。

### CI 配置

```yaml
# GitHub Actions — 见 .github/workflows/ci.yml
# e2e-test job 自动安装依赖 + Playwright 浏览器 + 跑 mock-based 测试
```

## 测试覆盖

| Spec | 验证点 | 需后端 |
|---|---|---|
| main-chain.spec.ts | 登录 → dashboard 6 个 stat card → /ai 命令栏 → POST /ai/run 拿 trace_id → /agents/[id]/trace/[traceId] 渲染 timeline + events → /actions/[id] 显示风险 → 批准 → 执行 → 状态变 executed → /agentos 显示 work queue | 是 |
| main-chain.spec.ts | /orders 列表页 table 渲染 | 是 |
| main-chain.spec.ts | /search 全局搜索 | 是 |
| main-chain.spec.ts | /agentos autonomy 控件 | 是 |
| **login.spec.ts** | 登录页渲染（品牌、表单） | **否**（Mock） |
| **login.spec.ts** | 提交凭证 → Dashboard 统计卡片 | **否**（Mock） |
| **login.spec.ts** | 错误凭证 → 保持在登录页 | **否**（Mock） |
| **products.spec.ts** | 商品列表 Table 渲染 + 分页 | **否**（Mock） |
| **products.spec.ts** | 新建商品 → 填写表单 → 提交 → 成功消息 | **否**（Mock） |
| **products.spec.ts** | 商品详情页加载 | **否**（Mock） |
| **owner-approval.spec.ts** | Owner 总控台风险统计 + Agent 建议表格 | **否**（Mock） |
| **owner-approval.spec.ts** | 建议状态标签正确渲染 | **否**（Mock） |
| **owner-approval.spec.ts** | 批准上架 → 确认弹窗 → 成功消息 | **否**（Mock） |
| sourcing.spec.ts | A8 选品采集完整流程（Mock） | **否**（Mock） |

### 何时跑 E2E

- **提交 PR 前**：跑所有 mock-based 测试（login、products、owner-approval、sourcing）
- **后端改动后**：跑 main-chain 主链路（需要后端 + 数据库运行）
- **CI 中**：`e2e-test` job 自动运行所有 mock-based 测试（`main-chain.spec.ts` 在后端不可用时 skip）
- **本地调试**：`npx playwright test --debug` 单步执行

## 测试设计原则

1. **Mock-based 测试**（`login.spec.ts`、`products.spec.ts`、`owner-approval.spec.ts`、`sourcing.spec.ts`）：
   - 使用 `page.route()` 拦截所有 API 调用
   - 使用 `page.addInitScript()` 注入 fake JWT token 以通过 AuthGuard
   - 无需后端运行，适合 CI 快速验证
   - 模拟真实用户操作流程（点击、填写、提交）
2. **Real-backend 测试**（`main-chain.spec.ts`）：
   - 需要后端 + 数据库运行
   - 后端不可达时自动 skip
   - 验证端到端数据流

## 故障排查

| 现象 | 原因 | 处理 |
|---|---|---|
| `Backend not reachable` skip | 后端没起或端口不对 | 确认 :8080 health，或设 `E2E_API_BASE` |
| login 失败 | 没有 /auth/register 路由 | 见 parity report，Go 端缺 register，需补或手动 seed 用户 |
| trace 页面无 events | orchestrator stub 未写 event | 检查 DB ai_trace_event 表 |
| action 批准按钮不可见 | action 状态不是 suggested | 检查 unified_action.status |
| Mock 测试首次运行失败 | Playwright 浏览器未安装 | `npx playwright install chromium` |
| 断言超时 | Mock 路由未正确拦截 | 确认 `page.route('**/api/**', ...)` 在 beforeEach 中注册 |
