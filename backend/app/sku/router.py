"""规格与SKU管理 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result
from app.sku.schemas import SpecDefine, SkuUpdate, SpecNameVO, SpecValueVO, SkuVO
from app.sku.service import SpecService

router = APIRouter(tags=["规格与SKU管理"])


def spec_name_to_vo(spec) -> SpecNameVO:
    return SpecNameVO(
        id=spec.id,
        name=spec.name,
        sort_order=spec.sort_order,
        values=[SpecValueVO(id=v.id, value=v.value, sort_order=v.sort_order) for v in spec.values],
    )


def sku_to_vo(sku) -> SkuVO:
    return SkuVO(
        id=sku.id,
        product_id=sku.product_id,
        code=sku.code,
        barcode=sku.barcode,
        spec_desc=sku.spec_desc,
        spec_values=sku.spec_values,
        price=float(sku.price) if sku.price else None,
        cost_price=float(sku.cost_price) if sku.cost_price else None,
        market_price=float(sku.market_price) if sku.market_price else None,
        stock=sku.stock or 0,
        lock_stock=sku.lock_stock or 0,
        warning_stock=sku.warning_stock or 0,
        weight=float(sku.weight) if sku.weight else None,
        image=sku.image,
        status=sku.status,
        created_at=sku.created_at,
        updated_at=sku.updated_at,
    )


@router.post("/products/{product_id}/specs", summary="定义规格模板")
async def define_specs(product_id: int, data: SpecDefine, db: AsyncSession = Depends(get_db)):
    specs = await SpecService.define_specs(db, product_id, [s.model_dump() for s in data.specs])
    return Result.ok([spec_name_to_vo(s) for s in specs])


@router.get("/products/{product_id}/specs", summary="获取规格模板")
async def get_specs(product_id: int, db: AsyncSession = Depends(get_db)):
    specs = await SpecService.get_specs(db, product_id)
    return Result.ok([spec_name_to_vo(s) for s in specs])


@router.post("/products/{product_id}/skus/generate", summary="生成SKU（笛卡尔积）")
async def generate_skus(product_id: int, db: AsyncSession = Depends(get_db)):
    skus = await SpecService.generate_skus(db, product_id)
    if not skus:
        return Result.bad_request("请先定义规格模板")
    return Result.ok({"total": len(skus), "skus": [sku_to_vo(s) for s in skus]})


@router.get("/products/{product_id}/skus", summary="获取商品的所有SKU")
async def get_skus(product_id: int, db: AsyncSession = Depends(get_db)):
    skus = await SpecService.get_skus_by_product(db, product_id)
    return Result.ok([sku_to_vo(s) for s in skus])


@router.put("/skus/{sku_id}", summary="更新SKU")
async def update_sku(sku_id: int, data: SkuUpdate, db: AsyncSession = Depends(get_db)):
    sku = await SpecService.update_sku(db, sku_id, data.model_dump(exclude_unset=True))
    if not sku:
        return Result.not_found("SKU不存在")
    return Result.ok(sku_to_vo(sku))


@router.get("/skus/{sku_id}", summary="SKU详情")
async def get_sku(sku_id: int, db: AsyncSession = Depends(get_db)):
    sku = await SpecService.get_sku_by_id(db, sku_id)
    if not sku:
        return Result.not_found("SKU不存在")
    return Result.ok(sku_to_vo(sku))
