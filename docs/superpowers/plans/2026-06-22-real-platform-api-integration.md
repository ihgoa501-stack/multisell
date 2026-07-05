> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# 真实平台 API 接入 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable production publishing to Ozon and Shopee from listing tasks — background execution, retry, credential validation, and failure recovery.

**Architecture:** Add a background worker that polls pending `ListingTaskItem` rows, executes them through existing adapters (Ozon/Shopee have real `publish()` implementations), handles transient failures with exponential backoff, and exposes retry/failure-recovery UI. Wire the credential `test_connection` endpoint to call the real adapter's `validate_credentials()` instead of returning mock success.

**Tech Stack:** FastAPI background tasks → asyncio worker loop, SQLAlchemy polling, existing adapters (ozon.py/shopee.py), existing `ListingTaskItem` model + status field.

**Current state:** Adapters (Ozon/Shopee) have real `publish()`/`sync_status()`/`validate_credentials()` calling live APIs with HMAC/auth headers. Rate limiter and API key decryption are committed. What's missing: nobody picks up and runs pending listing tasks; failed tasks have no retry; `test_connection` returns mock success; no webhook receivers.

---

### Task 1: Background listing task worker

**Files:**
- Create: `backend/app/listing/worker.py`
- Modify: `backend/app/main.py` (start/stop worker in lifespan)
- Test: `backend/tests/test_listing_worker.py`

**Purpose:** An asyncio worker that polls `ListingTaskItem` rows with `status="pending"`, calls the appropriate adapter's `publish()`, and updates status to `"success"` or `"failed"`.

- [ ] **Step 1: Write worker skeleton + test**

```python
# tests/test_listing_worker.py
import pytest
from app.listing.worker import ListingWorker

@pytest.mark.asyncio
async def test_worker_polls_and_executes_pending_items(async_client, db_session):
    """Worker picks up a pending ListingTaskItem and calls its adapter."""
    worker = ListingWorker(poll_interval=0.1)
    # Start worker in background
    await worker.start()
    await asyncio.sleep(0.3)
    await worker.stop()
    # Verify pending items were processed
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_listing_worker.py -q --tb=short`
Expected: ImportError or ModuleNotFoundError

- [ ] **Step 3: Implement the worker**

```python
# app/listing/worker.py
"""Background worker that executes pending listing task items."""
import asyncio
import logging

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import async_session_factory
from app.listing.adapters import get_listing_adapter
from app.models import ListingTaskItem, Product, Platform

logger = logging.getLogger(__name__)


class ListingWorker:
    """Polls and executes pending listing task items."""

    def __init__(self, poll_interval: float = 10.0):
        self._poll_interval = poll_interval
        self._task: asyncio.Task | None = None
        self._stop_event = asyncio.Event()

    async def start(self):
        self._stop_event.clear()
        self._task = asyncio.create_task(self._loop())
        logger.info("ListingWorker started (poll every %ss)", self._poll_interval)

    async def stop(self):
        self._stop_event.set()
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        logger.info("ListingWorker stopped")

    async def _loop(self):
        while not self._stop_event.is_set():
            try:
                await self._tick()
            except Exception:
                logger.exception("ListingWorker tick failed")
            await asyncio.sleep(self._poll_interval)

    async def _tick(self):
        async with async_session_factory() as db:
            stmt = (
                select(ListingTaskItem)
                .where(ListingTaskItem.status == "pending")
                .limit(5)
            )
            rows = (await db.execute(stmt)).scalars().all()
            for item in rows:
                await self._execute_one(db, item)

    async def _execute_one(self, db: AsyncSession, item: ListingTaskItem):
        try:
            platform = await db.get(Platform, item.platform_id)
            product = await db.get(Product, item.product_id)
            if not platform or not product:
                item.status = "failed"
                item.error_message = "Platform or Product not found"
                await db.flush()
                return

            adapter = get_listing_adapter(platform.code)
            result = await adapter.publish(
                product=product,
                platform=platform,
                skus=[],          # ponytail: wire from product
                prices={},        # ponytail: wire from product
                inventories={},   # ponytail: wire from product
                db=db,
            )
            item.status = "success"
            item.result = {
                "platform_product_id": result.platform_product_id,
                "platform_url": result.platform_url,
                "platform_sku": result.platform_sku,
            }
            item.executed_at = __import__("datetime").datetime.now(__import__("datetime").timezone.utc)
        except Exception as exc:
            item.status = "failed"
            item.error_message = str(exc)[:500]
            item.retry_count = (item.retry_count or 0) + 1
        await db.flush()
```

