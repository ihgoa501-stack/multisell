# 凌镜 LingMirror Roadmap

> 更新时间：2026-07-04
> 当前阶段：可信 AgentOS 执行门禁收口
> 当前执行口径见 [CURRENT_DIRECTION_AND_PRIORITIES.md](CURRENT_DIRECTION_AND_PRIORITIES.md)

## 当前阶段判断

项目已完成从 Python/FastAPI + Vue 到 Go/Gin + Next.js 的全站迁移。当前重点不再是“双栈迁移”，也不应继续平均增加模块数量，而是把新栈打磨成 Owner 能信任的 AgentOS：

- Agent 建议要可解释。
- 高风险动作要审批、审计、可追踪。
- Mock、sandbox、只读、生产模式要分清。
- 商品上架前决策和订单履约利润闭环要优先跑通。
- 真实外部平台写回要后置到安全门禁完成之后。

## Phase P0：可信执行门禁收口

状态：当前最高优先级。

目标：

- EventBus 和 Scheduler 生命周期随服务运行，而不是随 router 构造结束被取消。
- `/api/v1/ai/actions/:id/execute` 使用 `AgentAction + DispatchSafe + ActionCatalog + Approval + Audit` 的统一路径。
- approve / reject / execute 使用服务端认证身份，不信任请求体中的 operator。
- 高风险动作执行时复查 approval 仍然有效、未过期、未取消、未被替代。
- 外部平台发布、库存同步、追踪推送、订单状态变更默认 dry-run/sandbox/approval-gated。
- 审计日志对 API key、token、secret、password、credential 等字段脱敏。

验收：

- 有 focused backend tests 或明确的 runtime verification。
- Owner 能在 UI 上看到动作风险、执行后果、审批状态和审计入口。
- 价格、库存、订单、资金、发布、账号权限不允许绕过审批和审计。

## Phase P1：Owner 决策台产品化

状态：下一阶段。

目标：

- `/owner`、`/approval`、`/actions` 成为 Owner 的主工作路径。
- 所有高风险按钮使用同一确认组件。
- 英文状态枚举统一翻译成业务语言。
- 静态 mock 状态、sandbox 状态和真实 API 状态明确区分。
- CRUD 菜单退到业务底座，不再作为 Owner 首屏叙事。

验收：

- Owner 打开系统先看到“今天要决定什么、为什么、批准后会发生什么”。
- 每个建议都能采纳、拒绝、稍后处理，并留下记录。
- 每个失败都有业务影响和下一步处理入口。

## Phase P2：两条经营闭环跑通

状态：安全门禁后推进。

目标：

1. 商品闭环：

```text
候选商品 -> 完整度 -> 成本/物流/平台费 -> 利润 -> 上架建议 -> 审批 -> 上架任务 -> 复盘
```

2. 履约闭环：

```text
订单 -> 库存 -> 物流选择 -> 运费快照 -> 结算利润 -> 异常 -> Agent 建议 -> 审批/人工处理
```

验收：

- 每条链路有 demo data、API smoke、前端路径和业务语言说明。
- Agent 建议有输入数据、理由、风险等级、建议动作和结果追踪。

## Phase P3：质量门禁和生产化

状态：持续推进。

目标：

- `go test ./...`、`go vet ./...`、frontend tests/build/lint 的当前状态清晰。
- CI 接入核心检查。
- demo seed、acceptance script、migration rehearsal、rollback runbook 可用。
- 文档事实源和历史资料分层清楚。

验收：

- `docs/PROJECT_STATUS.md` 只描述当前事实和风险，历史更新迁到 changelog/orchestration。
- `docs/reference-module-catalog.md` 继续作为模块、API 和页面事实源。
- 旧方案、市场研究、历史计划均明确为参考资料。

## 历史路线 / 待整理 Backlog

以下 `Legacy Phase 0-6` 保留 2026-06-24 迁移收敛期的历史路线和待整理 backlog。
它们不再覆盖上方 `Phase P0-P3` 当前优先级。

如果 Legacy 内容与 `CURRENT_DIRECTION_AND_PRIORITIES.md`、治理文档或本文件上方 P0-P3 冲突，
以上方当前优先级为准。

## Legacy Phase 0：新栈事实源对齐

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

## Legacy Phase 1：质量门禁恢复

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

## Legacy Phase 2：API Contract 收敛

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

## Legacy Phase 3：新栈 Demo Seed / Acceptance

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

## Legacy Phase 4：测试覆盖补齐

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

## Legacy Phase 5：业务链路稳定化

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

## Legacy Phase 6：生产化准备

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

- `backend/`：已删除，仅可通过 git history 参考
- `frontend/`：已删除，仅可通过 git history 参考
- `docker-compose.legacy.yml`：已删除，仅可通过 git history 参考

新功能不得继续落到旧栈。
