"""pytest fixtures — ASGI transport + 测试数据库隔离"""

import os
import pytest_asyncio
from httpx import AsyncClient, ASGITransport

# 测试环境使用独立数据库
os.environ.setdefault("AUTH_ENABLED", "False")
os.environ.setdefault(
    "DATABASE_URL",
    os.environ.get(
        "TEST_DATABASE_URL",
        "postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test",
    ),
)

from app.main import app
from app.database import Base, async_session_factory, engine
from app.auth.service import AuthService


@pytest_asyncio.fixture(scope="session")
async def prepare_db():
    """会话级：在测试数据库中创建所有表 + 种子数据"""
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
        await conn.run_sync(Base.metadata.create_all)

    # 创建默认 admin 用户（幂等）
    async with async_session_factory() as session:
        from sqlalchemy import select
        from app.models import User

        stmt = select(User).where(User.username == "admin")
        result = await session.execute(stmt)
        if not result.scalar_one_or_none():
            await AuthService.register(
                session, "admin", "admin123", "系统管理员", "admin@example.com"
            )
        await session.commit()

    yield
    # 测试结束后清除测试 schema
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    await engine.dispose()


@pytest_asyncio.fixture
async def async_client(prepare_db):
    """使用 ASGI Transport 直连 FastAPI 应用（每次测试独立事务）"""
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        yield client
