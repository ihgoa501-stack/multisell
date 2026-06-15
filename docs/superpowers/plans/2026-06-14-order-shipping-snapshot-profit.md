# Order Shipping Snapshot Profit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make orders consume the existing shipping calculator by saving a selected shipping quote snapshot and exposing first-pass order profit fields.

**Architecture:** Keep `/api/shipping/calculate` as the source of quote options, then add an order-level snapshot endpoint that persists one selected quote into a dedicated `sales_order_shipping_snapshot` table. Store denormalized money, weight, provider/channel names, and calculation detail so historical orders do not change when shipping rules change later; keep `sales_order.shipping_fee` updated from the selected snapshot for backward compatibility.

**Tech Stack:** FastAPI, SQLAlchemy async ORM, Alembic, Pydantic, pytest/httpx, Vue 3, Naive UI, Vite.

---

## Current Context

- Existing order backend files:
  - `backend/app/order/router.py`
  - `backend/app/order/service.py`
  - `backend/app/order/schemas.py`
  - `backend/tests/test_order.py`
  - `backend/tests/test_order_auth_audit.py`
- Existing shipping backend files:
  - `backend/app/shipping/router.py`
  - `backend/app/shipping/service.py`
  - `backend/app/shipping/schemas.py`
  - `backend/tests/test_shipping_calculation.py`
  - `backend/tests/test_shipping_management.py`
- Existing frontend order files:
  - `frontend/src/api/modules/order.ts`
  - `frontend/src/views/order/OrderDetail.vue`
  - `frontend/src/views/order/OrderList.vue`
- Current order model has only `sales_order.shipping_fee`; it does not store quote source, chargeable weight, provider/channel, calculation detail, or profit numbers.
- Current shipping calculator returns sorted `results`, each with provider/channel ids and names, actual/volumetric/chargeable weights, base fee, surcharges, total fee, currency, and calculation detail.

## Files

- Modify: `backend/app/models.py`
  - Add `OrderShippingSnapshot`.
  - Add profit columns to `Order`.
- Create: `backend/alembic/versions/20260614_03_add_order_shipping_snapshot_profit.py`
  - Create snapshot table.
  - Add profit columns to `sales_order`.
- Modify: `backend/app/order/schemas.py`
  - Add shipping quote selection schema.
  - Add shipping snapshot and profit response schemas.
  - Add optional fee fields for manual profit inputs.
- Modify: `backend/app/order/service.py`
  - Save selected shipping quote snapshot.
  - Recalculate order `shipping_fee`, `pay_amount`, and profit fields.
  - Include snapshot and profit in `order_to_dict`.
- Modify: `backend/app/order/router.py`
  - Add `POST /api/orders/{order_id}/shipping-quote`.
  - Add `PUT /api/orders/{order_id}/profit-inputs`.
  - Add RBAC and audit logs.
- Create: `backend/tests/test_order_shipping_snapshot_profit.py`
  - Cover snapshot save, immutable history, missing shipping data, and profit math.
- Modify: `backend/tests/test_order_auth_audit.py`
  - Cover order shipping/profit permissions.
- Modify: `frontend/src/api/modules/order.ts`
  - Add `bindShippingQuote` and `updateProfitInputs`.
- Modify: `frontend/src/views/order/OrderDetail.vue`
  - Show shipping snapshot and profit section.
  - Allow calculating options and binding a selected quote.
- Modify: `docs/PROJECT_STATUS.md`
  - Mark order shipping snapshot/profit first pass as complete after implementation.
- Modify: `docs/ROADMAP.md`
  - Update Phase 6 status after implementation.
- Modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`
  - Align `ShippingQuoteSnapshot` section with actual table/API names after implementation.
- Modify: `docs/LOGISTICS_AND_SHIPPING_PRD.md`
  - Mark order snapshot as implemented after implementation.

---

### Task 1: Backend Model, Migration, And Schemas

**Files:**
- Modify: `backend/app/models.py`
- Create: `backend/alembic/versions/20260614_03_add_order_shipping_snapshot_profit.py`
- Modify: `backend/app/order/schemas.py`
- Test: `backend/tests/test_order_shipping_snapshot_profit.py`

- [ ] **Step 1: Write failing schema/model-level API tests**

Create `backend/tests/test_order_shipping_snapshot_profit.py` with this starting structure:

```python
"""订单运费快照与利润计算测试。"""

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


def _uc(prefix: str) -> str:
    return f"{prefix}_{uuid4().hex[:8]}"


async def _seed_product_sku(async_client, *, with_package: bool = True) -> int:
    payload = {"name": f"订单运费商品-{uuid4().hex[:8]}", "unit": "件", "status": 1}
    if with_package:
        payload.update({
            "package_length_cm": 30,
            "package_width_cm": 20,
            "package_height_cm": 10,
            "package_weight_kg": 0.8,
            "cargo_type": "normal",
        })
    resp = await async_client.post("/api/products", json=payload)
    assert resp.status_code == 200, resp.text
    product_id = resp.json()["data"]["id"]
    resp = await async_client.post(
        f"/api/products/{product_id}/specs",
        json={"specs": [{"name": "颜色", "values": ["红"]}]},
    )
    assert resp.status_code == 200, resp.text
    resp = await async_client.post(f"/api/products/{product_id}/skus/generate")
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]["skus"][0]["id"]


async def _seed_shipping_channel(async_client, country: str = "US") -> dict:
    resp = await async_client.post("/api/shipping/providers", json={
        "name": f"订单快照物流商-{uuid4().hex[:6]}",
        "code": _uc("order_provider"),
    })
    assert resp.status_code == 200, resp.text
    provider_id = resp.json()["data"]["id"]
    resp = await async_client.post("/api/shipping/channels", json={
        "provider_id": provider_id,
        "name": f"订单快照渠道-{uuid4().hex[:6]}",
        "code": _uc("order_channel"),
        "volumetric_divisor": 6000,
        "cargo_types": ["normal"],
        "currency": "CNY",
    })
    assert resp.status_code == 200, resp.text
    channel_id = resp.json()["data"]["id"]
    resp = await async_client.post(
        f"/api/shipping/channels/{channel_id}/zones",
        json={"country_code": country},
    )
    assert resp.status_code == 200, resp.text
    resp = await async_client.post(
        f"/api/shipping/channels/{channel_id}/rules",
        json={
            "rule_type": "fixed_plus_per_kg",
            "fixed_fee": 8,
            "per_kg_price": 42,
            "minimum_charge": 25,
            "rounding_increment": 0.1,
        },
    )
    assert resp.status_code == 200, resp.text
    return {"provider_id": provider_id, "channel_id": channel_id}


