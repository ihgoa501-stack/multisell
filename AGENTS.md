# 凌镜 LingMirror — Agent Instructions

<!-- CODEGRAPH_START -->
## CodeGraph

This repository is indexed by CodeGraph (`.codegraph/` exists at the repo root). Use it before grep/find or opening source files when you need to understand or locate code:

- MCP tools: `codegraph_explore` answers most code questions in one call with relevant symbols, verbatim source, and call paths. `codegraph_node` reads one symbol or a whole file with line numbers.
- Shell fallback: `codegraph explore "<question or symbols>"` and `codegraph node <symbol-or-file>`.
- Skip CodeGraph only for files it does not index well, such as Markdown, JSON, TOML, YAML, lockfiles, and generated artifacts.
<!-- CODEGRAPH_END -->

## Project

凌镜 LingMirror (technical name: MultiSell) is a cross-border e-commerce AI AgentOS.

Active stack (full-site migration complete):

- Backend: Go / Gin / GORM / PostgreSQL 15 under `backend-go/`
- Frontend: Next.js / React / TypeScript / Ant Design under `frontend-next/`
- Backend entry: `backend-go/cmd/server/main.go`
- Frontend entry: `frontend-next/src/app/`
- API prefix: `/api/v1`

Legacy stack is paused:

- `backend/` (Python / FastAPI) and `frontend/` (Vue 3) are reference-only.
- Do not add new product features to legacy directories.
- Legacy code may be touched only for migration, parity analysis, security rollback fixes, or documentation.
- See `docs/ACTIVE_STACK_POLICY.md`.

## Commands

| Action | Command |
|---|---|
| Docker full stack | `docker compose up -d` |
| Docker DB only | `docker compose up -d db` |
| Backend dev | `cd backend-go && go run cmd/server/main.go` |
| Backend test all | `cd backend-go && go test ./...` |
| Backend vet | `cd backend-go && go vet ./...` |
| Backend build | `cd backend-go && go build -o bin/server cmd/server/main.go` |
| Frontend dev | `cd frontend-next && npm run dev -- --hostname 127.0.0.1 --port 3000` |
| Frontend build | `cd frontend-next && npm run build` |
| Frontend lint | `cd frontend-next && npm run lint` |
| Legacy full stack | `docker compose -f docker-compose.legacy.yml up -d` |

New dev database: `multisell`.

## Backend

Active backend modules live under `backend-go/internal/`.

General pattern:

- `routes.go` registers Gin routes.
- `handler.go` handles HTTP request/response.
- `service.go` owns business logic.
- `model.go` owns GORM models and input/output structs.

`backend-go/internal/httpx/router.go` registers API routes under `/api/v1`.

Key modules:

| Module | Role |
|---|---|
| `category` `brand` `sku` `price` `inventory` `supplier` | Product catalog, categories, SKU generation, pricing, stock, suppliers |
| `platform` `listing` `listingtask` | Platform config, adapter-based publishing, multi-platform task queue |
| `ai` `agent` `agentos` | AI command center, trace/evidence/action lifecycle, AgentOS cockpit |
| `auth` `rbac` | JWT auth and permission checks |
| `operationlog` | Mutation audit logs |
| `dashboard` `search` | Overview stats and global search |
| `shipping` `platformfee` `finance` `order` `settlement` | Order-to-settlement, fees, ledger, profit |
| `decision` `allocation` | Pre-listing profitability and inventory allocation |
| `importbatch` `orderimport` | Batch imports and order ingestion |
| `imagegen` | Product image generation |
| `notification` | Notifications and alert rules |
| `exceptions` | Exception workbench |
| `integrations` | Platform connection management |

Legacy Python models under `backend/` are reference-only. Active GORM models live in each `backend-go/internal/**/model.go` file.

## Frontend

- App Router pages: `frontend-next/src/app/`.
- Shared components: `frontend-next/src/components/`.
- API client: `frontend-next/src/lib/api-client.ts`.
- Alias: `@` points to `frontend-next/src`.
- TypeScript strict mode is enabled.

Prefer existing Ant Design and React Query patterns in `frontend-next/`.

## Conventions

- Auth/RBAC should be enforced on non-public endpoints.
- Mutation routes should be auditable through the Go audit middleware or explicit operation log writes.
- Migrations live in `backend-go/migrations/`.
- Tests use Go `testing`; add focused tests for touched behavior.
- Frontend changes should keep `npm run build` and `npm run lint` green.
- Do not touch `.kilo/worktrees/`; it is managed by external tooling.

## Architecture Notes

- AI runtime lives in `backend-go/internal/ai/`, `backend-go/internal/agent/`, and `backend-go/internal/agentos/`.
- Realtime WebSocket route is `/ws`.
- Decision pipeline: SKU lookup -> shipping fee -> platform fee -> payment/other fees -> profit margin -> recommendation.
- Config uses `backend-go/configs/config.yaml` plus env overrides. Important env vars: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `JWT_SECRET`, `SERVER_PORT`, `LLM_PROVIDER`, `LLM_API_KEY`, `LLM_MODEL`.

## Documentation

- `docs/INDEX.md` — documentation index
- `docs/TIMELINE.md` — timeline and task board
- `docs/ACTIVE_STACK_POLICY.md` — active stack and legacy freeze policy
- `docs/features/` — feature specs; use `docs/features/TEMPLATE.md` for new feature docs
- `CLAUDE.md` — Claude Code-specific guidance; keep it consistent with this file
