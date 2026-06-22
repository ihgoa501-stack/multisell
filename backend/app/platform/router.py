"""平台管理 - 路由"""

from datetime import datetime, timezone, timedelta

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.common import Result
from app.listing.adapters import get_listing_adapter
from app.models import Platform, User
from app.platform.schemas import PlatformCreate, PlatformUpdate, PlatformVO
from app.platform.service import PlatformService
from app.operation_log.service import OperationLogService
from app.order_import.sync_worker import OrderSyncWorker

router = APIRouter(tags=["平台管理"])


def platform_to_vo(p) -> PlatformVO:
    """转换为VO（不返回api_key）"""
    return PlatformVO(
        id=p.id,
        name=p.name,
        code=p.code,
        api_base_url=p.api_base_url,
        client_id=p.client_id,
        extra_config=p.extra_config,
        status=p.status,
        sort_order=p.sort_order,
        created_at=p.created_at,
        updated_at=p.updated_at,
    )


@router.post("/platforms", summary="创建平台配置")
async def create_platform(
    data: PlatformCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform:create")),
):
    p = await PlatformService.create(db, data.model_dump())
    await OperationLogService.log(
        db,
        module="platform",
        action="create",
        resource_id=str(p.id),
        content=f"创建平台: {p.name}",
        operator=current_user.username,
    )
    return Result.ok(platform_to_vo(p))


@router.put("/platforms/{platform_id}", summary="更新平台配置")
async def update_platform(
    platform_id: int,
    data: PlatformUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform:update")),
):
    p = await PlatformService.update(db, platform_id, data.model_dump(exclude_unset=True))
    if not p:
        return Result.not_found("平台配置不存在")
    await OperationLogService.log(
        db,
        module="platform",
        action="update",
        resource_id=str(platform_id),
        content=f"更新平台: {p.name}",
        operator=current_user.username,
    )
    return Result.ok(platform_to_vo(p))


@router.get("/platforms", summary="平台列表")
async def list_platforms(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform:view")),
):
    platforms = await PlatformService.list_all(db)
    return Result.ok([platform_to_vo(p) for p in platforms])


@router.get("/platforms/{platform_id}", summary="平台详情")
async def get_platform(
    platform_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform:view")),
):
    p = await PlatformService.get_by_id(db, platform_id)
    if not p:
        return Result.not_found("平台配置不存在")
    return Result.ok(platform_to_vo(p))


@router.delete("/platforms/{platform_id}", summary="删除平台配置")
async def delete_platform(
    platform_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform:delete")),
):
    ok = await PlatformService.delete(db, platform_id)
    if not ok:
        return Result.not_found("平台配置不存在")
    await OperationLogService.log(
        db,
        module="platform",
        action="delete",
        resource_id=str(platform_id),
        content=f"删除平台: {platform_id}",
        operator=current_user.username,
    )
    return Result.ok(message="删除成功")


@router.post("/platforms/{platform_id}/backfill-orders", summary="回填历史订单")
async def backfill_orders(
    platform_id: int,
    days_back: int = Query(7, ge=1, le=90),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order:create")),
):
    """从平台 API 回填最近 N 天的历史订单。"""
    platform = await db.get(Platform, platform_id)
    if not platform:
        return Result.not_found("平台不存在")
    adapter = get_listing_adapter(platform.code)
    if not hasattr(adapter, "fetch_orders"):
        return Result.bad_request(f"平台 {platform.code} 不支持订单导入")

    since = datetime.now(timezone.utc) - timedelta(days=days_back)
    raw_orders = await adapter.fetch_orders(platform=platform, since=since, db=db)

    worker = OrderSyncWorker()
    count = 0
    for raw in raw_orders:
        await worker._upsert_order(db, raw, platform.id)
        count += 1
    await db.flush()

    await OperationLogService.log(
        db, module="order_import", action="backfill",
        resource_id=str(platform_id),
        content=f"回填订单: {count} 条, 平台={platform.code}, 天数={days_back}",
        operator=current_user.username,
    )
    return Result.ok({"platform_id": platform_id, "imported": count, "days_back": days_back})
