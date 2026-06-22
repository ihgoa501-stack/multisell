"""库存分配 - 服务层

管理多仓库、库存分配与分配规则。
"""

import logging
from typing import Optional

from sqlalchemy import select, func, delete as sa_delete
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    Sku, Product, Inventory,
    Warehouse, AllocationRule, InventoryWarehouse,
)

logger = logging.getLogger(__name__)


class WarehouseService:

    @staticmethod
    async def create(db: AsyncSession, data: dict) -> Warehouse:
        warehouse = Warehouse(**data)
        db.add(warehouse)
        await db.flush()
        await db.refresh(warehouse)
        return warehouse

    @staticmethod
    async def update(db: AsyncSession, warehouse_id: int, data: dict) -> Optional[Warehouse]:
        wh = await db.get(Warehouse, warehouse_id)
        if not wh:
            return None
        for k, v in data.items():
            if v is not None:
                setattr(wh, k, v)
        await db.flush()
        await db.refresh(wh)
        return wh

    @staticmethod
    async def delete(db: AsyncSession, warehouse_id: int) -> bool:
        wh = await db.get(Warehouse, warehouse_id)
        if not wh:
            return False
        await db.delete(wh)
        await db.flush()
        return True

    @staticmethod
    async def list_all(db: AsyncSession) -> list[Warehouse]:
        result = await db.execute(
            select(Warehouse).where(Warehouse.status == 1).order_by(Warehouse.id)
        )
        return list(result.scalars().all())

    @staticmethod
    async def get_by_id(db: AsyncSession, warehouse_id: int) -> Optional[Warehouse]:
        return await db.get(Warehouse, warehouse_id)


