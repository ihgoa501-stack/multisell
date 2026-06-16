# Order Import Operational Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make imported external orders enter LingMirror's real operating chain: valid multi-SKU order creation, ledger rebuild, exception generation, and clear import processing results.

**Architecture:** Keep CSV order import as the first real-data adapter, but do not leave imported orders as isolated records. Add a small orchestration service around existing modules instead of duplicating finance or exception logic: `order_import` creates/matches orders, `finance` rebuilds ledger, `exceptions` generates workbench items, and `order_import` stores per-row/per-batch processing results.

**Tech Stack:** FastAPI, SQLAlchemy async, PostgreSQL, Alembic, Pydantic, pytest, Vue 3, TypeScript, Vite, Naive UI.

---

## Current Reality

Current branch observed during planning:

```text
codex/profit-bi-dashboard
```

Current HEAD observed during planning:

```text
2a9c72c
```

Current dirty files observed during planning:

```text
 M frontend/package.json
?? .qoder/
?? frontend/pnpm-lock.yaml
?? frontend/pnpm-workspace.yaml
```

Before implementing this plan, the agent must confirm whether those frontend package-manager files belong to the current user's tool setup. Do not stage them unless the user explicitly confirms they are intended.

Stage 11 CSV order import exists:

- `backend/app/order_import/models.py`
- `backend/app/order_import/schemas.py`
- `backend/app/order_import/service.py`
- `backend/app/order_import/router.py`
- `backend/tests/test_order_import_csv_adapter.py`
- `frontend/src/api/modules/orderImport.ts`
- `frontend/src/views/order/OrderImport.vue`
- `frontend/src/router/modules/orderImport.ts`

Stage 11 also registered adapter:

```text
adapter_code = csv_order
supports_order_import = true
auth_type = none
```

## Why Stage 12 Is Needed

Stage 11 gets external orders into the system. That is necessary, but not enough.

The imported order must then enter the operating chain:

```text
CSV import
  -> order creation / duplicate skip / row failure
  -> ledger rebuild
  -> exception generation
  -> user sees chain status
```

Without this stage, imported orders can sit in the database without profit truth, exception visibility, or BI usefulness.

## Important Design Correction

The current `OrderImportService.process_batch()` path should be reviewed carefully for this case:

```text
same platform_order_no
multiple CSV rows
different sku_code values
```

The desired behavior is:

```text
one system order
multiple OrderItem rows
each import row linked to the same order_id
```

If the current implementation only marks the second row as imported but does not add a second `OrderItem`, fix that first. Do not build downstream automation on a broken order shape.

## Stage 12 Scope

Build:

- Correct multi-SKU import behavior for same `platform_order_no`.
- Add import chain processing fields to batch/items.
- Add explicit batch processing endpoint.
- Rebuild finance ledger for created/matched imported orders.
- Generate exceptions after ledger rebuild.
- Show processing result in order import UI.

Do not build:

- Real Amazon/TikTok/Temu/Walmart API.
- OAuth.
- Automatic background jobs.
- BI changes.
- New platform settlement logic.
- New shipping bill logic.
- AI agent execution.

## Data Flow

```text
POST /api/order-imports/csv
  -> parse rows
  -> create OrderImportBatch and OrderImportItem rows
  -> create or match sales_order rows
  -> create all sales_order_item rows
  -> return import result

POST /api/order-imports/{batch_id}/process-chain
  -> collect distinct created/matched order ids
  -> rebuild finance ledger for each order
  -> generate exception workbench items
  -> update batch chain counters
  -> return chain processing summary
```

## New Or Updated States

Keep import row statuses:

```text
imported
created_order
skipped_duplicate
failed
```

Add batch chain status:

```text
chain_pending
chain_processed
chain_failed
```

Add item chain status:

```text
chain_pending
ledger_rebuilt
exception_generated
chain_failed
```

## Files

Backend:

- Modify: `backend/app/order_import/models.py`
- Modify: `backend/app/order_import/schemas.py`
- Modify: `backend/app/order_import/service.py`
- Modify: `backend/app/order_import/router.py`
- Create: `backend/app/order_import/chain_service.py`
- Create: `backend/alembic/versions/20260616_01_add_order_import_chain_fields.py`
- Create: `backend/tests/test_order_import_operational_chain.py`
- Modify: `backend/seed.py`

Frontend:

- Modify: `frontend/src/api/modules/orderImport.ts`
- Modify: `frontend/src/views/order/OrderImport.vue`

Docs:

- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`
- Modify: `docs/PERMISSIONS_AND_AUDIT.md`

## Model Changes

Add to `OrderImportBatch`:

```python
chain_status = Column(String(50), server_default="chain_pending", comment="chain_pending/chain_processed/chain_failed")
ledger_rebuilt_count = Column(Integer, default=0, comment="已重建账本订单数")
exception_generated_count = Column(Integer, default=0, comment="生成异常数")
chain_failure_count = Column(Integer, default=0, comment="链路处理失败数")
processed_at = Column(DateTime(timezone=True), comment="链路处理时间")
```

Add to `OrderImportItem`:

```python
chain_status = Column(String(50), server_default="chain_pending", comment="chain_pending/ledger_rebuilt/exception_generated/chain_failed")
chain_failure_reason = Column(String(500), comment="链路处理失败原因")
```

## API

Add:

```text
POST /api/order-imports/{batch_id}/process-chain
GET /api/order-imports/{batch_id}/chain-summary
```

Permissions:

```text
order_import:process
```

Audit:

```text
order_import/process_chain
```

## Task 1: Fix Multi-SKU Import Integrity

**Files:**

- Modify: `backend/app/order_import/service.py`
- Test: `backend/tests/test_order_import_operational_chain.py`

- [ ] **Step 1: Write failing multi-SKU test**

Add this test to `backend/tests/test_order_import_operational_chain.py`:

```python
async def test_same_platform_order_no_creates_one_order_with_two_items(async_client):
    content = (
        "platform,store_name,platform_order_no,order_no,sku_code,quantity,unit_price,currency,recipient_name,recipient_phone,country_code,shipping_address,shipping_fee,paid_at\n"
        "amazon,US,AMZ-1001,,SKU-A,1,20,CNY,Alice,123,US,Street 1,5,2026-06-16\n"
        "amazon,US,AMZ-1001,,SKU-B,2,15,CNY,Alice,123,US,Street 1,5,2026-06-16\n"
    ).encode("utf-8")

    response = await async_client.post(
        "/api/order-imports/csv",
        files={"file": ("orders.csv", content, "text/csv")},
    )
    assert response.status_code == 200, response.text
    batch = response.json()["data"]

    items_response = await async_client.get(f"/api/order-imports/{batch['id']}/items")
    assert items_response.status_code == 200, items_response.text
    rows = items_response.json()["data"]
    assert len({row["order_id"] for row in rows}) == 1

    order_id = rows[0]["order_id"]
    order_response = await async_client.get(f"/api/orders/{order_id}")
    assert order_response.status_code == 200, order_response.text
    order = order_response.json()["data"]
    assert len(order["items"]) == 2
    assert order["total_amount"] == 50.0
```

- [ ] **Step 2: Run the failing test**

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest tests/test_order_import_operational_chain.py::test_same_platform_order_no_creates_one_order_with_two_items -q
```

Expected result:

```text
The test fails if the second SKU row is not added as an OrderItem.
```

- [ ] **Step 3: Update import service**

Change `OrderImportService.process_batch()` so rows are grouped by `platform_order_no` before order creation.

Required behavior:

- Validate every SKU row.
- For a new `platform_order_no`, create one `Order`.
- Add all valid rows for that `platform_order_no` as `OrderItem` rows.
- Link every import row to the same `order_id`.
- If an existing imported order is found, mark rows as `skipped_duplicate`.
- If any row in a group has invalid SKU, mark that row `failed` and still create the order with valid rows only.

- [ ] **Step 4: Re-run test**

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest tests/test_order_import_operational_chain.py::test_same_platform_order_no_creates_one_order_with_two_items -q
```

Expected:

```text
1 passed
```

- [ ] **Step 5: Commit**

Run:

```bash
git add backend/app/order_import/service.py backend/tests/test_order_import_operational_chain.py
git commit -m "fix: create multi-item orders from imported rows"
```

## Task 2: Add Chain Status Fields

**Files:**

- Modify: `backend/app/order_import/models.py`
- Modify: `backend/app/order_import/schemas.py`
- Create: `backend/alembic/versions/20260616_01_add_order_import_chain_fields.py`
- Test: `backend/tests/test_order_import_operational_chain.py`

- [ ] **Step 1: Add schema assertions**

Add test:

```python
async def test_import_batch_exposes_chain_status(async_client):
    content = (
        "platform,store_name,platform_order_no,sku_code,quantity,unit_price,recipient_name\n"
        "amazon,US,AMZ-2001,SKU-A,1,20,Alice\n"
    ).encode("utf-8")

    response = await async_client.post(
        "/api/order-imports/csv",
        files={"file": ("orders.csv", content, "text/csv")},
    )
    assert response.status_code == 200, response.text
    batch = response.json()["data"]
    assert batch["chain_status"] == "chain_pending"
    assert batch["ledger_rebuilt_count"] == 0
    assert batch["exception_generated_count"] == 0
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest tests/test_order_import_operational_chain.py::test_import_batch_exposes_chain_status -q
```

Expected:

```text
The test fails because chain fields are not exposed yet.
```

- [ ] **Step 3: Add model and schema fields**

Add fields listed in "Model Changes" to:

- `backend/app/order_import/models.py`
- `backend/app/order_import/schemas.py`
- `_batch_to_vo()` and `_item_to_vo()` in `backend/app/order_import/router.py`

- [ ] **Step 4: Add migration**

Create `backend/alembic/versions/20260616_01_add_order_import_chain_fields.py` with columns:

```python
op.add_column("order_import_batch", sa.Column("chain_status", sa.String(length=50), server_default="chain_pending"))
op.add_column("order_import_batch", sa.Column("ledger_rebuilt_count", sa.Integer(), server_default="0"))
op.add_column("order_import_batch", sa.Column("exception_generated_count", sa.Integer(), server_default="0"))
op.add_column("order_import_batch", sa.Column("chain_failure_count", sa.Integer(), server_default="0"))
op.add_column("order_import_batch", sa.Column("processed_at", sa.DateTime(timezone=True), nullable=True))
op.add_column("order_import_item", sa.Column("chain_status", sa.String(length=50), server_default="chain_pending"))
op.add_column("order_import_item", sa.Column("chain_failure_reason", sa.String(length=500), nullable=True))
```

- [ ] **Step 5: Re-run test**

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest tests/test_order_import_operational_chain.py::test_import_batch_exposes_chain_status -q
```

