# Order Inventory Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make orders and inventory form a reliable closed loop: order creation locks stock, payment confirms stock deduction, cancellation releases stock, and insufficient stock blocks unsafe state changes.

**Architecture:** Extend the existing `Inventory` model with `locked_quantity`, keep `quantity` as on-hand physical stock, and derive available stock as `quantity - locked_quantity`. Add focused inventory service methods for locking, confirming, and releasing stock, then call them from `OrderService.create` and `OrderService.update_status`. Inventory logs must record every stock movement with order number context.

**Tech Stack:** FastAPI, SQLAlchemy async ORM, Alembic, Pydantic, pytest/httpx, PostgreSQL row locks.

---

## Business Rules

Use this rule set unless the product owner explicitly changes it:

1. **Order creation locks stock**
   - When an order is created in `pending`, each SKU quantity is reserved by increasing `Inventory.locked_quantity`.
   - Physical `Inventory.quantity` does not decrease yet.

2. **Payment confirms stock deduction**
   - When order status changes `pending -> paid`, each SKU quantity is deducted from `Inventory.quantity`.
   - The same quantity is released from `Inventory.locked_quantity`.

3. **Cancellation releases stock**
   - `pending -> cancelled`: decrease `Inventory.locked_quantity`, leave `Inventory.quantity` unchanged.
   - `paid -> cancelled`: do **not** restore physical stock in this phase. Paid cancellation is treated as after-payment cancellation requiring manual refund/return workflow later.

4. **Stock availability**
   - `available_quantity = quantity - locked_quantity`.
   - Order creation must fail if any SKU has insufficient available stock.
   - Payment must fail if locked stock is missing or inconsistent.

5. **Idempotency**
   - Repeating the same status update must not double deduct or double release stock.
   - Existing status transition validation already blocks repeated transitions; keep it.

6. **Auditability**
   - Every lock, deduction, release, and failed stock operation must be covered by tests.
   - Every successful stock movement writes `InventoryLog`.

---

## Current Code Context

Read these files before editing:

- `backend/app/models.py`
- `backend/app/inventory/service.py`
- `backend/app/inventory/router.py`
- `backend/app/inventory/schemas.py`
- `backend/app/order/service.py`
- `backend/app/order/router.py`
- `backend/app/order/schemas.py`
- `backend/tests/test_order.py`
- `backend/tests/test_order_auth_audit.py`
- `backend/tests/test_inventory_price_auth_audit.py`

Current state:

- `Inventory.quantity` exists.
- `InventoryLog` exists.
- `OrderService.create()` creates order/items but does not touch inventory.
- `OrderService.update_status()` changes status but does not touch inventory.
- `InventoryService.check_stock()` checks only `quantity`, not locked stock.

---

## Files To Modify

- Modify: `backend/app/models.py`
  - Add `Inventory.locked_quantity`.

- Create: `backend/alembic/versions/20260615_02_add_inventory_locked_quantity.py`
  - Add `locked_quantity` with default `0`.

- Modify: `backend/app/inventory/schemas.py`
  - Add `locked_quantity` and `available_quantity` to `InventoryVO`.

- Modify: `backend/app/inventory/service.py`
  - Add row-locking stock methods.
  - Update stock check to use available stock.

- Modify: `backend/app/inventory/router.py`
  - Return locked and available stock.

- Modify: `backend/app/order/service.py`
  - Lock stock on create.
  - Confirm deduction on `pending -> paid`.
  - Release lock on `pending -> cancelled`.

- Modify: `backend/app/order/schemas.py`
  - Add inventory movement response fields if needed.

- Create: `backend/tests/test_order_inventory_closure.py`
  - Main behavior tests for lock/deduct/release/insufficient stock.

- Modify: `backend/tests/test_order.py`
  - Adjust helper setup so orders have inventory before creation.

- Modify: `docs/PROJECT_STATUS.md`
  - Mark order inventory closure after implementation.

- Modify: `docs/ROADMAP.md`
  - Update Phase 3 status after implementation.

---

## Task 1: Add Failing Order/Inventory Tests

**Files:**
- Create: `backend/tests/test_order_inventory_closure.py`
- Modify if needed: `backend/tests/test_order.py`

- [ ] **Step 1: Create test helpers**

