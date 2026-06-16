"""费用分摊测试。"""

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


async def _create_sku_with_order(async_client) -> tuple[int, int, str]:
    """Create product+SKU+order, return (sku_id, order_id, order_no)."""
    uid = uuid4().hex[:6]
    resp = await async_client.post("/api/products", json={
        "name": f"ALLOC_{uid}",
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
            "recipient_name": "Test",
            "shipping_address": "RU Moscow",
            "product_cost": 300,
        },
    )
    data = order_resp.json()["data"]
    return sku_id, data["id"], data["order_no"]


class TestAllocationImport:
    """POST /api/allocations/import"""

    async def test_import_creates_batch_and_items(self, async_client):
        header = ["sku_code", "quantity", "weight", "volume", "item_value"]
        fn, content = await _make_csv([header, ["SKU001", "2", "0.5", "0.01", "100"]])

        resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=quantity&total_amount=100&currency=CNY",
            files={"file": (fn, content, "text/csv")},
        )
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["batch_id"] is not None
        assert data["total_rows"] == 1

    async def test_import_rejects_non_csv(self, async_client):
        resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=quantity&total_amount=100",
            files={"file": ("test.txt", b"no", "text/plain")},
        )
        assert resp.json()["code"] == 400

    async def test_import_writes_audit(self, async_client):
        fn, content = await _make_csv([["sku_code", "quantity"], ["SKU001", "1"]])
        resp = await async_client.post(
            "/api/allocations/import?allocation_type=fba&allocation_method=quantity&total_amount=50",
            files={"file": (fn, content, "text/csv")},
        )
        assert resp.status_code == 200

        from app.database import async_session_factory
        from app.models import OperationLog
        from sqlalchemy import select
        async with async_session_factory() as session:
            stmt = select(OperationLog).where(
                OperationLog.module == "allocation",
                OperationLog.action == "import",
            )
            result = await session.execute(stmt)
            assert len(result.scalars().all()) >= 1


class TestAllocationCalculate:
    """POST /api/allocations/{batch_id}/calculate"""

    async def test_quantity_allocation(self, async_client):
        fn, content = await _make_csv([
            ["sku_code", "quantity", "weight", "volume", "item_value"],
            ["SKU-A", "3", "0.3", "0.01", "100"],
            ["SKU-B", "2", "0.5", "0.02", "200"],
        ])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=quantity&total_amount=100&currency=CNY",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]

        calc_resp = await async_client.post(f"/api/allocations/{batch_id}/calculate")
        assert calc_resp.status_code == 200
        data = calc_resp.json()["data"]
        assert data["status"] == "calculated"

        # SKU-A: qty=3 => 3/5 of 100 = 60
        # SKU-B: qty=2 => 2/5 of 100 = 40
        items = data["items"]
        assert float(items[0]["allocated_amount"]) == 60.0
        assert float(items[1]["allocated_amount"]) == 40.0

    async def test_weight_allocation(self, async_client):
        fn, content = await _make_csv([
            ["sku_code", "quantity", "weight", "volume", "item_value"],
            ["SKU-A", "1", "1.0", "0.01", "100"],
            ["SKU-B", "1", "3.0", "0.02", "200"],
        ])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=fba&allocation_method=weight&total_amount=80",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]

        calc_resp = await async_client.post(f"/api/allocations/{batch_id}/calculate")
        data = calc_resp.json()["data"]
        items = data["items"]
        # A: 1.0/(1.0+3.0)*80 = 20
        # B: 3.0/4.0*80 = 60
        assert float(items[0]["allocated_amount"]) == 20.0
        assert float(items[1]["allocated_amount"]) == 60.0

    async def test_volume_allocation(self, async_client):
        fn, content = await _make_csv([
            ["sku_code", "quantity", "weight", "volume", "item_value"],
            ["SKU-A", "1", "0.5", "0.01", "100"],
            ["SKU-B", "1", "0.3", "0.03", "200"],
        ])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=volume&total_amount=100",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]

        calc_resp = await async_client.post(f"/api/allocations/{batch_id}/calculate")
        data = calc_resp.json()["data"]
        items = data["items"]
        # A: 0.01/(0.01+0.03)*100 = 25
        # B: 0.03/0.04*100 = 75
        assert float(items[0]["allocated_amount"]) == 25.0
        assert float(items[1]["allocated_amount"]) == 75.0

    async def test_value_allocation(self, async_client):
        fn, content = await _make_csv([
            ["sku_code", "quantity", "weight", "volume", "item_value"],
            ["SKU-A", "1", "0.5", "0.01", "100"],
            ["SKU-B", "1", "0.3", "0.03", "300"],
        ])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=other&allocation_method=value&total_amount=200",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]

        calc_resp = await async_client.post(f"/api/allocations/{batch_id}/calculate")
        data = calc_resp.json()["data"]
        items = data["items"]
        # A: 100/(100+300)*200 = 50
        # B: 300/400*200 = 150
        assert float(items[0]["allocated_amount"]) == 50.0
        assert float(items[1]["allocated_amount"]) == 150.0

    async def test_rounding_controls_to_0_01(self, async_client):
        """尾差在 0.01 内，总和不大于 total_amount+0.01"""
        fn, content = await _make_csv([
            ["sku_code", "quantity", "weight", "volume", "item_value"],
            ["SKU-A", "1", "0.3", "0.01", "100"],
            ["SKU-B", "1", "0.3", "0.01", "100"],
            ["SKU-C", "1", "0.3", "0.01", "100"],
        ])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=quantity&total_amount=100",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]

        calc_resp = await async_client.post(f"/api/allocations/{batch_id}/calculate")
        data = calc_resp.json()["data"]
        items = data["items"]
        total = sum(float(it["allocated_amount"]) for it in items)
        assert abs(total - 100.0) <= 0.01

    async def test_cost_layer_is_allocated(self, async_client):
        fn, content = await _make_csv([["sku_code", "quantity"], ["SKU-X", "1"]])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=quantity&total_amount=50",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/allocations/{batch_id}/calculate")

        items_resp = await async_client.get(f"/api/allocations/{batch_id}/items")
        items = items_resp.json()["data"]
        assert items[0]["cost_layer"] == "allocated"


