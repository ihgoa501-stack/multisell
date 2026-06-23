"""订单模块 — 权限与审计测试。"""

from uuid import uuid4

import pytest

from tests.auth_helpers import register_and_login, grant_permission


pytestmark = pytest.mark.usefixtures("enable_auth")


def _order_payload(sku_id: int) -> dict:
    return {
        "recipient_name": "李四",
        "recipient_phone": "13900139000",
        "shipping_address": "广州市测试路 2 号",
        "payment_method": "mock",
        "shipping_fee": 10,
        "remark": "权限测试订单",
        "items": [{"sku_id": sku_id, "quantity": 1}],
    }


class TestOrderAuthAudit:
    async def _create_sku(self, async_client) -> int:
        """创建测试用的 SKU（使用 admin 绕过权限）"""
        from tests.auth_helpers import set_admin_role

        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        assert login_resp.status_code == 200
        token = login_resp.json()["data"]["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        resp = await async_client.post(
            "/api/products",
            json={
                "name": f"订单权限测试商品-{uuid4().hex[:8]}",
                "unit": "件",
                "status": 1,
            },
            headers=headers,
        )
        assert resp.status_code == 200
        product = resp.json()["data"]
        await async_client.post(
            f"/api/products/{product['id']}/specs",
            json={"specs": [{"name": "颜色", "values": ["红色"]}]},
            headers=headers,
        )
        sku_resp = await async_client.post(
            f"/api/products/{product['id']}/skus/generate",
            headers=headers,
        )
        assert sku_resp.status_code == 200
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]
        # 设置库存（管理员权限）
        inv_resp = await async_client.put(
            f"/api/inventory/{sku_id}",
            json={"quantity": 20},
            headers=headers,
        )
        assert inv_resp.status_code == 200, inv_resp.text
        return sku_id

    async def test_create_order_requires_login(self, async_client):
        resp = await async_client.post("/api/orders", json=_order_payload(1))
        assert resp.status_code == 401

    async def test_create_order_without_permission_is_forbidden(self, async_client):
        _uid, token = await register_and_login(async_client, "ord_no_perm")
        sku_id = await self._create_sku(async_client)
        resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_create_order_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await register_and_login(async_client, "ord_cre")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        await grant_permission(uid, "operation_log:view")

        resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

        # 验证审计日志
        logs_resp = await async_client.get(
            "/api/operation-logs?module=order&action=create",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200
        logs = logs_resp.json()["records"]
        assert any(log["resource_id"] == str(data["data"]["id"]) for log in logs)

    async def test_list_orders_requires_order_view(self, async_client):
        _uid, token = await register_and_login(async_client, "ord_list")
        resp = await async_client.get(
            "/api/orders",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_list_orders_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "ord_view")
        await grant_permission(uid, "order:view")
        resp = await async_client.get(
            "/api/orders",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200

    async def test_update_status_requires_update_status_permission(self, async_client):
        uid, token = await register_and_login(async_client, "ord_up1")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        # 没有 order:update_status → 403
        resp = await async_client.put(
            f"/api/orders/{order_id}/status",
            json={"status": "paid"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_cancel_requires_cancel_permission(self, async_client):
        uid, token = await register_and_login(async_client, "ord_can")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        await grant_permission(uid, "order:update_status")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        # 没有 order:cancel → 403
        resp = await async_client.put(
            f"/api/orders/{order_id}/status",
            json={"status": "cancelled"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_update_status_with_permission_succeeds_and_logs(self, async_client):
        uid, token = await register_and_login(async_client, "ord_up2")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        await grant_permission(uid, "order:update_status")
        await grant_permission(uid, "operation_log:view")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/orders/{order_id}/status",
            json={"status": "paid"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        # 审计日志
        logs_resp = await async_client.get(
            "/api/operation-logs?module=order&action=update_status",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200
        logs = logs_resp.json()["records"]
        assert any(log["resource_id"] == str(order_id) for log in logs)

    async def test_bind_shipping_quote_requires_order_update(self, async_client):
        uid, token = await register_and_login(async_client, "ord_ship_no_perm")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.post(
            f"/api/orders/{order_id}/shipping-quote",
            json={
                "sku_id": sku_id,
                "quantity": 1,
                "destination_country": "US",
                "cargo_type": "normal",
            },
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_update_profit_inputs_requires_order_update(self, async_client):
        uid, token = await register_and_login(async_client, "ord_profit_no_perm")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/orders/{order_id}/profit-inputs",
            json={"platform_fee": 10},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_update_profit_inputs_with_permission_logs(self, async_client):
        uid, token = await register_and_login(async_client, "ord_profit_ok")
        sku_id = await self._create_sku(async_client)
        await grant_permission(uid, "order:create")
        await grant_permission(uid, "order:update")
        await grant_permission(uid, "operation_log:view")
        order_resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        order_id = order_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/orders/{order_id}/profit-inputs",
            json={"platform_fee": 10, "payment_fee": 2},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        logs_resp = await async_client.get(
            "/api/operation-logs?module=order&action=update_profit_inputs",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200
        assert any(
            log["resource_id"] == str(order_id) for log in logs_resp.json()["records"]
        )

    async def test_admin_bypasses_order_permission(self, async_client):
        from tests.auth_helpers import set_admin_role

        await set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        token = login_resp.json()["data"]["access_token"]
        sku_id = await self._create_sku(async_client)

        resp = await async_client.post(
            "/api/orders",
            json=_order_payload(sku_id),
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200
