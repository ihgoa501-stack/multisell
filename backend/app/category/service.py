"""分类管理 - 服务层"""
from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Category


class CategoryService:

    @staticmethod
    async def create(db: AsyncSession, name: str, parent_id: int = 0, sort_order: int = 0) -> Category:
        level = 0
        if parent_id > 0:
            parent = await db.get(Category, parent_id)
            level = (parent.level + 1) if parent else 0

        category = Category(name=name, parent_id=parent_id, level=level, sort_order=sort_order)
        db.add(category)
        await db.flush()
        await db.refresh(category)
        return category

    @staticmethod
    async def update(db: AsyncSession, category_id: int, data: dict) -> Optional[Category]:
        category = await db.get(Category, category_id)
        if not category:
            return None
        for key, value in data.items():
            if value is not None:
                setattr(category, key, value)
        await db.flush()
        await db.refresh(category)
        return category

    @staticmethod
    async def get_tree(db: AsyncSession) -> list[Category]:
        """获取全部分类树"""
        stmt = select(Category).order_by(Category.sort_order, Category.id)
        result = await db.execute(stmt)
        categories = result.scalars().all()
        return list(categories)

    @staticmethod
    async def delete(db: AsyncSession, category_id: int) -> tuple[bool, str]:
        """删除分类，检查是否有子分类或商品使用"""
        category = await db.get(Category, category_id)
        if not category:
            return False, "分类不存在"

        # 检查是否有子分类
        stmt = select(func.count()).select_from(Category).where(Category.parent_id == category_id)
        child_count = await db.scalar(stmt) or 0
        if child_count > 0:
            return False, "该分类下有子分类，无法删除"

        # 检查是否有商品使用（导入 Product 避免循环引用）
        from app.models import Product
        stmt = select(func.count()).select_from(Product).where(Product.category_id == category_id)
        product_count = await db.scalar(stmt) or 0
        if product_count > 0:
            return False, f"有 {product_count} 个商品使用了该分类，无法删除"

        await db.delete(category)
        await db.flush()
        return True, "删除成功"
