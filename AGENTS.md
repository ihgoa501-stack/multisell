# 凌镜 LingMirror — Agent Instructions

<!-- CODEGRAPH_START -->
## CodeGraph

This repository is indexed by CodeGraph (`.codegraph/` exists at the repo root). Use it before grep/find or opening source files when you need to understand or locate code:

- MCP tools: `codegraph_explore` answers most code questions in one call with relevant symbols, verbatim source, and call paths. `codegraph_node` reads one symbol or a whole file with line numbers.
- Shell fallback: `codegraph explore "<question or symbols>"` and `codegraph node <symbol-or-file>`.
- Skip CodeGraph only for files it does not index well, such as Markdown, JSON, TOML, YAML, lockfiles, and generated artifacts.
<!-- CODEGRAPH_END -->

## Project

### Current Owner Direction (2026-07-12)

凌镜唯一开发路径是建设一个只供 Owner 本人使用的完整 AI 跨境电商经营平台。完整平台覆盖经营事实系统、经营决策系统、Owner AI 协作层和平台内核；从市场与机会、商品与货源、渠道准备，到订单、库存、履约、售后、结算、利润、现金和下一次经营行动。完整平台是目的地，按可独立验收的完整纵向单元推进；小单元不是产品上限。权威路径见 `docs/decisions/ADR-001-owner-complete-commerce-platform.md`，产品边界见 `docs/SELF_USE_OPERATING_DIRECTION.md`，统一术语见 `CONTEXT.md`。

项目运行方法遵循 `docs/decisions/ADR-002-practice-cognition-operating-method.md`：先通过建设、测试、部署和恢复形成对系统真实能力的工程认识；相关纵向流程达到安全可运行门槛后，立即进行范围受控、可停止、可对账的真实经营验收，再用外部结果修正经营和后续建设。工程证据不能冒充经营证据，真实结果也不能掩盖工程缺陷。当前执行入口为 `docs/plan/REAL_OPERATION_READINESS_PLAN.md`；每项工作必须对应其中一个真实可运行缺口和完成门槛。

任何计划、TODO、PR、QA 或发布必须映射到 ADR-001 的唯一开发路径；无法映射的工作不得进入当前开发队列。不得以“一个小工具已经够用”主动缩小 Owner 已确认的平台目标，也不得以“完整平台”为由加入外部 SaaS、多租户、订阅、计费、公共 API、更多 Agent/MoA、展示性仪表盘或其他无关能力。

平台真相与领域分类合同位于 `internal/domain/platformtruth/`，只读 API 为 `GET /api/v1/platform-truth`，Owner 页面为 `/platform-truth`。它统一展示事实等级、工程声明等级、事实/决策/协作/内核/支撑边界、对象身份、来源规则、全部领域处置和仍然未知的事项。合同必须覆盖 `internal/domain/` 全部目录；新增或改变领域职责时必须同步，漏分会使测试失败。`delete` 只表示退出目标架构的建议，不授权删除代码或数据。`xiao_q_support` 当前为 `not_applicable`，小Q不得自行修改此治理合同。

商品消费者、平台买家、供应商和物流服务商只是 Owner 自营业务中的交易对手，不是凌镜的软件用户。对消费者付款、签收、售后和最终利润的核验只用于确认交易事实及经济结果；不得据此宣称经营假设获得因果验证，也不得写成凌镜的“外部需求验证”、软件市场验证或产品化信号。

开始任何非平凡研究、规划、开发、审查、QA、发布或任务拆分前，必须按顺序阅读：`/Users/lc/gstack/ETHOS.md` → `docs/decisions/ADR-001-owner-complete-commerce-platform.md` → `docs/decisions/ADR-002-practice-cognition-operating-method.md` → `docs/plan/REAL_OPERATION_READINESS_PLAN.md` → `docs/research/project-truth-audit-2026-07-13.md` → `docs/research/project-truth-audit-2026-07-12.md` → `docs/research/project-truth-audit-2026-07-11.md`。不得依赖记忆摘要代替阅读。ADR-001确定目的地与范围，ADR-002和执行计划确定推进方法，带日期审计核对当前工程、运行和经营证据。严格区分 `policy / planned / implemented / automated_verified / manually_verified / external_observed / reconciled / mock / inferred / superseded`。不得把模块存在、测试通过、页面可见或多个 Agent 意见一致写成真实市场、真实成交、生产可用或最终利润已经成立。代码、方向或真实经营状态变化后，应重新核验并生成新的带日期审计，不能静默覆盖证据限制。

