# CLAUDE.md

> **当前唯一开发路径（2026-07-12）**：建设只供 Owner 本人使用的完整 AI 跨境电商经营平台。完整平台包含经营事实系统、经营决策系统、Owner AI 协作层和平台内核；完整平台是目的地，按完整纵向单元推进，小单元不是产品上限。任何计划、TODO、PR 和验收必须映射到 `docs/decisions/ADR-001-owner-complete-commerce-platform.md`。平台不服务外部软件用户，不规划 SaaS、多租户、订阅、计费、公共 API 或软件商业化。详见 `docs/SELF_USE_OPERATING_DIRECTION.md` 和 `CONTEXT.md`。

> **项目运行方法（2026-07-13）**：遵循 `docs/decisions/ADR-002-practice-cognition-operating-method.md`。系统建设、测试、部署和恢复只形成工程认识；达到安全可运行门槛后进行受控真实经营，外部事实与经济结果再修正经营和后续建设。当前执行入口为 `docs/plan/REAL_OPERATION_READINESS_PLAN.md`，不得用mock或测试冒充真实完成，也不得等待全部未来功能完成后才开始逐流程外部验收。

平台真相合同位于 `internal/domain/platformtruth/`，通过 `GET /api/v1/platform-truth` 和 Owner 页面 `/platform-truth` 只读展示事实等级、工程声明等级、系统边界、对象身份、来源规则、全领域处置及未知事项。合同测试必须覆盖 `internal/domain/` 全部目录；`delete` 分类不授权实际删除。领域职责变化时必须同步该合同；该治理合同的 `xiao_q_support` 为 `not_applicable`。

开始任何非平凡研究、规划、开发、审查、QA、发布或任务拆分前，必须按顺序完整阅读：`/Users/lc/gstack/ETHOS.md` → `docs/decisions/ADR-001-owner-complete-commerce-platform.md` → `docs/decisions/ADR-002-practice-cognition-operating-method.md` → `docs/plan/REAL_OPERATION_READINESS_PLAN.md` → `docs/research/project-truth-audit-2026-07-13.md` → `docs/research/project-truth-audit-2026-07-12.md` → `docs/research/project-truth-audit-2026-07-11.md`。不得依赖记忆摘要代替阅读。模块存在、测试通过、页面可见、mock 或 Agent 共识均不得升级为真实经营事实。代码、方向或现实状态变化时必须重新核验，而不是沿用旧完成声明。

当前经营工作可以按候选市场、Owner 决定、经营行动、订单、售后、结算、利润和下一步决定追踪，但这只是事实路径，不是工程意义上的经营闭环。这里的消费者和买家只是 Owner 自营商品业务的交易对手，不是凌镜的软件用户；商品购买事实不得写成凌镜的“外部需求验证”，也不得自动解释为经营假设的因果验证。

不得因为旧文档、旧代码或未来想象，主动恢复外部产品路线。旧材料中的外部客户、SaaS、设计伙伴、软件付费、跨客户聚合和商业化内容统一视为 `superseded`，只能用于历史追溯，不得进入当前计划、任务或验收标准。

Ozon 自动采集接口属于待按市场选择启用的平台连接器。没有明确决策用途和市场闸门时，不得启动平台商品采集。已有采集线索通过 `evidence_id` 引用不可变页面快照。

现有 `internal/domain/experiment/`、API `/api/v1/experiments` 和前端 `/experiments` 按“经营事实核验案卷”解释。`experiment_id` 关联机会、商品、订单、履约、售后、利润与现金只证明可追踪，不证明因果或反馈闭环。继续保留真实性、闸门、可信结算和同对象对账约束；但除非目标、可执行变量、真实市场作用、可靠观测、偏差判断、反馈规则和下一轮执行全部存在并验证，不得称其为经营闭环。

