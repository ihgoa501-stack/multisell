"""库存管理 - 路由"""

import asyncio
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.common import Result
from app.inventory.schemas import InventoryUpdate, InventoryCheck, InventoryVO, InventoryLogVO
from app.inventory.service import InventoryService
from app.models import User
from app.operation_log.service import OperationLogService
from app.events import emit_agent_event

router = APIRouter(tags=["库存管理"])


@router.get("/inventory/alerts", summary="库存预警列表")
async def get_inventory_alerts(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:view")),
):
    from sqlalchemy import select, and_
    from app.models import Inventory, Sku, Product

    stmt = (
        select(Inventory, Sku, Product.name)
        .join(Sku, Inventory.sku_id == Sku.id)
        .join(Product, Sku.product_id == Product.id)
        .where(and_(Inventory.safety_stock > 0, Inventory.quantity <= Inventory.safety_stock))
        .order_by(Inventory.quantity.asc())
    )
    result = await db.execute(stmt)
    alerts = []
    for inv, sku, product_name in result.all():
        alerts.append({
            "sku_id": inv.sku_id,
            "sku_code": sku.code,
            "spec_desc": sku.spec_desc,
            "product_name": product_name,
            "product_id": sku.product_id,
            "quantity": inv.quantity,
            "safety_stock": inv.safety_stock,
            "warehouse": inv.warehouse,
            "updated_at": inv.updated_at.isoformat() if inv.updated_at else None,
        })
    return Result.ok(alerts)


def inv_to_vo(inv) -> InventoryVO:
    locked = inv.locked_quantity or 0
    quantity = inv.quantity or 0
    return InventoryVO(
        id=inv.id,
        sku_id=inv.sku_id,
        warehouse=inv.warehouse,
        location=inv.location,
        quantity=quantity,
        locked_quantity=locked,
        available_quantity=quantity - locked,
        safety_stock=inv.safety_stock,
        created_at=inv.created_at,
        updated_at=inv.updated_at,
    )


def log_to_vo(log) -> InventoryLogVO:
    return InventoryLogVO(
        id=log.id,
        sku_id=log.sku_id,
        change_type=log.change_type,
        change_qty=log.change_qty,
        before_qty=log.before_qty,
        after_qty=log.after_qty,
        remark=log.remark,
        operator=log.operator,
        created_at=log.created_at,
    )


@router.put("/inventory/{sku_id}", summary="更新库存")
async def update_inventory(
    sku_id: int,
    data: InventoryUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    inv = await InventoryService.update_inventory(
        db, sku_id, data.quantity, data.warehouse or "默认仓库",
        data.location, data.safety_stock, data.remark
    )
    await OperationLogService.log(
        db,
        module="inventory",
        action="update",
        resource_id=str(sku_id),
        content=f"更新库存: sku_id={sku_id}, quantity={data.quantity}",
        operator=current_user.username,
    )

    # Agent 事件：库存异常通知
    if data.quantity is not None or data.safety_stock is not None:
        qty = data.quantity if data.quantity is not None else (inv.quantity if inv else 0)
        safe = data.safety_stock if data.safety_stock is not None else (inv.safety_stock if inv else 0)
        if safe > 0 and qty <= 0:
            asyncio.ensure_future(emit_agent_event("inventory.out_of_stock", {
                "sku_id": sku_id,
                "current_stock": qty,
                "safety_stock": safe,
            }, source="inventory.router"))
        elif safe > 0 and qty <= safe:
            asyncio.ensure_future(emit_agent_event("inventory.low_stock", {
                "sku_id": sku_id,
                "current_stock": qty,
                "safety_stock": safe,
            }, source="inventory.router"))

    return Result.ok(inv_to_vo(inv))


@router.get("/inventory/{sku_id}", summary="查询库存")
async def get_inventory(
    sku_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:view")),
):
    inv = await InventoryService.get_inventory(db, sku_id)
    if not inv:
        return Result.ok(InventoryVO(sku_id=sku_id, quantity=0, locked_quantity=0, available_quantity=0))
    return Result.ok(inv_to_vo(inv))


@router.post("/inventory/check", summary="库存预占检查")
async def check_inventory(
    data: InventoryCheck,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:view")),
):
    ok, message = await InventoryService.check_stock(db, data.sku_id, data.quantity)
    if not ok:
        return Result.error(message=message)
    return Result.ok({"sku_id": data.sku_id, "available": True})


@router.get("/inventory/{sku_id}/logs", summary="库存变动记录")
async def get_inventory_logs(
    sku_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:view")),
):
    logs = await InventoryService.get_inventory_logs(db, sku_id)
    return Result.ok([log_to_vo(l) for l in logs])