完整平台只指 Owner 自用经营能力完整。不得主动建设双产品、外部 SaaS、多租户、订阅计费、公共 API、外部 onboarding、设计伙伴、软件试点、跨客户聚合、未经市场选择的平台扩张、更多 Agent/MoA/自治升级或与当前纵向单元无关的大型视觉重构。真实商品成交或自用效果不会自动改变这一边界；只有 Owner 新的明确决策才能改变方向。自用不降低审批、审计和外部写安全要求。

旧文档、旧代码和旧研究中的外部客户、SaaS、设计伙伴、软件付费或商业化路线统一视为 `superseded` 历史材料，不得进入当前计划、TODO、验收标准或开发队列。

生产服务器初始化、SSH、部署、恢复、测试和回滚只有一个可执行入口：`docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md`。

现有 Ozon 采集器只是平台连接器，不是当前市场方向。`GET /api/v1/candidates/collection-evidence/:id` 只读取已有采集证据；除非 Ozon 对应的国家、消费者和渠道已通过市场闸门，否则不得把该接口列为当前采集任务。所有平台采集线索都必须通过 `evidence_id` 引用不可变快照，并声明它服务的经营决策。

现有 `internal/domain/experiment/`、`/api/v1/experiments` 和 `/experiments` 按“经营事实核验案卷”解释，技术命名暂不等于业务定义正确。每个案件以 `experiment_id` 关联现有业务对象；这种关联只支持追踪，不证明行动与订单或利润之间存在因果关系。证据作用区分 `support / counter / conflict`，真实性区分 `actual / quoted / estimated / unknown / mock / inferred`。普通录入不能直接声明 `actual`；利润与现金仍须保持可信来源、同一对象和对账约束。除非目标、可执行变量、真实市场作用、可靠观测、偏差判断、反馈规则和下一轮执行全部存在并验证，不得把该模块或其终局称为经营闭环。

1688 货源入口位于 `internal/domain/sourcing1688/`，API 根路径为 `/api/v1/sourcing-1688`，前端入口为 `/sourcing1688`。Owner 可在1688商品页主动点击插件，通过 `POST /private-collections` 先保存为Owner隔离的 `unverified_lead` 私人收藏；这一步不要求候选市场或实验，页面字段最高为 `quoted`，不得冒充商品机会、可信货源或受控采集证据。Owner决定继续研究后，才通过 `POST /:id/task-links` 关联最新 `selected` 市场下经 Owner 批准的商品机会，并冻结 `product_opportunity_id + opportunity_decision_id`；`experiment_id` 仅保留事实追踪，不能授权。受控采集、复核、草稿编辑、草稿审批、验收读取和发布请求/批准/执行都会重新核验该冻结授权，市场暂停/拒绝或机会失效后 fail closed。流程继续生成不可变快照、同款/变化、供应商与合规、SKU三段映射、图片处理、完整成本和渠道规则，再进入 `editing → pending_approval → approved_draft`。任务卡内的“精确成本 / 合规”工作台通过 `GET /:id/task-links/:linkId/sku-mappings` 选择 exact canonical SKU mapping；成本只提交整数 minor-unit 与十进制字符串汇率证据，合规只有当前、未撤销且经 Owner 批准的 `actual` 证据可通过，`quoted` 始终是 blocker。草稿批准仍保持 `product_listing.status=draft`；真实发布继续要求独立Owner审批和显式执行。`extension_click` 只表示Owner保存过页面声明，只有既有 `controlled_fetch` 来源可满足受控采集闸门。

插件在 `/settings/plugin` 完成Owner确认的设备配对，只使用 `/api/v1/extension/sourcing-1688` 和固定的 `sourcing1688.collect` 权限，不得接收或保存网页登录JWT。