1688采集与受控草稿链：后端 `internal/domain/sourcing1688/`，API `/api/v1/sourcing-1688`，前端 `/sourcing1688`。Owner可在1688详情页主动点击插件，经 `POST /private-collections` 先保存Owner隔离的 `unverified_lead` 私人收藏；无需预先建立实验，页面字段最高为`quoted`。决定继续研究后才用 `POST /:id/task-links` 关联最新selected市场下经Owner批准的商品机会，并冻结机会及决定ID；`experiment_id`仅作追踪。受控采集、复核、草稿、验收和发布的每个升级边界都会重验该权限，市场或机会失效后fail closed。任务卡“精确成本 / 合规”通过 `GET /:id/task-links/:linkId/sku-mappings` 选择exact canonical SKU mapping，十项成本只用minor-unit整数与十进制字符串汇率证据；六类合规中`quoted`、过期、撤销或未经Owner批准的记录均保持blocker。`approved_draft` 仍保持listing=`draft`，不得自动发布。`extension_click`不等于既有`controlled_fetch`受控采集真实性；真实发布仍需独立审批、显式执行和后续对账。

插件在 `/settings/plugin` 完成Owner确认的设备配对，只使用 `/api/v1/extension/sourcing-1688` 和固定的 `sourcing1688.collect` 权限，不得接收或保存网页登录JWT。

候选市场比较统一使用 `internal/domain/demandcase/` 和 `/api/v1/demand-cases`。候选市场必须包含地区、消费者、需求场景和销售渠道；八个决策维度及独立反证未齐全时保持 `evidence_missing`。平台连接器、AI 推断、mock 或无来源数字不能通过确定性裁决。

Owner 从 `/demand-cases` 查看候选市场。AI 研究 run 必须使用三类固定契约并保存可重算 SHA-256 原始快照；内置公开研究批次只产生权限待验证基线，不得解释为俄罗斯/Ozon 已入选。

市场评估不等于 Owner 决定：`experiment_ready` 只表示研究材料可供审议。Owner 决定使用 `/api/v1/demand-cases/:id/owner-decisions` 单独保存；只有最新 selected 市场可在 `/api/v1/product-opportunities` 创建商品机会。商品机会经完整性检查和 Owner 批准后只进入货源研究，不触发采购、Listing、投放或外部发布。契约见 `docs/features/market-opportunity-owner-flow.md`。

采购/补货权威链位于 `internal/domain/purchase/`，Owner API 为 `/api/v1/purchase/authorities`。采购请求冻结 Owner supplier、canonical SKU mapping、精确 cost version、inventory、minor-unit amount/currency 和 request SHA-256；批准必须引用经营决定系统中 exact target/input hash 的 `selected` Owner 决定。外部提交、ordered/failed 和 partial/full receiving 只接受服务端哈希的不可变 `external_observed` 回执；库存只在保存真实 receiving fact 与幂等 ledger 的同一事务中增加。旧 `/purchase/orders` 写路径及 `supplychain.order.received` 库存旁路已冻结。

Owner 权威路由使用专用 RBAC 权限：`purchase.owner`、`business_feedback.owner`、`aftersales.owner`。迁移只给启用的 `owner/admin` 角色授权，`ops/viewer` 必须被拒绝；Owner 行隔离不能因有 RBAC 而移除。小Q通过登记 Capability 调用领域 Service，不借用 Owner HTTP 身份。

小Q是唯一面向 Owner 的经营 Agent，固定 ID `xiao_q`。后端 `internal/domain/xiaoq/`，API `/api/v1/xiao-q`，前端 `/xiaoq`。它只能调用按 `docs/governance/XIAOQ_CAPABILITY_CONTRACT.md` 登记的 Capability，并继续使用现有领域 Service/Command、RBAC、审批、审计和事实闸门；第一版模型运行设计见 `docs/architecture/XIAOQ_AGENT_RUNTIME_V1.md`。需求案件已实现并自动验证`agent_runtime_v1`，模型可选择两个Owner隔离的只读能力、接收真实结果并继续判断；真实Provider人工验收仍为unknown。其他目标仍是固定领域读取和单次模型回答，active Capability不等于已迁入Agent循环。新增功能必须声明 `xiao_q_support: active | deferred | not_applicable`；没有完成 Capability、权限和回归测试时不得声称已接入小Q。当前 active 能力包括需求案件、决策卡、trace-only `experiment` 案卷、1688受控内部草稿，以及按 Owner + exact `order_id` 读取的订单、库存、履约、售后、结算、最终利润和可归属现金事实；小Q也可读取经营决定案卷并保存固定为 `inferred`、绑定冻结事实的 AI 建议。它不能形成 Owner 决定或执行 Command，多订单结算批次现金不得归属单订单，`businessfeedback` 仍为 deferred。

