# P1: 平台订单实时导入 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace manual CSV order import with automatic order sync from Ozon/Shopee APIs.

**Architecture:** Extend each adapter with `fetch_orders()`, create an order sync worker that polls for new/updated orders every N minutes, map platform order fields to local `Order`/`OrderItem` models, and handle dedup via `order_no` unique constraint. Follow the same worker pattern as listing worker (Task 1 of the platform API plan).

**Tech Stack:** FastAPI background worker, `httpx`, existing `OrderService`, `Platform` model, adapter pattern.

**Current state:** `OrderImportService` reads CSV via `csv.DictReader`. Adapters declare `supports_order_import=True` but implement nothing. Ozon/Shopee have real HTTP clients in `_client()`.

---

### Task 1: Add `fetch_orders()` to Adapter Protocol

**Files:**
- Modify: `backend/app/listing/adapters/base.py`

- [ ] **Step 1: Add protocol method**

```python
# base.py — add to ListingAdapter protocol
async def fetch_orders(
    self,
    *,
    platform: Platform,
    since: datetime,
    db: Optional[AsyncSession] = None,
) -> list[dict]:
    """从平台拉取订单列表。返回 list[dict]，每个 dict 包含:
    - order_sn: str — 平台订单号
    - status: str — 平台状态
    - total_amount: str — 金额字符串
    - shipping_fee: str — 运费字符串
    - paid_at: str — ISO 时间
    - recipient_name: str
    - recipient_phone: str
    - shipping_address: str
    - items: list[{"sku_code": str, "quantity": int, "unit_price": str}]
    """
```

- [ ] **Step 2: Commit**

```bash
git add backend/app/listing/adapters/base.py
git commit -m "feat: add fetch_orders to adapter protocol"
```

---

### Task 2: Implement Ozon `fetch_orders()`

**Files:**
- Modify: `backend/app/listing/adapters/ozon.py`
- Test: `backend/tests/test_listing_adapters.py`

**Ozon API:** `POST /v3/posting/fbs/list` returns FBS orders. Filter by `since` param.

- [ ] **Step 1: Write test (mock HTTP)**

```python
# tests/test_listing_adapters.py
@pytest.mark.asyncio
async def test_ozon_fetch_orders(async_client, db_session):
    adapter = OzonListingAdapter()
    platform = Platform(id=1, code="ozon", client_id="test", api_key="test",
                        api_base_url="https://api-seller.ozon.ru")
    with patch.object(adapter, '_client') as mock_client:
        mock_client.return_value.__aenter__.return_value.post.return_value.json.return_value = {
            "result": {"postings": [{
                "posting_number": "12345",
                "status": "delivered",
                "analytics_data": {"delivery_price": "5.00"},
                "financial_data": {"products": [{"sku": "SKU001", "quantity": 2, "price": "19.99"}]},
                "in_process_at": "2026-06-20T10:00:00Z",
            }]}
        }
        result = await adapter.fetch_orders(platform=platform, since=datetime(2026, 6, 19), db=db_session)
        assert len(result) == 1
        assert result[0]["order_sn"] == "12345"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_listing_adapters.py::test_ozon_fetch_orders -q --tb=short`
Expected: FAIL — OzonListingAdapter has no fetch_orders

- [ ] **Step 3: Implement `fetch_orders()`**

```python
# ozon.py — add to OzonListingAdapter
async def fetch_orders(self, *, platform: Platform, since: datetime,
                       db: Optional[AsyncSession] = None) -> list[dict]:
    """拉取 Ozon FBS 订单"""
    limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
    await limiter.acquire()
    
    payload = {
        "dir": "ASC",
        "filter": {"since": since.strftime("%Y-%m-%dT%H:%M:%S.000Z")},
        "limit": 100,
    }
    async with self._client(platform) as client:
        resp = await client.post("/v3/posting/fbs/list", json=payload)
        body = self._parse_response(resp, "fetch_orders")
    
    orders = []
    for p in body.get("result", {}).get("postings", []):
        items = []
        for prod in p.get("financial_data", {}).get("products", []):
            items.append({
                "sku_code": prod.get("sku", ""),
                "quantity": prod.get("quantity", 0),
                "unit_price": str(prod.get("price", "0")),
            })
        orders.append({
            "order_sn": p.get("posting_number", ""),
            "status": p.get("status", ""),
            "total_amount": str(sum(float(i["unit_price"]) * i["quantity"] for i in items)),
            "shipping_fee": str(p.get("analytics_data", {}).get("delivery_price", "0")),
            "paid_at": p.get("in_process_at", ""),
            "recipient_name": "",
            "recipient_phone": "",
            "shipping_address": "",
            "items": items,
        })
    return orders
```

- [ ] **Step 4: Run test**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_listing_adapters.py::test_ozon_fetch_orders -q --tb=short`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/app/listing/adapters/ozon.py backend/tests/test_listing_adapters.py
git commit -m "feat: Ozon fetch_orders implementation"
```

