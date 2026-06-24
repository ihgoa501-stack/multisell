# CLAUDE.md

This file gives Claude Code-specific guidance for working in this repository. The canonical cross-agent rules are in `AGENTS.md`; keep this file consistent with it.

## Project

凌镜 LingMirror (technical name: MultiSell) is a cross-border e-commerce AI AgentOS.

Active stack:

- Backend: Go / Gin / GORM / PostgreSQL 15 under `backend-go/`
- Frontend: Next.js / React / TypeScript / Ant Design under `frontend-next/`
- Backend entry: `backend-go/cmd/server/main.go`
- Frontend entry: `frontend-next/src/app/`
- API prefix: `/api/v1`
- Health check: `/api/health`

Legacy stack is paused:

- `backend/` (Python / FastAPI) and `frontend/` (Vue 3) are reference-only.
- Do not add new product features to legacy directories.
- Legacy code may be touched only for migration, parity analysis, security rollback fixes, or documentation.
- See `docs/ACTIVE_STACK_POLICY.md`.

## First Steps

1. This repo has `.codegraph/`; use CodeGraph before grep/find/opening source files when locating or understanding indexed code.
2. Check `git status --short` before edits. Preserve user changes and unrelated dirty files.
3. Prefer existing Go domain and Next page/component patterns over new abstractions.
4. Keep frontend API calls aligned with Go routes under `/api/v1`.
5. Run the smallest meaningful verification after changes, then broaden if the touched surface is shared.

## Commands

```bash
# Docker
docker compose up -d
docker compose up -d db

# Backend
cd backend-go && go run cmd/server/main.go
cd backend-go && go test ./...
cd backend-go && go vet ./...
cd backend-go && go build -o bin/server cmd/server/main.go

# Frontend
cd frontend-next && npm run dev -- --hostname 127.0.0.1 --port 3000
cd frontend-next && npm run build
cd frontend-next && npm run lint
cd frontend-next && npm test

# Legacy reference only
docker compose -f docker-compose.legacy.yml up -d
```

New dev database: `multisell`.

## Backend Architecture

Active backend modules live under `backend-go/internal/`.

General module pattern:

- `routes.go` registers Gin routes.
- `handler.go` handles HTTP request/response mapping.
- `service.go` owns business logic.
- `model.go` owns GORM models and request/response structs.

`backend-go/internal/httpx/router.go` registers API routes under `/api/v1`.

Important areas:

- `backend-go/internal/auth` and `backend-go/internal/rbac` — JWT auth and permissions.
- `backend-go/internal/httpx/middleware` — CORS, auth, audit, metrics, recovery.
- `backend-go/internal/domain/*` — business modules.
- `backend-go/internal/ai`, `backend-go/internal/agent`, `backend-go/internal/agentos` — AI runtime and AgentOS.
- `backend-go/internal/platform/eventbus`, `command`, `scheduler` — event-driven agent infrastructure.
- `backend-go/internal/realtime` — WebSocket hub.
- `backend-go/migrations` — SQL migrations and migration runbook.

Common conventions:

- Non-public endpoints should be protected by JWT auth.
- Mutation routes should be auditable through the Go audit middleware or explicit operation log writes.
- Keep new API contracts under `/api/v1/*`.
- Use GORM transactions for multi-step state changes.
- Add focused Go tests near touched behavior.

## Frontend Architecture

Active frontend lives under `frontend-next/`.

- App Router pages: `frontend-next/src/app/`
- Shared components: `frontend-next/src/components/`
- API client: `frontend-next/src/lib/api-client.ts`
- Menu config: `frontend-next/src/config/menu.ts`
- Stores: `frontend-next/src/stores/`
- Types: `frontend-next/src/types/`
- Alias: `@` maps to `frontend-next/src`

When adding a page:

1. Add the route under `src/app/(main)/.../page.tsx` or `src/app/(auth)/...`.
2. Add shared UI under `src/components/` only when reused.
3. Add menu entries in `src/config/menu.ts` only for navigable top-level pages.
4. Use `apiClient` with `/v1/*` paths so default base `http://localhost:8080/api` becomes `/api/v1/*`.
5. Prefer Ant Design, React Query, and existing layout patterns.
6. Run `npm run build`; run `npm run lint` when touching TypeScript/React.

## Agent System

AI and AgentOS currently span:

- `backend-go/internal/ai` — chat, run, traces, actions, streaming.
- `backend-go/internal/agent` — Agent registry and execution entry points.
- `backend-go/internal/agentos` — cockpit, work items, autonomy overview.
- `backend-go/internal/domain/agentrule` — agent rules.
- `backend-go/internal/domain/entropy` — entropy monitoring/defense.
- `backend-go/internal/domain/evolution` — evolution nudges.
- `backend-go/internal/domain/trustscore` — trust score and autonomy gating.
- `backend-go/internal/domain/actionpolicy` — action approval policy.

The legacy Hermes Python implementation under `backend/app/agent/` is reference-only.

## Configuration

Backend config uses `backend-go/configs/config.yaml` plus environment overrides.

Important env vars:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `JWT_SECRET`
- `SERVER_PORT`
- `LLM_PROVIDER`
- `LLM_API_KEY`
- `LLM_MODEL`

Frontend API base:

- `NEXT_PUBLIC_API_URL=http://localhost:8080/api`

## Verification

Use focused checks:

```bash
cd backend-go && go test ./...
cd backend-go && go vet ./...
cd frontend-next && npm test
cd frontend-next && npm run build
cd frontend-next && npm run lint
```

Current known state from 2026-06-24:

- Backend tests pass.
- Backend vet passes.
- Frontend tests pass.
- Frontend build passes.
- Frontend lint currently fails and should be fixed before treating the frontend quality gate as green.

## Documentation

- `AGENTS.md` — canonical cross-agent project instructions.
- `docs/INDEX.md` — documentation index.
- `docs/PROJECT_STATUS.md` — current new-stack status.
- `docs/ACTIVE_STACK_POLICY.md` — active stack and legacy freeze policy.
- `docs/FRONTEND_PAGES_AND_ROUTING.md` — Next App Router page map.
- `docs/FUNCTION_INVENTORY.md` — current feature inventory.
- `docs/features/` — feature specs and template.

Do not touch `.kilo/worktrees/`; it is managed by external tooling.
