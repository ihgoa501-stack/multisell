"""价格管理 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result
from app.price.schemas import PriceCreate, PriceBatchCreate, PriceVO, PriceChangeLogVO
from app.price.service import PriceService

router = APIRouter(tags=["价格管理"])


def price_to_vo(p) -> PriceVO:
    return PriceVO(
        id=p.id,
        sku_id=p.sku_id,
        price_type=p.price_type,
        price=float(p.price),
        start_time=p.start_time,
        end_time=p.end_time,
        status=p.status,
        created_at=p.created_at,
        updated_at=p.updated_at,
    )


def log_to_vo(log) -> PriceChangeLogVO:
    return PriceChangeLogVO(
        id=log.id,
        sku_id=log.sku_id,
        old_price=float(log.old_price) if log.old_price else None,
        new_price=float(log.new_price) if log.new_price else None,
        price_type=log.price_type,
        change_type=log.change_type,
        operator=log.operator,
        remark=log.remark,
        created_at=log.created_at,
    )


@router.post("/prices", summary="设置价格")
async def set_price(data: PriceCreate, db: AsyncSession = Depends(get_db)):
    p = await PriceService.set_price(db, data.sku_id, data.price_type, data.price, data.start_time, data.end_time)
    return Result.ok(price_to_vo(p))


@router.post("/prices/batch", summary="批量调价")
async def batch_set_price(data: PriceBatchCreate, db: AsyncSession = Depends(get_db)):
    count = await PriceService.batch_set_price(
        db, data.sku_ids, data.price_type, data.price, data.start_time, data.end_time
    )
    return Result.ok({"affected_count": count})


@router.get("/skus/{sku_id}/prices", summary="获取SKU所有价格")
async def get_prices(sku_id: int, db: AsyncSession = Depends(get_db)):
    prices = await PriceService.get_prices_by_sku(db, sku_id)
    return Result.ok([price_to_vo(p) for p in prices])


@router.get("/skus/{sku_id}/current-price", summary="获取当前售价")
async def get_current_price(sku_id: int, db: AsyncSession = Depends(get_db)):
    price = await PriceService.get_current_price(db, sku_id)
    if not price:
        return Result.ok({"sku_id": sku_id, "price": None, "price_type": None})
    return Result.ok({"sku_id": sku_id, "price": float(price.price), "price_type": price.price_type})


@router.get("/skus/{sku_id}/price-history", summary="调价历史")
async def get_price_history(sku_id: int, db: AsyncSession = Depends(get_db)):
    logs = await PriceService.get_price_history(db, sku_id)
    return Result.ok([log_to_vo(log) for log in logs])
