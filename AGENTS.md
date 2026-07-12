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

任何计划、TODO、PR、QA 或发布必须映射到 ADR-001 的唯一开发路径；无法映射的工作不得进入当前开发队列。不得以“一个小工具已经够用”主动缩小 Owner 已确认的平台目标，也不得以“完整平台”为由加入外部 SaaS、多租户、订阅、计费、公共 API、更多 Agent/MoA、展示性仪表盘或其他无关能力。

商品消费者、平台买家、供应商和物流服务商只是 Owner 自营业务中的交易对手，不是凌镜的软件用户。对消费者付款、签收、售后和最终利润的核验只用于确认交易事实及经济结果；不得据此宣称经营假设获得因果验证，也不得写成凌镜的“外部需求验证”、软件市场验证或产品化信号。

开始任何非平凡研究、规划、开发、审查、QA、发布或任务拆分前，必须按顺序阅读：`/Users/lc/gstack/ETHOS.md` → `docs/decisions/ADR-001-owner-complete-commerce-platform.md` → `docs/research/project-truth-audit-2026-07-12.md` → `docs/research/project-truth-audit-2026-07-11.md`。不得依赖记忆摘要代替阅读。前两份确定建设原则与唯一开发路径，后两份核对产品边界、代码和经营完成度证据。严格区分 `policy / planned / implemented / automated_verified / manually_verified / external_observed / reconciled / mock / inferred / superseded`。不得把模块存在、测试通过、页面可见或多个 Agent 意见一致写成真实市场、真实成交、生产可用或最终利润已经成立。代码、方向或真实经营状态变化后，应重新核验并生成新的带日期审计，不能静默覆盖证据限制。

完整平台只指 Owner 自用经营能力完整。不得主动建设双产品、外部 SaaS、多租户、订阅计费、公共 API、外部 onboarding、设计伙伴、软件试点、跨客户聚合、未经市场选择的平台扩张、更多 Agent/MoA/自治升级或与当前纵向单元无关的大型视觉重构。真实商品成交或自用效果不会自动改变这一边界；只有 Owner 新的明确决策才能改变方向。自用不降低审批、审计和外部写安全要求。

旧文档、旧代码和旧研究中的外部客户、SaaS、设计伙伴、软件付费或商业化路线统一视为 `superseded` 历史材料，不得进入当前计划、TODO、验收标准或开发队列。

生产服务器初始化、SSH、部署、恢复、测试和回滚只有一个可执行入口：`docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md`。

现有 Ozon 采集器只是平台连接器，不是当前市场方向。`GET /api/v1/candidates/collection-evidence/:id` 只读取已有采集证据；除非 Ozon 对应的国家、消费者和渠道已通过市场闸门，否则不得把该接口列为当前采集任务。所有平台采集线索都必须通过 `evidence_id` 引用不可变快照，并声明它服务的经营决策。

现有 `internal/domain/experiment/`、`/api/v1/experiments` 和 `/experiments` 按“经营事实核验案卷”解释，技术命名暂不等于业务定义正确。每个案件以 `experiment_id` 关联现有业务对象；这种关联只支持追踪，不证明行动与订单或利润之间存在因果关系。证据作用区分 `support / counter / conflict`，真实性区分 `actual / quoted / estimated / unknown / mock / inferred`。普通录入不能直接声明 `actual`；利润与现金仍须保持可信来源、同一对象和对账约束。除非目标、可执行变量、真实市场作用、可靠观测、偏差判断、反馈规则和下一轮执行全部存在并验证，不得把该模块或其终局称为经营闭环。

1688 货源到待上架草稿的受控入口位于 `internal/domain/sourcing1688/`，API 根路径为 `/api/v1/sourcing-1688`，前端入口为 `/sourcing1688`。它只接受已通过 opportunity gate 的 active 实验及 `experiment_ready` 候选市场，生成不可变快照、同款/变化记录、供应商与合规证据、SKU 三段映射、实际图片处理、完整成本与确定性渠道规则验证，再进入 `editing → pending_approval → approved_draft`；草稿批准仍必须保持 `product_listing.status=draft`，不得调用平台适配器。`GET /:id/acceptance-report` 只按 Owner 读取持久化证据并逐项返回 15 项 `passed / blocked / unknown`；只有服务器受控采集入口留下的 `controlled_fetch` 来源可证明真实采集，手工/历史快照不能冒充。真实发布是另一条高风险流程：必须创建独立 Owner 审批、冻结并哈希请求，再由 Owner 显式执行；无错误平台响应只记 `submitted`，超时进入 `reconcile_required`，两者都不代表真实上线。工程实现不代表真实 1688 来源、图片权利、费用或渠道契约已获外部验证。

