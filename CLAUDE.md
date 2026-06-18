# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

凌镜 LingMirror (tech name: MultiSell) — Cross-border e-commerce AI Agent operations platform.
**Stack**: Python 3.11+ / FastAPI / SQLAlchemy 2.0 async / PostgreSQL 15 | Vue 3 / TypeScript / Naive UI / Pinia / Vite

See `AGENTS.md` for project governance, agent workflow rules, and completion standards.

## Commands

```bash
# Docker
docker compose up -d                          # Full stack (DB + backend :8000 + frontend :3000)
docker compose up -d db                       # DB only for local dev

# Backend
cd backend && uvicorn app.main:app --reload --port 8001   # Dev server
cd backend && PYTHONPATH="$PWD" .venv/bin/py.test -q                        # All tests
cd backend && PYTHONPATH="$PWD" .venv/bin/py.test tests/test_foo.py -q      # Single test file
cd backend && PYTHONPATH="$PWD" .venv/bin/py.test tests/test_foo.py::test_bar -q  # Single test
cd backend && .venv/bin/alembic upgrade head                         # Apply migrations
cd backend && .venv/bin/alembic revision --autogenerate -m "desc"   # New migration
cd backend && PYTHONPATH="$PWD" python seed.py                               # Seed demo data
cd backend && PYTHONPATH="$PWD" python seed_agent_demo.py                     # Seed agent demo
cd backend && python scripts/db_reset.py                  # Reset DB

# Frontend
cd frontend && npm run dev      # Dev server (:3001, proxies /api → :8001)
cd frontend && npm run build    # Production build
cd frontend && npm run preview  # Preview production build
```

Tests use a separate database `product_management_test` (auto-created via `scripts/init-db.sql`). Set `TEST_DATABASE_URL` env var; tests disable auth by default (`AUTH_ENABLED=False`). `asyncio_mode = auto` in pytest config. Fixtures in `tests/conftest.py` provide `prepare_db` (session-level schema create) and `async_client` (per-test ASGI transport via httpx).

## Architecture

### Backend module pattern

