# CLAUDE.md

This file gives Claude Code-specific guidance for working in this repository.
Canonical cross-agent rules are in `AGENTS.md`; keep this file consistent with it.

## Purpose

Use this file as a concise working brief for Claude Code. It should contain durable coding rules, project conventions, and navigation pointers.

Product direction, business priorities, project status, and known issues are intentionally not defined here. Before making product recommendations or prioritization calls, read the current source documents listed under Documentation Map.

## Project

LingMirror, technical name MultiSell, is a proprietary cross-border e-commerce AI AgentOS.

Do not add open-source license language, publish code, or distribute project code unless the Owner explicitly requests it. See `LICENSE`.

| Stack | Directory | Entry |
| --- | --- | --- |
| Backend | `backend-go/` | `cmd/server/main.go` |
| Frontend | `frontend-next/` | `src/app/` |

API prefix: `/api/v1`. Health endpoint: `/api/health`. All non-auth business APIs require JWT.

## Required Reading

Before non-trivial development, refactor, review, QA, release work, or product advice, read:

- `AGENTS.md`
- `docs/governance/OWNER_FIRST_PROTOCOL.md`
- `docs/governance/PLATFORM_CONSTITUTION.md`
- `docs/governance/AGENT_DEVELOPMENT_PROTOCOL.md`
- `docs/governance/KERNEL_CONTRACTS.md`
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md`
- `docs/PROJECT_STATUS.md`
- `docs/reference-module-catalog.md`

When documents conflict, follow `docs/governance/*` unless the Owner explicitly overrides them.

## Commands

```bash
# Infrastructure
docker compose up -d
docker compose up -d db

# Backend
cd backend-go
go run cmd/server/main.go
go test ./...
go vet ./...
go build -o bin/server cmd/server/main.go

# Frontend
cd frontend-next
npm run dev -- --hostname 127.0.0.1 --port 3000
npm run build
npm run lint
npm test

# E2E
cd frontend-next/e2e && npx playwright test
```

New dev database: `multisell`. Migrations live under `backend-go/migrations/`.

## Start Rules

1. Check `git status --short` before edits.
2. Preserve unrelated dirty work.
3. `.codegraph/` exists; use CodeGraph before grep/find/source reads when locating or understanding code.
4. Read a file before editing it.
5. Keep changes scoped to the requested outcome.
6. Do not touch `.kilo/worktrees/`.
7. Do not run destructive git or database commands unless the Owner explicitly requests them.

## Architecture Boundaries

Every non-trivial change should identify the touched layer:

1. Platform Kernel: auth, RBAC, approval, audit, EventBus, Command, Scheduler, ToolBridge, Agent execution, config, observability, migrations.
2. Domain Modules: business capabilities under `backend-go/internal/domain/*`.
3. Agent Workflows: decision and automation flows built on kernel and domain modules.
4. Integrations: external platforms, tools, logistics providers, AI providers, credentials, platform writes.
5. UI / Experience: Next.js pages, components, layouts, Owner-facing workflows.
6. Documentation / Governance.

Business logic belongs in domain modules, not in Platform Kernel. Kernel provides mechanisms, not business-specific decisions.

## High-Risk Areas

Treat these as high risk unless the Owner and governance docs say otherwise:

- Prices, discounts, fees, profit, settlement, or money-impacting logic.
- Inventory changes, reservations, allocations, or stock sync.
- Order state, refunds, fulfillment, returns, or logistics state.
- External platform publishing, write-back, credential use, or account permissions.
- Auth, RBAC, approval, audit, Agent autonomy, EventBus, Command, Scheduler, ToolBridge.
- Database migrations, destructive operations, or data deletion.

High-risk production actions require approval, audit, and clear business impact. Autonomous execution is forbidden for critical business mutations unless a written policy explicitly allows it.

## Backend Pattern

Domain modules usually follow:

| File | Purpose |
| --- | --- |
| `routes.go` | Gin route registration |
| `handler.go` | HTTP request/response mapping |
| `service.go` | business logic |
| `model.go` | GORM models and request/response structs |

Standard response envelope:

```go
response.Success(c, data)
response.Error(c, http.StatusBadRequest, msg)
response.Paginated(c, data, total, page, size)
response.InternalError(c, err)
```

Use `common.ParsePagination(c)` and `common.ParseSort(c)` where applicable.

Test helper:

```go
import "github.com/lingmirror/backend-go/internal/dbtest"

func TestX(t *testing.T) {
    db := dbtest.NewDB(t, &MyModel{})
    svc := NewService(db, logger)
}
```

## Frontend Pattern

- Pages live under `frontend-next/src/app/(main)/{module}/page.tsx`.
- Reused UI belongs under `frontend-next/src/components/`.
- Top-level menu entries live in `frontend-next/src/config/menu.ts`.
- API calls go through `apiClient` with `/v1/*` paths.
- Use React Query for server state and Zustand for global client state.
- Alias `@` maps to `frontend-next/src/`.
- Match existing Ant Design and project component patterns.

## Verification

Run the smallest checks that cover the touched surface:

- Backend domain change: package tests, then broader backend tests when shared behavior is affected.
- Platform Kernel or high-risk workflow: focused tests plus `cd backend-go && go test ./...` when feasible.
- Frontend behavior: relevant unit tests, build/lint when feasible, and browser/E2E checks for critical flows.
- Docs-only change: verify links and internal consistency.

If a relevant check cannot be run, state why in the final report.

## Documentation Map

- `AGENTS.md` — canonical cross-agent project instructions.
- `docs/governance/` — mandatory Owner-first and platform-first rules.
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md` — current product direction and priority guidance.
- `docs/PROJECT_STATUS.md` — current implementation and verification status.
- `docs/reference-module-catalog.md` — canonical module, route, and page catalog.
- `docs/INDEX.md` — documentation index.
- `docs/ACTIVE_STACK_POLICY.md` — active stack and legacy policy.
- `docs/features/` — feature specs; use `TEMPLATE.md`.
