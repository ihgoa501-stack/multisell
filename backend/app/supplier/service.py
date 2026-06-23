"""供应商管理 - 服务层"""

from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Supplier, ProductSupplier


class SupplierService:
    @staticmethod
    async def create(db: AsyncSession, data: dict) -> Supplier:
        supplier = Supplier(**data)
        db.add(supplier)
        await db.flush()
        await db.refresh(supplier)
        return supplier

    @staticmethod
    async def update(
        db: AsyncSession, supplier_id: int, data: dict
    ) -> Optional[Supplier]:
        supplier = await db.get(Supplier, supplier_id)
        if not supplier:
            return None
        for key, value in data.items():
            if value is not None:
                setattr(supplier, key, value)
        await db.flush()
        await db.refresh(supplier)
        return supplier

    @staticmethod
    async def get_by_id(db: AsyncSession, supplier_id: int) -> Optional[Supplier]:
        return await db.get(Supplier, supplier_id)

    @staticmethod
    async def list_suppliers(
        db: AsyncSession, name: str = None, page: int = 1, page_size: int = 20
    ) -> tuple[list[Supplier], int]:
        stmt = select(Supplier)
        if name:
            stmt = stmt.where(Supplier.name.like(f"%{name}%"))

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(Supplier.created_at.desc()).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        return list(result.scalars().all()), total

    @staticmethod
    async def delete(db: AsyncSession, supplier_id: int) -> bool:
        supplier = await db.get(Supplier, supplier_id)
        if not supplier:
            return False
        await db.delete(supplier)
        await db.flush()
        return True

    @staticmethod
    async def bind_product(
        db: AsyncSession,
        product_id: int,
        supplier_id: int,
        supply_price: float = None,
        min_order_qty: int = 1,
    ) -> ProductSupplier:
        """绑定商品到供应商"""
        # 检查是否已绑定
        stmt = select(ProductSupplier).where(
            ProductSupplier.product_id == product_id,
            ProductSupplier.supplier_id == supplier_id,
        )
        result = await db.execute(stmt)
        existing = result.scalar_one_or_none()

        if existing:
            existing.supply_price = supply_price
            existing.min_order_qty = min_order_qty
            ps = existing
        else:
            ps = ProductSupplier(
                product_id=product_id,
                supplier_id=supplier_id,
                supply_price=supply_price,
                min_order_qty=min_order_qty,
            )
            db.add(ps)

        await db.flush()
        await db.refresh(ps)
        return ps

    @staticmethod
    async def get_product_suppliers(db: AsyncSession, product_id: int) -> list[dict]:
        """获取商品的供应商列表（含供应商名称）"""
        stmt = (
            select(ProductSupplier, Supplier.name)
            .join(Supplier, ProductSupplier.supplier_id == Supplier.id)
            .where(ProductSupplier.product_id == product_id)
        )
        result = await db.execute(stmt)
        rows = []
        for ps, supplier_name in result.all():
            rows.append(
                {
                    "id": ps.id,
                    "product_id": ps.product_id,
                    "supplier_id": ps.supplier_id,
                    "supplier_name": supplier_name,
                    "supply_price": float(ps.supply_price) if ps.supply_price else None,
                    "min_order_qty": ps.min_order_qty or 1,
                    "created_at": ps.created_at,
                }
            )
        return rows

    @staticmethod
    async def unbind_product(
        db: AsyncSession, product_id: int, supplier_id: int
    ) -> bool:
        stmt = select(ProductSupplier).where(
            ProductSupplier.product_id == product_id,
            ProductSupplier.supplier_id == supplier_id,
        )
        result = await db.execute(stmt)
        ps = result.scalar_one_or_none()
        if not ps:
            return False
        await db.delete(ps)
        await db.flush()
        return True