候选市场比较的统一入口位于 `internal/domain/demandcase/`，API 根路径为 `/api/v1/demand-cases`。每个候选必须明确“国家/地区 × 目标消费者 × 需求场景 × 销售渠道”，覆盖需求、竞争、获客、履约、合规、收款、售后和利润可验证性八个维度，并包含来自不同 run 的独立反证。关键维度为 unknown、mock、inferred 或缺少来源/观察时间时，只能保持 `evidence_missing`，不得生成可实验结论。

候选市场 Owner 页面为 `/demand-cases`。研究输入限定为 `scout_result / falsifier_result / data_reality_result`，原始 payload 与 SHA-256 快照必须一致，重复 run 幂等。内置静态公开资料基线只建立俄罗斯/Ozon 的权限待验证基线，不是实时研究，也不代表该市场已选中。

市场研究评估与 Owner 决定必须分开。`experiment_ready` 仅为兼容技术状态，业务含义是“研究材料可供 Owner 审议”，不是已选市场。Owner 通过 `POST /api/v1/demand-cases/:id/owner-decisions` 保存不可变、幂等的 `selected / rejected / paused / request_more_evidence` 决定。只有最新 `selected` 决定才能通过 `/api/v1/product-opportunities` 创建 Owner 隔离的商品机会；机会必须保存消费者问题、商品解法、渠道、价值/价格假设、来源、真实性、最强反证、unknown 和停止线，经完整性检查达到 `ready_for_owner` 后再由 Owner 批准。批准只授权进入货源研究，不得自动创建采购、Listing、投放或外部发布。完整契约见 `docs/features/market-opportunity-owner-flow.md`。

采购/补货权威入口位于 `internal/domain/purchase/`，API 根路径为 `/api/v1/purchase/authorities`。请求必须绑定同一 Owner 的权威 supplier、canonical SKU mapping、精确 cost version 和 inventory，金额只使用 minor unit + currency。`requested` 只有在经营决定系统存在绑定 exact purchase ID 与 request SHA-256 的最新 `selected` Owner 决定后才能批准；外部提交、供应商已下单/失败及部分/全部收货只由服务端哈希的不可变 `external_observed` 回执推进。库存增加必须和真实收货 fact、幂等 receipt ledger 在同一事务完成。旧 `/purchase/orders` mutation 路径固定返回 Gone，旧 `supplychain.order.received` 事件不再修改库存。

Owner 权威 HTTP 路由还必须通过专用 RBAC capability：采购 `purchase.owner`、经营行动反馈 `business_feedback.owner`、售后处置 `aftersales.owner`。权限只授予启用的 `owner/admin` 角色；`ops/viewer` 即使知道路径也必须返回 403。领域 Service 的 Owner 行隔离仍是第二道边界。小Q已登记的只读或建议能力继续调用领域 Service，不通过冒充 Owner HTTP 权限绕行。

小Q是凌镜唯一面向 Owner 的经营 Agent，稳定 ID 为 `xiao_q`。后端入口位于 `internal/domain/xiaoq/`，API 根路径为 `/api/v1/xiao-q`，前端入口为 `/xiaoq`。小Q只能通过登记的 Capability 调用现有领域 Service/Command，不得直接访问任意数据库表或绕过 RBAC、审批、审计和经营状态机。新增功能必须声明 `xiao_q_support: active | deferred | not_applicable`；只有 Capability、权限、失败处理、证据追踪和回归测试齐全时才能标记 active。完整契约见 `docs/governance/XIAOQ_CAPABILITY_CONTRACT.md`，第一版模型运行架构见 `docs/architecture/XIAOQ_AGENT_RUNTIME_V1.md`。需求案件已实现并自动验证`agent_runtime_v1`：模型可在两个Owner隔离的只读能力中选择、接收真实结果并继续判断，且受严格schema、Target绑定、Trace和硬停止控制；真实Provider人工验收仍为unknown。其他目标仍是固定领域读取和单次模型回答，active Capability不等于已迁入Agent循环。当前 active 能力为需求案件、决策卡、trace-only `experiment` 案卷及历史闸门、1688受控内部草稿，以及按 Owner + exact `order_id` 读取的不可变订单事实、库存账本、承运商事件、售后终局、平台结算、最终利润和可归属现金对账；另可读取经营决定案卷并保存绑定冻结事实、固定为 `inferred` 的 AI 建议。旧 experiment 派生经营闭环能力已退役；小Q不能形成 Owner 决定、执行 Command 或把多订单批次现金归属单订单，`businessfeedback` 仍为 deferred。

