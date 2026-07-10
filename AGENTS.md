# LingMirror Agent Instructions

<!-- CODEGRAPH_START -->
## CodeGraph

This repository is indexed by CodeGraph (`.codegraph/` exists at the repo root). Use it before grep/find or opening source files when you need to understand or locate code:

- MCP tools: `codegraph_explore` answers most code questions in one call with relevant symbols, verbatim source, and call paths. `codegraph_node` reads one symbol or a whole file with line numbers.
- Shell fallback: `codegraph explore "<question or symbols>"` and `codegraph node <symbol-or-file>`.
- Skip CodeGraph for files it does not index well, such as Markdown, JSON, TOML, YAML, lockfiles, and generated artifacts.
<!-- CODEGRAPH_END -->

## Purpose

This file is the durable working brief for coding agents in this repository. Keep it focused on how to work safely in the codebase.

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

- `docs/governance/OWNER_FIRST_PROTOCOL.md`
- `docs/governance/PLATFORM_CONSTITUTION.md`
- `docs/governance/AGENT_DEVELOPMENT_PROTOCOL.md`
- `docs/governance/KERNEL_CONTRACTS.md`
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md`
- `docs/PROJECT_STATUS.md`
- `docs/reference-module-catalog.md`

When documents conflict, follow `docs/governance/*` unless the Owner explicitly overrides them.

## Commands

| Action | Command |
| --- | --- |
| Docker full stack | `docker compose up -d` |
| Docker DB only | `docker compose up -d db` |
| Backend dev | `cd backend-go && go run cmd/server/main.go` |
| Backend test all | `cd backend-go && go test ./...` |
| Backend vet | `cd backend-go && go vet ./...` |
| Backend build | `cd backend-go && go build -o bin/server cmd/server/main.go` |
| Frontend dev | `cd frontend-next && npm run dev -- --hostname 127.0.0.1 --port 3000` |
| Frontend build | `cd frontend-next && npm run build` |
| Frontend lint | `cd frontend-next && npm run lint` |
| Frontend test | `cd frontend-next && npm test` |
| E2E | `cd frontend-next/e2e && npx playwright test` |

New dev database: `multisell`. Migrations live under `backend-go/migrations/`.

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

## Working Rules

- Preserve unrelated dirty work. Do not revert or overwrite user or agent changes you did not make.
- Do not touch `.kilo/worktrees/`.
- Do not run destructive git or database commands unless the Owner explicitly requests them.
- Use existing module patterns before inventing new abstractions.
- Keep changes scoped to the requested outcome.
- Use GORM transactions for multi-step business state changes.
- Non-public endpoints must use JWT auth. Mutation APIs need server-side permission checks and audit where appropriate.
- Frontend API calls use `apiClient` with `/v1/*` paths.
- If module names, API paths, directory layouts, or package structure change, update `AGENTS.md`, `CLAUDE.md`, and `docs/INDEX.md` or the relevant catalog docs.

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
- Use React Query for server state and Zustand for global client state.
- Alias `@` maps to `frontend-next/src/`.
- Match the existing Ant Design and project component patterns.

## Verification

Run the smallest checks that cover the touched surface:

- Backend domain change: package tests, then broader backend tests when shared behavior is affected.
- Platform Kernel or high-risk workflow: focused tests plus `cd backend-go && go test ./...` when feasible.
- Frontend behavior: relevant unit tests, build/lint when feasible, and browser/E2E checks for critical flows.
- Docs-only change: verify links and internal consistency.

If a relevant check cannot be run, state why in the final report.

## Documentation Map

- `docs/governance/` — mandatory Owner-first and platform-first rules.
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md` — current product direction and priority guidance.
- `docs/PROJECT_STATUS.md` — current implementation and verification status.
- `docs/reference-module-catalog.md` — canonical module, route, and page catalog.
- `docs/INDEX.md` — documentation index.
- `docs/ACTIVE_STACK_POLICY.md` — active stack and legacy policy.
- `docs/CODEBASE_ANALYSIS.md` — codebase analysis snapshot, knowledge graph usage, and regeneration guidance.
- `docs/features/` — feature specs; use `TEMPLATE.md`.
- `CLAUDE.md` — Claude Code-specific companion guidance; keep it consistent with this file.
