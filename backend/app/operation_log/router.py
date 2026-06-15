"""操作日志 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.common import Result, PageResult
from app.models import User, OperationLog
from app.operation_log.schemas import OperationLogVO
from app.operation_log.service import OperationLogService

router = APIRouter(tags=["操作日志"])


def log_to_vo(log) -> OperationLogVO:
    return OperationLogVO(
        id=log.id,
        module=log.module,
        action=log.action,
        resource_id=log.resource_id,
        content=log.content,
        operator=log.operator,
        ip=log.ip,
        duration=log.duration,
        created_at=log.created_at,
    )


@router.get("/operation-logs", summary="操作日志列表")
async def list_operation_logs(
    module: str = Query(None, description="模块"),
    action: str = Query(None, description="操作"),
    operator: str = Query(None, description="操作人"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("operation_log:view")),
):
    logs, total = await OperationLogService.list_logs(db, module, action, operator, page, page_size)
    items = [log_to_vo(l) for l in logs]
    return PageResult.ok(items, total, page, page_size)


@router.get("/operation-logs/modules", summary="获取所有模块列表（筛选用）")
async def get_log_modules(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("operation_log:view")),
):
    """获取有日志记录的所有模块名"""
    stmt = select(OperationLog.module).distinct().order_by(OperationLog.module)
    result = await db.execute(stmt)
    modules = [row[0] for row in result.all() if row[0]]
    return Result.ok(modules)
