> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# P3: 平台结算导入 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Import platform settlement/finance reports via API into local `Settlement` + `SettlementItem` tables, enabling profit reconciliation against platform data.

**Architecture:** Add `fetch_settlements()` to adapters, create a settlement sync worker (same pattern as order sync worker), map platform settlement lines to `SettlementItem` rows with `transaction_type` (order_sale/refund/shipping_fee/platform_fee).

**Tech Stack:** Same adapters, existing `Settlement` + `SettlementItem` models in `app/models.py`, existing `settlement/router.py` for query endpoints.

**Current state:** `Settlement` model exists with status workflow (pending→reconciling→reconciled). No API-based import, only CSV.

---

### Task 1: Add `fetch_settlements()` to adapters

**Files:**
- Modify: `backend/app/listing/adapters/base.py`
- Modify: `backend/app/listing/adapters/ozon.py`
- Modify: `backend/app/listing/adapters/shopee.py`
- Test: `backend/tests/test_listing_adapters.py`

**Ozon API:** `POST /v3/finance/transaction/list` returns transaction list filtered by date.

**Shopee API:** `GET /api/v2/payment/list_escrow_detail` — get transaction details.

- [ ] **Step 1: Add protocol method**

```python
# base.py
async def fetch_settlements(
    self,
    *,
    platform: Platform,
    since: datetime,
    db: Optional[AsyncSession] = None,
) -> list[dict]:
    """从平台拉取结算/交易记录。返回 list[dict]，每条包含：
    - transaction_id: str
    - transaction_type: str — order_sale / refund / shipping_fee / platform_fee / payment_fee
    - order_sn: str (optional)
    - amount: str
    - fee: str (optional)
    - currency: str
    - occurred_at: str (ISO datetime)
    - description: str (optional)
    """
```

- [ ] **Step 2: Implement Ozon**

```python
# ozon.py
async def fetch_settlements(self, *, platform: Platform, since: datetime,
                            db: Optional[AsyncSession] = None) -> list[dict]:
    limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
    await limiter.acquire()

    payload = {
        "filter": {"date": {"from": since.strftime("%Y-%m-%dT%H:%M:%S.000Z")}},
        "page": 1,
        "page_size": 100,
    }
    async with self._client(platform) as client:
        resp = await client.post("/v3/finance/transaction/list", json=payload)
        body = self._parse_response(resp, "fetch_settlements")

    TYPE_MAP = {
        "sale": "order_sale",
        "refund": "refund",
        "delivery": "shipping_fee",
        "commission": "platform_fee",
        "payment_commission": "payment_fee",
    }
    items = []
    for tx in body.get("result", {}).get("operations", []):
        items.append({
            "transaction_id": str(tx.get("operation_id", "")),
            "transaction_type": TYPE_MAP.get(tx.get("operation_type", ""), "other"),
            "order_sn": tx.get("posting", {}).get("posting_number", ""),
            "amount": str(abs(float(tx.get("amount", "0")))),
            "fee": "0",
            "currency": tx.get("currency_code", "RUB"),
            "occurred_at": tx.get("operation_date", ""),
            "description": tx.get("description", ""),
        })
    return items
```

- [ ] **Step 3: Implement Shopee**

Shopee's financial API is more complex (requires escrow details per order). For Phase 1, return empty list with a log warning.

```python
# shopee.py
async def fetch_settlements(self, *, platform: Platform, since: datetime,
                            db=None) -> list[dict]:
    logger.warning("Shopee settlement import not yet implemented")
    return []
```

- [ ] **Step 4: Commit**

```bash
git add backend/app/listing/adapters/base.py backend/app/listing/adapters/ozon.py backend/app/listing/adapters/shopee.py
git commit -m "feat: fetch_settlements adapter method + Ozon implementation"
```

---

### Task 2: Settlement sync worker

**Files:**
- Create: `backend/app/finance/settlement_sync_worker.py`
- Modify: `backend/app/main.py`
- Test: `backend/tests/test_settlement_sync_worker.py`

- [ ] **Step 1: Implement mapping + worker**

```python
# backend/app/finance/settlement_sync_worker.py
"""Settlement sync worker — poll platform APIs, upsert Settlement + SettlementItem."""
import asyncio, logging
from datetime import datetime, timezone, timedelta

from app.database import async_session_factory
from app.listing.adapters import get_listing_adapter
from app.models import Platform, Settlement, SettlementItem

logger = logging.getLogger(__name__)


class SettlementSyncWorker:
    def __init__(self, poll_interval: float = 3600.0):  # every hour
        self._poll_interval = poll_interval
        self._task: asyncio.Task | None = None
        self._stop = asyncio.Event()

    async def start(self): ...  # same pattern as OrderSyncWorker
    async def stop(self): ...

    async def _tick(self):
        async with async_session_factory() as db:
            platforms = (await db.execute(
                select(Platform).where(Platform.status == 1)
            )).scalars().all()
            for platform in platforms:
                adapter = get_listing_adapter(platform.code)
                if not hasattr(adapter, "fetch_settlements"):
                    continue
                since = datetime.now(timezone.utc) - timedelta(days=7)
                items = await adapter.fetch_settlements(platform=platform, since=since, db=db)
                for tx in items:
                    await self._upsert_tx(db, tx, platform.id)
            await db.commit()

    async def _upsert_tx(self, db, tx: dict, platform_id: int):
        existing = (await db.execute(
            select(SettlementItem).where(
                SettlementItem.transaction_id == tx["transaction_id"],
                SettlementItem.settlement_rel.has(Settlement.platform_id == platform_id),
            )
        )).first()
        if existing:
            return  # already imported
        # Find or create settlement batch for the period
        period_start = datetime.fromisoformat(tx["occurred_at"].replace("Z", "+00:00")).replace(day=1)
        settlement = (await db.execute(
            select(Settlement).where(
                Settlement.platform_id == platform_id,
                Settlement.period_start == period_start,
                Settlement.status == "pending",
            )
        )).scalar_one_or_none()
        if not settlement:
            settlement = Settlement(
                platform_id=platform_id,
                settlement_no=f"API-{platform_id}-{period_start.strftime('%Y%m')}",
                period_start=period_start,
                status="pending",
            )
            db.add(settlement)
            await db.flush()

        item = SettlementItem(
            settlement_id=settlement.id,
            transaction_id=tx["transaction_id"],
            transaction_type=tx["transaction_type"],
            order_sn=tx.get("order_sn", ""),
            amount=Decimal(str(tx.get("amount", "0"))),
            fee=Decimal(str(tx.get("fee", "0"))),
            net=Decimal(str(tx.get("amount", "0"))) - Decimal(str(tx.get("fee", "0"))),
            occurred_at=datetime.fromisoformat(tx["occurred_at"].replace("Z", "+00:00")),
        )
        db.add(item)
```

- [ ] **Step 2: Wire into app lifespan**

Same pattern as order worker — add start/stop in `app/main.py`.

- [ ] **Step 3: Run tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/ -q --tb=short`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/app/finance/settlement_sync_worker.py backend/app/main.py
git commit -m "feat: settlement sync worker"
```