class TestAllocationPostToLedger:
    """POST /api/allocations/{batch_id}/post-to-ledger"""

    async def test_post_creates_ledger_entry(self, async_client):
        sku_id, order_id, order_no = await _create_sku_with_order(async_client)

        fn, content = await _make_csv([
            ["sku_code", "order_no", "quantity", "weight", "volume", "item_value"],
            [f"SKU-{uuid4().hex[:4]}", order_no, "1", "0.5", "0.01", "300"],
        ])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=value&total_amount=30",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/allocations/{batch_id}/calculate")

        post_resp = await async_client.post(f"/api/allocations/{batch_id}/post-to-ledger")
        assert post_resp.status_code == 200
        assert post_resp.json()["data"]["posted_count"] >= 1

        # Verify ledger entry
        ledger_resp = await async_client.get(f"/api/finance/orders/{order_id}/ledger")
        entries = ledger_resp.json()["data"]["entries"]
        alloc_entries = [e for e in entries if e["entry_type"] == "allocated_cost"]
        assert len(alloc_entries) >= 1
        assert alloc_entries[0]["cost_layer"] == "allocated"

    async def test_post_is_idempotent(self, async_client):
        sku_id, order_id, order_no = await _create_sku_with_order(async_client)

        fn, content = await _make_csv([
            ["sku_code", "order_no", "quantity", "weight", "volume", "item_value"],
            [f"SKU-{uuid4().hex[:4]}", order_no, "1", "0.5", "0.01", "300"],
        ])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=value&total_amount=20",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]
        await async_client.post(f"/api/allocations/{batch_id}/calculate")

        post1 = await async_client.post(f"/api/allocations/{batch_id}/post-to-ledger")
        c1 = post1.json()["data"]["posted_count"]
        post2 = await async_client.post(f"/api/allocations/{batch_id}/post-to-ledger")
        c2 = post2.json()["data"]["posted_count"]

        assert c1 >= 1
        assert c2 == 0  # already posted


class TestAllocationList:
    async def test_list_batches(self, async_client):
        resp = await async_client.get("/api/allocations")
        assert resp.status_code == 200

    async def test_get_batch(self, async_client):
        fn, content = await _make_csv([["sku_code", "quantity"], ["X", "1"]])
        import_resp = await async_client.post(
            "/api/allocations/import?allocation_type=fba&allocation_method=quantity&total_amount=10",
            files={"file": (fn, content, "text/csv")},
        )
        batch_id = import_resp.json()["data"]["batch_id"]
        resp = await async_client.get(f"/api/allocations/{batch_id}")
        assert resp.status_code == 200


class TestAllocationAuth:
    async def test_import_permission(self, async_client):
        fn, content = await _make_csv([["sku_code", "quantity"], ["SKU-P", "1"]])
        resp = await async_client.post(
            "/api/allocations/import?allocation_type=first_leg&allocation_method=quantity&total_amount=10",
            files={"file": (fn, content, "text/csv")},
        )
        assert resp.status_code == 200

    async def test_view_permission(self, async_client):
        resp = await async_client.get("/api/allocations")
        assert resp.status_code == 200
