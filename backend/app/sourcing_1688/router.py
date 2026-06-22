"""1688 货源采集 - 路由"""
from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.common import Result, PageResult
from app.models import User
from app.sourcing_1688.schemas import CollectPayload, Sourcing1688ProductVO, ImportPayload
from app.sourcing_1688.service import Sourcing1688Service
from app.operation_log.service import OperationLogService

router = APIRouter(prefix="/1688-collect", tags=["1688 货源采集"])


def _to_vo(p) -> Sourcing1688ProductVO:
    return Sourcing1688ProductVO(
        id=p.id,
        source_url=p.source_url,
        title=p.title,
        price=float(p.price) if p.price else None,
        moq=p.moq,
        supplier_name=p.supplier_name,
        shop_url=p.shop_url,
        shop_location=p.shop_location,
        images=p.images,
        attributes=p.attributes,
        sku_variants=p.sku_variants,
        description=p.description,
        package_length_cm=float(p.package_length_cm) if p.package_length_cm else None,
        package_width_cm=float(p.package_width_cm) if p.package_width_cm else None,
        package_height_cm=float(p.package_height_cm) if p.package_height_cm else None,
        package_weight_kg=float(p.package_weight_kg) if p.package_weight_kg else None,
        status=p.status,
        product_id=p.product_id,
        supplier_id=p.supplier_id,
        collected_by=p.collected_by,
        imported_by=p.imported_by,
        imported_at=p.imported_at,
        created_at=p.created_at,
        updated_at=p.updated_at,
    )


@router.post("/products", summary="采集 1688 商品")
async def collect_product(
    data: CollectPayload,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sourcing:collect")),
):
    p = await Sourcing1688Service.collect(db, data, current_user.username)
    await OperationLogService.log(
        db,
        module="sourcing_1688",
        action="collect",
        resource_id=str(p.id),
        content=f"采集 1688 商品: {p.title or p.source_url}",
        operator=current_user.username,
    )
    return Result.ok(_to_vo(p))


@router.get("/products", summary="1688 货源候选池列表")
async def list_products(
    status: str = Query(None, description="状态筛选: collected/imported/rejected"),
    keyword: str = Query(None, description="标题/供应商搜索"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sourcing:view")),
):
    items, total = await Sourcing1688Service.list_products(
        db, status=status, keyword=keyword, page=page, page_size=page_size,
    )
    return PageResult.ok(
        [_to_vo(p) for p in items],
        total=total, page=page, page_size=page_size,
    )


@router.get("/products/{candidate_id}", summary="候选商品详情")
async def get_product(
    candidate_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sourcing:view")),
):
    p = await Sourcing1688Service.get_product(db, candidate_id)
    if not p:
        return Result.not_found("候选商品不存在")
    return Result.ok(_to_vo(p))


@router.post("/products/{candidate_id}/import", summary="确认导入")
async def import_product(
    candidate_id: int,
    data: ImportPayload,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sourcing:import")),
):
    try:
        p = await Sourcing1688Service.import_product(db, candidate_id, data, current_user.username)
    except ValueError as e:
        return Result.bad_request(str(e))

    await OperationLogService.log(
        db,
        module="sourcing_1688",
        action="import",
        resource_id=str(candidate_id),
        content=f"导入 1688 商品到正式库: {p.title}, product_id={p.product_id}",
        operator=current_user.username,
    )
    return Result.ok(_to_vo(p))


@router.post("/products/{candidate_id}/reject", summary="驳回候选商品")
async def reject_product(
    candidate_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sourcing:import")),
):
    p = await Sourcing1688Service.reject_product(db, candidate_id)
    if not p:
        return Result.not_found("候选商品不存在")

    await OperationLogService.log(
        db,
        module="sourcing_1688",
        action="reject",
        resource_id=str(candidate_id),
        content=f"驳回 1688 候选商品: {p.title or p.source_url}",
        operator=current_user.username,
    )
    return Result.ok(_to_vo(p))
