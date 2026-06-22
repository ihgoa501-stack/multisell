"""售后管理 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import PageResult, Result
from app.database import get_db
from app.models import User
from app.aftersales.schemas import (
    AfterSalesApprove,
    AfterSalesCreate,
    AfterSalesReceive,
    AfterSalesReject,
    AfterSalesRefund,
)
from app.aftersales.service import AfterSalesService

router = APIRouter(tags=["售后管理"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.post("/aftersales", summary="创建售后单")
async def create_return(
    data: AfterSalesCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:create")),
):
    try:
        after = await AfterSalesService.create_return(db, data, _operator(current_user))
    except ValueError as e:
        return Result.bad_request(str(e))
    return Result.ok(after)


@router.get("/aftersales", summary="售后单列表")
async def list_returns(
    status: str = Query(None, description="售后单状态"),
    order_id: int = Query(None, description="订单ID"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:view")),
):
    rows, total = await AfterSalesService.list_returns(db, status, order_id, page, page_size)
    return PageResult.ok(rows, total, page, page_size)


@router.get("/aftersales/{after_id}", summary="售后单详情")
async def get_return(
    after_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:view")),
):
    after = await AfterSalesService.get_return(db, after_id)
    if not after:
        return Result.not_found("售后单不存在")
    return Result.ok(after)


@router.post("/aftersales/{after_id}/approve", summary="审批售后单")
async def approve_return(
    after_id: int,
    data: AfterSalesApprove,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:approve")),
):
    try:
        after = await AfterSalesService.approve(db, after_id, data, _operator(current_user))
    except ValueError as e:
        return Result.bad_request(str(e))
    if not after:
        return Result.not_found("售后单不存在")
    return Result.ok(after)


@router.post("/aftersales/{after_id}/reject", summary="驳回售后单")
async def reject_return(
    after_id: int,
    data: AfterSalesReject,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:approve")),
):
    try:
        after = await AfterSalesService.reject(db, after_id, data, _operator(current_user))
    except ValueError as e:
        return Result.bad_request(str(e))
    if not after:
        return Result.not_found("售后单不存在")
    return Result.ok(after)


@router.post("/aftersales/{after_id}/receive", summary="收货入库")
async def receive_return(
    after_id: int,
    data: AfterSalesReceive,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:operate")),
):
    try:
        after = await AfterSalesService.receive_return(db, after_id, data, _operator(current_user))
    except ValueError as e:
        return Result.bad_request(str(e))
    if not after:
        return Result.not_found("售后单不存在")
    return Result.ok(after)


@router.post("/aftersales/{after_id}/refund", summary="完成退款")
async def complete_refund(
    after_id: int,
    data: AfterSalesRefund,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:operate")),
):
    try:
        after = await AfterSalesService.complete_refund(db, after_id, data, _operator(current_user))
    except ValueError as e:
        return Result.bad_request(str(e))
    if not after:
        return Result.not_found("售后单不存在")
    return Result.ok(after)
