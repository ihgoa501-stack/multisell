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

- Backend: Python 3.11+ / FastAPI / SQLAlchemy 2.0 async / PostgreSQL 15
- Frontend: Vue 3 / TypeScript / Naive UI / Pinia / Vite
- Backend entry: `backend/app/main.py` auto-discovers routers under `/api`
- Frontend entry: `frontend/src/main.ts`
- Python environment: `backend/.venv/` (Python 3.12, dependencies managed with `uv`/venv)

## Commands

| Action | Command |
|---|---|
| Docker full stack | `docker compose up -d` |
| Docker DB only | `docker compose up -d db` |
| Backend dev | `cd backend && .venv/bin/uvicorn app.main:app --reload --host 127.0.0.1 --port 8001` |
| Backend test all | `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest -q` |
| Backend test file | `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_foo.py -q` |
| Backend test single | `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_foo.py::test_bar -q` |
| Alembic migrate | `cd backend && .venv/bin/alembic upgrade heads` |
| Alembic current | `cd backend && .venv/bin/alembic current --verbose` |
| Alembic new migration | `cd backend && .venv/bin/alembic revision --autogenerate -m "desc"` |
| Seed demo data | `cd backend && PYTHONPATH="$PWD" .venv/bin/python seed.py` |
| Seed agent demo | `cd backend && PYTHONPATH="$PWD" .venv/bin/python seed_agent_demo.py` |
| Reset DB | `cd backend && .venv/bin/python scripts/db_reset.py --force` |
| Frontend dev | `cd frontend && npm run dev -- --host 127.0.0.1 --port 3001` |
| Frontend build | `cd frontend && npm run build` |
| Frontend preview | `cd frontend && npm run preview` |

Test database: `product_management_test`. Set `TEST_DATABASE_URL` when needed. Tests disable auth by default via `AUTH_ENABLED=False` in `backend/tests/conftest.py`.

## Backend

Module pattern: every module under `backend/app/<module>/` should use:

- `__init__.py` exporting `router`
- `router.py` for FastAPI endpoints
- `service.py` for business logic, usually static async service methods
- `schemas.py` for Pydantic models

`backend/app/main.py` auto-discovers subpackages with a `router` attribute and mounts them under `/api`.

Key modules:

| Module | Role |
|---|---|
| `core` `category` `brand` `sku` `price` `inventory` `supplier` | Product catalog, categories, SKU generation, pricing, stock, suppliers |
| `platform` `listing` `listing_task` | Platform config, adapter-based publishing, multi-platform task queue |
| `agent` `agent_actions` | Hermes AI agents, evolution stages, action audit lifecycle |
| `agentos` | AgentOS 运营总控台 — 聚合层、WorkItem 归一化、Squad/Agent 视图 |
| `auth` `rbac` | JWT auth and permission checks |
| `operation_log` | Mutation audit logs |
| `dashboard` `search` | Overview stats and global search |
| `shipping` `platform_fee` `finance` `order` `settlement` | Order-to-settlement, fees, ledger, profit |
| `decision` `allocation` | Pre-listing profitability and inventory allocation |
| `import_batch` `order_import` | Batch imports and order ingestion |
| `image_gen` | Product image generation |
| `notification` | Notifications and alert rules |
| `exceptions` | Exception workbench |
| `platform_integrations` | Platform connection management |

All primary SQLAlchemy models live in `backend/app/models.py`; agent-specific models also exist under `backend/app/agent/models.py` and import-related models under `backend/app/order_import/models.py`.

## Frontend

- API modules: `frontend/src/api/modules/*.ts`, auto-merged by `frontend/src/api/index.ts`.
- Routes: `frontend/src/router/modules/*.ts`, auto-merged as layout children.
- Views: `frontend/src/views/`.
- Components: `frontend/src/components/`.
- Alias: `@` points to `frontend/src`.
- TypeScript strict mode is enabled.

Prefer existing Naive UI patterns and existing module API style. If a component imports named functions from an API module, export named functions from that module as well as any object-style API used elsewhere.

## Conventions

- Responses: use `Result.ok()`, `Result.error()`, `Result.bad_request()`, `Result.not_found()`, or `PageResult.ok()` from `app.common.schemas`; do not return raw dicts from routers.
- DB session: use `Depends(get_db)`. The dependency commits on success and rolls back on exceptions.
- Auth: every endpoint should use `require_permission("module:action")` unless it is intentionally public. Admin bypasses checks. `AUTH_ENABLED=False` returns a mock admin.
- Audit log: mutation endpoints should call `OperationLogService.log(...)` after the data change.
- Async backend: use async SQLAlchemy and asyncpg. Do not introduce sync DB calls in request paths.
- Migrations: this repo currently has multiple Alembic heads. Use `alembic upgrade heads`, not `upgrade head`.
- Tests: pytest with `asyncio_mode = auto`; use focused tests for touched behavior.
- Agent tests: phase files `test_agent_phase1.py` through `test_agent_phase6.py`.
- Do not touch `.kilo/worktrees/`; it is managed by external tooling.

## Architecture Notes

- Listing adapters live in `backend/app/listing/adapters/`; `_ADAPTER_REGISTRY` maps platform codes to adapter classes (`ozon`, `shopee`, `wildberries`, fallback `mock`).
- Agent system (`backend/app/agent/`) has 4 stages: `OBSERVATION` -> `SUGGESTION` -> `SEMI_AUTONOMOUS` -> `FULL_AUTONOMOUS`.
- Agents are registered with `@register_agent`; 10 agents cover governors G1-G3, analysts A5-A7, and specialists A1-A4.
- Entropy defenses live in `backend/app/agent/entropy/`; `EntropyService.run_defenses()` runs TTL, budget, decay, merge detection, regret, SPC, and health scoring.
- Decision pipeline: SKU lookup -> shipping fee -> platform fee -> payment/other fees -> profit margin -> recommendation.
- Config uses `backend/app/config.py` and pydantic-settings. Important env vars: `DATABASE_URL`, `DATABASE_URL_SYNC`, `AUTH_ENABLED`, `DEBUG`, `ENCRYPTION_KEY`, `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`.

## Documentation

- `docs/INDEX.md` — documentation index
- `docs/TIMELINE.md` — timeline and task board
- `docs/features/` — feature specs; use `docs/features/TEMPLATE.md` for new feature docs
- `CLAUDE.md` — Claude Code-specific guidance; keep it consistent with this file
