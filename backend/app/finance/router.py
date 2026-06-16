"""订单利润账本 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import get_current_user, require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.finance.ledger_service import LedgerService
from app.finance.reports_service import ReportService

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


# ── Reports ────────────────────────────────────────────────────────────


@router.get("/finance/reports/profit-summary", summary="利润摘要")
async def report_profit_summary(
    date_from: str | None = None,
    date_to: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:report:view")),
):
    result = await ReportService.profit_summary(db, date_from=date_from, date_to=date_to)
    return Result.ok(result)


@router.get("/finance/reports/order-profit", summary="订单利润列表")
async def report_order_profit(
    date_from: str | None = None,
    date_to: str | None = None,
    page: int = 1,
    page_size: int = 20,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:report:view")),
):
    result = await ReportService.order_profit(db, date_from=date_from, date_to=date_to, page=page, page_size=page_size)
    return Result.ok(result)


@router.get("/finance/reports/cost-variance", summary="运费差异列表")
async def report_cost_variance(
    date_from: str | None = None,
    date_to: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:report:view")),
):
    result = await ReportService.cost_variance(db, date_from=date_from, date_to=date_to)
    return Result.ok(result)


@router.get("/finance/reports/negative-profit", summary="负利润订单")
async def report_negative_profit(
    date_from: str | None = None,
    date_to: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:report:view")),
):
    result = await ReportService.negative_profit(db, date_from=date_from, date_to=date_to)
    return Result.ok(result)


@router.get("/finance/reports/cost-layer-mix", summary="成本层分布")
async def report_cost_layer_mix(
    date_from: str | None = None,
    date_to: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:report:view")),
):
    result = await ReportService.cost_layer_mix(db, date_from=date_from, date_to=date_to)
    return Result.ok(result)