Create `backend/tests/test_order_inventory_closure.py`:

```python
"""订单库存闭环测试。"""

from uuid import uuid4

import pytest

from app.config import settings


pytestmark = [pytest.mark.asyncio]


@pytest.fixture(autouse=True)
def _auth_disabled():
    original = settings.AUTH_ENABLED
    settings.AUTH_ENABLED = False
    yield
    settings.AUTH_ENABLED = original


async def _create_sku(async_client) -> int:
    resp = await async_client.post("/api/products", json={
        "name": f"库存闭环商品-{uuid4().hex[:8]}",
        "unit": "件",
        "status": 1,
        "package_length_cm": 10,
        "package_width_cm": 10,
        "package_height_cm": 10,
        "package_weight_kg": 0.5,
        "cargo_type": "normal",
    })
    assert resp.status_code == 200, resp.text
    product_id = resp.json()["data"]["id"]

    resp = await async_client.post(
        f"/api/products/{product_id}/specs",
        json={"specs": [{"name": "颜色", "values": ["标准"]}]},
    )
    assert resp.status_code == 200, resp.text

    resp = await async_client.post(f"/api/products/{product_id}/skus/generate")
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]["skus"][0]["id"]


async def _set_inventory(async_client, sku_id: int, quantity: int):
    resp = await async_client.put(
        f"/api/inventory/{sku_id}",
        json={"quantity": quantity, "warehouse": "默认仓库"},
    )
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]


async def _get_inventory(async_client, sku_id: int):
    resp = await async_client.get(f"/api/inventory/{sku_id}")
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]


async def _create_order(async_client, sku_id: int, quantity: int = 2):
    resp = await async_client.post("/api/orders", json={
        "recipient_name": "库存闭环测试",
        "recipient_phone": "13900000000",
        "shipping_address": "测试地址",
        "payment_method": "mock",
        "items": [{"sku_id": sku_id, "quantity": quantity, "unit_price": 100}],
    })
    return resp
```

- [ ] **Step 2: Add stock lock test**

```python
async def test_create_order_locks_available_stock(async_client):
    sku_id = await _create_sku(async_client)
    await _set_inventory(async_client, sku_id, 10)

    resp = await _create_order(async_client, sku_id, quantity=3)

    assert resp.status_code == 200, resp.text
    assert resp.json()["code"] == 200
    inv = await _get_inventory(async_client, sku_id)
    assert inv["quantity"] == 10
    assert inv["locked_quantity"] == 3
    assert inv["available_quantity"] == 7
```

- [ ] **Step 3: Add insufficient available stock test**

```python
async def test_create_order_blocks_when_available_stock_is_insufficient(async_client):
    sku_id = await _create_sku(async_client)
    await _set_inventory(async_client, sku_id, 2)

    resp = await _create_order(async_client, sku_id, quantity=3)

    assert resp.status_code == 200
    body = resp.json()
    assert body["code"] == 400
    assert "库存不足" in body["message"]
    inv = await _get_inventory(async_client, sku_id)
    assert inv["quantity"] == 2
    assert inv["locked_quantity"] == 0
    assert inv["available_quantity"] == 2
```

- [ ] **Step 4: Add payment deduction test**

```python
async def test_paid_status_deducts_locked_stock(async_client):
    sku_id = await _create_sku(async_client)
    await _set_inventory(async_client, sku_id, 10)
    order_resp = await _create_order(async_client, sku_id, quantity=3)
    order_id = order_resp.json()["data"]["id"]

    resp = await async_client.put(f"/api/orders/{order_id}/status", json={"status": "paid"})

    assert resp.status_code == 200, resp.text
    inv = await _get_inventory(async_client, sku_id)
    assert inv["quantity"] == 7
    assert inv["locked_quantity"] == 0
    assert inv["available_quantity"] == 7
```

- [ ] **Step 5: Add pending cancellation release test**

```python
async def test_pending_cancel_releases_locked_stock(async_client):
    sku_id = await _create_sku(async_client)
    await _set_inventory(async_client, sku_id, 10)
    order_resp = await _create_order(async_client, sku_id, quantity=3)
    order_id = order_resp.json()["data"]["id"]

    resp = await async_client.put(f"/api/orders/{order_id}/status", json={"status": "cancelled"})

    assert resp.status_code == 200, resp.text
    inv = await _get_inventory(async_client, sku_id)
    assert inv["quantity"] == 10
    assert inv["locked_quantity"] == 0
    assert inv["available_quantity"] == 10
```

