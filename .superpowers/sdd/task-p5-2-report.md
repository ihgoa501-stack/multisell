# P5 Task 2 — Report

**Task:** Hook local inventory changes to auto-sync back to platforms.

## Files Changed

- **Created:** `backend/app/inventory/sync_service.py`
  - `sync_inventory_to_platforms(db, sku_id, sku_code, quantity)` — looks up `ProductListing` via the SKU's `product_id`, resolves the platform adapter for each listing, and calls `adapter.sync_inventory()`.

- **Modified:** `backend/app/inventory/service.py`
  - Added module-level `_enqueue_inventory_sync(sku_id, quantity)` helper that spawns an `asyncio.create_task` with its own database session to call `sync_inventory_to_platforms`.
  - Hook added to `update_inventory()` — fires after the inventory log is written.
  - Hook added to `release_locked_stock()` — fires after the locked stock is released.
  - Hook added to `confirm_locked_stock_deduction()` — fires after the deduction is committed.

- **Created:** `tests/test_inventory_sync.py`
  - 4 tests covering all three mutation methods + a read-only sanity check.
  - Uses `@patch("app.inventory.service.sync_inventory_to_platforms")` to verify the hook fires with correct arguments.

## Design Notes

- Fire-and-forget via `asyncio.create_task` (as specified in the brief). The task creates its own `async_session_factory()` session to avoid interfering with the request transaction.
- The sync function queries `ProductListing` by the SKU's `product_id` (ProductListing has no direct `sku_id` column; it maps at the product level).

## Verification

```
tests/test_inventory_sync.py ....                                         [100%]
4 passed in 2.22s
```

All pre-existing test failures (`enable_auth` fixture missing) are unrelated to this change.
