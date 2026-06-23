"""库存分配 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.allocation.schemas import (
    WarehouseCreate, WarehouseUpdate, WarehouseVO,
    AllocationRuleCreate, AllocationRuleVO,
    InventoryAllocateRequest,
)
from app.allocation.service import WarehouseService, AllocationService
from app.operation_log.service import OperationLogService

router = APIRouter(tags=["库存分配"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


# ── 仓库 ────────────────────────────────────────────────────────


@router.post("/warehouses", summary="创建仓库")
async def create_warehouse(
    data: WarehouseCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    wh = await WarehouseService.create(db, data.model_dump())
    await OperationLogService.log(db, module="allocation", action="create_warehouse",
                                   resource_id=str(wh.id), content=f"创建仓库: {wh.name}",
                                   operator=_operator(current_user))
    return Result.ok({"id": wh.id, "name": wh.name, "code": wh.code})


@router.get("/warehouses", summary="仓库列表")
async def list_warehouses(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:view")),
):
    warehouses = await WarehouseService.list_all(db)
    return Result.ok([WarehouseVO.model_validate(w).model_dump() for w in warehouses])


@router.put("/warehouses/{warehouse_id}", summary="更新仓库")
async def update_warehouse(
    warehouse_id: int, data: WarehouseUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    wh = await WarehouseService.update(db, warehouse_id, data.model_dump(exclude_unset=True))
    if not wh:
        return Result.not_found("仓库不存在")
    return Result.ok({"id": wh.id, "name": wh.name})


@router.delete("/warehouses/{warehouse_id}", summary="删除仓库")
async def delete_warehouse(
    warehouse_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    ok = await WarehouseService.delete(db, warehouse_id)
    return Result.ok(message="删除成功") if ok else Result.not_found("仓库不存在")


# ── 分配规则 ────────────────────────────────────────────────────


@router.post("/allocation-rules", summary="创建分配规则")
async def create_allocation_rule(
    data: AllocationRuleCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    rule = await AllocationService.create_rule(db, data.model_dump())
    return Result.ok({"id": rule.id, "name": rule.name})


@router.get("/allocation-rules", summary="分配规则列表")
async def list_allocation_rules(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:view")),
):
    rules = await AllocationService.list_rules(db)
    return Result.ok([AllocationRuleVO.model_validate(r).model_dump() for r in rules])


@router.delete("/allocation-rules/{rule_id}", summary="删除分配规则")
async def delete_allocation_rule(
    rule_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    ok = await AllocationService.delete_rule(db, rule_id)
    return Result.ok(message="删除成功") if ok else Result.not_found("规则不存在")


# ── 库存分配 ────────────────────────────────────────────────────


@router.get("/inventory/warehouse/{sku_id}", summary="SKU在各仓库的库存")
async def get_warehouse_inventory(
    sku_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:view")),
):
    rows = await AllocationService.get_warehouse_inventory(db, sku_id)
    return Result.ok(rows)


@router.post("/inventory/allocate", summary="分配库存到仓库")
async def allocate_inventory(
    data: InventoryAllocateRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    try:
        result = await AllocationService.allocate(
            db, data.sku_id, data.warehouse_id, data.quantity
        )
    except ValueError as e:
        return Result.bad_request(str(e))
    await OperationLogService.log(db, module="allocation", action="allocate",
                                   resource_id=f"sku={data.sku_id}",
                                   content=f"分配库存: SKU={data.sku_id} 仓库={result['warehouse_name']} 数量={data.quantity}",
                                   operator=_operator(current_user))
    return Result.ok(result)


@router.post("/inventory/auto-allocate/{sku_id}", summary="按规则自动分配库存")
async def auto_allocate(
    sku_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    try:
        results = await AllocationService.auto_allocate(db, sku_id)
    except ValueError as e:
        return Result.bad_request(str(e))
    return Result.ok({"sku_id": sku_id, "allocations": results})


# ── 模拟数据 ────────────────────────────────────────────────────


@router.post("/warehouses/mock", summary="生成模拟仓库和分配数据")
async def generate_mock_warehouses(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("inventory:update")),
):
    result = await AllocationService.generate_mock_data(db)
    return Result.ok(result)
