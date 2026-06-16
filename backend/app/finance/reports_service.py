"""财务报表 - 服务层"""

from datetime import datetime, timezone
from decimal import Decimal
from typing import Optional

from sqlalchemy import func, select, case, cast, Float
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    FinanceLedgerEntry,
    Order,
    OrderShippingSnapshot,
    ShippingBillItem,
)
from app.finance.ledger_service import (
    ENTRY_REVENUE,
    ENTRY_PRODUCT_COST,
    ENTRY_SHIPPING_COST,
    ENTRY_PLATFORM_FEE,
    ENTRY_PAYMENT_FEE,
    ENTRY_REFUND,
    ENTRY_ADJUSTMENT,
    ENTRY_OTHER_FEE,
    ENTRY_ALLOCATED_COST,
)


def _decimal(val) -> Decimal:
    return Decimal(str(val or 0))


def _float(val) -> float:
    return float(val or 0)


def _pct(numerator: Decimal, denominator: Decimal) -> float:
    if denominator <= 0:
        return 0.0
    return float(((numerator / denominator) * Decimal("100")).quantize(Decimal("0.01")))


def _build_date_filter(stmt, date_from=None, date_to=None):
    """Apply date filter to the statement if provided."""
    if date_from:
        stmt = stmt.where(Order.created_at >= date_from)
    if date_to:
        stmt = stmt.where(Order.created_at <= date_to)
    return stmt


def _build_order_ids_from_filters(db, date_from, date_to, order_no, platform_id, sku_id):
    """Build order_id list from filter conditions."""
    stmt = select(Order.id)
    if date_from:
        stmt = stmt.where(Order.created_at >= date_from)
    if date_to:
        stmt = stmt.where(Order.created_at <= date_to)
    if order_no:
        stmt = stmt.where(Order.order_no == order_no)
    if platform_id:
        pass  # Order does not have platform_id directly; skip
    if sku_id:
        pass  # Resolution via OrderItem; skip for simplicity
    return stmt


