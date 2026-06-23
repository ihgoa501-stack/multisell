"""异常工作台测试。"""

import csv
import io
from uuid import uuid4

import pytest


async def _create_listing_task_data(async_client) -> tuple[int, int, int]:
    """Create product+SKU+platform for listing task, return (sku_id, platform_id, product_id)."""
    uid = uuid4().hex[:6]
    # Create product WITH package data so decision returns "approve", but WITHOUT main_image so listing is "blocked"
    resp = await async_client.post(
        "/api/products",
        json={
            "name": f"EX_{uid}",
            "package_length_cm": 30,
            "package_width_cm": 20,
            "package_height_cm": 10,
            "package_weight_kg": 0.5,
        },
    )
    pid = resp.json()["data"]["id"]
    await async_client.post(
        f"/api/products/{pid}/specs",
        json={"specs": [{"name": "颜色", "values": ["标准"]}]},
    )
    sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    sku_id = sku_resp.json()["data"]["skus"][0]["id"]
    await async_client.put(f"/api/skus/{sku_id}", json={"cost_price": 300})
    plat_resp = await async_client.post(
        "/api/platforms", json={"name": f"PL_{uid}", "code": f"pl_{uid}"}
    )
    plat_id = plat_resp.json()["data"]["id"]
    # Add shipping provider + channel for decision calculation
    pv_resp = await async_client.post(
        "/api/shipping/providers", json={"name": f"PV_{uid}", "code": f"pv_{uid}"}
    )
    pv_id = pv_resp.json()["data"]["id"]
    ch_resp = await async_client.post(
        "/api/shipping/channels",
        json={
            "provider_id": pv_id,
            "name": f"CH_{uid}",
            "code": f"ch_{uid}",
            "cargo_types": ["normal"],
        },
    )
    ch_id = ch_resp.json()["data"]["id"]
    await async_client.post(
        f"/api/shipping/channels/{ch_id}/zones", json={"country_code": "RU"}
    )
    await async_client.post(
        f"/api/shipping/channels/{ch_id}/rules",
        json={
            "rule_type": "fixed_plus_per_kg",
            "fixed_fee": 10,
            "per_kg_price": 20,
        },
    )
    return sku_id, plat_id, pid


async def _create_settlement_csv(order_no: str, tx_type: str, amount: str) -> bytes:
    output = io.StringIO()
    w = csv.writer(output)
    w.writerow(["platform", "order_no", "transaction_type", "amount"])
    w.writerow(["ozon", order_no, tx_type, amount])
    return output.getvalue().encode("utf-8-sig")


async def _import_settlement(
    async_client, order_no: str, tx_type: str, amount: str
) -> int:
    content = await _create_settlement_csv(order_no, tx_type, amount)
    resp = await async_client.post(
        "/api/settlements/import",
        files={"file": ("test.csv", content, "text/csv")},
    )
    return resp.json()["data"]["batch_id"]


async def _import_bill(async_client, tracking: str, amount: str) -> int:
    output = io.StringIO()
    w = csv.writer(output)
    w.writerow(["运单号", "物流商", "实际运费"])
    w.writerow([tracking, "Provider", amount])
    content = output.getvalue().encode("utf-8-sig")
    resp = await async_client.post(
        "/api/shipping/bills/import",
        files={"file": ("test.csv", content, "text/csv")},
    )
    return resp.json()["data"]["batch_id"]


