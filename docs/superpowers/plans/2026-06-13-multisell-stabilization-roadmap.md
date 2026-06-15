# MultiSell Stabilization And Productization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:writing-plans to expand each subsystem below into its own task-by-task implementation plan before editing code. Use superpowers:subagent-driven-development or superpowers:executing-plans only after a subsystem plan exists. Steps in subsystem plans must use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current MultiSell prototype into a stable cross-border ecommerce product operations system whose core flows can be tested, deployed, and extended safely.

**Architecture:** Keep the existing FastAPI + Vue 3 + PostgreSQL shape and preserve the module auto-discovery convention. Stabilize the platform by first fixing environment, migrations, testability, and module boundaries, then complete the missing business loops: product creation, SKU, price, inventory, supplier, order, listing, AI enrichment, and reporting.

**Tech Stack:** Python 3.11+, FastAPI, SQLAlchemy 2.0 async, Alembic, PostgreSQL, pytest, Vue 3, TypeScript, Vite, Naive UI, Axios, Docker Compose, Nginx.

---

## Current State

- Frontend production build passes with `cd frontend && npm run build`.
- Frontend build emits a Vite warning because one chunk is larger than 500 kB.
- Backend tests do not currently reach business assertions because `product_management_test` on `localhost:5432` is not reachable.
- `backend/alembic/versions/c065b94903eb_baseline.py` exists but does not create tables.
- `frontend/src/router/modules/example.ts` registers the same order routes as `frontend/src/router/modules/order.ts`.
- `frontend/src/api/modules/example.ts` exports `orderApi`, which duplicates `frontend/src/api/modules/order.ts`.
- Frontend order pages and API calls exist, but backend `Order` and `OrderItem` models/routes are missing.
- Existing backend modules already cover product, category, brand, SKU, price, inventory, supplier, platform, listing, dashboard, search, auth, RBAC, and operation logs.

## Scope Split

This is too broad for one implementation plan. Execute it as these independent subsystem plans:

1. `docs/superpowers/plans/2026-06-13-runtime-and-migrations.md`
2. `docs/superpowers/plans/2026-06-13-core-product-quality.md`
3. `docs/superpowers/plans/2026-06-13-auth-rbac-audit.md`
4. `docs/superpowers/plans/2026-06-13-order-management.md`
5. `docs/superpowers/plans/2026-06-13-platform-listing-adapters.md`
6. `docs/superpowers/plans/2026-06-13-ai-excel-search-reporting.md`
7. `docs/superpowers/plans/2026-06-13-frontend-ux-and-performance.md`
8. `docs/superpowers/plans/2026-06-13-release-readiness.md`

Each subsystem must be independently testable and shippable.

## Target Boundaries

### Backend Module Contract

Every business module under `backend/app/<module>/` should own:

- `router.py`: HTTP endpoints only, request/response mapping only.
- `schemas.py`: Pydantic request/response models.
- `service.py`: business rules and database queries.
- `__init__.py`: exports `router`.
- Tests in `backend/tests/test_<module>.py`.

Do not put new business rules in `backend/app/main.py`. Keep the current auto-registration behavior, but cover it with tests.

### Frontend Module Contract

Every feature should own:

- Page views under `frontend/src/views/<module>/`.
- Route registration under `frontend/src/router/modules/<module>.ts`.
- API calls under `frontend/src/api/modules/<module>.ts` if the module is optional, or `frontend/src/api/index.ts` only for core shared APIs.

Remove runnable examples from `router/modules` and `api/modules`; examples should be documentation files, not imported production code.

### Database Contract

`backend/app/models.py` may remain a single model file for now, but schema changes must be expressed through Alembic migrations. Stop relying on `Base.metadata.create_all` outside test setup.

## Phase 0: Runtime And Migration Baseline

**Purpose:** Make the project reproducible before changing business behavior.

