"""通知与预警 - 服务层

自动检查各项业务条件，生成通知消息。
支持: 库存预警/结算待对账/发布失败/订单超时。
"""

import logging
from datetime import datetime, timezone, timedelta
from typing import Optional

from sqlalchemy import select, func, and_, desc
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    Notification,
    AlertRule,
    Inventory,
    Settlement,
    SettlementItem,
    ProductListing,
    Order,
)

logger = logging.getLogger(__name__)

# 默认预警规则定义
DEFAULT_ALERT_RULES = [
    {
        "name": "库存低于安全库存",
        "alert_type": "inventory_low_stock",
        "config": {},
        "description": "库存数量 ≤ 安全库存且安全库存 > 0",
    },
    {
        "name": "库存为零",
        "alert_type": "inventory_out_of_stock",
        "config": {},
        "description": "库存数量 ≤ 0",
    },
    {
        "name": "结算待对账",
        "alert_type": "settlement_pending",
        "config": {},
        "description": "有待对账的结算单超过24小时",
    },
    {
        "name": "结算对账差异",
        "alert_type": "settlement_discrepancy",
        "config": {},
        "description": "结算明细对账状态为差异",
    },
    {
        "name": "发布失败",
        "alert_type": "listing_failed",
        "config": {},
        "description": "商品发布到平台失败",
    },
    {
        "name": "待付款订单超时",
        "alert_type": "order_pending",
        "config": {"timeout_hours": 24},
        "description": "待付款订单超过24小时",
    },
    {
        "name": "系统通知",
        "alert_type": "system",
        "config": {},
        "description": "系统级通知",
    },
]


