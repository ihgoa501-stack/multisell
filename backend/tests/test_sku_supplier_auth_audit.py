"""SKU & 供应商模块 — 权限与审计测试。"""

from uuid import uuid4

import pytest

from tests.auth_helpers import register_and_login, grant_permission


pytestmark = pytest.mark.usefixtures("enable_auth")


def _admin_headers(async_client):
    """获取 admin token + headers 用于测试数据准备。"""
    # set_admin_role is async
    return None  # 不在此处执行


class TestSkuAuthAudit:
    async def _admin_token(self, async_client) -> tuple[int, str]:
        from tests.auth_helpers import set_admin_role

        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        data = login_resp.json()["data"]
        return data["user"]["id"], data["access_token"]

    async def _create_product(self, async_client, token: str) -> int:
        headers = {"Authorization": f"Bearer {token}"}
        resp = await async_client.post(
            "/api/products",
            json={"name": f"SKU测试-{uuid4().hex[:8]}", "unit": "件", "status": 1},
            headers=headers,
        )
        return resp.json()["data"]["id"]

    # --- sku:view ---
    async def test_get_specs_requires_sku_view(self, async_client):
        _uid, token = await register_and_login(async_client, "sk_vw1")
        resp = await async_client.get(
            "/api/products/1/specs",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_get_skus_requires_sku_view(self, async_client):
        _uid, token = await register_and_login(async_client, "sk_vw2")
        resp = await async_client.get(
            "/api/products/1/skus",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_get_sku_detail_requires_sku_view(self, async_client):
        _uid, token = await register_and_login(async_client, "sk_vw3")
        resp = await async_client.get(
            "/api/skus/1",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    # --- sku:create ---
    async def test_define_specs_requires_sku_create(self, async_client):
        _uid, token = await register_and_login(async_client, "sk_cr1")
        resp = await async_client.post(
            "/api/products/1/specs",
            json={"specs": [{"name": "颜色", "values": ["红"]}]},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_generate_skus_requires_sku_create(self, async_client):
        _uid, token = await register_and_login(async_client, "sk_cr2")
        resp = await async_client.post(
            "/api/products/1/skus/generate",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_define_specs_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await self._admin_token(async_client)
        await grant_permission(uid, "sku:create")
        pid = await self._create_product(async_client, token)
        resp = await async_client.post(
            f"/api/products/{pid}/specs",
            json={"specs": [{"name": "颜色", "values": ["红", "蓝"]}]},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=sku&action=define_specs",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200

    async def test_generate_skus_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await self._admin_token(async_client)
        await grant_permission(uid, "sku:create")
        pid = await self._create_product(async_client, token)
        # create specs first
        await async_client.post(
            f"/api/products/{pid}/specs",
            json={"specs": [{"name": "大小", "values": ["大"]}]},
            headers={"Authorization": f"Bearer {token}"},
        )
        resp = await async_client.post(
            f"/api/products/{pid}/skus/generate",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=sku&action=generate_skus",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200

    # --- sku:update ---
    async def test_update_sku_requires_sku_update(self, async_client):
        _uid, token = await register_and_login(async_client, "sk_up1")
        resp = await async_client.put(
            "/api/skus/1",
            json={"price": 100},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_update_sku_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await self._admin_token(async_client)
        pid = await self._create_product(async_client, token)
        # create specs + generate sku
        await async_client.post(
            f"/api/products/{pid}/specs",
            json={"specs": [{"name": "尺寸", "values": ["M"]}]},
            headers={"Authorization": f"Bearer {token}"},
        )
        sku_resp = await async_client.post(
            f"/api/products/{pid}/skus/generate",
            headers={"Authorization": f"Bearer {token}"},
        )
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]

        # Now test as the permission-granted user
        await grant_permission(uid, "sku:update")
        resp = await async_client.put(
            f"/api/skus/{sku_id}",
            json={"price": 199},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=sku&action=update",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200


class TestSupplierAuthAudit:
    async def _admin_token(self, async_client) -> tuple[int, str]:
        from tests.auth_helpers import set_admin_role

        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        data = login_resp.json()["data"]
        return data["user"]["id"], data["access_token"]

    # --- supplier:view ---
    async def test_list_suppliers_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "sp_vw1")
        resp = await async_client.get(
            "/api/suppliers", headers={"Authorization": f"Bearer {token}"}
        )
        assert resp.status_code == 403

    async def test_get_supplier_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "sp_vw2")
        resp = await async_client.get(
            "/api/suppliers/1", headers={"Authorization": f"Bearer {token}"}
        )
        assert resp.status_code == 403

    async def test_product_suppliers_requires_view(self, async_client):
        _uid, token = await register_and_login(async_client, "sp_vw3")
        resp = await async_client.get(
            "/api/products/1/suppliers", headers={"Authorization": f"Bearer {token}"}
        )
        assert resp.status_code == 403

    # --- supplier:create ---
    async def test_create_supplier_requires_create(self, async_client):
        _uid, token = await register_and_login(async_client, "sp_cr1")
        resp = await async_client.post(
            "/api/suppliers",
            json={"name": "测试供应商"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_bind_product_supplier_requires_create(self, async_client):
        _uid, token = await register_and_login(async_client, "sp_cr2")
        resp = await async_client.post(
            "/api/product-supplier",
            json={"product_id": 1, "supplier_id": 1},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_create_supplier_with_permission_succeeds_and_logs(
        self, async_client
    ):
        uid, token = await self._admin_token(async_client)
        await grant_permission(uid, "supplier:create")
        resp = await async_client.post(
            "/api/suppliers",
            json={"name": f"测试供应-{uuid4().hex[:8]}", "contact_person": "张三"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=supplier&action=create",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200

    # --- supplier:update ---
    async def test_update_supplier_requires_update(self, async_client):
        _uid, token = await register_and_login(async_client, "sp_up1")
        resp = await async_client.put(
            "/api/suppliers/1",
            json={"contact_person": "李四"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_update_supplier_with_permission_succeeds_and_logs(
        self, async_client
    ):
        uid, token = await self._admin_token(async_client)
        await grant_permission(uid, "supplier:create")
        await grant_permission(uid, "supplier:update")
        # create first
        create_resp = await async_client.post(
            "/api/suppliers",
            json={"name": f"更新测试-{uuid4().hex[:8]}"},
            headers={"Authorization": f"Bearer {token}"},
        )
        sid = create_resp.json()["data"]["id"]
        resp = await async_client.put(
            f"/api/suppliers/{sid}",
            json={"contact_person": "李四"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=supplier&action=update",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200

    # --- supplier:delete ---
    async def test_delete_supplier_requires_delete(self, async_client):
        _uid, token = await register_and_login(async_client, "sp_dl1")
        resp = await async_client.delete(
            "/api/suppliers/1", headers={"Authorization": f"Bearer {token}"}
        )
        assert resp.status_code == 403

    async def test_unbind_product_supplier_requires_delete(self, async_client):
        _uid, token = await register_and_login(async_client, "sp_dl2")
        resp = await async_client.delete(
            "/api/product-supplier?product_id=1&supplier_id=1",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_delete_supplier_with_permission_succeeds_and_logs(
        self, async_client
    ):
        uid, token = await self._admin_token(async_client)
        await grant_permission(uid, "supplier:create")
        await grant_permission(uid, "supplier:delete")
        create_resp = await async_client.post(
            "/api/suppliers",
            json={"name": f"删除测试-{uuid4().hex[:8]}"},
            headers={"Authorization": f"Bearer {token}"},
        )
        sid = create_resp.json()["data"]["id"]
        resp = await async_client.delete(
            f"/api/suppliers/{sid}",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=supplier&action=delete",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200
