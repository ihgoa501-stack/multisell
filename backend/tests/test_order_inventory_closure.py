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
    resp = await async_client.post(
        "/api/products",
        json={
            "name": f"库存闭环商品-{uuid4().hex[:8]}",
            "unit": "件",
            "status": 1,
            "package_length_cm": 10,
            "package_width_cm": 10,
            "package_height_cm": 10,
            "package_weight_kg": 0.5,
            "cargo_type": "normal",
        },
    )
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
    resp = await async_client.post(
        "/api/orders",
        json={
            "recipient_name": "库存闭环测试",
            "recipient_phone": "13900000000",
            "shipping_address": "测试地址",
            "payment_method": "mock",
            "items": [{"sku_id": sku_id, "quantity": quantity, "unit_price": 100}],
        },
    )
    return resp


class TestOrderInventoryClosure:
    async def test_create_order_locks_available_stock(self, async_client):
        """创建订单锁定可用库存"""
        sku_id = await _create_sku(async_client)
        await _set_inventory(async_client, sku_id, 10)

        resp = await _create_order(async_client, sku_id, quantity=3)

        assert resp.status_code == 200, resp.text
        assert resp.json()["code"] == 200
        inv = await _get_inventory(async_client, sku_id)
        assert inv["quantity"] == 10
        assert inv["locked_quantity"] == 3
        assert inv["available_quantity"] == 7

    async def test_create_order_blocks_when_available_stock_is_insufficient(
        self, async_client
    ):
        """库存不足时阻止创建订单"""
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

    async def test_paid_status_deducts_locked_stock(self, async_client):
        """支付状态扣减库存并释放锁定"""
        sku_id = await _create_sku(async_client)
        await _set_inventory(async_client, sku_id, 10)
        order_resp = await _create_order(async_client, sku_id, quantity=3)
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/orders/{order_id}/status", json={"status": "paid"}
        )

        assert resp.status_code == 200, resp.text
        inv = await _get_inventory(async_client, sku_id)
        assert inv["quantity"] == 7
        assert inv["locked_quantity"] == 0
        assert inv["available_quantity"] == 7

    async def test_pending_cancel_releases_locked_stock(self, async_client):
        """待支付取消释放锁定库存"""
        sku_id = await _create_sku(async_client)
        await _set_inventory(async_client, sku_id, 10)
        order_resp = await _create_order(async_client, sku_id, quantity=3)
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/orders/{order_id}/status", json={"status": "cancelled"}
        )

        assert resp.status_code == 200, resp.text
        inv = await _get_inventory(async_client, sku_id)
        assert inv["quantity"] == 10
        assert inv["locked_quantity"] == 0
        assert inv["available_quantity"] == 10

    async def test_order_stock_movements_write_inventory_logs(self, async_client):
        """库存变动写入 InventoryLog"""
        sku_id = await _create_sku(async_client)
        await _set_inventory(async_client, sku_id, 10)
        order_resp = await _create_order(async_client, sku_id, quantity=2)
        order_no = order_resp.json()["data"]["order_no"]
        order_id = order_resp.json()["data"]["id"]
        await async_client.put(
            f"/api/orders/{order_id}/status", json={"status": "paid"}
        )

        logs_resp = await async_client.get(f"/api/inventory/{sku_id}/logs")

        assert logs_resp.status_code == 200
        logs = logs_resp.json()["data"]
        change_types = [log["change_type"] for log in logs]
        assert "lock" in change_types
        assert "deduct" in change_types
        assert any(order_no in (log["remark"] or "") for log in logs)
