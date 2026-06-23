"""订单管理 - 路由"""

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import get_current_user, require_permission
from app.common import PageResult, Result
from app.config import settings
from app.database import get_db
from app.models import Permission, Role, RolePermission, User, UserRole
from app.order.schemas import (
    OrderCreate,
    OrderProfitInputsUpdate,
    OrderShippingQuoteBind,
    OrderStatusUpdate,
)
from app.order.service import OrderService
from app.operation_log.service import OperationLogService

router = APIRouter(tags=["订单管理"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


async def _require_order_status_or_cancel(
    data: OrderStatusUpdate,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
) -> User:
    """正常状态变更要求 order:update_status，取消操作要求 order:cancel。"""
    if not settings.AUTH_ENABLED or current_user.role == "admin":
        return current_user
    needed = "order:cancel" if data.status == "cancelled" else "order:update_status"
    stmt = (
        select(Permission.id)
        .join(RolePermission, RolePermission.permission_id == Permission.id)
        .join(Role, Role.id == RolePermission.role_id)
        .join(UserRole, UserRole.role_id == Role.id)
        .where(
            UserRole.user_id == current_user.id,
            Role.status == 1,
            Permission.code == needed,
        )
        .limit(1)
    )
    if not await db.scalar(stmt):
        raise HTTPException(status_code=403, detail="无权限")
    return current_user


@router.post("/orders", summary="创建订单")
async def create_order(
    data: OrderCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:create")),
):
    try:
        order = await OrderService.create(db, data)
    except ValueError as e:
        return Result.bad_request(str(e))
    await OperationLogService.log(
        db,
        module="order",
        action="create",
        resource_id=str(order["id"]),
        content=f"创建订单: {order['order_no']}",
        operator=_operator(current_user),
    )
    return Result.ok(order)


@router.get("/orders", summary="订单列表")
async def list_orders(
    status: str = Query(None, description="订单状态"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:view")),
):
    rows, total = await OrderService.list_orders(db, status, page, page_size)
    return PageResult.ok(rows, total, page, page_size)


@router.get("/orders/{order_id}", summary="订单详情")
async def get_order(
    order_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:view")),
):
    order = await OrderService.get_detail(db, order_id)
    if not order:
        return Result.not_found("订单不存在")
    return Result.ok(order)


@router.post("/orders/{order_id}/shipping-quote", summary="绑定订单运费快照")
async def bind_order_shipping_quote(
    order_id: int,
    data: OrderShippingQuoteBind,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:update")),
):
    try:
        order = await OrderService.bind_shipping_quote(db, order_id, data)
    except ValueError as e:
        return Result.bad_request(str(e))
    if not order:
        return Result.not_found("订单不存在")
    await OperationLogService.log(
        db,
        module="order",
        action="bind_shipping_quote",
        resource_id=str(order_id),
        content=f"绑定订单运费快照: {order['order_no']} 运费={order['shipping_fee']}",
        operator=_operator(current_user),
    )
    return Result.ok(order)


@router.put("/orders/{order_id}/profit-inputs", summary="更新订单利润输入")
async def update_order_profit_inputs(
    order_id: int,
    data: OrderProfitInputsUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:update")),
):
    order = await OrderService.update_profit_inputs(db, order_id, data)
    if not order:
        return Result.not_found("订单不存在")
    await OperationLogService.log(
        db,
        module="order",
        action="update_profit_inputs",
        resource_id=str(order_id),
        content=f"更新订单利润输入: {order['order_no']} 利润={order['profit_amount']}",
        operator=_operator(current_user),
    )
    return Result.ok(order)


@router.put("/orders/{order_id}/status", summary="更新订单状态")
async def update_order_status(
    order_id: int,
    data: OrderStatusUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(_require_order_status_or_cancel),
):
    try:
        order = await OrderService.update_status(
            db,
            order_id,
            data.status,
            data.remark,
            data.tracking_number,
            _operator(current_user),
        )
    except ValueError as e:
        return Result.bad_request(str(e))
    if not order:
        return Result.not_found("订单不存在")
    action = "cancel" if data.status == "cancelled" else "update_status"
    await OperationLogService.log(
        db,
        module="order",
        action=action,
        resource_id=str(order_id),
        content=f"订单{action}: {order['order_no']} -> {data.status}",
        operator=_operator(current_user),
    )
    return Result.ok(order)
