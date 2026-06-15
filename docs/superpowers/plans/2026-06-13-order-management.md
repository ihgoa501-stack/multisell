# Order Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the missing backend order module and connect it to the existing order frontend.

**Architecture:** Add `Order`, `OrderItem`, and `OrderStatusLog` models to the existing shared SQLAlchemy model file. Keep order HTTP endpoints in `backend/app/order/router.py`, validation schemas in `schemas.py`, and business rules in `service.py`.

**Tech Stack:** FastAPI, SQLAlchemy async, PostgreSQL, pytest, Vue 3, Vite.

---

### Task 1: Add Failing Order API Tests

**Files:**
- Create: `backend/tests/test_order.py`

- [ ] **Step 1: Test order creation**

Use API calls to create a product, generate one SKU, set price, then `POST /api/orders`.

- [ ] **Step 2: Test paginated list and detail**

Assert `GET /api/orders` returns `records`, `total`, and frontend fields including `order_no`, `product_name`, `quantity`, `total_amount`, `status`, and `created_at`.

- [ ] **Step 3: Test status transitions**

Assert `pending -> shipped` is rejected, while `pending -> paid -> shipped` succeeds and status logs are returned in detail.

### Task 2: Implement Backend Order Module

**Files:**
- Modify: `backend/app/models.py`
- Create: `backend/app/order/__init__.py`
- Create: `backend/app/order/schemas.py`
- Create: `backend/app/order/service.py`
- Create: `backend/app/order/router.py`

- [ ] **Step 1: Add models**

Add `Order`, `OrderItem`, and `OrderStatusLog`.

- [ ] **Step 2: Add schemas**

Add create, status update, item, list, and detail schemas.

- [ ] **Step 3: Add service**

Generate stable order numbers, snapshot SKU/product data, calculate totals, and validate status transitions.

- [ ] **Step 4: Add router**

Expose `POST /orders`, `GET /orders`, `GET /orders/{id}`, and `PUT /orders/{id}/status`.

### Task 3: Clean Duplicate Frontend Example Modules

**Files:**
- Delete: `frontend/src/router/modules/example.ts`
- Delete: `frontend/src/api/modules/example.ts`

- [ ] **Step 1: Remove imported runnable examples**

Delete the two example modules because Vite imports all files in those folders.

### Task 4: Verify

**Files:**
- No further edits unless verification exposes root cause.

- [ ] **Step 1: Run order tests**

```bash
cd backend && python3 -m pytest tests/test_order.py -q
```

- [ ] **Step 2: Run full backend tests**

```bash
cd backend && python3 -m pytest -q
```

- [ ] **Step 3: Run frontend build**

```bash
cd frontend && npm run build
```
