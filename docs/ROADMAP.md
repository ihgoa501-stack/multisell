# 凌镜 LingMirror Roadmap

> 更新时间：2026-06-24
> 当前阶段：全站新栈迁移后收敛

## 当前阶段判断

项目已完成从 Python/FastAPI + Vue 到 Go/Gin + Next.js 的全站迁移。当前重点不再是“双栈迁移”，而是把新栈打磨到可持续交付状态：

- API 路径和前后端 contract 收敛
- 前端 lint 质量门禁恢复
- 新栈 demo seed / acceptance 重建
- 高风险业务模块测试补齐
- AgentOS、财务、发布、订单履约链路稳定化

## Phase 0：新栈事实源对齐

状态：进行中。

已完成：

- `README.md` 指向 `backend-go/` 和 `frontend-next/`。
- `AGENTS.md` 指向新栈。
- `CLAUDE.md` 已同步到新栈。
- `docs/PROJECT_STATUS.md` 已重写为当前状态。
- `docs/FRONTEND_PAGES_AND_ROUTING.md` 已重写为 Next App Router 路由说明。
- `docs/FUNCTION_INVENTORY.md` 已重写为新栈功能清单。
- `docs/UI_FRAMEWORK_GAP_ANALYSIS.md` 已改为新栈页面覆盖审计。

待完成：

- 清理或标注剩余旧栈历史文档。
- archived plans 保持历史原貌，但在索引中避免误导为当前待办。

## Phase 1：质量门禁恢复

状态：待做。

目标：

- `cd frontend-next && npm run lint` 通过。
- lint 修复不通过降级规则绕过完成。

当前主要问题：

- `no-explicit-any`
- `no-unused-vars`
- `react-hooks/set-state-in-effect`

建议优先级：

1. 修 `AuthGuard` hooks 规则。
2. 清理登录页、决策页、刊登任务详情、结算详情中的 `any`。
3. 清理未使用 import / 变量。

## Phase 2：API Contract 收敛

状态：待做。

目标：

- 前端所有 `apiClient` 调用统一使用 `/v1/*`。
- 后端所有业务 API 保持 `/api/v1/*`。
- 增加 smoke test 防止回退到 `/api/*`。

已知风险调用：

- `/ai/actions`
- `/policy/rules`
- `/evolution/nudges`
- `/trust-scores/summary`

## Phase 3：新栈 Demo Seed / Acceptance

状态：待做。

目标：

- 新增 Go demo seed。
- 新增 `/api/v1/*` acceptance script。
- 支持一键准备演示数据。

建议覆盖：

- 商品 / SKU / 库存 / 价格
- 平台 / 平台费用 / 刊登任务
- 物流报价
- 订单 / 订单导入
- 结算 / 财务
- 异常
- AI chat / trace / action
- AgentOS work items / entropy / trust score

## Phase 4：测试覆盖补齐

状态：待做。

后端优先模块：

- `listing`
- `listingtask`
- `inventory`
- `finance`
- `decision`
- `allocation`
- `agentos`
- `actionpolicy`
- `evolution`
- `entropy`
- `trustscore`

前端优先页面：

- 登录与 AuthGuard
- Dashboard
- 商品列表 / 商品详情
- 订单详情
- 刊登任务详情
- AI / Agent actions
- AgentOS work items
- 设置 / RBAC / policy

## Phase 5：业务链路稳定化

状态：进行中。

优先链路：

1. 商品 -> SKU -> 库存 -> 价格
2. SKU -> 物流报价 -> 平台费用 -> 上架前决策
3. 决策 -> 刊登任务 -> 发布动作
4. 订单 -> 库存 -> 结算 -> 财务账本
5. 异常 -> Agent action -> 审批 -> 执行 -> 复盘

每条链路需要：

- 后端 focused tests
- 前端 smoke / e2e
- demo data
- 文档中的 API 路径和页面路径一致

## Phase 6：生产化准备

状态：待做。

重点：

- CI 接入 Go test / vet / build。
- CI 接入 frontend test / build / lint。
- PostgreSQL migration rehearsal。
- Sentry release/source map 配置策略。
- Docker compose / prod compose 验证。
- 回滚 runbook 复核。

## 已完成迁移边界

旧栈状态：

- `backend/`：reference-only
- `frontend/`：reference-only
- `docker-compose.legacy.yml`：rollback/reference only

新功能不得继续落到旧栈。