Owner经营决策与反馈分别由 `internal/domain/businessdecision/` 和 `internal/domain/businessfeedback/` 承担。AI建议不等于Owner决定；受控行动必须与最新 `selected` 决定冻结的 capability/command/target/input hash 完全一致并经过审批；结果只记录支持、反证或冲突，不建立因果。旧 `experiment` 为 `trace_only` 事实核验案卷，不得作为经营授权或闭环证明。

商品图片工作台 `/product-images` 复用 `internal/domain/productimage/` 与 `/api/v1/product-images`。每个候选冻结 exact SKU、版本化配方与输出哈希；`POST /tasks/:id/feedback` 保存不可变选择/拒绝/返工，`GET /recipes/:recipe_key/summary?sku_id=` 实时汇总同 Owner、同 SKU 配方表现。付费派发结果未知后禁止重试，只能凭可信证据结案为追回输出、未扣费或已扣费但无可恢复输出。制作统计不是渠道或经营效果；Provider 未通过权利、预算和批准门禁时不得启用。`xiao_q_support: not_applicable`。

This file gives Claude Code-specific guidance for working in this repository.
Canonical cross-agent rules are in `AGENTS.md`; keep this file consistent with it.

## Onboarding

Before working, read [Agent Capabilities](./docs/AGENT_CAPABILITIES.md) — it lists
all MCP servers, API endpoints, CLI tools, database schemas, and development
commands available to you.

Before non-trivial development, refactor, review, QA, or release work, also read the governance documents:

- [Owner-First Development Protocol](./docs/governance/OWNER_FIRST_PROTOCOL.md) — the Owner describes business goals; Agents own technical translation and must report in business language.
- [Platform Constitution](./docs/governance/PLATFORM_CONSTITUTION.md) — highest-level platform rules, system layers, risk levels, forbidden actions, and Owner decision boundaries.
- [Agent Development Protocol](./docs/governance/AGENT_DEVELOPMENT_PROTOCOL.md) — multi-Agent roles, start checklist, review checklist, QA checklist, and handoff rules.
- [Kernel Contracts](./docs/governance/KERNEL_CONTRACTS.md) — EventBus, Command, Scheduler, ToolBridge, Approval, Audit, RBAC, Observability, and Migration contracts.

When these governance docs conflict with older project docs, follow the governance docs unless the Owner explicitly overrides them.

## Project

凌镜 LingMirror (technical name: MultiSell) — cross-border e-commerce AI AgentOS.
Version `v0.3.0.0` in `VERSION`, tracked on `main`.

## License And Ownership

This repository is proprietary and not open source. Do not add open-source
license language or publish/distribute project code unless the Owner explicitly
requests it. See `LICENSE`.

| Stack | Dir | Entry |
|-------|-----|-------|
| Backend | `backend-go/` | `cmd/server/main.go` — Go 1.25, Gin, GORM, PostgreSQL 15 |
| Frontend | `frontend-next/` | `src/app/` — Next.js 16, React 19, TypeScript, Ant Design 6 |
| Image Service | `services/image-service/` | `cmd/server/main.go` — private image execution, durable jobs/workers, safe blobs |

The legacy Prism runtime client, trigger route, Listing/loop injection, and `PRISM_*` configuration are retired and must not be restored to production paths. Historical `imagegen` data and the standalone `/Users/lc/prism` repository remain preserved.

API prefix: `/api/v1`. Health: `/api/health`. Swagger: `GET /swagger/index.html` (44 endpoints annotated). All non-auth endpoints require JWT.

## Commands

Image Service: `cd services/image-service && go test -race ./... && go vet ./...`

```bash
# Infrastructure
docker compose up -d                 # Postgres 15
docker compose up -d db              # Postgres only

# Backend
cd backend-go
go run cmd/server/main.go            # dev server
go test ./...                        # all tests
go test -v ./internal/domain/order/  # single package
go vet ./...                         # static analysis
go build -o bin/server cmd/server/main.go

# Frontend
cd frontend-next
npm run dev -- --hostname 127.0.0.1 --port 3000
npm run build
npm run lint                          # eslint, known failures
npm test                              # vitest

# E2E (separate sub-project under frontend-next/)
cd frontend-next/e2e && npx playwright test

# Smoke tests (end-to-end pipeline verification)
cd backend-go
./scripts/smoke_test_setup.sh           # setup test DB + seed data
./scripts/smoke_test.sh                 # run 10-step pipeline verification

```