- [ ] **Step 4: Write a realistic test**

```python
# tests/test_listing_worker.py
import asyncio
import pytest
from unittest.mock import patch, AsyncMock

from app.listing.worker import ListingWorker
from app.models import ListingTaskItem


@pytest.mark.asyncio
async def test_worker_marks_item_failed_on_adapter_error(async_client, db_session):
    """If the adapter raises, item gets status='failed' + error_message."""
    worker = ListingWorker(poll_interval=0.05)
    try:
        await worker.start()
        await asyncio.sleep(0.2)
    finally:
        await worker.stop()

    async with db_session.begin():
        stmt = select(ListingTaskItem).where(ListingTaskItem.id == item_id)
        row = (await db_session.execute(stmt)).scalar_one_or_none()
        assert row.status == "failed"
        assert row.error_message
```

- [ ] **Step 5: Wire worker into app lifespan**

```python
# app/main.py — add to lifespan
from app.listing.worker import ListingWorker as _ListingWorker

_worker = _ListingWorker()

@asynccontextmanager
async def lifespan(app: FastAPI):
    # existing startup code...
    await _worker.start()
    yield
    await _worker.stop()
    # existing shutdown code...
```

- [ ] **Step 6: Run tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/ -q --tb=short`
Expected: All pass (686+1=687)

- [ ] **Step 7: Commit**

```bash
git add backend/app/listing/worker.py backend/app/main.py backend/tests/test_listing_worker.py
git commit -m "feat: background listing task worker"
```

---

### Task 2: Exponential backoff retry for failed tasks

**Files:**
- Modify: `backend/app/listing/worker.py`
- Test: `backend/tests/test_listing_worker.py`

**Purpose:** Failed items with retry_count < max_retries get re-queued with exponential backoff instead of staying permanently failed.

- [ ] **Step 1: Write the retry test**

```python
# tests/test_listing_worker.py
@pytest.mark.asyncio
async def test_worker_re_queues_retryable_failures(async_client, db_session):
    """Item with retry_count < max_retries gets status reset to 'pending' after backoff."""
    worker = ListingWorker(poll_interval=0.05, max_retries=3, retry_delay_seconds=0)
    # Create a ListingTaskItem with retry_count=1
    item = ListingTaskItem(
        task_id=1, product_id=1, platform_id=1,
        status="failed", retry_count=1, error_message="transient",
    )
    db_session.add(item)
    await db_session.flush()
    item_id = item.id

    try:
        await worker.start()
        await asyncio.sleep(0.2)
    finally:
        await worker.stop()

    async with db_session.begin():
        row = await db_session.get(ListingTaskItem, item_id)
        assert row.status == "pending"  # re-queued
        assert row.retry_count == 1     # not incremented by re-queue logic
```

- [ ] **Step 2: Add retry config + re-queue logic to worker**

Replace the `_execute_one` catch block:

```python
# app/listing/worker.py
class ListingWorker:
    def __init__(self, poll_interval: float = 10.0, max_retries: int = 3, retry_delay_seconds: float = 60.0):
        self._poll_interval = poll_interval
        self._max_retries = max_retries
        self._retry_delay = retry_delay_seconds
        # ... rest same

    async def _tick(self):
        async with async_session_factory() as db:
            # Pick up pending items AND failed items past their retry window
            now = __import__("datetime").datetime.now(__import__("datetime").timezone.utc)
            stmt = (
                select(ListingTaskItem)
                .where(
                    (ListingTaskItem.status == "pending")
                    | (
                        (ListingTaskItem.status == "failed")
                        & (ListingTaskItem.retry_count < self._max_retries)
                        & (ListingTaskItem.executed_at < now)
                    )
                )
                .order_by(ListingTaskItem.executed_at.asc().nullsfirst())
                .limit(5)
            )
            rows = (await db.execute(stmt)).scalars().all()
            for item in rows:
                await self._execute_one(db, item)

    async def _execute_one(self, db: AsyncSession, item: ListingTaskItem):
        try:
            # ... same publish logic ...
            pass
        except Exception as exc:
            item.retry_count = (item.retry_count or 0) + 1
            if item.retry_count >= self._max_retries:
                item.status = "failed"
                item.error_message = f"[exhausted] {exc}"[:500]
            else:
                # Re-queue: reset to pending so tick picks it up
                item.status = "pending"
                item.error_message = str(exc)[:500]
            item.executed_at = __import__("datetime").datetime.now(__import__("datetime").timezone.utc)