候选市场比较的统一入口位于 `internal/domain/demandcase/`，API 根路径为 `/api/v1/demand-cases`。每个候选必须明确“国家/地区 × 目标消费者 × 需求场景 × 销售渠道”，覆盖需求、竞争、获客、履约、合规、收款、售后和利润可验证性八个维度，并包含来自不同 run 的独立反证。关键维度为 unknown、mock、inferred 或缺少来源/观察时间时，只能保持 `evidence_missing`，不得生成可实验结论。

候选市场 Owner 页面为 `/demand-cases`。研究输入限定为 `scout_result / falsifier_result / data_reality_result`，原始 payload 与 SHA-256 快照必须一致，重复 run 幂等。内置静态公开资料基线只建立俄罗斯/Ozon 的权限待验证基线，不是实时研究，也不代表该市场已选中。

小Q是凌镜唯一面向 Owner 的经营 Agent，稳定 ID 为 `xiao_q`。后端入口位于 `internal/domain/xiaoq/`，API 根路径为 `/api/v1/xiao-q`，前端入口为 `/xiaoq`。小Q只能通过登记的 Capability 调用现有领域 Service/Command，不得直接访问任意数据库表或绕过 RBAC、审批、审计和经营状态机。新增功能必须声明 `xiao_q_support: active | deferred | not_applicable`；只有 Capability、权限、失败处理、证据追踪和回归测试齐全时才能标记 active。完整契约见 `docs/governance/XIAOQ_CAPABILITY_CONTRACT.md`。当前 active 能力为需求案件、决策卡、现有 `experiment` 经营事实案卷及闸门状态、1688受控内部草稿，以及从该案卷派生的脱敏订单履约、结算对账和最终利润只读；售后闭合、现金一致性及其他系统能力仍为 deferred 或 unknown。

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

API prefix: `/api/v1`. Health: `/api/health`. All non-auth endpoints require JWT.

## Commands

| Action | Command |
|---|---|
| Docker full stack | `docker compose up -d` |
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

Four in-process coordination primitives for agent-to-agent and agent-to-system communication:

- **Event Bus** (`eventbus/bus.go`) — pub/sub with glob topic matching (`order.*`). Used for agent pipeline chains, scheduler ticks, cross-module async events. ~15 subscriptions in `router.go`.
- **Command Dispatcher** (`command/command.go`) — typed handler registry: `stock_alert`, `replenish`, `price_review`, `listing_optimize`, `compliance_check`.
- **Scheduler** (`scheduler/`) — periodic task runner (5 min to 6 hr intervals). Publishes `scheduler.tick.{agent_id}` events.
- **ToolBridge** (`toolbridge/bridge.go`) — plugin-driver-based tool execution bridge for agents to run external tools.

### Agent Pipeline Chain

Event bus subscriptions chain agent decisions automatically (defined in `router.go`):

```
A5 stock_alert (red)              → G3 discount_risk_check
G3 discount_risk_check (block)     → A6 profit_watch
A6 profit_watch (loss/threshold)   → A2 listing_optimize
G0 system_health (anomaly > 3)    → G1 dashboard_overview

Scheduled agents: G0/A4/G1/A5/G3/A6/A3/G2/A7/M1/trustscore/entropy
```

### WebSocket

`internal/realtime/` — hub for AI streaming and live updates. Endpoint: `GET /ws`.

### Configuration

`backend-go/configs/config.yaml`, overridden by env vars:

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
| `internal/ai/` | LLM orchestration, chat, streaming, traces, provider abstraction |
| `internal/agent/` | Agent registry + execution |
| `internal/agentos/` | Cockpit dashboard, work items, autonomy |
| `domain/agentrule/` | Agent behavior rules |
| `domain/entropy/` | Self-cleansing: SPC control, health scoring |
| `domain/evolution/` | Agent evolution nudges |
| `domain/logistics/` | Cross-border shipping rate engine (A10) |
| `domain/sourcing/` | Sourcing profit formula engine (A8) |
| `domain/trustscore/` | Trust score + autonomy gating |
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
| P0 | Agent output is stub (fake data, not real LLM) | `orchestrator.go:172` — `synthesizeOutput()` |
| P1 | MoA aggregation is structured but still deterministic, not LLM-synthesized | `moa.go` — `synthesize()` returns structured findings/conflicts/recommendation |
| P1 | Owner dashboard /owner is Mock | `frontend-next/src/app/(main)/owner/` |
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
