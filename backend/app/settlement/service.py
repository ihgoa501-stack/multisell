"""结算管理 - 服务层"""

import logging
from datetime import datetime, timezone
from decimal import Decimal
from typing import Optional

from sqlalchemy import select, func, and_, delete as sa_delete, case
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models import Settlement, SettlementItem, Order, Platform

logger = logging.getLogger(__name__)


class SettlementService:

    # ── 导入 ────────────────────────────────────────────────────────

    @staticmethod
    async def import_settlement(
        db: AsyncSession,
        data: dict,
        items_data: list[dict],
    ) -> Settlement:
        """导入结算单及其明细"""
        settlement = Settlement(**data)
        db.add(settlement)
        await db.flush()

        for item_data in items_data:
            item = SettlementItem(settlement_id=settlement.id, **item_data)
            db.add(item)

        await db.flush()
        await db.refresh(settlement)

        # 重新统计汇总金额
        await SettlementService._recalc_totals(db, settlement.id)

        return settlement

    @staticmethod
    async def _recalc_totals(db: AsyncSession, settlement_id: int):
        """根据明细重新计算结算单汇总"""
        stmt = select(
            func.coalesce(func.sum(SettlementItem.amount), 0),
            func.coalesce(func.sum(SettlementItem.fee), 0),
            func.coalesce(func.sum(SettlementItem.net), 0),
        ).where(SettlementItem.settlement_id == settlement_id)

        row = (await db.execute(stmt)).one()
        settlement = await db.get(Settlement, settlement_id)
        if settlement:
            settlement.total_revenue = float(row[0])
            settlement.total_fee = float(row[1])
            settlement.total_net = float(row[2])
            await db.flush()

    # ── 查询 ────────────────────────────────────────────────────────

    @staticmethod
    async def list_settlements(
        db: AsyncSession,
        platform_id: Optional[int] = None,
        status: Optional[str] = None,
        keyword: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict], int]:
        """分页查询结算单列表（含汇总统计）"""
        stmt = select(Settlement)
        count_stmt = select(func.count()).select_from(Settlement)

        if platform_id:
            stmt = stmt.where(Settlement.platform_id == platform_id)
            count_stmt = count_stmt.where(Settlement.platform_id == platform_id)
        if status:
            stmt = stmt.where(Settlement.status == status)
            count_stmt = count_stmt.where(Settlement.status == status)
        if keyword:
            like = f"%{keyword}%"
            stmt = stmt.where(Settlement.settlement_no.ilike(like))
            count_stmt = count_stmt.where(Settlement.settlement_no.ilike(like))

        total = (await db.execute(count_stmt)).scalar() or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(Settlement.created_at.desc()).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        settlements = result.scalars().all()

        rows = []
        for s in settlements:
            platform_name = None
            if s.platform:
                platform_name = s.platform.name

            # 统计对账状态
            item_counts = await SettlementService._item_status_counts(db, s.id)

            rows.append({
                "id": s.id,
                "platform_id": s.platform_id,
                "platform_name": platform_name,
                "settlement_no": s.settlement_no,
                "period_start": s.period_start.isoformat() if s.period_start else None,
                "period_end": s.period_end.isoformat() if s.period_end else None,
                "currency": s.currency,
                "total_revenue": float(s.total_revenue),
                "total_fee": float(s.total_fee),
                "total_refund": float(s.total_refund),
                "total_net": float(s.total_net),
                "status": s.status,
                "item_count": item_counts["total"],
                "matched_count": item_counts["matched"],
                "unmatched_count": item_counts["unmatched"],
                "discrepancy_count": item_counts["discrepancy"],
                "imported_at": s.imported_at.isoformat() if s.imported_at else None,
                "created_at": s.created_at.isoformat() if s.created_at else None,
                "updated_at": s.updated_at.isoformat() if s.updated_at else None,
            })

        return rows, total

    @staticmethod
    async def _item_status_counts(db: AsyncSession, settlement_id: int) -> dict:
        """获取明细对账状态统计"""
        stmt = select(
            func.count().label("total"),
            func.sum(case((SettlementItem.reconciliation_status == "matched", 1), else_=0)).label("matched"),
            func.sum(case((SettlementItem.reconciliation_status == "unmatched", 1), else_=0)).label("unmatched"),
            func.sum(case((SettlementItem.reconciliation_status == "discrepancy", 1), else_=0)).label("discrepancy"),
        ).where(SettlementItem.settlement_id == settlement_id)

        row = (await db.execute(stmt)).one()
        return {
            "total": row[0] or 0,
            "matched": row[1] or 0,
            "unmatched": row[2] or 0,
            "discrepancy": row[3] or 0,
        }

    @staticmethod
    async def get_settlement_detail(db: AsyncSession, settlement_id: int) -> Optional[dict]:
        """获取结算单详情"""
        settlement = await db.get(Settlement, settlement_id)
        if not settlement:
            return None

        platform_name = settlement.platform.name if settlement.platform else None
        item_counts = await SettlementService._item_status_counts(db, settlement_id)

        return {
            "id": settlement.id,
            "platform_id": settlement.platform_id,
            "platform_name": platform_name,
            "settlement_no": settlement.settlement_no,
            "period_start": settlement.period_start.isoformat() if settlement.period_start else None,
            "period_end": settlement.period_end.isoformat() if settlement.period_end else None,
            "currency": settlement.currency,
            "total_revenue": float(settlement.total_revenue),
            "total_fee": float(settlement.total_fee),
            "total_refund": float(settlement.total_refund),
            "total_net": float(settlement.total_net),
            "status": settlement.status,
            "item_count": item_counts["total"],
            "matched_count": item_counts["matched"],
            "unmatched_count": item_counts["unmatched"],
            "discrepancy_count": item_counts["discrepancy"],
            "imported_at": settlement.imported_at.isoformat() if settlement.imported_at else None,
            "created_at": settlement.created_at.isoformat() if settlement.created_at else None,
            "updated_at": settlement.updated_at.isoformat() if settlement.updated_at else None,
        }

    # ── 明细 ────────────────────────────────────────────────────────

    @staticmethod
    async def list_items(
        db: AsyncSession,
        settlement_id: int,
        reconciliation_status: Optional[str] = None,
        transaction_type: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict], int]:
        """分页查询结算明细"""
        stmt = select(SettlementItem).where(SettlementItem.settlement_id == settlement_id)
        count_stmt = select(func.count()).select_from(SettlementItem).where(
            SettlementItem.settlement_id == settlement_id
        )

        if reconciliation_status:
            stmt = stmt.where(SettlementItem.reconciliation_status == reconciliation_status)
            count_stmt = count_stmt.where(SettlementItem.reconciliation_status == reconciliation_status)
        if transaction_type:
            stmt = stmt.where(SettlementItem.transaction_type == transaction_type)
            count_stmt = count_stmt.where(SettlementItem.transaction_type == transaction_type)

        total = (await db.execute(count_stmt)).scalar() or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(SettlementItem.occurred_at.desc().nulls_last()).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        items = result.scalars().all()

        rows = []
        for item in items:
            rows.append({
                "id": item.id,
                "settlement_id": item.settlement_id,
                "transaction_type": item.transaction_type,
                "transaction_id": item.transaction_id,
                "order_no": item.order_no,
                "order_id": item.order_id,
                "sku_id": item.sku_id,
                "amount": float(item.amount),
                "fee": float(item.fee),
                "net": float(item.net),
                "quantity": item.quantity,
                "occurred_at": item.occurred_at.isoformat() if item.occurred_at else None,
                "created_at": item.created_at.isoformat() if item.created_at else None,
                "reconciliation_status": item.reconciliation_status,
                "reconciliation_note": item.reconciliation_note,
                "reconciled_at": item.reconciled_at.isoformat() if item.reconciled_at else None,
                "reconciled_by": item.reconciled_by,
            })

        return rows, total

    # ── 对账 ────────────────────────────────────────────────────────

    @staticmethod
    async def reconcile(
        db: AsyncSession,
        settlement_id: int,
        auto_match: bool = True,
        strategy: str = "by_order_no",
    ) -> dict:
        """执行对账

        - auto_match: 是否自动按策略匹配
        - strategy: by_order_no（按订单号匹配）/ by_transaction_id（按交易ID匹配）
        """
        settlement = await db.get(Settlement, settlement_id)
        if not settlement:
            raise ValueError("结算单不存在")

        # 更新结算单状态
        settlement.status = "reconciling"
        await db.flush()

        # 获取所有待对账的明细
        stmt = select(SettlementItem).where(
            SettlementItem.settlement_id == settlement_id,
            SettlementItem.reconciliation_status.in_(["pending", "unmatched"]),
        )
        result = await db.execute(stmt)
        items = result.scalars().all()

        matched = 0
        unmatched = 0

        now = datetime.now(timezone.utc)

        for item in items:
            if not auto_match:
                # 仅标记为未对账
                item.reconciliation_status = "unmatched"
                unmatched += 1
                continue

            if strategy == "by_order_no" and item.order_no:
                # 按订单号匹配
                order_stmt = select(Order).where(Order.order_no == item.order_no)
                order = (await db.execute(order_stmt)).scalar_one_or_none()
                if order:
                    item.order_id = order.id
                    item.reconciliation_status = "matched"

                    # 检查金额是否一致
                    expected_amount = float(order.pay_amount or 0)
                    actual_amount = float(item.amount)
                    if abs(expected_amount - actual_amount) > 0.01:
                        item.reconciliation_status = "discrepancy"
                        item.reconciliation_note = (
                            f"金额不一致: 内部订单 {expected_amount} vs 平台结算 {actual_amount}"
                        )

                    matched += 1
                else:
                    item.reconciliation_status = "unmatched"
                    unmatched += 1

            elif strategy == "by_transaction_id" and item.transaction_id:
                # 按交易ID匹配 — 宽松匹配
                order_stmt = select(Order).where(
                    Order.remark.ilike(f"%{item.transaction_id}%")
                )
                order = (await db.execute(order_stmt)).scalar_one_or_none()
                if order:
                    item.order_id = order.id
                    item.order_no = order.order_no
                    item.reconciliation_status = "matched"
                    matched += 1
                else:
                    item.reconciliation_status = "unmatched"
                    unmatched += 1
            else:
                item.reconciliation_status = "unmatched"
                unmatched += 1

            item.reconciled_at = now

        await db.flush()

        # 重新统计
        item_counts = await SettlementService._item_status_counts(db, settlement_id)

        # 若全部匹配则自动标记为已对账
        if item_counts["unmatched"] == 0 and item_counts["discrepancy"] == 0 and item_counts["total"] > 0:
            settlement.status = "reconciled"
        else:
            settlement.status = "reconciling"
        await db.flush()

        return {
            "settlement_id": settlement_id,
            "matched": matched,
            "unmatched": unmatched,
            "total": len(items),
            "summary": item_counts,
        }

    @staticmethod
    async def update_item_reconciliation(
        db: AsyncSession,
        item_id: int,
        status: str,
        note: Optional[str] = None,
        reconciled_by: Optional[str] = None,
    ) -> Optional[dict]:
        """手动更新明细对账状态"""
        item = await db.get(SettlementItem, item_id)
        if not item:
            return None

        item.reconciliation_status = status
        if note is not None:
            item.reconciliation_note = note
        if reconciled_by is not None:
            item.reconciled_by = reconciled_by
        item.reconciled_at = datetime.now(timezone.utc)
        await db.flush()

        return {
            "id": item.id,
            "reconciliation_status": item.reconciliation_status,
            "reconciliation_note": item.reconciliation_note,
            "reconciled_at": item.reconciled_at.isoformat(),
        }

    # ── 删除 ────────────────────────────────────────────────────────

    @staticmethod
    async def delete_settlement(db: AsyncSession, settlement_id: int) -> bool:
        """删除结算单（级联删除明细）"""
        settlement = await db.get(Settlement, settlement_id)
        if not settlement:
            return False

        await db.delete(settlement)
        await db.flush()
        return True

    # ── 模拟数据生成 ────────────────────────────────────────────────

    @staticmethod
    async def generate_mock_data(
        db: AsyncSession,
        platform_id: int,
        count: int = 10,
    ) -> Settlement:
        """生成模拟结算数据，用于演示和测试"""
        from datetime import timedelta
        import random

        now = datetime.now(timezone.utc)
        settlement_no = f"STL-{now.strftime('%Y%m%d')}-{random.randint(1000, 9999)}"

        # 获取最近的订单作为模拟数据源
        order_stmt = select(Order).order_by(Order.created_at.desc()).limit(count)
        orders = (await db.execute(order_stmt)).scalars().all()

        settlement = Settlement(
            platform_id=platform_id,
            settlement_no=settlement_no,
            period_start=now - timedelta(days=30),
            period_end=now,
            currency="CNY",
            status="pending",
        )
        db.add(settlement)
        await db.flush()

        items_data = []
        for order in orders:
            amount = float(order.pay_amount or 0)
            fee = float(order.platform_fee or 0) + float(order.payment_fee or 0)
            items_data.append(SettlementItem(
                settlement_id=settlement.id,
                transaction_type="order_sale",
                transaction_id=f"TXN-{order.order_no}",
                order_no=order.order_no,
                order_id=order.id,
                amount=amount,
                fee=fee,
                net=amount - fee,
                quantity=1,
                occurred_at=order.paid_at or order.created_at,
            ))

        # 加入一些退款和费用
        if orders:
            refund_amount = float(orders[0].pay_amount or 0) * 0.5
            items_data.append(SettlementItem(
                settlement_id=settlement.id,
                transaction_type="refund",
                transaction_id=f"REF-{settlement_no}",
                order_no=orders[0].order_no,
                order_id=orders[0].id,
                amount=-refund_amount,
                fee=0,
                net=-refund_amount,
                quantity=1,
                occurred_at=now - timedelta(days=1),
            ))
            items_data.append(SettlementItem(
                settlement_id=settlement.id,
                transaction_type="platform_fee",
                transaction_id=f"FEE-{settlement_no}",
                amount=0,
                fee=50.00,
                net=-50.00,
                quantity=1,
                occurred_at=now - timedelta(days=1),
            ))

        for item in items_data:
            db.add(item)

        await db.flush()

        # 重算汇总
        await SettlementService._recalc_totals(db, settlement.id)
        await db.refresh(settlement)

        return settlement
