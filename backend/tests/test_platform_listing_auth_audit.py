"""平台 & 发布模块 — 权限与审计测试。"""

from uuid import uuid4

import pytest

from tests.auth_helpers import register_and_login, grant_permission


pytestmark = pytest.mark.usefixtures("enable_auth")


class TestPlatformAuthAudit:
    async def _admin_token(self, async_client) -> tuple[int, str]:
        from tests.auth_helpers import set_admin_role
        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        data = login_resp.json()["data"]
        return data["user"]["id"], data["access_token"]

    # --- platform:view ---
    async def test_list_platforms_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "pl_vw1")
        resp = await async_client.get("/api/platforms", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_get_platform_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "pl_vw2")
        resp = await async_client.get("/api/platforms/1", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    # --- platform:create ---
    async def test_create_platform_requires_create(self, async_client):
        _uid, token = await register_and_login(async_client, "pl_cr1")
        resp = await async_client.post(
            "/api/platforms",
            json={"name": "Test", "code": "test"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_create_platform_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await self._admin_token(async_client)
        await grant_permission(uid, "platform:create")
        resp = await async_client.post(
            "/api/platforms",
            json={"name": f"测试平台-{uuid4().hex[:8]}", "code": f"t_{uuid4().hex[:4]}"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=platform&action=create",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200

    # --- platform:update ---
    async def test_update_platform_requires_update(self, async_client):
        _uid, token = await register_and_login(async_client, "pl_up1")
        resp = await async_client.put(
            "/api/platforms/1",
            json={"name": "更新"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    # --- platform:delete ---
    async def test_delete_platform_requires_delete(self, async_client):
        _uid, token = await register_and_login(async_client, "pl_dl1")
        resp = await async_client.delete("/api/platforms/1", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403


class TestListingAuthAudit:
    async def _admin_token(self, async_client) -> tuple[int, str]:
        from tests.auth_helpers import set_admin_role
        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        data = login_resp.json()["data"]
        return data["user"]["id"], data["access_token"]

    async def _create_sku_and_platform(self, async_client, token: str) -> tuple[int, int]:
        """创建商品+SKU 和平台，返回 (product_id, platform_id)"""
        headers = {"Authorization": f"Bearer {token}"}
        # 商品
        resp = await async_client.post(
            "/api/products",
            json={
                "name": f"发布测试-{uuid4().hex[:8]}",
                "unit": "件",
                "status": 1,
                "main_image": "https://example.com/img.jpg",
                "package_length_cm": 20.0,
                "package_width_cm": 12.0,
                "package_height_cm": 8.0,
                "package_weight_kg": 1.5,
            },
            headers=headers,
        )
        product = resp.json()["data"]
        pid = product["id"]
        # 设置SKU+价格+库存
        await async_client.post(
            f"/api/products/{pid}/specs",
            json={"specs": [{"name": "颜色", "values": ["白"]}]},
            headers=headers,
        )
        sku_resp = await async_client.post(
            f"/api/products/{pid}/skus/generate",
            headers=headers,
        )
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]
        await async_client.post(
            "/api/prices",
            json={"sku_id": sku_id, "price_type": "sale_price", "price": 50},
            headers=headers,
        )
        await async_client.put(
            f"/api/inventory/{sku_id}",
            json={"quantity": 100},
            headers=headers,
        )
        # 平台
        plat_resp = await async_client.post(
            "/api/platforms",
            json={"name": f"测试平台-{uuid4().hex[:8]}", "code": f"t_{uuid4().hex[:4]}"},
            headers=headers,
        )
        platform_id = plat_resp.json()["data"]["id"]
        return pid, platform_id

    # --- listing:view ---
    async def test_get_product_listings_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "li_vw1")
        resp = await async_client.get("/api/products/1/listings", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_get_all_listings_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "li_vw2")
        resp = await async_client.get("/api/listings", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    # --- listing:publish ---
    async def test_publish_requires_publish_permission(self, async_client):
        _uid, token = await register_and_login(async_client, "li_pb1")
        resp = await async_client.post(
            "/api/products/1/publish/1",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_publish_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await self._admin_token(async_client)
        await grant_permission(uid, "listing:publish")
        pid, plat_id = await self._create_sku_and_platform(async_client, token)
        resp = await async_client.post(
            f"/api/products/{pid}/publish/{plat_id}",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=listing&action=publish",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200