- [ ] **Step 6: Add inventory log test**

```python
async def test_order_stock_movements_write_inventory_logs(async_client):
    sku_id = await _create_sku(async_client)
    await _set_inventory(async_client, sku_id, 10)
    order_resp = await _create_order(async_client, sku_id, quantity=2)
    order_no = order_resp.json()["data"]["order_no"]
    order_id = order_resp.json()["data"]["id"]
    await async_client.put(f"/api/orders/{order_id}/status", json={"status": "paid"})

    logs_resp = await async_client.get(f"/api/inventory/{sku_id}/logs")

    assert logs_resp.status_code == 200
    logs = logs_resp.json()["data"]
    change_types = [log["change_type"] for log in logs]
    assert "lock" in change_types
    assert "deduct" in change_types
    assert any(order_no in (log["remark"] or "") for log in logs)
```

- [ ] **Step 7: Verify tests fail**

Run:

```bash
cd backend && python3 -m pytest tests/test_order_inventory_closure.py -q
```

Expected: tests fail because `locked_quantity` and stock movement behavior do not exist yet.

---

## Task 2: Add Inventory Lock Field And Response Contract

**Files:**
- Modify: `backend/app/models.py`
- Create: `backend/alembic/versions/20260615_02_add_inventory_locked_quantity.py`
- Modify: `backend/app/inventory/schemas.py`
- Modify: `backend/app/inventory/router.py`

- [ ] **Step 1: Add model field**

In `backend/app/models.py`, add to `Inventory`:

```python
locked_quantity = Column(Integer, default=0, nullable=False, comment="锁定库存")
```

- [ ] **Step 2: Add migration**

Create `backend/alembic/versions/20260615_02_add_inventory_locked_quantity.py`:

```python
"""add inventory locked quantity

Revision ID: 20260615_02
Revises: 20260615_01
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa


revision = "20260615_02"
down_revision = "20260615_01"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column(
        "inventory",
        sa.Column("locked_quantity", sa.Integer(), server_default="0", nullable=False, comment="锁定库存"),
    )


def downgrade() -> None:
    op.drop_column("inventory", "locked_quantity")
```

- [ ] **Step 3: Extend schema**

In `backend/app/inventory/schemas.py`, extend `InventoryVO`:

```python
locked_quantity: int = 0
available_quantity: int = 0
```

- [ ] **Step 4: Extend router serializer**

In `backend/app/inventory/router.py`, update `inv_to_vo`:

```python
def inv_to_vo(inv) -> InventoryVO:
    locked = inv.locked_quantity or 0
    quantity = inv.quantity or 0
    return InventoryVO(
        id=inv.id,
        sku_id=inv.sku_id,
        warehouse=inv.warehouse,
        location=inv.location,
        quantity=quantity,
        locked_quantity=locked,
        available_quantity=quantity - locked,
        safety_stock=inv.safety_stock,
        created_at=inv.created_at,
        updated_at=inv.updated_at,
    )
```

For missing inventory, return:

```python
return Result.ok(InventoryVO(sku_id=sku_id, quantity=0, locked_quantity=0, available_quantity=0))
```

- [ ] **Step 5: Run migration and tests**

Run:

```bash
cd backend && python3 -m alembic upgrade head
cd backend && python3 -m pytest tests/test_order_inventory_closure.py::test_create_order_locks_available_stock -q
```

Expected: migration passes; behavior test still fails because service logic is missing.

---

## Task 3: Implement Inventory Stock Movement Service

**Files:**
- Modify: `backend/app/inventory/service.py`

- [ ] **Step 1: Add row-lock helper**

In `InventoryService`, add:

```python
    @staticmethod
    async def _get_inventory_for_update(db: AsyncSession, sku_id: int) -> Optional[Inventory]:
        stmt = select(Inventory).where(Inventory.sku_id == sku_id).with_for_update()
        result = await db.execute(stmt)
        return result.scalar_one_or_none()
```