Owner 经营决策权威位于 `internal/domain/businessdecision/`，受控行动与反馈位于 `internal/domain/businessfeedback/`。AI 建议、Owner 决定、Command 执行和结果观察必须分层保存；`selected` 决定只授权其冻结的 exact capability/command/target/input hash。结果只可记录为 `support / counter / conflict`，不得宣称因果成立。旧 `experiment` 始终是 `trace_only` 经营事实核验案卷，不能写最终决定或反馈闭环终态。

正式 Owner 导航只展示真实纵向经营入口；明确标记为 Mock、Sandbox 或壳的演示入口不得出现。物流读取要求 `shipping.read`，物流变更要求只授予启用 Owner/Admin 的 `shipping.write`；固定返回成功的 mock carrier 路由只允许 development，acceptance 与 production 不注册。

凌镜 LingMirror (technical name: MultiSell) — cross-border e-commerce AI AgentOS.
Version `v0.3.0.0`.

## License And Ownership

This repository is proprietary and not open source. Do not add open-source
license language or publish/distribute project code unless the Owner explicitly
requests it. See `LICENSE`.

## Governance First

This repository uses an Owner-first, platform-first multi-Agent workflow. Before non-trivial development, refactor, review, QA, or release work, read the governance documents:

- `docs/governance/OWNER_FIRST_PROTOCOL.md` — the Owner describes business goals; Agents own technical translation and must report in business language.
- `docs/governance/PLATFORM_CONSTITUTION.md` — highest-level platform rules: system layers, risk levels, forbidden actions, and Owner decision boundaries.
- `docs/governance/AGENT_DEVELOPMENT_PROTOCOL.md` — multi-Agent roles, start checklist, review checklist, QA checklist, and handoff rules.
- `docs/governance/KERNEL_CONTRACTS.md` — EventBus, Command, Scheduler, ToolBridge, Approval, Audit, RBAC, Observability, and Migration contracts.

When these governance docs conflict with older project docs, follow the governance docs unless the Owner explicitly overrides them.

| Stack | Dir | Entry |
|-------|-----|-------|
| Backend | `backend-go/` | `cmd/server/main.go` — Go 1.25, Gin, GORM, PostgreSQL 15 |
| Frontend | `frontend-next/` | `src/app/` — Next.js 16, React 19, TypeScript, Ant Design 6 |
| Image Service | `services/image-service/` | `cmd/server/main.go` — Owner内部图片执行、持久化Job/Worker和安全Blob；不负责发布裁决 |

旧 Prism 运行客户端、触发路由、Listing/loop 注入和 `PRISM_*` 配置已退役；不得重新接入生产路径。历史 `imagegen` 数据与独立 `/Users/lc/prism` 仓库仍保留。

商品图片 Owner 工作台为 `/product-images`，后端根路径为 `/api/v1/product-images`。任务必须冻结 exact `sku_id`、版本化 `recipe_manifest` 和哈希；`POST /tasks/:id/feedback` 保存不可变选择/拒绝/返工反馈，`GET /recipes/:recipe_key/summary?sku_id=` 只按当前 Owner、exact SKU 和配方实时聚合。付费派发结果未知后禁止重试；只能凭可信证据结案为追回输出、未扣费或已扣费但无可恢复输出。该统计是制作反馈，不证明真实渠道或经营效果；外部 Provider 未通过配置、预算、权利与批准门禁时保持不可用。`xiao_q_support` 当前为 `not_applicable`。

API prefix: `/api/v1`. Health: `/api/health`. All non-auth endpoints require JWT.

## Commands