New dev database: `multisell`. Migrations: `backend-go/migrations/` (run via Docker).

## Implementation Rigor

根据改动的模块选择节奏：

- **AI Platform** (`internal/ai/`, `internal/agent/`, `internal/agentos/`, `domain/agentrule/`, `domain/entropy/`, `domain/evolution/`, `domain/trustscore/`, `domain/actionpolicy/`, `domain/decision/`) + **Commerce** (`domain/order/`, `domain/shipping/`, `domain/settlement/`, `domain/finance/`, `domain/platformfee/`, `domain/exchangerate/`) — 先用 /plan → 写测试 → 小心实现。这是核心 + 钱走的地方。
- **所有其他模块** (CRUD 为主的 product catalog, integration, UI 等) — 直接按现有 pattern 快速实现，不需要过度设计。

## Backend Architecture

### Module Pattern

All domain modules under `internal/domain/*/` follow a consistent layout:

| File | Purpose |
|------|---------|
| `routes.go` | Gin route registration |
| `handler.go` | HTTP request/response mapping |
| `service.go` | Business logic |
| `model.go` | GORM models + request/response structs |

Modules register in `internal/httpx/router.go` under the JWT-protected `/api/v1` group. The router also wires event bus subscriptions, scheduler ticks, and the WebSocket hub.

### Standard Response Envelope

```go
response.Success(c, data)                       // {"code":0, "message":"ok", "data":...}
response.Error(c, http.StatusBadRequest, msg)   // {"code":400, "message":msg}
response.Paginated(c, data, total, page, size)  // + pagination fields
response.InternalError(c, err)                  // 500, masks details in release mode
```

Pagination: `common.ParsePagination(c)`, `common.ParseSort(c)` from `internal/common/`.

### Middleware Stack

`internal/httpx/middleware/`: `CORS` → `RequestID` → `Metrics` (opt-in) → `RecoveryWithSentry` → `Audit` (mutation logging). `Auth` (JWT) applied to the `/api/v1` protected group. Rate limiting in `ratelimit.go`.

### Platform Infrastructure (`internal/platform/`)

Six in-process coordination primitives + two compliance registries:

- **Event Bus** (`eventbus/bus.go`) — pub/sub with glob topic matching (`order.*`). Used for agent pipeline chains, scheduler ticks, and cross-module async events. ~15 subscriptions in `router.go`.
- **Command Dispatcher** (`command/command.go`) — typed handler registry bridging agent decisions to domain services: `stock_alert`, `replenish`, `price_review`, `listing_optimize`, `compliance_check`.
- **Scheduler** (`scheduler/`) — periodic task runner (5 min to 6 hr intervals). Publishes `scheduler.tick.{agent_id}` events.
- **ToolBridge** (`toolbridge/bridge.go`) — plugin-driver-based tool execution bridge that lets agents run external tools via registered plugins.
- **Action Catalog** (`actioncatalog/catalog.go`) — canonical registry of all action types with risk level, autonomy level, and approval requirements. Used by `command.DispatchSafe` for production mode gating.
- **Route Catalog** (`routecatalog/registry.go`) — HTTP route ↔ action type binding table. The approval middleware checks against this before allowing high-risk mutations.
- **Kill Switch** (`killswitch/switch.go`) — global production write kill switch. When active, all high-risk mutations return HTTP 503. Toggle via `POST /kill-switch/activate`.

### Compliance & Security Middleware (`internal/httpx/middleware/`)

- **Auth** (`auth.go`) — JWT validation on the `/api/v1` protected group.
- **RBAC** (`rbac.go`) — permission check per route group (`product.read`, `order.write`, etc.).
- **Audit** (`audit.go`) — operation log for all mutating requests (POST/PUT/PATCH/DELETE) + sensitive GET paths.
- **ApprovalRequired** (`approval.go`) — intercepts high-risk routes (price, inventory, order, integrations, rbac), validates `X-Approval-ID` header against the `approval_request` table. Checks kill switch first.
- **CI validation** — scripts/verify_pr.py parses PR body checkboxes; scripts/check_known_issues.sh checks KNOWN_ISSUES expiry; scripts/check_audit_coverage.sh validates middleware/registry alignment.

### Retired Multi-Agent Runtime

