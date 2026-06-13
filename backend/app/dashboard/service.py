"""仪表盘 - 服务层"""

from sqlalchemy import select, func, and_
from sqlalchemy.ext.asyncio import AsyncSession
from datetime import datetime, timedelta, timezone
from app.models import Product, Sku, Inventory, OperationLog, Brand, Supplier, Platform, ProductListing, Category


class DashboardService:

    @staticmethod
    async def get_stats(db: AsyncSession) -> dict:
        """获取统计数据"""

        # 商品统计
        total_products = await db.scalar(select(func.count(Product.id))) or 0
        on_shelf = await db.scalar(
            select(func.count(Product.id)).where(Product.status == 1)
        ) or 0
        draft = await db.scalar(
            select(func.count(Product.id)).where(Product.status == 0)
        ) or 0
        off_shelf = await db.scalar(
            select(func.count(Product.id)).where(Product.status == 2)
        ) or 0

        # SKU统计
        total_skus = await db.scalar(select(func.count(Sku.id))) or 0

        # 库存预警（库存 <= 安全库存 的SKU数）
        low_stock = await db.scalar(
            select(func.count(Inventory.id)).where(
                and_(Inventory.quantity <= Inventory.safety_stock, Inventory.safety_stock > 0)
            )
        ) or 0

        # 品牌/供应商统计
        total_brands = await db.scalar(select(func.count(Brand.id)).where(Brand.status == 1)) or 0
        total_suppliers = await db.scalar(select(func.count(Supplier.id)).where(Supplier.status == 1)) or 0

        # 平台统计
        total_platforms = await db.scalar(select(func.count(Platform.id)).where(Platform.status == 1)) or 0
        # 各平台已发布商品数
        platform_stmt = (
            select(Platform.name, Platform.code, func.count(ProductListing.id))
            .outerjoin(ProductListing, and_(
                ProductListing.platform_id == Platform.id,
                ProductListing.status == "synced",
            ))
            .where(Platform.status == 1)
            .group_by(Platform.id, Platform.name, Platform.code)
            .order_by(Platform.sort_order)
        )
        platform_result = await db.execute(platform_stmt)
        platforms_published = []
        total_published = 0
        for p_name, p_code, p_count in platform_result.all():
            platforms_published.append({"name": p_name, "code": p_code, "count": p_count})
            total_published += p_count

        # 近期发布动态（近10条）
        recent_listings = []
        listing_stmt = (
            select(ProductListing, Product.name, Platform.name, Platform.code)
            .join(Product, ProductListing.product_id == Product.id)
            .join(Platform, ProductListing.platform_id == Platform.id)
            .order_by(ProductListing.created_at.desc())
            .limit(10)
        )
        listing_result = await db.execute(listing_stmt)
        for listing, prod_name, plat_name, plat_code in listing_result.all():
            recent_listings.append({
                "product_name": prod_name,
                "platform_name": plat_name,
                "platform_code": plat_code,
                "status": listing.status,
                "created_at": listing.created_at.isoformat() if listing.created_at else None,
            })

        # 近7天操作日志数量
        seven_days_ago = datetime.now(timezone.utc) - timedelta(days=7)
        recent_logs = await db.scalar(
            select(func.count(OperationLog.id)).where(OperationLog.created_at >= seven_days_ago)
        ) or 0

        # 近10条操作日志
        stmt = select(OperationLog).order_by(OperationLog.created_at.desc()).limit(10)
        result = await db.execute(stmt)
        logs = result.scalars().all()

        return {
            "products": {
                "total": total_products,
                "on_shelf": on_shelf,
                "draft": draft,
                "off_shelf": off_shelf,
            },
            "skus": {
                "total": total_skus,
            },
            "inventory": {
                "low_stock": low_stock,
            },
            "brands": {
                "total": total_brands,
            },
            "suppliers": {
                "total": total_suppliers,
            },
            "platforms": {
                "total": total_platforms,
                "published": total_published,
                "detail": platforms_published,
            },
            "recent_listings": recent_listings,
            "recent_logs": {
                "total_7days": recent_logs,
                "items": [
                    {
                        "id": log.id,
                        "module": log.module,
                        "action": log.action,
                        "content": log.content,
                        "operator": log.operator,
                        "created_at": log.created_at.isoformat() if log.created_at else None,
                    }
                    for log in logs
                ],
            },
        }

    @staticmethod
    async def get_product_stats(db: AsyncSession) -> dict:
        """商品统计（总数/上架/草稿/下架/各分类分布）"""
        total = await db.scalar(select(func.count(Product.id))) or 0
        on_shelf = await db.scalar(
            select(func.count(Product.id)).where(Product.status == 1)
        ) or 0
        draft = await db.scalar(
            select(func.count(Product.id)).where(Product.status == 0)
        ) or 0
        off_shelf = await db.scalar(
            select(func.count(Product.id)).where(Product.status == 2)
        ) or 0

        # 各分类分布
        cat_stmt = (
            select(Product.category_id, Category.name, func.count(Product.id))
            .outerjoin(Category, Product.category_id == Category.id)
            .group_by(Product.category_id, Category.name)
            .order_by(func.count(Product.id).desc())
        )
        cat_result = await db.execute(cat_stmt)
        category_distribution = []
        for cat_id, cat_name, cnt in cat_result.all():
            category_distribution.append({
                "category_id": cat_id,
                "category_name": cat_name or f"分类{cat_id}",
                "count": cnt,
            })

        return {
            "total": total,
            "on_shelf": on_shelf,
            "draft": draft,
            "off_shelf": off_shelf,
            "category_distribution": category_distribution,
        }

    @staticmethod
    async def get_platform_stats(db: AsyncSession) -> dict:
        """平台发布统计（各平台已发布/待发布/失败数量）"""
        platforms_stmt = (
            select(Platform.id, Platform.name, Platform.code)
            .where(Platform.status == 1)
            .order_by(Platform.sort_order)
        )
        platforms_result = await db.execute(platforms_stmt)
        platforms = platforms_result.all()

        result = []
        for p_id, p_name, p_code in platforms:
            published = await db.scalar(
                select(func.count(ProductListing.id)).where(
                    and_(ProductListing.platform_id == p_id, ProductListing.status == "synced")
                )
            ) or 0
            pending = await db.scalar(
                select(func.count(ProductListing.id)).where(
                    and_(ProductListing.platform_id == p_id, ProductListing.status.in_(["draft", "pending"]))
                )
            ) or 0
            failed = await db.scalar(
                select(func.count(ProductListing.id)).where(
                    and_(ProductListing.platform_id == p_id, ProductListing.status == "failed")
                )
            ) or 0

            result.append({
                "platform_id": p_id,
                "platform_name": p_name,
                "platform_code": p_code,
                "published": published,
                "pending": pending,
                "failed": failed,
            })

        return {"items": result}
