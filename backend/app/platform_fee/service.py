"""平台费用规则 - 服务层"""

from typing import Optional

from sqlalchemy import select, and_
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models import PlatformFeeRule, Platform
from app.platform_fee.schemas import (
    PlatformFeeRuleCreate,
    PlatformFeeRuleUpdate,
    PlatformFeeRuleMatchRequest,
    PlatformFeeRuleVO,
)


def _rule_to_vo(rule: PlatformFeeRule) -> dict:
    platform_name = None
    try:
        platform_name = rule.platform.name if rule.platform else None
    except Exception:
        platform_name = None
    return {
        "id": rule.id,
        "platform_id": rule.platform_id,
        "platform_name": platform_name,
        "site_code": rule.site_code,
        "category_id": rule.category_id,
        "commission_pct": float(rule.commission_pct or 0),
        "payment_fee_pct": float(rule.payment_fee_pct or 0),
        "fixed_fee": float(rule.fixed_fee or 0),
        "advertising_pct": float(rule.advertising_pct or 0),
        "other_reserve_fee": float(rule.other_reserve_fee or 0),
        "priority": rule.priority or 0,
        "status": rule.status or 0,
        "remark": rule.remark,
        "created_at": str(rule.created_at) if rule.created_at else "",
        "updated_at": str(rule.updated_at) if rule.updated_at else "",
    }


def _normalize_site_code(code: Optional[str]) -> Optional[str]:
    """空字符串转为 None，非空转大写。"""
    if code is None or code.strip() == "":
        return None
    return code.strip().upper()