- [ ] **Step 2: Update stock check**

Replace `check_stock` with available-stock logic:

```python
    @staticmethod
    async def check_stock(db: AsyncSession, sku_id: int, quantity: int) -> tuple[bool, str]:
        inv = await InventoryService.get_inventory(db, sku_id)
        if not inv:
            return False, "库存记录不存在"
        available = (inv.quantity or 0) - (inv.locked_quantity or 0)
        if available < quantity:
            return False, f"库存不足，可用库存: {available}，需要: {quantity}"
        return True, "库存充足"
```

- [ ] **Step 3: Add log helper**

```python
    @staticmethod
    async def _add_log(
        db: AsyncSession,
        sku_id: int,
        change_type: str,
        change_qty: int,
        before_qty: int,
        after_qty: int,
        remark: str,
        operator: str,
    ) -> None:
        db.add(InventoryLog(
            sku_id=sku_id,
            change_type=change_type,
            change_qty=change_qty,
            before_qty=before_qty,
            after_qty=after_qty,
            remark=remark,
            operator=operator,
        ))
        await db.flush()
```

- [ ] **Step 4: Add lock stock**

```python
    @staticmethod
    async def lock_stock(
        db: AsyncSession,
        sku_id: int,
        quantity: int,
        order_no: str,
        operator: str = "system",
    ) -> Inventory:
        inv = await InventoryService._get_inventory_for_update(db, sku_id)
        if not inv:
            raise ValueError("库存记录不存在")
        available = (inv.quantity or 0) - (inv.locked_quantity or 0)
        if available < quantity:
            raise ValueError(f"库存不足，可用库存: {available}，需要: {quantity}")
        before_locked = inv.locked_quantity or 0
        inv.locked_quantity = before_locked + quantity
        await db.flush()
        await InventoryService._add_log(
            db, sku_id, "lock", quantity, before_locked, inv.locked_quantity,
            f"订单锁定库存: {order_no}", operator,
        )
        return inv
```

- [ ] **Step 5: Add release stock**

```python
    @staticmethod
    async def release_locked_stock(
        db: AsyncSession,
        sku_id: int,
        quantity: int,
        order_no: str,
        operator: str = "system",
    ) -> Inventory:
        inv = await InventoryService._get_inventory_for_update(db, sku_id)
        if not inv:
            raise ValueError("库存记录不存在")
        before_locked = inv.locked_quantity or 0
        if before_locked < quantity:
            raise ValueError(f"锁定库存不足，当前锁定: {before_locked}，需要释放: {quantity}")
        inv.locked_quantity = before_locked - quantity
        await db.flush()
        await InventoryService._add_log(
            db, sku_id, "release", -quantity, before_locked, inv.locked_quantity,
            f"订单释放锁定库存: {order_no}", operator,
        )
        return inv
```

- [ ] **Step 6: Add confirm deduction**

```python
    @staticmethod
    async def confirm_locked_stock_deduction(
        db: AsyncSession,
        sku_id: int,
        quantity: int,
        order_no: str,
        operator: str = "system",
    ) -> Inventory:
        inv = await InventoryService._get_inventory_for_update(db, sku_id)
        if not inv:
            raise ValueError("库存记录不存在")
        before_qty = inv.quantity or 0
        before_locked = inv.locked_quantity or 0
        if before_locked < quantity:
            raise ValueError(f"锁定库存不足，当前锁定: {before_locked}，需要扣减: {quantity}")
        if before_qty < quantity:
            raise ValueError(f"库存不足，当前库存: {before_qty}，需要扣减: {quantity}")
        inv.quantity = before_qty - quantity
        inv.locked_quantity = before_locked - quantity
        await db.flush()
        await InventoryService._add_log(
            db, sku_id, "deduct", -quantity, before_qty, inv.quantity,
            f"订单支付扣减库存: {order_no}", operator,
        )
        return inv
```

- [ ] **Step 7: Update manual inventory set behavior**

In `update_inventory`, preserve or normalize locked stock:

```python
if inv:
    before_qty = inv.quantity
    inv.quantity = quantity
    if (inv.locked_quantity or 0) > quantity:
        inv.locked_quantity = quantity
```

This prevents available stock from becoming negative after manual adjustments.