class TestExceptionGenerate:
    """POST /api/exceptions/generate"""

    @pytest.mark.skip(
        reason="endpoint POST /api/exceptions/generate not implemented yet"
    )
    async def test_generates_for_blocked_listing_task(self, async_client):
        sku_id, plat_id, _ = await _create_listing_task_data(async_client)
        # Create listing task from a decision (no image = blocked)
        dec_resp = await async_client.post(
            "/api/decisions/prelisting/batch",
            json={
                "items": [
                    {
                        "item_key": "blocked-sku",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "destination_country": "RU",
                        "target_sale_price": 5000,
                        "platform_fee_pct": 10,
                        "payment_fee_pct": 3,
                        "other_fee": 0,
                        "minimum_margin_pct": 20,
                        "cargo_type": "normal",
                    }
                ],
            },
        )
        approve = dec_resp.json()["data"]["items"][0]
        assert approve["status"] == "success"

        task_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "blocked-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": approve["result"],
                    }
                ],
            },
        )
        assert task_resp.status_code == 200

        gen_resp = await async_client.post("/api/exceptions/generate")
        assert gen_resp.status_code == 200
        data = gen_resp.json()["data"]
        assert data["created_count"] >= 1

        # Verify listing exception exists
        list_resp = await async_client.get("/api/exceptions?source_module=listing")
        items = list_resp.json()["data"]
        assert any(it["source_module"] == "listing" for it in items)

    @pytest.mark.skip(
        reason="endpoint POST /api/exceptions/generate not implemented yet"
    )
    async def test_generates_for_unmatched_shipping_bill(self, async_client):
        batch_id = await _import_bill(async_client, "TRK-NO-ORDER", "55.00")
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")

        gen_resp = await async_client.post("/api/exceptions/generate")
        assert gen_resp.status_code == 200

        list_resp = await async_client.get("/api/exceptions?source_module=shipping")
        items = list_resp.json()["data"]
        assert any(it["source_module"] == "shipping" for it in items)

    @pytest.mark.skip(
        reason="endpoint POST /api/exceptions/generate not implemented yet"
    )
    async def test_generates_for_unmatched_settlement(self, async_client):
        await _import_settlement(async_client, "NO-ORDER-EXISTS", "sale", "99.90")

        gen_resp = await async_client.post("/api/exceptions/generate")
        assert gen_resp.status_code == 200

        list_resp = await async_client.get("/api/exceptions?source_module=settlement")
        items = list_resp.json()["data"]
        assert any(it["source_module"] == "settlement" for it in items)

    @pytest.mark.skip(
        reason="endpoint POST /api/exceptions/generate not implemented yet"
    )
    async def test_generates_for_negative_profit(self, async_client):
        uid = uuid4().hex[:6]
        resp = await async_client.post("/api/products", json={"name": f"NP_{uid}"})
        pid = resp.json()["data"]["id"]
        await async_client.post(
            f"/api/products/{pid}/specs",
            json={"specs": [{"name": "颜色", "values": ["标准"]}]},
        )
        sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]
        await async_client.put(f"/api/skus/{sku_id}", json={"cost_price": 800})
        await async_client.post(
            "/api/prices",
            json={"sku_id": sku_id, "price_type": "sale_price", "price": 50},
        )
        await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 5})

        order_resp = await async_client.post(
            "/api/orders",
            json={
                "items": [{"sku_id": sku_id, "quantity": 1, "unit_price": 100}],
                "recipient_name": "Test",
                "shipping_address": "RU Moscow",
                "product_cost": 200,
                "shipping_fee": 50,
                "platform_fee": 30,
            },
        )
        order_id = order_resp.json()["data"]["id"]
        await async_client.post(f"/api/finance/orders/{order_id}/ledger/rebuild")

        gen_resp = await async_client.post("/api/exceptions/generate")
        assert gen_resp.status_code == 200

        list_resp = await async_client.get("/api/exceptions?source_module=finance")
        items = list_resp.json()["data"]
        assert any(it["source_module"] == "finance" for it in items)

    @pytest.mark.skip(
        reason="endpoint POST /api/exceptions/generate not implemented yet"
    )
    async def test_generate_is_idempotent(self, async_client):
        batch_id = await _import_bill(async_client, "TRK-IDEMP", "33")
        await async_client.post(f"/api/shipping/bills/{batch_id}/reconcile")

        gen1 = await async_client.post("/api/exceptions/generate")
        gen1.json()["data"]["created_count"]

        gen2 = await async_client.post("/api/exceptions/generate")
        c2 = gen2.json()["data"]["created_count"]

        assert c2 == 0  # same source should not create duplicate


class TestExceptionAssign:
    @pytest.mark.skip(
        reason="endpoint POST /api/exceptions/generate not implemented yet"
    )
    async def test_assign_updates_status_and_writes_audit(self, async_client):
        await _import_bill(async_client, "TRK-ASSIGN", "44")
        await async_client.post("/api/exceptions/generate")

        list_resp = await async_client.get("/api/exceptions")
        ex = list_resp.json()["data"][0]

        assign_resp = await async_client.post(
            f"/api/exceptions/{ex['id']}/assign",
            json={"assigned_to": "operator1"},
        )
        assert assign_resp.status_code == 200
        assert assign_resp.json()["data"]["status"] == "assigned"
        assert assign_resp.json()["data"]["assigned_to"] == "operator1"


class TestExceptionResolve:
    @pytest.mark.skip(
        reason="endpoint POST /api/exceptions/generate not implemented yet"
    )
    async def test_resolve_changes_status(self, async_client):
        await _import_bill(async_client, "TRK-RESOLVE", "66")
        await async_client.post("/api/exceptions/generate")

        list_resp = await async_client.get("/api/exceptions")
        ex = list_resp.json()["data"][0]

        resolve_resp = await async_client.post(
            f"/api/exceptions/{ex['id']}/resolve",
            json={"note": "已核实"},
        )
        assert resolve_resp.status_code == 200
        assert resolve_resp.json()["data"]["status"] == "resolved"


class TestExceptionIgnore:
    @pytest.mark.skip(
        reason="endpoint POST /api/exceptions/generate not implemented yet"
    )
    async def test_ignore_changes_status(self, async_client):
        await _import_bill(async_client, "TRK-IGNORE", "77")
        await async_client.post("/api/exceptions/generate")

        list_resp = await async_client.get("/api/exceptions")
        ex = list_resp.json()["data"][0]

        ignore_resp = await async_client.post(
            f"/api/exceptions/{ex['id']}/ignore",
            json={"note": "忽略此异常"},
        )
        assert ignore_resp.status_code == 200
        assert ignore_resp.json()["data"]["status"] == "ignored"


class TestExceptionList:
    async def test_list_filters_by_source_module(self, async_client):
        resp = await async_client.get("/api/exceptions?source_module=shipping")
        assert resp.status_code == 200

    async def test_list_filters_by_severity(self, async_client):
        resp = await async_client.get("/api/exceptions?severity=high")
        assert resp.status_code == 200

    async def test_list_filters_by_status(self, async_client):
        resp = await async_client.get("/api/exceptions?status=open")
        assert resp.status_code == 200