| Action | Command |
|---|---|
| Docker full stack | `docker compose up -d` |
| Image Service test | `cd services/image-service && go test -race ./... && go vet ./...` |
| Docker DB only | `docker compose up -d db` |
| Backend dev | `cd backend-go && go run cmd/server/main.go` |
| Backend test all | `cd backend-go && go test ./...` |
| Backend test pkg | `cd backend-go && go test -v ./internal/domain/order/` |
| Backend vet | `cd backend-go && go vet ./...` |
| Backend build | `cd backend-go && go build -o bin/server cmd/server/main.go` |
| Frontend dev | `cd frontend-next && npm run dev -- --hostname 127.0.0.1 --port 3000` |
| Frontend build | `cd frontend-next && npm run build` |
| Frontend lint | `cd frontend-next && npm run lint` |
| Frontend test | `cd frontend-next && npm test` |
| E2E | `cd frontend-next/e2e && npx playwright test` |

New dev database: `multisell`. Migrations under `backend-go/migrations/`.

## Backend Architecture

### Module Pattern

Every domain module under `internal/domain/*/` follows a consistent layout: `routes.go` (route registration), `handler.go` (HTTP mapping), `service.go` (business logic), `model.go` (GORM models + request/response structs). Modules register in `internal/httpx/router.go`.

### Standard Response Envelope

```go
response.Success(c, data)                       // {"code":0, "message":"ok", "data":...}
response.Error(c, http.StatusBadRequest, msg)
response.Paginated(c, data, total, page, size)  // + pagination fields
response.InternalError(c, err)                  // 500, masked in release mode
```

Pagination: `common.ParsePagination(c)`, `common.ParseSort(c)`.

### Middleware Stack

`internal/httpx/middleware/`: CORS → RequestID → Metrics (opt-in) → RecoveryWithSentry → Audit (mutation logging). JWT `Auth` on the `/api/v1` protected group. Rate limiting via `ratelimit.go`.

### Platform Infrastructure (`internal/platform/`)

Four in-process coordination primitives for deterministic domain coordination and xiao_q capabilities:

- **Event Bus** (`eventbus/bus.go`) — pub/sub with glob topic matching (`order.*`). Used for scheduler ticks and cross-module async events.
- **Command Dispatcher** (`command/command.go`) — typed handler registry: `stock_alert`, `replenish`, `price_review`, `listing_optimize`, `compliance_check`.
- **Scheduler** (`scheduler/`) — deterministic periodic task runner. It no longer schedules the retired A/G Agent roster.
- **ToolBridge** (`toolbridge/bridge.go`) — controlled bridge for approved external tools.

### Retired Multi-Agent Runtime

The A1-A12/G0-G3 scheduler chain, MoA, autonomy upgrades, AgentOS mutation routes and `agent.decided.*` DAG are no longer registered in the production router. Historical `/api/v1/ai/traces` and `/api/v1/ai/actions` remain read-only for audit. `internal/ai/`, `internal/agent/` and `internal/aios/` retain migration source only; do not add new runtime callers. The only active Owner Agent is `xiao_q`.

### WebSocket

`internal/realtime/` — authenticated live-update hub. Endpoint: `GET /ws`. The retired generic AI chat handler is no longer attached;小Q交互使用受控 `/api/v1/xiao-q` HTTP 契约。

### Configuration

`backend-go/configs/config.yaml`, overridden by env vars:

| Env | Config Path |
|-----|-------------|
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `database.*` |
| `JWT_SECRET` | `jwt.secret` |
| `SERVER_PORT` | `server.port` |
| `REDIS_ADDR` / `REDIS_PASSWORD` | `redis.*` |
| `SENTRY_DSN` | `sentry.dsn` |
| `DEPLOYMENT_ENVIRONMENT` | `server.deployment_environment` (`development` / `acceptance` / `production`; empty follows server mode) |
| `CORS_ALLOWED_ORIGINS` | `cors.allowed_origins` |
| `METRICS_ENABLED` | `metrics.enabled` |

### Platform Integrations (`domain/integrations/`)

