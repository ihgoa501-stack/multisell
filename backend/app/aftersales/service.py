"""售后管理 - 服务层"""

from datetime import datetime
from decimal import Decimal
from typing import Optional

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models import (
    AfterSalesOrder,
    FinanceLedgerEntry,
    Inventory,
    Order,
    OrderItem,
    Sku,
)
from app.aftersales.schemas import (
    AfterSalesApprove,
    AfterSalesCreate,
    AfterSalesReceive,
    AfterSalesReject,
    AfterSalesRefund,
    AfterSalesOrderVO,
)
from app.inventory.service import InventoryService
from app.operation_log.service import OperationLogService
from app.finance.cost_layers import COST_LAYER_ESTIMATED


_AFTERSALES_STATUS_FLOW = {
    "requested": {"approved", "rejected"},
    "approved": {"received"},
    "received": {"refunded"},
    "refunded": set(),
    "rejected": set(),
}


def _aftersales_to_dict(after: AfterSalesOrder) -> dict:
    order = after.order if hasattr(after, "order") and after.order else None
    sku = after.sku if hasattr(after, "sku") and after.sku else None
    return {
        "id": after.id,
        "order_id": after.order_id,
        "order_no": order.order_no if order else None,
        "item_id": after.item_id,
        "sku_id": after.sku_id,
        "sku_code": sku.code if sku else None,
        "return_quantity": after.return_quantity,
        "reason": after.reason,
        "status": after.status,
        "refund_amount": float(after.refund_amount or 0),
        "inspection_result": after.inspection_result,
        "rejection_reason": after.rejection_reason,
        "created_by": after.created_by,
        "approved_by": after.approved_by,
        "approved_at": after.approved_at,
        "rejected_by": after.rejected_by,
        "rejected_at": after.rejected_at,
        "received_by": after.received_by,
        "received_at": after.received_at,
        "refunded_by": after.refunded_by,
        "refunded_at": after.refunded_at,
        "created_at": after.created_at,
        "updated_at": after.updated_at,
    }


