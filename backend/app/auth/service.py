"""认证 - 服务层"""

from datetime import datetime, timedelta, timezone
from typing import Optional
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from jose import JWTError, jwt
from passlib.context import CryptContext
from app.config import settings
from app.models import User

# 密码加密
pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")

# JWT配置
SECRET_KEY = settings.ENCRYPTION_KEY
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 60 * 24 * 7  # 7天


def hash_password(password: str) -> str:
    return pwd_context.hash(password)


def verify_password(plain: str, hashed: str) -> bool:
    return pwd_context.verify(plain, hashed)


def create_access_token(data: dict) -> str:
    to_encode = data.copy()
    expire = datetime.now(timezone.utc) + timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)
    to_encode.update({"exp": expire})
    return jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)


def decode_access_token(token: str) -> Optional[dict]:
    try:
        return jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
    except JWTError:
        return None


class AuthService:

    @staticmethod
    async def register(db: AsyncSession, username: str, password: str,
                       display_name: str = None, email: str = None) -> tuple[User, str]:
        """注册用户，返回(用户, token)"""
        # 检查用户名是否已存在
        stmt = select(User).where(User.username == username)
        result = await db.execute(stmt)
        if result.scalar_one_or_none():
            raise ValueError("用户名已存在")

        user = User(
            username=username,
            password_hash=hash_password(password),
            display_name=display_name or username,
            email=email,
            role="user",
        )
        db.add(user)
        await db.flush()
        await db.refresh(user)

        token = create_access_token({"sub": str(user.id), "username": user.username, "role": user.role})
        return user, token

    @staticmethod
    async def login(db: AsyncSession, username: str, password: str) -> tuple[User, str]:
        """登录，返回(用户, token)"""
        stmt = select(User).where(User.username == username)
        result = await db.execute(stmt)
        user = result.scalar_one_or_none()

        if not user or not verify_password(password, user.password_hash):
            raise ValueError("用户名或密码错误")

        if user.status != 1:
            raise ValueError("账号已被禁用")

        user.last_login_at = datetime.now(timezone.utc)
        token = create_access_token({"sub": str(user.id), "username": user.username, "role": user.role})
        return user, token

    @staticmethod
    async def get_current_user(db: AsyncSession, token: str) -> Optional[User]:
        """通过token获取当前用户"""
        payload = decode_access_token(token)
        if not payload:
            return None
        user_id = payload.get("sub")
        if not user_id:
            return None
        try:
            return await db.get(User, int(user_id))
        except (ValueError, TypeError):
            return None
