"""平台管理 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.common import Result
from app.models import User
from app.platform.schemas import PlatformCreate, PlatformUpdate, PlatformVO
from app.platform.service import PlatformService
from app.operation_log.service import OperationLogService

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
