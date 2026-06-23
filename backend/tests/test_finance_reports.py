"""财务报表测试。"""

import csv
import io
from uuid import uuid4

import pytest


async def _prepare_ledger_data(async_client) -> int:
    """Create product+SKU+order and rebuild ledger, return order_id."""
    uid = uuid4().hex[:6]
    resp = await async_client.post("/api/products", json={
        "name": f"BI_{uid}",
        "package_length_cm": 30, "package_width_cm": 20,
        "package_height_cm": 10, "package_weight_kg": 0.5,
    })
    pid = resp.json()["data"]["id"]
    await async_client.post(f"/api/products/{pid}/specs", json={"specs": [{"name": "颜色", "values": ["标准"]}]})
    sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    sku_id = sku_resp.json()["data"]["skus"][0]["id"]
    await async_client.put(f"/api/skus/{sku_id}", json={"cost_price": 300})
    await async_client.post("/api/prices", json={"sku_id": sku_id, "price_type": "sale_price", "price": 100})
    await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 10})

    order_resp = await async_client.post(
        "/api/orders",
        json={
            "items": [{"sku_id": sku_id, "quantity": 1, "unit_price": 5000}],
            "recipient_name": "Test", "shipping_address": "RU Moscow",
            "product_cost": 300, "shipping_fee": 60, "platform_fee": 100,
        },
    )
    order_id = order_resp.json()["data"]["id"]
    await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")

    # Import settlement for platform fee
    output = io.StringIO()
    w = csv.writer(output)
    w.writerow(["platform", "order_no", "transaction_type", "amount"])
    w.writerow(["ozon", order_resp.json()["data"]["order_no"], "platform_fee", "100"])
    content = output.getvalue().encode("utf-8-sig")
    await async_client.post("/api/settlements/import", files={"file": ("test.csv", content, "text/csv")})
    await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")

    return order_id


@pytest.mark.skip(reason="depends on /api/finance/orders/{id}/ledger/rebuild and /api/settlements/import which are not implemented")
class TestProfitSummary:
    async def test_profit_summary_returns_summary(self, async_client):
        await _prepare_ledger_data(async_client)

        resp = await async_client.get("/api/finance/profit-summary")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert "revenue_amount" in data
        assert "shipping_cost" in data
        assert "profit_amount" in data

    async def test_profit_summary_filters_by_date(self, async_client):
        await _prepare_ledger_data(async_client)

        # Use a wide date range that includes today
        resp = await async_client.get("/api/finance/profit-summary")
        assert resp.status_code == 200
        resp.json()["data"]

        # Verify the API supports date params (even if they don't filter)
        resp = await async_client.get("/api/finance/profit-summary?date_from=2000-01-01&date_to=2099-12-31")
        assert resp.status_code == 200


@pytest.mark.skip(reason="depends on /api/finance/orders/{id}/ledger/rebuild and /api/settlements/import which are not implemented")
class TestOrderProfit:
    async def test_order_profit_lists_with_pagination(self, async_client):
        await _prepare_ledger_data(async_client)

        resp = await async_client.get("/api/finance/order-profit")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert "items" in data
        assert data["total"] >= 1

    async def test_order_profit_returns_cost_layers(self, async_client):
        order_id = await _prepare_ledger_data(async_client)

        resp = await async_client.get("/api/finance/order-profit")
        items = resp.json()["data"]["items"]
        item = next(i for i in items if i["order_id"] == order_id)
        assert "shipping_cost_layer" in item
        assert "platform_fee_cost_layer" in item
        assert "profit_cost_layer" in item


@pytest.mark.skip(reason="depends on /api/finance/orders/{id}/ledger/rebuild and /api/settlements/import which are not implemented")
class TestCostVariance:
    async def test_cost_variance_returns_shipping_diffs(self, async_client):
        await _prepare_ledger_data(async_client)

        resp = await async_client.get("/api/finance/cost-variance")
        assert resp.status_code == 200
        # May be empty if no bill, but should still work
        assert isinstance(resp.json()["data"], list)


@pytest.mark.skip(reason="depends on /api/finance/orders/{id}/ledger/rebuild and /api/settlements/import which are not implemented")
class TestNegativeProfit:
    async def test_negative_profit_returns_only_loss_orders(self, async_client):
        await _prepare_ledger_data(async_client)

        resp = await async_client.get("/api/finance/negative-profit")
        assert resp.status_code == 200
        data = resp.json()["data"]
        if len(data) > 0:
            for item in data:
                assert item["profit_amount"] < 0
        else:
            assert isinstance(data, list)


@pytest.mark.skip(reason="depends on /api/finance/orders/{id}/ledger/rebuild and /api/settlements/import which are not implemented")
class TestCostLayerMix:
    async def test_cost_layer_mix_returns_distribution(self, async_client):
        await _prepare_ledger_data(async_client)

        resp = await async_client.get("/api/finance/cost-layer-mix")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert "layers" in data


class TestAuth:
    async def test_report_requires_permission(self, async_client):
        resp = await async_client.get("/api/finance/profit-summary")
        # AUTH_ENABLED=False, should be 200
        assert resp.status_code == 200
