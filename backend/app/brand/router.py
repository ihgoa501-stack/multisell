"""品牌管理 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result, PageResult
from app.brand.schemas import BrandCreate, BrandUpdate, BrandVO
from app.brand.service import BrandService

router = APIRouter(tags=["品牌管理"])


def brand_to_vo(b) -> BrandVO:
    return BrandVO(
        id=b.id,
        name=b.name,
        logo=b.logo,
        description=b.description,
        status=b.status,
        sort_order=b.sort_order,
        created_at=b.created_at,
        updated_at=b.updated_at,
    )


@router.post("/brands", summary="创建品牌")
async def create_brand(data: BrandCreate, db: AsyncSession = Depends(get_db)):
    b = await BrandService.create(db, data.model_dump())
    return Result.ok(brand_to_vo(b))


@router.put("/brands/{brand_id}", summary="更新品牌")
async def update_brand(brand_id: int, data: BrandUpdate, db: AsyncSession = Depends(get_db)):
    b = await BrandService.update(db, brand_id, data.model_dump(exclude_unset=True))
    if not b:
        return Result.not_found("品牌不存在")
    return Result.ok(brand_to_vo(b))


@router.get("/brands/{brand_id}", summary="品牌详情")
async def get_brand(brand_id: int, db: AsyncSession = Depends(get_db)):
    b = await BrandService.get_by_id(db, brand_id)
    if not b:
        return Result.not_found("品牌不存在")
    return Result.ok(brand_to_vo(b))


@router.get("/brands", summary="品牌列表")
async def list_brands(
    name: str = Query(None, description="品牌名称"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
):
    brands, total = await BrandService.list_brands(db, name, page, page_size)
    items = [brand_to_vo(b) for b in brands]
    return PageResult.ok(items, total, page, page_size)


@router.get("/brands/all", summary="所有品牌（下拉选择用）")
async def get_all_brands(db: AsyncSession = Depends(get_db)):
    brands = await BrandService.get_all(db)
    items = [{"id": b.id, "name": b.name} for b in brands]
    return Result.ok(items)


@router.delete("/brands/{brand_id}", summary="删除品牌")
async def delete_brand(brand_id: int, db: AsyncSession = Depends(get_db)):
    try:
        ok = await BrandService.delete(db, brand_id)
        if not ok:
            return Result.not_found("品牌不存在")
        return Result.ok(message="删除成功")
    except ValueError as e:
        return Result.error(message=str(e))