class NotificationService:
    # ── 检查并创建预警 ────────────────────────────────────────

    @staticmethod
    async def check_and_create_alerts(db: AsyncSession) -> dict:
        """扫描所有启用的预警规则，生成通知"""
        results = {}
        rules = await db.execute(select(AlertRule).where(AlertRule.enabled == 1))
        for rule in rules.scalars().all():
            count = 0
            try:
                if rule.alert_type == "inventory_low_stock":
                    count = await NotificationService._check_low_stock(db)
                elif rule.alert_type == "inventory_out_of_stock":
                    count = await NotificationService._check_out_of_stock(db)
                elif rule.alert_type == "settlement_pending":
                    count = await NotificationService._check_settlement_pending(db)
                elif rule.alert_type == "settlement_discrepancy":
                    count = await NotificationService._check_settlement_discrepancy(db)
                elif rule.alert_type == "listing_failed":
                    count = await NotificationService._check_listing_failed(db)
                elif rule.alert_type == "order_pending":
                    timeout = (rule.config or {}).get("timeout_hours", 24)
                    count = await NotificationService._check_pending_orders(db, timeout)
                results[rule.alert_type] = count
            except Exception as e:
                logger.error("预警检查失败 %s: %s", rule.alert_type, e)
                results[rule.alert_type] = -1
        return results

    @staticmethod
    async def _check_low_stock(db: AsyncSession) -> int:
        """库存预警: quantity <= safety_stock 且 safety_stock > 0"""
        stmt = select(Inventory).where(
            and_(
                Inventory.quantity <= Inventory.safety_stock, Inventory.safety_stock > 0
            )
        )
        result = await db.execute(stmt)
        count = 0
        for inv in result.scalars().all():
            exists = await db.execute(
                select(Notification.id).where(
                    Notification.alert_type == "inventory_low_stock",
                    Notification.source_id == f"sku={inv.sku_id}",
                    Notification.is_read == 0,
                )
            )
            if not exists.scalar_one_or_none():
                notif = Notification(
                    user_id=1,
                    alert_type="inventory_low_stock",
                    title=f"SKU {inv.sku_id} 库存预警",
                    content=f"当前库存 {inv.quantity}，安全库存 {inv.safety_stock}",
                    link_url=f"/inventory/{inv.sku_id}",
                    severity="warning",
                    source_id=f"sku={inv.sku_id}",
                )
                db.add(notif)
                count += 1
        if count:
            await db.flush()
        return count

    @staticmethod
    async def _check_out_of_stock(db: AsyncSession) -> int:
        """缺货预警: quantity <= 0"""
        stmt = select(Inventory).where(Inventory.quantity <= 0)
        result = await db.execute(stmt)
        count = 0
        for inv in result.scalars().all():
            exists = await db.execute(
                select(Notification.id).where(
                    Notification.alert_type == "inventory_out_of_stock",
                    Notification.source_id == f"sku={inv.sku_id}",
                    Notification.is_read == 0,
                )
            )
            if not exists.scalar_one_or_none():
                notif = Notification(
                    user_id=1,
                    alert_type="inventory_out_of_stock",
                    title=f"SKU {inv.sku_id} 已缺货",
                    content="库存数量为 0，请及时补货",
                    link_url=f"/inventory/{inv.sku_id}",
                    severity="error",
                    source_id=f"sku={inv.sku_id}",
                )
                db.add(notif)
                count += 1
        if count:
            await db.flush()
        return count

    @staticmethod
    async def _check_settlement_pending(db: AsyncSession) -> int:
        """结算待对账预警"""
        stmt = select(Settlement).where(Settlement.status == "pending")
        result = await db.execute(stmt)
        count = 0
        for s in result.scalars().all():
            exists = await db.execute(
                select(Notification.id).where(
                    Notification.alert_type == "settlement_pending",
                    Notification.source_id == f"settlement={s.id}",
                    Notification.is_read == 0,
                )
            )
            if not exists.scalar_one_or_none():
                notif = Notification(
                    user_id=1,
                    alert_type="settlement_pending",
                    title=f"结算单 {s.settlement_no} 待对账",
                    content=f"收入 ¥{float(s.total_revenue)}，费用 ¥{float(s.total_fee)}",
                    link_url=f"/settlements/{s.id}",
                    severity="info",
                    source_id=f"settlement={s.id}",
                )
                db.add(notif)
                count += 1
        if count:
            await db.flush()
        return count

    @staticmethod
    async def _check_settlement_discrepancy(db: AsyncSession) -> int:
        """结算对账差异预警"""
        stmt = select(SettlementItem).where(
            SettlementItem.reconciliation_status == "discrepancy"
        )
        result = await db.execute(stmt)
        count = 0
        for item in result.scalars().all():
            exists = await db.execute(
                select(Notification.id).where(
                    Notification.alert_type == "settlement_discrepancy",
                    Notification.source_id == f"item={item.id}",
                    Notification.is_read == 0,
                )
            )
            if not exists.scalar_one_or_none():
                notif = Notification(
                    user_id=1,
                    alert_type="settlement_discrepancy",
                    title="结算明细金额差异",
                    content=item.reconciliation_note
                    or f"差异金额: ¥{float(item.amount)}",
                    link_url=f"/settlements/{item.settlement_id}",
                    severity="error",
                    source_id=f"item={item.id}",
                )
                db.add(notif)
                count += 1
        if count:
            await db.flush()
        return count

    @staticmethod
    async def _check_listing_failed(db: AsyncSession) -> int:
        """发布失败预警"""
        stmt = select(ProductListing).where(ProductListing.status == "failed")
        result = await db.execute(stmt)
        count = 0
        for listing in result.scalars().all():
            exists = await db.execute(
                select(Notification.id).where(
                    Notification.alert_type == "listing_failed",
                    Notification.source_id == f"listing={listing.id}",
                    Notification.is_read == 0,
                )
            )
            if not exists.scalar_one_or_none():
                notif = Notification(
                    user_id=1,
                    alert_type="listing_failed",
                    title="商品发布失败",
                    content=listing.sync_message
                    or f"platform_id={listing.platform_id}",
                    link_url=f"/products/{listing.product_id}",
                    severity="error",
                    source_id=f"listing={listing.id}",
                )
                db.add(notif)
                count += 1
        if count:
            await db.flush()
        return count

    @staticmethod
    async def _check_pending_orders(db: AsyncSession, timeout_hours: int = 24) -> int:
        """待付款订单超时预警"""
        cutoff = datetime.now(timezone.utc) - timedelta(hours=timeout_hours)
        stmt = select(Order).where(
            and_(Order.status == "pending", Order.created_at <= cutoff)
        )
        result = await db.execute(stmt)
        count = 0
        for order in result.scalars().all():
            exists = await db.execute(
                select(Notification.id).where(
                    Notification.alert_type == "order_pending",
                    Notification.source_id == f"order={order.id}",
                    Notification.is_read == 0,
                )
            )
            if not exists.scalar_one_or_none():
                notif = Notification(
                    user_id=1,
                    alert_type="order_pending",
                    title=f"订单 {order.order_no} 超时未支付",
                    content=f"金额 ¥{float(order.pay_amount or 0)}，已超过 {timeout_hours} 小时",
                    link_url=f"/orders/{order.id}",
                    severity="warning",
                    source_id=f"order={order.id}",
                )
                db.add(notif)
                count += 1
        if count:
            await db.flush()
        return count

    # ── 通知 CRUD ──────────────────────────────────────────────

    @staticmethod
    async def list_notifications(
        db: AsyncSession,
        user_id: int = 1,
        unread_only: bool = False,
        alert_type: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict], int]:
        """分页查询通知"""
        stmt = select(Notification).where(Notification.user_id == user_id)
        count_stmt = (
            select(func.count())
            .select_from(Notification)
            .where(Notification.user_id == user_id)
        )

        if unread_only:
            stmt = stmt.where(Notification.is_read == 0)
            count_stmt = count_stmt.where(Notification.is_read == 0)
        if alert_type:
            stmt = stmt.where(Notification.alert_type == alert_type)
            count_stmt = count_stmt.where(Notification.alert_type == alert_type)

        total = (await db.execute(count_stmt)).scalar() or 0
        offset = (page - 1) * page_size
        stmt = (
            stmt.order_by(desc(Notification.created_at)).offset(offset).limit(page_size)
        )
        result = await db.execute(stmt)
        notifications = result.scalars().all()

        rows = []
        for n in notifications:
            rows.append(
                {
                    "id": n.id,
                    "user_id": n.user_id,
                    "alert_type": n.alert_type,
                    "title": n.title,
                    "content": n.content,
                    "link_url": n.link_url,
                    "severity": n.severity,
                    "is_read": bool(n.is_read),
                    "source_id": n.source_id,
                    "created_at": n.created_at.isoformat() if n.created_at else None,
                }
            )

        return rows, total

    @staticmethod
    async def get_unread_count(db: AsyncSession, user_id: int = 1) -> dict:
        """获取未读通知数（分类型）"""
        stmt = (
            select(
                Notification.alert_type,
                func.count(Notification.id),
            )
            .where(
                Notification.user_id == user_id,
                Notification.is_read == 0,
            )
            .group_by(Notification.alert_type)
        )

        result = await db.execute(stmt)
        by_type = {}
        total = 0
        for row in result.all():
            by_type[row[0]] = row[1]
            total += row[1]

        return {"total": total, "by_type": by_type}

    @staticmethod
    async def mark_read(
        db: AsyncSession, notification_id: int, user_id: int = 1
    ) -> bool:
        """标记单条已读"""
        notif = await db.get(Notification, notification_id)
        if not notif or notif.user_id != user_id:
            return False
        notif.is_read = 1
        await db.flush()
        return True

    @staticmethod
    async def mark_all_read(db: AsyncSession, user_id: int = 1) -> int:
        """标记全部已读，返回影响条数"""
        stmt = select(Notification).where(
            Notification.user_id == user_id, Notification.is_read == 0
        )
        result = await db.execute(stmt)
        count = 0
        for notif in result.scalars().all():
            notif.is_read = 1
            count += 1
        if count:
            await db.flush()
        return count

    @staticmethod
    async def delete_notification(
        db: AsyncSession, notification_id: int, user_id: int = 1
    ) -> bool:
        """删除通知"""
        notif = await db.get(Notification, notification_id)
        if not notif or notif.user_id != user_id:
            return False
        await db.delete(notif)
        await db.flush()
        return True

    # ── 预警规则 ──────────────────────────────────────────────

    @staticmethod
    async def initialize_rules(db: AsyncSession) -> list[AlertRule]:
        """创建默认预警规则（如不存在）"""
        created = []
        for rule_data in DEFAULT_ALERT_RULES:
            existing = await db.execute(
                select(AlertRule).where(AlertRule.alert_type == rule_data["alert_type"])
            )
            if not existing.scalar_one_or_none():
                rule = AlertRule(**rule_data)
                db.add(rule)
                await db.flush()
                created.append(rule)
        return created

    @staticmethod
    async def list_rules(db: AsyncSession) -> list[AlertRule]:
        result = await db.execute(select(AlertRule).order_by(AlertRule.id))
        return list(result.scalars().all())

    @staticmethod
    async def update_rule(
        db: AsyncSession, rule_id: int, data: dict
    ) -> Optional[AlertRule]:
        rule = await db.get(AlertRule, rule_id)
        if not rule:
            return None
        for k, v in data.items():
            if v is not None:
                setattr(rule, k, v)
        await db.flush()
        await db.refresh(rule)
        return rule
