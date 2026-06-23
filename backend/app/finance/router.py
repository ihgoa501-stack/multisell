"""财务管理 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result, PageResult
from app.database import get_db
from app.models import User
from app.finance.schemas import (
    FinanceAccountCreate, FinanceTransactionCreate,
)
from app.finance.ledger_service import LedgerService
from app.finance.service import FinanceService

router = APIRouter(tags=["财务管理"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


# ── 账户管理 ────────────────────────────────────────────────────


@router.post("/finance/accounts", summary="创建财务账户")
async def create_account(
    data: FinanceAccountCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:manage")),
):
    account = await FinanceService.create_account(db, data.model_dump())
    return Result.ok({"id": account.id, "name": account.name})


@router.get("/finance/accounts", summary="账户列表")
async def list_accounts(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:view")),
):
    accounts = await FinanceService.list_accounts(db)
    return Result.ok(accounts)


# ── 财务流水 ────────────────────────────────────────────────────


@router.post("/finance/transactions", summary="创建财务流水")
async def create_transaction(
    data: FinanceTransactionCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:manage")),
):
    txn = await FinanceService.create_transaction(db, data.model_dump())
    return Result.ok({
        "id": txn.id,
        "account_id": txn.account_id,
        "amount": float(txn.amount),
        "transaction_type": txn.transaction_type,
    })


@router.get("/finance/transactions", summary="流水列表")
async def list_transactions(
    account_id: int = Query(None),
    transaction_type: str = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:view")),
):
    rows, total = await FinanceService.list_transactions(
        db, account_id, transaction_type, page, page_size
    )
    return PageResult.ok(records=rows, total=total, page=page, page_size=page_size)


# ── 利润汇总 ────────────────────────────────────────────────────


@router.get("/finance/profit-summary", summary="利润汇总报表")
async def get_profit_summary(
    period_start: str = Query(None, description="开始日期 ISO"),
    period_end: str = Query(None, description="结束日期 ISO"),
    platform_id: int = Query(None),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:view")),
):
    summary = await FinanceService.get_profit_summary(
        db, period_start, period_end, platform_id
    )
    return Result.ok(summary)


# ── 订单利润账本 ────────────────────────────────────────────────────


@router.post("/finance/orders/{order_id}/ledger/rebuild", summary="重建订单利润账本")
async def rebuild_order_ledger(
    order_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:ledger:rebuild")),
):
    try:
        result = await LedgerService.rebuild(db, order_id, _operator(current_user))
        return Result.ok(result)
    except ValueError as e:
        return Result.not_found(str(e))


@router.get("/finance/orders/{order_id}/ledger", summary="订单利润账本明细")
async def get_order_ledger(
    order_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:ledger:view")),
):
    try:
        result = await LedgerService.get_ledger(db, order_id)
        return Result.ok(result)
    except ValueError as e:
        return Result.not_found(str(e))


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
        return Result.not_found(str(e))


# ── 模拟数据 ────────────────────────────────────────────────────


@router.post("/finance/mock", summary="生成模拟财务数据")
async def generate_mock_finance(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("finance:manage")),
):
    accounts = await FinanceService.generate_mock_data(db)
    return Result.ok({
        "accounts_created": len(accounts),
        "message": "模拟财务数据生成成功",
    })
