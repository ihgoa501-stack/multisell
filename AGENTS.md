# 凌镜 LingMirror — 跨境电商 AgentOS

技术名 MultiSell；Python FastAPI + Vue 3 + PostgreSQL 的 AI Agent 协作跨境电商运营平台。

## Project

- **Stack**: Python 3.11+ / FastAPI / SQLAlchemy 2.0 async / PostgreSQL 15
- **Frontend**: Vue 3 / TypeScript / Naive UI / Pinia / Vite
- **Entry (backend)**: `backend/app/main.py` — auto-discovered routers under `/api`
- **Entry (frontend)**: `frontend/src/main.ts`

## Commands

| Action | Command |
|---|---|
| Docker up | `docker compose up -d` |
| Backend dev | `cd backend && uvicorn app.main:app --reload --port 8001` |
| Backend test all | `cd backend && python3 -m pytest -q` |
| Backend test file | `cd backend && python3 -m pytest tests/test_foo.py -q` |
| Backend test single | `cd backend && python3 -m pytest tests/test_foo.py::test_bar -q` |
| Alembic migrate | `cd backend && alembic upgrade head` |
| Alembic new migration | `cd backend && alembic revision --autogenerate -m "describe"` |
| Seed demo data | `cd backend && python seed.py` |
| Reset DB | `cd backend && python scripts/db_reset.py` |
| Frontend dev | `cd frontend && npm run dev` (:3001, proxies /api → :8001) |
| Frontend build | `cd frontend && npm run build` |
| Frontend preview | `cd frontend && npm run preview` |

**Test database**: `product_management_test` (auto-created via `backend/scripts/init-db.sql`).  
Set `TEST_DATABASE_URL` env var; tests disable auth by default (`AUTH_ENABLED=False` in `conftest.py`).

## Backend

**Module pattern**: Every module under `backend/app/<module>/` = `__init__.py` (exports `router`), `router.py`, `service.py` (static async methods), `schemas.py`. Some add extras (e.g. `auth/dependencies.py`). New module = create these files — `main.py` auto-discovers and mounts at `/api`.

**Key modules**:

| Module | Role |
|---|---|
| `core` `category` `brand` `sku` `price` `inventory` `supplier` | Product CRUD, category tree, SKU cartesian gen, pricing, stock, supplier binding |
| `platform` `listing` | Platform API key config, one-click publish (adapter registry) |
| `agent` | Hermes AI Agent framework: 10 agents, 4 evolution stages, entropy management |
| `auth` `rbac` | JWT auth + RBAC (`require_permission("module:action")`) |
| `operation_log` | Audit log for all mutation endpoints |
| `dashboard` `search` | Overview stats, global search (hotkey `/`) |
| `common` | Shared schemas (`Result`, `PageResult`, ...), upload, utilities |
| `shipping` `platform_fee` `finance` `order` `settlement` | Order-to-settlement pipeline (quotes, fees, P&L) |
| `decision` `allocation` | Pre-listing profitability analysis, inventory allocation rules |
| `import_batch` `order_import` | Batch import / order ingestion |

All models in a single file: `backend/app/models.py` (~930 lines).

## Frontend

- `src/api/index.ts` — legacy direct exports (`productApi`, etc.) + `import.meta.glob` merge of `src/api/modules/*.ts` → access via `apiModules.<name>.*`
- `src/router/index.ts` — `src/router/modules/*.ts` auto-merged as Layout children
- `src/views/` — page components
- `src/components/` — only `Layout.vue`
- Auto-imports: Vue + Naive UI via `unplugin-auto-import`
- Path alias: `@` → `src/`
- TypeScript strict mode

## Conventions

- **Responses**: `Result.ok()` / `Result.error()` / `PageResult.ok()` from `app.common.schemas`. Never raw dicts.
- **DB session**: `Depends(get_db)` — auto-commits on success, rolls back on exception.
- **Auth**: `require_permission("module:action")` on every endpoint. Admin role bypasses. `AUTH_ENABLED=False` returns mock admin.
- **Audit log**: Mutation endpoints call `OperationLogService.log(db, module, action, resource_id, ...)` after data change.
- **Async everywhere**: FastAPI + SQLAlchemy async engine + asyncpg. No sync DB calls.
- **Migrations**: Alembic with autogenerate. Run `alembic upgrade head` after pulling.
- **Tests**: `pytest` with `asyncio_mode = auto`. Fixtures: `prepare_db` (session schema + admin seed), `async_client` (per-test ASGI). Auth-enabled tests use `enable_auth` fixture from `tests/auth_helpers.py`.
- **Agent tests**: Phase-based (`test_agent_phase1.py`–`test_agent_phase6.py`).

## Architecture notes

- **Listing adapters** (`backend/app/listing/adapters/`): registry pattern — `_ADAPTER_REGISTRY` maps platform_code → adapter class (`ozon`, `shopee`, `wildberries` + fallback `mock`). New platform = add class + register.
- **Agent system** (Hermes, `backend/app/agent/`): 4 evolution stages (`OBSERVATION` → `SUGGESTION` → `SEMI_AUTONOMOUS` → `FULL_AUTONOMOUS`) with confidence thresholds. 10 agents (Governors G1-G3, Analysts A5-A7, Specialists A1-A4) via `@register_agent` decorator.
- **Entropy system** (`backend/app/agent/entropy/`): TTL sweeper, budget enforcer, decay scheduler, merge detector, regret analyzer, SPC controller, rule health scorer. `EntropyService.run_defenses()` runs all.
- **Decision pipeline** (`backend/app/decision/`): SKU lookup → shipping fee → platform fee → profit margin → recommendation.

For full architecture details, see `CLAUDE.md`.