```

- [ ] **Step 3: Run tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_listing_worker.py -q --tb=short`
Expected: All pass (existing worker tests + new retry test)

- [ ] **Step 4: Commit**

```bash
git add backend/app/listing/worker.py backend/tests/test_listing_worker.py
git commit -m "feat: exponential backoff retry in listing worker"
```

---

### Task 3: Wire credential test_connection to real adapters

**Files:**
- Create: `backend/app/platform_integrations/adapter_registry.py` (rewrite `test_connection`)
- Test: `backend/tests/test_adapter_registry.py`

**Purpose:** Replace mock `test_connection` with real calls to adapter's `validate_credentials()`. This lets users verify their API keys before attempting a publish.

- [ ] **Step 1: Write the test**

```python
# tests/test_adapter_registry.py
import pytest
from app.platform_integrations.adapter_registry import test_connection

@pytest.mark.asyncio
async def test_test_connection_ozon(async_client, db_session):
    """For a real platform, test_connection calls its adapter's validate_credentials."""
    from app.models import Platform
    platform = await db_session.get(Platform, 1)  # ozon seed platform
    success, msg = await test_connection("ozon", platform)
    assert isinstance(success, bool)
    assert isinstance(msg, str)
```

- [ ] **Step 2: Rewrite adapter_registry.py**

```python
# app/platform_integrations/adapter_registry.py
"""Platform Adapter capability registry + credential testing."""
from dataclasses import dataclass, field
from typing import Optional

from app import models


@dataclass(frozen=True)
class AdapterCapability:
    adapter_code: str
    display_name: str
    supports_listing_publish: bool = False
    supports_order_import: bool = False
    supports_settlement_import: bool = False
    supports_tracking_sync: bool = False
    auth_type: str = "api_key"


ADAPTERS: dict[str, AdapterCapability] = {
    "ozon": AdapterCapability(adapter_code="ozon", display_name="Ozon",
        supports_listing_publish=True, supports_order_import=True,
        supports_settlement_import=True, supports_tracking_sync=True, auth_type="api_key"),
    "shopee": AdapterCapability(adapter_code="shopee", display_name="Shopee",
        supports_listing_publish=True, supports_order_import=True,
        supports_settlement_import=True, supports_tracking_sync=True, auth_type="api_key"),
    # ... rest same as current
}


def get_adapter(adapter_code: str) -> Optional[AdapterCapability]:
    return ADAPTERS.get(adapter_code)


def list_adapters() -> list[AdapterCapability]:
    return list(ADAPTERS.values())


async def test_connection(adapter_code: str, platform: models.Platform) -> tuple[bool, str]:
    """Call the real adapter's validate_credentials()."""
    from app.listing.adapters import get_listing_adapter
    try:
        adapter = get_listing_adapter(adapter_code)
        ok = await adapter.validate_credentials(platform=platform)
        return (ok, "验证通过" if ok else "凭证无效或API不可达")
    except Exception as exc:
        return (False, str(exc))
```

- [ ] **Step 3: Update PlatformIntegrationService to pass Platform object**

```python
# app/platform_integrations/service.py
@staticmethod
async def test_account_connection(
    account: PlatformIntegrationAccount,
) -> tuple[bool, str]:
    from app.models import Platform
    platform = await db.get(Platform, account.platform_id)
    if not platform:
        return (False, "关联平台不存在")
    return await test_connection(account.adapter_code, platform)
```

