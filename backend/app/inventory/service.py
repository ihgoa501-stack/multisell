"""库存管理 - 服务层"""
from typing import Optional
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Inventory, InventoryLog


class InventoryService:

    @staticmethod
    async def _get_inventory_for_update(db: AsyncSession, sku_id: int) -> Optional[Inventory]:
        stmt = select(Inventory).where(Inventory.sku_id == sku_id).with_for_update()
        result = await db.execute(stmt)
        return result.scalar_one_or_none()

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
            if (inv.locked_quantity or 0) > quantity:
                inv.locked_quantity = quantity
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
                locked_quantity=0,
                safety_stock=safety_stock or 0,
            )
            db.add(inv)

        await db.flush()
        await db.refresh(inv)

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
        """检查库存是否充足（考虑锁定库存）"""
        inv = await InventoryService.get_inventory(db, sku_id)
        if not inv:
            return False, "库存记录不存在"
        available = (inv.quantity or 0) - (inv.locked_quantity or 0)
        if available < quantity:
            return False, f"库存不足，可用库存: {available}，需要: {quantity}"
        return True, "库存充足"

    @staticmethod
    async def _add_log(
        db: AsyncSession,
        sku_id: int,
        change_type: str,
        change_qty: int,
        before_qty: int,
        after_qty: int,
        remark: str,
        operator: str,
    ) -> None:
        db.add(InventoryLog(
            sku_id=sku_id,
            change_type=change_type,
            change_qty=change_qty,
            before_qty=before_qty,
            after_qty=after_qty,
            remark=remark,
            operator=operator,
        ))
        await db.flush()

    @staticmethod
    async def lock_stock(
        db: AsyncSession,
        sku_id: int,
        quantity: int,
        order_no: str,
        operator: str = "system",
    ) -> Inventory:
        inv = await InventoryService._get_inventory_for_update(db, sku_id)
        if not inv:
            raise ValueError("库存记录不存在")
        available = (inv.quantity or 0) - (inv.locked_quantity or 0)
        if available < quantity:
            raise ValueError(f"库存不足，可用库存: {available}，需要: {quantity}")
        before_locked = inv.locked_quantity or 0
        inv.locked_quantity = before_locked + quantity
        await db.flush()
        await InventoryService._add_log(
            db, sku_id, "lock", quantity, before_locked, inv.locked_quantity,
            f"订单锁定库存: {order_no}", operator,
        )
        return inv

    @staticmethod
    async def release_locked_stock(
        db: AsyncSession,
        sku_id: int,
        quantity: int,
        order_no: str,
        operator: str = "system",
    ) -> Inventory:
        inv = await InventoryService._get_inventory_for_update(db, sku_id)
        if not inv:
            raise ValueError("库存记录不存在")
        before_locked = inv.locked_quantity or 0
        if before_locked < quantity:
            raise ValueError(f"锁定库存不足，当前锁定: {before_locked}，需要释放: {quantity}")
        inv.locked_quantity = before_locked - quantity
        await db.flush()
        await InventoryService._add_log(
            db, sku_id, "release", -quantity, before_locked, inv.locked_quantity,
            f"订单释放锁定库存: {order_no}", operator,
        )
        return inv

    @staticmethod
    async def confirm_locked_stock_deduction(
        db: AsyncSession,
        sku_id: int,
        quantity: int,
        order_no: str,
        operator: str = "system",
    ) -> Inventory:
        inv = await InventoryService._get_inventory_for_update(db, sku_id)
        if not inv:
            raise ValueError("库存记录不存在")
        before_qty = inv.quantity or 0
        before_locked = inv.locked_quantity or 0
        if before_locked < quantity:
            raise ValueError(f"锁定库存不足，当前锁定: {before_locked}，需要扣减: {quantity}")
        if before_qty < quantity:
            raise ValueError(f"库存不足，当前库存: {before_qty}，需要扣减: {quantity}")
        inv.quantity = before_qty - quantity
        inv.locked_quantity = before_locked - quantity
        await db.flush()
        await InventoryService._add_log(
            db, sku_id, "deduct", -quantity, before_qty, inv.quantity,
            f"订单支付扣减库存: {order_no}", operator,
        )
        return inv

    @staticmethod
    async def restock(
        db: AsyncSession,
        sku_id: int,
        quantity: int,
        remark: str = None,
        operator: str = "system",
    ) -> Inventory:
        """退货入库：增加库存数量并记录日志。"""
        inv = await InventoryService._get_inventory_for_update(db, sku_id)
        if not inv:
            raise ValueError("库存记录不存在")
        before_qty = inv.quantity or 0
        inv.quantity = before_qty + quantity
        await db.flush()
        await InventoryService._add_log(
            db, sku_id, "in", quantity, before_qty, inv.quantity,
            remark or f"退货入库: +{quantity}", operator,
        )
        return inv

    @staticmethod
    async def get_inventory_logs(db: AsyncSession, sku_id: int, limit: int = 50) -> list[InventoryLog]:
        stmt = select(InventoryLog).where(
            InventoryLog.sku_id == sku_id
        ).order_by(InventoryLog.created_at.desc()).limit(limit)
        result = await db.execute(stmt)
        return list(result.scalars().all())