class AfterSalesService:

    @staticmethod
    async def create_return(db: AsyncSession, data: AfterSalesCreate, operator: str = "system") -> dict:
        """创建售后单"""
        # 验证订单存在
        order = await db.get(Order, data.order_id)
        if not order:
            raise ValueError("订单不存在")
        if order.status in ("pending", "cancelled"):
            raise ValueError("订单尚未支付或已取消，无法发起售后")

        # 验证SKU存在
        sku = await db.get(Sku, data.sku_id)
        if not sku:
            raise ValueError("SKU不存在")

        # 如果指定了item_id，验证其属于该订单
        if data.item_id:
            item = await db.get(OrderItem, data.item_id)
            if not item or item.order_id != data.order_id:
                raise ValueError("订单明细不属于该订单")

        after = AfterSalesOrder(
            order_id=data.order_id,
            item_id=data.item_id,
            sku_id=data.sku_id,
            return_quantity=data.return_quantity,
            reason=data.reason,
            status="requested",
            refund_amount=Decimal(str(data.refund_amount)),
            created_by=operator,
        )
        db.add(after)
        await db.flush()
        await db.refresh(after)

        await OperationLogService.log(
            db,
            module="aftersales",
            action="create",
            resource_id=str(after.id),
            content=f"创建售后单: order_id={data.order_id}, sku_id={data.sku_id}, qty={data.return_quantity}",
            operator=operator,
        )

        return _aftersales_to_dict(after)

    @staticmethod
    async def approve(db: AsyncSession, after_id: int, data: AfterSalesApprove, operator: str = "system") -> Optional[dict]:
        """审批通过售后单"""
        after = await db.get(AfterSalesOrder, after_id)
        if not after:
            return None
        if after.status != "requested":
            raise ValueError(f"售后单状态为 {after.status}，无法审批")

        after.status = "approved"
        after.approved_by = operator
        after.approved_at = datetime.utcnow()
        if data.inspection_result:
            after.inspection_result = data.inspection_result
        await db.flush()
        await db.refresh(after)

        await OperationLogService.log(
            db,
            module="aftersales",
            action="approve",
            resource_id=str(after_id),
            content=f"审批通过售后单: {after_id}",
            operator=operator,
        )

        return _aftersales_to_dict(after)

    @staticmethod
    async def reject(db: AsyncSession, after_id: int, data: AfterSalesReject, operator: str = "system") -> Optional[dict]:
        """驳回售后单"""
        after = await db.get(AfterSalesOrder, after_id)
        if not after:
            return None
        if after.status != "requested":
            raise ValueError(f"售后单状态为 {after.status}，无法驳回")

        after.status = "rejected"
        after.rejected_by = operator
        after.rejected_at = datetime.utcnow()
        after.rejection_reason = data.rejection_reason
        await db.flush()
        await db.refresh(after)

        await OperationLogService.log(
            db,
            module="aftersales",
            action="reject",
            resource_id=str(after_id),
            content=f"驳回售后单: {after_id}, 原因: {data.rejection_reason}",
            operator=operator,
        )

        return _aftersales_to_dict(after)

    @staticmethod
    async def receive_return(db: AsyncSession, after_id: int, data: AfterSalesReceive, operator: str = "system") -> Optional[dict]:
        """收货入库：仓库收到退货，恢复库存"""
        after = await db.get(AfterSalesOrder, after_id)
        if not after:
            return None
        if after.status != "approved":
            raise ValueError(f"售后单状态为 {after.status}，无法收货")

        # 恢复库存
        try:
            await InventoryService.restock(
                db,
                sku_id=after.sku_id,
                quantity=after.return_quantity,
                remark=f"退货入库: 售后单 #{after_id}",
                operator=operator,
            )
        except ValueError as e:
            raise ValueError(f"库存恢复失败: {e}")

        after.status = "received"
        after.received_by = operator
        after.received_at = datetime.utcnow()
        if data.inspection_result:
            after.inspection_result = data.inspection_result
        await db.flush()
        await db.refresh(after)

        await OperationLogService.log(
            db,
            module="aftersales",
            action="receive",
            resource_id=str(after_id),
            content=f"收货入库售后单: {after_id}, sku_id={after.sku_id}, qty={after.return_quantity}",
            operator=operator,
        )

        return _aftersales_to_dict(after)

    @staticmethod
    async def complete_refund(db: AsyncSession, after_id: int, data: AfterSalesRefund, operator: str = "system") -> Optional[dict]:
        """完成退款：创建财务账本条目"""
        after = await db.get(AfterSalesOrder, after_id)
        if not after:
            return None
        if after.status != "received":
            raise ValueError(f"售后单状态为 {after.status}，无法退款")

        order = await db.get(Order, after.order_id)
        if not order:
            raise ValueError("关联订单不存在")

        refund_amount = Decimal(str(after.refund_amount))
        order_id = after.order_id

        # 创建财务账本条目 - 退款
        entry = FinanceLedgerEntry(
            order_id=order_id,
            entry_type="refund",
            amount=-refund_amount,
            currency="CNY",
            cost_layer=COST_LAYER_ESTIMATED,
            source_type="aftersales",
            source_id=after.id,
            description=f"售后退款: 售后单 #{after_id}, SKU {after.sku_id}, qty={after.return_quantity}",
        )
        db.add(entry)

        # 反向商品成本
        if order.product_cost and order.product_cost > 0:
            order_total_refund_ratio = refund_amount / Decimal(str(order.pay_amount or 1))
            product_cost_refund = Decimal(str(order.product_cost)) * order_total_refund_ratio
            cost_entry = FinanceLedgerEntry(
                order_id=order_id,
                entry_type="product_cost",
                amount=product_cost_refund,
                currency="CNY",
                cost_layer=COST_LAYER_ESTIMATED,
                source_type="aftersales",
                source_id=after.id,
                description=f"售后冲销商品成本: 售后单 #{after_id}",
            )
            db.add(cost_entry)

        after.status = "refunded"
        after.refunded_by = operator
        after.refunded_at = datetime.utcnow()
        await db.flush()
        await db.refresh(after)

        await OperationLogService.log(
            db,
            module="aftersales",
            action="refund",
            resource_id=str(after_id),
            content=f"完成退款: 售后单 {after_id}, 金额={refund_amount}",
            operator=operator,
        )

        return _aftersales_to_dict(after)

    @staticmethod
    async def list_returns(
        db: AsyncSession,
        status: Optional[str] = None,
        order_id: Optional[int] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict], int]:
        """分页查询售后单列表"""
        stmt = (
            select(AfterSalesOrder)
            .options(selectinload(AfterSalesOrder.order), selectinload(AfterSalesOrder.sku))
        )
        if status:
            stmt = stmt.where(AfterSalesOrder.status == status)
        if order_id:
            stmt = stmt.where(AfterSalesOrder.order_id == order_id)

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        stmt = stmt.order_by(AfterSalesOrder.created_at.desc())
        stmt = stmt.offset((page - 1) * page_size).limit(page_size)
        result = await db.execute(stmt)
        afters = list(result.scalars().all())

        return [_aftersales_to_dict(a) for a in afters], total

    @staticmethod
    async def get_return(db: AsyncSession, after_id: int) -> Optional[dict]:
        """获取售后单详情"""
        stmt = (
            select(AfterSalesOrder)
            .options(selectinload(AfterSalesOrder.order), selectinload(AfterSalesOrder.sku))
            .where(AfterSalesOrder.id == after_id)
        )
        result = await db.execute(stmt)
        after = result.scalar_one_or_none()
        if not after:
            return None
        return _aftersales_to_dict(after)
