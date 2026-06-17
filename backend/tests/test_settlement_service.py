"""结算模块 — 功能测试"""

import pytest
import uuid


def _ok(resp):
    assert resp.status_code == 200, f"HTTP {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body.get("code") == 200, f"业务错误: {body}"
    return body.get("data")


async def _ensure_order(async_client):
    """确保有创建的订单可生成结算"""
    # 创建平台（使用唯一 code 避免跨测试冲突）
    code = f"st{uuid.uuid4().hex[:8]}"
    r = await async_client.post("/api/platforms", json={
        "name": f"Settle-{code}", "code": code, "api_key": "k",
    })
    pid = _ok(r)["id"]

    # 创建商品
    r = await async_client.post("/api/products", json={
        "name": "SettleProd", "unit": "件", "status": 1,
        "package_length_cm": 10, "package_width_cm": 10, "package_height_cm": 10,
        "package_weight_kg": 0.5, "cargo_type": "normal",
        "main_image": "https://ex.com/i.jpg",
    })
    prod = _ok(r)

    # SKU
    r = await async_client.post(f"/api/products/{prod['id']}/specs", json={
        "specs": [{"name": "C", "values": ["A"]}],
    })
    _ok(r)
    r = await async_client.post(f"/api/products/{prod['id']}/skus/generate")
    sku = _ok(r)["skus"][0]

    r = await async_client.post("/api/prices", json={
        "sku_id": sku["id"], "price_type": "sale_price", "price": 100,
    })
    _ok(r)
    r = await async_client.put(f"/api/inventory/{sku['id']}", json={"quantity": 50})
    _ok(r)

    # 订单
    r = await async_client.post("/api/orders", json={
        "recipient_name": "T", "recipient_phone": "13800000000",
        "shipping_address": "Addr", "payment_method": "card",
        "shipping_fee": 10,
        "items": [{"sku_id": sku["id"], "quantity": 1}],
    })
    order = _ok(r)
    r = await async_client.put(f"/api/orders/{order['id']}/status", json={"status": "paid"})
    _ok(r)

    return pid


class TestSettlementAPI:

    async def test_generate_and_list_settlement(self, async_client):
        pid = await _ensure_order(async_client)
        r = await async_client.post(f"/api/settlements/mock?platform_id={pid}&count=2")
        s = _ok(r)
        assert s["id"] > 0
        assert float(s["total_revenue"]) > 0

        r = await async_client.get("/api/settlements")
        body = r.json()
        records = body.get("data", {}).get("records") or body.get("records", [])
        assert len(records) >= 1

    async def test_settlement_detail_and_items(self, async_client):
        pid = await _ensure_order(async_client)
        r = await async_client.post(f"/api/settlements/mock?platform_id={pid}&count=1")
        s = _ok(r)

        r = await async_client.get(f"/api/settlements/{s['id']}")
        detail = _ok(r)
        assert detail["item_count"] >= 1

        r = await async_client.get(f"/api/settlements/{s['id']}/items")
        body = r.json()
        items = body.get("data", {}).get("records") or body.get("records", [])
        assert len(items) >= 1

    async def test_settlement_reconcile(self, async_client):
        pid = await _ensure_order(async_client)
        r = await async_client.post(f"/api/settlements/mock?platform_id={pid}&count=1")
        s = _ok(r)

        r = await async_client.post(
            f"/api/settlements/{s['id']}/reconcile",
            json={"auto_match": True, "strategy": "by_order_no"},
        )
        result = _ok(r)
        assert result["settlement_id"] == s["id"]
        assert result["total"] >= 1

    async def test_settlement_delete(self, async_client):
        pid = await _ensure_order(async_client)
        r = await async_client.post(f"/api/settlements/mock?platform_id={pid}&count=1")
        s = _ok(r)

        r = await async_client.delete(f"/api/settlements/{s['id']}")
        assert r.json()["code"] == 200

        r = await async_client.get(f"/api/settlements/{s['id']}")
        assert r.json()["code"] == 404
