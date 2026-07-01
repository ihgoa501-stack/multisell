# CLAUDE.md

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
Version `v0.2.1` in `VERSION`, tracked on `main`.

| Stack | Dir | Entry |
|-------|-----|-------|
| Backend | `backend-go/` | `cmd/server/main.go` — Go 1.25, Gin, GORM, PostgreSQL 15 |
| Frontend | `frontend-next/` | `src/app/` — Next.js 16, React 19, TypeScript, Ant Design 6 |

API prefix: `/api/v1`. Health: `/api/health`. All non-auth endpoints require JWT.

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

Four in-process coordination primitives for agent-to-agent and agent-to-system communication:

- **Event Bus** (`eventbus/bus.go`) — pub/sub with glob topic matching (`order.*`). Used for agent pipeline chains, scheduler ticks, and cross-module async events. ~15 subscriptions in `router.go`.
- **Command Dispatcher** (`command/command.go`) — typed handler registry bridging agent decisions to domain services: `stock_alert`, `replenish`, `price_review`, `listing_optimize`, `compliance_check`.
- **Scheduler** (`scheduler/`) — periodic task runner (5 min to 6 hr intervals). Publishes `scheduler.tick.{agent_id}` events.
- **ToolBridge** (`toolbridge/bridge.go`) — plugin-driver-based tool execution bridge that lets agents run external tools via registered plugins.

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
cd frontend-next && npm run lint  # pass
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

## Documentation

- `AGENTS.md` — canonical cross-agent project instructions.
- `docs/governance/` — Owner-first and platform-first multi-Agent governance rules.
- `docs/INDEX.md` — full documentation index.
- `docs/PROJECT_STATUS.md` — current new-stack status.
- `docs/ACTIVE_STACK_POLICY.md` — active stack and legacy freeze policy.
- `docs/FRONTEND_PAGES_AND_ROUTING.md` — Next App Router page map.
- `docs/FUNCTION_INVENTORY.md` — complete feature inventory.
- `docs/features/` — feature specs and template.

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
