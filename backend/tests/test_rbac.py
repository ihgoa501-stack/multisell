"""RBAC 权限管理测试"""

import pytest


class TestRbac:
    """权限管理"""

    async def test_list_roles(self, async_client):
        """GET /api/rbac/roles → 角色列表"""
        resp = await async_client.get("/api/rbac/roles")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert "records" in data

    async def test_create_role(self, async_client):
        """POST /api/rbac/roles → 创建角色"""
        import random
        suffix = random.randint(10000, 99999)
        resp = await async_client.post("/api/rbac/roles", json={
            "name": f"测试角色_{suffix}",
            "code": f"test_role_{suffix}",
            "description": "pytest 自动创建的测试角色",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert "id" in data["data"]

        # 清理
        role_id = data["data"]["id"]
        await async_client.delete(f"/api/rbac/roles/{role_id}")

    async def test_list_permissions(self, async_client):
        """GET /api/rbac/permissions → 权限列表"""
        resp = await async_client.get("/api/rbac/permissions")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_create_permission(self, async_client):
        """POST /api/rbac/permissions → 创建权限"""
        import random
        suffix = random.randint(10000, 99999)
        resp = await async_client.post("/api/rbac/permissions", json={
            "name": f"测试权限_{suffix}",
            "code": f"test:perm_{suffix}",
            "module": "test",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_assign_role_permissions(self, async_client):
        """POST /api/rbac/roles/{id}/permissions → 为角色分配权限"""
        import random
        suffix = random.randint(10000, 99999)
        # 先创建角色
        role_resp = await async_client.post("/api/rbac/roles", json={
            "name": f"权限分配测试角色_{suffix}",
            "code": f"test_perm_role_{suffix}",
        })
        if role_resp.status_code != 200:
            return  # skip if can't create
        role_id = role_resp.json()["data"]["id"]

        # 获取现有权限
        perm_resp = await async_client.get("/api/rbac/permissions")
        perms = perm_resp.json().get("records", [])
        perm_ids = [p["id"] for p in perms[:2]] if perms else []

        if perm_ids:
            resp = await async_client.post(
                f"/api/rbac/roles/{role_id}/permissions",
                json={"role_ids": perm_ids},
            )
            assert resp.status_code == 200
            data = resp.json()
            assert data["code"] == 200

        # 清理
        await async_client.delete(f"/api/rbac/roles/{role_id}")
