"""Agent 数据补齐服务

从数据库自动读取业务数据，填充 Agent 决策上下文。
Agent 只需传入最少标识字段（如 sku_code），剩余字段由本服务补齐。
"""

import logging
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Sku, Product, Inventory, Category

logger = logging.getLogger(__name__)


class AgentDataService:
    """从 DB 读取业务数据，补齐 Agent 决策上下文"""

    @staticmethod
    async def fill_sku_context(db: AsyncSession, context: dict) -> dict:
        """根据 sku_code 补齐库存/产品相关字段"""
        ctx = dict(context)
        sku_code = ctx.get("sku_code", "")
        if not sku_code:
            return ctx

        # 查找 SKU
        stmt = select(Sku).where(Sku.code == sku_code)
        result = await db.execute(stmt)
        sku = result.scalar_one_or_none()
        if not sku:
            # 也尝试按 spec_desc 匹配
            stmt2 = select(Sku).where(Sku.spec_desc == sku_code)
            result2 = await db.execute(stmt2)
            sku = result2.scalar_one_or_none()
        if not sku:
            return ctx

        # SKU 级别的数据
        ctx.setdefault("sku_code", sku.code or sku.spec_desc or "")
        ctx.setdefault("selling_price", float(sku.price or 0))
        ctx.setdefault("cost_price", float(sku.cost_price or 0))

        # 库存数据
        inv_stmt = select(Inventory).where(Inventory.sku_id == sku.id)
        inv_result = await db.execute(inv_stmt)
        inv = inv_result.scalar_one_or_none()
        if inv:
            ctx.setdefault("sellable_stock", inv.quantity or 0)
            ctx.setdefault("locked_stock", inv.locked_quantity or 0)
            ctx.setdefault("safety_stock", inv.safety_stock or 0)

        # 商品级数据
        prod_stmt = select(Product).where(Product.id == sku.product_id)
        prod_result = await db.execute(prod_stmt)
        product = prod_result.scalar_one_or_none()
        if product:
            ctx.setdefault("product_name", product.name or "")
            ctx.setdefault("weight_kg", float(product.package_weight_kg or 0))
            ctx.setdefault("package_length_cm", float(product.package_length_cm or 0))
            ctx.setdefault("package_width_cm", float(product.package_width_cm or 0))
            ctx.setdefault("package_height_cm", float(product.package_height_cm or 0))
            ctx.setdefault("cargo_type", product.cargo_type or "normal")
            ctx.setdefault(
                "category",
                await AgentDataService._get_category_name(db, product.category_id),
            )

        return ctx

    @staticmethod
    async def fill_product_context(db: AsyncSession, context: dict) -> dict:
        """根据 product_id 补齐产品上下文"""
        ctx = dict(context)
        product_id = ctx.get("product_id")
        if not product_id:
            return ctx

        stmt = select(Product).where(Product.id == int(product_id))
        result = await db.execute(stmt)
        product = result.scalar_one_or_none()
        if not product:
            return ctx

        ctx.setdefault("product_name", product.name or "")
        ctx.setdefault("category_id", product.category_id)
        ctx.setdefault(
            "category",
            await AgentDataService._get_category_name(db, product.category_id),
        )
        ctx.setdefault("cargo_type", product.cargo_type or "normal")
        ctx.setdefault("weight_kg", float(product.package_weight_kg or 0))

        # 查找所有 SKU 获取价格范围
        skus_stmt = select(Sku).where(Sku.product_id == product.id)
        skus_result = await db.execute(skus_stmt)
        skus = skus_result.scalars().all()
        if skus:
            prices = [float(s.price or 0) for s in skus if s.price]
            ctx.setdefault("selling_price", max(prices) if prices else 0)
            costs = [float(s.cost_price or 0) for s in skus if s.cost_price]
            ctx.setdefault("cost_price", min(costs) if costs else 0)

        return ctx

    @staticmethod
    async def _get_category_name(db: AsyncSession, category_id) -> str:
        if not category_id:
            return ""
        stmt = select(Category).where(Category.id == int(category_id))
        result = await db.execute(stmt)
        cat = result.scalar_one_or_none()
        return cat.name if cat else ""
