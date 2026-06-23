"""结算管理 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result, PageResult
from app.database import get_db
from app.models import Settlement, SettlementItem, User
from app.operation_log.service import OperationLogService
from app.settlement.schemas import (
    SettlementCreate,
    SettlementUpdate,
    SettlementItemCreate,
    SettlementReconcileRequest,
)
from app.settlement.service import SettlementService

router = APIRouter(tags=["结算管理"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


# ── 结算单 CRUD ─────────────────────────────────────────────────


@router.post("/settlements", summary="导入结算单")
async def import_settlement(
    data: SettlementCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:create")),
):
    """导入结算单"""
    settlement = await SettlementService.import_settlement(
        db, data.model_dump(), items_data=[]
    )
    await OperationLogService.log(
        db,
        module="settlement",
        action="import",
        resource_id=str(settlement.id),
        content=f"导入结算单: {settlement.settlement_no}",
        operator=_operator(current_user),
    )
    return Result.ok(
        {
            "id": settlement.id,
            "settlement_no": settlement.settlement_no,
            "total_revenue": float(settlement.total_revenue),
            "total_fee": float(settlement.total_fee),
            "total_net": float(settlement.total_net),
            "status": settlement.status,
        }
    )


@router.get("/settlements", summary="结算单列表")
async def list_settlements(
    platform_id: int = Query(None, description="平台ID"),
    status: str = Query(None, description="状态"),
    keyword: str = Query(None, description="搜索关键词"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:view")),
):
    rows, total = await SettlementService.list_settlements(
        db,
        platform_id=platform_id,
        status=status,
        keyword=keyword,
        page=page,
        page_size=page_size,
    )
    return PageResult.ok(records=rows, total=total, page=page, page_size=page_size)


@router.get("/settlements/{settlement_id}", summary="结算单详情")
async def get_settlement(
    settlement_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:view")),
):
    detail = await SettlementService.get_settlement_detail(db, settlement_id)
    if not detail:
        return Result.not_found("结算单不存在")
    return Result.ok(detail)


@router.put("/settlements/{settlement_id}", summary="更新结算单")
async def update_settlement(
    settlement_id: int,
    data: SettlementUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:update")),
):
    settlement = await db.get(Settlement, settlement_id)
    if not settlement:
        return Result.not_found("结算单不存在")

    update_data = data.model_dump(exclude_unset=True)
    for key, val in update_data.items():
        setattr(settlement, key, val)
    await db.flush()

    await OperationLogService.log(
        db,
        module="settlement",
        action="update",
        resource_id=str(settlement_id),
        content=f"更新结算单: {settlement.settlement_no}",
        operator=_operator(current_user),
    )
    detail = await SettlementService.get_settlement_detail(db, settlement_id)
    return Result.ok(detail)


@router.delete("/settlements/{settlement_id}", summary="删除结算单")
async def delete_settlement(
    settlement_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:delete")),
):
    ok = await SettlementService.delete_settlement(db, settlement_id)
    if not ok:
        return Result.not_found("结算单不存在")
    await OperationLogService.log(
        db,
        module="settlement",
        action="delete",
        resource_id=str(settlement_id),
        content=f"删除结算单: {settlement_id}",
        operator=_operator(current_user),
    )
    return Result.ok(message="删除成功")


# ── 结算明细 ─────────────────────────────────────────────────


@router.post("/settlements/{settlement_id}/items", summary="添加结算明细")
async def add_settlement_item(
    settlement_id: int,
    data: SettlementItemCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:create")),
):
    settlement = await db.get(Settlement, settlement_id)
    if not settlement:
        return Result.not_found("结算单不存在")

    item = SettlementItem(settlement_id=settlement_id, **data.model_dump())
    db.add(item)
    await db.flush()
    await db.refresh(item)

    # 重算汇总
    await SettlementService._recalc_totals(db, settlement_id)

    return Result.ok(
        {
            "id": item.id,
            "transaction_type": item.transaction_type,
            "amount": float(item.amount),
            "fee": float(item.fee),
            "net": float(item.net),
        }
    )


@router.get("/settlements/{settlement_id}/items", summary="结算明细列表")
async def list_settlement_items(
    settlement_id: int,
    reconciliation_status: str = Query(None, description="对账状态"),
    transaction_type: str = Query(None, description="交易类型过滤"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:view")),
):
    rows, total = await SettlementService.list_items(
        db,
        settlement_id,
        reconciliation_status,
        transaction_type,
        page,
        page_size,
    )
    return PageResult.ok(records=rows, total=total, page=page, page_size=page_size)


# ── 对账 ─────────────────────────────────────────────────────


@router.post("/settlements/{settlement_id}/reconcile", summary="执行对账")
async def reconcile_settlement(
    settlement_id: int,
    data: SettlementReconcileRequest = SettlementReconcileRequest(),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:reconcile")),
):
    try:
        result = await SettlementService.reconcile(
            db,
            settlement_id,
            auto_match=data.auto_match,
            strategy=data.strategy,
        )
    except ValueError as e:
        return Result.bad_request(str(e))

    await OperationLogService.log(
        db,
        module="settlement",
        action="reconcile",
        resource_id=str(settlement_id),
        content=f"对账完成: 匹配 {result['matched']}/{result['total']} 笔",
        operator=_operator(current_user),
    )
    return Result.ok(result)


@router.put("/settlements/items/{item_id}/reconciliation", summary="更新明细对账状态")
async def update_item_reconciliation(
    item_id: int,
    data: dict,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:reconcile")),
):
    """手动更新某条结算明细的对账状态"""
    status = data.get("status", "matched")
    note = data.get("note")
    result = await SettlementService.update_item_reconciliation(
        db,
        item_id,
        status=status,
        note=note,
        reconciled_by=_operator(current_user),
    )
    if not result:
        return Result.not_found("明细不存在")
    await OperationLogService.log(
        db,
        module="settlement",
        action="update_reconciliation",
        resource_id=str(item_id),
        content=f"更新对账状态: {status}",
        operator=_operator(current_user),
    )
    return Result.ok(result)


# ── 模拟数据 ─────────────────────────────────────────────────


@router.post("/settlements/mock", summary="生成模拟结算数据")
async def generate_mock_settlement(
    platform_id: int = Query(..., description="平台ID"),
    count: int = Query(5, ge=1, le=50, description="模拟订单数量"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:create")),
):
    """基于现有订单生成模拟结算数据，用于演示"""
    settlement = await SettlementService.generate_mock_data(db, platform_id, count)
    await OperationLogService.log(
        db,
        module="settlement",
        action="generate_mock",
        resource_id=str(settlement.id),
        content=f"生成模拟结算: {settlement.settlement_no}",
        operator=_operator(current_user),
    )
    return Result.ok(
        {
            "id": settlement.id,
            "settlement_no": settlement.settlement_no,
            "total_revenue": float(settlement.total_revenue),
            "total_fee": float(settlement.total_fee),
            "total_net": float(settlement.total_net),
            "status": settlement.status,
        }
    )
