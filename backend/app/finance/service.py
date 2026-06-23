"""财务管理 - 服务层

提供账户管理、财务流水、利润汇总报表。
"""

import logging
from datetime import datetime
from decimal import Decimal
from typing import Optional

from sqlalchemy import select, func, and_
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    FinanceAccount,
    FinanceTransaction,
    Order,
    Platform,
)

logger = logging.getLogger(__name__)


class FinanceService:
    @staticmethod
    async def create_account(db: AsyncSession, data: dict) -> FinanceAccount:
        account = FinanceAccount(**data)
        db.add(account)
        await db.flush()
        await db.refresh(account)
        return account

    @staticmethod
    async def list_accounts(db: AsyncSession) -> list[dict]:
        result = await db.execute(select(FinanceAccount).order_by(FinanceAccount.id))
        accounts = result.scalars().all()
        rows = []
        for a in accounts:
            platform_name = None
            if a.platform_id:
                p = await db.get(Platform, a.platform_id)
                platform_name = p.name if p else None
            rows.append(
                {
                    "id": a.id,
                    "name": a.name,
                    "account_type": a.account_type,
                    "platform_id": a.platform_id,
                    "platform_name": platform_name,
                    "currency": a.currency,
                    "balance": float(a.balance),
                    "status": a.status,
                }
            )
        return rows

    @staticmethod
    async def create_transaction(db: AsyncSession, data: dict) -> FinanceTransaction:
        txn = FinanceTransaction(**data)
        db.add(txn)

        # 更新账户余额
        account = await db.get(FinanceAccount, data["account_id"])
        if account:
            account.balance = float(account.balance) + float(data["amount"])
        await db.flush()
        await db.refresh(txn)
        return txn

    @staticmethod
    async def list_transactions(
        db: AsyncSession,
        account_id: Optional[int] = None,
        transaction_type: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict], int]:
        stmt = select(FinanceTransaction)
        count_stmt = select(func.count()).select_from(FinanceTransaction)

        if account_id:
            stmt = stmt.where(FinanceTransaction.account_id == account_id)
            count_stmt = count_stmt.where(FinanceTransaction.account_id == account_id)
        if transaction_type:
            stmt = stmt.where(FinanceTransaction.transaction_type == transaction_type)
            count_stmt = count_stmt.where(
                FinanceTransaction.transaction_type == transaction_type
            )

        total = (await db.execute(count_stmt)).scalar() or 0
        offset = (page - 1) * page_size
        stmt = (
            stmt.order_by(FinanceTransaction.created_at.desc())
            .offset(offset)
            .limit(page_size)
        )
        result = await db.execute(stmt)
        txns = result.scalars().all()

        rows = []
        for t in txns:
            account = await db.get(FinanceAccount, t.account_id)
            platform_name = None
            if t.platform_id:
                p = await db.get(Platform, t.platform_id)
                platform_name = p.name if p else None
            rows.append(
                {
                    "id": t.id,
                    "account_id": t.account_id,
                    "account_name": account.name if account else None,
                    "transaction_type": t.transaction_type,
                    "amount": float(t.amount),
                    "currency": t.currency,
                    "order_id": t.order_id,
                    "settlement_id": t.settlement_id,
                    "platform_id": t.platform_id,
                    "platform_name": platform_name,
                    "description": t.description,
                    "transaction_date": t.transaction_date.isoformat()
                    if t.transaction_date
                    else None,
                    "created_at": t.created_at.isoformat() if t.created_at else None,
                }
            )
        return rows, total

    @staticmethod
    async def get_profit_summary(
        db: AsyncSession,
        period_start: Optional[str] = None,
        period_end: Optional[str] = None,
        platform_id: Optional[int] = None,
    ) -> dict:
        """利润汇总报表"""
        stmt = select(Order)
        conditions = [Order.status.in_(["paid", "shipped", "delivered", "completed"])]

        if period_start:
            conditions.append(Order.paid_at >= datetime.fromisoformat(period_start))
        if period_end:
            conditions.append(Order.paid_at <= datetime.fromisoformat(period_end))

        stmt = stmt.where(and_(*conditions)).order_by(Order.paid_at)

        result = await db.execute(stmt)
        orders = result.scalars().all()

        total_revenue = Decimal("0")
        total_cost = Decimal("0")
        total_shipping = Decimal("0")
        total_platform_fee = Decimal("0")
        total_payment_fee = Decimal("0")
        total_other = Decimal("0")

        platform_map: dict[int, dict] = {}

        for o in orders:
            rev = Decimal(str(o.pay_amount or 0))
            cost = Decimal(str(o.product_cost or 0))
            ship = Decimal(str(o.shipping_fee or 0))
            pf = Decimal(str(o.platform_fee or 0))
            payf = Decimal(str(o.payment_fee or 0))
            other = Decimal(str(o.other_fee or 0))

            total_revenue += rev
            total_cost += cost
            total_shipping += ship
            total_platform_fee += pf
            total_payment_fee += payf
            total_other += other

            # 按平台分组（从订单中无法直接获取平台ID，展示为整体）
            key = 0
            if key not in platform_map:
                platform_map[key] = {
                    "platform_name": "全部平台",
                    "order_count": 0,
                    "revenue": Decimal("0"),
                    "cost": Decimal("0"),
                    "profit": Decimal("0"),
                }
            pm = platform_map[key]
            pm["order_count"] += 1
            pm["revenue"] += rev
            pm["cost"] += cost + ship + pf + payf + other

        total_profit = (
            total_revenue
            - total_cost
            - total_shipping
            - total_platform_fee
            - total_payment_fee
            - total_other
        )
        margin = float(total_profit / total_revenue * 100) if total_revenue > 0 else 0

        platform_breakdown = []
        for _, pm in platform_map.items():
            p_profit = pm["revenue"] - pm["cost"]
            platform_breakdown.append(
                {
                    "platform_name": pm["platform_name"],
                    "order_count": pm["order_count"],
                    "revenue": float(pm["revenue"]),
                    "cost": float(pm["cost"]),
                    "profit": float(p_profit),
                    "margin": float(p_profit / pm["revenue"] * 100)
                    if pm["revenue"] > 0
                    else 0,
                }
            )

        return {
            "total_revenue": float(total_revenue),
            "total_product_cost": float(total_cost),
            "total_shipping_fee": float(total_shipping),
            "total_platform_fee": float(total_platform_fee),
            "total_payment_fee": float(total_payment_fee),
            "total_other_fee": float(total_other),
            "total_profit": float(total_profit),
            "profit_margin": round(margin, 2),
            "order_count": len(orders),
            "platform_breakdown": platform_breakdown,
        }

    @staticmethod
    async def generate_mock_data(db: AsyncSession) -> list[FinanceAccount]:
        """生成模拟财务账户"""
        mock_accounts = [
            ("Ozon 收款账户", "platform", "ozon", "RUB"),
            ("Shopee 收款账户", "platform", "shopee", "USD"),
            ("Wildberries 收款账户", "platform", "wb", "RUB"),
            ("银行账户-人民币", "bank", None, "CNY"),
            ("支付宝", "payment", None, "CNY"),
        ]

        accounts = []
        for name, atype, pcode, currency in mock_accounts:
            pid = None
            if pcode:
                result = await db.execute(
                    select(Platform).where(Platform.code == pcode)
                )
                p = result.scalar_one_or_none()
                pid = p.id if p else None

            existing = await db.execute(
                select(FinanceAccount).where(
                    FinanceAccount.name == name,
                    FinanceAccount.account_type == atype,
                )
            )
            if not existing.scalar_one_or_none():
                account = FinanceAccount(
                    name=name,
                    account_type=atype,
                    platform_id=pid,
                    currency=currency,
                    balance=100000.00,
                )
                db.add(account)
                await db.flush()
                accounts.append(account)

        return accounts
