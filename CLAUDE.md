# CLAUDE.md

> **当前最高优先级方向（2026-07-12）**：凌镜是只供 Owner 本人使用的 AI 跨境商品经营内部系统。它不服务外部软件用户，不验证谁愿意为凌镜付费，也不规划客户访谈、设计伙伴、软件试点、SaaS、订阅、计费、公共 API 或软件商业化。真实商品成交或 Owner 自用效果不会自动改变这一边界；只有 Owner 新的明确决策才能改变方向。详见 `docs/SELF_USE_OPERATING_DIRECTION.md` 和 `CONTEXT.md`。

开始非平凡工作前，必须先读 `docs/research/project-truth-audit-2026-07-12.md` 确认当前产品边界，再读 `docs/research/project-truth-audit-2026-07-11.md` 核对代码与经营完成度证据。模块存在、测试通过、页面可见、mock 或 Agent 共识均不得升级为真实经营事实。代码、方向或现实状态变化时必须重新核验，而不是沿用旧完成声明。

当前唯一主线是：候选市场比较 → Owner 批准已选市场 → 商品经营证据与反证 → 商品机会 → Owner 批准的最小真实实验 → 非关联买家付款与签收 → 售后/争议关闭 → 最终贡献利润与现金对账 → 停止、换品、修正后再试或小幅加码。这里的消费者和买家只是 Owner 自营商品业务的交易对手，不是凌镜的软件用户；商品购买事实不得写成凌镜的“外部需求验证”。

不得因为旧文档、旧代码或未来想象，主动恢复外部产品路线。旧材料中的外部客户、SaaS、设计伙伴、软件付费、跨客户聚合和商业化内容统一视为 `superseded`，只能用于历史追溯，不得进入当前计划、任务或验收标准。

Ozon 自动采集接口属于待按市场选择启用的平台连接器。没有明确决策用途和市场闸门时，不得启动平台商品采集。已有采集线索通过 `evidence_id` 引用不可变页面快照。

经营实验统一事实链：后端 `internal/domain/experiment/`，API `/api/v1/experiments`，前端 `/experiments`。使用 `experiment_id` 关联机会、商品规格、供应、订单、履约、售后、利润与现金对象；闸门结果限定为 `pass / conditional / return / reject / expired`，证据作用限定为 `support / counter / conflict`，真实性限定为 `actual / quoted / estimated / unknown / mock / inferred`。普通录入不能直接声明 `actual`；最终利润封账必须校验可信已对账结算和最终订单利润记录，现金回收必须校验同一订单与结算的银行/现金交易。最终利润与现金回收不得合并为一个状态。

1688 受控草稿链：后端 `internal/domain/sourcing1688/`，API `/api/v1/sourcing-1688`，前端 `/sourcing1688`。仅允许已批准候选市场和已通过 opportunity gate 的 active 实验进入；保存不可变快照、同款与变化、供应商/合规、SKU 三段映射、实际图片处理、成本与渠道规则验证，再走 Owner 草稿审批。`approved_draft` 仍必须保持 listing=`draft`，不得自动发布。真实发布必须另建高风险 Owner 审批并再次显式执行；平台响应只记 `submitted` 或 `reconcile_required`，不能当作真实上线。真实商品人工验收完成前只能声明工程实现。

候选市场比较统一使用 `internal/domain/demandcase/` 和 `/api/v1/demand-cases`。候选市场必须包含地区、消费者、需求场景和销售渠道；八个决策维度及独立反证未齐全时保持 `evidence_missing`。平台连接器、AI 推断、mock 或无来源数字不能通过确定性裁决。

Owner 从 `/demand-cases` 查看候选市场。AI 研究 run 必须使用三类固定契约并保存可重算 SHA-256 原始快照；内置公开研究批次只产生权限待验证基线，不得解释为俄罗斯/Ozon 已入选。

小Q是唯一面向 Owner 的经营 Agent，固定 ID `xiao_q`。后端 `internal/domain/xiaoq/`，API `/api/v1/xiao-q`，前端 `/xiaoq`。它只能调用按 `docs/governance/XIAOQ_CAPABILITY_CONTRACT.md` 登记的 Capability，并继续使用现有领域 Service/Command、RBAC、审批、审计和事实闸门。新增功能必须声明 `xiao_q_support: active | deferred | not_applicable`；没有完成 Capability、权限和回归测试时不得声称已接入小Q。当前 active 能力为需求案件、决策卡、经营实验详情、实验闸门状态以及1688受控内部草稿只读。

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

API prefix: `/api/v1`. Health: `/api/health`. Swagger: `GET /swagger/index.html` (44 endpoints annotated). All non-auth endpoints require JWT.

## Commands

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

### Agent Pipeline Chain

Event bus subscriptions chain agent decisions automatically (defined in `router.go`):

```
A5 stock_alert (red)              → G3 discount_risk_check
G3 discount_risk_check (block)     → A6 profit_watch
A6 profit_watch (loss/threshold)   → A2 listing_optimize
G0 system_health (anomaly > 3)    → G1 dashboard_overview

All scheduled agents: G0/A4/G1/A5/G3/A6/A3/G2/A7/M1/trustscore/entropy
```

### WebSocket

`internal/realtime/` — hub for AI streaming and live updates. Endpoint: `GET /ws`. Integrated into AI route streaming for real-time chat/decision output.

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
| `internal/ai/` | LLM orchestration, chat, streaming, traces, provider abstraction |
| `internal/agent/` | Agent registry + execution entry points |
| `internal/agentos/` | Cockpit dashboard, work items, autonomy overview |
| `domain/agentrule/` | Agent behavior rules |
| `domain/entropy/` | Self-cleansing: SPC control, health scoring, defenses |
| `domain/evolution/` | Agent evolution nudges |
| `domain/logistics/` | Cross-border shipping rate engine (A10) |
| `domain/sourcing/` | Sourcing profit formula engine (A8) |
| `domain/trustscore/` | Trust score calculation, autonomy gating |
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