---

### Task 3: Implement Shopee `fetch_orders()`

**Files:**
- Modify: `backend/app/listing/adapters/shopee.py`
- Test: `backend/tests/test_listing_adapters.py`

**Shopee API:** `GET /api/v2/order/get_order_list` with `time_range_field=create_time` + `page_size=100`.

- [ ] **Step 1: Implement**

```python
# shopee.py
async def fetch_orders(self, *, platform: Platform, since: datetime,
                       db: Optional[AsyncSession] = None) -> list[dict]:
    api_path = "/api/v2/order/get_order_list"
    params = self._build_auth_params(platform, api_path)
    params["time_range_field"] = "create_time"
    params["page_size"] = 100
    params["create_time_from"] = int(since.timestamp())
    params["create_time_to"] = int(datetime.now(timezone.utc).timestamp())
    
    limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
    await limiter.acquire()
    
    async with self._client(platform) as client:
        resp = await client.get(api_path, params=params)
        body = self._parse_response(resp, "fetch_orders")
    
    orders = []
    for order_sn in body.get("response", {}).get("order_list", []):
        orders.append(await self._fetch_order_detail(platform, order_sn))
    return orders

async def _fetch_order_detail(self, platform: Platform, order_sn: str) -> dict:
    api_path = "/api/v2/order/get_order_detail"
    params = self._build_auth_params(platform, api_path)
    params["order_sn_list"] = order_sn
    async with self._client(platform) as client:
        resp = await client.get(api_path, params=params)
        body = self._parse_response(resp, "fetch_order_detail")
    detail = body.get("response", {}).get("order_list", [{}])[0]
    items = [{
        "sku_code": i.get("item_sku", ""),
        "quantity": i.get("model_quantity_purchased", 0),
        "unit_price": str(i.get("model_original_price", "0")),
    } for i in detail.get("item_list", [])]
    return {
        "order_sn": order_sn,
        "status": detail.get("order_status", ""),
        "total_amount": str(detail.get("total_amount", {}).get("value", "0")),
        "shipping_fee": "0",
        "paid_at": detail.get("pay_time", ""),
        "recipient_name": detail.get("recipient_address", {}).get("name", ""),
        "recipient_phone": detail.get("recipient_address", {}).get("phone", ""),
        "shipping_address": detail.get("recipient_address", {}).get("full_address", ""),
        "items": items,
    }
```

- [ ] **Step 2: Write test + run**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_listing_adapters.py -q --tb=short`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/app/listing/adapters/shopee.py
git commit -m "feat: Shopee fetch_orders implementation"
```

---

### Task 4: Order sync worker

**Files:**
- Create: `backend/app/order_import/sync_worker.py`
- Modify: `backend/app/main.py`
- Test: `backend/tests/test_order_sync_worker.py`

**Purpose:** Background worker that polls adapters' `fetch_orders()`, maps platform orders to local models, and upserts via `order_no` unique constraint.

- [ ] **Step 1: Write the mapping function + test**

```python
# tests/test_order_sync_worker.py
from app.order_import.sync_worker import map_platform_order

def test_map_platform_order_to_local():
    platform_order = {
        "order_sn": "OZON-123",
        "status": "delivered",
        "total_amount": "29.99",
        "shipping_fee": "5.00",
        "paid_at": "2026-06-20T10:00:00Z",
        "recipient_name": "Alice",
        "recipient_phone": "123456",
        "shipping_address": "Moscow, Red Square 1",
        "items": [{"sku_code": "SKU001", "quantity": 2, "unit_price": "14.995"}],
    }
    local = map_platform_order(platform_order, platform_id=1)
    assert local["order_no"] == "OZON-123"
    assert local["status"] == "delivered"
    assert local["total_amount"] == 29.99
    assert len(local["items"]) == 1
```

- [ ] **Step 2: Implement mapping**

