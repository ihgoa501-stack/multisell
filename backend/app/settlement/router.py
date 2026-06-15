"""平台结算导入 - 路由"""

from fastapi import APIRouter, Depends, File, UploadFile
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.settlement.service import SettlementService

router = APIRouter(tags=["平台结算"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.post("/settlements/import", summary="导入平台结算 CSV")
async def import_settlement(
    file: UploadFile = File(...),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:import")),
):
    if not file.filename or not file.filename.lower().endswith(".csv"):
        return Result.bad_request("仅支持 .csv 格式的结算文件")

    content = await file.read()
    try:
        result = await SettlementService.import_csv(
            db, file.filename, content, operator=_operator(current_user),
        )
    except ValueError as e:
        return Result.bad_request(str(e))

    if result.get("batch_id") is None and result.get("error_rows", 0) > 0:
        return Result.bad_request(f"解析失败: {result['errors'][0]['message']}")

    return Result.ok(result)


@router.get("/settlements", summary="结算批次列表")
async def list_settlements(
    status: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:view")),
):
    batches = await SettlementService.list_batches(db, status=status)
    return Result.ok(batches)


@router.get("/settlements/unmatched", summary="未匹配结算行")
async def list_unmatched_settlements(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:view")),
):
    items = await SettlementService.list_unmatched_items(db)
    return Result.ok(items)


@router.get("/settlements/{batch_id}", summary="结算批次详情")
async def get_settlement_batch(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:view")),
):
    batch = await SettlementService.get_batch(db, batch_id)
    if not batch:
        return Result.not_found("结算批次不存在")
    return Result.ok(batch)


@router.get("/settlements/{batch_id}/items", summary="结算行列表")
async def list_settlement_items(
    batch_id: int,
    match_status: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("settlement:view")),
):
    items = await SettlementService.list_items(db, batch_id, match_status=match_status)
    return Result.ok(items)
