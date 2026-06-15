"""运费账单导入与对账测试。"""

import csv
import io
from uuid import uuid4

from app.database import async_session_factory
from app.models import (
    Order,
    OrderShippingSnapshot,
    ShippingBillBatch,
    ShippingBillItem,
)


async def _make_csv(rows: list[list]) -> tuple[str, bytes]:
    output = io.StringIO()
    writer = csv.writer(output)
    for row in rows:
        writer.writerow(row)
    content = output.getvalue().encode("utf-8-sig")
    return "test.csv", content


async def _create_provider_and_channel(async_client) -> tuple[int, int]:
    uid = f"pv_{uuid4().hex[:6]}"
    resp = await async_client.post("/api/shipping/providers", json={"name": f"PV_{uid}", "code": uid})
    provider_id = resp.json()["data"]["id"]
    resp = await async_client.post(
        "/api/shipping/channels",
        json={"provider_id": provider_id, "name": f"CH_{uid}", "code": f"ch_{uid}", "cargo_types": ["normal"]},
    )
    channel_id = resp.json()["data"]["id"]
    return provider_id, channel_id


# ── Model Tests ──────────────────────────────────────────────────────────────

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
                batch_id=batch.id,
                row_number=2,
                reconciliation_status="unmatched_bill",
                tracking_number="TRACK123",
                provider_name="TestProvider",
                channel_name="TestChannel",
                destination_country="RU",
                actual_shipping_fee=50.0,
            )
            session.add(item)
            await session.flush()
            assert item.id is not None
            await session.rollback()


# ── Import Tests ─────────────────────────────────────────────────────────────

class TestBillImport:
    async def test_import_csv_creates_batch_and_items(self, async_client):
        header = ["运单号", "物流商", "渠道", "目的国", "计费重量(kg)", "实际运费"]
        _, content = await _make_csv([header, ["TRK001", "云途物流", "美国普货", "US", "0.5", "35.00"]])
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

    async def test_import_with_missing_required_columns(self, async_client):
        _, content = await _make_csv([["运单号"], ["TRK001"]])
        resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        assert resp.json()["code"] == 400


# ── Reconciliation ───────────────────────────────────────────────────────────

class TestBillReconciliation:
    async def test_reconcile_marks_unmatched_bill(self, async_client):
        _, content = await _make_csv([["运单号", "物流商", "实际运费"], ["TRK-NO-ORDER", "P", "40"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        items = list_resp.json()["data"]
        assert items[0]["matched_order_id"] is None


# ── List / Summary ───────────────────────────────────────────────────────────

class TestBillList:
    async def test_list_batches(self, async_client):
        resp = await async_client.get("/api/shipping/bills")
        assert resp.status_code == 200

    async def test_get_batch_summary(self, async_client):
        _, content = await _make_csv([["运单号", "物流商", "实际运费"], ["TRK-SUM", "P", "30"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        resp = await async_client.get(f"/api/shipping/bills/{batch_id}")
        assert resp.status_code == 200
        assert resp.json()["data"]["row_count"] == 1

    async def test_list_items(self, async_client):
        _, content = await _make_csv([["运单号", "物流商", "实际运费"], ["TRK-ITM", "P", "30"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        assert resp.status_code == 200
        assert len(resp.json()["data"]) == 1


# ── Manual Resolve ───────────────────────────────────────────────────────────

class TestBillResolve:
    async def test_manual_resolve(self, async_client):
        _, content = await _make_csv([["运单号", "物流商", "实际运费"], ["TRK-RESOLVE", "P", "30"]])
        import_resp = await async_client.post("/api/shipping/bills/import", files={"file": ("test.csv", content, "text/csv")})
        batch_id = import_resp.json()["data"]["batch_id"]
        list_resp = await async_client.get(f"/api/shipping/bills/{batch_id}/items")
        item_id = list_resp.json()["data"][0]["id"]
        resolve_resp = await async_client.post(f"/api/shipping/bills/items/{item_id}/resolve", json={"note": "已确认"})
        assert resolve_resp.status_code == 200
        assert resolve_resp.json()["data"]["reconciliation_status"] == "manual_resolved"