async def _create_order(async_client, sku_id: int, quantity: int = 1) -> dict:
    resp = await async_client.post("/api/orders", json={
        "recipient_name": "订单运费测试",
        "recipient_phone": "13900000000",
        "shipping_address": "测试地址",
        "payment_method": "mock",
        "items": [{"sku_id": sku_id, "quantity": quantity, "unit_price": 120}],
        "platform_fee": 12,
        "payment_fee": 3,
        "other_fee": 5,
    })
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]
```

Add the first failing test:

```python
async def test_bind_shipping_quote_saves_snapshot_and_profit(async_client):
    sku_id = await _seed_product_sku(async_client)
    await _seed_shipping_channel(async_client, "US")
    order = await _create_order(async_client, sku_id, quantity=1)

    resp = await async_client.post(
        f"/api/orders/{order['id']}/shipping-quote",
        json={
            "sku_id": sku_id,
            "quantity": 1,
            "destination_country": "US",
            "cargo_type": "normal",
            "channel_id": None,
        },
    )

    assert resp.status_code == 200, resp.text
    data = resp.json()["data"]
    assert data["shipping_fee"] == 50.0
    assert data["pay_amount"] == 170.0
    assert data["profit"]["revenue_amount"] == 120.0
    assert data["profit"]["shipping_fee"] == 50.0
    assert data["profit"]["platform_fee"] == 12.0
    assert data["profit"]["payment_fee"] == 3.0
    assert data["profit"]["other_fee"] == 5.0
    assert data["profit"]["profit_amount"] == 50.0
    assert data["profit"]["profit_margin"] == pytest.approx(41.666, rel=0.01)
    assert data["shipping_snapshot"]["provider_name"]
    assert data["shipping_snapshot"]["channel_name"]
    assert data["shipping_snapshot"]["chargeable_weight_kg"] == 1.0
    assert data["shipping_snapshot"]["total_shipping_fee"] == 50.0
```

- [ ] **Step 2: Run the failing test**

Run:

```bash
cd backend && python3 -m pytest tests/test_order_shipping_snapshot_profit.py::test_bind_shipping_quote_saves_snapshot_and_profit -q
```

Expected: fail because `OrderCreate` rejects `platform_fee` / `payment_fee` / `other_fee`, and `/api/orders/{id}/shipping-quote` does not exist.

- [ ] **Step 3: Add model fields**

In `backend/app/models.py`, extend `Order` with first-pass profit input/output fields:

```python
    platform_fee = Column(Numeric(10, 2), default=0, comment="平台佣金/平台费")
    payment_fee = Column(Numeric(10, 2), default=0, comment="支付手续费")
    other_fee = Column(Numeric(10, 2), default=0, comment="其他费用")
    product_cost = Column(Numeric(10, 2), default=0, comment="商品成本")
    profit_amount = Column(Numeric(10, 2), default=0, comment="订单利润")
    profit_margin = Column(Numeric(10, 4), default=0, comment="利润率百分比")
```

Add this class after `OrderStatusLog`:

```python
class OrderShippingSnapshot(Base):
    """订单运费快照"""
    __tablename__ = "sales_order_shipping_snapshot"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    order_id = Column(BigInteger, ForeignKey("sales_order.id"), nullable=False, unique=True, comment="订单ID")
    sku_id = Column(BigInteger, ForeignKey("sku.id"), nullable=False, comment="试算SKU ID")
    quantity = Column(Integer, nullable=False, comment="试算数量")
    destination_country = Column(String(10), nullable=False, comment="目的地国家")
    postal_code = Column(String(20), comment="目的地邮编")
    cargo_type = Column(String(50), default="normal", comment="货品类型")
    package_source = Column(String(20), comment="包装数据来源: product/sku")
    package_length_cm = Column(Numeric(10, 2), nullable=False, comment="包装长cm")
    package_width_cm = Column(Numeric(10, 2), nullable=False, comment="包装宽cm")
    package_height_cm = Column(Numeric(10, 2), nullable=False, comment="包装高cm")
    package_weight_kg = Column(Numeric(10, 3), nullable=False, comment="单件包装重量kg")
    provider_id = Column(BigInteger, ForeignKey("shipping_provider.id"), nullable=False, comment="物流供应商ID")
    provider_name = Column(String(200), nullable=False, comment="物流供应商名称快照")
    channel_id = Column(BigInteger, ForeignKey("shipping_channel.id"), nullable=False, comment="物流渠道ID")
    channel_name = Column(String(200), nullable=False, comment="物流渠道名称快照")
    currency = Column(String(10), default="CNY", comment="币种")
    actual_weight_kg = Column(Numeric(10, 4), nullable=False, comment="实际重量kg")
    volumetric_weight_kg = Column(Numeric(10, 4), nullable=False, comment="体积重量kg")
    chargeable_weight_kg = Column(Numeric(10, 4), nullable=False, comment="计费重量kg")
    base_shipping_fee = Column(Numeric(10, 2), nullable=False, comment="基础运费")
    surcharge_fee = Column(Numeric(10, 2), default=0, comment="固定附加费")
    fuel_surcharge_fee = Column(Numeric(10, 2), default=0, comment="燃油附加费")
    total_shipping_fee = Column(Numeric(10, 2), nullable=False, comment="总运费")
    calculation_detail = Column(Text, comment="计算说明")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")
