"""订单导入 - 路由"""

from fastapi import APIRouter, Depends, Query, UploadFile, File
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result, PageResult
from app.database import get_db
from app.models import User
from app.operation_log.service import OperationLogService
from app.order_import.schemas import OrderImportRequest
from app.order_import.service import OrderImportService

router = APIRouter(tags=["订单导入"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.post("/order-import", summary="导入订单")
async def import_orders(
    data: OrderImportRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:create")),
):
    """批量导入订单数据"""
    result = await OrderImportService.import_orders(
        db,
        source_type=data.source_type,
        orders_data=data.orders,
        platform_id=data.platform_id,
        created_by=_operator(current_user),
    )
    await OperationLogService.log(
        db,
        module="order_import",
        action="import",
        resource_id=str(result["import_id"]),
        content=f"导入订单: {result['source_type']} 成功={result['success']} 失败={result['failed']}",
        operator=_operator(current_user),
    )
    return Result.ok(result)


@router.post("/order-import/csv", summary="从CSV导入订单")
async def import_orders_csv(
    source_type: str = Query(..., description="来源类型: ozon/shopee/wb"),
    file: UploadFile = File(...),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:create")),
):
    """上传平台导出的CSV文件导入订单"""
    content = (await file.read()).decode("utf-8-sig")
    rows = await OrderImportService.parse_csv(content, source_type)
    if not rows:
        return Result.bad_request("CSV中未找到有效订单数据")

    result = await OrderImportService.import_orders(
        db,
        source_type=source_type,
        orders_data=rows,
        created_by=_operator(current_user),
    )
    return Result.ok(result)


@router.get("/order-imports", summary="导入记录列表")
async def list_imports(
    source_type: str = Query(None, description="来源类型"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:view")),
):
    rows, total = await OrderImportService.list_imports(
        db, source_type, page, page_size
    )
    return PageResult.ok(records=rows, total=total, page=page, page_size=page_size)


@router.post("/order-import/mock", summary="生成模拟订单数据")
async def generate_mock_orders(
    platform_id: int = Query(..., description="平台ID"),
    count: int = Query(5, ge=1, le=50, description="订单数量"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:create")),
):
    """基于现有SKU生成模拟订单数据并导入，用于演示"""
    rows = await OrderImportService.generate_mock_orders(db, platform_id, count)
    if not rows:
        return Result.bad_request("系统中暂无SKU数据，请先创建商品")

    result = await OrderImportService.import_orders(
        db,
        source_type="mock",
        orders_data=rows,
        platform_id=platform_id,
        created_by=_operator(current_user),
    )
    return Result.ok(result)
