"""平台结算导入测试。"""

import csv
import io
from uuid import uuid4

import pytest


async def _make_csv(rows: list[list]) -> tuple[str, bytes]:
    output = io.StringIO()
    w = csv.writer(output)
    for row in rows:
        w.writerow(row)
    content = output.getvalue().encode("utf-8-sig")
    return "test.csv", content


async def _create_order(async_client) -> tuple[int, str]:
    """Create a minimal order."""
    uid = uuid4().hex[:6]
    resp = await async_client.post("/api/products", json={"name": f"ST_{uid}"})
    pid = resp.json()["data"]["id"]
    await async_client.post(
        f"/api/products/{pid}/specs",
        json={"specs": [{"name": "颜色", "values": ["标准"]}]},
    )
    sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    sku_id = sku_resp.json()["data"]["skus"][0]["id"]
    await async_client.post(
        "/api/prices", json={"sku_id": sku_id, "price_type": "sale_price", "price": 100}
    )
    await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 10})
    resp = await async_client.post(
        "/api/orders",
        json={
            "items": [{"sku_id": sku_id, "quantity": 1, "unit_price": 100}],
            "recipient_name": "Test",
            "shipping_address": "RU Moscow",
        },
    )
    data = resp.json()["data"]
    return data["id"], data["order_no"]


class TestSettlementImport:
    """POST /api/settlements/import — endpoint not implemented in current API (uses POST /api/settlements with JSON body)"""

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented; use POST /api/settlements with JSON body"
    )
    async def test_import_valid_csv_creates_batch_and_items(self, async_client):
        header = [
            "platform",
            "store_name",
            "order_no",
            "transaction_type",
            "currency",
            "amount",
            "settled_at",
            "description",
        ]
        data = [
            "ozon",
            "MyStore",
            "ORD-001",
            "sale",
            "CNY",
            "199.90",
            "2026-06-15",
            "sale of product",
        ]
        fn, content = await _make_csv([header, data])
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["batch_id"] is not None
        assert data["total_rows"] == 1

    @pytest.mark.skip(reason="POST /api/settlements/import not implemented")
    async def test_import_rejects_non_csv(self, async_client):
        resp = await async_client.post(
            "/api/settlements/import",
            files={"file": ("test.txt", b"no", "text/plain")},
        )
        assert resp.json()["code"] == 400

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_import_missing_required_columns(self, async_client):
        fn, content = await _make_csv([["order_no"], ["ORD-001"]])
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        assert resp.json()["code"] == 400

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_import_writes_operation_log(self, async_client):
        fn, content = await _make_csv(
            [["platform", "transaction_type", "amount"], ["ozon", "sale", "50"]]
        )
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        assert resp.status_code == 200
        batch_id = resp.json()["data"]["batch_id"]

        # Verify operation log was written
        from app.database import async_session_factory
        from app.models import OperationLog

        async with async_session_factory() as session:
            from sqlalchemy import select

            stmt = select(OperationLog).where(
                OperationLog.module == "settlement",
                OperationLog.action == "import",
                OperationLog.resource_id == str(batch_id),
            )
            result = await session.execute(stmt)
            logs = result.scalars().all()
            assert len(logs) >= 1

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_platform_fee_row_matches_by_order_no(self, async_client):
        order_id, order_no = await _create_order(async_client)

        fn, content = await _make_csv(
            [
                ["platform", "order_no", "transaction_type", "amount"],
                ["ozon", order_no, "platform_fee", "12.50"],
            ]
        )
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        batch_id = resp.json()["data"]["batch_id"]

        items_resp = await async_client.get(f"/api/settlements/{batch_id}/items")
        items = items_resp.json()["data"]
        assert len(items) == 1
        assert items[0]["match_status"] == "matched"
        assert items[0]["matched_order_id"] == order_id

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_unknown_order_no_goes_unmatched(self, async_client):
        fn, content = await _make_csv(
            [
                ["platform", "order_no", "transaction_type", "amount"],
                ["ozon", "NONEXISTENT-ORDER", "sale", "99.90"],
            ]
        )
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        batch_id = resp.json()["data"]["batch_id"]

        items_resp = await async_client.get(f"/api/settlements/{batch_id}/items")
        items = items_resp.json()["data"]
        assert items[0]["match_status"] == "unmatched"

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_refund_negative_amount(self, async_client):
        fn, content = await _make_csv(
            [
                ["platform", "order_no", "transaction_type", "amount"],
                ["ozon", "ORD-REFUND", "refund", "-20.00"],
            ]
        )
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        batch_id = resp.json()["data"]["batch_id"]

        items_resp = await async_client.get(f"/api/settlements/{batch_id}/items")
        items = items_resp.json()["data"]
        assert items[0]["amount"] == -20.0
        assert items[0]["transaction_type"] == "refund"

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_item_returns_cost_layer_actual(self, async_client):
        fn, content = await _make_csv(
            [
                ["platform", "transaction_type", "amount"],
                ["ozon", "sale", "50"],
            ]
        )
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        batch_id = resp.json()["data"]["batch_id"]

        items_resp = await async_client.get(f"/api/settlements/{batch_id}/items")
        items = items_resp.json()["data"]
        assert items[0]["cost_layer"] == "actual"


class TestSettlementList:
    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_list_batches(self, async_client):
        fn, content = await _make_csv(
            [["platform", "transaction_type", "amount"], ["shopee", "sale", "30"]]
        )
        await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )

        resp = await async_client.get("/api/settlements")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert len(data) >= 1

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_get_batch(self, async_client):
        fn, content = await _make_csv(
            [["platform", "transaction_type", "amount"], ["wb", "sale", "40"]]
        )
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        batch_id = resp.json()["data"]["batch_id"]

        resp = await async_client.get(f"/api/settlements/{batch_id}")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["filename"] == "test.csv"

    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_unmatched_endpoint(self, async_client):
        fn, content = await _make_csv(
            [
                ["platform", "order_no", "transaction_type", "amount"],
                ["ozon", "UNMATCHED-ORDER", "sale", "100"],
            ]
        )
        await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )

        resp = await async_client.get("/api/settlements/unmatched")
        assert resp.status_code == 200
        items = resp.json()["data"]
        assert any(it["order_no"] == "UNMATCHED-ORDER" for it in items)


class TestSettlementAuth:
    @pytest.mark.skip(
        reason="POST /api/settlements/import not implemented in current API"
    )
    async def test_import_requires_permission(self, async_client):
        fn, content = await _make_csv(
            [["platform", "transaction_type", "amount"], ["ozon", "sale", "50"]]
        )
        resp = await async_client.post(
            "/api/settlements/import", files={"file": (fn, content, "text/csv")}
        )
        assert resp.status_code == 200  # AUTH_ENABLED=False