The A1-A12/G0-G3 scheduler chain, MoA, autonomy upgrades, AgentOS mutation routes and `agent.decided.*` DAG are no longer registered in the production router. Historical `/api/v1/ai/traces` and `/api/v1/ai/actions` remain read-only for audit. `internal/ai/`, `internal/agent/` and `internal/aios/` retain migration source only; do not add new runtime callers. The only active Owner Agent is `xiao_q`.

### WebSocket

`internal/realtime/` — authenticated live-update hub. Endpoint: `GET /ws`. The retired generic AI chat handler is no longer attached;小Q交互使用受控 `/api/v1/xiao-q` HTTP 契约。

### Configuration

`backend-go/configs/config.yaml`, overridden by env vars. Full struct in `internal/config/config.go`.

| Env | Config Path |
|-----|-------------|
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `database.*` |
| `JWT_SECRET` | `jwt.secret` |
| `SERVER_PORT` | `server.port` |
| `REDIS_ADDR` / `REDIS_PASSWORD` | `redis.*` |
| `SENTRY_DSN` | `sentry.dsn` |
| `CORS_ALLOWED_ORIGINS` | `cors.allowed_origins` |
| `METRICS_ENABLED` | `metrics.enabled` |

### Platform Integrations (`domain/integrations/`)

E-commerce platforms implement the `PlatformAdapter` interface in `adapter.go` (Publish, SyncStatus, ValidateCredentials, SyncInventory, PushTracking, FetchOrders, etc.). Register via `RegisterAdapter("lazada", &Adapter{})` in `init()` — see `registry.go`. Current: `ozon`, `shopee`.

### Auth & RBAC

- `internal/auth/` — JWT login/register/refresh (public routes)
- `internal/rbac/` — role-based permissions on protected routes

### Monitoring

- Prometheus metrics (opt-in): `/metrics` endpoint, middleware tracks request count/duration
- Sentry in Go (`middleware.RecoveryWithSentry`) and frontend (`@sentry/nextjs`)
- Audit middleware logs mutations to operationlog table

### Test DB Helper

```go
import "github.com/lingmirror/backend-go/internal/dbtest"

func TestX(t *testing.T) {
    db := dbtest.NewDB(t, &MyModel{})  // in-memory SQLite, per-call isolation
    svc := NewService(db, logger)
}
```

Safe for `t.Parallel()`. Also provides `NewLogger(t)`, `StringPtr`/`IntPtr`/`FloatPtr`.

## Frontend Architecture

```
src/
├── app/
│   ├── (auth)/login/       # public login page
│   ├── (main)/             # authenticated pages
│   │   ├── layout.tsx      # sidebar + header + content
│   │   ├── dashboard/
│   │   ├── products/
│   │   └── ...             # one dir per backend domain module
│   └── page.tsx            # landing → dashboard
├── components/
│   ├── auth/AuthGuard.tsx
│   ├── crud/CrudListPage.tsx       # reusable CRUD table + search + pagination
│   ├── layout/             # AntdProvider, AppHeader, AppSidebar, Breadcrumbs, CommandPalette
│   └── ui/                 # PageContainer, FilterBar, ConfirmDialog, StatCard, etc.
├── lib/
│   ├── api-client.ts       # fetch wrapper with JWT refresh + request dedup
│   ├── auth.ts             # token helpers (localStorage)
│   ├── query-client.ts     # TanStack React Query client
│   └── product-analysis.ts
├── stores/                 # Zustand (app, auth, permission)
├── config/menu.ts          # sidebar menu items
└── types/api.ts            # Result / PageResult types
```

Key deps: Ant Design 6, TanStack React Query 5, Zustand 5, dayjs, cmdk, reconnecting-websocket.

E2E tests: `frontend-next/e2e/` (Playwright, separate sub-project).

### Page Patterns

- Route → `src/app/(main)/{module}/page.tsx`
- Shared UI → `src/components/` only when reused
- Menu entry → `src/config/menu.ts` for top-level pages
- API calls → `apiClient` with `/v1/*` paths
- State → Zustand for global, React Query for server state
- Alias `@` → `src/`

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

Legacy Hermes Python (`backend/app/agent/`) is reference-only.

## Verification

```bash
cd backend-go && go test ./...   # pass
cd backend-go && go vet ./...    # pass
cd frontend-next && npm test     # pass
cd frontend-next && npm run build # pass
cd frontend-next && npm run lint  # fail — known issue
cd frontend-next/e2e && npx playwright test
```