E-commerce platforms implement the `PlatformAdapter` interface (Publish, SyncStatus, SyncInventory, FetchOrders, etc.). Register via `RegisterAdapter("{code}", &Adapter{})` in `init()`. Current: `ozon`, `shopee`.

### Auth & RBAC

- `internal/auth/` — JWT login/register/refresh (public routes).
- `internal/rbac/` — role-based permissions on protected routes.

### Monitoring

- Prometheus metrics (opt-in): `/metrics` endpoint + request tracking middleware.
- Sentry in Go (`middleware.RecoveryWithSentry`) and frontend (`@sentry/nextjs`).
- Audit middleware logs mutations to `operationlog` table.

### Test DB Helper

```go
import "github.com/lingmirror/backend-go/internal/dbtest"

func TestX(t *testing.T) {
    db := dbtest.NewDB(t, &MyModel{})  // in-memory SQLite, isolated per call
    svc := NewService(db, logger)
}
```

Safe for `t.Parallel()`. No PostgreSQL needed.

## Frontend

```
src/
├── app/(auth)/login/       # public login
├── app/(main)/{mod}/page.tsx  # one per domain module
├── components/
│   ├── crud/CrudListPage.tsx   # reusable CRUD table + search
│   ├── layout/                 # AntdProvider, AppHeader, AppSidebar
│   └── ui/                     # PageContainer, FilterBar, ConfirmDialog, etc.
├── lib/api-client.ts        # fetch wrapper with JWT refresh + dedup
├── stores/                  # Zustand (app, auth, permission)
├── config/menu.ts           # sidebar items
└── types/api.ts             # Result / PageResult types
```

Alias `@` → `src/`. E2E: `frontend-next/e2e/` (Playwright).

## Conventions

- Non-public endpoints must use JWT auth. Mutation routes should be auditable.
- Use GORM transactions for multi-step state changes.
- Add focused Go tests near touched behavior.
- Frontend: keep `npm run build` and `npm run lint` green (lint has known issues).
- Do not touch `.kilo/worktrees/` — managed by external tooling.
- **Documentation must stay in sync with code.** Any PR that changes module names, API paths, directory layouts, or removes/adds packages must update `CLAUDE.md`, `AGENTS.md`, and `docs/INDEX.md` as needed. PRs with stale doc references will be rejected by CI (`doc-links` job).

## AI & AgentOS

| Package | Purpose |
|---------|---------|
| `internal/domain/xiaoq/` | 唯一 Owner Agent、Capability Catalog、Agent Runtime 与 Trace |
| `internal/ai/` | Provider adapter及只读历史Trace；旧A/G Orchestrator已失败关闭 |
| `internal/agent/`, `internal/agentos/` | 冻结的旧运行外壳，不再注册生产路由 |
| `domain/agentrule/`, `domain/entropy/`, `domain/evolution/` | 冻结的旧自治治理实现，不再注册生产路由 |
| `domain/logistics/` | Cross-border shipping rate engine (A10) |
| `domain/sourcing/` | 冻结的旧 A8/mock 市场来源实现，不再注册生产路由 |
| `domain/trustscore/` | 冻结的旧自治等级实现，不再注册生产路由 |
| `domain/actionpolicy/` | Action approval policy |

## Documentation

