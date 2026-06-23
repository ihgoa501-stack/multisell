"""管理面（Dashboard / 报表 / RBAC / 操作日志）— 权限测试。"""

import pytest

from tests.auth_helpers import register_and_login, grant_permission


pytestmark = pytest.mark.usefixtures("enable_auth")


class TestDashboardReportAuth:
    async def test_dashboard_stats_requires_dashboard_view(self, async_client):
        _uid, token = await register_and_login(async_client, "db_vw")
        resp = await async_client.get("/api/dashboard/stats", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_product_stats_requires_report_view(self, async_client):
        _uid, token = await register_and_login(async_client, "rp_vw1")
        resp = await async_client.get("/api/reports/product-stats", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_platform_stats_requires_report_view(self, async_client):
        _uid, token = await register_and_login(async_client, "rp_vw2")
        resp = await async_client.get("/api/reports/platform-stats", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_dashboard_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "db_ok")
        await grant_permission(uid, "dashboard:view")
        resp = await async_client.get("/api/dashboard/stats", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 200

    async def test_report_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "rp_ok")
        await grant_permission(uid, "report:view")
        resp = await async_client.get("/api/reports/product-stats", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 200


class TestRbacAuth:
    async def test_list_roles_requires_rbac_view(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_vw1")
        resp = await async_client.get("/api/rbac/roles", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_list_permissions_requires_rbac_view(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_vw2")
        resp = await async_client.get("/api/rbac/permissions", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_get_role_requires_rbac_view(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_vw3")
        resp = await async_client.get("/api/rbac/roles/1", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_get_permission_requires_rbac_view(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_vw4")
        resp = await async_client.get("/api/rbac/permissions/1", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_list_users_requires_rbac_view(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_vw5")
        resp = await async_client.get("/api/rbac/users", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_get_user_permissions_requires_rbac_view(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_vw6")
        resp = await async_client.get("/api/rbac/users/1/permissions", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_create_role_requires_rbac_manage(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_mg1")
        resp = await async_client.post(
            "/api/rbac/roles",
            json={"name": "x", "code": "x"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_update_role_requires_rbac_manage(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_mg2")
        resp = await async_client.put(
            "/api/rbac/roles/1",
            json={"name": "x"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_delete_role_requires_rbac_manage(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_mg3")
        resp = await async_client.delete("/api/rbac/roles/1", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_create_permission_requires_rbac_manage(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_mg4")
        resp = await async_client.post(
            "/api/rbac/permissions",
            json={"name": "x", "code": "x:x", "module": "x"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_assign_role_permissions_requires_rbac_manage(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_mg5")
        resp = await async_client.post(
            "/api/rbac/roles/1/permissions",
            json={"role_ids": [1]},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_assign_user_roles_requires_rbac_manage(self, async_client):
        _uid, token = await register_and_login(async_client, "rb_mg6")
        resp = await async_client.post(
            "/api/rbac/users/1/roles",
            json={"role_ids": [1]},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_rbac_view_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "rb_ok1")
        await grant_permission(uid, "rbac:view")
        resp = await async_client.get("/api/rbac/roles", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 200

    async def test_rbac_manage_with_permission_succeeds(self, async_client):
        """注意：此测试需要先有 rbac:view 读取已有权限，再用 rbac:manage 写入。"""
        from uuid import uuid4
        uid, token = await register_and_login(async_client, "rb_ok2")
        await grant_permission(uid, "rbac:manage")
        suffix = uuid4().hex[:4]
        resp = await async_client.post(
            "/api/rbac/roles",
            json={"name": f"测试角色-{suffix}", "code": f"test_{suffix}"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200


class TestOperationLogAuth:
    async def test_list_logs_requires_operation_log_view(self, async_client):
        _uid, token = await register_and_login(async_client, "ol_vw1")
        resp = await async_client.get("/api/operation-logs", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_log_modules_requires_operation_log_view(self, async_client):
        _uid, token = await register_and_login(async_client, "ol_vw2")
        resp = await async_client.get("/api/operation-logs/modules", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 403

    async def test_list_logs_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "ol_ok")
        await grant_permission(uid, "operation_log:view")
        resp = await async_client.get("/api/operation-logs", headers={"Authorization": f"Bearer {token}"})
        assert resp.status_code == 200
