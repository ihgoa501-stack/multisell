"""认证 & RBAC 测试助手 — 可复用函数和 fixture。"""

from uuid import uuid4

import pytest
from sqlalchemy import select

from app.config import settings
from app.database import async_session_factory
from app.models import Permission, Role, RolePermission, User, UserRole


@pytest.fixture(autouse=True)
def enable_auth():
    """将此 fixture 放入测试文件后，该文件中所有测试都会启用 AUTH_ENABLED。"""
    original = settings.AUTH_ENABLED
    settings.AUTH_ENABLED = True
    try:
        yield
    finally:
        settings.AUTH_ENABLED = original


async def register_and_login(
    async_client, username_prefix: str = "rbac_user"
) -> tuple[int, str]:
    """注册一个新用户并返回 (user_id, access_token)。"""
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


async def grant_permission(user_id: int, permission_code: str):
    """为用户授予指定的权限码（幂等）。"""
    async with async_session_factory() as session:
        # 检查是否已有该权限
        existing_perm = await session.scalar(
            select(Permission).where(Permission.code == permission_code).limit(1)
        )
        if existing_perm:
            permission = existing_perm
        else:
            suffix = uuid4().hex[:8]
            permission = Permission(
                name=f"测试权限 {permission_code} {suffix}",
                code=permission_code,
                module=permission_code.split(":")[0]
                if ":" in permission_code
                else "test",
            )
            session.add(permission)
            await session.flush()

        # 创建一个角色并关联权限
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


async def set_admin_role(username: str = "admin"):
    """将指定用户的 role 字段设为 admin。"""
    async with async_session_factory() as session:
        result = await session.execute(select(User).where(User.username == username))
        user = result.scalar_one()
        user.role = "admin"
        await session.commit()
