> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Auth RBAC Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make authentication, permission checks, and operation audit logs enforce real product-management behavior.

**Architecture:** Add reusable auth dependencies that return the current user and enforce permission codes. Apply the first complete slice to product state-changing endpoints and record operation logs after successful writes. Keep `AUTH_ENABLED=False` behavior compatible with existing tests and local demos.

**Tech Stack:** FastAPI dependencies, SQLAlchemy async, pytest, PostgreSQL.

---

### Task 1: Failing Auth/RBAC/Audit Tests

**Files:**
- Create: `backend/tests/test_auth_rbac_audit_integration.py`

- [x] **Step 1: Test protected product write requires login**

Assert `POST /api/products` returns HTTP 401 when `AUTH_ENABLED=True` and no token is supplied.

- [x] **Step 2: Test normal user without permission is forbidden**

Register/login a normal user and assert `POST /api/products` returns HTTP 403 without `product:create`.

- [x] **Step 3: Test permission grants product creation and writes audit log**

Create permission `product:create`, role, assign the role to the user, then assert product creation succeeds and `/api/operation-logs` includes a product create log.

- [x] **Step 4: Test admin role bypasses explicit permissions**

Mark the seeded admin user as `role="admin"`, login, and assert product creation succeeds without role-permission setup.

### Task 2: Add Permission Dependencies

**Files:**
- Create: `backend/app/auth/dependencies.py`
- Modify: `backend/app/auth/__init__.py`

- [x] **Step 1: Add `require_permission(permission_code)`**

Return the authenticated user when auth is disabled, when the user has `role="admin"`, or when RBAC grants the requested permission. Raise HTTP 403 otherwise.

- [x] **Step 2: Export auth dependencies**

Expose `get_current_user`, `require_auth`, and `require_permission` from `app.auth`.

### Task 3: Protect Product Writes And Add Audit Logs

**Files:**
- Modify: `backend/app/core/router.py`

- [x] **Step 1: Require product permissions**

Apply `product:create`, `product:update`, `product:delete`, `product:import`, `product:export`, and `product:ai` to product state-changing or sensitive endpoints.

- [x] **Step 2: Write operation logs after successful product mutations**

Record module `product`, action, resource id, content, and operator for create/update/delete/batch/duplicate/import/ai operations.

### Task 4: Verify

- [x] **Step 1: Run focused integration tests**

```bash
cd backend && python3 -m pytest tests/test_auth_rbac_audit_integration.py -q
```

- [x] **Step 2: Run full backend tests**

```bash
cd backend && python3 -m pytest -q
```

- [x] **Step 3: Run frontend build**

```bash
cd frontend && npm run build
```