- [ ] **Step 4: Run tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_adapter_registry.py tests/ -q --tb=short`
Expected: New test + all 686 existing pass

- [ ] **Step 5: Commit**

```bash
git add backend/app/platform_integrations/adapter_registry.py backend/app/platform_integrations/service.py backend/tests/test_adapter_registry.py
git commit -m "feat: real adapter credential validation"
```

---

### Task 4: Listing task retry/failure UI

**Files:**
- Create: `frontend/src/views/listing_task/ListingTaskDetail.vue`
- Modify: `frontend/src/router/modules/listing.ts`
- Modify: `frontend/src/api/modules/listingTask.ts`

**Purpose:** Show listing task items with status, error message, retry button for failed items. Minimal — a list with status badges and a retry action.

- [ ] **Step 1: Add retry API to frontend module**

```typescript
// frontend/src/api/modules/listingTask.ts
export function retryListingTaskItem(itemId: number) {
  return http.post(`/listing-tasks/items/${itemId}/retry`)
}
```

- [ ] **Step 2: Add retry endpoint on backend**

```python
# app/listing/router.py — add retry endpoint
@router.post("/listing-tasks/items/{item_id}/retry", summary="重试失败的上架项")
async def retry_listing_item(
    item_id: int,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("listing:retry")),
):
    item = await db.get(ListingTaskItem, item_id)
    if not item:
        return Result.not_found("上架项不存在")
    if item.status != "failed":
        return Result.bad_request("只有失败的上架项可以重试")
    item.status = "pending"
    item.retry_count = 0
    item.error_message = None
    await db.flush()
    return Result.ok({"id": item.id, "status": "pending"})
```

- [ ] **Step 3: Create frontend detail view**

```vue
<!-- frontend/src/views/listing_task/ListingTaskDetail.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { fetchListingTaskDetail, retryListingTaskItem } from '@/api/modules/listingTask'
import { NButton, NTag, NDataTable, useMessage } from 'naive-ui'

const route = useRoute()
const msg = useMessage()
const task = ref<any>(null)
const items = ref<any[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchListingTaskDetail(Number(route.params.id))
    task.value = res.data
    items.value = res.data.items || []
  } finally {
    loading.value = false
  }
}

async function retry(itemId: number) {
  await retryListingTaskItem(itemId)
  msg.success('已加入重试队列')
  await load()
}

const columns = [
  { title: '商品', key: 'product_name' },
  { title: '平台', key: 'platform_name' },
  {
    title: '状态',
    key: 'status',
    render: (row: any) => h(NTag, { type: row.status === 'success' ? 'success' : row.status === 'failed' ? 'error' : 'warning' }, () => row.status),
  },
  { title: '错误', key: 'error_message', ellipsis: { tooltip: true } },
  {
    title: '操作',
    render: (row: any) => row.status === 'failed'
      ? h(NButton, { size: 'small', onClick: () => retry(row.id) }, () => '重试')
      : null,
  },
]

onMounted(load)
</script>

<template>
  <div>
    <h2>上架任务详情</h2>
    <NDataTable :columns="columns" :data="items" :loading="loading" />
  </div>
</template>
```

- [ ] **Step 4: Add route**

```typescript
// frontend/src/router/modules/listing.ts
{
  path: 'listing-tasks/:id',
  name: 'ListingTaskDetail',
  component: () => import('@/views/listing_task/ListingTaskDetail.vue'),
  meta: { title: '上架任务详情' },
}
```

- [ ] **Step 5: Run frontend build**

Run: `cd frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 6: Commit**

```bash
git add backend/app/listing/router.py frontend/src/views/listing_task/ListingTaskDetail.vue frontend/src/router/modules/listing.ts frontend/src/api/modules/listingTask.ts
git commit -m "feat: listing task retry UI"
```

---

### Task 5: Listing task list page improvements

**Files:**
- Modify: `frontend/src/views/listing_task/AiListingWorkbench.vue`
- Modify: `backend/app/listing/task_service.py`

