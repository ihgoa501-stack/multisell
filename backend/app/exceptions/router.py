"""异常工作台 - 路由"""

from datetime import datetime, timezone

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result
from app.database import get_db
from app.models import ExceptionItem, User
from app.exceptions.schemas import (
    ExceptionAssignRequest,
    ExceptionNoteRequest,
)
from app.exceptions.service import ExceptionService

router = APIRouter(tags=["异常工作台"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.post("/exceptions/generate", summary="扫描生成异常条目")
async def generate_exceptions(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("exception:generate")),
):
    result = await ExceptionService.generate(db, operator=_operator(current_user))
    return Result.ok(result)


@router.get("/exceptions", summary="异常条目列表")
async def list_exceptions(
    source_module: str | None = Query(None, description="来源模块筛选"),
    severity: str | None = Query(None, description="严重程度筛选"),
    status: str | None = Query(None, description="状态筛选"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("exception:view")),
):
    items = await ExceptionService.list_items(
        db, source_module=source_module, severity=severity, status=status,
    )
    return Result.ok(items)


@router.get("/exceptions/{exception_id}", summary="异常条目详情")
async def get_exception(
    exception_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("exception:view")),
):
    item = await ExceptionService.get_item(db, exception_id)
    if not item:
        raise HTTPException(status_code=404, detail="异常不存在")
    return Result.ok(item)


@router.post("/exceptions/{exception_id}/assign", summary="分配异常")
async def assign_exception(
    exception_id: int,
    data: ExceptionAssignRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("exception:manage")),
):
    item = await ExceptionService.assign_item(
        db, exception_id, assigned_to=data.assigned_to, operator=_operator(current_user),
    )
    if not item:
        raise HTTPException(status_code=404, detail="异常不存在")
    return Result.ok(item)


@router.post("/exceptions/{exception_id}/resolve", summary="解决异常")
async def resolve_exception(
    exception_id: int,
    data: ExceptionNoteRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("exception:manage")),
):
    item = await ExceptionService.resolve_item(
        db, exception_id, note=data.note, operator=_operator(current_user),
    )
    if not item:
        raise HTTPException(status_code=404, detail="异常不存在")
    return Result.ok(item)


@router.post("/exceptions/{exception_id}/ignore", summary="忽略异常")
async def ignore_exception(
    exception_id: int,
    data: ExceptionNoteRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("exception:manage")),
):
    item = await ExceptionService.ignore_item(
        db, exception_id, note=data.note, operator=_operator(current_user),
    )
    if not item:
        raise HTTPException(status_code=404, detail="异常不存在")
    return Result.ok(item)