```

- [ ] **Step 4: Add Alembic migration**

Create `backend/alembic/versions/20260614_03_add_order_shipping_snapshot_profit.py`:

```python
"""add order shipping snapshot and profit fields

Revision ID: 20260614_03
Revises: 20260614_02
Create Date: 2026-06-14
"""

from alembic import op
import sqlalchemy as sa


revision = "20260614_03"
down_revision = "20260614_02"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column("sales_order", sa.Column("platform_fee", sa.Numeric(10, 2), server_default="0", nullable=True, comment="平台佣金/平台费"))
    op.add_column("sales_order", sa.Column("payment_fee", sa.Numeric(10, 2), server_default="0", nullable=True, comment="支付手续费"))
    op.add_column("sales_order", sa.Column("other_fee", sa.Numeric(10, 2), server_default="0", nullable=True, comment="其他费用"))
    op.add_column("sales_order", sa.Column("product_cost", sa.Numeric(10, 2), server_default="0", nullable=True, comment="商品成本"))
    op.add_column("sales_order", sa.Column("profit_amount", sa.Numeric(10, 2), server_default="0", nullable=True, comment="订单利润"))
    op.add_column("sales_order", sa.Column("profit_margin", sa.Numeric(10, 4), server_default="0", nullable=True, comment="利润率百分比"))

    op.create_table(
        "sales_order_shipping_snapshot",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("order_id", sa.BigInteger(), nullable=False, comment="订单ID"),
        sa.Column("sku_id", sa.BigInteger(), nullable=False, comment="试算SKU ID"),
        sa.Column("quantity", sa.Integer(), nullable=False, comment="试算数量"),
        sa.Column("destination_country", sa.String(length=10), nullable=False, comment="目的地国家"),
        sa.Column("postal_code", sa.String(length=20), nullable=True, comment="目的地邮编"),
        sa.Column("cargo_type", sa.String(length=50), server_default="normal", nullable=True, comment="货品类型"),
        sa.Column("package_source", sa.String(length=20), nullable=True, comment="包装数据来源"),
        sa.Column("package_length_cm", sa.Numeric(10, 2), nullable=False, comment="包装长cm"),
        sa.Column("package_width_cm", sa.Numeric(10, 2), nullable=False, comment="包装宽cm"),
        sa.Column("package_height_cm", sa.Numeric(10, 2), nullable=False, comment="包装高cm"),
        sa.Column("package_weight_kg", sa.Numeric(10, 3), nullable=False, comment="单件包装重量kg"),
        sa.Column("provider_id", sa.BigInteger(), nullable=False, comment="物流供应商ID"),
        sa.Column("provider_name", sa.String(length=200), nullable=False, comment="物流供应商名称快照"),
        sa.Column("channel_id", sa.BigInteger(), nullable=False, comment="物流渠道ID"),
        sa.Column("channel_name", sa.String(length=200), nullable=False, comment="物流渠道名称快照"),
        sa.Column("currency", sa.String(length=10), server_default="CNY", nullable=True, comment="币种"),
        sa.Column("actual_weight_kg", sa.Numeric(10, 4), nullable=False, comment="实际重量kg"),
        sa.Column("volumetric_weight_kg", sa.Numeric(10, 4), nullable=False, comment="体积重量kg"),
        sa.Column("chargeable_weight_kg", sa.Numeric(10, 4), nullable=False, comment="计费重量kg"),
        sa.Column("base_shipping_fee", sa.Numeric(10, 2), nullable=False, comment="基础运费"),
        sa.Column("surcharge_fee", sa.Numeric(10, 2), server_default="0", nullable=True, comment="固定附加费"),
        sa.Column("fuel_surcharge_fee", sa.Numeric(10, 2), server_default="0", nullable=True, comment="燃油附加费"),
        sa.Column("total_shipping_fee", sa.Numeric(10, 2), nullable=False, comment="总运费"),
        sa.Column("calculation_detail", sa.Text(), nullable=True, comment="计算说明"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=True, comment="创建时间"),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=True, comment="更新时间"),
        sa.ForeignKeyConstraint(["channel_id"], ["shipping_channel.id"]),
        sa.ForeignKeyConstraint(["order_id"], ["sales_order.id"]),
        sa.ForeignKeyConstraint(["provider_id"], ["shipping_provider.id"]),
        sa.ForeignKeyConstraint(["sku_id"], ["sku.id"]),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint("order_id", name="uq_sales_order_shipping_snapshot_order_id"),
    )
    op.create_index("ix_order_shipping_snapshot_order_id", "sales_order_shipping_snapshot", ["order_id"])


def downgrade() -> None:
    op.drop_index("ix_order_shipping_snapshot_order_id", table_name="sales_order_shipping_snapshot")
    op.drop_table("sales_order_shipping_snapshot")
    op.drop_column("sales_order", "profit_margin")
    op.drop_column("sales_order", "profit_amount")
    op.drop_column("sales_order", "product_cost")
    op.drop_column("sales_order", "other_fee")
    op.drop_column("sales_order", "payment_fee")
    op.drop_column("sales_order", "platform_fee")
```

- [ ] **Step 5: Add order schemas**

In `backend/app/order/schemas.py`, extend `OrderCreate`:

```python
    platform_fee: float = Field(0, ge=0)
    payment_fee: float = Field(0, ge=0)
    other_fee: float = Field(0, ge=0)
    product_cost: float = Field(0, ge=0)
```

Add:

```python
class OrderShippingQuoteBind(BaseModel):
    sku_id: int = Field(..., description="用于试算的 SKU ID")
    quantity: int = Field(1, ge=1, description="试算数量")
    destination_country: str = Field(..., min_length=2, max_length=10)
    postal_code: Optional[str] = Field(None, max_length=20)
    cargo_type: str = Field("normal", max_length=50)
    channel_id: Optional[int] = Field(None, description="为空时选择最低价渠道")


class OrderProfitInputsUpdate(BaseModel):
    platform_fee: Optional[float] = Field(None, ge=0)
    payment_fee: Optional[float] = Field(None, ge=0)
    other_fee: Optional[float] = Field(None, ge=0)
    product_cost: Optional[float] = Field(None, ge=0)


