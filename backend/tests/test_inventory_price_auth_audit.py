"""库存 & 价格模块 — 权限与审计测试。"""

from uuid import uuid4

import pytest

from tests.auth_helpers import register_and_login, grant_permission


pytestmark = pytest.mark.usefixtures("enable_auth")


class TestInventoryAuthAudit:
    async def _create_sku(self, async_client) -> int:
        """创建测试 SKU（admin）"""
        from tests.auth_helpers import set_admin_role
        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        token = login_resp.json()["data"]["access_token"]
        headers = {"Authorization": f"Bearer {token}"}
        resp = await async_client.post(
            "/api/products",
            json={"name": f"库存测试-{uuid4().hex[:8]}", "unit": "件", "status": 1},
            headers=headers,
        )
        product = resp.json()["data"]
        await async_client.post(
            f"/api/products/{product['id']}/specs",
            json={"specs": [{"name": "容量", "values": ["大"]}]},
            headers=headers,
        )
        sku_resp = await async_client.post(
            f"/api/products/{product['id']}/skus/generate",
            headers=headers,
        )
        return sku_resp.json()["data"]["skus"][0]["id"]

    async def test_inventory_view_requires_login(self, async_client):
        resp = await async_client.get("/api/inventory/1")
        assert resp.status_code == 401

    async def test_inventory_view_without_permission_is_forbidden(self, async_client):
        _uid, token = await register_and_login(async_client, "inv_vw")
        resp = await async_client.get("/api/inventory/1", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_inventory_view_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "inv_ok")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "inventory:view")
        resp = await async_client.get(
            f"/api/inventory/{sku_id}",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200

    async def test_inventory_alerts_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "inv_alt")
        resp = await async_client.get("/api/inventory/alerts", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_inventory_logs_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "inv_log")
        resp = await async_client.get("/api/inventory/1/logs", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_inventory_check_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "inv_chk")
        resp = await async_client.post(
            "/api/inventory/check",
            json={"sku_id": 1, "quantity": 1},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_inventory_update_requires_update_permission(self, async_client):
        _uid, token = await register_and_login(async_client, "inv_up")
        resp = await async_client.put(
            "/api/inventory/1",
            json={"quantity": 10},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_inventory_update_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await register_and_login(async_client, "inv_up2")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "inventory:view")
        await grant_permission(uid, "inventory:update")
        await grant_permission(uid, "operation_log:view")
        resp = await async_client.put(
            f"/api/inventory/{sku_id}",
            json={"quantity": 50, "remark": "入库"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        # 审计日志
        logs_resp = await async_client.get(
            "/api/operation-logs?module=inventory&action=update",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200
        logs = logs_resp.json()["records"]
        assert any(str(sku_id) in log["resource_id"] for log in logs)


class TestPriceAuthAudit:
    async def _create_sku(self, async_client) -> int:
        from tests.auth_helpers import set_admin_role
        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        token = login_resp.json()["data"]["access_token"]
        headers = {"Authorization": f"Bearer {token}"}
        resp = await async_client.post(
            "/api/products",
            json={"name": f"价格测试-{uuid4().hex[:8]}", "unit": "件", "status": 1},
            headers=headers,
        )
        product = resp.json()["data"]
        await async_client.post(
            f"/api/products/{product['id']}/specs",
            json={"specs": [{"name": "大小", "values": ["中"]}]},
            headers=headers,
        )
        sku_resp = await async_client.post(
            f"/api/products/{product['id']}/skus/generate",
            headers=headers,
        )
        return sku_resp.json()["data"]["skus"][0]["id"]

    async def test_price_view_requires_login(self, async_client):
        resp = await async_client.get("/api/skus/1/prices")
        assert resp.status_code == 401

    async def test_price_view_without_permission_is_forbidden(self, async_client):
        _uid, token = await register_and_login(async_client, "prc_vw")
        resp = await async_client.get("/api/skus/1/prices", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_current_price_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "prc_cu")
        resp = await async_client.get("/api/skus/1/current-price", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_price_history_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "prc_hi")
        resp = await async_client.get("/api/skus/1/price-history", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_price_set_requires_update_permission(self, async_client):
        _uid, token = await register_and_login(async_client, "prc_se")
        resp = await async_client.post(
            "/api/prices",
            json={"sku_id": 1, "price_type": "sale_price", "price": 10},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_price_set_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await register_and_login(async_client, "prc_ok")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "price:update")
        await grant_permission(uid, "operation_log:view")
        resp = await async_client.post(
            "/api/prices",
            json={"sku_id": sku_id, "price_type": "sale_price", "price": 99.0},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=price&action=set_price",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200

    async def test_price_batch_requires_batch_update_permission(self, async_client):
        _uid, token = await register_and_login(async_client, "prc_bt")
        resp = await async_client.post(
            "/api/prices/batch",
            json={"sku_ids": [1, 2], "price_type": "sale_price", "price": 10},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_price_batch_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await register_and_login(async_client, "prc_bt2")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "price:batch_update")
        await grant_permission(uid, "operation_log:view")
        resp = await async_client.post(
            "/api/prices/batch",
            json={"sku_ids": [sku_id], "price_type": "sale_price", "price": 88.0},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=price&action=batch_update",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200

    async def test_admin_bypasses_price_permission(self, async_client):
        from tests.auth_helpers import set_admin_role
        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        token = login_resp.json()["data"]["access_token"]
        sku_id = await self._create_sku(async_client)
        resp = await async_client.post(
            "/api/prices",
            json={"sku_id": sku_id, "price_type": "sale_price", "price": 50},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200
