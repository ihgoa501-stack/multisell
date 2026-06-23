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
        payload.update(
            {
                "package_length_cm": 30,
                "package_width_cm": 20,
                "package_height_cm": 10,
                "package_weight_kg": 0.8,
                "cargo_type": "normal",
            }
        )
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
    sku_id = resp.json()["data"]["skus"][0]["id"]
    # 设置库存
    inv_resp = await async_client.put(
        f"/api/inventory/{sku_id}",
        json={"quantity": 50},
    )
    assert inv_resp.status_code == 200, inv_resp.text
    return sku_id


async def _seed_shipping_channel(async_client, country: str = "US") -> dict:
    resp = await async_client.post(
        "/api/shipping/providers",
        json={
            "name": f"订单快照物流商-{uuid4().hex[:6]}",
            "code": _uc("order_provider"),
        },
    )
    assert resp.status_code == 200, resp.text
    provider_id = resp.json()["data"]["id"]
    resp = await async_client.post(
        "/api/shipping/channels",
        json={
            "provider_id": provider_id,
            "name": f"订单快照渠道-{uuid4().hex[:6]}",
            "code": _uc("order_channel"),
            "volumetric_divisor": 6000,
            "cargo_types": ["normal"],
            "currency": "CNY",
        },
    )
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
    resp = await async_client.post(
        "/api/orders",
        json={
            "recipient_name": "订单运费测试",
            "recipient_phone": "13900000000",
            "shipping_address": "测试地址",
            "payment_method": "mock",
            "items": [{"sku_id": sku_id, "quantity": quantity, "unit_price": 120}],
            "platform_fee": 12,
            "payment_fee": 3,
            "other_fee": 5,
        },
    )
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]


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


async def test_shipping_snapshot_does_not_change_when_rule_changes(async_client):
    sku_id = await _seed_product_sku(async_client)
    seeded = await _seed_shipping_channel(async_client, "US")
    order = await _create_order(async_client, sku_id, quantity=1)

    bind_resp = await async_client.post(
        f"/api/orders/{order['id']}/shipping-quote",
        json={
            "sku_id": sku_id,
            "quantity": 1,
            "destination_country": "US",
            "cargo_type": "normal",
        },
    )
    assert bind_resp.status_code == 200, bind_resp.text
    assert bind_resp.json()["data"]["shipping_snapshot"]["total_shipping_fee"] == 50.0

    rules_resp = await async_client.get(
        f"/api/shipping/channels/{seeded['channel_id']}/rules"
    )
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


async def test_bind_shipping_quote_requires_complete_package_data(async_client):
    sku_id = await _seed_product_sku(async_client, with_package=False)
    await _seed_shipping_channel(async_client, "US")
    order = await _create_order(async_client, sku_id, quantity=1)

    resp = await async_client.post(
        f"/api/orders/{order['id']}/shipping-quote",
        json={
            "sku_id": sku_id,
            "quantity": 1,
            "destination_country": "US",
            "cargo_type": "normal",
        },
    )

    assert resp.status_code == 200
    body = resp.json()
    assert body["code"] == 400
    assert "物流数据不完整" in body["message"]


async def test_update_profit_inputs_recalculates_profit(async_client):
    sku_id = await _seed_product_sku(async_client)
    await _seed_shipping_channel(async_client, "US")
    order = await _create_order(async_client, sku_id, quantity=1)
    await async_client.post(
        f"/api/orders/{order['id']}/shipping-quote",
        json={
            "sku_id": sku_id,
            "quantity": 1,
            "destination_country": "US",
            "cargo_type": "normal",
        },
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