class ReportService:

    @staticmethod
    async def profit_summary(
        db: AsyncSession,
        date_from: Optional[str] = None,
        date_to: Optional[str] = None,
    ) -> dict:
        """汇总 ledger 条目得到利润摘要。"""
        # Get order IDs within date range
        order_ids = await ReportService._filtered_order_ids(db, date_from, date_to)

        if not order_ids:
            return {
                "revenue_amount": 0, "product_cost": 0, "shipping_cost": 0,
                "platform_fee": 0, "payment_fee": 0, "refund": 0,
                "adjustment": 0, "allocated_cost": 0, "other_fee": 0,
                "profit_amount": 0, "profit_margin": 0,
            }

        stmt = select(
            FinanceLedgerEntry.entry_type,
            func.sum(FinanceLedgerEntry.amount).label("total"),
        ).where(
            FinanceLedgerEntry.order_id.in_(order_ids),
        ).group_by(FinanceLedgerEntry.entry_type)

        result = await db.execute(stmt)
        rows = result.all()

        totals = {row.entry_type: _decimal(row.total) for row in rows}

        revenue = _float(totals.get(ENTRY_REVENUE, 0))
        product_cost = -_float(totals.get(ENTRY_PRODUCT_COST, 0))
        shipping_cost = -_float(totals.get(ENTRY_SHIPPING_COST, 0))
        platform_fee = -_float(totals.get(ENTRY_PLATFORM_FEE, 0))
        payment_fee = -_float(totals.get(ENTRY_PAYMENT_FEE, 0))
        refund = _float(totals.get(ENTRY_REFUND, 0))
        adjustment = -_float(totals.get(ENTRY_ADJUSTMENT, 0))
        allocated_cost = -_float(totals.get(ENTRY_ALLOCATED_COST, 0))
        other_fee = -_float(totals.get(ENTRY_OTHER_FEE, 0))

        profit_amount = revenue - product_cost - shipping_cost - platform_fee - payment_fee + refund - adjustment - allocated_cost - other_fee
        profit_margin = _pct(_decimal(profit_amount), _decimal(revenue)) if revenue > 0 else 0

        return {
            "revenue_amount": round(revenue, 2),
            "product_cost": round(product_cost, 2),
            "shipping_cost": round(shipping_cost, 2),
            "platform_fee": round(platform_fee, 2),
            "payment_fee": round(payment_fee, 2),
            "refund": round(refund, 2),
            "adjustment": round(adjustment, 2),
            "allocated_cost": round(allocated_cost, 2),
            "other_fee": round(other_fee, 2),
            "profit_amount": round(profit_amount, 2),
            "profit_margin": profit_margin,
        }

    @staticmethod
    async def order_profit(
        db: AsyncSession,
        date_from: Optional[str] = None,
        date_to: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """按订单展示利润。"""
        order_ids = await ReportService._filtered_order_ids(db, date_from, date_to)

        if not order_ids:
            return {"items": [], "total": 0, "page": page, "page_size": page_size}

        total = len(order_ids)
        # Paginate
        paginated_ids = order_ids[(page - 1) * page_size: page * page_size]

        items = []
        for oid in paginated_ids:
            profit = await ReportService._get_order_profit(db, oid)
            if profit:
                items.append(profit)

        return {"items": items, "total": total, "page": page, "page_size": page_size}

    @staticmethod
    async def cost_variance(
        db: AsyncSession,
        date_from: Optional[str] = None,
        date_to: Optional[str] = None,
    ) -> list[dict]:
        """运费差异：快照 vs 实际账单。"""
        order_ids = await ReportService._filtered_order_ids(db, date_from, date_to)

        if not order_ids:
            return []

        snap_stmt = select(
            OrderShippingSnapshot.order_id,
            OrderShippingSnapshot.total_shipping_fee,
        ).where(
            OrderShippingSnapshot.order_id.in_(order_ids),
        )
        snap_result = await db.execute(snap_stmt)
        snapshots = {row.order_id: _float(row.total_shipping_fee) for row in snap_result.all()}

        bill_stmt = select(
            ShippingBillItem.matched_order_id,
            ShippingBillItem.total_actual_fee,
            ShippingBillItem.reconciliation_status,
        ).where(
            ShippingBillItem.matched_order_id.in_(order_ids),
            ShippingBillItem.reconciliation_status.in_(["matched", "amount_mismatch"]),
        )
        bill_result = await db.execute(bill_stmt)
        bills = {}
        for row in bill_result.all():
            if row.matched_order_id not in bills:
                bills[row.matched_order_id] = {
                    "amount": _float(row.total_actual_fee),
                    "status": row.reconciliation_status,
                }

        order_stmt = select(Order.id, Order.order_no).where(Order.id.in_(order_ids))
        order_result = await db.execute(order_stmt)
        orders = {row.id: row.order_no for row in order_result.all()}

        results = []
        for oid in order_ids:
            if oid not in orders:
                continue
            snap_amt = snapshots.get(oid)
            bill_info = bills.get(oid)
            if bill_info:
                variance = bill_info["amount"] - (snap_amt or 0)
                variance_pct = round((variance / (snap_amt or 1)) * 100, 2) if snap_amt else None
                results.append({
                    "order_id": oid,
                    "order_no": orders[oid],
                    "snapshot_amount": snap_amt,
                    "bill_amount": bill_info["amount"],
                    "variance_amount": round(variance, 2),
                    "variance_pct": variance_pct,
                    "status": bill_info["status"],
                })
            elif snap_amt is not None:
                results.append({
                    "order_id": oid,
                    "order_no": orders[oid],
                    "snapshot_amount": snap_amt,
                    "bill_amount": None,
                    "variance_amount": None,
                    "variance_pct": None,
                    "status": "no_bill",
                })

        return results

    @staticmethod
    async def negative_profit(
        db: AsyncSession,
        date_from: Optional[str] = None,
        date_to: Optional[str] = None,
    ) -> list[dict]:
        """利润为负的订单列表。"""
        order_ids = await ReportService._filtered_order_ids(db, date_from, date_to)

        if not order_ids:
            return []

        # Get profitable from order model
        stmt = select(Order).where(
            Order.id.in_(order_ids),
            Order.profit_amount < 0,
        ).order_by(Order.profit_amount.asc()).limit(100)
        result = await db.execute(stmt)
        orders = result.scalars().all()

        items = []
        for o in orders:
            profit = await ReportService._get_order_profit(db, o.id)
            if profit:
                items.append({
                    "order_id": o.id,
                    "order_no": o.order_no,
                    "profit_amount": profit["profit_amount"],
                    "profit_margin": profit["profit_margin"],
                    "shipping_cost_layer": profit.get("shipping_cost_layer", "estimated"),
                    "platform_fee_cost_layer": profit.get("platform_fee_cost_layer", "estimated"),
                    "profit_cost_layer": profit.get("profit_cost_layer", "estimated"),
                })
        return items

    @staticmethod
    async def cost_layer_mix(
        db: AsyncSession,
        date_from: Optional[str] = None,
        date_to: Optional[str] = None,
    ) -> dict:
        """成本层分布统计。"""
        order_ids = await ReportService._filtered_order_ids(db, date_from, date_to)

        if not order_ids:
            return {"layers": []}

        stmt = select(
            FinanceLedgerEntry.cost_layer,
            func.count(FinanceLedgerEntry.id).label("entry_count"),
            func.sum(cast(FinanceLedgerEntry.amount, Float)).label("total_amount"),
        ).where(
            FinanceLedgerEntry.order_id.in_(order_ids),
        ).group_by(FinanceLedgerEntry.cost_layer)

        result = await db.execute(stmt)
        rows = result.all()

        layers = []
        for row in rows:
            layers.append({
                "cost_layer": row.cost_layer or "unknown",
                "entry_count": row.entry_count or 0,
                "total_amount": round(_float(row.total_amount), 2),
            })

        return {"layers": layers}

    @staticmethod
    async def _get_order_profit(db: AsyncSession, order_id: int) -> Optional[dict]:
        """Get profit summary for one order from its ledger entries."""
        order = await db.get(Order, order_id)
        if not order:
            return None

        stmt = select(
            FinanceLedgerEntry.entry_type,
            func.sum(FinanceLedgerEntry.amount).label("total"),
        ).where(
            FinanceLedgerEntry.order_id == order_id,
        ).group_by(FinanceLedgerEntry.entry_type)
        result = await db.execute(stmt)
        rows = result.all()

        totals = {row.entry_type: _decimal(row.total) for row in rows}
        revenue = _float(totals.get(ENTRY_REVENUE, 0))
        product_cost = -_float(totals.get(ENTRY_PRODUCT_COST, 0))
        shipping_cost = -_float(totals.get(ENTRY_SHIPPING_COST, 0))
        platform_fee = -_float(totals.get(ENTRY_PLATFORM_FEE, 0))
        payment_fee = -_float(totals.get(ENTRY_PAYMENT_FEE, 0))
        refund = _float(totals.get(ENTRY_REFUND, 0))
        adjustment = -_float(totals.get(ENTRY_ADJUSTMENT, 0))
        allocated_cost = -_float(totals.get(ENTRY_ALLOCATED_COST, 0))
        other_fee = -_float(totals.get(ENTRY_OTHER_FEE, 0))

        profit_amount = revenue - product_cost - shipping_cost - platform_fee - payment_fee + refund - adjustment - allocated_cost - other_fee
        profit_margin = _pct(_decimal(profit_amount), _decimal(revenue)) if revenue > 0 else 0

        # Cost layers
        layer_stmt = select(
            FinanceLedgerEntry.entry_type,
            FinanceLedgerEntry.cost_layer,
        ).where(
            FinanceLedgerEntry.order_id == order_id,
            FinanceLedgerEntry.entry_type.in_([ENTRY_SHIPPING_COST, ENTRY_PLATFORM_FEE]),
        ).distinct()
        layer_result = await db.execute(layer_stmt)
        layers = {}
        for row in layer_result.all():
            layers[row.entry_type] = row.cost_layer

        shipping_cost_layer = layers.get(ENTRY_SHIPPING_COST, "estimated")
        platform_fee_cost_layer = layers.get(ENTRY_PLATFORM_FEE, "estimated")

        # Determine profit cost layer
        from app.finance.cost_layers import resolve_profit_cost_layer
        profit_cost_layer = resolve_profit_cost_layer(shipping_cost_layer, platform_fee_cost_layer)

        return {
            "order_id": order_id,
            "order_no": order.order_no,
            "revenue_amount": round(revenue, 2),
            "product_cost": round(product_cost, 2),
            "shipping_cost": round(shipping_cost, 2),
            "platform_fee": round(platform_fee, 2),
            "payment_fee": round(payment_fee, 2),
            "other_fee": round(other_fee, 2),
            "allocated_cost": round(allocated_cost, 2),
            "refund": round(refund, 2),
            "adjustment": round(adjustment, 2),
            "profit_amount": round(profit_amount, 2),
            "profit_margin": profit_margin,
            "shipping_cost_layer": shipping_cost_layer,
            "platform_fee_cost_layer": platform_fee_cost_layer,
            "profit_cost_layer": profit_cost_layer,
        }

    @staticmethod
    async def _filtered_order_ids(
        db: AsyncSession,
        date_from: Optional[str] = None,
        date_to: Optional[str] = None,
    ) -> list[int]:
        stmt = select(Order.id)
        if date_from:
            try:
                dt = datetime.fromisoformat(date_from.replace("Z", "+00:00"))
            except ValueError:
                dt = datetime.fromisoformat(date_from + "T00:00:00+00:00")
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            stmt = stmt.where(Order.created_at >= dt)
        if date_to:
            try:
                dt = datetime.fromisoformat(date_to.replace("Z", "+00:00"))
            except ValueError:
                dt = datetime.fromisoformat(date_to + "T23:59:59+00:00")
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            stmt = stmt.where(Order.created_at <= dt)
        result = await db.execute(stmt)
        return [row[0] for row in result.all()]