Every module in `backend/app/<module>/` follows a strict 4-file pattern:
- `__init__.py` — re-exports `router` (the module's APIRouter)
- `router.py` — FastAPI routes, uses `Depends(get_db)` + `require_permission("module:action")`
- `service.py` — static-method Service class (business logic, SQLAlchemy async queries)
- `schemas.py` — Pydantic request/response models

**Routers are auto-discovered**: `main.py` scans `app.*` subpackages and includes any with a `router` attribute under `/api`. Adding a new module requires only these 4 files — no manual registration.

**Config**: `backend/app/config.py` uses `pydantic-settings` `BaseSettings`. Reads from environment / `.env` automatically.

**Exceptions**: `backend/app/exceptions/` for custom exception handlers. Global `HTTPException` handler in `main.py` returns `{"code", "message", "data"}`.

**Listing adapters**: `backend/app/listing/adapters/` implements a registry pattern for platform-specific publishing:
- `base.py` — `ListingAdapter` abstract base
- `adapters/__init__.py` — `_ADAPTER_REGISTRY` dict + `get_listing_adapter(platform_code)` factory
- Per-platform files: `ozon.py`, `shopee.py`, `wildberries.py` + fallback `mock.py`
- To add a new platform: create the adapter class, register it in `_ADAPTER_REGISTRY`

**Common utilities**: `backend/app/common/` exports:
- `schemas.py` — `Result`, `PageResult`, `PageParam`, `ProductStatus`, `IdSchema`, `IdsSchema`, `StatusSchema`
- `upload.py` — File upload router
- `utils.py` — `build_tree()`, `save_upload_file()`, `allowed_file()`, `generate_filename()`, `utc_now()`

### Frontend auto-merge patterns

**API clients** in `frontend/src/api/modules/*.ts` are auto-merged via `import.meta.glob` in `frontend/src/api/index.ts`. New files in `modules/` are picked up automatically. Each file exports API objects directly (e.g. `export const allocationApi = { ... }`). Access via `import { apiModules } from '@/api'` and `apiModules.allocationApi.list()`.

**Routes** in `frontend/src/router/modules/*.ts` are auto-merged into the main router at `frontend/src/router/index.ts`. Each file exports `export const routes: RouteRecordRaw[]` with children under `path: 'agents'` etc. The route meta supports: `title`, `icon`, `menu` (show in nav), `perm` (permission code), `noAuth` (skip auth).

**Entry point**: `frontend/src/main.ts` — creates Pinia store + router, mounts App.
**No separate stores/composables/utils directories yet** — state management and helpers live inside view files or as needed.

### Agent system (Hermes architecture)

The agent framework (`backend/app/agent/`) implements a multi-agent system with lifecycle evolution stages:

**Evolution stages** (in `base.py`): `OBSERVATION` → `SUGGESTION` → `SEMI_AUTONOMOUS` → `FULL_AUTONOMOUS`. Each stage has a confidence threshold (0.0 / 0.85 / 0.90 / 0.95); an agent only acts when confidence exceeds its current stage threshold.

**10 agents** across 3 classes, registered via `@register_agent` decorator on `AgentRegistry`:
- **Governors (G1-G3)**: G1 Dashboard overview, G2 Warehouse/customs advice, G3 Discount risk guard
- **Analysts (A5-A7)**: A5 Inventory alerts, A6 Profit watch, A7 Compliance guard
- **Specialists (A1-A4)**: A1 Product scout, A2 Listing optimizer, A3 Ad advice, A4 Customer service

Each agent defines `agent_id`, `decision_points` (named capabilities), and `DEFAULT_STAGES` (initial evolution stage per decision point). Agents implement `async decide(decision_point, context, db) → dict`.

**Agent sub-modules** (all under `backend/app/agent/`):
| Path | Role |
|------|------|
| `base.py` | `BaseAgent`, `EvolutionStage` enum |
| `registry.py` | `AgentRegistry` (class-level `_agents` dict), `@register_agent` |
| `service.py` | `AgentService` — orchestrates agent invocations, records decisions |
| `action_service.py` | `AgentActionService` — extracts actions from agent outputs, supports execute/reject lifecycle |
| `llm_service.py` | `AgentLlmService` — LLM integration for agent natural-language explanations |
| `config_service.py` | Per-agent/user configuration persistence |
| `config_router.py` | REST endpoints for agent config CRUD |
| `data_service.py` | Data provisioning for agent contexts |
| `entropy/` | Self-cleaning rule management (see below) |

### Entropy / self-cleaning system

`backend/app/agent/entropy/` manages rule entropy — preventing rule bloat and staleness:

- **TTL Sweeper** — expires stale rules past TTL
- **Budget Enforcer** — caps rule count per agent/user
- **Decay Scheduler** — reduces unused rule weights over time
- **Merge Detector** — finds duplicate/similar rules (candidates only, no auto-merge)
- **Regret Analyzer** — tracks user overrides to improve rules
- **SPC Controller** — statistical process control limits on agent behavior
- **Rule Health Scorer** — scores each rule (active / shadow / warning / unhealthy)

`EntropyService.run_defenses()` runs all defenses and returns affected rules. The entropy index is a composite of unhealthy ratio × 0.4 + shadow ratio × 0.3 + (1 - avg_score) × 0.3.

### Decision pipeline

`backend/app/decision/` implements pre-listing profitability decisions. `PreListingDecisionService.calculate()` chains: SKU lookup → shipping fee calculation (`shipping` module) → platform fee calculation (`platform_fee` module) → payment/other fees → profit margin → recommendation (approve/reject/needs_data). The compare endpoint runs the same logic across multiple platforms.

### Full module list (31 modules)

| Module | Role |
|--------|------|
| `core` `category` `brand` `sku` `price` `inventory` `supplier` | Product CRUD, category tree, SKU cartesian gen |
| `platform` `listing` `listing_task` | Platform API keys, publish adapter registry, multi-platform task queue |
| `agent` `agent_actions` | Hermes multi-agent system + action audit lifecycle |
| `auth` `rbac` | JWT auth + RBAC permission system |
| `operation_log` | Mutation audit trail |
| `dashboard` `search` | Overview stats, global search |
| `common` | Shared Result/PageResult schemas, upload, utilities |
| `shipping` `platform_fee` `finance` `order` `settlement` | Order-to-settlement pipeline (quotes, fees, P&L) |
| `decision` `allocation` | Pre-listing profitability analysis, inventory allocation |
| `import_batch` `order_import` | Batch import jobs, order ingestion (CSV/JSON) |
| `image_gen` | AI product image generation |
| `notification` | System notifications and alerts |
| `exceptions` | Exception workbench |
| `platform_integrations` | Platform connection management |
| `activity` | Cross-module activity feed |

### Database

All models in single `backend/app/models.py` (1146 lines). Key entities: Product, Sku, Category, Brand, Price, Inventory, Supplier, Platform, ProductListing, ListingTask, Order, Warehouse, AllocationRule, ShippingQuoteRule, ImportBatch, PlatformFeeRule, Settlement, plus agent tables (AgentAction, AgentDecision, AgentEpisode, PersonalRule, SpcControlLimit, etc.) and new modules (ProductImageGen, PromptTemplate, Notification, AlertRule, FinanceAccount, etc.).

### Quickstart: creating a new module

**Backend**: Create `backend/app/<module>/` with 4 files → `__init__.py` exports `router`, `router.py` defines endpoints, `service.py` has business logic, `schemas.py` has Pydantic models. That's all — `main.py` auto-discovers the router.

**Frontend**: Optionally create `frontend/src/api/modules/<module>.ts` for API client (auto-merged) and `frontend/src/router/modules/<module>.ts` for routes (auto-merged, children of the Layout route). View goes in `frontend/src/views/<module>/`.

### Key conventions

- **Responses**: Always use `Result.ok()` / `Result.error()` / `PageResult.ok()` from `app.common.schemas` — never raw dicts
- **DB session**: `Depends(get_db)` — auto-commits on success, rolls back on exception
- **Auth**: `require_permission("module:action")` dependency on every endpoint. Admin role bypasses permission checks. `AUTH_ENABLED=False` in tests returns mock admin user
- **Audit log**: Mutation endpoints call `OperationLogService.log()` after data changes (see `core/router.py` for example pattern)
- **Async everywhere**: FastAPI + SQLAlchemy async engine + asyncpg driver. No sync DB calls
- **Migrations**: Alembic with autogenerate. Run `alembic upgrade head` after pulling
- **Frontend**: TypeScript strict mode. Naive UI components. API clients via `@/api` central export or `apiModules` from auto-merged `modules/*.ts`
- **Agent tests**: Phase-based test files (`test_agent_phase1.py` through `test_agent_phase6.py`) — phase 1=base, 2=G3/A5/A6, 5=entropy, 6=A1-A4/A7/G2
- **Settings**: pydantic-settings in `backend/app/config.py` reads from env vars with defaults. Key overrides: `DATABASE_URL`, `AUTH_ENABLED`, `DEBUG`, `ENCRYPTION_KEY`. Version `2.0.0`.

## Documentation

- [`docs/INDEX.md`](docs/INDEX.md) — Master document index (54 docs, categorized)
- [`docs/TIMELINE.md`](docs/TIMELINE.md) — Development timeline + P0/P1/P2 task board
- [`docs/features/`](docs/features/) — Feature requests (use [`TEMPLATE.md`](docs/features/TEMPLATE.md))
