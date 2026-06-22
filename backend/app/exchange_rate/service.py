"""汇率服务层"""
from decimal import Decimal
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import ExchangeRate


class ExchangeRateService:

    @staticmethod
    async def get_rate(db: Optional[AsyncSession], from_currency: str, to_currency: str) -> Optional[float]:
        """查询汇率，返回 float 或 None"""
        if db is None:
            return None
        stmt = select(ExchangeRate).where(
            ExchangeRate.from_currency == from_currency.upper(),
            ExchangeRate.to_currency == to_currency.upper(),
        )
        row = (await db.execute(stmt)).scalar_one_or_none()
        return float(row.rate) if row else None

    @staticmethod
    async def get_rate_or_fallback(db: Optional[AsyncSession], from_currency: str, to_currency: str, fallback: float) -> float:
        """查询汇率，查不到返回 fallback"""
        rate = await ExchangeRateService.get_rate(db, from_currency, to_currency)
        return rate if rate is not None else fallback

    @staticmethod
    async def list_all(db: AsyncSession) -> list[ExchangeRate]:
        rows = (await db.execute(select(ExchangeRate).order_by(ExchangeRate.from_currency, ExchangeRate.to_currency))).scalars().all()
        return list(rows)

    @staticmethod
    async def upsert(db: AsyncSession, from_currency: str, to_currency: str, rate: float) -> ExchangeRate:
        """创建或更新汇率"""
        from_currency = from_currency.upper()
        to_currency = to_currency.upper()
        stmt = select(ExchangeRate).where(
            ExchangeRate.from_currency == from_currency,
            ExchangeRate.to_currency == to_currency,
        )
        row = (await db.execute(stmt)).scalar_one_or_none()
        if row:
            row.rate = Decimal(str(rate))
        else:
            row = ExchangeRate(from_currency=from_currency, to_currency=to_currency, rate=Decimal(str(rate)))
            db.add(row)
        await db.flush()
        await db.refresh(row)
        return row

    @staticmethod
    async def delete(db: AsyncSession, rate_id: int) -> bool:
        row = await db.get(ExchangeRate, rate_id)
        if not row:
            return False
        await db.delete(row)
        await db.flush()
        return True
