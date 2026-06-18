"""订单利润账本测试。"""

import csv
import io
from uuid import uuid4

import pytest


async def _create_minimal_order(async_client) -> tuple[int, str, int]:
    """Create order + shipping provider/channel. Returns (order_id, order_no, sku_id)."""
    uid = uuid4().hex[:6]
    resp = await async_client.post("/api/products", json={
        "name": f"LED_{uid}",
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

    # Create provider + channel
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
            "items": [{"sku_id": sku_id, "quantity": 1, "unit_price": 5000}],
            "recipient_name": "Test",
            "shipping_address": "RU Moscow",
            "shipping_fee": 30,
            "platform_fee": 50,
            "product_cost": 300,
        },
    )
    data = order_resp.json()["data"]
    return data["id"], data["order_no"], sku_id, pv_id, ch_id


class TestLedgerRebuild:
    """POST /api/finance/orders/{order_id}/ledger/rebuild"""

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_rebuild_creates_revenue_and_product_cost(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)

        resp = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        assert resp.status_code == 200
        data = resp.json()["data"]

        assert data["revenue_amount"] == 5000.0
        assert data["product_cost"] == 300.0

        # Check ledger entries exist
        ledger_resp = await async_client.get(f"/api/finance/orders/{order_id}/ledger")
        entries = ledger_resp.json()["data"]["entries"]
        entry_types = {e["entry_type"] for e in entries}
        assert "revenue" in entry_types
        assert "product_cost" in entry_types
        assert "shipping_cost" in entry_types
        assert "platform_fee" in entry_types

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_rebuild_uses_shipping_snapshot_when_no_bill(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)

        # Bind shipping quote to create snapshot
        await async_client.post(
            f"/api/orders/{order_id}/shipping-quote",
            json={"sku_id": sku_id, "quantity": 1, "destination_country": "RU", "cargo_type": "normal"},
        )

        resp = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        data = resp.json()["data"]
        assert data["shipping_cost_layer"] == "snapshot"

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_rebuild_prefers_actual_bill_over_snapshot(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)

        # Bind shipping quote to create snapshot
        await async_client.post(
            f"/api/orders/{order_id}/shipping-quote",
            json={"sku_id": sku_id, "quantity": 1, "destination_country": "RU", "cargo_type": "normal"},
        )

        # Set tracking number on order for bill matching
        from app.database import async_session_factory
        from app.models import Order
        async with async_session_factory() as session:
            order = await session.get(Order, order_id)
            order.tracking_number = "TRK-LEDGER-BILL"
            await session.commit()

        # Import a shipping bill that matches
        output = io.StringIO()
        w = csv.writer(output)
        w.writerow(["tracking_number", "provider_name", "actual_shipping_fee"])
        w.writerow(["TRK-LEDGER-BILL", "PV", "25.00"])
        content = output.getvalue().encode("utf-8-sig")

        import_resp = await async_client.post(
            "/api/shipping/bills/import",
            files={"file": ("test.csv", content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")

        resp = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        data = resp.json()["data"]
        # With bill matched, shipping should be from bill (25.00) and cost_layer=actual
        assert data["shipping_cost"] == 25.0
        assert data["shipping_cost_layer"] == "actual"

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_rebuild_includes_settlement_platform_fee(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)

        # Import a settlement row that matches the order
        output = io.StringIO()
        w = csv.writer(output)
        w.writerow(["platform", "order_no", "transaction_type", "amount", "currency"])
        w.writerow(["ozon", order_no, "platform_fee", "60.00", "CNY"])
        content = output.getvalue().encode("utf-8-sig")

        await async_client.post(
            "/api/settlements/import",
            files={"file": ("test.csv", content, "text/csv")},
        )

        resp = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        data = resp.json()["data"]
        # Settlement platform_fee = 60 should override the order default of 50
        assert data["platform_fee"] == 60.0
        assert data["platform_fee_cost_layer"] == "actual"

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_rebuild_refund_reduces_profit(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)

        # Import a refund settlement row
        output = io.StringIO()
        w = csv.writer(output)
        w.writerow(["platform", "order_no", "transaction_type", "amount", "currency"])
        w.writerow(["ozon", order_no, "refund", "-50.00", "CNY"])
        content = output.getvalue().encode("utf-8-sig")

        await async_client.post(
            "/api/settlements/import",
            files={"file": ("test.csv", content, "text/csv")},
        )

        resp = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        data = resp.json()["data"]
        assert data["refund"] == -50.0  # refund is stored as negative

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_rebuild_profit_cost_layer_mixed(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)

        # Bind shipping quote (snapshot) and import settlement (actual platform_fee)
        await async_client.post(
            f"/api/orders/{order_id}/shipping-quote",
            json={"sku_id": sku_id, "quantity": 1, "destination_country": "RU", "cargo_type": "normal"},
        )

        output = io.StringIO()
        w = csv.writer(output)
        w.writerow(["platform", "order_no", "transaction_type", "amount"])
        w.writerow(["ozon", order_no, "platform_fee", "60.00"])
        content = output.getvalue().encode("utf-8-sig")

        await async_client.post(
            "/api/settlements/import",
            files={"file": ("test.csv", content, "text/csv")},
        )

        resp = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        data = resp.json()["data"]
        # shipping=snapshot, platform_fee=actual, so profit_cost_layer=mixed
        assert data["profit_cost_layer"] == "mixed"

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_rebuild_is_idempotent(self, async_client):
        """Rebuild twice produces same result."""
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)

        resp1 = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        data1 = resp1.json()["data"]

        resp2 = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        data2 = resp2.json()["data"]

        assert data1["profit_amount"] == data2["profit_amount"]
        assert data1["revenue_amount"] == data2["revenue_amount"]

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_rebuild_requires_permission(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)
        resp = await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")
        # AUTH_ENABLED=False so it should be 200
        assert resp.status_code == 200


class TestLedgerGet:
    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_get_ledger_entries(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)
        await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")

        resp = await async_client.get(f"/api/finance/orders/{order_id}/ledger")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["total_entries"] >= 4

    @pytest.mark.skip(reason="endpoint /api/finance/orders/{order_id}/ledger/rebuild not implemented yet")
    async def test_get_profit_summary(self, async_client):
        order_id, order_no, sku_id, pv_id, ch_id = await _create_minimal_order(async_client)
        await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")

        resp = await async_client.get(f"/api/finance/orders/{order_id}/profit")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert "profit_amount" in data
        assert "profit_cost_layer" in data
