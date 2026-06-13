"""全局搜索 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy import select, or_
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result
from app.models import Product, Sku, Supplier

router = APIRouter(tags=["全局搜索"])


@router.get("/search", summary="全局搜索")
async def global_search(
    q: str = Query(..., min_length=1, max_length=100, description="搜索关键词"),
    limit: int = Query(10, ge=1, le=50, description="每类结果数量"),
    db: AsyncSession = Depends(get_db),
):
    """同时搜索商品、SKU、供应商"""

    keyword = f"%{q}%"
    results = {}

    # 搜索商品
    product_stmt = (
        select(Product)
        .where(or_(
            Product.name.ilike(keyword),
            Product.subtitle.ilike(keyword),
        ))
        .order_by(Product.created_at.desc())
        .limit(limit)
    )
    product_result = await db.execute(product_stmt)
    results["products"] = [
        {"id": p.id, "name": p.name, "subtitle": p.subtitle, "status": p.status}
        for p in product_result.scalars().all()
    ]

    # 搜索SKU
    sku_stmt = (
        select(Sku, Product.name)
        .join(Product, Sku.product_id == Product.id)
        .where(or_(
            Sku.code.ilike(keyword),
            Sku.barcode.ilike(keyword),
            Sku.spec_desc.ilike(keyword),
        ))
        .order_by(Sku.id)
        .limit(limit)
    )
    sku_result = await db.execute(sku_stmt)
    results["skus"] = [
        {"id": sku.id, "code": sku.code, "spec_desc": sku.spec_desc,
         "product_name": product_name, "product_id": sku.product_id}
        for sku, product_name in sku_result.all()
    ]

    # 搜索供应商
    supplier_stmt = (
        select(Supplier)
        .where(or_(
            Supplier.name.ilike(keyword),
            Supplier.contact_person.ilike(keyword),
            Supplier.contact_phone.ilike(keyword),
        ))
        .order_by(Supplier.name)
        .limit(limit)
    )
    supplier_result = await db.execute(supplier_stmt)
    results["suppliers"] = [
        {"id": s.id, "name": s.name, "contact_person": s.contact_person}
        for s in supplier_result.scalars().all()
    ]

    return Result.ok(results)
