"""多平台上架任务 - 路由"""

from typing import Optional
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common.schemas import Result, PageResult, PageParam
from app.database import get_db
from app.models import User
from app.listing_task.schemas import ListingTaskCreate
from app.listing_task.service import ListingTaskService
from app.operation_log.service import OperationLogService

router = APIRouter(prefix="/listing-tasks", tags=["多平台上架任务"])


@router.post("", summary="创建上架任务")
async def create_task(
    data: ListingTaskCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:publish")),
):
    if not data.name.strip():
        return Result.error("任务名称不能为空")
    if not data.product_ids or not data.platform_ids:
        return Result.error("请选择产品和平台")

    task = await ListingTaskService.create(
        db,
        name=data.name.strip(),
        product_ids=data.product_ids,
        platform_ids=data.platform_ids,
        created_by=current_user.id,
    )

    await OperationLogService.log(
        db,
        module="listing_task",
        action="create",
        resource_id=str(task.id),
        content=f"创建上架任务: {task.name} ({task.total_count}个条目)",
        operator=current_user.username,
    )

    return Result.ok(
        {
            "id": task.id,
            "name": task.name,
            "status": task.status,
            "total_count": task.total_count,
        }
    )


@router.get("", summary="上架任务列表")
async def list_tasks(
    params: PageParam = Depends(),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:view")),
):
    tasks, total = await ListingTaskService.list_tasks(
        db, page=params.page, page_size=params.page_size
    )
    records = []
    for t in tasks:
        records.append(
            {
                "id": t.id,
                "name": t.name,
                "status": t.status,
                "total_count": t.total_count,
                "success_count": t.success_count,
                "failed_count": t.failed_count,
                "created_by": t.created_by,
                "created_at": t.created_at.isoformat() if t.created_at else None,
                "updated_at": t.updated_at.isoformat() if t.updated_at else None,
            }
        )
    return PageResult.ok(records, total, params.page, params.page_size)


@router.get("/{task_id}", summary="任务详情")
async def get_task(
    task_id: int,
    page: int = 1,
    page_size: int = 20,
    status: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:view")),
):
    task = await ListingTaskService.get_task(db, task_id)
    if not task:
        return Result.not_found("任务不存在")

    items, items_total = await ListingTaskService.get_task_items(
        db, task_id, status_filter=status, page=page, page_size=page_size
    )

    return Result.ok(
        {
            "id": task.id,
            "name": task.name,
            "status": task.status,
            "total_count": task.total_count,
            "success_count": task.success_count,
            "failed_count": task.failed_count,
            "created_by": task.created_by,
            "created_at": task.created_at.isoformat() if task.created_at else None,
            "updated_at": task.updated_at.isoformat() if task.updated_at else None,
            "items": items,
            "items_total": items_total,
        }
    )


@router.delete("/{task_id}", summary="删除任务")
async def delete_task(
    task_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:publish")),
):
    task = await ListingTaskService.get_task(db, task_id)
    if not task:
        return Result.not_found("任务不存在")
    if task.status == "in_progress":
        return Result.error("执行中的任务无法删除")

    await ListingTaskService.delete_task(db, task)

    await OperationLogService.log(
        db,
        module="listing_task",
        action="delete",
        resource_id=str(task_id),
        content=f"删除上架任务: {task.name}",
        operator=current_user.username,
    )

    return Result.ok(message="删除成功")


@router.post("/{task_id}/execute", summary="执行任务")
async def execute_task(
    task_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:publish")),
):
    task = await ListingTaskService.get_task(db, task_id)
    if not task:
        return Result.not_found("任务不存在")
    if task.status == "in_progress":
        return Result.error("任务正在执行中")

    try:
        task = await ListingTaskService.execute_task(db, task)
    except Exception as e:
        return Result.error(f"执行失败: {str(e)}")

    await OperationLogService.log(
        db,
        module="listing_task",
        action="execute",
        resource_id=str(task_id),
        content=f"执行上架任务: {task.name} (成功:{task.success_count}, 失败:{task.failed_count})",
        operator=current_user.username,
    )

    return Result.ok(
        {
            "id": task.id,
            "status": task.status,
            "success_count": task.success_count,
            "failed_count": task.failed_count,
            "total_count": task.total_count,
        }
    )


@router.post("/{task_id}/retry-failed", summary="重试任务下所有失败项")
async def retry_all_failed_items(
    task_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:publish")),
):
    task = await ListingTaskService.get_task(db, task_id)
    if not task:
        return Result.not_found("任务不存在")

    count = await ListingTaskService.retry_all_failed(db, task_id)

    await OperationLogService.log(
        db,
        module="listing_task",
        action="retry_all_failed",
        resource_id=str(task_id),
        content=f"重试所有失败条目: task_id={task_id}, count={count}",
        operator=current_user.username,
    )

    return Result.ok({"reset_count": count})


@router.post("/{task_id}/items/{item_id}/retry", summary="重试单个条目")
async def retry_item(
    task_id: int,
    item_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:publish")),
):
    task = await ListingTaskService.get_task(db, task_id)
    if not task:
        return Result.not_found("任务不存在")

    try:
        item = await ListingTaskService.retry_item(db, task, item_id)
    except ValueError as e:
        return Result.error(str(e))

    await OperationLogService.log(
        db,
        module="listing_task",
        action="retry",
        resource_id=str(item_id),
        content=f"重试上架条目: task_id={task_id}, product_id={item.product_id}, platform_id={item.platform_id}",
        operator=current_user.username,
    )

    return Result.ok(
        {
            "id": item.id,
            "status": item.status,
            "error_message": item.error_message,
        }
    )