**Files:**
- Modify: `README.md`
- Modify: `docker-compose.yml`
- Modify: `backend/alembic/env.py`
- Modify: `backend/alembic/versions/c065b94903eb_baseline.py`
- Modify: `backend/tests/conftest.py`
- Modify: `backend/app/config.py`
- Create: `.env.example`

**Work:**

- Standardize backend runtime on Python 3.11+.
- Align local database credentials across `README.md`, `docker-compose.yml`, `backend/app/config.py`, and test setup.
- Replace empty Alembic baseline with a real schema baseline for all current models.
- Make test database setup explicit: either create `product_management_test` in Docker or use a dedicated test container.
- Document exact startup path:
  - `docker compose up -d db`
  - `cd backend && python3 -m venv .venv`
  - `cd backend && .venv/bin/pip install -r requirements.txt`
  - `cd backend && .venv/bin/alembic upgrade head`
  - `cd backend && .venv/bin/python seed.py`
  - `cd frontend && npm install`
  - `cd frontend && npm run dev`

**Acceptance Gates:**

- `cd backend && python3 -m pytest -q` reaches test assertions instead of failing on database connection.
- `cd backend && alembic upgrade head` creates the full schema from an empty database.
- `cd frontend && npm run build` still passes.

## Phase 1: Core Product Quality

**Purpose:** Stabilize the core money path before adding larger workflows.

**Files:**
- Modify: `backend/app/core/service.py`
- Modify: `backend/app/core/router.py`
- Modify: `backend/app/core/schemas.py`
- Modify: `backend/app/category/service.py`
- Modify: `backend/app/sku/service.py`
- Modify: `backend/app/price/service.py`
- Modify: `backend/app/inventory/service.py`
- Modify: `backend/app/supplier/service.py`
- Modify: `backend/app/models.py`
- Create: `backend/tests/test_product_lifecycle.py`

**Work:**

- Define one canonical product lifecycle: draft, active, inactive.
- Enforce delete rules for products with SKU, price, inventory, listing, or supplier records.
- Make SKU generation idempotent for the same spec matrix.
- Decide whether `Sku.stock` is retained as a denormalized read field or removed from active usage. Current model comment says it is deprecated, so services should read from `Inventory.quantity`.
- Add transaction-focused tests for:
  - create product
  - assign category and brand
  - define specs
  - generate SKUs
  - set price
  - update inventory
  - bind supplier
  - delete protections

**Acceptance Gates:**

- `backend/tests/test_product_lifecycle.py` passes against a fresh test database.
- No endpoint returns stale SKU stock when inventory has changed.
- Product deletion cannot orphan SKU, price, inventory, supplier, or listing records.

## Phase 2: Auth, RBAC, And Audit

**Purpose:** Make authorization behavior explicit and production-safe.