```python
# backend/app/order_import/sync_worker.py
"""Orders sync worker — poll platform APIs, upsert locally."""
import asyncio
import logging
from datetime import datetime, timezone, timedelta
from decimal import Decimal

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import async_session_factory
from app.listing.adapters import get_listing_adapter
from app.models import Order, OrderItem, OrderStatusLog, Platform

logger = logging.getLogger(__name__)

STATUS_MAP = {
    "delivered": "delivered",
    "shipped": "shipped",
    "ready_to_ship": "paid",
    "cancelled": "cancelled",
}


def map_platform_order(platform_order: dict, platform_id: int) -> dict:
    items = []
    for i in platform_order.get("items", []):
        items.append({
            "sku_code": i.get("sku_code", ""),
            "quantity": i.get("quantity", 0),
            "unit_price": Decimal(str(i.get("unit_price", "0"))),
            "subtotal": Decimal(str(i.get("unit_price", "0"))) * i.get("quantity", 0),
        })
    total = sum(i["subtotal"] for i in items)
    fee_str = platform_order.get("shipping_fee", "0")
    return {
        "order_no": platform_order["order_sn"],
        "status": STATUS_MAP.get(platform_order.get("status", ""), "pending"),
        "total_amount": total,
        "shipping_fee": Decimal(str(fee_str)),
        "pay_amount": total + Decimal(str(fee_str)),
        "recipient_name": platform_order.get("recipient_name", ""),
        "recipient_phone": platform_order.get("recipient_phone", ""),
        "shipping_address": platform_order.get("shipping_address", ""),
        "paid_at": datetime.fromisoformat(platform_order["paid_at"].replace("Z", "+00:00"))
            if platform_order.get("paid_at") else None,
        "items": items,
    }


class OrderSyncWorker:
    def __init__(self, poll_interval: float = 300.0):
        self._poll_interval = poll_interval
        self._task: asyncio.Task | None = None
        self._stop = asyncio.Event()

    async def start(self):
        self._stop.clear()
        self._task = asyncio.create_task(self._loop())
        logger.info("OrderSyncWorker started (poll every %ss)", self._poll_interval)

    async def stop(self):
        self._stop.set()
        if self._task:
            self._task.cancel()
            try: await self._task
            except asyncio.CancelledError: pass

    async def _loop(self):
        while not self._stop.is_set():
            try:
                await self._tick()
            except Exception:
                logger.exception("OrderSyncWorker tick failed")
            await asyncio.sleep(self._poll_interval)

    async def _tick(self):
        async with async_session_factory() as db:
            platforms = (await db.execute(
                select(Platform).where(Platform.status == 1)
            )).scalars().all()
            for platform in platforms:
                adapter = get_listing_adapter(platform.code)
                if not hasattr(adapter, "fetch_orders"):
                    continue
                since = datetime.now(timezone.utc) - timedelta(hours=1)
                try:
                    orders = await adapter.fetch_orders(platform=platform, since=since, db=db)
                    for order in orders:
                        await self._upsert_order(db, order, platform.id)
                except Exception:
                    logger.exception("Failed to fetch orders for %s", platform.code)
            await db.commit()

    async def _upsert_order(self, db: AsyncSession, platform_order: dict, platform_id: int):
        from app.orderimport.service import OrderImportService
        mapped = map_platform_order(platform_order, platform_id)
        existing = (await db.execute(
            select(Order).where(Order.order_no == mapped["order_no"])
        )).scalar_one_or_none()
        if existing:
            if existing.status != mapped["status"]:
                existing.status = mapped["status"]
        else:
            order = Order(
                order_no=mapped["order_no"],
                status=mapped["status"],
                total_amount=mapped["total_amount"],
                shipping_fee=mapped["shipping_fee"],
                pay_amount=mapped["pay_amount"],
                recipient_name=mapped["recipient_name"],
                recipient_phone=mapped["recipient_phone"],
                shipping_address=mapped["shipping_address"],
                paid_at=mapped["paid_at"],
            )
            db.add(order)
```

- [ ] **Step 3: Wire into app lifespan**

```python
# app/main.py
from app.order_import.sync_worker import OrderSyncWorker as _OrderSyncWorker
_order_sync_worker = _OrderSyncWorker()
# add to lifespan: start after listing worker, stop before agent scheduler
```

- [ ] **Step 4: Run full test suite**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/ -q --tb=short`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/app/order_import/sync_worker.py backend/app/main.py backend/tests/test_order_sync_worker.py
git commit -m "feat: order sync worker"
```

---

### Task 5: Order sync admin command + initial backfill

**Files:**
- Create: `backend/app/order_import/management.py`

**Purpose:** CLI/admin endpoint to trigger a backfill of orders from a platform (for initial load when setting up a new platform connection).

- [ ] **Step 1: Implement backfill function**

```python
# backend/app/order_import/management.py
async def backfill_orders(db: AsyncSession, platform_id: int, since: datetime, days_back: int = 7):
    """Backfill orders from a platform for the given time range."""
    platform = await db.get(Platform, platform_id)
    if not platform:
        raise ValueError(f"Platform {platform_id} not found")
    adapter = get_listing_adapter(platform.code)
    if not hasattr(adapter, "fetch_orders"):
        raise ValueError(f"Platform {platform.code} does not support order import")
    since = since or datetime.now(timezone.utc) - timedelta(days=days_back)
    orders = await adapter.fetch_orders(platform=platform, since=since, db=db)
    count = 0
    for order in orders:
        from app.order_import.sync_worker import _upsert_order
        await _upsert_order(db, order, platform.id)
        count += 1
    return count
```

- [ ] **Step 2: Commit**

```bash
git add backend/app/order_import/management.py
git commit -m "feat: order backfill command"
```
