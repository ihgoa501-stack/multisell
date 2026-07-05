> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# P2: 物流追踪回传 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When an order is shipped (tracking number assigned locally), automatically push the tracking number + carrier code back to the platform.

**Architecture:** Add `push_tracking()` to adapter protocol. Hook into `OrderService.update_status` when status transitions to `"shipped"`. If the order has a `tracking_number` and originated from a platform, push via the adapter.

**Tech Stack:** Same adapters, `OrderService.update_status` hook, existing `shipping_provider` model for carrier codes.

**Current state:** Orders have `tracking_number` (varchar). `OrderService.update_status()` sets `shipped_at` and `tracking_number`. No tracking push happens.

---

### Task 1: Add `push_tracking()` to adapter protocol + Ozon implementation

**Files:**
- Modify: `backend/app/listing/adapters/base.py`
- Modify: `backend/app/listing/adapters/ozon.py`
- Test: `backend/tests/test_listing_adapters.py`

**Ozon API:** `POST /v3/posting/fbs/ship` — send posting numbers + tracking info.

- [ ] **Step 1: Add protocol method**

```python
# base.py
async def push_tracking(
    self,
    *,
    platform: Platform,
    order_sn: str,
    tracking_number: str,
    carrier_code: str = "",
    db: Optional[AsyncSession] = None,
) -> bool:
    """将物流追踪号推回平台。"""
```

- [ ] **Step 2: Implement Ozon**

```python
# ozon.py
async def push_tracking(self, *, platform: Platform, order_sn: str,
                        tracking_number: str, carrier_code: str = "",
                        db: Optional[AsyncSession] = None) -> bool:
    limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
    await limiter.acquire()
    payload = {
        "posting_number": order_sn,
        "tracking_number": tracking_number,
    }
    if carrier_code:
        payload["carrier_code"] = carrier_code
    async with self._client(platform) as client:
        resp = await client.post("/v3/posting/fbs/ship", json=payload)
        body = self._parse_response(resp, "push_tracking")
    return body.get("result", False)
```

- [ ] **Step 3: Implement Shopee**

```python
# shopee.py — NOT YET: Shopee requires integrated logistics.
# Return True (no-op) if platform is not using Shopee's logistics.
```

- [ ] **Step 4: Commit**

```bash
git add backend/app/listing/adapters/base.py backend/app/listing/adapters/ozon.py
git commit -m "feat: push_tracking adapter method + Ozon impl"
```

---

### Task 2: Hook into order shipping flow

**Files:**
- Modify: `backend/app/order/service.py`

- [ ] **Step 1: Add tracking push hook in `update_status`**

```python
# order/service.py — inside update_status, after setting shipped_at:
if status == "shipped" and order.tracking_number:
    # Try to push tracking number back to platform
    # order_no format: platform-specific prefix, e.g. "OZON-..." or "SHOPEE-..."
    for platform_code in ("ozon", "shopee", "wb"):
        if order.order_no.upper().startswith(platform_code.upper()):
            try:
                from app.listing.adapters import get_listing_adapter
                adapter = get_listing_adapter(platform_code)
                platform = await db.get(Platform, order.platform_id)
                if platform and hasattr(adapter, "push_tracking"):
                    await adapter.push_tracking(
                        platform=platform,
                        order_sn=order.order_no,
                        tracking_number=order.tracking_number,
                    )
            except Exception:
                logger.warning("Failed to push tracking for %s", order.order_no)
            break
```

- [ ] **Step 2: Run tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_order.py -q --tb=short`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/app/order/service.py
git commit -m "feat: push tracking number on order shipped"
```
