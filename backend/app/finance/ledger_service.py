"""订单真实利润账本 - 服务层"""

from decimal import Decimal
from typing import Optional

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    FinanceLedgerEntry,
    Order,
    OrderShippingSnapshot,
    PlatformSettlementItem,
    ShippingBillItem,
)
from app.operation_log.service import OperationLogService
from app.finance.cost_layers import (
    COST_LAYER_SNAPSHOT,
    COST_LAYER_ACTUAL,
    COST_LAYER_ESTIMATED,
    COST_LAYER_MIXED,
    resolve_profit_cost_layer,
)

# Ledger entry types
ENTRY_REVENUE = "revenue"
ENTRY_PRODUCT_COST = "product_cost"
ENTRY_SHIPPING_COST = "shipping_cost"
ENTRY_PLATFORM_FEE = "platform_fee"
ENTRY_PAYMENT_FEE = "payment_fee"
ENTRY_REFUND = "refund"
ENTRY_ADJUSTMENT = "adjustment"
ENTRY_OTHER_FEE = "other_fee"
ENTRY_ALLOCATED_COST = "allocated_cost"


def _money(val) -> float:
    return float(val or 0)


def _decimal(val) -> Decimal:
    return Decimal(str(val or 0))


def _pct(numerator: Decimal, denominator: Decimal) -> float:
    if denominator <= 0:
        return 0.0
    return float(((numerator / denominator) * Decimal("100")).quantize(Decimal("0.01")))


