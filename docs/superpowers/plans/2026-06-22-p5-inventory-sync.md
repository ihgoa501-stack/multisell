> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# P5: 库存回写 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When local inventory changes (via `InventoryService.update_inventory` or `adjust_stock`), sync the new quantity back to all platforms where the SKU is listed.

**Architecture:** Add `sync_inventory()` to adapters. Hook into `InventoryService` mutation methods to enqueue inventory sync tasks. Process via a lightweight worker that calls each platform's stock update API. Track sync state on `ProductListing` model.

**Tech Stack:** Same adapters, existing `InventoryService`, `ProductListing` model (has `platform_id`, `sku_id`, `platform_sku`), existing `rate_limiter`.

**Current state:** `InventoryService.update_inventory()` writes local changes. `ProductListing` links local SKUs to platform SKUs. No sync back to platforms.

---

### Task 1: Add `sync_inventory()` to adapters + implementations

**Files:**
- Modify: `backend/app/listing/adapters/base.py`
- Modify: `backend/app/listing/adapters/ozon.py`
- Modify: `backend/app/listing/adapters/shopee.py`
- Test: `backend/tests/test_listing_adapters.py`

**Ozon API:** `POST /v4/product/import/stocks` — accepts list of `{"sku": "…", "stock": N}`.

**Shopee API:** `POST /api/v2/product/update_stock` — updates stock per model_id.

- [ ] **Step 1: Add protocol method**

```python
# base.py
async def sync_inventory(
    self,
    *,
    platform: Platform,
    sku_code: str,
    platform_sku: str,
    quantity: int,
    db: Optional[AsyncSession] = None,
) -> bool:
    """将本地库存数量同步到平台。
    - sku_code: 本地SKU编码
    - platform_sku: 平台SKU ID (from ProductListing.platform_sku)
    - quantity: 新的库存数量
    """
```

- [ ] **Step 2: Implement Ozon**

```python
# ozon.py
async def sync_inventory(self, *, platform: Platform, sku_code: str,
                         platform_sku: str, quantity: int,
                         db: Optional[AsyncSession] = None) -> bool:
    limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
    await limiter.acquire()
    payload = {"stocks": [{"sku": platform_sku or sku_code, "stock": quantity}]}
    async with self._client(platform) as client:
        resp = await client.post("/v4/product/import/stocks", json=payload)
        body = self._parse_response(resp, "sync_inventory")
    return body.get("result", False)
```

- [ ] **Step 3: Implement Shopee**

```python
# shopee.py
async def sync_inventory(self, *, platform: Platform, sku_code: str,
                         platform_sku: str, quantity: int,
                         db: Optional[AsyncSession] = None) -> bool:
    extra = platform.extra_config or {}
    api_path = "/api/v2/product/update_stock"
    params = self._build_auth_params(platform, api_path)
    params["model_id"] = int(platform_sku) if platform_sku.isdigit() else 0
    params["stock"] = quantity
    async with self._client(platform) as client:
        resp = await client.post(api_path, params=params)
        body = self._parse_response(resp, "sync_inventory")
    return body.get("response", {}).get("warning") is None
```

- [ ] **Step 4: Commit**

```bash
git add backend/app/listing/adapters/base.py backend/app/listing/adapters/ozon.py backend/app/listing/adapters/shopee.py
git commit -m "feat: sync_inventory adapter method + Ozon/Shopee impl"
```

---

### Task 2: Hook into InventoryService

**Files:**
- Modify: `backend/app/inventory/service.py`
- Create: `backend/app/inventory/sync_service.py`

- [ ] **Step 1: Create inventory sync enqueue function**

```python
# backend/app/inventory/sync_service.py
"""Inventory sync — after local stock changes, push to all listed platforms."""
import asyncio, logging
from sqlalchemy import select
from app.models import ProductListing, Platform

logger = logging.getLogger(__name__)


async def sync_inventory_to_platforms(db, sku_id: int, sku_code: str, quantity: int):
    """Push new quantity to all platforms where this SKU is listed."""
    listings = (await db.execute(
        select(ProductListing, Platform)
        .join(Platform, ProductListing.platform_id == Platform.id)
        .where(ProductListing.sku_id == sku_id)
    )).all()

    for listing, platform in listings:
        adapter = get_listing_adapter(platform.code)
        if not hasattr(adapter, "sync_inventory"):
            continue
        try:
            ok = await adapter.sync_inventory(
                platform=platform,
                sku_code=sku_code,
                platform_sku=listing.platform_sku or "",
                quantity=quantity,
            )
            if ok:
                logger.info("Inventory synced for SKU %s on %s", sku_code, platform.code)
            else:
                logger.warning("Inventory sync failed for SKU %s on %s", sku_code, platform.code)
        except Exception:
            logger.exception("Inventory sync error for SKU %s on %s", sku_code, platform.code)
```

- [ ] **Step 2: Hook into `update_inventory`**

```python
# inventory/service.py — at end of update_inventory()
try:
    sku = await db.get(Sku, sku_id)
    sku_code = sku.code if sku else str(sku_id)
    asyncio.create_task(sync_inventory_to_platforms(
        async_session_factory(), sku_id, sku_code, quantity
    ))
except Exception:
    logger.exception("Failed to enqueue inventory sync")
```

**ponytail:** Fire-and-forget via `asyncio.create_task`. If reliability matters, use a proper task queue.

- [ ] **Step 3: Run tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/ -q --tb=short`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/app/inventory/sync_service.py backend/app/inventory/service.py
git commit -m "feat: auto-sync inventory to platforms on stock change"
```
