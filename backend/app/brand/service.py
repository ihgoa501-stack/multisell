"""品牌管理 - 服务层"""
from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Brand


class BrandService:

    @staticmethod
    async def create(db: AsyncSession, data: dict) -> Brand:
        brand = Brand(**data)
        db.add(brand)
        await db.flush()
        await db.refresh(brand)
        return brand

    @staticmethod
    async def update(db: AsyncSession, brand_id: int, data: dict) -> Optional[Brand]:
        brand = await db.get(Brand, brand_id)
        if not brand:
            return None
        for key, value in data.items():
            if value is not None:
                setattr(brand, key, value)
        await db.flush()
        await db.refresh(brand)
        return brand

    @staticmethod
    async def get_by_id(db: AsyncSession, brand_id: int) -> Optional[Brand]:
        return await db.get(Brand, brand_id)

    @staticmethod
    async def list_brands(db: AsyncSession, name: str = None,
                           page: int = 1, page_size: int = 20) -> tuple[list[Brand], int]:
        stmt = select(Brand)
        if name:
            stmt = stmt.where(Brand.name.like(f"%{name}%"))

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(Brand.sort_order, Brand.id).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        return list(result.scalars().all()), total

    @staticmethod
    async def get_all(db: AsyncSession) -> list[Brand]:
        """获取所有启用品牌（供下拉选择）"""
        stmt = select(Brand).where(Brand.status == 1).order_by(Brand.sort_order, Brand.id)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def delete(db: AsyncSession, brand_id: int) -> bool:
        brand = await db.get(Brand, brand_id)
        if not brand:
            return False
        # 检查是否有商品使用该品牌
        from app.models import Product
        stmt = select(func.count()).select_from(Product).where(Product.brand_id == brand_id)
        used_count = await db.scalar(stmt) or 0
        if used_count > 0:
            raise ValueError(f"有 {used_count} 个商品使用了该品牌，无法删除")
        await db.delete(brand)
        await db.flush()
        return True
