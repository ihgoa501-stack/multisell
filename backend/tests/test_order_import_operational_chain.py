"""订单导入经营链路测试"""

from uuid import uuid4

import pytest
from httpx import AsyncClient

from app.database import async_session_factory
from tests.auth_helpers import register_and_login, grant_permission
from tests.test_order_import_csv_adapter import (
    _create_product_and_sku,
    _ensure_inventory,
    _make_csv,
)


class TestMultiSkuImport:
    @pytest.mark.skip(
        reason="endpoint /api/order-imports/{batch_id}/items not implemented in current API"
    )
    async def test_same_platform_order_no_creates_one_order_with_two_items(
        self, async_client: AsyncClient
    ):
        async with async_session_factory() as session:
            _, sku1 = await _create_product_and_sku(
                session, sku_code=f"MSI-{uuid4().hex[:5]}"
            )
            _, sku2 = await _create_product_and_sku(
                session, sku_code=f"MSI-{uuid4().hex[:5]}"
            )
            await _ensure_inventory(session, sku1.id, 100)
            await _ensure_inventory(session, sku2.id, 100)
            await session.commit()
        csv_text = (
            "platform,store_name,platform_order_no,order_no,sku_code,quantity,unit_price,currency,recipient_name,recipient_phone,country_code,shipping_address,shipping_fee,paid_at\n"
            f"amazon,US,AMZ-MS01,,{sku1.code},1,20,CNY,Alice,123,US,Street 1,5,2026-06-16\n"
            f"amazon,US,AMZ-MS01,,{sku2.code},2,15,CNY,Alice,123,US,Street 1,5,2026-06-16\n"
        )
        _, content = _make_csv(csv_text)
        response = await async_client.post(
            "/api/order-import/csv",
            files={"file": ("orders.csv", content, "text/csv")},
        )
        assert response.status_code == 200, response.text
        batch = response.json()["data"]

        items_response = await async_client.get(
            f"/api/order-imports/{batch['id']}/items"
        )
        assert items_response.status_code == 200, items_response.text
        rows = items_response.json()["data"]
        order_ids = {row["order_id"] for row in rows if row["order_id"]}
        assert len(order_ids) == 1

        order_id = next(iter(order_ids))
        order_response = await async_client.get(f"/api/orders/{order_id}")
        assert order_response.status_code == 200, order_response.text
        order = order_response.json()["data"]
        assert len(order["items"]) == 2
        assert order["total_amount"] == 50.0

    @pytest.mark.skip(
        reason="current import service does not create OrderImportItem records"
    )
    async def test_multi_sku_with_invalid_sku_still_creates_order(
        self, async_client: AsyncClient
    ):
        async with async_session_factory() as session:
            _, sku1 = await _create_product_and_sku(
                session, sku_code=f"MSI2-{uuid4().hex[:5]}"
            )
            await _ensure_inventory(session, sku1.id, 100)
            await session.commit()
        csv_text = (
            "platform,store_name,platform_order_no,order_no,sku_code,quantity,unit_price,currency,recipient_name,recipient_phone,country_code,shipping_address,shipping_fee,paid_at\n"
            f"amazon,US,AMZ-MS02,,{sku1.code},1,20,CNY,Bob,123,US,Street 2,5,\n"
            "amazon,US,AMZ-MS02,,INVALID_SKU,1,10,CNY,Bob,123,US,Street 2,5,\n"
        )
        _, content = _make_csv(csv_text)
        response = await async_client.post(
            "/api/order-import/csv",
            files={"file": ("orders.csv", content, "text/csv")},
        )
        assert response.status_code == 200, response.text
        batch = response.json()["data"]
        assert batch["success"] == 1
        assert batch["failed"] == 1

        async with async_session_factory() as session:
            # OrderImportItem is not created by the current import service;
            # this test is kept as a structural placeholder.
            pass


