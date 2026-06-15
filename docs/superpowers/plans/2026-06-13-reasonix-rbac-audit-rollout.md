# Reasonix RBAC Audit Rollout Plan

> **For Reasonix / coding agent:** Execute this plan with TDD. Do not refactor unrelated code. Keep changes scoped to permission checks, audit logs, and tests.

**Goal:** Extend the existing `require_permission(...)` and operation audit pattern from the product module to the rest of the system.

**Current Context:** Product write endpoints already use `require_permission(...)` and write operation logs. The reusable dependency lives in `backend/app/auth/dependencies.py`. The guide is documented in `docs/PERMISSIONS_AND_AUDIT.md`.

**Tech Stack:** FastAPI, SQLAlchemy async, pytest, PostgreSQL.

---

## Must Read First

Read these files before editing:

- `docs/PROJECT_STATUS.md`
- `docs/PERMISSIONS_AND_AUDIT.md`
- `docs/DEVELOPMENT_GUIDE.md`
- `backend/app/auth/dependencies.py`
- `backend/app/core/router.py`
- `backend/tests/test_auth_rbac_audit_integration.py`

Use the product module as the implementation pattern.

## Non-Negotiable Rules

- Write failing tests first.
- Verify the focused test fails for the expected reason.
- Implement the smallest code needed.
- Run focused tests.
- Run full backend tests.
- Run frontend build.
- Do not change database schema unless a test proves it is required.
- Do not expose secrets in responses or logs.
- Do not remove existing behavior when `AUTH_ENABLED=False`.

## Permission Code Matrix

Use these permission codes:

| Module | Permission Codes |
| --- | --- |
| SKU | `sku:view`, `sku:create`, `sku:update`, `sku:delete` |
| Price | `price:view`, `price:update`, `price:batch_update` |
| Inventory | `inventory:view`, `inventory:update`, `inventory:adjust` |
| Supplier | `supplier:view`, `supplier:create`, `supplier:update`, `supplier:delete` |
| Platform | `platform:view`, `platform:create`, `platform:update`, `platform:delete` |
| Listing | `listing:view`, `listing:publish`, `listing:sync` |
| Order | `order:view`, `order:create`, `order:update_status`, `order:cancel` |
| Report | `report:view`, `dashboard:view` |
| RBAC | `rbac:view`, `rbac:manage` |
| Operation Log | `operation_log:view` |

## Audit Log Format

For successful write operations call:

```python
await OperationLogService.log(
    db,
    module="<module>",
    action="<action>",
    resource_id=str(resource_id),
    content="<human readable summary>",
    operator=current_user.username,
)
```

Do not log passwords, API keys, tokens, or raw platform secrets.

---

## Task 1: Add Test Helpers For Auth/RBAC

**Files:**

- Modify: `backend/tests/test_auth_rbac_audit_integration.py`
- Or create: `backend/tests/auth_helpers.py`

### Requirements

Extract reusable helpers from `test_auth_rbac_audit_integration.py`:

- enable auth fixture
- register/login user helper
- grant permission helper
- set admin role helper

### Acceptance

Existing test still passes:

```bash
cd backend && python3 -m pytest tests/test_auth_rbac_audit_integration.py -q
```

---

## Task 2: Protect Order APIs

**Files:**

- Modify: `backend/app/order/router.py`
- Test: `backend/tests/test_order_auth_audit.py`

### Required Behavior

- `POST /api/orders` requires `order:create`.
- `GET /api/orders` and `GET /api/orders/{id}` require `order:view`.
- `PUT /api/orders/{id}/status` requires:
  - `order:update_status` for normal transitions.
  - `order:cancel` when target status is `cancelled`.
- Successful create writes `module="order", action="create"`.
- Successful status update writes `module="order", action="update_status"` or `action="cancel"`.

### Tests

Cover:

- no token returns 401 when `AUTH_ENABLED=True`
- token without permission returns 403
- granted permission succeeds
- successful write creates operation log
- admin succeeds without explicit permission

---

## Task 3: Protect Inventory And Price APIs

**Files:**

- Modify: `backend/app/inventory/router.py`
- Modify: `backend/app/price/router.py`
- Test: `backend/tests/test_inventory_price_auth_audit.py`

### Required Behavior

Inventory:

- `GET /inventory/{sku_id}`, `/inventory/alerts`, `/inventory/{sku_id}/logs` require `inventory:view`.
- `PUT /inventory/{sku_id}` requires `inventory:update`.
- `POST /inventory/check` requires `inventory:view`.
- Successful inventory update writes `module="inventory", action="update"`.

Price:

- `GET /skus/{sku_id}/prices`, `/current-price`, `/price-history` require `price:view`.
- `POST /prices` requires `price:update`.
- `POST /prices/batch` requires `price:batch_update`.
- Successful price changes write audit logs.

---

## Task 4: Protect SKU And Supplier APIs

**Files:**

- Modify: `backend/app/sku/router.py`
- Modify: `backend/app/supplier/router.py`
- Test: `backend/tests/test_sku_supplier_auth_audit.py`

### Required Behavior

SKU:

- read endpoints require `sku:view`.
- spec definition and SKU generation require `sku:create`.
- SKU update requires `sku:update`.
- Successful spec definition, generation, and update write audit logs.

Supplier:

- list/detail/product-supplier reads require `supplier:view`.
- create/bind require `supplier:create`.
- update requires `supplier:update`.
- delete/unbind require `supplier:delete`.
- Successful writes create audit logs.

---

## Task 5: Protect Platform And Listing APIs

**Files:**

- Modify: `backend/app/platform/router.py`
- Modify: `backend/app/listing/router.py`
- Test: `backend/tests/test_platform_listing_auth_audit.py`

### Required Behavior

Platform:

- list/detail require `platform:view`.
- create requires `platform:create`.
- update requires `platform:update`.
- delete requires `platform:delete`.
- Successful writes create audit logs.

Listing:

- list/status endpoints require `listing:view`.
- publish endpoint requires `listing:publish`.
- Successful publish writes `module="listing", action="publish"`.
- Failed publish writes `module="listing", action="publish_failed"` only if a failed `ProductListing` was persisted.

---

## Task 6: Protect Reports, Dashboard, RBAC, And Operation Logs

**Files:**

- Modify: `backend/app/dashboard/router.py`
- Modify: `backend/app/rbac/router.py`
- Modify: `backend/app/operation_log/router.py`
- Test: `backend/tests/test_admin_surface_auth.py`

### Required Behavior

Dashboard and reports:

- `/dashboard/stats` requires `dashboard:view`.
- `/reports/*` requires `report:view`.

RBAC:

- role/user/permission read endpoints require `rbac:view`.
- role/user/permission write endpoints require `rbac:manage`.

Operation logs:

- `/operation-logs` and `/operation-logs/modules` require `operation_log:view`.

### Important

Be careful testing RBAC write endpoints. The test user needs permission to manage RBAC, or use admin.

---

## Task 7: Frontend Permission Awareness

**Files:**

- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/Layout.vue`
- Modify: relevant views only if needed.

### Required Behavior

- Preserve redirect target when unauthenticated user is sent to `/login`.
- Read user permissions from a frontend-accessible source.
- Hide menu entries and major action buttons if the current user lacks permission.

### Note

If the backend does not yet return permissions in `/auth/me`, add a backend test and implement a minimal `permissions` list in the user response.

---

## Final Verification

Run:

```bash
cd backend && python3 -m pytest -q
cd frontend && npm run build
```

Expected:

- all backend tests pass
- frontend build passes

Update these docs after completion:

- `docs/PROJECT_STATUS.md`
- `docs/PERMISSIONS_AND_AUDIT.md`
- `docs/ROADMAP.md`