Expected:

```text
1 passed
```

- [ ] **Step 6: Commit**

Run:

```bash
git add backend/app/order_import/models.py backend/app/order_import/schemas.py backend/app/order_import/router.py backend/alembic/versions/20260616_01_add_order_import_chain_fields.py backend/tests/test_order_import_operational_chain.py
git commit -m "feat: add order import chain status fields"
```

## Task 3: Add Chain Processing Service

**Files:**

- Create: `backend/app/order_import/chain_service.py`
- Modify: `backend/app/order_import/router.py`
- Modify: `backend/seed.py`
- Test: `backend/tests/test_order_import_operational_chain.py`

- [ ] **Step 1: Write chain processing test**

Add test:

```python
async def test_process_chain_rebuilds_ledger_and_generates_exceptions(async_client):
    content = (
        "platform,store_name,platform_order_no,sku_code,quantity,unit_price,recipient_name,shipping_fee\n"
        "amazon,US,AMZ-3001,SKU-A,1,20,Alice,50\n"
    ).encode("utf-8")

    import_response = await async_client.post(
        "/api/order-imports/csv",
        files={"file": ("orders.csv", content, "text/csv")},
    )
    assert import_response.status_code == 200, import_response.text
    batch_id = import_response.json()["data"]["id"]

    process_response = await async_client.post(f"/api/order-imports/{batch_id}/process-chain")
    assert process_response.status_code == 200, process_response.text
    summary = process_response.json()["data"]
    assert summary["processed_order_count"] == 1
    assert summary["ledger_rebuilt_count"] == 1
    assert "exception_generated_count" in summary

    batch_response = await async_client.get(f"/api/order-imports/{batch_id}")
    assert batch_response.status_code == 200, batch_response.text
    batch = batch_response.json()["data"]
    assert batch["chain_status"] == "chain_processed"
    assert batch["ledger_rebuilt_count"] == 1
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest tests/test_order_import_operational_chain.py::test_process_chain_rebuilds_ledger_and_generates_exceptions -q
```

Expected:

```text
The test fails because process-chain endpoint does not exist.
```

- [ ] **Step 3: Implement `OrderImportChainService`**

Create `backend/app/order_import/chain_service.py`.

The service must:

- Load the batch.
- Collect distinct `order_id` from rows with statuses `created_order`, `imported`, or `skipped_duplicate`.
- Call `LedgerService.rebuild(db, order_id, operator=operator)` for each distinct order.
- Call `ExceptionService.generate(db, operator=operator)` once after rebuilding ledgers.
- Update batch counters.
- Update item `chain_status`.
- Write audit log `order_import/process_chain`.
- Commit through the router, not inside the service.

- [ ] **Step 4: Add router endpoints**

Add to `backend/app/order_import/router.py`:

```text
POST /order-imports/{batch_id}/process-chain
GET /order-imports/{batch_id}/chain-summary
```

Use permission:

```text
order_import:process
```

- [ ] **Step 5: Add seed permission**

Add to `backend/seed.py`:

```python
{"code": "order_import:process", "name": "处理订单导入链路", "module": "order_import"}
```

- [ ] **Step 6: Re-run tests**

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest tests/test_order_import_operational_chain.py -q
```

Expected:

```text
All Stage 12 tests pass.
```

- [ ] **Step 7: Commit**

Run:

```bash
git add backend/app/order_import/chain_service.py backend/app/order_import/router.py backend/seed.py backend/tests/test_order_import_operational_chain.py
git commit -m "feat: process imported orders through ledger and exceptions"
```

## Task 4: Update Frontend Import Page

**Files:**

- Modify: `frontend/src/api/modules/orderImport.ts`
- Modify: `frontend/src/views/order/OrderImport.vue`

- [ ] **Step 1: Add API functions**

Add:

```ts
export function processOrderImportChain(batchId: number) {
  return http.post(`/order-imports/${batchId}/process-chain`)
}

