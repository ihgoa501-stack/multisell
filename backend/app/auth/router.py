"""认证 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result
from app.auth.schemas import UserRegister, UserLogin, UserVO, TokenVO
from app.auth.service import AuthService
from app.auth.dependencies import get_current_user
from app.models import User

router = APIRouter(tags=["认证"])


def user_to_vo(user: User) -> UserVO:
    return UserVO(
        id=user.id,
        username=user.username,
        display_name=user.display_name,
        role=user.role,
        email=user.email,
        status=user.status,
        created_at=user.created_at,
    )


@router.post("/auth/register", summary="用户注册")
async def register(data: UserRegister, db: AsyncSession = Depends(get_db)):
    try:
        user, token = await AuthService.register(
            db, data.username, data.password, data.display_name, data.email
        )
        return Result.ok(TokenVO(
            access_token=token,
            user=user_to_vo(user),
        ).model_dump())
    except ValueError as e:
        return Result.bad_request(str(e))


@router.post("/auth/login", summary="用户登录")
async def login(data: UserLogin, db: AsyncSession = Depends(get_db)):
    try:
        user, token = await AuthService.login(db, data.username, data.password)
        return Result.ok(TokenVO(
            access_token=token,
            user=user_to_vo(user),
        ).model_dump())
    except ValueError as e:
        return Result.bad_request(str(e))


@router.get("/auth/me", summary="获取当前用户信息")
async def get_me(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    # 加载权限列表
    permissions: list[str] = []
    if current_user.role == "admin":
        # admin 拥有所有权限，返回空列表表示无限制
        permissions = []
    else:
        from sqlalchemy import select
        from app.models import Permission, RolePermission, UserRole
        stmt = (
            select(Permission.code)
            .join(RolePermission, RolePermission.permission_id == Permission.id)
            .join(UserRole, UserRole.role_id == RolePermission.role_id)
            .where(UserRole.user_id == current_user.id)
        )
        result = await db.execute(stmt)
        seen: set[str] = set()
        for (code,) in result.all():
            if code and code not in seen:
                seen.add(code)
                permissions.append(code)

    vo = user_to_vo(current_user)
    vo.permissions = permissions
    return Result.ok(vo)
