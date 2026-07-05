> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Platform Listing Adapters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace inline mock publishing with a service and adapter boundary, and block publishing when product data is incomplete.

**Architecture:** Keep existing HTTP routes, move publishing rules into `ListingService`, and route all platform-specific behavior through adapter classes. Start with `MockListingAdapter` and a registry that maps platform codes to adapters.

**Tech Stack:** FastAPI, SQLAlchemy async, PostgreSQL, pytest.

---

### Task 1: Add Failing Listing Tests

**Files:**
- Create: `backend/tests/test_listing.py`

- [x] **Step 1: Test publish preflight**

Create a product with no main image, SKU, price, or inventory. Assert publishing returns code 400 and lists missing requirements.

- [x] **Step 2: Test mock publish success**

Create a complete product with main image, generated SKU, sale price, inventory, and an active platform. Assert publishing creates a synced listing with a platform URL and published data.

- [x] **Step 3: Test platform secrets remain hidden**

Create a platform with an API key. Assert `GET /api/platforms` and `GET /api/platforms/{id}` do not include `api_key`.

### Task 2: Add Listing Service And Adapter Boundary

**Files:**
- Create: `backend/app/listing/service.py`
- Create: `backend/app/listing/adapters/__init__.py`
- Create: `backend/app/listing/adapters/base.py`
- Create: `backend/app/listing/adapters/mock.py`
- Modify: `backend/app/listing/router.py`

- [x] **Step 1: Add adapter interface**

Define `PublishResult` and `ListingAdapter`.

- [x] **Step 2: Add mock adapter**

Return deterministic platform product ID, platform URL, and payload.

- [x] **Step 3: Add preflight completeness checks**

Require product name, main image, at least one active SKU, sale price for every active SKU, inventory for every active SKU, and active platform.

- [x] **Step 4: Move publish logic into service**

Create or update `ProductListing`, update `Product.platform_statuses`, record failures, and return the same response shape used by the frontend.

### Task 3: Verify

- [x] **Step 1: Run listing tests**

```bash
cd backend && python3 -m pytest tests/test_listing.py -q
```

- [x] **Step 2: Run full backend tests**

```bash
cd backend && python3 -m pytest -q
```

- [x] **Step 3: Run frontend build**

```bash
cd frontend && npm run build
```
