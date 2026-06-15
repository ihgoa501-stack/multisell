"""异常工作台 - 服务层"""

from datetime import datetime, timezone
from typing import Optional

from sqlalchemy import or_, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    ExceptionItem,
    ListingTask,
    Order,
    PlatformSettlementItem,
    ShippingBillItem,
)
from app.operation_log.service import OperationLogService

SEVERITY_MAP = {
    "listing": {"blocked": "medium", "failed": "high"},
    "shipping": {"unmatched_bill": "medium", "amount_mismatch": "high", "currency_mismatch": "high", "missing_snapshot": "low"},
    "settlement": {"unmatched": "medium"},
    "finance": {"negative_profit": "high"},
}


class ExceptionService:

    # ── Generate ────────────────────────────────────────────────────────

    @staticmethod
    async def generate(
        db: AsyncSession,
        operator: Optional[str] = None,
    ) -> dict:
        """扫描各模块，生成新的异常条目（幂等）。"""
        created_count = 0
        total_scanned = 0

        # 1. Listing tasks: blocked / failed
        stmt = select(ListingTask).where(
            ListingTask.status.in_(["blocked", "failed"]),
        )
        result = await db.execute(stmt)
        for task in result.scalars().all():
            total_scanned += 1
            if await ExceptionService._find_existing(db, "listing", "listing_task", task.id):
                continue
            severity = SEVERITY_MAP["listing"].get(task.status, "medium")
            missing = ", ".join(task.missing_requirements or []) if task.status == "blocked" else task.last_error or ""
            ExceptionService._create(
                db, "listing", "listing_task", task.id, severity,
                title=f"上架任务状态: {task.status}",
                description=f"商品ID={task.product_id}, 平台ID={task.platform_id}. {missing}",
                recommended_action="检查商品数据完整性后重新检查任务",
            )
            created_count += 1

        # 2. Shipping bill items: unmatched / amount_mismatch
        stmt = select(ShippingBillItem).where(
            ShippingBillItem.reconciliation_status.in_(
                ["unmatched_bill", "amount_mismatch", "currency_mismatch", "missing_snapshot"]
            ),
        )
        result = await db.execute(stmt)
        for item in result.scalars().all():
            total_scanned += 1
            if await ExceptionService._find_existing(db, "shipping", "shipping_bill_item", item.id):
                continue
            severity = SEVERITY_MAP["shipping"].get(item.reconciliation_status, "medium")
            ExceptionService._create(
                db, "shipping", "shipping_bill_item", item.id, severity,
                title=f"运费账单异常: {item.reconciliation_status}",
                description=f"运单号={item.tracking_number or '-'}, 物流商={item.provider_name or '-'}, "
                            f"差异={item.variance_amount or 0}",
                recommended_action="检查账单与订单匹配情况",
            )
            created_count += 1

        # 3. Settlement items: unmatched
        stmt = select(PlatformSettlementItem).where(
            PlatformSettlementItem.match_status == "unmatched",
        )
        result = await db.execute(stmt)
        for item in result.scalars().all():
            total_scanned += 1
            if await ExceptionService._find_existing(db, "settlement", "settlement_item", item.id):
                continue
            ExceptionService._create(
                db, "settlement", "settlement_item", item.id, "medium",
                title=f"结算未匹配: {item.transaction_type} ({item.platform})",
                description=f"订单号={item.order_no or '-'}, 金额={item.amount}",
                recommended_action="确认订单号后重新导入或手动匹配",
            )
            created_count += 1

        # 4. Finance: orders with negative profit
        stmt = select(Order).where(
            Order.profit_amount < 0,
        )
        result = await db.execute(stmt)
        for order in result.scalars().all():
            total_scanned += 1
            if await ExceptionService._find_existing(db, "finance", "order", order.id):
                continue
            ExceptionService._create(
                db, "finance", "order", order.id, "high",
                title=f"订单负利润: {order.profit_amount}",
                description=f"订单ID={order.id}, 订单号={order.order_no}, 利润={order.profit_amount}",
                recommended_action="检查成本输入和费用数据",
            )
            created_count += 1

        await db.flush()

        await OperationLogService.log(
            db,
            module="exception",
            action="generate",
            resource_id="0",
            content=f"扫描生成异常: 新建={created_count}, 扫描={total_scanned}",
            operator=operator or "system",
        )

        return {"created_count": created_count, "total_scanned": total_scanned}

    @staticmethod
    async def _find_existing(
        db: AsyncSession,
        source_module: str,
        source_type: str,
        source_id: int,
    ) -> bool:
        stmt = select(ExceptionItem.id).where(
            ExceptionItem.source_module == source_module,
            ExceptionItem.source_type == source_type,
            ExceptionItem.source_id == source_id,
            ExceptionItem.status.in_(["open", "assigned"]),
        ).limit(1)
        result = await db.execute(stmt)
        return result.scalar_one_or_none() is not None

    @staticmethod
    def _create(
        db: AsyncSession,
        source_module: str,
        source_type: str,
        source_id: int,
        severity: str,
        title: str,
        description: str = "",
        recommended_action: str = "",
    ) -> ExceptionItem:
        item = ExceptionItem(
            source_module=source_module,
            source_type=source_type,
            source_id=source_id,
            severity=severity,
            status="open",
            title=title,
            description=description,
            recommended_action=recommended_action,
        )
        db.add(item)
        return item

    # ── List / Get ─────────────────────────────────────────────────────

    @staticmethod
    async def list_items(
        db: AsyncSession,
        source_module: Optional[str] = None,
        severity: Optional[str] = None,
        status: Optional[str] = None,
    ) -> list[dict]:
        stmt = select(ExceptionItem).order_by(ExceptionItem.created_at.desc())
        if source_module:
            stmt = stmt.where(ExceptionItem.source_module == source_module)
        if severity:
            stmt = stmt.where(ExceptionItem.severity == severity)
        if status:
            stmt = stmt.where(ExceptionItem.status == status)

        result = await db.execute(stmt)
        items = result.scalars().all()
        return [ExceptionService._to_dict(it) for it in items]

    @staticmethod
    async def get_item(db: AsyncSession, exception_id: int) -> Optional[dict]:
        item = await db.get(ExceptionItem, exception_id)
        if not item:
            return None
        return ExceptionService._to_dict(item)

    @staticmethod
    def _to_dict(item: ExceptionItem) -> dict:
        return {
            "id": item.id,
            "source_module": item.source_module,
            "source_type": item.source_type,
            "source_id": item.source_id,
            "severity": item.severity,
            "status": item.status,
            "title": item.title,
            "description": item.description,
            "recommended_action": item.recommended_action,
            "assigned_to": item.assigned_to,
            "resolved_at": item.resolved_at.isoformat() if item.resolved_at else None,
            "resolved_by": item.resolved_by,
            "note": item.note,
            "created_at": item.created_at.isoformat() if item.created_at else None,
            "updated_at": item.updated_at.isoformat() if item.updated_at else None,
        }

    # ── Actions ────────────────────────────────────────────────────────

    @staticmethod
    async def assign_item(
        db: AsyncSession,
        exception_id: int,
        assigned_to: str,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        item = await db.get(ExceptionItem, exception_id)
        if not item:
            return None
        item.status = "assigned"
        item.assigned_to = assigned_to
        await db.flush()
        await db.refresh(item)

        await OperationLogService.log(
            db, module="exception", action="assign",
            resource_id=str(exception_id),
            content=f"分配异常给 {assigned_to}: {item.title}",
            operator=operator or "system",
        )
        return ExceptionService._to_dict(item)

    @staticmethod
    async def resolve_item(
        db: AsyncSession,
        exception_id: int,
        note: str = "",
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        item = await db.get(ExceptionItem, exception_id)
        if not item:
            return None
        item.status = "resolved"
        item.resolved_at = datetime.now(timezone.utc)
        item.resolved_by = operator
        if note:
            item.note = note
        await db.flush()
        await db.refresh(item)

        await OperationLogService.log(
            db, module="exception", action="resolve",
            resource_id=str(exception_id),
            content=f"解决异常: {item.title}",
            operator=operator or "system",
        )
        return ExceptionService._to_dict(item)

    @staticmethod
    async def ignore_item(
        db: AsyncSession,
        exception_id: int,
        note: str = "",
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        item = await db.get(ExceptionItem, exception_id)
        if not item:
            return None
        item.status = "ignored"
        if note:
            item.note = note
        await db.flush()
        await db.refresh(item)

        await OperationLogService.log(
            db, module="exception", action="ignore",
            resource_id=str(exception_id),
            content=f"忽略异常: {item.title}",
            operator=operator or "system",
        )
        return ExceptionService._to_dict(item)
