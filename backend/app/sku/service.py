"""规格与SKU管理 - 服务层"""
import itertools
from typing import Optional
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload
from app.models import Inventory, Price, SpecName, SpecValue, Sku


class SpecService:

    @staticmethod
    async def define_specs(db: AsyncSession, product_id: int, specs: list[dict]) -> list[SpecName]:
        """定义商品规格模板（先清空旧规格再新建）"""
        # 删除旧规格
        old_specs = await db.execute(
            select(SpecName).where(SpecName.product_id == product_id)
        )
        for spec in old_specs.scalars().all():
            await db.delete(spec)

        # 创建新规格
        created = []
        for sort_idx, spec in enumerate(specs):
            spec_name = SpecName(product_id=product_id, name=spec["name"], sort_order=sort_idx)
            db.add(spec_name)
            await db.flush()
            await db.refresh(spec_name)

            for val_idx, val in enumerate(spec["values"]):
                spec_value = SpecValue(
                    spec_name_id=spec_name.id,
                    product_id=product_id,
                    value=val,
                    sort_order=val_idx,
                )
                db.add(spec_value)

            created.append(spec_name)

        return created

    @staticmethod
    async def get_specs(db: AsyncSession, product_id: int) -> list[SpecName]:
        """获取商品的规格定义"""
        stmt = (
            select(SpecName)
            .where(SpecName.product_id == product_id)
            .options(selectinload(SpecName.values))
            .order_by(SpecName.sort_order)
        )
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def generate_skus(db: AsyncSession, product_id: int) -> list[Sku]:
        """根据规格自动生成SKU（笛卡尔积）"""
        specs = await SpecService.get_specs(db, product_id)
        if not specs:
            return []

        # 构建规格值组合
        spec_groups = []
        spec_names = []
        for spec in specs:
            spec_names.append(spec.name)
            values = [{"spec_name_id": spec.id, "name": spec.name, "value": sv.value} for sv in spec.values]
            spec_groups.append(values)

        # 笛卡尔积生成所有组合
        combinations = list(itertools.product(*spec_groups))

        existing_result = await db.execute(
            select(Sku).where(Sku.product_id == product_id).order_by(Sku.id)
        )
        existing_skus = list(existing_result.scalars().all())
        existing_by_spec = {
            tuple(sorted((sku.spec_values or {}).items())): sku
            for sku in existing_skus
        }

        desired_keys = set()
        skus = []
        for idx, combo in enumerate(combinations):
            spec_desc_parts = []
            spec_values_map = {}
            for item in combo:
                spec_desc_parts.append(f"{item['name']}:{item['value']}")
                spec_values_map[item["name"]] = item["value"]

            spec_desc = "-".join(spec_desc_parts)
            code = f"SKU-{product_id}-{idx + 1:04d}"
            key = tuple(sorted(spec_values_map.items()))
            desired_keys.add(key)

            sku = existing_by_spec.get(key)
            if sku:
                sku.code = sku.code or code
                sku.spec_desc = spec_desc
                sku.status = 1
            else:
                sku = Sku(
                    product_id=product_id,
                    code=code,
                    spec_desc=spec_desc,
                    spec_values=spec_values_map,
                )
                db.add(sku)
            skus.append(sku)

        for sku in existing_skus:
            key = tuple(sorted((sku.spec_values or {}).items()))
            if key in desired_keys:
                continue
            has_inventory = await db.scalar(
                select(Inventory.id).where(Inventory.sku_id == sku.id).limit(1)
            )
            has_price = await db.scalar(
                select(Price.id).where(Price.sku_id == sku.id).limit(1)
            )
            if has_inventory or has_price:
                sku.status = 0
            else:
                await db.delete(sku)

        await db.flush()
        return skus

    @staticmethod
    async def update_sku(db: AsyncSession, sku_id: int, data: dict) -> Optional[Sku]:
        sku = await db.get(Sku, sku_id)
        if not sku:
            return None
        for key, value in data.items():
            if value is not None:
                setattr(sku, key, value)
        await db.flush()
        await db.refresh(sku)
        return sku

    @staticmethod
    async def get_skus_by_product(db: AsyncSession, product_id: int) -> list[Sku]:
        stmt = select(Sku).where(Sku.product_id == product_id).order_by(Sku.id)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def get_sku_by_id(db: AsyncSession, sku_id: int) -> Optional[Sku]:
        return await db.get(Sku, sku_id)