- [ ] **Step 8: Run inventory-focused tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_order_inventory_closure.py::test_create_order_locks_available_stock tests/test_order_inventory_closure.py::test_create_order_blocks_when_available_stock_is_insufficient -q
```

Expected: still fails until order service calls the new methods.

---

## Task 4: Wire Inventory Into Order Creation And Status Flow

**Files:**
- Modify: `backend/app/order/service.py`

- [ ] **Step 1: Import inventory service**

Add:

```python
from app.inventory.service import InventoryService
```

- [ ] **Step 2: Lock stock during order creation**

In `OrderService.create`, after all `OrderItem` rows are built and before the status log is created:

```python
        try:
            for item in items:
                await InventoryService.lock_stock(
                    db,
                    sku_id=item.sku_id,
                    quantity=item.quantity,
                    order_no=order.order_no,
                    operator="system",
                )
        except ValueError as e:
            raise ValueError(str(e))
```

Important: this must happen inside the same transaction as order creation. If locking fails, the whole order creation rolls back.

- [ ] **Step 3: Confirm deduction on paid**

In `OrderService.update_status`, after status transition validation and before writing status log:

```python
        items = await OrderService._get_items(db, order.id)
        if old_status == "pending" and status == "paid":
            for item in items:
                await InventoryService.confirm_locked_stock_deduction(
                    db,
                    sku_id=item.sku_id,
                    quantity=item.quantity,
                    order_no=order.order_no,
                    operator=operator,
                )
```

- [ ] **Step 4: Release lock on pending cancellation**

In the same method:

```python
        if old_status == "pending" and status == "cancelled":
            for item in items:
                await InventoryService.release_locked_stock(
                    db,
                    sku_id=item.sku_id,
                    quantity=item.quantity,
                    order_no=order.order_no,
                    operator=operator,
                )
```

- [ ] **Step 5: Leave paid cancellation unchanged**

Do not restore stock for `paid -> cancelled` in this phase. Add a code comment:

```python
        # paid -> cancelled does not restore physical stock in this phase.
        # Returns/refunds need a separate after-sale workflow.
```

- [ ] **Step 6: Ensure router returns business 400**

`backend/app/order/router.py` should already catch `ValueError` on create/update. If create does not catch it, add:

```python
    try:
        order = await OrderService.create(db, data)
    except ValueError as e:
        return Result.bad_request(str(e))
```

- [ ] **Step 7: Run order inventory tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_order_inventory_closure.py -q
```

Expected: all tests pass.

---

## Task 5: Update Existing Order Tests And API Expectations

**Files:**
- Modify: `backend/tests/test_order.py`
- Modify: `backend/tests/test_order_auth_audit.py` if needed.

- [ ] **Step 1: Update existing order helper setup**

Any existing test that creates orders must ensure inventory exists first.

Pattern:

```python
await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 20})
```

Do this before `POST /api/orders`.

- [ ] **Step 2: Preserve existing order behavior assertions**

Existing assertions for:

- `order_no`
- `status`
- `total_amount`
- `shipping_fee`
- `pay_amount`
- `items`

must keep passing. Do not weaken them.

- [ ] **Step 3: Add status regression test**

Add to `backend/tests/test_order.py`:

```python
async def test_order_status_flow_updates_inventory(async_client):
    product, sku = await _create_product_with_sku(async_client)
    await async_client.put(f"/api/inventory/{sku['id']}", json={"quantity": 5})
    order = await _create_order(async_client, sku["id"], quantity=2)

    inv_resp = await async_client.get(f"/api/inventory/{sku['id']}")
    assert inv_resp.json()["data"]["locked_quantity"] == 2

    await async_client.put(f"/api/orders/{order['id']}/status", json={"status": "paid"})
    inv_resp = await async_client.get(f"/api/inventory/{sku['id']}")
    assert inv_resp.json()["data"]["quantity"] == 3
    assert inv_resp.json()["data"]["locked_quantity"] == 0
```

- [ ] **Step 4: Run order suites**

Run:

```bash
cd backend && python3 -m pytest tests/test_order.py tests/test_order_auth_audit.py tests/test_order_inventory_closure.py -q
```

Expected: all pass.

---

