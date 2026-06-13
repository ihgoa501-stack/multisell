"""商品管理 - 服务层"""

from typing import Optional
from sqlalchemy import select, func, or_
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Product
from app.core.schemas import ProductCreate, ProductUpdate, ProductQuery


class ProductService:

    @staticmethod
    async def create(db: AsyncSession, data: ProductCreate) -> Product:
        product = Product(**data.model_dump())
        db.add(product)
        await db.flush()
        await db.refresh(product)
        return product

    @staticmethod
    async def update(db: AsyncSession, product_id: int, data: ProductUpdate) -> Optional[Product]:
        product = await db.get(Product, product_id)
        if not product:
            return None
        update_data = data.model_dump(exclude_unset=True)
        for key, value in update_data.items():
            setattr(product, key, value)
        await db.flush()
        await db.refresh(product)
        return product

    @staticmethod
    async def get_by_id(db: AsyncSession, product_id: int) -> Optional[Product]:
        return await db.get(Product, product_id)

    @staticmethod
    async def delete(db: AsyncSession, product_id: int) -> bool:
        product = await db.get(Product, product_id)
        if not product:
            return False
        await db.delete(product)
        await db.flush()
        return True

    @staticmethod
    async def list_products(db: AsyncSession, query: ProductQuery) -> tuple[list[Product], int]:
        """分页查询商品列表"""
        stmt = select(Product)

        # 筛选条件
        if query.name:
            stmt = stmt.where(Product.name.like(f"%{query.name}%"))
        if query.category_id:
            stmt = stmt.where(Product.category_id == query.category_id)
        if query.status is not None:
            stmt = stmt.where(Product.status == query.status)
        if query.brand_id:
            stmt = stmt.where(Product.brand_id == query.brand_id)

        # 先查总数
        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        # 分页
        offset = (query.page - 1) * query.page_size
        stmt = stmt.order_by(Product.created_at.desc()).offset(offset).limit(query.page_size)

        result = await db.execute(stmt)
        products = result.scalars().all()

        return list(products), total
