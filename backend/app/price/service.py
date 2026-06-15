"""价格管理 - 服务层"""
from datetime import datetime
from typing import Optional
from sqlalchemy import select, delete
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Price, PriceChangeLog, Sku


class PriceService:

    @staticmethod
    async def set_price(db: AsyncSession, sku_id: int, price_type: str, price_amount: float,
                        start_time: datetime = None, end_time: datetime = None) -> Price:
        """设置价格（已有则更新，没有则创建）"""
        stmt = select(Price).where(
            Price.sku_id == sku_id,
            Price.price_type == price_type,
        )
        result = await db.execute(stmt)
        existing = result.scalar_one_or_none()

        old_price_value = None
        if existing:
            old_price_value = float(existing.price)
            existing.price = price_amount
            existing.start_time = start_time
            existing.end_time = end_time
            price_obj = existing
        else:
            price_obj = Price(
                sku_id=sku_id,
                price_type=price_type,
                price=price_amount,
                start_time=start_time,
                end_time=end_time,
            )
            db.add(price_obj)

        await db.flush()
        await db.refresh(price_obj)

        # 记录调价日志
        log = PriceChangeLog(
            sku_id=sku_id,
            old_price=old_price_value,
            new_price=price_amount,
            price_type=price_type,
            change_type="manual",
            operator="system",
            remark="手动调价",
        )
        db.add(log)
        await db.flush()

        return price_obj

    @staticmethod
    async def batch_set_price(db: AsyncSession, sku_ids: list[int], price_type: str,
                              price_amount: float, start_time: datetime = None,
                              end_time: datetime = None) -> int:
        """批量设置价格，返回影响的SKU数"""
        count = 0
        for sku_id in sku_ids:
            await PriceService.set_price(db, sku_id, price_type, price_amount, start_time, end_time)
            count += 1
        return count

    @staticmethod
    async def get_prices_by_sku(db: AsyncSession, sku_id: int) -> list[Price]:
        stmt = select(Price).where(Price.sku_id == sku_id).order_by(Price.price_type)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def get_current_price(db: AsyncSession, sku_id: int) -> Optional[Price]:
        """获取当前有效销售价（按 start_time <= now < end_time 计算）"""
        now = datetime.utcnow()
        stmt = select(Price).where(
            Price.sku_id == sku_id,
            Price.price_type == "sale_price",
            Price.status == 1,
            (Price.start_time.is_(None) | (Price.start_time <= now)),
            (Price.end_time.is_(None) | (Price.end_time > now)),
        ).order_by(Price.created_at.desc()).limit(1)
        result = await db.execute(stmt)
        return result.scalar_one_or_none()

    @staticmethod
    async def get_price_history(db: AsyncSession, sku_id: int) -> list[PriceChangeLog]:
        stmt = select(PriceChangeLog).where(
            PriceChangeLog.sku_id == sku_id
        ).order_by(PriceChangeLog.created_at.desc()).limit(100)
        result = await db.execute(stmt)
        return list(result.scalars().all())
