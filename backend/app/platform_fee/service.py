"""平台费用 - 服务层"""

from decimal import Decimal
from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import PlatformFeeRule
from app.platform_fee.schemas import (
    PlatformFeeCalculateRequest,
    PlatformFeeCalculateItem,
    PlatformFeeCalculateResponse,
)


class PlatformFeeService:

    TYPE_LABELS = {
        "commission": "佣金",
        "fixed": "固定费用",
        "payment": "支付手续费",
        "storage": "仓储费",
        "other": "其他",
    }

    @staticmethod
    async def list_rules(
        db: AsyncSession,
        platform_id: Optional[int] = None,
        country_code: Optional[str] = None,
        fee_type: Optional[str] = None,
        status: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[PlatformFeeRule], int]:
        stmt = select(PlatformFeeRule)

        if platform_id is not None:
            stmt = stmt.where(PlatformFeeRule.platform_id == platform_id)
        if country_code is not None:
            stmt = stmt.where(PlatformFeeRule.country_code == country_code)
        if fee_type is not None:
            stmt = stmt.where(PlatformFeeRule.fee_type == fee_type)
        if status is not None:
            stmt = stmt.where(PlatformFeeRule.status == status)

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(
            PlatformFeeRule.platform_id,
            PlatformFeeRule.priority,
            PlatformFeeRule.id,
        ).offset(offset).limit(page_size)

        result = await db.execute(stmt)
        rules = list(result.scalars().all())
        return rules, total

    @staticmethod
    async def get_rule(db: AsyncSession, rule_id: int) -> Optional[PlatformFeeRule]:
        stmt = select(PlatformFeeRule).where(PlatformFeeRule.id == rule_id)
        result = await db.execute(stmt)
        return result.scalar_one_or_none()

    @staticmethod
    async def create_rule(db: AsyncSession, data: dict) -> PlatformFeeRule:
        rule = PlatformFeeRule(**data)
        db.add(rule)
        await db.flush()
        await db.refresh(rule)
        return rule

    @staticmethod
    async def update_rule(db: AsyncSession, rule: PlatformFeeRule, data: dict) -> PlatformFeeRule:
        for key, value in data.items():
            if value is not None:
                setattr(rule, key, value)
        await db.flush()
        await db.refresh(rule)
        return rule

    @staticmethod
    async def delete_rule(db: AsyncSession, rule: PlatformFeeRule) -> None:
        await db.delete(rule)
        await db.flush()

    @staticmethod
    def _calculate_rule_amount(rule: PlatformFeeRule, sale_price: Decimal) -> Decimal:
        amount = Decimal("0")
        if rule.fee_rate_pct and rule.fee_rate_pct > 0:
            amount = sale_price * rule.fee_rate_pct / Decimal("100")
        if rule.fixed_amount and rule.fixed_amount > 0:
            amount += rule.fixed_amount
        if rule.min_amount is not None and amount < rule.min_amount:
            amount = rule.min_amount
        if rule.max_amount is not None and amount > rule.max_amount:
            amount = rule.max_amount
        return amount

    @staticmethod
    def _describe_rule(rule: PlatformFeeRule) -> str:
        label = PlatformFeeService.TYPE_LABELS.get(rule.fee_type, rule.fee_type)
        parts = [label]
        if rule.fee_rate_pct and rule.fee_rate_pct > 0:
            parts.append(f"费率{rule.fee_rate_pct}%")
        if rule.fixed_amount and rule.fixed_amount > 0:
            parts.append(f"固定{rule.fixed_amount}{rule.currency}")
        if rule.remark:
            parts.append(f"({rule.remark})")
        return " ".join(parts)

    @staticmethod
    async def calculate_fee(
        db: AsyncSession,
        req: PlatformFeeCalculateRequest,
    ) -> PlatformFeeCalculateResponse:
        sale_price_dec = Decimal(str(req.sale_price))

        candidate_sets = [
            (req.platform_id, req.country_code, req.category_id),
            (req.platform_id, req.country_code, None),
            (req.platform_id, None, None),
        ]

        matched_rule_ids = set()
        matched_rules = []

        for pid, cc, cid in candidate_sets:
            stmt = select(PlatformFeeRule).where(
                PlatformFeeRule.platform_id == pid,
                PlatformFeeRule.status == "active",
            )
            if cc is not None:
                stmt = stmt.where(PlatformFeeRule.country_code == cc)
            else:
                stmt = stmt.where(PlatformFeeRule.country_code.is_(None))
            if cid is not None:
                stmt = stmt.where(PlatformFeeRule.category_id == cid)
            else:
                stmt = stmt.where(PlatformFeeRule.category_id.is_(None))

            stmt = stmt.order_by(PlatformFeeRule.priority, PlatformFeeRule.id)
            result = await db.execute(stmt)
            candidates = list(result.scalars().all())

            for r in candidates:
                if r.id not in matched_rule_ids:
                    matched_rule_ids.add(r.id)
                    matched_rules.append(r)

        items = []
        total_fee = Decimal("0")

        for rule in matched_rules:
            amount = PlatformFeeService._calculate_rule_amount(rule, sale_price_dec)
            items.append(PlatformFeeCalculateItem(
                fee_type=rule.fee_type,
                rule_id=rule.id,
                description=PlatformFeeService._describe_rule(rule),
                amount=float(amount),
            ))
            total_fee += amount

        return PlatformFeeCalculateResponse(
            platform_id=req.platform_id,
            country_code=req.country_code,
            category_id=req.category_id,
            sale_price=req.sale_price,
            total_fee=float(total_fee),
            items=items,
            rules_matched=len(matched_rules),
        )