class OrderShippingSnapshotVO(BaseModel):
    id: int
    order_id: int
    sku_id: int
    quantity: int
    destination_country: str
    postal_code: Optional[str] = None
    cargo_type: Optional[str] = None
    package_source: Optional[str] = None
    package_length_cm: float
    package_width_cm: float
    package_height_cm: float
    package_weight_kg: float
    provider_id: int
    provider_name: str
    channel_id: int
    channel_name: str
    currency: str = "CNY"
    actual_weight_kg: float
    volumetric_weight_kg: float
    chargeable_weight_kg: float
    base_shipping_fee: float
    surcharge_fee: float = 0
    fuel_surcharge_fee: float = 0
    total_shipping_fee: float
    calculation_detail: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


class OrderProfitVO(BaseModel):
    revenue_amount: float
    product_cost: float
    shipping_fee: float
    platform_fee: float
    payment_fee: float
    other_fee: float
    profit_amount: float
    profit_margin: float
```

Extend `OrderVO`:

```python
    platform_fee: float = 0
    payment_fee: float = 0
    other_fee: float = 0
    product_cost: float = 0
    profit_amount: float = 0
    profit_margin: float = 0
    profit: Optional[OrderProfitVO] = None
    shipping_snapshot: Optional[OrderShippingSnapshotVO] = None
```

- [ ] **Step 6: Run migration and schema smoke checks**

Run:

```bash
cd backend && python3 -m alembic upgrade head
cd backend && python3 -m pytest tests/test_order_shipping_snapshot_profit.py::test_bind_shipping_quote_saves_snapshot_and_profit -q
```

Expected after this task: migration passes; test still fails because service/router are not implemented.

---

### Task 2: Backend Snapshot Binding And Profit Calculation

**Files:**
- Modify: `backend/app/order/service.py`
- Modify: `backend/app/order/router.py`
- Test: `backend/tests/test_order_shipping_snapshot_profit.py`

- [ ] **Step 1: Add failing behavior tests**

Append these tests to `backend/tests/test_order_shipping_snapshot_profit.py`:

```python
async def test_shipping_snapshot_does_not_change_when_rule_changes(async_client):
    sku_id = await _seed_product_sku(async_client)
    seeded = await _seed_shipping_channel(async_client, "US")
    order = await _create_order(async_client, sku_id, quantity=1)

    bind_resp = await async_client.post(
        f"/api/orders/{order['id']}/shipping-quote",
        json={"sku_id": sku_id, "quantity": 1, "destination_country": "US", "cargo_type": "normal"},
    )
    assert bind_resp.status_code == 200, bind_resp.text
    assert bind_resp.json()["data"]["shipping_snapshot"]["total_shipping_fee"] == 50.0

    rules_resp = await async_client.get(f"/api/shipping/channels/{seeded['channel_id']}/rules")
    rule_id = rules_resp.json()["data"][0]["id"]
    update_resp = await async_client.put(
        f"/api/shipping/rules/{rule_id}",
        json={"fixed_fee": 100, "per_kg_price": 100, "rule_type": "fixed_plus_per_kg"},
    )
    assert update_resp.status_code == 200, update_resp.text

    detail_resp = await async_client.get(f"/api/orders/{order['id']}")
    detail = detail_resp.json()["data"]
    assert detail["shipping_fee"] == 50.0
    assert detail["shipping_snapshot"]["total_shipping_fee"] == 50.0
```

```python
async def test_bind_shipping_quote_requires_complete_package_data(async_client):
    sku_id = await _seed_product_sku(async_client, with_package=False)
    await _seed_shipping_channel(async_client, "US")
    order = await _create_order(async_client, sku_id, quantity=1)

    resp = await async_client.post(
        f"/api/orders/{order['id']}/shipping-quote",
        json={"sku_id": sku_id, "quantity": 1, "destination_country": "US", "cargo_type": "normal"},
    )

    assert resp.status_code == 200
    body = resp.json()
    assert body["code"] == 400
    assert "物流数据不完整" in body["message"]
```

```python
async def test_update_profit_inputs_recalculates_profit(async_client):
    sku_id = await _seed_product_sku(async_client)
    await _seed_shipping_channel(async_client, "US")
    order = await _create_order(async_client, sku_id, quantity=1)
    await async_client.post(
        f"/api/orders/{order['id']}/shipping-quote",
        json={"sku_id": sku_id, "quantity": 1, "destination_country": "US", "cargo_type": "normal"},
    )

    resp = await async_client.put(
        f"/api/orders/{order['id']}/profit-inputs",
        json={"platform_fee": 10, "payment_fee": 2, "other_fee": 1, "product_cost": 30},
    )

    assert resp.status_code == 200, resp.text
    profit = resp.json()["data"]["profit"]
    assert profit["revenue_amount"] == 120.0
    assert profit["product_cost"] == 30.0
    assert profit["shipping_fee"] == 50.0
    assert profit["profit_amount"] == 27.0
    assert profit["profit_margin"] == pytest.approx(22.5, rel=0.01)
```

- [ ] **Step 2: Run the failing tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_order_shipping_snapshot_profit.py -q
```

Expected: all new tests fail until service/router are implemented.

- [ ] **Step 3: Implement service helpers**

In `backend/app/order/service.py`:

Update imports:

```python
from app.models import Order, OrderItem, OrderShippingSnapshot, OrderStatusLog, Price, Product, Sku
from app.order.schemas import OrderCreate, OrderProfitInputsUpdate, OrderShippingQuoteBind
from app.shipping.schemas import CalculateRequest
from app.shipping.service import CalculateService
```

Add helpers near `_money`:

```python
def _decimal(value) -> Decimal:
    return Decimal(str(value or 0))


def _pct(numerator: Decimal, denominator: Decimal) -> Decimal:
    if denominator <= 0:
        return Decimal("0")
    return (numerator / denominator) * Decimal("100")
```

Update `OrderService.create` to set fee fields:

```python
            platform_fee=Decimal(str(data.platform_fee)),
            payment_fee=Decimal(str(data.payment_fee)),
            other_fee=Decimal(str(data.other_fee)),
            product_cost=Decimal(str(data.product_cost)),
```

After `order.pay_amount = ...`, call:

```python
        OrderService._recalculate_profit(order)
```

Add methods to `OrderService`:

```python
    @staticmethod
    def _recalculate_profit(order: Order) -> None:
        revenue = _decimal(order.total_amount)
        costs = (
            _decimal(order.product_cost)
            + _decimal(order.shipping_fee)
            + _decimal(order.platform_fee)
            + _decimal(order.payment_fee)
            + _decimal(order.other_fee)
        )
        profit = revenue - costs
        order.profit_amount = profit
        order.profit_margin = _pct(profit, revenue)
        order.pay_amount = revenue + _decimal(order.shipping_fee)

    @staticmethod
    async def bind_shipping_quote(db: AsyncSession, order_id: int, data: OrderShippingQuoteBind) -> Optional[dict]:
        order = await db.get(Order, order_id)
        if not order:
            return None

        calc = await CalculateService.calculate(
            db,
            CalculateRequest(
                sku_id=data.sku_id,
                quantity=data.quantity,
                destination_country=data.destination_country,
                postal_code=data.postal_code,
                cargo_type=data.cargo_type,
            ),
        )
        if not calc.results:
            raise ValueError("没有可用物流报价")

        selected = None
        if data.channel_id is not None:
            selected = next((item for item in calc.results if item.channel_id == data.channel_id), None)
            if selected is None:
                raise ValueError("指定物流渠道不可用")
        else:
            selected = calc.results[0]

        existing = await OrderService._get_shipping_snapshot_model(db, order_id)
        snapshot = existing or OrderShippingSnapshot(order_id=order_id)
        snapshot.sku_id = data.sku_id
        snapshot.quantity = data.quantity
        snapshot.destination_country = calc.destination_country
        snapshot.postal_code = data.postal_code
        snapshot.cargo_type = data.cargo_type
        snapshot.package_source = calc.package.source
        snapshot.package_length_cm = Decimal(str(calc.package.length_cm))
        snapshot.package_width_cm = Decimal(str(calc.package.width_cm))
        snapshot.package_height_cm = Decimal(str(calc.package.height_cm))
        snapshot.package_weight_kg = Decimal(str(calc.package.weight_kg))
        snapshot.provider_id = selected.provider_id
        snapshot.provider_name = selected.provider_name
        snapshot.channel_id = selected.channel_id
        snapshot.channel_name = selected.channel_name
        snapshot.currency = selected.currency
        snapshot.actual_weight_kg = Decimal(str(selected.actual_weight_kg))
        snapshot.volumetric_weight_kg = Decimal(str(selected.volumetric_weight_kg))
        snapshot.chargeable_weight_kg = Decimal(str(selected.chargeable_weight_kg))
        snapshot.base_shipping_fee = Decimal(str(selected.base_shipping_fee))
        snapshot.surcharge_fee = Decimal(str(selected.surcharge_fee))
        snapshot.fuel_surcharge_fee = Decimal(str(selected.fuel_surcharge_fee))
        snapshot.total_shipping_fee = Decimal(str(selected.total_shipping_fee))
        snapshot.calculation_detail = selected.calculation_detail
        if existing is None:
            db.add(snapshot)

        order.shipping_fee = Decimal(str(selected.total_shipping_fee))
        OrderService._recalculate_profit(order)

        await db.flush()
        await db.refresh(order)
        await db.refresh(snapshot)
        return await OrderService.get_detail(db, order_id)

    @staticmethod
    async def update_profit_inputs(db: AsyncSession, order_id: int, data: OrderProfitInputsUpdate) -> Optional[dict]:
        order = await db.get(Order, order_id)
        if not order:
            return None
        update_data = data.model_dump(exclude_unset=True)
        for field, value in update_data.items():
            setattr(order, field, Decimal(str(value)))
        OrderService._recalculate_profit(order)
        await db.flush()
        await db.refresh(order)
        return await OrderService.get_detail(db, order_id)
```

Add snapshot fetch and serializer helpers:

```python
    @staticmethod
    async def _get_shipping_snapshot_model(db: AsyncSession, order_id: int) -> Optional[OrderShippingSnapshot]:
        result = await db.execute(
            select(OrderShippingSnapshot).where(OrderShippingSnapshot.order_id == order_id)
        )
        return result.scalar_one_or_none()

    @staticmethod
    async def _get_shipping_snapshot(db: AsyncSession, order_id: int) -> Optional[dict]:
        snapshot = await OrderService._get_shipping_snapshot_model(db, order_id)
        return _shipping_snapshot_to_dict(snapshot) if snapshot else None
```

Add module-level serializer:

```python
def _shipping_snapshot_to_dict(snapshot: OrderShippingSnapshot) -> dict:
    return {
        "id": snapshot.id,
        "order_id": snapshot.order_id,
        "sku_id": snapshot.sku_id,
        "quantity": snapshot.quantity,
        "destination_country": snapshot.destination_country,
        "postal_code": snapshot.postal_code,
        "cargo_type": snapshot.cargo_type,
        "package_source": snapshot.package_source,
        "package_length_cm": _money(snapshot.package_length_cm),
        "package_width_cm": _money(snapshot.package_width_cm),
        "package_height_cm": _money(snapshot.package_height_cm),
        "package_weight_kg": float(snapshot.package_weight_kg or 0),
        "provider_id": snapshot.provider_id,
        "provider_name": snapshot.provider_name,
        "channel_id": snapshot.channel_id,
        "channel_name": snapshot.channel_name,
        "currency": snapshot.currency or "CNY",
        "actual_weight_kg": float(snapshot.actual_weight_kg or 0),
        "volumetric_weight_kg": float(snapshot.volumetric_weight_kg or 0),
        "chargeable_weight_kg": float(snapshot.chargeable_weight_kg or 0),
        "base_shipping_fee": _money(snapshot.base_shipping_fee),
        "surcharge_fee": _money(snapshot.surcharge_fee),
        "fuel_surcharge_fee": _money(snapshot.fuel_surcharge_fee),
        "total_shipping_fee": _money(snapshot.total_shipping_fee),
        "calculation_detail": snapshot.calculation_detail,
        "created_at": snapshot.created_at,
        "updated_at": snapshot.updated_at,
    }
```