- `CLAUDE.md` — Claude Code guidance (keep consistent).
- `docs/governance/` — Owner-first and platform-first multi-Agent governance rules.
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md` — current execution direction, safety priorities, and documentation cleanup rules.
- `docs/INDEX.md` — full doc index.
- `docs/PROJECT_STATUS.md` — current new-stack status.
- `docs/ACTIVE_STACK_POLICY.md` — active/legacy policy.
- `docs/CODEBASE_ANALYSIS.md` — codebase analysis snapshot, knowledge graph usage, and regeneration guidance.
- `docs/FRONTEND_PAGES_AND_ROUTING.md` — page map.

## Project Medical Record

> Last updated: 2026-07-06. Read this before any work. It prevents repeating mistakes.
> For the latest verification status, run: `cd backend-go && go test ./...`

### ✅ What Works (verified this session)

- `go build ./...` — passes
- `go vet ./...` — passes
- `go test ./...` — 96 packages green, 11 pkgs no-test (107 total), 0 failures
- Frontend: `npm run dev` — starts on port 3001 (but dev server can exit unexpectedly)
- Login: the historical `admin / admin123456` credential is no longer valid in the current local database; use an existing valid Owner account or the approved credential-reset procedure.
- All 30+ frontend pages render (product hub, categories, brands, SKU, inventory, orders, agents, AI command center, etc.)
- Seed data in DB: 5 categories, 3 brands, 2 platforms (Ozon + Shopee), product + SKU + inventory

### 🐛 Known Issues (unfixed)

| Priority | Issue | Location |
|----------|-------|----------|
| P0 | 小Q Agent Runtime 尚未完成一次受预算限制的真实 Provider 人工验收 | `docs/architecture/XIAOQ_AGENT_RUNTIME_V1.md` |
| P1 | 旧 Owner mock 工作台已从正式入口退役；`/owner` 仅跳转 `/platform-truth`，后端 fixture 路由只允许 development | `frontend-next/src/app/(main)/owner/`, `backend-go/internal/httpx/router.go` |
| P2 | Only 3 platform adapters (Ozon + Shopee + Shopify), still thinly tested | `domain/integrations/` |
| P2 | Frontend dev server has no watchdog — exits silently | `npm run dev` process |
| P3 | No real CI trigger yet (doc-links job added but not tested) | `.github/workflows/ci.yml` |

### 🛠️ What Was Fixed (2026-07-06)

- **工程可信度恢复**: `go build ./...` / `go vet ./...` / `go test ./...` 全绿
  - 修复 `internal/common/types.go` 中 `UserIDFromCtx` 重复定义（删除第二个副本）
  - 清理 `internal/ai/` 6 个文件共 19 处 merge conflict（handler.go, service.go, orchestrator.go, routes.go, model.go, ai_test.go）
  - 冲突来自 merge commit `964e0624`（合并远程 main v0.4.0），保留 HEAD 版本
- Merge conflicts in `routes.go` + `router.go` (HEAD won over worktree-wf)
- Duplicate `UserIDFromCtx` in `types.go` (kept first, deleted second)
- AuthGuard SSR crash (`useState` reading localStorage during server render → `useEffect` + `mounted`)
- RBAC endpoint 404 (frontend called `/v1/rbac/current/permissions` but route was unregistered)
- Inventory + product-hub 403 (operator users not linked to `ops` RBAC role → migration `000064`)
- Supplier test failure (handler read `c.Query()` but route used path param → fixed to `c.Param()`)
- Owner test failure (test CREATE TABLE missing `requester_user_id` + `reviewer_user_id` columns)
- Doc dead links removed from CLAUDE.md, INDEX.md, KERNEL_CONTRACTS.md

### 🏛️ Project Rules (Do Not Violate)

1. **Doc sync is mandatory.** Changing module names, API paths, or package layout requires updating AGENTS.md, CLAUDE.md, and docs/INDEX.md. CI `doc-links` job rejects stale references.
2. **Do not touch `.kilo/worktrees/`** — managed by external tooling.
3. **Do not rewrite history.** No `git rebase` on shared branches (main, feat/*, codex/*).
4. **Test before commit.** Minimum: `go build ./...` + `go vet ./...` + `go test ./...` for touched packages.
5. **No unrequested refactors.** Match existing patterns. Drive-by style changes are rejected.
6. **Old-stack docs (superpowers/plans/ etc.) are marked deprecated** — do not treat as executable instructions.
7. **Keep frontend API path format consistent:** `/api/v1/*` prefix, apiClient with `/v1/*` paths.

### 🔔 Cron Jobs

| Name | Schedule | What it does |
|------|----------|-------------|
| 文档链接审计 | Mon 9:00 | Checks AGENTS.md/CLAUDE.md/INDEX.md for dead links |
| 依赖安全检查 | Mon 10:00 | go mod verify + npm audit |
| 每周健康检查 | Mon 9:00 | Full test suite + git status + service check |
- `docs/features/` — feature specs; use `TEMPLATE.md`.
