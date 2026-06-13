"""库存管理 - 服务层"""
from typing import Optional
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Inventory, InventoryLog, Sku


class InventoryService:

    @staticmethod
    async def update_inventory(db: AsyncSession, sku_id: int, quantity: int,
                                warehouse: str = "默认仓库", location: str = None,
                                safety_stock: int = None, remark: str = None,
                                operator: str = "system") -> Inventory:
        """更新库存（设置最终值）"""
        stmt = select(Inventory).where(Inventory.sku_id == sku_id)
        result = await db.execute(stmt)
        inv = result.scalar_one_or_none()

        before_qty = 0
        if inv:
            before_qty = inv.quantity
            inv.quantity = quantity
            if warehouse:
                inv.warehouse = warehouse
            if location:
                inv.location = location
            if safety_stock is not None:
                inv.safety_stock = safety_stock
        else:
            inv = Inventory(
                sku_id=sku_id,
                warehouse=warehouse,
                location=location,
                quantity=quantity,
                safety_stock=safety_stock or 0,
            )
            db.add(inv)

        await db.flush()
        await db.refresh(inv)

        # 同步更新SKU的库存字段
        sku = await db.get(Sku, sku_id)
        if sku:
            sku.stock = quantity

        await db.flush()

        # 记录日志
        log = InventoryLog(
            sku_id=sku_id,
            change_type="adjust",
            change_qty=quantity - before_qty,
            before_qty=before_qty,
            after_qty=quantity,
            remark=remark,
            operator=operator,
        )
        db.add(log)
        await db.flush()

        return inv

    @staticmethod
    async def get_inventory(db: AsyncSession, sku_id: int) -> Optional[Inventory]:
        stmt = select(Inventory).where(Inventory.sku_id == sku_id)
        result = await db.execute(stmt)
        return result.scalar_one_or_none()

    @staticmethod
    async def check_stock(db: AsyncSession, sku_id: int, quantity: int) -> tuple[bool, str]:
        """检查库存是否充足"""
        inv = await InventoryService.get_inventory(db, sku_id)
        if not inv:
            return False, "库存记录不存在"
        if inv.quantity < quantity:
            return False, f"库存不足，当前库存: {inv.quantity}，需要: {quantity}"
        return True, "库存充足"

    @staticmethod
    async def get_inventory_logs(db: AsyncSession, sku_id: int, limit: int = 50) -> list[InventoryLog]:
        stmt = select(InventoryLog).where(
            InventoryLog.sku_id == sku_id
        ).order_by(InventoryLog.created_at.desc()).limit(limit)
        result = await db.execute(stmt)
        return list(result.scalars().all())
