"""平台管理 - 服务层"""
from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Platform
from app.common.crypto import encrypt_api_key


class PlatformService:

    @staticmethod
    async def create(db: AsyncSession, data: dict) -> Platform:
        # 加密API密钥
        if data.get("api_key"):
            data["api_key"] = encrypt_api_key(data["api_key"])
        platform = Platform(**data)
        db.add(platform)
        await db.flush()
        await db.refresh(platform)
        return platform

    @staticmethod
    async def update(db: AsyncSession, platform_id: int, data: dict) -> Optional[Platform]:
        platform = await db.get(Platform, platform_id)
        if not platform:
            return None
        # 加密API密钥
        if data.get("api_key"):
            data["api_key"] = encrypt_api_key(data["api_key"])
        for key, value in data.items():
            if value is not None:
                setattr(platform, key, value)
        await db.flush()
        await db.refresh(platform)
        return platform

    @staticmethod
    async def get_by_id(db: AsyncSession, platform_id: int) -> Optional[Platform]:
        return await db.get(Platform, platform_id)

    @staticmethod
    async def list_all(db: AsyncSession) -> list[Platform]:
        stmt = select(Platform).order_by(Platform.sort_order, Platform.id)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def delete(db: AsyncSession, platform_id: int) -> bool:
        platform = await db.get(Platform, platform_id)
        if not platform:
            return False
        await db.delete(platform)
        await db.flush()
        return True
