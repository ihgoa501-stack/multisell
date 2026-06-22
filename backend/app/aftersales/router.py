"""售后退货 — router"""
from typing import Optional
from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.aftersales.schemas import (
    AfterSalesCreate, AfterSalesApprove, AfterSalesReject,
    AfterSalesReceive, AfterSalesRefund, AfterSalesVO,
)
from app.aftersales.service import AfterSalesService

router = APIRouter(prefix="/aftersales", tags=["售后退货"])


def _to_vo(rma) -> AfterSalesVO:
    return AfterSalesVO(
        id=rma.id,
        order_id=rma.order_id,
        item_id=rma.item_id,
        sku_id=rma.sku_id,
        return_quantity=rma.return_quantity,
        reason=rma.reason,
        status=rma.status,
        refund_amount=float(rma.refund_amount) if rma.refund_amount else None,
        inspection_result=rma.inspection_result,
        rejection_reason=rma.rejection_reason,
        created_by=rma.created_by,
        approved_by=rma.approved_by,
        approved_at=rma.approved_at,
        rejected_by=rma.rejected_by,
        rejected_at=rma.rejected_at,
        received_by=rma.received_by,
        received_at=rma.received_at,
        refunded_by=rma.refunded_by,
        refunded_at=rma.refunded_at,
        created_at=rma.created_at,
        updated_at=rma.updated_at,
    )


@router.post("", summary="创建退货申请")
async def create_rma(
    data: AfterSalesCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:create")),
):
    rma = await AfterSalesService.create(
        db, data.model_dump(), created_by=current_user.username,
    )
    return Result.ok(_to_vo(rma))


@router.post("/{rma_id}/approve", summary="审批通过")
async def approve_rma(
    rma_id: int,
    data: AfterSalesApprove,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:approve")),
):
    rma = await AfterSalesService.approve(db, rma_id, current_user.username, data.refund_amount)
    if not rma:
        return Result.bad_request("退货单不存在或状态不允许审批")
    return Result.ok(_to_vo(rma))


@router.post("/{rma_id}/reject", summary="驳回退货")
async def reject_rma(
    rma_id: int,
    data: AfterSalesReject,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:approve")),
):
    rma = await AfterSalesService.reject(db, rma_id, current_user.username, data.rejection_reason)
    if not rma:
        return Result.bad_request("退货单不存在或状态不允许驳回")
    return Result.ok(_to_vo(rma))


@router.post("/{rma_id}/receive", summary="入库验收")
async def receive_rma(
    rma_id: int,
    data: AfterSalesReceive,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:receive")),
):
    rma = await AfterSalesService.receive(db, rma_id, current_user.username, data.inspection_result)
    if not rma:
        return Result.bad_request("退货单不存在或状态不允许入库")
    return Result.ok(_to_vo(rma))


@router.post("/{rma_id}/refund", summary="确认退款")
async def refund_rma(
    rma_id: int,
    data: AfterSalesRefund,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("aftersales:refund")),
):
    rma = await AfterSalesService.refund(db, rma_id, current_user.username)
    if not rma:
        return Result.bad_request("退货单不存在或状态不允许退款")
    return Result.ok(_to_vo(rma))


@router.get("/{rma_id}", summary="退货单详情")
async def get_rma(
    rma_id: int,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("aftersales:view")),
):
    rma = await AfterSalesService.get_by_id(db, rma_id)
    if not rma:
        return Result.not_found("退货单不存在")
    return Result.ok(_to_vo(rma))


@router.get("", summary="退货单列表")
async def list_rma(
    order_id: Optional[int] = Query(None),
    status: Optional[str] = Query(None),
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("aftersales:view")),
):
    if order_id:
        items = await AfterSalesService.list_by_order(db, order_id)
    else:
        items = await AfterSalesService.list_all(db, status=status)
    return Result.ok([_to_vo(r) for r in items])