export function getOrderImportChainSummary(batchId: number) {
  return http.get(`/order-imports/${batchId}/chain-summary`)
}
```

- [ ] **Step 2: Add UI actions**

In `OrderImport.vue`, add:

- A `处理经营链路` button on selected batch.
- Batch columns for `chain_status`, `ledger_rebuilt_count`, `exception_generated_count`, `chain_failure_count`.
- Item column for `chain_status`.

Use existing Naive UI patterns in the file.

- [ ] **Step 3: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
Build passes.
```

- [ ] **Step 4: Commit**

Run:

```bash
git add frontend/src/api/modules/orderImport.ts frontend/src/views/order/OrderImport.vue
git commit -m "feat: add order import chain processing UI"
```

## Task 5: Docs And Verification

**Files:**

- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`
- Modify: `docs/PERMISSIONS_AND_AUDIT.md`

- [ ] **Step 1: Update docs**

Document:

- Stage 12 completed capability.
- New endpoints.
- New permission `order_import:process`.
- Audit action `order_import/process_chain`.
- Remaining limits: no real platform API, no background job, no automatic shipping quote binding.

- [ ] **Step 2: Run focused backend tests**

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest tests/test_order_import_operational_chain.py tests/test_order_import_csv_adapter.py tests/test_order_profit_ledger.py tests/test_exception_workbench.py -q
```

Expected:

```text
All focused tests pass.
```

- [ ] **Step 3: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
Build passes.
```

- [ ] **Step 4: Try backend full test suite**

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
```

Expected:

```text
Report the real result. If unrelated baseline failures appear, list them separately from Stage 12.
```

- [ ] **Step 5: Commit docs**

Run:

```bash
git add docs/PROJECT_STATUS.md docs/ROADMAP.md docs/PERMISSIONS_AND_AUDIT.md
git commit -m "docs: document order import operational chain"
```

## Stage 13 Preview: First Real Platform Decision

Do not execute Stage 13 until Stage 12 is complete.

Stage 13 should only choose a real platform if the user can provide at least one of:

- API sandbox credentials.
- Real platform export files.
- Public API documentation and a test store.
- Settlement export samples.

If none are available, Stage 13 should be:

```text
Real Platform Data Pack: collect sample exports and adapter requirements
```

Preferred platform selection rule:

```text
available data and credentials > business priority > platform fame
```

## Agent Prompt

Use this prompt for the next implementation agent:

```text
你接手 /Users/lc/multisell 项目。

任务：执行 docs/superpowers/plans/2026-06-16-order-import-operational-chain.md。

目标：
让 CSV 导入订单进入完整经营链路：正确创建多 SKU 订单、重建利润账本、生成异常，并在订单导入页面展示处理状态。

开始前：
1. git status --short
2. 如果看到 frontend/package.json、pnpm-lock.yaml、pnpm-workspace.yaml、.qoder/，不要自动 stage，先确认它们是否属于用户工具配置。
3. 阅读：
   - backend/app/order_import/
   - backend/app/order/service.py
   - backend/app/finance/ledger_service.py
   - backend/app/exceptions/service.py
   - frontend/src/views/order/OrderImport.vue
   - docs/PERMISSIONS_AND_AUDIT.md

必须做：
1. 修正同一 platform_order_no 多 SKU 导入，确保一个订单多个 OrderItem。
2. 添加 order import batch/item chain 状态字段和 migration。
3. 新增 process-chain endpoint。
4. process-chain 对导入订单执行 ledger rebuild 和 exception generate。
5. 新增 order_import:process 权限。
6. 新增 order_import/process_chain 审计。
7. 前端订单导入页展示 chain 状态并提供处理按钮。
8. 更新 PROJECT_STATUS.md、ROADMAP.md、PERMISSIONS_AND_AUDIT.md。

不要做：
- 真实平台 API
- OAuth
- 自动后台任务
- BI 新功能
- AI 自动处理
- 物流 API
- 平台结算新逻辑

完成后运行：
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest tests/test_order_import_operational_chain.py tests/test_order_import_csv_adapter.py tests/test_order_profit_ledger.py tests/test_exception_workbench.py -q
cd frontend && npm run build

如果可以，再跑：
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q

交付：
1. 当前分支和 commit hash
2. 是否还有未提交文件
3. 多 SKU 导入修正结果
4. 新增 API
5. chain 状态流
6. ledger rebuild / exception generate 串联方式
7. 权限和审计
8. 测试结果
9. 剩余限制
```