class AllocationService:

    @staticmethod
    async def create_rule(db: AsyncSession, data: dict) -> AllocationRule:
        rule = AllocationRule(**data)
        db.add(rule)
        await db.flush()
        await db.refresh(rule)
        return rule

    @staticmethod
    async def list_rules(db: AsyncSession) -> list[AllocationRule]:
        result = await db.execute(
            select(AllocationRule).order_by(AllocationRule.priority, AllocationRule.id)
        )
        rules = result.scalars().all()

        # 补充仓库名称
        for r in rules:
            if r.warehouse_id:
                wh = await db.get(Warehouse, r.warehouse_id)
                r.warehouse_name = wh.name if wh else None
        return list(rules)

    @staticmethod
    async def delete_rule(db: AsyncSession, rule_id: int) -> bool:
        rule = await db.get(AllocationRule, rule_id)
        if not rule:
            return False
        await db.delete(rule)
        await db.flush()
        return True

    @staticmethod
    async def get_warehouse_inventory(
        db: AsyncSession, sku_id: int,
    ) -> list[dict]:
        """查询SKU在各仓库的库存分布"""
        stmt = select(InventoryWarehouse).where(
            InventoryWarehouse.sku_id == sku_id
        )
        result = await db.execute(stmt)
        records = result.scalars().all()

        rows = []
        for r in records:
            wh = await db.get(Warehouse, r.warehouse_id)
            rows.append({
                "id": r.id,
                "sku_id": r.sku_id,
                "warehouse_id": r.warehouse_id,
                "warehouse_name": wh.name if wh else "未知仓库",
                "quantity": r.quantity,
                "locked_quantity": r.locked_quantity,
                "safety_stock": r.safety_stock,
                "available_qty": max(r.quantity - r.locked_quantity, 0),
            })
        return rows

    @staticmethod
    async def allocate(
        db: AsyncSession, sku_id: int, warehouse_id: int, quantity: int,
    ) -> dict:
        """将库存从总库存分配到指定仓库"""
        # 获取总库存
        inv = await db.execute(
            select(Inventory).where(Inventory.sku_id == sku_id)
        )
        total_inv = inv.scalar_one_or_none()
        if not total_inv:
            raise ValueError("SKU库存不存在")

        available = total_inv.quantity - total_inv.locked_quantity
        if quantity > available:
            raise ValueError(f"可用库存不足: 需要{quantity}, 可用{available}")

        # 获取仓库库存记录
        stmt = select(InventoryWarehouse).where(
            InventoryWarehouse.sku_id == sku_id,
            InventoryWarehouse.warehouse_id == warehouse_id,
        )
        result = await db.execute(stmt)
        wh_inv = result.scalar_one_or_none()

        if wh_inv:
            wh_inv.quantity += quantity
        else:
            wh_inv = InventoryWarehouse(
                sku_id=sku_id,
                warehouse_id=warehouse_id,
                quantity=quantity,
            )
            db.add(wh_inv)

        # 扣减总库存
        total_inv.quantity -= quantity
        await db.flush()

        wh = await db.get(Warehouse, warehouse_id)
        sku = await db.get(Sku, sku_id)

        return {
            "sku_id": sku_id,
            "sku_code": sku.code if sku else None,
            "warehouse_id": warehouse_id,
            "warehouse_name": wh.name if wh else None,
            "allocated_qty": quantity,
            "remaining_total": total_inv.quantity,
            "warehouse_quantity": wh_inv.quantity,
        }

    @staticmethod
    async def auto_allocate(db: AsyncSession, sku_id: int) -> list[dict]:
        """按分配规则自动分配到各仓库"""
        rules = await db.execute(
            select(AllocationRule)
            .where(AllocationRule.status == 1)
            .order_by(AllocationRule.priority, AllocationRule.id)
        )
        rules = rules.scalars().all()

        inv = await db.execute(
            select(Inventory).where(Inventory.sku_id == sku_id)
        )
        total_inv = inv.scalar_one_or_none()
        if not total_inv:
            raise ValueError("SKU库存不存在")

        available = total_inv.quantity - total_inv.locked_quantity
        if available <= 0:
            return []

        results = []
        remaining = available

        for rule in rules:
            if remaining <= 0:
                break

            if rule.rule_type == "percentage":
                qty = int(available * float(rule.allocation_pct) / 100)
            elif rule.rule_type == "fixed":
                qty = min(rule.allocation_qty, remaining)
            else:  # priority
                qty = remaining

            if qty > 0:
                result = await AllocationService.allocate(
                    db, sku_id, rule.warehouse_id, qty
                )
                results.append(result)
                remaining -= qty

        return results

    @staticmethod
    async def generate_mock_data(db: AsyncSession) -> dict:
        """生成模拟仓库和分配规则"""
        import random

        mock_warehouses = [
            ("深圳保税仓", "sz-bonded", "深圳市南山区"),
            ("广州中心仓", "gz-central", "广州市白云区"),
            ("义乌分拣仓", "yw-sorting", "义乌市稠江街道"),
            ("海外仓-莫斯科", "ru-moscow", "Moscow, Russia"),
            ("海外仓-曼谷", "th-bangkok", "Bangkok, Thailand"),
        ]

        warehouses = []
        for i, (name, code, addr) in enumerate(mock_warehouses):
            existing = await db.execute(
                select(Warehouse).where(Warehouse.code == code)
            )
            existing_wh = existing.scalar_one_or_none()
            if not existing_wh:
                wh = Warehouse(
                    name=name, code=code, address=addr,
                    is_default=1 if i == 0 else 0,
                )
                db.add(wh)
                await db.flush()
                warehouses.append(wh)
            else:
                warehouses.append(existing_wh)

        # 创建默认分配规则
        rules_data = [
            ("优先本地仓", 1, "percentage", warehouses[0].id, 60),
            ("广州分仓", 2, "percentage", warehouses[1].id, 30),
            ("海外备货", 3, "percentage", warehouses[3].id, 10),
        ]

        created_rules = 0
        for name, pri, rtype, wid, pct in rules_data:
            rule_check = await db.execute(
                select(AllocationRule).where(
                    AllocationRule.name == name,
                    AllocationRule.warehouse_id == wid,
                )
            )
            if not rule_check.scalar_one_or_none():
                rule = AllocationRule(
                    name=name, priority=pri, rule_type=rtype,
                    warehouse_id=wid, allocation_pct=pct,
                )
                db.add(rule)
                created_rules += 1

        await db.flush()

        # 对现有SKU执行一轮自动分配
        sku_result = await db.execute(select(Sku).limit(10))
        skus = sku_result.scalars().all()
        allocated_skus = 0
        for sku in skus:
            try:
                await AllocationService.auto_allocate(db, sku.id)
                allocated_skus += 1
            except ValueError:
                pass

        return {
            "warehouses_created": len(warehouses),
            "rules_created": created_rules,
            "skus_allocated": allocated_skus,
        }