## Task 6: Documentation And Verification

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Update `PROJECT_STATUS.md`**

Add under order module:

```markdown
### 订单库存闭环

状态：已完成。

已实现：
- 创建订单锁定库存。
- 支付订单扣减库存并释放锁定。
- 待支付订单取消释放锁定库存。
- 库存可用量 = 实物库存 - 锁定库存。
- 库存不足时阻止创建订单。
- 库存变动写入 `InventoryLog`。

暂未实现：
- 付款后取消自动退库存。
- 售后退货入库。
- 多仓库分配。
```

- [ ] **Step 2: Update `ROADMAP.md`**

Update Phase 3:

```markdown
状态：订单库存闭环已完成第一版。
```

Mark these complete:

- 订单创建锁库存
- 支付扣减库存
- 取消释放库存
- 库存不足阻止订单
- 库存日志

Keep these as future:

- 并发压力测试
- 售后退货入库
- 多仓库发货分配

- [ ] **Step 3: Full verification**

Run:

```bash
cd backend && python3 -m alembic upgrade head
cd backend && python3 -m pytest tests/test_order_inventory_closure.py tests/test_order.py tests/test_inventory_price_auth_audit.py -q
cd backend && python3 -m pytest -q
cd frontend && npm run build
git diff --check
```

Expected:

- Migration succeeds.
- Focused tests pass.
- Full backend tests pass.
- Frontend build passes.
- `git diff --check` prints no output.

---

## Risks And Guardrails

### Risk: double deduction

Do not deduct stock in both order creation and payment. Creation only locks; payment deducts.

### Risk: negative available stock

Always calculate available as:

```python
available = quantity - locked_quantity
```

Never check `quantity` alone for new orders.

### Risk: partial order creation

If an order has multiple SKUs and one SKU cannot lock stock, the whole order creation must fail and rollback.

### Risk: concurrent oversell

Use `SELECT ... FOR UPDATE` when mutating inventory rows. Do not rely on a read-then-write without row lock.

### Risk: old tests failing

Existing order tests did not need inventory before. Update their setup, not their business assertions.

---

## Handoff Prompt For Another Agent

```text
你接手 MultiSell 的“订单库存闭环”任务。

必须阅读：
- docs/superpowers/plans/2026-06-15-order-inventory-closure.md
- backend/app/order/service.py
- backend/app/order/router.py
- backend/app/inventory/service.py
- backend/app/inventory/router.py
- backend/app/inventory/schemas.py
- backend/app/models.py
- backend/tests/test_order.py

目标：
1. 创建订单时锁定库存。
2. pending -> paid 时扣减库存并释放锁定。
3. pending -> cancelled 时释放锁定库存。
4. paid -> cancelled 暂不自动退库存。
5. 库存不足时阻止创建订单。
6. InventoryVO 返回 quantity、locked_quantity、available_quantity。
7. 所有库存变动写入 InventoryLog。
8. 使用行锁防止并发超卖。

范围限制：
- 不做售后退货。
- 不做多仓库发货分配。
- 不做平台订单导入。
- 不重构整个订单模块。

验收命令：
- cd backend && python3 -m alembic upgrade head
- cd backend && python3 -m pytest tests/test_order_inventory_closure.py tests/test_order.py tests/test_inventory_price_auth_audit.py -q
- cd backend && python3 -m pytest -q
- cd frontend && npm run build
- git diff --check

完成后汇报：
- 改了哪些文件
- 新增了哪些字段/迁移
- 订单状态如何影响库存
- 测试结果
- 剩余下一阶段能力
```

---

## After This Plan: Next Planning Queue

When this phase is complete, plan these in order:

1. **报价版本管理**
   - Add shipping rate version/effective date.
   - Orders keep quote version snapshot.

2. **Excel 模板下载和导入预览**
   - Download standard shipping quote template.
   - Preview rows before import.
   - Export row-level errors.

3. **平台订单接入**
   - Import platform orders.
   - Match SKU.
   - Trigger the same inventory lock/deduct workflow.

4. **报表一期**
   - Order profit dashboard.
   - Inventory risk dashboard.
   - Shipping cost by channel/provider.

5. **真实物流商 API 适配层**
   - Add provider adapter interface.
   - Implement one carrier or aggregator first.
