"""规格与SKU管理 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from app.auth import require_permission
from app.database import get_db
from app.common import Result
from app.models import User
from app.sku.schemas import SpecDefine, SkuUpdate, SpecNameVO, SpecValueVO, SkuVO
from app.sku.service import SpecService
from app.models import Inventory
import asyncio
from app.operation_log.service import OperationLogService
from app.events import emit_agent_event

router = APIRouter(tags=["规格与SKU管理"])


def spec_name_to_vo(spec) -> SpecNameVO:
    return SpecNameVO(
        id=spec.id,
        name=spec.name,
        sort_order=spec.sort_order,
        values=[SpecValueVO(id=v.id, value=v.value, sort_order=v.sort_order) for v in spec.values],
    )


def sku_to_vo(sku, stock=None) -> SkuVO:
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
        stock=stock if stock is not None else (sku.stock or 0),
        lock_stock=sku.lock_stock or 0,
        warning_stock=sku.warning_stock or 0,
        weight=float(sku.weight) if sku.weight else None,
        sku_length_cm=float(sku.sku_length_cm) if sku.sku_length_cm is not None else None,
        sku_width_cm=float(sku.sku_width_cm) if sku.sku_width_cm is not None else None,
        sku_height_cm=float(sku.sku_height_cm) if sku.sku_height_cm is not None else None,
        sku_weight_kg=float(sku.sku_weight_kg) if sku.sku_weight_kg is not None else None,
        image=sku.image,
        status=sku.status,
        created_at=sku.created_at,
        updated_at=sku.updated_at,
    )


@router.post("/products/{product_id}/specs", summary="定义规格模板")
async def define_specs(
    product_id: int,
    data: SpecDefine,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sku:create")),
):
    specs = await SpecService.define_specs(db, product_id, [s.model_dump() for s in data.specs])
    await OperationLogService.log(
        db,
        module="sku",
        action="define_specs",
        resource_id=str(product_id),
        content=f"定义规格模板: product_id={product_id}",
        operator=current_user.username,
    )
    return Result.ok([spec_name_to_vo(s) for s in specs])


@router.get("/products/{product_id}/specs", summary="获取规格模板")
async def get_specs(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sku:view")),
):
    specs = await SpecService.get_specs(db, product_id)
    return Result.ok([spec_name_to_vo(s) for s in specs])


@router.post("/products/{product_id}/skus/generate", summary="生成SKU（笛卡尔积）")
async def generate_skus(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sku:create")),
):
    skus = await SpecService.generate_skus(db, product_id)
    if not skus:
        return Result.bad_request("请先定义规格模板")
    await OperationLogService.log(
        db,
        module="sku",
        action="generate_skus",
        resource_id=str(product_id),
        content=f"生成SKU: product_id={product_id}, count={len(skus)}",
        operator=current_user.username,
    )
    return Result.ok({"total": len(skus), "skus": [sku_to_vo(s) for s in skus]})


@router.get("/products/{product_id}/skus", summary="获取商品的所有SKU")
async def get_skus(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sku:view")),
):
    skus = await SpecService.get_skus_by_product(db, product_id)
    sku_ids = [sku.id for sku in skus]
    stock_by_sku: dict[int, int] = {}
    if sku_ids:
        result = await db.execute(
            select(Inventory.sku_id, Inventory.quantity).where(Inventory.sku_id.in_(sku_ids))
        )
        stock_by_sku = {sku_id: quantity for sku_id, quantity in result.all()}
    return Result.ok([sku_to_vo(s, stock_by_sku.get(s.id)) for s in skus])


@router.put("/skus/{sku_id}", summary="更新SKU")
async def update_sku(
    sku_id: int,
    data: SkuUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sku:update")),
):
    sku = await SpecService.update_sku(db, sku_id, data.model_dump(exclude_unset=True))
    if not sku:
        return Result.not_found("SKU不存在")
    await OperationLogService.log(
        db,
        module="sku",
        action="update",
        resource_id=str(sku_id),
        content=f"更新SKU: sku_id={sku_id}",
        operator=current_user.username,
    )

    # Agent 事件：价格变动触发折扣风险检查
    if data.price is not None and sku.price is not None and float(data.price) != float(sku.price):
        asyncio.ensure_future(emit_agent_event("price.changed", {
            "sku_code": sku.code or sku.spec_desc or f"SKU#{sku.id}",
            "original_price": float(sku.price),
            "new_price": float(data.price),
            "sku_id": sku_id,
        }, source="sku.router"))

    return Result.ok(sku_to_vo(sku))


@router.get("/skus/{sku_id}", summary="SKU详情")
async def get_sku(
    sku_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("sku:view")),
):
    sku = await SpecService.get_sku_by_id(db, sku_id)
    if not sku:
        return Result.not_found("SKU不存在")
    # 从 Inventory 表查询真实库存（sku.stock 已废弃）
    inventory = await db.scalar(
        select(Inventory.quantity).where(Inventory.sku_id == sku_id)
    )
    return Result.ok(sku_to_vo(sku, stock=inventory))
