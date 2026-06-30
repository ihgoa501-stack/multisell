# 凌镜 LingMirror — Agent Instructions

<!-- CODEGRAPH_START -->
## CodeGraph

This repository is indexed by CodeGraph (`.codegraph/` exists at the repo root). Use it before grep/find or opening source files when you need to understand or locate code:

- MCP tools: `codegraph_explore` answers most code questions in one call with relevant symbols, verbatim source, and call paths. `codegraph_node` reads one symbol or a whole file with line numbers.
- Shell fallback: `codegraph explore "<question or symbols>"` and `codegraph node <symbol-or-file>`.
- Skip CodeGraph only for files it does not index well, such as Markdown, JSON, TOML, YAML, lockfiles, and generated artifacts.
<!-- CODEGRAPH_END -->

## Project

凌镜 LingMirror (technical name: MultiSell) — cross-border e-commerce AI AgentOS.
Version `v0.2.1`.

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
- `docs/INDEX.md` — full doc index.
- `docs/PROJECT_STATUS.md` — current new-stack status.
- `docs/ACTIVE_STACK_POLICY.md` — active/legacy policy.
- `docs/FRONTEND_PAGES_AND_ROUTING.md` — page map.
- `docs/features/` — feature specs; use `TEMPLATE.md`.
