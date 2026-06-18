# CLAUDE.md

This file gives Claude Code-specific guidance for working in this repository. The canonical cross-agent rules are in `AGENTS.md`; keep this file consistent with it.

## Project

凌镜 LingMirror (technical name: MultiSell) is a cross-border e-commerce AI AgentOS.

Stack:
- Backend: Python 3.11+ / FastAPI / SQLAlchemy 2.0 async / PostgreSQL 15
- Frontend: Vue 3 / TypeScript / Naive UI / Pinia / Vite
- Backend entry: `backend/app/main.py`
- Frontend entry: `frontend/src/main.ts`

## First Steps

1. This repo has `.codegraph/`; use CodeGraph before grep/find/opening source files when locating or understanding code.
2. Check `git status --short` before edits. Preserve user changes and unrelated dirty files.
3. Prefer the existing module patterns over new abstractions.
4. Keep backend and frontend API contracts in sync.
5. Run the smallest meaningful verification after changes, then broaden if the touched surface is shared.

## Commands

```bash
# Docker
docker compose up -d
docker compose up -d db

# Backend
cd backend && .venv/bin/uvicorn app.main:app --reload --host 127.0.0.1 --port 8001
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest -q
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_foo.py -q
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_foo.py::test_bar -q
cd backend && .venv/bin/alembic upgrade heads
cd backend && .venv/bin/alembic current --verbose
cd backend && .venv/bin/alembic revision --autogenerate -m "desc"
cd backend && PYTHONPATH="$PWD" .venv/bin/python seed.py
cd backend && PYTHONPATH="$PWD" .venv/bin/python seed_agent_demo.py

# Frontend
cd frontend && npm run dev -- --host 127.0.0.1 --port 3001
cd frontend && npm run build
cd frontend && npm run preview
```

Tests use `product_management_test`. `backend/tests/conftest.py` sets `AUTH_ENABLED=False` and creates schema via fixtures. Use `TEST_DATABASE_URL` to override.

## Backend Architecture

Every backend module under `backend/app/<module>/` should follow this shape:

- `__init__.py` exports `router`
- `router.py` defines FastAPI endpoints, `Depends(get_db)`, and permission dependencies
- `service.py` contains business logic and async SQLAlchemy queries
- `schemas.py` contains Pydantic request/response models

`backend/app/main.py` auto-discovers routers from `app.*` subpackages and mounts them under `/api`.

Common conventions:
- Router responses use `Result` / `PageResult`; avoid raw dict returns.
- Endpoints use `require_permission("module:action")` unless intentionally public.
- Mutation endpoints log with `OperationLogService.log(...)`.
- Async DB only in request paths.
- `get_db` commits on success and rolls back on exception.
- Alembic currently has multiple heads; use `upgrade heads`.

Important backend areas:
- `backend/app/models.py` — primary SQLAlchemy models.
- `backend/app/common/` — `Result`, `PageResult`, upload, shared helpers.
- `backend/app/auth/` and `backend/app/rbac/` — auth and permissions.
- `backend/app/listing/adapters/` — platform adapter registry.
- `backend/app/agent/` — Hermes agent framework.
- `backend/app/agent/entropy/` — rule entropy defenses.
- `backend/app/decision/` — pre-listing profitability.
- `backend/app/finance/` — finance accounts, reports, ledger.
- `backend/app/order_import/` — order ingestion chain.

## Frontend Architecture

Frontend patterns:
- `frontend/src/api/modules/*.ts` are auto-merged by `frontend/src/api/index.ts`.
- `frontend/src/router/modules/*.ts` are auto-merged by `frontend/src/router/index.ts`.
- Views live in `frontend/src/views/`.
- Shared components live in `frontend/src/components/`.
- Alias `@` maps to `frontend/src`.

When adding a page:
1. Add or update an API module under `src/api/modules`.
2. Add route metadata under `src/router/modules`.
3. Add the view under `src/views`.
4. Use Naive UI components and existing layout patterns.
5. Run `npm run build`.

## Agent System

Hermes agents live under `backend/app/agent/`.

Stages:
- `OBSERVATION`
- `SUGGESTION`
- `SEMI_AUTONOMOUS`
- `FULL_AUTONOMOUS`

Agents are registered with `@register_agent`.

Agent groups:
- Governors: G1 dashboard, G2 warehouse/customs, G3 discount risk
- Analysts: A5 inventory, A6 profit, A7 compliance
- Specialists: A1 product scout, A2 listing optimizer, A3 ads, A4 customer service

Agent decisions are recorded by `AgentService`. Action extraction/execution goes through `AgentActionService`. Entropy controls live in `backend/app/agent/entropy/`.

## Database And Initialization

Use `.env` / environment variables for:
- `DATABASE_URL`
- `DATABASE_URL_SYNC`
- `TEST_DATABASE_URL`
- `AUTH_ENABLED`
- `ENCRYPTION_KEY`
- `LLM_API_URL`
- `LLM_API_KEY`
- `LLM_MODEL`

For local development, prefer explicit IPv4 host `127.0.0.1` when a local PostgreSQL and Docker PostgreSQL may both exist.

Do not reset or drop an existing database unless the user explicitly approves the destructive action. For safe initialization, create a separate development database and seed it.

## Verification

Use focused checks:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_health.py -q
cd frontend && npm run build
```

For backend changes, add or run tests near the touched module. For frontend API/view changes, run the production build at minimum.

## Documentation

- `AGENTS.md` — canonical cross-agent project instructions
- `docs/INDEX.md` — documentation index
- `docs/TIMELINE.md` — timeline and task board
- `docs/features/` — feature specs and template

Do not touch `.kilo/worktrees/`; it is managed by external tooling.