class LedgerService:

    @staticmethod
    async def rebuild(
        db: AsyncSession,
        order_id: int,
        operator: Optional[str] = None,
    ) -> dict:
        """重建指定订单的利润账本。"""
        order = await db.get(Order, order_id)
        if not order:
            raise ValueError("订单不存在")

        # 1. 清空旧账本
        stmt = select(FinanceLedgerEntry).where(FinanceLedgerEntry.order_id == order_id)
        result = await db.execute(stmt)
        for entry in result.scalars().all():
            await db.delete(entry)
        await db.flush()

        # 2. Revenue — 订单商品总额
        revenue = _decimal(order.total_amount)
        await LedgerService._add_entry(
            db, order_id, ENTRY_REVENUE, revenue,
            cost_layer=COST_LAYER_ACTUAL,
            source_type="order", source_id=order.id,
            description=f"订单商品总额",
        )

        # 3. Product cost
        product_cost = _decimal(order.product_cost)
        await LedgerService._add_entry(
            db, order_id, ENTRY_PRODUCT_COST, -product_cost,
            cost_layer=COST_LAYER_ESTIMATED,
            source_type="order", source_id=order.id,
            description=f"商品成本",
        )

        # 4. Shipping cost — prefer actual bill over snapshot over order default
        shipping_cost_layer = COST_LAYER_ESTIMATED
        shipping_amount = _decimal(order.shipping_fee)
        shipping_description = "订单运费（未绑定快照）"
        shipping_source_type = "order"
        shipping_source_id = order.id

        # Try to find matched shipping bill row (actual)
        bill_stmt = (
            select(ShippingBillItem)
            .where(
                ShippingBillItem.matched_order_id == order_id,
                ShippingBillItem.reconciliation_status.in_(["matched", "amount_mismatch", "manual_resolved"]),
            )
            .order_by(ShippingBillItem.created_at.desc())
            .limit(1)
        )
        bill_result = await db.execute(bill_stmt)
        bill_item = bill_result.scalar_one_or_none()

        if bill_item:
            shipping_amount = _decimal(bill_item.total_actual_fee)
            shipping_cost_layer = COST_LAYER_ACTUAL
            shipping_description = f"物流商账单: {bill_item.provider_name} #{bill_item.tracking_number or ''}"
            shipping_source_type = "shipping_bill_row"
            shipping_source_id = bill_item.id
        else:
            # Try shipping snapshot
            snap_stmt = select(OrderShippingSnapshot).where(
                OrderShippingSnapshot.order_id == order_id
            )
            snap_result = await db.execute(snap_stmt)
            snapshot = snap_result.scalar_one_or_none()
            if snapshot:
                shipping_amount = _decimal(snapshot.total_shipping_fee)
                shipping_cost_layer = COST_LAYER_SNAPSHOT
                shipping_description = f"运费快照: {snapshot.provider_name} {snapshot.channel_name}"
                shipping_source_type = "shipping_snapshot"
                shipping_source_id = snapshot.id

        await LedgerService._add_entry(
            db, order_id, ENTRY_SHIPPING_COST, -shipping_amount,
            cost_layer=shipping_cost_layer,
            source_type=shipping_source_type, source_id=shipping_source_id,
            description=shipping_description,
        )

        # 5. Platform settlement rows matched to this order
        settlement_stmt = (
            select(PlatformSettlementItem)
            .where(
                PlatformSettlementItem.matched_order_id == order_id,
                PlatformSettlementItem.match_status == "matched",
            )
        )
        settlement_result = await db.execute(settlement_stmt)
        settlement_items = settlement_result.scalars().all()

        platform_fee_layer = COST_LAYER_ESTIMATED
        platform_fee_total = Decimal("0")
        payment_fee_total = Decimal("0")
        refund_total = Decimal("0")
        adjustment_total = Decimal("0")
        other_fee_total = Decimal("0")

        for si in settlement_items:
            amount = _decimal(si.amount)
            desc = si.description or f"{si.transaction_type} ({si.platform})"

            if si.transaction_type == "platform_fee":
                platform_fee_total += amount
                platform_fee_layer = COST_LAYER_ACTUAL
                await LedgerService._add_entry(
                    db, order_id, ENTRY_PLATFORM_FEE, -amount,
                    cost_layer=COST_LAYER_ACTUAL,
                    source_type="settlement_row", source_id=si.id,
                    description=desc,
                )
            elif si.transaction_type == "payment_fee":
                payment_fee_total += amount
                await LedgerService._add_entry(
                    db, order_id, ENTRY_PAYMENT_FEE, -amount,
                    cost_layer=COST_LAYER_ACTUAL,
                    source_type="settlement_row", source_id=si.id,
                    description=desc,
                )
            elif si.transaction_type == "refund":
                refund_total += amount
                await LedgerService._add_entry(
                    db, order_id, ENTRY_REFUND, amount,  # refund reduces profit, keep signed value
                    cost_layer=COST_LAYER_ACTUAL,
                    source_type="settlement_row", source_id=si.id,
                    description=desc,
                )
            elif si.transaction_type == "adjustment":
                adjustment_total += amount
                await LedgerService._add_entry(
                    db, order_id, ENTRY_ADJUSTMENT, -amount,
                    cost_layer=COST_LAYER_ACTUAL,
                    source_type="settlement_row", source_id=si.id,
                    description=desc,
                )
            elif si.transaction_type == "other":
                other_fee_total += amount
                await LedgerService._add_entry(
                    db, order_id, ENTRY_OTHER_FEE, -amount,
                    cost_layer=COST_LAYER_ACTUAL,
                    source_type="settlement_row", source_id=si.id,
                    description=desc,
                )

        # If no settlement platform_fee, use order default
        if platform_fee_total == 0 and _decimal(order.platform_fee) > 0:
            platform_fee_layer = COST_LAYER_ESTIMATED
            await LedgerService._add_entry(
                db, order_id, ENTRY_PLATFORM_FEE, -_decimal(order.platform_fee),
                cost_layer=COST_LAYER_ESTIMATED,
                source_type="order", source_id=order.id,
                description="平台佣金（未导入结算时）",
            )

        await db.flush()

        # 6. Calculate profit
        profit_layer = resolve_profit_cost_layer(shipping_cost_layer, platform_fee_layer)

        # Audit log
        await OperationLogService.log(
            db,
            module="finance_ledger",
            action="rebuild",
            resource_id=str(order_id),
            content=f"重建订单利润账本: order_id={order_id}",
            operator=operator or "system",
        )

        return await LedgerService.get_profit(db, order_id)

    @staticmethod
    async def _add_entry(
        db: AsyncSession,
        order_id: int,
        entry_type: str,
        amount: Decimal,
        cost_layer: str,
        source_type: Optional[str] = None,
        source_id: Optional[int] = None,
        description: Optional[str] = None,
    ) -> FinanceLedgerEntry:
        entry = FinanceLedgerEntry(
            order_id=order_id,
            entry_type=entry_type,
            amount=amount,
            currency="CNY",
            cost_layer=cost_layer,
            source_type=source_type,
            source_id=source_id,
            description=description,
        )
        db.add(entry)
        return entry

    @staticmethod
    async def get_ledger(db: AsyncSession, order_id: int) -> dict:
        """获取订单账本条目列表。"""
        order = await db.get(Order, order_id)
        if not order:
            raise ValueError("订单不存在")

        stmt = (
            select(FinanceLedgerEntry)
            .where(FinanceLedgerEntry.order_id == order_id)
            .order_by(FinanceLedgerEntry.id)
        )
        result = await db.execute(stmt)
        entries = result.scalars().all()

        return {
            "order_id": order_id,
            "entries": [
                {
                    "id": e.id,
                    "order_id": e.order_id,
                    "entry_type": e.entry_type,
                    "amount": float(e.amount),
                    "currency": e.currency or "CNY",
                    "cost_layer": e.cost_layer,
                    "source_type": e.source_type,
                    "source_id": e.source_id,
                    "description": e.description,
                    "created_at": e.created_at.isoformat() if e.created_at else None,
                }
                for e in entries
            ],
            "total_entries": len(entries),
        }

    @staticmethod
    async def get_profit(db: AsyncSession, order_id: int) -> dict:
        """获取订单利润账本汇总。"""
        order = await db.get(Order, order_id)
        if not order:
            raise ValueError("订单不存在")

        # Calculate from ledger if exists, else from order defaults
        stmt = (
            select(FinanceLedgerEntry)
            .where(FinanceLedgerEntry.order_id == order_id)
        )
        result = await db.execute(stmt)
        entries = result.scalars().all()

        if not entries:
            # No ledger — return default from order model
            revenue = _money(order.total_amount)
            product_cost = _money(order.product_cost)
            shipping_fee = _money(order.shipping_fee)
            platform_fee = _money(order.platform_fee)
            payment_fee = _money(order.payment_fee)
            other_fee = _money(order.other_fee)
            shipping_cost_layer = COST_LAYER_ESTIMATED
            platform_fee_cost_layer = COST_LAYER_ESTIMATED
            profit_amount = revenue - product_cost - shipping_fee - platform_fee - payment_fee - other_fee
            profit_margin = _pct(_decimal(profit_amount), _decimal(revenue)) if revenue > 0 else 0
            profit_cost_layer = resolve_profit_cost_layer(shipping_cost_layer, platform_fee_cost_layer)
            return {
                "order_id": order_id,
                "revenue_amount": revenue,
                "product_cost": product_cost,
                "shipping_cost": shipping_fee,
                "platform_fee": platform_fee,
                "payment_fee": payment_fee,
                "refund": 0,
                "adjustment": 0,
                "other_fee": other_fee,
                "profit_amount": profit_amount,
                "profit_margin": profit_margin,
                "shipping_cost_layer": shipping_cost_layer,
                "platform_fee_cost_layer": platform_fee_cost_layer,
                "profit_cost_layer": profit_cost_layer,
                "ledger_built": False,
            }

        # Sum from ledger entries
        totals = {}
        layers = set()
        for e in entries:
            totals[e.entry_type] = (_decimal(totals.get(e.entry_type, 0)) + _decimal(e.amount))
            if e.cost_layer:
                layers.add(e.cost_layer)

        revenue_amount = float(totals.get(ENTRY_REVENUE, 0))
        product_cost = -float(totals.get(ENTRY_PRODUCT_COST, 0))  # stored negative
        shipping_cost = -float(totals.get(ENTRY_SHIPPING_COST, 0))
        platform_fee = -float(totals.get(ENTRY_PLATFORM_FEE, 0))
        payment_fee = -float(totals.get(ENTRY_PAYMENT_FEE, 0))
        refund = float(totals.get(ENTRY_REFUND, 0))
        adjustment = -float(totals.get(ENTRY_ADJUSTMENT, 0))
        other_fee = -float(totals.get(ENTRY_OTHER_FEE, 0))

        profit_amount = revenue_amount - product_cost - shipping_cost - platform_fee - payment_fee + refund - adjustment - other_fee
        profit_margin = _pct(_decimal(profit_amount), _decimal(revenue_amount)) if revenue_amount > 0 else 0

        # Determine cost layers
        shipping_cost_layer = COST_LAYER_ESTIMATED
        platform_fee_cost_layer = COST_LAYER_ESTIMATED
        for e in entries:
            if e.entry_type == ENTRY_SHIPPING_COST and e.cost_layer:
                shipping_cost_layer = e.cost_layer
            if e.entry_type == ENTRY_PLATFORM_FEE and e.cost_layer:
                platform_fee_cost_layer = e.cost_layer

        profit_cost_layer = resolve_profit_cost_layer(shipping_cost_layer, platform_fee_cost_layer)

        return {
            "order_id": order_id,
            "revenue_amount": round(revenue_amount, 2),
            "product_cost": round(product_cost, 2),
            "shipping_cost": round(shipping_cost, 2),
            "platform_fee": round(platform_fee, 2),
            "payment_fee": round(payment_fee, 2),
            "refund": round(refund, 2),
            "adjustment": round(adjustment, 2),
            "other_fee": round(other_fee, 2),
            "profit_amount": round(profit_amount, 2),
            "profit_margin": profit_margin,
            "shipping_cost_layer": shipping_cost_layer,
            "platform_fee_cost_layer": platform_fee_cost_layer,
            "profit_cost_layer": profit_cost_layer,
            "ledger_built": True,
        }