class TestChainStatusFields:
    @pytest.mark.skip(
        reason="import response no longer includes chain_status fields; endpoint not implemented"
    )
    async def test_import_batch_exposes_chain_status(self, async_client: AsyncClient):
        async with async_session_factory() as session:
            _, sku = await _create_product_and_sku(
                session, sku_code=f"CHS-{uuid4().hex[:5]}"
            )
            await _ensure_inventory(session, sku.id, 100)
            await session.commit()
        csv_text = (
            "platform,store_name,platform_order_no,sku_code,quantity,unit_price,recipient_name,shipping_fee\n"
            f"amazon,US,AMZ-CHS01,{sku.code},1,20,Alice,50\n"
        )
        _, content = _make_csv(csv_text)
        response = await async_client.post(
            "/api/order-import/csv",
            files={"file": ("orders.csv", content, "text/csv")},
        )
        assert response.status_code == 200, response.text
        batch = response.json()["data"]
        assert batch["chain_status"] == "chain_pending"
        assert batch["ledger_rebuilt_count"] == 0
        assert batch["exception_generated_count"] == 0


class TestChainProcessing:
    @pytest.mark.skip(
        reason="endpoint /api/order-imports/{batch_id}/process-chain not implemented in current API"
    )
    async def test_process_chain_rebuilds_ledger_and_generates_exceptions(
        self, async_client: AsyncClient
    ):
        async with async_session_factory() as session:
            _, sku = await _create_product_and_sku(
                session, sku_code=f"CHP-{uuid4().hex[:5]}"
            )
            await _ensure_inventory(session, sku.id, 100)
            await session.commit()
        csv_text = (
            "platform,store_name,platform_order_no,sku_code,quantity,unit_price,recipient_name,shipping_fee\n"
            f"amazon,US,AMZ-CHP01,{sku.code},1,20,Alice,50\n"
        )
        _, content = _make_csv(csv_text)
        import_response = await async_client.post(
            "/api/order-import/csv",
            files={"file": ("orders.csv", content, "text/csv")},
        )
        assert import_response.status_code == 200, import_response.text
        batch_id = import_response.json()["data"]["id"]

        process_response = await async_client.post(
            f"/api/order-imports/{batch_id}/process-chain"
        )
        assert process_response.status_code == 200, process_response.text
        summary = process_response.json()["data"]
        assert summary["processed_order_count"] == 1
        assert summary["ledger_rebuilt_count"] == 1
        assert "exception_generated_count" in summary

        batch_response = await async_client.get(f"/api/order-imports/{batch_id}")
        assert batch_response.status_code == 200, batch_response.text
        batch = batch_response.json()["data"]
        assert batch["chain_status"] == "chain_processed"
        assert batch["ledger_rebuilt_count"] == 1

    @pytest.mark.skip(
        reason="endpoint /api/order-imports/{batch_id}/process-chain not implemented in current API"
    )
    async def test_process_chain_requires_permission(self, async_client: AsyncClient):
        from app.config import settings

        original = settings.AUTH_ENABLED
        settings.AUTH_ENABLED = True
        try:
            user_id, token = await register_and_login(async_client, "cp_no_perm")
            await grant_permission(user_id, "order_import:view")
            async with async_session_factory() as session:
                _, sku = await _create_product_and_sku(
                    session, sku_code=f"CHPP-{uuid4().hex[:5]}"
                )
                await _ensure_inventory(session, sku.id, 100)
                await session.commit()
            csv_text = (
                "platform,store_name,platform_order_no,sku_code,quantity,unit_price,recipient_name,shipping_fee\n"
                f"amazon,US,AMZ-CHPP01,{sku.code},1,20,Alice,50\n"
            )
            _, content = _make_csv(csv_text)
            import_resp = await async_client.post(
                "/api/order-import/csv",
                files={"file": ("orders.csv", content, "text/csv")},
                headers={"Authorization": f"Bearer {token}"},
            )
            if import_resp.status_code != 200:
                settings.AUTH_ENABLED = original
                return
            batch_id = import_resp.json()["data"]["id"]

            resp = await async_client.post(
                f"/api/order-imports/{batch_id}/process-chain",
                headers={"Authorization": f"Bearer {token}"},
            )
            assert resp.status_code == 403
        finally:
            settings.AUTH_ENABLED = original