class PlatformFeeRuleService:

    @staticmethod
    async def list(
        db: AsyncSession,
        platform_id: Optional[int] = None,
        site_code: Optional[str] = None,
        category_id: Optional[int] = None,
    ) -> list[dict]:
        stmt = (
            select(PlatformFeeRule)
            .options(selectinload(PlatformFeeRule.platform))
            .order_by(PlatformFeeRule.priority, PlatformFeeRule.id)
        )
        conditions = []
        if platform_id is not None:
            conditions.append(PlatformFeeRule.platform_id == platform_id)
        if site_code is not None:
            conditions.append(PlatformFeeRule.site_code == site_code)
        if category_id is not None:
            conditions.append(PlatformFeeRule.category_id == category_id)
        if conditions:
            stmt = stmt.where(and_(*conditions))
        result = await db.execute(stmt)
        rules = result.scalars().all()
        return [_rule_to_vo(r) for r in rules]

    @staticmethod
    async def get_by_id(db: AsyncSession, rule_id: int) -> Optional[dict]:
        stmt = (
            select(PlatformFeeRule)
            .options(selectinload(PlatformFeeRule.platform))
            .where(PlatformFeeRule.id == rule_id)
        )
        result = await db.execute(stmt)
        rule = result.scalar_one_or_none()
        return _rule_to_vo(rule) if rule else None

    @staticmethod
    async def create(db: AsyncSession, data: PlatformFeeRuleCreate) -> dict:
        rule = PlatformFeeRule(
            platform_id=data.platform_id,
            site_code=_normalize_site_code(data.site_code),
            category_id=data.category_id,
            commission_pct=data.commission_pct,
            payment_fee_pct=data.payment_fee_pct,
            fixed_fee=data.fixed_fee,
            advertising_pct=data.advertising_pct,
            other_reserve_fee=data.other_reserve_fee,
            priority=data.priority,
            status=data.status,
            remark=data.remark,
        )
        db.add(rule)
        await db.flush()
        await db.refresh(rule, ["platform"])
        return _rule_to_vo(rule)

    @staticmethod
    async def update(db: AsyncSession, rule_id: int, data: PlatformFeeRuleUpdate) -> Optional[dict]:
        stmt = (
            select(PlatformFeeRule)
            .options(selectinload(PlatformFeeRule.platform))
            .where(PlatformFeeRule.id == rule_id)
        )
        result = await db.execute(stmt)
        rule = result.scalar_one_or_none()
        if not rule:
            return None
        update_data = data.model_dump(exclude_unset=True)
        if "site_code" in update_data:
            update_data["site_code"] = _normalize_site_code(update_data["site_code"])
        for k, v in update_data.items():
            setattr(rule, k, v)
        await db.flush()
        await db.refresh(rule)
        # Reload with relationship for complete VO
        loaded = await db.execute(
            select(PlatformFeeRule)
            .options(selectinload(PlatformFeeRule.platform))
            .where(PlatformFeeRule.id == rule_id)
        )
        reloaded = loaded.scalar_one()
        return _rule_to_vo(reloaded)

    @staticmethod
    async def delete(db: AsyncSession, rule_id: int) -> bool:
        stmt = select(PlatformFeeRule).where(PlatformFeeRule.id == rule_id)
        result = await db.execute(stmt)
        rule = result.scalar_one_or_none()
        if not rule:
            return False
        await db.delete(rule)
        await db.flush()
        return True

    @staticmethod
    async def match(
        db: AsyncSession,
        req: PlatformFeeRuleMatchRequest,
    ) -> Optional[dict]:
        """
        按优先级匹配平台费用规则:
        1. platform_id + site_code + category_id
        2. platform_id + site_code + category_id IS NULL
        3. platform_id + site_code IS NULL + category_id IS NULL
        """
        site_code = _normalize_site_code(req.site_code)
        base_filter = and_(
            PlatformFeeRule.platform_id == req.platform_id,
            PlatformFeeRule.status == 1,
        )

        # Priority 1: exact category match
        if req.category_id is not None and site_code is not None:
            stmt = (
                select(PlatformFeeRule)
                .where(
                    and_(
                        base_filter,
                        PlatformFeeRule.site_code == site_code,
                        PlatformFeeRule.category_id == req.category_id,
                    )
                )
                .order_by(PlatformFeeRule.priority, PlatformFeeRule.id)
                .limit(1)
            )
            result = await db.execute(stmt)
            rule = result.scalar_one_or_none()
            if rule:
                loaded = await db.execute(
                    select(PlatformFeeRule)
                    .options(selectinload(PlatformFeeRule.platform))
                    .where(PlatformFeeRule.id == rule.id)
                )
                return _rule_to_vo(loaded.scalar_one())

        # Priority 2: site-level (category IS NULL)
        if site_code is not None:
            stmt = (
                select(PlatformFeeRule)
                .where(
                    and_(
                        base_filter,
                        PlatformFeeRule.site_code == site_code,
                        PlatformFeeRule.category_id.is_(None),
                    )
                )
                .order_by(PlatformFeeRule.priority, PlatformFeeRule.id)
                .limit(1)
            )
            result = await db.execute(stmt)
            rule = result.scalar_one_or_none()
            if rule:
                loaded = await db.execute(
                    select(PlatformFeeRule)
                    .options(selectinload(PlatformFeeRule.platform))
                    .where(PlatformFeeRule.id == rule.id)
                )
                return _rule_to_vo(loaded.scalar_one())

        # Priority 3: global (site_code IS NULL, category_id IS NULL)
        stmt = (
            select(PlatformFeeRule)
            .where(
                and_(
                    base_filter,
                    PlatformFeeRule.site_code.is_(None),
                    PlatformFeeRule.category_id.is_(None),
                )
            )
            .order_by(PlatformFeeRule.priority, PlatformFeeRule.id)
            .limit(1)
        )
        result = await db.execute(stmt)
        rule = result.scalar_one_or_none()
        if rule:
            loaded = await db.execute(
                select(PlatformFeeRule)
                .options(selectinload(PlatformFeeRule.platform))
                .where(PlatformFeeRule.id == rule.id)
            )
            return _rule_to_vo(loaded.scalar_one())

        return None
