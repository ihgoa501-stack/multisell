"""运费账单导入与对账测试。"""

import csv
import io
from uuid import uuid4

import pytest
from httpx import AsyncClient

from app.database import async_session_factory
from app.models import (
    Order,
    OrderShippingSnapshot,
    ShippingBillBatch,
    ShippingBillItem,
)


def _code(prefix: str) -> str:
    return f"{prefix}_{uuid4().hex[:6]}"


async def _create_order_with_snapshot(
    async_client,
    provider_id: int,
    channel_id: int,
    tracking_no: str = None,
    snapshot_fee: float = 50.0,
    currency: str = "CNY",
) -> tuple[int, int, str]:
    """创建订单+运费快照，返回 (order_id, snapshot_id, order_no)"""
    uid = uuid4().hex[:6]
    resp = await async_client.post("/api/products", json={"name": f"BillTest_{uid}"})
    pid = resp.json()["data"]["id"]
    await async_client.post(f"/api/products/{pid}/specs", json={"specs": [{"name": "颜色", "values": ["标准"]}]})
    sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    sku_id = sku_resp.json()["data"]["skus"][0]["id"]
    await async_client.post("/api/prices", json={"sku_id": sku_id, "price_type": "sale_price", "price": 100})
    await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 10})

    order_no = f"ORD-{uid}"
    resp = await async_client.post(
        "/api/orders",
        json={
            "order_no": order_no,
            "items": [{"sku_id": sku_id, "quantity": 1, "unit_price": 100}],
            "recipient_name": "Test",
            "shipping_address": "RU Moscow",
        },
    )
    order_id = resp.json()["data"]["id"]
    actual_order_no = resp.json()["data"]["order_no"]

    async with async_session_factory() as session:
        order = await session.get(Order, order_id)
        if tracking_no:
            order.tracking_number = tracking_no
        await session.flush()

        snapshot = OrderShippingSnapshot(
            order_id=order_id,
            sku_id=sku_id,
            quantity=1,
            destination_country="RU",
            cargo_type="normal",
            package_source="product",
            package_length_cm=30,
            package_width_cm=20,
            package_height_cm=10,
            package_weight_kg=0.5,
            provider_id=provider_id,
            provider_name=f"Provider_{uid}",
            channel_id=channel_id,
            channel_name=f"Channel_{uid}",
            currency=currency,
            actual_weight_kg=0.5,
            volumetric_weight_kg=0.2,
            chargeable_weight_kg=0.5,
            base_shipping_fee=snapshot_fee,
            surcharge_fee=0,
            fuel_surcharge_fee=0,
            total_shipping_fee=snapshot_fee,
            calculation_detail="test snapshot",
        )
        session.add(snapshot)
        await session.flush()
        snap_id = snapshot.id
        await session.commit()

    return order_id, snap_id, actual_order_no


async def _create_provider_and_channel(async_client) -> tuple[int, int]:
    uid = _code("pv")
    resp = await async_client.post("/api/shipping/providers", json={"name": f"PV_{uid}", "code": uid})
    provider_id = resp.json()["data"]["id"]
    resp = await async_client.post(
        "/api/shipping/channels",
        json={"provider_id": provider_id, "name": f"CH_{uid}", "code": f"ch_{uid}", "cargo_types": ["normal"]},
    )
    channel_id = resp.json()["data"]["id"]
    return provider_id, channel_id


def _make_csv(rows: list[list]) -> tuple[str, bytes]:
    output = io.StringIO()
    writer = csv.writer(output)
    for row in rows:
        writer.writerow(row)
    content = output.getvalue().encode("utf-8-sig")
    return "test.csv", content


# ── Model Tests ──────────────────────────────────────────────────────────

