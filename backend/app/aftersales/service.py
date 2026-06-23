"""售后退货 — service"""

from datetime import datetime, timezone
from decimal import Decimal
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.inventory.service import InventoryService
from app.models import AfterSalesOrder


ALLOWED_TRANSITIONS = {
    "pending": {"approved", "rejected"},
    "approved": {"received", "rejected"},
    "rejected": set(),
    "received": {"refunded"},
    "refunded": set(),
}


class AfterSalesService:
    @staticmethod
    async def create(db: AsyncSession, data: dict, created_by: str) -> AfterSalesOrder:
        """创建退货申请"""
        rma = AfterSalesOrder(
            order_id=data["order_id"],
            item_id=data.get("item_id"),
            sku_id=data["sku_id"],
            return_quantity=data["return_quantity"],
            reason=data["reason"],
            status="pending",
            created_by=created_by,
        )
        db.add(rma)
        await db.flush()
        await db.refresh(rma)
        return rma

    @staticmethod
    async def approve(
        db: AsyncSession, rma_id: int, approved_by: str, refund_amount: Decimal
    ) -> Optional[AfterSalesOrder]:
        """审批通过退货"""
        rma = await db.get(AfterSalesOrder, rma_id)
        if not rma or rma.status != "pending":
            return None
        rma.status = "approved"
        rma.refund_amount = refund_amount
        rma.approved_by = approved_by
        rma.approved_at = datetime.now(timezone.utc)
        await db.flush()
        await db.refresh(rma)
        return rma

    @staticmethod
    async def reject(
        db: AsyncSession, rma_id: int, rejected_by: str, reason: str
    ) -> Optional[AfterSalesOrder]:
        """驳回退货"""
        rma = await db.get(AfterSalesOrder, rma_id)
        if not rma or rma.status not in ("pending", "approved"):
            return None
        rma.status = "rejected"
        rma.rejected_by = rejected_by
        rma.rejected_at = datetime.now(timezone.utc)
        rma.rejection_reason = reason
        await db.flush()
        await db.refresh(rma)
        return rma

    @staticmethod
    async def receive(
        db: AsyncSession,
        rma_id: int,
        received_by: str,
        inspection: Optional[str] = None,
    ) -> Optional[AfterSalesOrder]:
        """入库验收 — 恢复库存"""
        rma = await db.get(AfterSalesOrder, rma_id)
        if not rma or rma.status != "approved":
            return None
        rma.status = "received"
        rma.received_by = received_by
        rma.received_at = datetime.now(timezone.utc)
        rma.inspection_result = inspection
        # restore inventory — add returned quantity back to stock
        inv = await InventoryService.get_inventory(db, rma.sku_id)
        current = inv.quantity if inv else 0
        await InventoryService.update_inventory(
            db,
            rma.sku_id,
            current + rma.return_quantity,
            remark=f"退货入库 rma_id={rma_id}",
        )
        await db.flush()
        await db.refresh(rma)
        return rma

    @staticmethod
    async def refund(
        db: AsyncSession, rma_id: int, refunded_by: str
    ) -> Optional[AfterSalesOrder]:
        """确认退款"""
        rma = await db.get(AfterSalesOrder, rma_id)
        if not rma or rma.status != "received":
            return None
        rma.status = "refunded"
        rma.refunded_by = refunded_by
        rma.refunded_at = datetime.now(timezone.utc)
        await db.flush()
        await db.refresh(rma)
        return rma

    @staticmethod
    async def get_by_id(db: AsyncSession, rma_id: int) -> Optional[AfterSalesOrder]:
        return await db.get(AfterSalesOrder, rma_id)

    @staticmethod
    async def list_by_order(db: AsyncSession, order_id: int) -> list[AfterSalesOrder]:
        stmt = (
            select(AfterSalesOrder)
            .where(AfterSalesOrder.order_id == order_id)
            .order_by(AfterSalesOrder.created_at.desc())
        )
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def list_all(
        db: AsyncSession, status: Optional[str] = None, limit: int = 50
    ) -> list[AfterSalesOrder]:
        stmt = select(AfterSalesOrder)
        if status:
            stmt = stmt.where(AfterSalesOrder.status == status)
        stmt = stmt.order_by(AfterSalesOrder.created_at.desc()).limit(limit)
        result = await db.execute(stmt)
        return list(result.scalars().all())
