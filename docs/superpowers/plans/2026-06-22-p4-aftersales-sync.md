> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# P4: 售后单同步 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sync returns/refunds initiated on platforms back to local `AfterSalesOrder` table, so local staff can process inspection and restocking.

**Architecture:** Add `fetch_returns()` to adapters. Create a return sync worker that polls platforms for new return requests. If a return matches a known order, auto-create an `AfterSalesOrder` record with status `"pending"`. Add a frontend list view to see platform-initiated returns alongside locally created ones.

**Tech Stack:** Same adapters, existing `AfterSalesOrder` model + service + router from previous sprint, frontend `aftersales` views.

**Current state:** `AfterSalesOrder` has full state machine (pending→approved/rejected→received→refunded) + 入库自动恢复库存. No API-based import from platforms.

---

### Task 1: Add `fetch_returns()` to adapters + Ozon implementation

**Files:**
- Modify: `backend/app/listing/adapters/base.py`
- Modify: `backend/app/listing/adapters/ozon.py`
- Test: `backend/tests/test_listing_adapters.py`

**Ozon API:** `POST /v3/returns/list` returns FBS return requests.

- [ ] **Step 1: Add protocol method**

```python
# base.py
async def fetch_returns(
    self,
    *,
    platform: Platform,
    since: datetime,
    db: Optional[AsyncSession] = None,
) -> list[dict]:
    """从平台拉取售后/退货申请。返回 list[dict]:
    - return_id: str — 平台退货ID
    - order_sn: str
    - sku_code: str
    - quantity: int
    - reason: str
    - status: str — platform's return status
    - created_at: str (ISO)
    - refund_amount: str (optional)
    """
```

- [ ] **Step 2: Implement Ozon**

```python
# ozon.py
async def fetch_returns(self, *, platform: Platform, since: datetime,
                        db: Optional[AsyncSession] = None) -> list[dict]:
    limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
    await limiter.acquire()
    payload = {
        "filter": {"last_change_from": since.strftime("%Y-%m-%dT%H:%M:%S.000Z")},
        "limit": 100,
    }
    async with self._client(platform) as client:
        resp = await client.post("/v3/returns/list", json=payload)
        body = self._parse_response(resp, "fetch_returns")

    items = []
    for r in body.get("result", {}).get("returns", []):
        items.append({
            "return_id": str(r.get("return_id", "")),
            "order_sn": r.get("posting_number", ""),
            "sku_code": r.get("sku", ""),
            "quantity": r.get("quantity", 1),
            "reason": r.get("reason", "平台发起退货"),
            "status": r.get("status", "pending"),
            "created_at": r.get("created_at", ""),
            "refund_amount": str(r.get("refund_amount", "0")),
        })
    return items
```

- [ ] **Step 3: Commit**

```bash
git add backend/app/listing/adapters/base.py backend/app/listing/adapters/ozon.py
git commit -m "feat: fetch_returns adapter method + Ozon impl"
```

---

### Task 2: Return sync worker

**Files:**
- Create: `backend/app/aftersales/sync_worker.py`
- Modify: `backend/app/main.py`
- Test: `backend/tests/test_aftersales_sync_worker.py`

- [ ] **Step 1: Implement worker**

```python
# backend/app/aftersales/sync_worker.py
"""Return sync worker — poll platform APIs, create AfterSalesOrder records."""
import asyncio, logging
from datetime import datetime, timezone, timedelta
from decimal import Decimal

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import async_session_factory
from app.listing.adapters import get_listing_adapter
from app.models import AfterSalesOrder, Order, OrderItem, Platform, Sku

logger = logging.getLogger(__name__)


class ReturnSyncWorker:
    def __init__(self, poll_interval: float = 600.0):  # 10 min
        self._poll_interval = poll_interval
        self._task = None
        self._stop = asyncio.Event()

    async def start(self): ...  # same pattern
    async def stop(self): ...

    async def _tick(self):
        async with async_session_factory() as db:
            platforms = (await db.execute(
                select(Platform).where(Platform.status == 1)
            )).scalars().all()
            for platform in platforms:
                adapter = get_listing_adapter(platform.code)
                if not hasattr(adapter, "fetch_returns"):
                    continue
                since = datetime.now(timezone.utc) - timedelta(hours=1)
                returns = await adapter.fetch_returns(platform=platform, since=since, db=db)
                for r in returns:
                    await self._upsert_return(db, r)
            await db.commit()

    async def _upsert_return(self, db: AsyncSession, r: dict):
        # dedup by platform return_id
        # match order_sn to local Order
        # match sku_code to local Sku
        # create AfterSalesOrder with status="pending"
        ...
```

- [ ] **Step 2: Wire into app lifespan**

- [ ] **Step 3: Run tests + commit**

---

### Task 3: Frontend — platform return list

**Files:**
- Create: `frontend/src/views/aftersales/ReturnList.vue`
- Modify: `frontend/src/router/modules/aftersales.ts`

- [ ] **Step 1: Create return list view**

Same pattern as `ListingTaskDetail.vue` — NDataTable with columns: order_no, sku, quantity, reason, status, created_at, action buttons.

- [ ] **Step 2: Add route**

- [ ] **Step 3: Run frontend build**

Run: `cd frontend && npm run build`
Expected: PASS

- [ ] **Step 4: Commit**