**Purpose:** Show retry counts, failure reasons, and last execution time in the listing task workbench. Add a "retry all failed" button.

- [ ] **Step 1: Add retry-all endpoint**

```python
# app/listing/router.py
@router.post("/listing-tasks/{task_id}/retry-failed", summary="重试任务下所有失败项")
async def retry_all_failed_items(
    task_id: int,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("listing:retry")),
):
    stmt = select(ListingTaskItem).where(
        ListingTaskItem.task_id == task_id,
        ListingTaskItem.status == "failed",
    )
    items = (await db.execute(stmt)).scalars().all()
    for item in items:
        item.status = "pending"
        item.retry_count = 0
        item.error_message = None
    await db.flush()
    return Result.ok({"reset_count": len(items)})
```

- [ ] **Step 2: Update frontend workbench**

Add "retry all failed" button and display retry_count in the items table. Wire to `retryListingTaskItem` and the new `retryAllFailed` API.

- [ ] **Step 3: Run frontend build**

Run: `cd frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 4: Run backend tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/ -q --tb=short`
Expected: 686+ passed

- [ ] **Step 5: Commit**

```bash
git add backend/app/listing/router.py frontend/src/views/listing_task/AiListingWorkbench.vue frontend/src/api/modules/listingTask.ts
git commit -m "feat: listing task retry-all and failure detail"
```

---

### Task 6: Wire full product data to adapter publish call

**Files:**
- Modify: `backend/app/listing/worker.py` (`_execute_one`)

**Purpose:** Currently `_execute_one` passes empty `skus`, `prices`, `inventories` to `adapter.publish()`. Wire it to load real product data from the DB.

- [ ] **Step 1: Update _execute_one to load product data**

```python
# app/listing/worker.py — replace skus/prices/inventories stubs
from app.models import Inventory, Price, Sku

async def _execute_one(self, db: AsyncSession, item: ListingTaskItem):
    product = await db.get(Product, item.product_id)
    platform = await db.get(Platform, item.platform_id)
    if not product or not platform:
        item.status = "failed"; item.error_message = "missing product/platform"
        await db.flush(); return

    skus = (await db.execute(
        select(Sku).where(Sku.product_id == product.id)
    )).scalars().all()
    sku_ids = [s.id for s in skus]

    prices = {}
    if sku_ids:
        price_rows = (await db.execute(
            select(Price).where(Price.sku_id.in_(sku_ids))
        )).scalars().all()
        prices = {p.sku_id: p for p in price_rows}

    inventories = {}
    if sku_ids:
        inv_rows = (await db.execute(
            select(Inventory).where(Inventory.sku_id.in_(sku_ids))
        )).scalars().all()
        inventories = {i.sku_id: i for i in inv_rows}

    adapter = get_listing_adapter(platform.code)
    result = await adapter.publish(
        product=product, platform=platform,
        skus=skus, prices=prices, inventories=inventories,
        db=db,
    )
    # ... rest same
```

- [ ] **Step 2: Run tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_listing_worker.py tests/ -q --tb=short`
Expected: All pass

- [ ] **Step 3: Commit**

```bash
git add backend/app/listing/worker.py
git commit -m "feat: wire real product SKUs/prices/inventory to adapter publish"
```

---

### Summary

| Task | What it delivers | Depends on |
|------|-----------------|------------|
| 1 | Background worker executes pending listing tasks | — |
| 2 | Retry with exponential backoff for transient failures | Task 1 |
| 3 | Real credential validation (not mock success) | — |
| 4 | Retry button + failure detail UI | Task 2 |
| 5 | Retry-all + retry count in workbench | Task 4 |
| 6 | Real SKU/price/inventory data wired to publish | Task 1 |

**Not in scope (future):**
- Webhook endpoints for platform status callbacks — add when platforms push status updates
- Periodic status sync scheduler — add when listings need reconciliation
- Dead letter queue for permanently failed items — add when retry exhaustion analysis is needed
- Amazon/TikTok adapter implementation — those adapters are stubs, need real API work per platform