## First Steps

1. `.codegraph/` exists — use `codegraph explore <query>` before grep/find/Read.
2. Check `git status --short` before edits. Preserve unrelated dirty files.
3. Follow existing module patterns — no unrequested abstractions.
4. Keep frontend API calls aligned with Go `/api/v1` routes.
5. Run the smallest verification covering your touch surface.
6. Pre-commit hooks in `.pre-commit-config.yaml`.
7. Do not touch `.kilo/worktrees/` — managed by external tooling.
8. **Documentation must stay in sync with code.** Any change to module names, API paths, directory layouts, or packages must update `AGENTS.md`, this file, and `docs/INDEX.md`. The `doc-links` CI job rejects PRs with dead references.

## Documentation

- `AGENTS.md` — canonical cross-agent project instructions. **Read the "Project Medical Record" section first — it lists known issues, what was fixed, and project rules.**
- `docs/governance/` — Owner-first and platform-first multi-Agent governance rules.
- `docs/INDEX.md` — full documentation index.
- `docs/PROJECT_STATUS.md` — current new-stack status.
- `docs/ACTIVE_STACK_POLICY.md` — active stack and legacy freeze policy.
- `docs/FRONTEND_PAGES_AND_ROUTING.md` — Next App Router page map.
- `docs/FUNCTION_INVENTORY.md` — complete feature inventory.
- `docs/features/` — feature specs and template.
- `docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md` — the only server deployment, recovery, testing, and operations runbook.
- `docs/api-inventory.md` — complete API route inventory (71+ modules).
- `backend-go/scripts/smoke_test.sh` — 10-step end-to-end pipeline verification.
- Swagger UI: `GET /swagger/index.html` (dev only, 44 annotated endpoints).

## Skill routing

When the user's request matches an available skill, invoke it via the Skill tool. When in doubt, invoke the skill.

Key routing rules:
- Product ideas/brainstorming → invoke /office-hours
- Strategy/scope → invoke /plan-ceo-review
- Architecture → invoke /plan-eng-review
- Design system/plan review → invoke /design-consultation or /plan-design-review
- Full review pipeline → invoke /autoplan
- Bugs/errors → invoke /investigate
- QA/testing site behavior → invoke /qa or /qa-only
- Code review/diff check → invoke /review
- Visual polish → invoke /design-review
- Ship/deploy/PR → invoke /ship or /land-and-deploy
- Save progress → invoke /context-save
- Resume context → invoke /context-restore
- Author a backlog-ready spec/issue → invoke /spec

## Design System
Always read DESIGN.md before making any visual or UI decisions.
All font choices, colors, spacing, and aesthetic direction are defined there.
Do not deviate without explicit user approval.
In QA mode, flag any code that doesn't match DESIGN.md.

## gstack (REQUIRED — global install)

**Before doing ANY work, verify gstack is installed:**

```bash
test -d ~/.claude/skills/gstack/bin && echo "GSTACK_OK" || echo "GSTACK_MISSING"
```

If GSTACK_MISSING: STOP. Do not proceed. Tell the user:

> gstack is required for all AI-assisted work in this repo.
> Install it:
> ```bash
> git clone --depth 1 https://github.com/garrytan/gstack.git ~/.claude/skills/gstack
> cd ~/.claude/skills/gstack && ./setup --team
> ```
> Then restart your AI coding tool.

Do not skip skills, ignore gstack errors, or work around missing gstack.

### Web Browsing

Use gstack's `/browse` skill for ALL web browsing. Never use `mcp__claude-in-chrome__*` tools — they are not available in this project.

### Available gstack Skills

After install, the following gstack skills are available:

/office-hours, /plan-ceo-review, /plan-eng-review, /plan-design-review, /design-consultation, /design-shotgun, /design-html, /review, /ship, /land-and-deploy, /canary, /benchmark, /browse, /connect-chrome, /qa, /qa-only, /design-review, /setup-browser-cookies, /setup-deploy, /setup-gbrain, /retro, /investigate, /document-release, /document-generate, /codex, /cso, /autoplan, /plan-devex-review, /devex-review, /careful, /freeze, /guard, /unfreeze, /gstack-upgrade, /learn

Use `~/.claude/skills/gstack/...` for gstack file paths (the global path).
