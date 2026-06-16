"""费用分摊 - 路由"""

from fastapi import APIRouter, Depends, File, HTTPException, Query, UploadFile
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.allocation.service import AllocationService, ALLOCATION_METHODS

router = APIRouter(tags=["费用分摊"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.post("/allocations/import", summary="导入分摊 CSV")
async def import_allocation(
    file: UploadFile = File(...),
    allocation_type: str = Query(..., description="分摊类型: first_leg/fba/overseas_warehouse/other"),
    allocation_method: str = Query(..., description="分摊方法: quantity/weight/volume/value"),
    total_amount: float = Query(..., gt=0, description="分摊总金额"),
    currency: str = Query("CNY", description="币种"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("allocation:import")),
):
    if not file.filename or not file.filename.lower().endswith(".csv"):
        return Result.bad_request("仅支持 .csv 格式")
    if allocation_method not in ALLOCATION_METHODS:
        return Result.bad_request(f"不支持的分摊方法: {allocation_method}")

    content = await file.read()
    try:
        result = await AllocationService.import_csv(
            db, file.filename, content,
            allocation_type=allocation_type, allocation_method=allocation_method,
            total_amount=total_amount, currency=currency,
            operator=_operator(current_user),
        )
    except ValueError as e:
        return Result.bad_request(str(e))

    if result.get("batch_id") is None and result.get("error_rows", 0) > 0:
        return Result.bad_request(f"解析失败: {result['errors'][0]['message']}")
    return Result.ok(result)


@router.get("/allocations", summary="分摊批次列表")
async def list_allocations(
    status: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("allocation:view")),
):
    batches = await AllocationService.list_batches(db, status=status)
    return Result.ok(batches)


@router.get("/allocations/{batch_id}", summary="分摊批次详情")
async def get_allocation_batch(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("allocation:view")),
):
    batch = await AllocationService.get_batch(db, batch_id)
    if not batch:
        raise HTTPException(status_code=404, detail="分摊批次不存在")
    return Result.ok(batch)


@router.get("/allocations/{batch_id}/items", summary="分摊明细列表")
async def list_allocation_items(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("allocation:view")),
):
    items = await AllocationService.list_items(db, batch_id)
    return Result.ok(items)


@router.post("/allocations/{batch_id}/calculate", summary="计算分摊")
async def calculate_allocation(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("allocation:calculate")),
):
    try:
        result = await AllocationService.calculate(db, batch_id, operator=_operator(current_user))
    except ValueError as e:
        return Result.bad_request(str(e))
    if not result:
        raise HTTPException(status_code=404, detail="分摊批次不存在")
    return Result.ok(result)


@router.post("/allocations/{batch_id}/post-to-ledger", summary="入账到利润账本")
async def post_allocation_to_ledger(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("allocation:post")),
):
    result = await AllocationService.post_to_ledger(db, batch_id, operator=_operator(current_user))
    if not result:
        raise HTTPException(status_code=404, detail="分摊批次不存在")
    return Result.ok(result)
