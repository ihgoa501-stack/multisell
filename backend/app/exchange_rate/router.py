"""汇率管理 API"""
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.exchange_rate.schemas import ExchangeRateCreate, ExchangeRateUpdate, ExchangeRateVO
from app.exchange_rate.service import ExchangeRateService
from app.operation_log.service import OperationLogService

router = APIRouter(prefix="/exchange-rates", tags=["汇率管理"])


@router.get("", summary="汇率列表")
async def list_rates(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:view")),
):
    rows = await ExchangeRateService.list_all(db)
    return Result.ok([ExchangeRateVO.model_validate(r).model_dump() for r in rows])


@router.put("/{from_currency}/{to_currency}", summary="创建或更新汇率")
async def upsert_rate(
    from_currency: str, to_currency: str,
    body: ExchangeRateUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:update")),
):
    row = await ExchangeRateService.upsert(db, from_currency, to_currency, body.rate)
    await OperationLogService.log(db, "exchange_rate", "upsert", str(row.id),
                                  f"更新汇率 {from_currency}/{to_currency} = {body.rate}", current_user.username)
    return Result.ok(ExchangeRateVO.model_validate(row).model_dump())


@router.post("", summary="创建汇率（完整字段）")
async def create_rate(
    body: ExchangeRateCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:update")),
):
    row = await ExchangeRateService.upsert(db, body.from_currency, body.to_currency, body.rate)
    await OperationLogService.log(db, "exchange_rate", "create", str(row.id),
                                  f"创建汇率 {body.from_currency}/{body.to_currency} = {body.rate}", current_user.username)
    return Result.ok(ExchangeRateVO.model_validate(row).model_dump())


@router.delete("/{rate_id}", summary="删除汇率")
async def delete_rate(
    rate_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:update")),
):
    ok = await ExchangeRateService.delete(db, rate_id)
    if ok:
        await OperationLogService.log(db, "exchange_rate", "delete", str(rate_id), "删除汇率", current_user.username)
    return Result.ok(message="删除成功") if ok else Result.not_found("汇率不存在")