**Files:**
- Modify: `backend/app/auth/service.py`
- Modify: `backend/app/auth/router.py`
- Modify: `backend/app/rbac/service.py`
- Modify: `backend/app/rbac/router.py`
- Modify: `backend/app/operation_log/service.py`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/views/Login.vue`
- Create: `backend/tests/test_auth_rbac_integration.py`

**Work:**

- Keep `AUTH_ENABLED=True` as the production default.
- Define permission codes for product, SKU, inventory, price, supplier, platform, listing, order, report, user, and role management.
- Add dependency helpers for authenticated user and required permission.
- Ensure operation logs record actor, module, action, resource id, and outcome for state-changing endpoints.
- Make frontend route guard preserve the originally requested URL and redirect back after login.

**Acceptance Gates:**

- Anonymous users cannot access protected APIs when auth is enabled.
- Admin users can access all protected modules.
- Users without permission receive a 403 response.
- State-changing APIs write operation logs.

## Phase 3: Order Management

**Purpose:** Complete the half-built order module.

**Files:**
- Modify: `backend/app/models.py`
- Create: `backend/app/order/__init__.py`
- Create: `backend/app/order/router.py`
- Create: `backend/app/order/schemas.py`
- Create: `backend/app/order/service.py`
- Create: `backend/tests/test_order.py`
- Modify: `frontend/src/views/order/OrderList.vue`
- Modify: `frontend/src/views/order/OrderDetail.vue`
- Modify: `frontend/src/api/modules/order.ts`
- Modify: `frontend/src/router/modules/order.ts`
- Remove or move out of import path: `frontend/src/router/modules/example.ts`
- Remove or move out of import path: `frontend/src/api/modules/example.ts`

**Work:**

- Add `Order` and `OrderItem` tables with explicit order number, buyer data, status, totals, and item snapshots.
- Support order statuses: draft, pending, paid, shipped, delivered, cancelled.
- Validate status transitions in service code.
- Lock or validate inventory before paid/shipped transitions.
- Add paginated list, detail, create, and status update endpoints.
- Make existing frontend pages consume the real backend API.

**Acceptance Gates:**

- `GET /api/orders` returns paginated results.
- `GET /api/orders/{id}` returns order header and item lines.
- `POST /api/orders` creates an order with item snapshot totals.
- `PUT /api/orders/{id}/status` rejects invalid transitions.
- Frontend no longer has duplicate order routes or duplicate `orderApi` exports.

## Phase 4: Platform Listing Adapters

**Purpose:** Replace pure mock publishing with an adapter boundary that can support Ozon, Shopee, Wildberries, AliExpress, and Temu.

**Files:**
- Modify: `backend/app/platform/service.py`
- Modify: `backend/app/listing/router.py`
- Create: `backend/app/listing/service.py`
- Create: `backend/app/listing/adapters/base.py`
- Create: `backend/app/listing/adapters/mock.py`
- Create: `backend/tests/test_listing.py`
- Modify: `frontend/src/views/listing/ListingManage.vue`
- Modify: `frontend/src/views/platform/PlatformList.vue`

**Work:**

- Move publish logic out of router into service.
- Add adapter interface with `publish_product`, `sync_status`, and `validate_credentials`.
- Keep a mock adapter for local development.
- Encrypt platform API secrets before storage and never return raw secrets to frontend.
- Store platform-specific request and response payloads in `ProductListing.published_data` with redaction for secrets.

**Acceptance Gates:**

- Publishing through the mock adapter creates or updates a `ProductListing`.
- Platform secrets are not returned by list/detail APIs.
- Failed publishing records `status=failed` and a clear `sync_message`.

## Phase 5: AI, Excel, Search, And Reporting

**Purpose:** Consolidate useful automation around a stable product model.

**Files:**
- Modify: `backend/app/core/ai_service.py`
- Modify: `backend/app/core/excel_service.py`
- Modify: `backend/app/search/router.py`
- Modify: `backend/app/dashboard/service.py`
- Modify: `backend/app/dashboard/router.py`
- Create: `backend/tests/test_ai_service.py`
- Create: `backend/tests/test_excel_import_export.py`
- Create: `backend/tests/test_reports.py`
- Modify: `frontend/src/api/modules/excel.ts`
- Modify: `frontend/src/api/modules/report.ts`
- Modify: `frontend/src/views/report/index.vue`

**Work:**

- Keep AI service resilient: no API key means deterministic mock output; API failures return controlled errors and do not corrupt product records.
- Parse AI responses as JSON with schema validation.
- Make Excel import idempotent by a stable product code or SKU code strategy.
- Add report endpoints for product status, inventory health, listing status, and order status after Phase 3.
- Keep global search fast enough for current data volume; add indexes in migrations where needed.

**Acceptance Gates:**

- AI enhancement works with and without `LLM_API_KEY`.
- Excel export/import round-trips a small product set.
- Search returns product, SKU, and supplier records with stable response shape.
- Report endpoints have tests for empty database and seeded database.

## Phase 6: Frontend UX And Performance

**Purpose:** Turn generated pages into a coherent admin product.

**Files:**
- Modify: `frontend/src/components/Layout.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/api/http.ts`
- Modify: `frontend/src/views/product/ProductList.vue`
- Modify: `frontend/src/views/product/ProductForm.vue`
- Modify: `frontend/src/views/product/ProductDetail.vue`
- Modify: `frontend/src/views/sku/SkuManage.vue`
- Modify: `frontend/src/views/inventory/InventoryManage.vue`
- Modify: `frontend/src/views/order/OrderList.vue`
- Modify: `frontend/src/views/order/OrderDetail.vue`
- Modify: `frontend/vite.config.ts`

**Work:**

- Remove emoji from core controls and replace with existing icon components where practical.
- Add consistent empty, loading, error, and success states.
- Make tables usable on small screens by horizontal overflow and compact action groups.
- Add manual chunking for Naive UI and icon packages to address the current 500 kB chunk warning.
- Keep API error handling centralized in `frontend/src/api/http.ts`.

**Acceptance Gates:**

- `cd frontend && npm run build` passes.
- No duplicate routes are emitted in the router.
- Main admin flows are usable at 375px, 768px, and 1200px widths.
- Vite large chunk warning is removed or justified with an explicit budget.

## Phase 7: Release Readiness

**Purpose:** Make the project deployable and maintainable.

**Files:**
- Modify: `README.md`
- Modify: `docker-compose.yml`
- Modify: `backend/Dockerfile`
- Modify: `frontend/Dockerfile`
- Modify: `frontend/nginx.conf`
- Create: `.github/workflows/ci.yml`
- Create: `docs/architecture.md`
- Create: `docs/runbook.md`

**Work:**

- Align documented ports with Docker Compose ports. Current README says frontend `3001`, while Docker Compose exposes `3000:80`.
- Add CI checks for backend tests and frontend build.
- Add health checks for backend and frontend containers.
- Document database backup, migration, seed, reset, and rollback procedures.
- Define release checklist with migration, smoke test, and rollback steps.

**Acceptance Gates:**

- Fresh clone can be started from documented commands.
- CI runs backend tests and frontend build.
- Docker Compose starts database, backend, and frontend with documented URLs.
- `docs/architecture.md` explains module boundaries and data flow.

## Execution Order

1. Phase 0 first. Do not start feature work until runtime, migrations, and tests are reliable.
2. Phase 1 second. Product/SKU/price/inventory correctness is the foundation for all later modules.
3. Phase 2 third. Auth and audit should wrap stable business endpoints before production use.
4. Phase 3 fourth. Orders need stable product and inventory behavior.
5. Phase 4 fifth. Publishing depends on stable products and platform config.
6. Phase 5 sixth. AI, Excel, search, and reporting become safer after core data is reliable.
7. Phase 6 can run in parallel with Phases 3-5 if API contracts are frozen.
8. Phase 7 last, after major behavior has settled.

## Verification Matrix

Run these after each phase:

```bash
cd backend && python3 -m pytest -q
cd frontend && npm run build
```

Run these after migration or deployment changes:

```bash
docker compose build
docker compose up -d db
cd backend && alembic upgrade head
```

Run these before a release branch:

```bash
cd backend && python3 -m pytest -q
cd frontend && npm run build
docker compose up -d
```

## Non-Goals For The First Stabilization Pass

- Do not rewrite the backend in another framework.
- Do not replace Vue or Naive UI.
- Do not add real marketplace integrations before adapter boundaries and tests exist.
- Do not add new AI features beyond product enrichment until core product data is reliable.
- Do not split `backend/app/models.py` into many files until migrations and tests are stable.

## Decision Points For The Owner

Before Phase 3, decide whether orders reduce inventory at `paid` or `shipped`.

Before Phase 4, decide which marketplace is the first real adapter after mock publishing.

Before Phase 5, decide the canonical product import identity: product code, SKU code, or platform SKU.

Before Phase 7, decide target deployment: single VPS with Docker Compose, managed container platform, or cloud app platform.