Update `order_to_dict` signature to accept `shipping_snapshot: Optional[dict] = None`, and include:

```python
        "platform_fee": _money(order.platform_fee),
        "payment_fee": _money(order.payment_fee),
        "other_fee": _money(order.other_fee),
        "product_cost": _money(order.product_cost),
        "profit_amount": _money(order.profit_amount),
        "profit_margin": float(order.profit_margin or 0),
        "profit": {
            "revenue_amount": _money(order.total_amount),
            "product_cost": _money(order.product_cost),
            "shipping_fee": _money(order.shipping_fee),
            "platform_fee": _money(order.platform_fee),
            "payment_fee": _money(order.payment_fee),
            "other_fee": _money(order.other_fee),
            "profit_amount": _money(order.profit_amount),
            "profit_margin": float(order.profit_margin or 0),
        },
        "shipping_snapshot": shipping_snapshot,
```

Update `get_detail` to fetch snapshot:

```python
        shipping_snapshot = await OrderService._get_shipping_snapshot(db, order.id)
        return order_to_dict(order, items, logs, shipping_snapshot)
```

For `list_orders`, do not include snapshots to keep list fast. The `shipping_snapshot` key may be `None`.

- [ ] **Step 4: Add routes**

In `backend/app/order/router.py`, update imports:

```python
from app.order.schemas import OrderCreate, OrderProfitInputsUpdate, OrderShippingQuoteBind, OrderStatusUpdate
```

Add after `get_order`:

```python
@router.post("/orders/{order_id}/shipping-quote", summary="绑定订单运费快照")
async def bind_order_shipping_quote(
    order_id: int,
    data: OrderShippingQuoteBind,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:update")),
):
    try:
        order = await OrderService.bind_shipping_quote(db, order_id, data)
    except ValueError as e:
        return Result.bad_request(str(e))
    if not order:
        return Result.not_found("订单不存在")
    await OperationLogService.log(
        db,
        module="order",
        action="bind_shipping_quote",
        resource_id=str(order_id),
        content=f"绑定订单运费快照: {order['order_no']} 运费={order['shipping_fee']}",
        operator=_operator(current_user),
    )
    return Result.ok(order)


@router.put("/orders/{order_id}/profit-inputs", summary="更新订单利润输入")
async def update_order_profit_inputs(
    order_id: int,
    data: OrderProfitInputsUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:update")),
):
    order = await OrderService.update_profit_inputs(db, order_id, data)
    if not order:
        return Result.not_found("订单不存在")
    await OperationLogService.log(
        db,
        module="order",
        action="update_profit_inputs",
        resource_id=str(order_id),
        content=f"更新订单利润输入: {order['order_no']} 利润={order['profit_amount']}",
        operator=_operator(current_user),
    )
    return Result.ok(order)
```

Use `order:update` for both endpoints. If seed permissions do not include `order:update`, add it in `backend/seed.py` with name `更新订单`.