class TestBillModel:
    async def test_bill_batch_model_is_mapped(self, async_client):
        async with async_session_factory() as session:
            batch = ShippingBillBatch(source_filename="test.csv", row_count=5, status="imported")
            session.add(batch)
            await session.flush()
            assert batch.id is not None
            await session.rollback()

    async def test_bill_item_model_is_mapped(self, async_client):
        async with async_session_factory() as session:
            batch = ShippingBillBatch(source_filename="test_items.csv", row_count=1)
            session.add(batch)
            await session.flush()
            item = ShippingBillItem(
                batch_id=batch.id, row_number=2, reconciliation_status="unmatched_bill",
                tracking_number="TRACK123", provider_name="TestProvider",
                actual_shipping_fee=50.0, surcharge_fee=0, total_actual_fee=50.0,
            )
            session.add(item)
            await session.flush()
            assert item.id is not None
            await session.rollback()


# ── Import Tests ─────────────────────────────────────────────────────────

class TestBillImport:
    """POST /api/shipping/bills/import"""

    async def test_import_csv_en_format(self, async_client):
        """英文格式 CSV 导入"""
        header = ["order_no", "tracking_number", "provider_name", "channel_name", "currency", "actual_shipping_fee"]
        _, content = _make_csv([header, ["SO001", "TRK001", "云途", "US", "CNY", "35.00"]])
        resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["total_rows"] == 1
        assert data["batch_id"] is not None

    async def test_import_csv_cn_format(self, async_client):
        """中文格式 CSV 导入"""
        header = ["运单号", "订单号", "物流商", "渠道", "实际运费", "币种"]
        _, content = _make_csv([header, ["TRK001", "SO001", "云途", "US", "35.00", "CNY"]])
        resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["total_rows"] == 1
        assert data["batch_id"] is not None

    async def test_import_rejects_non_csv(self, async_client):
        resp = await async_client.post(
            "/api/shipping/bills/import", files={"file": ("test.txt", b"no", "text/plain")}
        )
        assert resp.json()["code"] == 400

    async def test_import_missing_required_columns(self, async_client):
        _, content = _make_csv([["order_no"], ["SO001"]])
        resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        assert resp.json()["code"] == 400


# ── Reconciliation Tests ─────────────────────────────────────────────────

