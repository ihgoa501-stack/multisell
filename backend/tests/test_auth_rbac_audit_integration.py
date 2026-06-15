"""认证、RBAC 权限和审计日志集成测试。"""

from uuid import uuid4

import pytest
from sqlalchemy import select

from app.config import settings
from app.database import async_session_factory
from app.models import Permission, Role, RolePermission, User, UserRole


@pytest.fixture(autouse=True)
def enable_auth_for_test():
    original = settings.AUTH_ENABLED
    settings.AUTH_ENABLED = True
    try:
        yield
    finally:
        settings.AUTH_ENABLED = original


def _product_payload(name: str = None) -> dict:
    return {
        "name": name or f"权限测试商品-{uuid4().hex[:8]}",
        "unit": "件",
        "status": 1,
    }


async def _register_and_login(async_client, username_prefix: str = "rbac_user") -> tuple[int, str]:
    username = f"{username_prefix}_{uuid4().hex[:8]}"
    resp = await async_client.post(
        "/api/auth/register",
        json={
            "username": username,
            "password": "testpass123",
            "display_name": username,
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]["user"]["id"], data["data"]["access_token"]


async def _grant_permission(user_id: int, permission_code: str):
    async with async_session_factory() as session:
        # 幂等：检查是否已有该权限
        from sqlalchemy import select as _select
        existing = await session.scalar(
            _select(Permission).where(Permission.code == permission_code).limit(1)
        )
        if existing:
            permission = existing
        else:
            suffix = uuid4().hex[:8]
            permission = Permission(
                name=f"测试权限 {permission_code} {suffix}",
                code=permission_code,
                module=permission_code.split(":")[0] if ":" in permission_code else "product",
            )
            session.add(permission)
            await session.flush()

        suffix = uuid4().hex[:8]
        role = Role(
            name=f"测试角色 {suffix}",
            code=f"test_role_{suffix}",
        )
        session.add(role)
        await session.flush()
        session.add(RolePermission(role_id=role.id, permission_id=permission.id))
        session.add(UserRole(user_id=user_id, role_id=role.id))
        await session.commit()


async def _set_admin_role():
    async with async_session_factory() as session:
        result = await session.execute(select(User).where(User.username == "admin"))
        admin = result.scalar_one()
        admin.role = "admin"
        await session.commit()


class TestAuthRbacAuditIntegration:
    async def test_product_create_requires_login_when_auth_enabled(self, async_client):
        resp = await async_client.post("/api/products", json=_product_payload())

        assert resp.status_code == 401
        assert resp.json()["code"] == 401

    async def test_user_without_product_create_permission_is_forbidden(self, async_client):
        _user_id, token = await _register_and_login(async_client, "no_perm")

        resp = await async_client.post(
            "/api/products",
            json=_product_payload(),
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403
        assert resp.json()["code"] == 403

    async def test_product_create_permission_allows_write_and_records_audit_log(self, async_client):
        user_id, token = await _register_and_login(async_client, "with_perm")
        await _grant_permission(user_id, "product:create")
        await _grant_permission(user_id, "operation_log:view")
        product_name = f"审计商品-{uuid4().hex[:8]}"

        resp = await async_client.post(
            "/api/products",
            json=_product_payload(product_name),
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["name"] == product_name

        logs_resp = await async_client.get(
            "/api/operation-logs?module=product&action=create",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert logs_resp.status_code == 200
        logs = logs_resp.json()["records"]
        assert any(
            log["resource_id"] == str(data["data"]["id"])
            and log["operator"].startswith("with_perm_")
            and product_name in log["content"]
            for log in logs
        )

    async def test_admin_role_bypasses_explicit_permission(self, async_client):
        await _set_admin_role()
        login_resp = await async_client.post(
            "/api/auth/login",
            json={"username": "admin", "password": "admin123"},
        )
        assert login_resp.status_code == 200
        token = login_resp.json()["data"]["access_token"]

        resp = await async_client.post(
            "/api/products",
            json=_product_payload(),
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 200
        assert resp.json()["code"] == 200
