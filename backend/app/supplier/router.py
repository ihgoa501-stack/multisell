"""供应商管理 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.common import Result, PageResult
from app.models import User
from app.supplier.schemas import (
    SupplierCreate, SupplierUpdate, SupplierVO,
    ProductSupplierBind, ProductSupplierVO,
)
from app.supplier.service import SupplierService
from app.operation_log.service import OperationLogService

router = APIRouter(tags=["供应商管理"])


def supplier_to_vo(s) -> SupplierVO:
    return SupplierVO(
        id=s.id,
        name=s.name,
        contact_person=s.contact_person,
        contact_phone=s.contact_phone,
        email=s.email,
        address=s.address,
        status=s.status,
        remark=s.remark,
        created_at=s.created_at,
        updated_at=s.updated_at,
    )


# ========== 供应商CRUD ==========

@router.post("/suppliers", summary="创建供应商")
async def create_supplier(
    data: SupplierCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("supplier:create")),
):
    s = await SupplierService.create(db, data.model_dump())
    await OperationLogService.log(
        db,
        module="supplier",
        action="create",
        resource_id=str(s.id),
        content=f"创建供应商: {s.name}",
        operator=current_user.username,
    )
    return Result.ok(supplier_to_vo(s))


@router.put("/suppliers/{supplier_id}", summary="更新供应商")
async def update_supplier(
    supplier_id: int,
    data: SupplierUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("supplier:update")),
):
    s = await SupplierService.update(db, supplier_id, data.model_dump(exclude_unset=True))
    if not s:
        return Result.not_found("供应商不存在")
    await OperationLogService.log(
        db,
        module="supplier",
        action="update",
        resource_id=str(supplier_id),
        content=f"更新供应商: {s.name}",
        operator=current_user.username,
    )
    return Result.ok(supplier_to_vo(s))


@router.get("/suppliers/{supplier_id}", summary="供应商详情")
async def get_supplier(
    supplier_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("supplier:view")),
):
    s = await SupplierService.get_by_id(db, supplier_id)
    if not s:
        return Result.not_found("供应商不存在")
    return Result.ok(supplier_to_vo(s))


@router.get("/suppliers", summary="供应商列表")
async def list_suppliers(
    name: str = Query(None, description="供应商名称"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("supplier:view")),
):
    suppliers, total = await SupplierService.list_suppliers(db, name, page, page_size)
    items = [supplier_to_vo(s) for s in suppliers]
    return PageResult.ok(items, total, page, page_size)


@router.delete("/suppliers/{supplier_id}", summary="删除供应商")
async def delete_supplier(
    supplier_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("supplier:delete")),
):
    ok = await SupplierService.delete(db, supplier_id)
    if not ok:
        return Result.not_found("供应商不存在")
    await OperationLogService.log(
        db,
        module="supplier",
        action="delete",
        resource_id=str(supplier_id),
        content=f"删除供应商: {supplier_id}",
        operator=current_user.username,
    )
    return Result.ok(message="删除成功")


# ========== 商品-供应商关联 ==========

@router.post("/product-supplier", summary="绑定商品到供应商")
async def bind_product_supplier(
    data: ProductSupplierBind,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("supplier:create")),
):
    ps = await SupplierService.bind_product(db, data.product_id, data.supplier_id,
                                             data.supply_price, data.min_order_qty)
    await OperationLogService.log(
        db,
        module="supplier",
        action="bind_product",
        resource_id=f"product={data.product_id},supplier={data.supplier_id}",
        content=f"绑定商品到供应商: product_id={data.product_id}, supplier_id={data.supplier_id}",
        operator=current_user.username,
    )
    return Result.ok(ProductSupplierVO(
        id=ps.id,
        product_id=ps.product_id,
        supplier_id=ps.supplier_id,
        supply_price=float(ps.supply_price) if ps.supply_price else None,
        min_order_qty=ps.min_order_qty or 1,
        created_at=ps.created_at,
    ))


@router.get("/products/{product_id}/suppliers", summary="查询商品的供应商列表")
async def get_product_suppliers(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("supplier:view")),
):
    rows = await SupplierService.get_product_suppliers(db, product_id)
    return Result.ok(rows)


@router.delete("/product-supplier", summary="解除商品-供应商绑定")
async def unbind_product_supplier(
    product_id: int = Query(...),
    supplier_id: int = Query(...),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("supplier:delete")),
):
    ok = await SupplierService.unbind_product(db, product_id, supplier_id)
    if not ok:
        return Result.not_found("绑定关系不存在")
    await OperationLogService.log(
        db,
        module="supplier",
        action="unbind_product",
        resource_id=f"product={product_id},supplier={supplier_id}",
        content=f"解除商品-供应商绑定: product_id={product_id}, supplier_id={supplier_id}",
        operator=current_user.username,
    )
    return Result.ok(message="解绑成功")
