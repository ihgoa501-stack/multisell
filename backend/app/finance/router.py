"""订单利润账本 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import get_current_user, require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.finance.ledger_service import LedgerService

router = APIRouter(tags=["财务账本"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.post("/finance/orders/{order_id}/ledger/rebuild", summary="重建订单利润账本")
async def rebuild_order_ledger(
    order_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:ledger:rebuild")),
):
    try:
        result = await LedgerService.rebuild(db, order_id, operator=_operator(current_user))
        return Result.ok(result)
    except ValueError as e:
        return Result.bad_request(str(e))


@router.get("/finance/orders/{order_id}/ledger", summary="订单账本条目")
async def get_order_ledger(
    order_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:ledger:view")),
):
    try:
        result = await LedgerService.get_ledger(db, order_id)
        return Result.ok(result)
    except ValueError as e:
        return Result.bad_request(str(e))


@router.get("/finance/orders/{order_id}/profit", summary="订单利润账本汇总")
async def get_order_profit(
    order_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:ledger:view")),
):
    try:
        result = await LedgerService.get_profit(db, order_id)
        return Result.ok(result)
    except ValueError as e:
        return Result.bad_request(str(e))