class TestBillReconciliation:
    """POST /api/shipping/bills/{batch_id}/reconcile"""

    async def test_reconcile_matches_by_tracking_number(self, async_client):
        pv_id, ch_id = await _create_provider_and_channel(async_client)
        order_id, snap_id, _ = await _create_order_with_snapshot(
            async_client, pv_id, ch_id, tracking_no="TRK-MATCH-001", snapshot_fee=50.0
        )
        _, content = _make_csv([["tracking_number", "provider_name", "actual_shipping_fee"],
                                ["TRK-MATCH-001", "Provider", "50.00"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        items = list_resp.json()["data"]
        assert len(items) == 1
        assert items[0]["reconciliation_status"] == "matched"

    async def test_reconcile_detects_amount_mismatch(self, async_client):
        pv_id, ch_id = await _create_provider_and_channel(async_client)
        order_id, snap_id, _ = await _create_order_with_snapshot(
            async_client, pv_id, ch_id, tracking_no="TRK-AMT-001", snapshot_fee=50.0
        )
        _, content = _make_csv([["tracking_number", "provider_name", "actual_shipping_fee"],
                                ["TRK-AMT-001", "Provider", "75.00"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        items = list_resp.json()["data"]
        assert items[0]["reconciliation_status"] == "amount_mismatch"

    async def test_reconcile_detects_currency_mismatch(self, async_client):
        pv_id, ch_id = await _create_provider_and_channel(async_client)
        order_id, snap_id, _ = await _create_order_with_snapshot(
            async_client, pv_id, ch_id, tracking_no="TRK-CUR-001", snapshot_fee=50.0, currency="USD"
        )
        _, content = _make_csv([["tracking_number", "provider_name", "currency", "actual_shipping_fee"],
                                ["TRK-CUR-001", "Provider", "CNY", "50.00"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        items = list_resp.json()["data"]
        assert items[0]["reconciliation_status"] == "currency_mismatch"

    async def test_reconcile_marks_missing_snapshot(self, async_client):
        pv_id, ch_id = await _create_provider_and_channel(async_client)
        uid = uuid4().hex[:6]
        resp = await async_client.post("/api/products", json={"name": f"NoSnap_{uid}"})
        pid = resp.json()["data"]["id"]
        await async_client.post(f"/api/products/{pid}/specs", json={"specs": [{"name": "颜色", "values": ["标准"]}]})
        sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]
        await async_client.post("/api/prices", json={"sku_id": sku_id, "price_type": "sale_price", "price": 100})
        await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 10})
        resp = await async_client.post(
            "/api/orders",
            json={"order_no": f"NO-SNAP-{uid}", "items": [{"sku_id": sku_id, "quantity": 1, "unit_price": 100}],
                  "recipient_name": "Test", "shipping_address": "RU Moscow"},
        )
        order_id = resp.json()["data"]["id"]
        async with async_session_factory() as session:
            order = await session.get(Order, order_id)
            order.tracking_number = "TRK-NO-SNAP"
            await session.commit()
        _, content = _make_csv([["tracking_number", "provider_name", "actual_shipping_fee"],
                                ["TRK-NO-SNAP", "Provider", "30.00"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        items = list_resp.json()["data"]
        assert items[0]["reconciliation_status"] == "missing_snapshot"

    async def test_reconcile_marks_unmatched_bill(self, async_client):
        _, content = _make_csv([["tracking_number", "provider_name", "actual_shipping_fee"],
                                ["TRK-NO-ORDER", "Provider", "40.00"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        items = list_resp.json()["data"]
        assert items[0]["reconciliation_status"] == "unmatched_bill"

    async def test_reconcile_matches_by_order_no(self, async_client):
        pv_id, ch_id = await _create_provider_and_channel(async_client)
        order_id, snap_id, order_no = await _create_order_with_snapshot(
            async_client, pv_id, ch_id, tracking_no=None, snapshot_fee=50.0
        )
        _, content = _make_csv([["order_no", "provider_name", "actual_shipping_fee"],
                                [order_no, "Provider", "50.00"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        items = list_resp.json()["data"]
        assert items[0]["reconciliation_status"] == "matched"


# ── List / Summary Tests ─────────────────────────────────────────────────

class TestBillList:
    async def test_list_batches(self, async_client):
        resp = await async_client.get("/api/shipping/bills")
        assert resp.status_code == 200

    async def test_get_batch(self, async_client):
        _, content = _make_csv([["tracking_number", "provider_name", "actual_shipping_fee"],
                                ["TRK-SUM", "P", "30"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        resp = await async_client.get(f"/api/shipping/bills/{batch_id}")
        assert resp.status_code == 200
        assert resp.json()["data"]["row_count"] == 1

    async def test_list_items(self, async_client):
        _, content = _make_csv([["tracking_number", "provider_name", "actual_shipping_fee"],
                                ["TRK-ITM", "P", "30"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        assert resp.status_code == 200
        assert len(resp.json()["data"]) == 1

    async def test_reconciliation_summary(self, async_client):
        resp = await async_client.get("/api/shipping/reconciliation/summary")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert "total_batches" in data
        assert "total_items" in data


# ── Manual Resolve ───────────────────────────────────────────────────────

class TestBillResolve:
    async def test_manual_resolve(self, async_client):
        _, content = _make_csv([["tracking_number", "provider_name", "actual_shipping_fee"],
                                ["TRK-RESOLVE", "P", "30"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        item_id = list_resp.json()["data"][0]["id"]
        resolve_resp = await async_client.post(f"/api/shipping/bills/items/{item_id}/resolve", json={"note": "已确认"})
        assert resolve_resp.status_code == 200
        assert resolve_resp.json()["data"]["reconciliation_status"] == "manual_resolved"


# ── Auth / Audit ─────────────────────────────────────────────────────────

class TestBillAuthAudit:
    async def test_import_requires_permission(self, async_client):
        resp = await async_client.post(
            "/api/shipping/bills/import", files={"file": ("test.csv", b"a,b\n1,2", "text/csv")}
        )
        assert resp.json()["code"] == 400

    async def test_list_batches_allowed(self, async_client):
        resp = await async_client.get("/api/shipping/bills")
        assert resp.status_code == 200
