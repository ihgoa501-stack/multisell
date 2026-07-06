# Route Security Audit

**Date:** 2026-07-06
**Scope:** All `/api/v1` routes + public routes in `backend-go/internal/httpx/router.go`
**Method:** Trace each route through its middleware stack — JWT, RBAC, Audit.
**Status:** READ-ONLY audit. No code changed.

---

## Legend

| Icon | Meaning |
|------|---------|
| ✅ | Properly covered |
| ⚠️ | Present but with a caveat |
| ❌ | Missing |

---

## 1. Public Routes (no JWT)

These routes are registered on the root Gin engine (`r`) or the unauthenticated `/api/v1` group.

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/health` | ✅ N/A | ✅ N/A | ✅ Skipped (health suffix) | Unauthenticated health check |
| GET | `/metrics` | ✅ N/A | ✅ N/A | ✅ Skipped (GET, non-sensitive) | Prometheus; only when `metrics.enabled` |
| GET | `/swagger/*any` | ✅ N/A | ✅ N/A | ✅ Skipped (GET, non-sensitive) | Swagger UI |
| GET | `/ws` | ⚠️ N/A | ✅ N/A | ✅ N/A (WS) | JWT verified inside the WebSocket upgrade handler |
| GET | `/ws/extension` | ⚠️ N/A | ✅ N/A | ✅ N/A (WS) | JWT verified inside the extension WS handler |
| POST | `/api/v1/webhooks/:platform` | ✅ N/A | ✅ N/A | ✅ Audited (mutation) | External platform webhooks; signature-verified per adapter |
| GET | `/api/v1/health` | ✅ N/A | ✅ N/A | ✅ Skipped (health suffix) | API health |
| POST | `/api/v1/auth/login` | ✅ N/A | ✅ N/A | ✅ Audited (mutation) | Rate-limited (10/min/IP) |
| POST | `/api/v1/auth/register` | ✅ N/A | ✅ N/A | ✅ Audited (mutation) | Rate-limited (5/min/IP) |
| POST | `/api/v1/auth/refresh` | ✅ N/A | ✅ N/A | ✅ Audited (mutation) | Rate-limited (20/min/IP) |
| GET | `/api/v1/aios/scheduler/tasks` | ❌ Missing | ✅ N/A | ✅ Skipped (GET, non-sensitive) | **Vulnerability**: Exposes scheduler task run state (agent schedule, intervals, last-run times) to unauthenticated callers. Registered on root engine, not under the `protected` group. |

### Public routes verdict

One finding: `GET /api/v1/aios/scheduler/tasks` leaks internal scheduler state without authentication. This is not a direct data breach but exposes agent schedules and run history.

---

## 2. Authenticated Routes (JWT required)

All routes below have `middleware.Auth(cfg)` applied via the `protected` group.

### 2a. RBAC-protected routes (fine-grained permission check)

#### `rbac.manage` — RBAC administration

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/v1/rbac/roles` | ✅ | ✅ | ❌ GET, non-sensitive path | Read-only, low risk |
| POST | `/api/v1/rbac/roles` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/rbac/roles/:id` | ✅ | ✅ | ❌ GET | |
| PUT | `/api/v1/rbac/roles/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/rbac/roles/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/rbac/roles/:id/permissions` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/rbac/roles/:id/permissions` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/rbac/permissions` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/rbac/permissions` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/rbac/permissions/:id` | ✅ | ✅ | ❌ GET | |
| PUT | `/api/v1/rbac/permissions/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/rbac/permissions/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/rbac/users/:id/roles` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/rbac/users/:id/roles` | ✅ | ✅ | ✅ Mutation | |

#### `rbac.*` — RBAC self-service (any authenticated user)

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/v1/rbac/current/permissions` | ✅ | ✅ Public | ❌ GET | Read-only, user's own perms |

#### `product.read` — Product master, SKU, Product Hub

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/v1/product-master` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/product-master/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/product-master` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/product-master/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/product-master/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/product-master/:id/specs` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/product-master/:id/specs` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/product-master/:id/specs/:spec_id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/product-master/:id/specs/:spec_id` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/product-master/:id/specs/:spec_id/values` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/product-master/:id/skus` | ✅ | ✅ | ❌ GET | |
| PUT | `/api/v1/spec-values/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/spec-values/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/skus` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/skus/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/skus` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/skus/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/skus/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/product-hub` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/product-hub/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/product-hub` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/product-hub/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/product-hub/:id` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/product-hub/:id/transition` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/product-hub/:id/hub` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/product-hub/:id/variants` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/product-hub/variants` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/product-hub/:id/offers` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/product-hub/offers` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/product-hub/:id/samples` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/product-hub/samples` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/product-hub/:id/costs` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/product-hub/costs` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/product-hub/costs/:costId/confirm` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/products/:id/versions` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/products/:id/versions/:versionId` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/products/:id/versions/:versionId/rollback` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/products/:id/decisions` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/products/:id/freshness` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/products/freshness/stale` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/products/:id/freshness/verify` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/products/:id/relations` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/products/:id/discover-relations` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/products/360/summary` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/products/decision` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/products/relations` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/products/relations/:id` | ✅ | ✅ | ✅ Mutation | |

#### `inventory.read` — Inventory

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/v1/inventory` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/inventory/:id` | ✅ | ✅ | ❌ GET | |
| PUT | `/api/v1/inventory/:id` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/inventory/:id/lock` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/inventory/:id/unlock` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/inventory/logs` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/inventory/sku/:sku_id/warehouses` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/inventory/locations` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/inventory/transfers` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/inventory/sync-cross-platform/:productId` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/inventory/oversell-report` | ✅ | ✅ | ❌ GET | |

#### `listing.read` — Listing + Listing Tasks

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/v1/listings` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/listings/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/listings` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/listings/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/listings/:id` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listings/:id/publish` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listings/:id/sync` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listing` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listing/products/:product_id/publish/:platform_id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/listing/products/:product_id/listings` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/listing/listing-tasks/from-decisions` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listing/listing-tasks/:task_id/recheck` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listing/listing-tasks/:task_id/cancel` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listing/listing-tasks/:task_id/publish` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/listing-tasks` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/listing-tasks/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/listing-tasks` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/listing-tasks/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/listing-tasks/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/listing-tasks/:id/items` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/listing-tasks/:id/items` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/listing-tasks/:id/items/:item_id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/listing-tasks/:id/items/:item_id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/listing-task/stats` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/listing-task/retry-all` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listing-task/:task_id/execute` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listing-task/:task_id/retry-failed` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/listing-task/:task_id/items/:item_id/retry` | ✅ | ✅ | ✅ Mutation | |

#### `shipping.read` — Shipping

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| POST | `/api/v1/shipping/quote` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/providers` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/shipping/providers/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/providers` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/shipping/providers/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/shipping/providers/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/channels` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/shipping/channels/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/channels` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/shipping/channels/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/shipping/channels/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/zones` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/zones` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/shipping/zones/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/rules` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/rules` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/shipping/rules/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/bill-batches` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/shipping/bill-batches/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/bill-batches` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/shipping/bill-batches/import` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/shipping/bill-batches/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/bill-batches/:id/items` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/bill-batches/:id/reconcile` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/bill-batches/:id/anomalies` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/quote-unified` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/shipping/snapshots` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/snapshots` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/shipping/snapshots/:orderId` | ✅ | ✅ | ❌ GET | |
| PUT | `/api/v1/shipping/bill-items/:id/review` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/rules/:id/versions` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/tracking` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/tracking` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/shipping/tracking/:orderId` | ✅ | ✅ | ❌ GET | |
| PUT | `/api/v1/shipping/tracking/:id/event` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/shipping/tracking/:id/exception` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/shipping/carrier-performance` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/shipping/carriers` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/shipping/carriers/:code/quote` | ✅ | ✅ | ✅ Mutation | |

#### `order.read` — Order + Order Import

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/v1/order` | ✅ | ✅ | ⚠️ Sensitive GET | Path matches sensitiveReadPaths (`/api/v1/orders`) |
| GET | `/api/v1/order/summary` | ✅ | ✅ | ⚠️ Sensitive GET | |
| GET | `/api/v1/order/:id` | ✅ | ✅ | ⚠️ Sensitive GET | |
| POST | `/api/v1/order` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/order/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/order/:id` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/order/:id/status` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/order-import` | ✅ | ✅ | ⚠️ Sensitive GET | |
| GET | `/api/v1/order-import/summary` | ✅ | ✅ | ⚠️ Sensitive GET | |
| GET | `/api/v1/order-import/:id` | ✅ | ✅ | ⚠️ Sensitive GET | |
| POST | `/api/v1/order-import` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/order-import/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/order-import/:id` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/order-import/:id/process` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/order-import/:id/complete` | ✅ | ✅ | ✅ Mutation | |

#### `settlement.read` — Settlement

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/v1/settlement/summary` | ✅ | ✅ | ⚠️ Sensitive GET | Path matches sensitiveReadPaths |
| GET | `/api/v1/settlement` | ✅ | ✅ | ⚠️ Sensitive GET | |
| POST | `/api/v1/settlement` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/settlement/:id` | ✅ | ✅ | ⚠️ Sensitive GET | |
| PUT | `/api/v1/settlement/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/settlement/:id` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/settlement/:id/reconcile` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/settlement/:id/items` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/settlement/:id/items` | ✅ | ✅ | ⚠️ Sensitive GET | |
| PUT | `/api/v1/settlement/items/:item_id/reconciliation` | ✅ | ✅ | ✅ Mutation | |

#### `finance.read` — Finance + Price

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| POST | `/api/v1/finance/profit/calculate` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/finance/profit/batch-calculate` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/finance/profit/summary` | ✅ | ✅ | ⚠️ Sensitive GET | Path matches sensitiveReadPaths |
| GET | `/api/v1/finance/profit/ranking` | ✅ | ✅ | ⚠️ Sensitive GET | |
| GET | `/api/v1/finance/summary` | ✅ | ✅ | ⚠️ Sensitive GET | |
| GET | `/api/v1/finance/profit-summary` | ✅ | ✅ | ⚠️ Sensitive GET | |
| GET | `/api/v1/finance/ledger` | ✅ | ✅ | ⚠️ Sensitive GET | |
| POST | `/api/v1/finance/mock` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/finance/orders/:order_id/ledger` | ✅ | ✅ | ⚠️ Sensitive GET | |
| GET | `/api/v1/finance/orders/:order_id/profit` | ✅ | ✅ | ⚠️ Sensitive GET | |
| POST | `/api/v1/finance/orders/:order_id/ledger/rebuild` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/finance/accounts` | ✅ | ✅ | ⚠️ Sensitive GET | |
| POST | `/api/v1/finance/accounts` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/finance/accounts/:id` | ✅ | ✅ | ⚠️ Sensitive GET | |
| PUT | `/api/v1/finance/accounts/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/finance/accounts/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/finance/transactions` | ✅ | ✅ | ⚠️ Sensitive GET | |
| POST | `/api/v1/finance/transactions` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/prices` | ✅ | ✅ | ❌ GET | Behind finance group |
| GET | `/api/v1/prices/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/prices` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/prices/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/prices/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/skus/:id/prices` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/skus/:id/current-price` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/skus/:id/price-history` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/competitor-prices` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/competitor-prices/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/competitor-prices` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/competitor-prices/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/pricing-strategies` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/pricing-strategies/:id` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/pricing-strategies` | ✅ | ✅ | ✅ Mutation | |
| PUT | `/api/v1/pricing-strategies/:id` | ✅ | ✅ | ✅ Mutation | |
| DELETE | `/api/v1/pricing-strategies/:id` | ✅ | ✅ | ✅ Mutation | |
| GET | `/api/v1/pricing-recommendations` | ✅ | ✅ | ❌ GET | |
| POST | `/api/v1/pricing-recommendations/generate` | ✅ | ✅ | ✅ Mutation | |
| POST | `/api/v1/pricing-recommendations/:id/apply` | ✅ | ✅ | ✅ Mutation | |

#### `report.read` — Reports

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET | `/api/v1/report/sales` | ✅ | ✅ | ❌ GET | Read-only |
| GET | `/api/v1/report/profit` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/report/inventory` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/report/settlement` | ✅ | ✅ | ❌ GET | |
| GET | `/api/v1/report/platform-fee` | ✅ | ✅ | ❌ GET | |

---

### 2b. JWT-only routes (no RBAC)

All routes below require a valid JWT but have **no fine-grained RBAC permission check**. Any authenticated user can access them.

#### AI / Agent routes

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| POST | `/api/v1/ai/chat` | ✅ | ❌ | ✅ Mutation | |
| POST | `/api/v1/ai/run` | ✅ | ❌ | ✅ Mutation | |
| GET | `/api/v1/ai/traces` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/ai/actions` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/ai/agents` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/ai/agents/specs` | ✅ | ❌ | ❌ GET | |
| POST | `/api/v1/ai/actions` | ✅ | ❌ | ✅ Mutation | |
| GET | `/api/v1/ai/traces/:trace_id` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/ai/actions/:id` | ✅ | ❌ | ❌ GET | |
| POST | `/api/v1/ai/actions/:id/approve` | ✅ | ❌ | ✅ Mutation | |
| POST | `/api/v1/ai/actions/:id/reject` | ✅ | ❌ | ✅ Mutation | |
| POST | `/api/v1/ai/actions/:id/execute` | ✅ | ❌ | ✅ Mutation | |
| POST | `/api/v1/ai/actions/:id/review` | ✅ | ❌ | ✅ Mutation | |
| POST | `/api/v1/ai/moa` | ✅ | ❌ | ✅ Mutation | MOA (conditional) |
| POST | `/api/v1/auth/me` | ✅ | ❌ | ❌ GET | Actually: GET /api/v1/auth/me (JWT-protected in auth routes) |
| GET | `/api/v1/agents` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/agents/evolution` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/agents/entropy` | ✅ | ❌ | ❌ GET | |
| POST | `/api/v1/agents` | ✅ | ❌ | ✅ Mutation | |
| GET | `/api/v1/agents/:id` | ✅ | ❌ | ❌ GET | |
| POST | `/api/v1/agents/:id/actions` | ✅ | ❌ | ✅ Mutation | |
| GET | `/api/v1/agentos` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/agentos/status` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/agentos/work-items` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/agentos/autonomy` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/agentos/work-items/:id` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/agentos/agent-timeline` | ✅ | ❌ | ❌ GET | |
| GET | `/api/v1/agentos/failures` | ✅ | ❌ | ❌ GET | |

#### Category, Brand, Supplier, Purchase

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET/POST/PUT/DEL | `/api/v1/categories/*` | ✅ | ❌ | ✅ Mutation / ❌ GET | Standard CRUD |
| GET/POST/PUT/DEL | `/api/v1/brands/*` | ✅ | ❌ | ✅ Mutation / ❌ GET | Standard CRUD |
| GET/POST/PUT/DEL | `/api/v1/suppliers/*` | ✅ | ❌ | ✅ Mutation / ❌ GET | Standard CRUD |
| GET/POST/PUT/DEL | `/api/v1/product-suppliers/*` | ✅ | ❌ | ✅ Mutation / ❌ GET | |
| GET/POST | `/api/v1/purchase/orders` | ✅ | ❌ | ✅ POST Mutation | |
| POST | `/api/v1/purchase/orders/:id/approve` | ✅ | ❌ | ✅ Mutation | |
| POST | `/api/v1/purchase/orders/:id/receive` | ✅ | ❌ | ✅ Mutation | |
| POST | `/api/v1/purchase/orders/:id/cancel` | ✅ | ❌ | ✅ Mutation | |
| GET | `/api/v1/purchase/suggestions` | ✅ | ❌ | ❌ GET | |
| POST | `/api/v1/purchase/suggestions/generate` | ✅ | ❌ | ✅ Mutation | |

#### Commerce & Finance (no RBAC)

| Module | Method | Path | JWT? | RBAC? | Audited? |
|--------|--------|------|:----:|:-----:|:--------:|
| Platform | GET/POST/PUT/DEL | `/api/v1/platforms/*` | ✅ | ❌ | ✅ Mutation |
| Platform | GET/POST/PUT/DEL | `/api/v1/stores/*` | ✅ | ❌ | ✅ Mutation |
| PlatformFee | GET/POST/PUT/DEL | `/api/v1/platform-fee/*` | ✅ | ❌ | ✅ Mutation |
| ExchangeRate | GET/POST/DEL/PUT | `/api/v1/exchange-rates/*` | ✅ | ❌ | ✅ Mutation |
| Cost | GET | `/api/v1/cost/dashboard` | ✅ | ❌ | ❌ GET |
| Profit | GET | `/api/v1/profit/summaries` | ✅ | ❌ | ❌ GET |
| Profit | GET | `/api/v1/profit/summary/:productId` | ✅ | ❌ | ❌ GET |

#### Candidate / Completeness / Compliance / Loop / Mock

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET/POST/PUT/DEL | `/api/v1/candidates/*` | ✅ | ❌ | ✅ Mutation | Full CRUD + seed |
| GET/POST | `/api/v1/completeness/*` | ✅ | ❌ | ✅ Mutation | |
| GET/POST/PUT | `/api/v1/compliance/*` | ✅ | ❌ | ✅ Mutation | |
| GET | `/api/v1/loop/recommendations` | ✅ | ❌ | ❌ GET | |
| POST | `/api/v1/loop/evaluate/:productId` | ✅ | ❌ | ✅ Mutation | |
| POST/GET | `/api/v1/mock/*` | ✅ | ❌ | ✅ Mutation | Mock data seeds |

#### Decision / Allocation / Approval / Workflow / Orchestration

| Module | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| Decision | `/api/v1/decision/*` | ✅ | ❌ | ✅ Mutation | Full CRUD + approve/reject |
| Allocation | `/api/v1/allocation/*` | ✅ | ❌ | ✅ Mutation | Warehouses, rules, cost batches |
| Approval | `/api/v1/approval/*` | ✅ | ❌ | ✅ Mutation | + GET for pending/stats |
| Workflow | `/api/v1/workflow/*` | ✅ | ❌ | ✅ Mutation | Defs + runs |
| Orchestration | `/api/v1/orchestration/*` | ✅ | ❌ | ✅ Mutation | Pipeline start/retry |

#### Content / ImageGen / Sentiment / ProductAnalysis

| Module | Path | JWT? | RBAC? | Audited? |
|--------|------|:----:|:-----:|:--------:|
| Content | `/api/v1/content/*` | ✅ | ❌ | ✅ Mutation (POST) |
| ImageGen | `/api/v1/image-gen/*` | ✅ | ❌ | ✅ Mutation |
| Sentiment | `/api/v1/sentiment/*` | ✅ | ❌ | ✅ Mutation |
| ProductAnalysis | `/api/v1/product-analysis/*` | ✅ | ❌ | ✅ Mutation |

#### Support / Notification / Exceptions / Search / Dashboard / Settings

| Module | Path | JWT? | RBAC? | Audited? |
|--------|------|:----:|:-----:|:--------:|
| Support | `/api/v1/support/conversations/*` | ✅ | ❌ | ✅ Mutation |
| Support | `/api/v1/support/templates/*` | ✅ | ❌ | ✅ Mutation |
| Support | `/api/v1/support/blacklist/*` | ✅ | ❌ | ✅ Mutation |
| Notification | `/api/v1/notification/*` | ✅ | ❌ | ✅ Mutation |
| Exceptions | `/api/v1/exceptions/*` | ✅ | ❌ | ✅ Mutation |
| Search | `/api/v1/search/*` | ✅ | ❌ | ❌ GET (read-only) |
| Dashboard | `/api/v1/dashboard/*` | ✅ | ❌ | ❌ GET (read-only) |
| Settings | `/api/v1/settings/*` | ✅ | ❌ | ✅ Mutation (PUT) |

#### Import / Export / Integrations

| Module | Path | JWT? | RBAC? | Audited? |
|--------|------|:----:|:-----:|:--------:|
| ImportBatch | `/api/v1/import-batch/*` | ✅ | ❌ | ✅ Mutation |
| OperationLog | `/api/v1/operation-log/*` | ✅ | ❌ | ✅ Mutation |
| Integrations | `/api/v1/platform-integrations/*` | ✅ | ❌ | ✅ Mutation |
| WebhookAdmin | `/api/v1/platform-webhooks/*` | ✅ | ❌ | ✅ Mutation |

#### Supply Chain / Logistics / Tariff / Sourcing

| Module | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| ActionPolicy | `/api/v1/policy/*` | ✅ | ❌ | ✅ Mutation | |
| Aftersales | `/api/v1/aftersales/*` | ✅ | ❌ | ✅ Mutation | |
| Sourcing | `/api/v1/sourcing/*` | ✅ | ❌ | ✅ Mutation | POST fetch |
| Sourcing1688 | `/api/v1/sourcing-1688/*` | ✅ | ❌ | ✅ Mutation | Full CRUD |
| Tariff | `/api/v1/tariff/*` | ✅ | ❌ | ✅ Mutation | Full CRUD |
| Logistics | `/api/v1/logistics/quote` | ✅ | ❌ | ✅ Mutation | POST |
| Consolidation | `/api/v1/consolidation/*` | ✅ | ❌ | ✅ Mutation | |
| SupplyChain | `/api/v1/supply-chain/flows/*` | ✅ | ❌ | ✅ Mutation | |
| SupplyChain | `/api/v1/supply-chain/tracking/*` | ✅ | ❌ | ✅ Mutation | |
| LandedCost | `/api/v1/landed-cost/*` | ✅ | ❌ | ✅ Mutation | POST calculate |

#### Agent System modules (no RBAC)

| Module | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| TrustScore | `/api/v1/trust-scores/*` | ✅ | ❌ | ✅ Mutation | POST recalculate, PUT level |
| AgentRule | `/api/v1/agent-rules/*` | ✅ | ❌ | ✅ Mutation | Full CRUD |
| PersonalRule | `/api/v1/agents/rules/*` | ✅ | ❌ | ✅ Mutation | |
| AgentLearning | `/api/v1/agent-learning/*` | ✅ | ❌ | ✅ Mutation | |
| Evolution | `/api/v1/evolution/*` | ✅ | ❌ | ✅ Mutation | |
| Entropy | `/api/v1/entropy/*` | ✅ | ❌ | ✅ Mutation | POST defense |
| Metabolism | `/api/v1/metabolism/*` | ✅ | ❌ | ✅ Mutation | POST dry-run, execute |
| Owner | `/api/v1/owner/*` | ✅ | ❌ | ✅ POST mutation | POST suggestions feedback |

#### Feedback (authenticated group only)

| Method | Path | JWT? | RBAC? | Audited? | Notes |
|--------|------|:----:|:-----:|:--------:|-------|
| GET/POST/PUT/DEL | `/api/v1/feedback/*` | ✅ | ❌ | ✅ Mutation | **Comment says "public" but sits under JWT-protected parent group** |

---

## 3. WebSocket Routes

| Path | JWT? | RBAC? | Notes |
|------|:----:|:-----:|-------|
| `/ws` | ⚠️ In-handler | ❌ | JWT verified during WS upgrade via `realtime.NewHandler`; token in query string |
| `/ws/extension` | ⚠️ In-handler | ❌ | Same pattern; JWT verified during upgrade |

---

## Summary of Findings

### Finding 1 — `GET /api/v1/aios/scheduler/tasks` has no JWT (HIGH)
**File:** `backend-go/internal/httpx/router.go`, line 521
**Issue:** This route is registered on the root engine `r` instead of the `protected` group. It exposes the full scheduler task run state: agent IDs, decision points, intervals, last-run times, and run counts.
**Fix:** Move under the `protected` group, or add `middleware.Auth(cfg)`.

### Finding 2 — Most modules lack RBAC (MEDIUM)
**Issue:** 40+ domain modules registered directly on the `protected` group have **no fine-grained RBAC permission check**. Any authenticated user can access every route in these modules. While this is common in single-owner/MVP systems, it becomes a risk as multi-tenant or multi-role scenarios emerge.
**Modules affected:** AI, agents, agentos, category, brand, supplier, purchase, candidate, completeness, compliance, profit, loop, mock, owner, agentlearning, approval, landedcost, orchestration, workflow, personalrule, platform, platformfee, decision, allocation, exceptions, notification, dashboard, support, search, settings, imagegen, importbatch, content, integrations, actionpolicy, aftersales, sourcing, sourcing1688, tariff, logistics, consolidation, supplychain, productanalysis, trustscore, exchangerate, agentrule, evolution, entropy, cost, metabolism, feedback, sentiment, webhook admin.

### Finding 3 — Feedback "public" routes are behind JWT (LOW)
**File:** `backend-go/internal/domain/feedback/routes.go`, line 69
**Issue:** The code comment says `// Public routes (no auth required — for widget/portal submissions)` but the `rg` parameter is the JWT-protected `protected` group, so these routes actually require authentication. This is either a misleading comment or routes that were supposed to be widget-accessible without auth but never were.

### Finding 4 — Audit middleware covers all mutations globally (GOOD)
**Verdict:** ✅ The Audit middleware is registered as global middleware on the root Gin engine, so all mutations (POST/PUT/PATCH/DELETE) across all routes — including public ones like login/register/webhook — are logged to `operation_log`. Sensitive GET paths (finance, orders, settlement, user, rbac) are also logged. No route has deliberately missing audit coverage.

### Finding 5 — Sensitive-path prefix matching for GET audit is incomplete (LOW)
**File:** `backend-go/internal/httpx/middleware/audit.go`, lines 37-43
**Issue:** The `sensitiveReadPaths` list includes `/api/v1/finance` and `/api/v1/settlement` but uses string prefix matching so `/api/v1/finance/...` sub-routes are covered. However, sensitive modules like `/api/v1/trust-scores`, `/api/v1/evolution`, `/api/v1/entropy`, and `/api/v1/candidates` are not in this list — their GET operations go un-audited.
**Risk:** Low — these are not directly financial, but trust scores and candidate data are commercially sensitive.

### Finding 6 — AI routes execute agents without RBAC (MEDIUM)
**Issue:** `POST /api/v1/ai/run` and `POST /api/v1/ai/chat` allow any authenticated user to invoke any agent (A1-A12, G0-G3, M1) without checking whether the user has permission for that agent or action. The `POST /api/v1/ai/actions/:id/approve` and `POST /api/v1/agents/:id/actions` similarly allow any user to approve agent actions without verifying ownership or permission.

---

## Audit Coverage Summary

| Layer | Coverage | Score |
|-------|----------|:-----:|
| JWT on protected routes | ✅ All 50+ modules behind JWT | ✅ |
| JWT on public routes | 1 route exposed (`scheduler/tasks`) | ⚠️ |
| RBAC on commerce/finance modules | ✅ 8 modules with fine-grained RBAC | ✅ |
| RBAC on system/agent modules | ❌ 40+ modules without RBAC | ❌ |
| Audit on mutations | ✅ Global middleware, all mutations captured | ✅ |
| Audit on sensitive GETs | ⚠️ Partial — 5 sensitive paths, gaps remain | ⚠️ |
| Rate limiting on auth | ✅ Login (10/min), register (5/min), refresh (20/min) | ✅ |
| Webhook signature verification | ✅ Platform adapters implement WebhookVerifier | ✅ |
| WebSocket JWT verification | ✅ In-handler upgrade validation | ✅ |
