"""认证 - 路由"""

from fastapi import APIRouter, Depends, Header, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result
from app.auth.schemas import UserRegister, UserLogin, UserVO, TokenVO
from app.auth.service import AuthService
from app.config import settings
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


async def get_current_user(
    authorization: str = Header(None),
    db: AsyncSession = Depends(get_db),
) -> User:
    """依赖注入：获取当前登录用户。
    当 AUTH_ENABLED=False 时返回一个 mock 用户，跳过鉴权。
    """
    if not settings.AUTH_ENABLED:
        mock_user = User(
            id=0, username="system", display_name="系统管理员",
            role="admin", status=1,
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
async def get_me(current_user: User = Depends(get_current_user)):
    return Result.ok(user_to_vo(current_user))
