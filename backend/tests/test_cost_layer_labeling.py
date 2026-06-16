"""成本层标识测试。"""

from uuid import uuid4

import pytest


async def _setup_data(async_client) -> tuple[int, int, int]:
    """Create product+SKU+platform for decision tests, return (sku_id, platform_id, product_id)"""
    uid = uuid4().hex[:6]
    resp = await async_client.post("/api/products", json={
        "name": f"CL_{uid}",
        "package_length_cm": 30, "package_width_cm": 20,
        "package_height_cm": 10, "package_weight_kg": 0.5,
    })
    pid = resp.json()["data"]["id"]
    await async_client.post(f"/api/products/{pid}/specs", json={"specs": [{"name": "颜色", "values": ["标准"]}]})
    sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    sku_id = sku_resp.json()["data"]["skus"][0]["id"]
    await async_client.put(f"/api/skus/{sku_id}", json={"cost_price": 300})
    plat_resp = await async_client.post("/api/platforms", json={"name": f"PL_{uid}", "code": f"pl_{uid}"})
    plat_id = plat_resp.json()["data"]["id"]
    return sku_id, plat_id, pid


class TestDecisionCostLayer:
    """决策 API 返回 cost_layer"""

    async def test_prelisting_decision_returns_estimated_cost_layers(self, async_client):
        sku_id, plat_id, _ = await _setup_data(async_client)

        resp = await async_client.post(
            "/api/decisions/prelisting",
            json={
                "sku_id": sku_id,
                "destination_country": "RU",
                "target_sale_price": 5000,
                "platform_fee_pct": 10,
                "payment_fee_pct": 3,
                "other_fee": 100,
                "minimum_margin_pct": 20,
                "cargo_type": "normal",
            },
        )
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["shipping_cost_layer"] == "estimated"
        assert data["platform_fee_cost_layer"] == "estimated"
        assert data["profit_cost_layer"] == "estimated"

    async def test_batch_decision_returns_cost_layers(self, async_client):
        sku_id, plat_id, _ = await _setup_data(async_client)

        resp = await async_client.post(
            "/api/decisions/prelisting/batch",
            json={
                "items": [{
                    "item_key": "row-1",
                    "sku_id": sku_id,
                    "destination_country": "RU",
                    "target_sale_price": 5000,
                    "platform_fee_pct": 10,
                    "payment_fee_pct": 3,
                    "other_fee": 100,
                    "minimum_margin_pct": 20,
                    "cargo_type": "normal",
                }],
            },
        )
        assert resp.status_code == 200
        data = resp.json()["data"]
        item = data["items"][0]
        assert item["result"]["shipping_cost_layer"] == "estimated"
        assert item["result"]["platform_fee_cost_layer"] == "estimated"


class TestOrderCostLayer:
    """订单利润 API 返回 cost_layer"""

    async def test_order_without_snapshot_shows_estimated_shipping_layer(self, async_client):
        """无运费快照的订单 shipping_cost_layer = estimated"""
        uid = uuid4().hex[:6]
        resp = await async_client.post("/api/products", json={"name": f"OE_{uid}"})
        pid = resp.json()["data"]["id"]
        await async_client.post(f"/api/products/{pid}/specs", json={"specs": [{"name": "颜色", "values": ["标准"]}]})
        sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]
        await async_client.post("/api/prices", json={"sku_id": sku_id, "price_type": "sale_price", "price": 50})
        await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 10})

        resp = await async_client.post(
            "/api/orders",
            json={
                "order_no": f"ORD-{uid}",
                "items": [{"sku_id": sku_id, "quantity": 1, "unit_price": 100}],
                "recipient_name": "Test",
                "shipping_address": "RU Moscow",
                "shipping_fee": 30,
                "platform_fee": 10,
            },
        )
        order_id = resp.json()["data"]["id"]

        detail = await async_client.get(f"/api/orders/{order_id}")
        profit = detail.json()["data"]["profit"]
        assert profit["shipping_cost_layer"] == "estimated"
        assert profit["platform_fee_cost_layer"] == "estimated"
        assert profit["profit_cost_layer"] == "estimated"

    async def test_order_with_snapshot_shows_snapshot_shipping_layer(self, async_client):
        """有运费快照的订单 shipping_cost_layer = snapshot"""
        uid = uuid4().hex[:6]
        resp = await async_client.post("/api/products", json={
            "name": f"OS_{uid}",
            "main_image": "/img.jpg",
            "package_length_cm": 30, "package_width_cm": 20,
            "package_height_cm": 10, "package_weight_kg": 0.5,
        })
        pid = resp.json()["data"]["id"]
        await async_client.post(f"/api/products/{pid}/specs", json={"specs": [{"name": "颜色", "values": ["标准"]}]})
        sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]
        await async_client.post("/api/prices", json={"sku_id": sku_id, "price_type": "sale_price", "price": 50})
        await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 10})

        pv_resp = await async_client.post("/api/shipping/providers", json={"name": f"PV_{uid}", "code": f"pv_{uid}"})
        pv_id = pv_resp.json()["data"]["id"]
        ch_resp = await async_client.post("/api/shipping/channels", json={
            "provider_id": pv_id, "name": f"CH_{uid}", "code": f"ch_{uid}", "cargo_types": ["normal"],
        })
        ch_id = ch_resp.json()["data"]["id"]
        await async_client.post(f"/api/shipping/channels/{ch_id}/zones", json={"country_code": "RU"})
        await async_client.post(f"/api/shipping/channels/{ch_id}/rules", json={
            "rule_type": "fixed_plus_per_kg", "fixed_fee": 10, "per_kg_price": 20,
        })

        order_resp = await async_client.post(
            "/api/orders",
            json={
                "order_no": f"ORD-SNAP-{uid}",
                "items": [{"sku_id": sku_id, "quantity": 1, "unit_price": 100}],
                "recipient_name": "Test",
                "shipping_address": "RU Moscow",
                "shipping_fee": 0,
            },
        )
        order_id = order_resp.json()["data"]["id"]

        # bind shipping quote -> creates snapshot
        await async_client.post(
            f"/api/orders/{order_id}/shipping-quote",
            json={
                "sku_id": sku_id,
                "quantity": 1,
                "destination_country": "RU",
                "cargo_type": "normal",
            },
        )

        detail = await async_client.get(f"/api/orders/{order_id}")
        profit = detail.json()["data"]["profit"]
        assert profit["shipping_cost_layer"] == "snapshot"
        assert profit["profit_cost_layer"] == "mixed"  # shipping=snapshot, platform_fee=estimated


class TestShippingBillCostLayer:
    """运费账单行返回 cost_layer = actual"""

    async def test_bill_item_returns_actual_cost_layer(self, async_client):
        import csv
        import io

        output = io.StringIO()
        w = csv.writer(output)
        w.writerow(["运单号", "物流商", "实际运费"])
        w.writerow(["TRK-CL-001", "P", "50"])
        content = output.getvalue().encode("utf-8-sig")

        import_resp = await async_client.post(
            "/api/shipping/bills/import",
            files={"file": ("test.csv", content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]

        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        items = list_resp.json()["data"]
        assert len(items) == 1
        assert items[0]["actual_shipping_cost_layer"] == "actual"
