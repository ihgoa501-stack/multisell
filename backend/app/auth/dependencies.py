"""认证和权限依赖。"""

from collections.abc import Callable

from fastapi import Depends, Header, HTTPException
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth.service import AuthService
from app.config import settings
from app.database import get_db
from app.models import Permission, Role, RolePermission, User, UserRole


async def get_current_user(
    authorization: str = Header(None),
    db: AsyncSession = Depends(get_db),
) -> User:
    """获取当前登录用户。AUTH_ENABLED=False 时返回系统用户。"""
    if not settings.AUTH_ENABLED:
        mock_user = User(
            id=0,
            username="system",
            display_name="系统管理员",
            role="admin",
            status=1,
        )
        mock_user.roles = []
        return mock_user

    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="未登录")

    token = authorization[7:]
    user = await AuthService.get_current_user(db, token)
    if not user:
        raise HTTPException(status_code=401, detail="登录已过期")
    if user.status != 1:
        raise HTTPException(status_code=403, detail="账号已被禁用")
    return user


def require_auth(current_user: User = Depends(get_current_user)) -> User:
    return current_user


def require_permission(permission_code: str) -> Callable:
    async def dependency(
        current_user: User = Depends(get_current_user),
        db: AsyncSession = Depends(get_db),
    ) -> User:
        if not settings.AUTH_ENABLED or current_user.role == "admin":
            return current_user

        stmt = (
            select(Permission.id)
            .join(RolePermission, RolePermission.permission_id == Permission.id)
            .join(Role, Role.id == RolePermission.role_id)
            .join(UserRole, UserRole.role_id == Role.id)
            .where(
                UserRole.user_id == current_user.id,
                Role.status == 1,
                Permission.code == permission_code,
            )
            .limit(1)
        )
        has_permission = await db.scalar(stmt)
        if not has_permission:
            raise HTTPException(status_code=403, detail="无权限")
        return current_user

    return dependency
