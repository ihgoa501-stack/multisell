"""Test that inventory changes trigger platform sync."""

import asyncio
from unittest.mock import AsyncMock, patch

import pytest

from app.database import async_session_factory
from app.inventory.service import InventoryService


@pytest.fixture(autouse=True)
def _mock_sync_service():
    """Prevent actual platform adapter calls during tests.

    Patch at the import site used by service.py so the fire-and-forget
    task calls the mock instead.
    """
    with patch(
        "app.inventory.service.sync_inventory_to_platforms",
        new_callable=AsyncMock,
    ) as mock:
        yield mock


async def _create_test_product(async_client):
    """Create a product + SKU and return product info + sku."""
    resp = await async_client.post(
        "/api/products",
        json={"name": "sync-test-product", "unit": "件", "status": 1},
    )
    assert resp.status_code == 200
    product = resp.json()["data"]

    spec_resp = await async_client.post(
        f"/api/products/{product['id']}/specs",
        json={"specs": [{"name": "颜色", "values": ["红"]}]},
    )
    assert spec_resp.status_code == 200

    sku_resp = await async_client.post(f"/api/products/{product['id']}/skus/generate")
    assert sku_resp.status_code == 200
    sku = sku_resp.json()["data"]["skus"][0]

    return {"product": product, "sku": sku}


class TestInventorySync:
    """Inventory sync hooks fire on stock changes."""

    @pytest.mark.asyncio
    async def test_update_inventory_triggers_sync(
        self, async_client, _mock_sync_service
    ):
        """When update_inventory is called via API, the sync task is enqueued."""
        data = await _create_test_product(async_client)
        sku_id = data["sku"]["id"]

        resp = await async_client.put(
            f"/api/inventory/{sku_id}",
            json={"quantity": 50, "remark": "test sync"},
        )
        assert resp.status_code == 200

        # Yield control so asyncio.create_task executes the mock
        await asyncio.sleep(0.2)

        assert _mock_sync_service.await_count >= 1, (
            f"Expected sync_inventory_to_platforms to be called, "
            f"got {_mock_sync_service.await_count}"
        )
        call = _mock_sync_service.await_args
        assert call is not None
        _, called_sku_id, _, called_qty = call.args
        assert called_sku_id == sku_id
        assert called_qty == 50

    @pytest.mark.asyncio
    async def test_release_locked_stock_triggers_sync(
        self, async_client, _mock_sync_service
    ):
        """release_locked_stock enqueues a sync."""
        data = await _create_test_product(async_client)
        sku_id = data["sku"]["id"]

        await async_client.put(
            f"/api/inventory/{sku_id}",
            json={"quantity": 100},
        )
        await asyncio.sleep(0.1)

        # Reset mock after the update_inventory call above
        _mock_sync_service.reset_mock()

        async with async_session_factory() as db:
            try:
                await InventoryService.lock_stock(db, sku_id, 30, "test-order-rel")
                await db.commit()
            except Exception:
                await db.rollback()
                raise

        await asyncio.sleep(0.1)

        async with async_session_factory() as db:
            try:
                await InventoryService.release_locked_stock(
                    db, sku_id, 10, "test-order-rel"
                )
                await db.commit()
            except Exception:
                await db.rollback()
                raise

        await asyncio.sleep(0.2)

        assert _mock_sync_service.await_count >= 1, (
            "Expected sync after release_locked_stock"
        )

    @pytest.mark.asyncio
    async def test_confirm_deduction_triggers_sync(
        self, async_client, _mock_sync_service
    ):
        """confirm_locked_stock_deduction enqueues a sync."""
        data = await _create_test_product(async_client)
        sku_id = data["sku"]["id"]

        await async_client.put(
            f"/api/inventory/{sku_id}",
            json={"quantity": 100},
        )
        await asyncio.sleep(0.1)
        _mock_sync_service.reset_mock()

        async with async_session_factory() as db:
            try:
                await InventoryService.lock_stock(db, sku_id, 20, "test-order-ded")
                await db.commit()
            except Exception:
                await db.rollback()
                raise

        await asyncio.sleep(0.1)

        async with async_session_factory() as db:
            try:
                await InventoryService.confirm_locked_stock_deduction(
                    db, sku_id, 20, "test-order-ded"
                )
                await db.commit()
            except Exception:
                await db.rollback()
                raise

        await asyncio.sleep(0.2)

        assert _mock_sync_service.await_count >= 1, (
            "Expected sync after confirm_locked_stock_deduction"
        )

    @pytest.mark.asyncio
    async def test_inventory_read_does_not_trigger_sync(
        self, async_client, _mock_sync_service
    ):
        """GET inventory should NOT trigger sync."""
        data = await _create_test_product(async_client)
        sku_id = data["sku"]["id"]

        await async_client.put(
            f"/api/inventory/{sku_id}",
            json={"quantity": 30},
        )
        await asyncio.sleep(0.1)
        _mock_sync_service.reset_mock()

        resp = await async_client.get(f"/api/inventory/{sku_id}")
        assert resp.status_code == 200
        await asyncio.sleep(0.1)

        assert _mock_sync_service.await_count == 0, (
            "GET inventory should not trigger sync"
        )
