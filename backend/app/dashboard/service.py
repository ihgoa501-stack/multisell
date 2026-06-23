"""仪表盘 - 服务层（增强版）

提供全面的运营分析数据：
- 核心KPI（商品/订单/收入/利润/库存）
- 订单趋势（日/周/月）
- 平台对比分析
- 热销商品排行
- 财务概览
- 结算对账统计
"""

from sqlalchemy import select, func, and_
from sqlalchemy.ext.asyncio import AsyncSession
from datetime import datetime, timedelta, timezone
from app.models import (
    Product,
    Sku,
    Inventory,
    OperationLog,
    Brand,
    Supplier,
    Platform,
    ProductListing,
    Category,
    Order,
    Settlement,
    FinanceAccount,
)


class DashboardService:
    @staticmethod
    async def get_stats(db: AsyncSession) -> dict:
        """获取统计数据（兼容原接口）"""
        return await DashboardService.get_dashboard(db)

    @staticmethod
    async def get_dashboard(db: AsyncSession) -> dict:
        """运营驾驶舱 — 完整看板数据"""

        # ═══════════════════════════════════════════════════════
        # 1. 核心计数
        # ═══════════════════════════════════════════════════════
        total_products = await db.scalar(select(func.count(Product.id))) or 0
        on_shelf = (
            await db.scalar(select(func.count(Product.id)).where(Product.status == 1))
            or 0
        )
        draft = (
            await db.scalar(select(func.count(Product.id)).where(Product.status == 0))
            or 0
        )
        total_skus = await db.scalar(select(func.count(Sku.id))) or 0
        total_brands = (
            await db.scalar(select(func.count(Brand.id)).where(Brand.status == 1)) or 0
        )
        total_suppliers = (
            await db.scalar(select(func.count(Supplier.id)).where(Supplier.status == 1))
            or 0
        )

        # ═══════════════════════════════════════════════════════
        # 2. 库存健康
        # ═══════════════════════════════════════════════════════
        low_stock = (
            await db.scalar(
                select(func.count(Inventory.id)).where(
                    and_(
                        Inventory.quantity <= Inventory.safety_stock,
                        Inventory.safety_stock > 0,
                    )
                )
            )
            or 0
        )
        out_of_stock = (
            await db.scalar(
                select(func.count(Inventory.id)).where(Inventory.quantity <= 0)
            )
            or 0
        )
        total_inventories = await db.scalar(select(func.count(Inventory.id))) or 0

        # ═══════════════════════════════════════════════════════
        # 3. 订单统计
        # ═══════════════════════════════════════════════════════
        total_orders = await db.scalar(select(func.count(Order.id))) or 0
        paid_orders = (
            await db.scalar(
                select(func.count(Order.id)).where(
                    Order.status.in_(["paid", "shipped", "delivered", "completed"])
                )
            )
            or 0
        )

        # 订单状态分布
        order_statuses = [
            "pending",
            "paid",
            "shipped",
            "delivered",
            "completed",
            "cancelled",
        ]
        order_status_dist = {}
        for s in order_statuses:
            cnt = (
                await db.scalar(select(func.count(Order.id)).where(Order.status == s))
                or 0
            )
            if cnt > 0:
                order_status_dist[s] = cnt

        # 近30天订单趋势（使用 raw SQL 避免 ORM 列自动注入）
        thirty_days_ago = datetime.now(timezone.utc) - timedelta(days=30)
        from sqlalchemy import text as sa_text

        trend_sql = sa_text("""
            SELECT date_trunc('day', created_at) AS day,
                   count(id) AS cnt,
                   coalesce(sum(pay_amount), 0) AS rev
            FROM sales_order
            WHERE created_at >= :since
            GROUP BY date_trunc('day', created_at)
            ORDER BY date_trunc('day', created_at)
        """)
        daily_order_counts = await db.execute(trend_sql, {"since": thirty_days_ago})
        order_trend = []
        for day_row in daily_order_counts.all():
            order_trend.append(
                {
                    "date": day_row[0].strftime("%Y-%m-%d") if day_row[0] else "N/A",
                    "orders": day_row[1],
                    "revenue": float(day_row[2]),
                }
            )

        # ═══════════════════════════════════════════════════════
        # 4. 收入/利润概览
        # ═══════════════════════════════════════════════════════
        total_revenue = (
            await db.scalar(
                select(func.coalesce(func.sum(Order.pay_amount), 0)).where(
                    Order.status.in_(["paid", "shipped", "delivered", "completed"])
                )
            )
            or 0
        )
        total_profit = (
            await db.scalar(
                select(func.coalesce(func.sum(Order.profit_amount), 0)).where(
                    Order.status.in_(["paid", "shipped", "delivered", "completed"])
                )
            )
            or 0
        )
        total_product_cost = (
            await db.scalar(
                select(func.coalesce(func.sum(Order.product_cost), 0)).where(
                    Order.status.in_(["paid", "shipped", "delivered", "completed"])
                )
            )
            or 0
        )

        overall_margin = (
            round(float(total_profit) / float(total_revenue) * 100, 2)
            if float(total_revenue) > 0
            else 0
        )

        # ═══════════════════════════════════════════════════════
        # 5. 热销商品（使用 raw SQL 避免 ORM 列注入）
        # ═══════════════════════════════════════════════════════
        top_products = []
        try:
            from sqlalchemy import text as _t

            top_sql = _t("""
                SELECT oi.product_id, p.name AS product_name,
                       count(oi.id) AS sold_count,
                       coalesce(sum(oi.subtotal), 0) AS revenue
                FROM sales_order_item oi
                JOIN product p ON oi.product_id = p.id
                GROUP BY oi.product_id, p.name
                ORDER BY revenue DESC
                LIMIT 10
            """)
            top_result = await db.execute(top_sql)
            for row in top_result.all():
                top_products.append(
                    {
                        "product_id": row[0],
                        "product_name": row[1],
                        "sold_count": row[2],
                        "revenue": float(row[3]),
                    }
                )
        except Exception:
            top_products = []

        # ═══════════════════════════════════════════════════════
        # 6. 平台发布统计
        # ═══════════════════════════════════════════════════════
        total_platforms = (
            await db.scalar(select(func.count(Platform.id)).where(Platform.status == 1))
            or 0
        )

        platform_stmt = (
            select(Platform.name, Platform.code, func.count(ProductListing.id))
            .outerjoin(
                ProductListing,
                and_(
                    ProductListing.platform_id == Platform.id,
                    ProductListing.status == "synced",
                ),
            )
            .where(Platform.status == 1)
            .group_by(Platform.id, Platform.name, Platform.code)
            .order_by(Platform.sort_order)
        )
        platform_result = await db.execute(platform_stmt)
        platforms_published = []
        total_published = 0
        for p_name, p_code, p_count in platform_result.all():
            platforms_published.append(
                {"name": p_name, "code": p_code, "count": p_count}
            )
            total_published += p_count

        # ═══════════════════════════════════════════════════════
        # 7. 结算概览
        # ═══════════════════════════════════════════════════════
        total_settlements = await db.scalar(select(func.count(Settlement.id))) or 0
        reconciled_settlements = (
            await db.scalar(
                select(func.count(Settlement.id)).where(
                    Settlement.status == "reconciled"
                )
            )
            or 0
        )
        pending_settlements = (
            await db.scalar(
                select(func.count(Settlement.id)).where(Settlement.status == "pending")
            )
            or 0
        )

        total_settlement_net = (
            await db.scalar(
                select(func.coalesce(func.sum(Settlement.total_net), 0)).where(
                    Settlement.status.in_(["reconciled", "reconciling"])
                )
            )
            or 0
        )

        # ═══════════════════════════════════════════════════════
        # 8. 财务账户概览
        # ═══════════════════════════════════════════════════════
        total_accounts = await db.scalar(select(func.count(FinanceAccount.id))) or 0
        total_balance = (
            await db.scalar(select(func.coalesce(func.sum(FinanceAccount.balance), 0)))
            or 0
        )

        # ═══════════════════════════════════════════════════════
        # 9. 近期动态
        # ═══════════════════════════════════════════════════════
        seven_days_ago = datetime.now(timezone.utc) - timedelta(days=7)
        recent_logs = (
            await db.scalar(
                select(func.count(OperationLog.id)).where(
                    OperationLog.created_at >= seven_days_ago
                )
            )
            or 0
        )

        stmt = select(OperationLog).order_by(OperationLog.created_at.desc()).limit(10)
        result = await db.execute(stmt)
        logs = result.scalars().all()

        # ═══════════════════════════════════════════════════════
        # 返回汇总
        # ═══════════════════════════════════════════════════════
        return {
            # 商品
            "products": {
                "total": total_products,
                "on_shelf": on_shelf,
                "draft": draft,
                "off_shelf": total_products - on_shelf - draft,
                "skus": total_skus,
            },
            # 库存健康
            "inventory": {
                "total": total_inventories,
                "low_stock": low_stock,
                "out_of_stock": out_of_stock,
                "healthy": max(0, total_inventories - low_stock - out_of_stock),
                "health_pct": round(
                    (total_inventories - low_stock - out_of_stock)
                    / total_inventories
                    * 100,
                    1,
                )
                if total_inventories > 0
                else 100,
            },
            # 品牌/供应商
            "brands": {"total": total_brands},
            "suppliers": {"total": total_suppliers},
            # 平台
            "platforms": {
                "total": total_platforms,
                "published": total_published,
                "detail": platforms_published,
            },
            # 订单
            "orders": {
                "total": total_orders,
                "paid": paid_orders,
                "status_distribution": order_status_dist,
                "trend_30d": order_trend,
            },
            # 财务
            "finance": {
                "total_revenue": float(total_revenue),
                "total_product_cost": float(total_product_cost),
                "total_profit": float(total_profit),
                "profit_margin": overall_margin,
                "total_balance": float(total_balance),
            },
            # 结算
            "settlements": {
                "total": total_settlements,
                "reconciled": reconciled_settlements,
                "pending": pending_settlements,
                "net_revenue": float(total_settlement_net),
            },
            # 财务账户
            "accounts": {"total": total_accounts},
            # 排行
            "top_products": top_products,
            # 动态
            "recent_logs": {
                "total_7days": recent_logs,
                "items": [
                    {
                        "id": log.id,
                        "module": log.module,
                        "action": log.action,
                        "content": log.content,
                        "operator": log.operator,
                        "created_at": log.created_at.isoformat()
                        if log.created_at
                        else None,
                    }
                    for log in logs
                ],
            },
        }

    @staticmethod
    async def get_product_stats(db: AsyncSession) -> dict:
        """商品统计（兼容原接口）"""
        total = await db.scalar(select(func.count(Product.id))) or 0
        on_shelf = (
            await db.scalar(select(func.count(Product.id)).where(Product.status == 1))
            or 0
        )
        draft = (
            await db.scalar(select(func.count(Product.id)).where(Product.status == 0))
            or 0
        )
        off_shelf = (
            await db.scalar(select(func.count(Product.id)).where(Product.status == 2))
            or 0
        )

        cat_stmt = (
            select(Product.category_id, Category.name, func.count(Product.id))
            .outerjoin(Category, Product.category_id == Category.id)
            .group_by(Product.category_id, Category.name)
            .order_by(func.count(Product.id).desc())
        )
        cat_result = await db.execute(cat_stmt)
        category_distribution = []
        for cat_id, cat_name, cnt in cat_result.all():
            category_distribution.append(
                {
                    "category_id": cat_id,
                    "category_name": cat_name or f"分类{cat_id}",
                    "count": cnt,
                }
            )

        return {
            "total": total,
            "on_shelf": on_shelf,
            "draft": draft,
            "off_shelf": off_shelf,
            "category_distribution": category_distribution,
        }

    @staticmethod
    async def get_platform_stats(db: AsyncSession) -> dict:
        """平台发布统计（兼容原接口）"""
        platforms_stmt = (
            select(Platform.id, Platform.name, Platform.code)
            .where(Platform.status == 1)
            .order_by(Platform.sort_order)
        )
        platforms_result = await db.execute(platforms_stmt)
        platforms = platforms_result.all()

        result = []
        for p_id, p_name, p_code in platforms:
            published = (
                await db.scalar(
                    select(func.count(ProductListing.id)).where(
                        and_(
                            ProductListing.platform_id == p_id,
                            ProductListing.status == "synced",
                        )
                    )
                )
                or 0
            )
            pending = (
                await db.scalar(
                    select(func.count(ProductListing.id)).where(
                        and_(
                            ProductListing.platform_id == p_id,
                            ProductListing.status.in_(["draft", "pending"]),
                        )
                    )
                )
                or 0
            )
            failed = (
                await db.scalar(
                    select(func.count(ProductListing.id)).where(
                        and_(
                            ProductListing.platform_id == p_id,
                            ProductListing.status == "failed",
                        )
                    )
                )
                or 0
            )

            result.append(
                {
                    "platform_id": p_id,
                    "platform_name": p_name,
                    "platform_code": p_code,
                    "published": published,
                    "pending": pending,
                    "failed": failed,
                }
            )

        return {"items": result}