- [ ] **Step 5: Run backend task tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_order_shipping_snapshot_profit.py -q
```

Expected: pass.

---

### Task 3: Permissions, Audit, And Regression Tests

**Files:**
- Modify: `backend/tests/test_order_auth_audit.py`
- Modify: `backend/seed.py`
- Modify: `docs/PERMISSIONS_AND_AUDIT.md`

- [ ] **Step 1: Add permission tests**

Append to `backend/tests/test_order_auth_audit.py`:

```python
    async def test_bind_shipping_quote_requires_order_update(self, async_client):
        uid, token = await register_and_login(async_client, "ord_ship_no_perm")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.post(
            f"/api/orders/{order_id}/shipping-quote",
            json={"sku_id": sku_id, "quantity": 1, "destination_country": "US", "cargo_type": "normal"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_update_profit_inputs_requires_order_update(self, async_client):
        uid, token = await register_and_login(async_client, "ord_profit_no_perm")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/orders/{order_id}/profit-inputs",
            json={"platform_fee": 10},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403
```

Add one positive audit test:

```python
    async def test_update_profit_inputs_with_permission_logs(self, async_client):
        uid, token = await register_and_login(async_client, "ord_profit_ok")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        await grant_permission(uid, "order:update")
        await grant_permission(uid, "operation_log:view")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/orders/{order_id}/profit-inputs",
            json={"platform_fee": 10, "payment_fee": 2},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=order&action=update_profit_inputs",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200
        assert any(log["resource_id"] == str(order_id) for log in logs_resp.json()["records"])
```

- [ ] **Step 2: Ensure `order:update` seed permission exists**

In `backend/seed.py`, make sure the order permission list includes:

```python
{"code": "order:update", "name": "更新订单", "module": "order"},
```

Do not remove existing `order:view`, `order:create`, `order:update_status`, or `order:cancel`.

- [ ] **Step 3: Update permission docs**

In `docs/PERMISSIONS_AND_AUDIT.md`, update the order row to include:

```markdown
| 订单 | `order:view`, `order:create`, `order:update`, `order:update_status`, `order:cancel` | 已覆盖 |
```

Mention `bind_shipping_quote` and `update_profit_inputs` under order audit actions.

- [ ] **Step 4: Run focused permission tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_order_auth_audit.py tests/test_order_shipping_snapshot_profit.py -q
```

Expected: all tests pass.

---

### Task 4: Frontend Order Detail Integration

**Files:**
- Modify: `frontend/src/api/modules/order.ts`
- Modify: `frontend/src/views/order/OrderDetail.vue`

- [ ] **Step 1: Extend order API module**

In `frontend/src/api/modules/order.ts`, add methods:

```ts
  bindShippingQuote(id: number | string, data: {
    sku_id: number
    quantity: number
    destination_country: string
    postal_code?: string
    cargo_type?: string
    channel_id?: number | null
  }) {
    return http.post(`/orders/${id}/shipping-quote`, data)
  },
  updateProfitInputs(id: number | string, data: {
    platform_fee?: number
    payment_fee?: number
    other_fee?: number
    product_cost?: number
  }) {
    return http.put(`/orders/${id}/profit-inputs`, data)
  },
```

- [ ] **Step 2: Add shipping snapshot display**

In `frontend/src/views/order/OrderDetail.vue`, add a card after basic info:

```vue
      <n-card title="运费快照" style="margin-top: 12px;" :bordered="false">
        <n-empty v-if="!detail.shipping_snapshot" description="暂无运费快照" />
        <n-descriptions v-else :column="2" label-placement="left">
          <n-descriptions-item label="物流商">{{ detail.shipping_snapshot.provider_name }}</n-descriptions-item>
          <n-descriptions-item label="渠道">{{ detail.shipping_snapshot.channel_name }}</n-descriptions-item>
          <n-descriptions-item label="目的地">{{ detail.shipping_snapshot.destination_country }}</n-descriptions-item>
          <n-descriptions-item label="币种">{{ detail.shipping_snapshot.currency }}</n-descriptions-item>
          <n-descriptions-item label="实际重">{{ detail.shipping_snapshot.actual_weight_kg }} kg</n-descriptions-item>
          <n-descriptions-item label="体积重">{{ detail.shipping_snapshot.volumetric_weight_kg }} kg</n-descriptions-item>
          <n-descriptions-item label="计费重">{{ detail.shipping_snapshot.chargeable_weight_kg }} kg</n-descriptions-item>
          <n-descriptions-item label="总运费">¥{{ money(detail.shipping_snapshot.total_shipping_fee) }}</n-descriptions-item>
          <n-descriptions-item label="计算说明" :span="2">{{ detail.shipping_snapshot.calculation_detail || '-' }}</n-descriptions-item>
        </n-descriptions>
      </n-card>
```

Add helper in script:

```ts
function money(value: any) {
  return Number(value || 0).toFixed(2)
}
```

- [ ] **Step 3: Add simple bind form**

Add a compact form in the same card before the display:

```vue
        <n-space style="margin-bottom: 12px;" align="center">
          <n-input v-model:value="shippingForm.destination_country" placeholder="目的国家，如 US" style="width: 140px;" />
          <n-input v-model:value="shippingForm.postal_code" placeholder="邮编，可选" style="width: 140px;" />
          <n-input v-model:value="shippingForm.cargo_type" placeholder="货品类型" style="width: 120px;" />
          <n-button type="primary" :loading="bindingShipping" @click="handleBindShippingQuote">
            计算并保存最低运费
          </n-button>
        </n-space>
```

Add state and method:

```ts
const bindingShipping = ref(false)
const shippingForm = ref({
  destination_country: 'US',
  postal_code: '',
  cargo_type: 'normal',
})

async function handleBindShippingQuote() {
  const firstItem = detail.value?.items?.[0]
  if (!firstItem?.sku_id) {
    message.error('订单缺少 SKU，无法计算运费')
    return
  }
  bindingShipping.value = true
  try {
    const res: any = await apiModules.orderApi.bindShippingQuote(orderId, {
      sku_id: firstItem.sku_id,
      quantity: firstItem.quantity || 1,
      destination_country: shippingForm.value.destination_country,
      postal_code: shippingForm.value.postal_code || undefined,
      cargo_type: shippingForm.value.cargo_type || 'normal',
      channel_id: null,
    })
    detail.value = res?.data || res || {}
    message.success('已保存运费快照')
  } catch (e: any) {
    message.error(e.message || '保存运费快照失败')
  } finally {
    bindingShipping.value = false
  }
}
```

This first pass uses the lowest-price channel returned by the backend. A later UI can expose a full selectable quote table.

- [ ] **Step 4: Add profit display and edit form**

Add another card after shipping snapshot:

```vue
      <n-card title="利润测算" style="margin-top: 12px;" :bordered="false">
        <n-descriptions :column="3" label-placement="left">
          <n-descriptions-item label="销售额">¥{{ money(detail.profit?.revenue_amount) }}</n-descriptions-item>
          <n-descriptions-item label="商品成本">¥{{ money(detail.profit?.product_cost) }}</n-descriptions-item>
          <n-descriptions-item label="运费">¥{{ money(detail.profit?.shipping_fee) }}</n-descriptions-item>
          <n-descriptions-item label="平台费">¥{{ money(detail.profit?.platform_fee) }}</n-descriptions-item>
          <n-descriptions-item label="支付费">¥{{ money(detail.profit?.payment_fee) }}</n-descriptions-item>
          <n-descriptions-item label="其他费用">¥{{ money(detail.profit?.other_fee) }}</n-descriptions-item>
          <n-descriptions-item label="利润">¥{{ money(detail.profit?.profit_amount) }}</n-descriptions-item>
          <n-descriptions-item label="利润率">{{ money(detail.profit?.profit_margin) }}%</n-descriptions-item>
        </n-descriptions>
        <n-space style="margin-top: 12px;" align="center">
          <n-input-number v-model:value="profitForm.product_cost" placeholder="商品成本" :min="0" />
          <n-input-number v-model:value="profitForm.platform_fee" placeholder="平台费" :min="0" />
          <n-input-number v-model:value="profitForm.payment_fee" placeholder="支付费" :min="0" />
          <n-input-number v-model:value="profitForm.other_fee" placeholder="其他费用" :min="0" />
          <n-button :loading="savingProfit" @click="handleSaveProfitInputs">保存利润输入</n-button>
        </n-space>
      </n-card>
```

Add state and load values in `fetchDetail` after assigning `detail.value`:

```ts
const savingProfit = ref(false)
const profitForm = ref({
  product_cost: 0,
  platform_fee: 0,
  payment_fee: 0,
  other_fee: 0,
})
```

Inside `fetchDetail`, after `detail.value = ...`:

```ts
    profitForm.value = {
      product_cost: Number(detail.value.product_cost || 0),
      platform_fee: Number(detail.value.platform_fee || 0),
      payment_fee: Number(detail.value.payment_fee || 0),
      other_fee: Number(detail.value.other_fee || 0),
    }
```

Add method:

```ts
async function handleSaveProfitInputs() {
  savingProfit.value = true
  try {
    const res: any = await apiModules.orderApi.updateProfitInputs(orderId, profitForm.value)
    detail.value = res?.data || res || {}
    message.success('利润输入已保存')
  } catch (e: any) {
    message.error(e.message || '保存利润输入失败')
  } finally {
    savingProfit.value = false
  }
}
```

- [ ] **Step 5: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build passes.

---

### Task 5: Documentation And Final Verification

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`
- Modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`
- Modify: `docs/LOGISTICS_AND_SHIPPING_PRD.md`

- [ ] **Step 1: Update project status**

In `docs/PROJECT_STATUS.md`, add a section after the shipping phase:

```markdown
### 物流和运费第三阶段 — 订单运费快照与利润一期

状态：已完成。

已实现：
- `sales_order_shipping_snapshot` 保存订单选用的运费快照。
- `POST /api/orders/{id}/shipping-quote` 调用现有运费计算并保存最低价或指定渠道报价。
- 订单详情返回 `shipping_snapshot` 和 `profit`。
- 历史订单运费不受后续报价规则变化影响。
- 利润一期字段：销售额、商品成本、运费、平台费、支付费、其他费用、利润、利润率。
- 前端订单详情页展示运费快照和利润测算。

未实现：
- 多包裹装箱优化。
- 真实物流商 API 报价。
- 面单、追踪号、物流轨迹。
- 运费对账。
```

- [ ] **Step 2: Update roadmap**

In `docs/ROADMAP.md`, update Phase 6:

```markdown
状态：第三阶段已完成。订单可保存运费快照并进行利润一期测算。
```

Change checklist item:

```markdown
5. ✅ 订单运费快照。
```

Add:

```markdown
9. ✅ 订单利润一期测算。
```

Update remaining line:

```markdown
待下一阶段：真实承运商 API、装箱优化、面单/追踪、运费对账。
```

- [ ] **Step 3: Update logistics technical spec**

In `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`, update the `ShippingQuoteSnapshot` section to name the actual table:

```markdown
实际表名：`sales_order_shipping_snapshot`。

实际绑定接口：
- `POST /api/orders/{id}/shipping-quote`
- `PUT /api/orders/{id}/profit-inputs`
```

Mention that `sales_order.shipping_fee` remains a denormalized compatibility field.

- [ ] **Step 4: Full verification**

Run:

```bash
cd backend && python3 -m alembic upgrade head
cd backend && python3 -m pytest tests/test_order_shipping_snapshot_profit.py tests/test_order.py tests/test_order_auth_audit.py tests/test_shipping_calculation.py -q
cd backend && python3 -m pytest -q
cd frontend && npm run build
git diff --check
```

Expected:
- Migration succeeds.
- Focused backend tests pass.
- Full backend suite passes.
- Frontend build passes.
- `git diff --check` prints no output.

---

## Reasonix Handoff Prompt

```text
你接手 MultiSell 的“订单运费快照 + 利润计算一期”任务。

必须先阅读：
- docs/superpowers/plans/2026-06-14-order-shipping-snapshot-profit.md
- backend/app/order/service.py
- backend/app/order/router.py
- backend/app/order/schemas.py
- backend/app/shipping/service.py
- backend/app/shipping/schemas.py
- backend/app/models.py
- frontend/src/views/order/OrderDetail.vue
- frontend/src/api/modules/order.ts

目标：
1. 订单可以调用现有 /api/shipping/calculate 并保存选中的运费结果。
2. 保存的是历史快照，不是只保存 rule_id 或 channel_id。
3. 报价规则后续变化不能改变历史订单运费。
4. 订单详情返回并展示 shipping_snapshot。
5. 订单详情返回并展示 profit。
6. 支持利润输入：product_cost、platform_fee、payment_fee、other_fee。
7. shipping_fee 继续保留并等于当前订单选用快照的 total_shipping_fee。
8. 新接口接入 RBAC 和审计日志。

范围限制：
- 不接真实物流商 API。
- 不做面单、追踪、物流轨迹。
- 不做多包裹装箱优化。
- 不做运费对账。
- 不重构整个订单模块。

验收命令：
- cd backend && python3 -m alembic upgrade head
- cd backend && python3 -m pytest tests/test_order_shipping_snapshot_profit.py tests/test_order.py tests/test_order_auth_audit.py tests/test_shipping_calculation.py -q
- cd backend && python3 -m pytest -q
- cd frontend && npm run build
- git diff --check

完成后汇报：
- 改了哪些文件
- 新增了哪些 API
- 测试结果
- 未完成的下一阶段能力
```

## Acceptance Checklist

- [ ] `sales_order_shipping_snapshot` table exists and has one row per order.
- [ ] `POST /api/orders/{id}/shipping-quote` saves a selected quote snapshot.
- [ ] Omitting `channel_id` selects the cheapest result returned by `CalculateService`.
- [ ] Passing `channel_id` saves that available channel or returns a 400 when unavailable.
- [ ] Missing package dimensions/weight returns a 400 business response.
- [ ] Updating shipping quote rules does not alter an existing order snapshot.
- [ ] Order detail includes `shipping_snapshot`.
- [ ] Order detail includes `profit`.
- [ ] `shipping_fee` and `pay_amount` remain compatible with older order UI.
- [ ] `PUT /api/orders/{id}/profit-inputs` recalculates profit.
- [ ] New write endpoints require `order:update`.
- [ ] New write endpoints create operation logs.
- [ ] Backend focused tests pass.
- [ ] Backend full test suite passes.
- [ ] Frontend build passes.
- [ ] `git diff --check` passes.
