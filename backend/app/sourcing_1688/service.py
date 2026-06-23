"""1688 货源采集 - 服务层"""

from typing import Optional
from decimal import Decimal
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import (
    Sourcing1688Product,
    Product,
    Supplier,
    ProductSupplier,
    Sku,
)
from app.sourcing_1688.schemas import CollectPayload, ImportPayload


class Sourcing1688Service:
    @staticmethod
    async def collect(
        db: AsyncSession, data: CollectPayload, username: str
    ) -> Sourcing1688Product:
        """采集/更新 1688 商品到候选池（按 source_url upsert）"""
        # 按 source_url 查找已有记录
        stmt = select(Sourcing1688Product).where(
            Sourcing1688Product.source_url == data.url
        )
        result = await db.execute(stmt)
        existing = result.scalar_one_or_none()

        # 字段映射
        weight_kg = None
        if data.weight_g is not None:
            weight_kg = Decimal(str(data.weight_g)) / Decimal("1000")

        fields = {
            "title": data.title,
            "price": Decimal(str(data.price)) if data.price is not None else None,
            "moq": data.moq,
            "supplier_name": data.supplier,
            "shop_url": data.shop_url,
            "shop_location": data.shop_location,
            "images": data.images,
            "attributes": data.attributes,
            "sku_variants": data.skuVariants,
            "description": data.description,
            "package_length_cm": Decimal(str(data.length_cm))
            if data.length_cm is not None
            else None,
            "package_width_cm": Decimal(str(data.width_cm))
            if data.width_cm is not None
            else None,
            "package_height_cm": Decimal(str(data.height_cm))
            if data.height_cm is not None
            else None,
            "package_weight_kg": weight_kg,
            "raw_data": data.model_dump(),
            "collected_by": username,
        }

        if existing:
            # 更新已有记录（不覆盖 imported 的关联 ID）
            for key, value in fields.items():
                setattr(existing, key, value)
            if existing.status in ("imported", "rejected"):
                existing.status = "collected"
                existing.product_id = None
                existing.supplier_id = None
                existing.imported_by = None
                existing.imported_at = None
            product = existing
        else:
            product = Sourcing1688Product(
                source_url=data.url,
                status="collected",
                **fields,
            )
            db.add(product)

        await db.flush()
        await db.refresh(product)
        return product

    @staticmethod
    async def list_products(
        db: AsyncSession,
        status: Optional[str] = None,
        keyword: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[Sourcing1688Product], int]:
        """分页查询候选池"""
        stmt = select(Sourcing1688Product)
        count_stmt = select(func.count()).select_from(Sourcing1688Product)

        if status:
            stmt = stmt.where(Sourcing1688Product.status == status)
            count_stmt = count_stmt.where(Sourcing1688Product.status == status)
        if keyword:
            like = f"%{keyword}%"
            stmt = stmt.where(
                Sourcing1688Product.title.ilike(like)
                | Sourcing1688Product.supplier_name.ilike(like)
            )
            count_stmt = count_stmt.where(
                Sourcing1688Product.title.ilike(like)
                | Sourcing1688Product.supplier_name.ilike(like)
            )

        total = await db.scalar(count_stmt) or 0
        offset = (page - 1) * page_size
        stmt = (
            stmt.order_by(Sourcing1688Product.created_at.desc())
            .offset(offset)
            .limit(page_size)
        )
        result = await db.execute(stmt)
        return list(result.scalars().all()), total

    @staticmethod
    async def get_product(
        db: AsyncSession, product_id: int
    ) -> Optional[Sourcing1688Product]:
        """获取候选商品详情"""
        return await db.get(Sourcing1688Product, product_id)

    @staticmethod
    async def import_product(
        db: AsyncSession,
        candidate_id: int,
        payload: ImportPayload,
        username: str,
    ) -> Sourcing1688Product:
        """将候选商品导入为正式商品/供应商/SKU"""
        candidate = await db.get(Sourcing1688Product, candidate_id)
        if not candidate:
            raise ValueError("候选商品不存在")
        if candidate.status == "imported":
            raise ValueError("该商品已导入，请勿重复操作")

        # 1. 创建 Product
        main_image = candidate.images[0] if candidate.images else None
        product = Product(
            name=candidate.title or "",
            description=candidate.description,
            main_image=main_image,
            images=candidate.images,
            package_length_cm=candidate.package_length_cm,
            package_width_cm=candidate.package_width_cm,
            package_height_cm=candidate.package_height_cm,
            package_weight_kg=candidate.package_weight_kg,
            category_id=payload.category_id,
            brand_id=payload.brand_id,
            cargo_type=payload.cargo_type or "normal",
            unit=payload.unit or "件",
            status=0,  # 草稿
        )
        db.add(product)
        await db.flush()
        await db.refresh(product)

        # 2. Supplier 匹配/创建
        supplier = None
        if candidate.supplier_name:
            stmt = select(Supplier).where(Supplier.name == candidate.supplier_name)
            result = await db.execute(stmt)
            supplier = result.scalar_one_or_none()
            if not supplier:
                supplier = Supplier(
                    name=candidate.supplier_name,
                    address=candidate.shop_location,
                    remark=f"1688 店铺: {candidate.shop_url or ''}",
                )
                db.add(supplier)
                await db.flush()
                await db.refresh(supplier)

        # 3. ProductSupplier 绑定
        if supplier:
            ps = ProductSupplier(
                product_id=product.id,
                supplier_id=supplier.id,
                supply_price=candidate.price,
                min_order_qty=candidate.moq or 1,
            )
            db.add(ps)

        # 4. SKU 创建
        if candidate.sku_variants and len(candidate.sku_variants) > 0:
            for variant in candidate.sku_variants:
                spec = variant.get("spec", "")
                spec_values = {}
                if ":" in spec:
                    try:
                        parts = spec.split(";")
                        for part in parts:
                            if ":" in part:
                                k, v = part.split(":", 1)
                                spec_values[k.strip()] = v.strip()
                    except Exception:
                        spec_values = {"规格": spec}
                else:
                    spec_values = {"规格": spec}

                cost_price = variant.get("price")
                sku = Sku(
                    product_id=product.id,
                    spec_desc=spec,
                    spec_values=spec_values if spec_values else None,
                    cost_price=Decimal(str(cost_price))
                    if cost_price is not None
                    else None,
                    stock=variant.get("stock", 0) or 0,
                )
                db.add(sku)
        else:
            # 无变体：创建默认 SKU
            sku = Sku(
                product_id=product.id,
                spec_desc="默认",
                cost_price=candidate.price,
            )
            db.add(sku)

        # 5. 更新候选状态
        candidate.status = "imported"
        candidate.product_id = product.id
        if supplier:
            candidate.supplier_id = supplier.id
        candidate.imported_by = username
        candidate.imported_at = func.now()

        await db.flush()
        await db.refresh(candidate)
        return candidate

    @staticmethod
    async def reject_product(
        db: AsyncSession, candidate_id: int
    ) -> Optional[Sourcing1688Product]:
        """驳回候选商品"""
        candidate = await db.get(Sourcing1688Product, candidate_id)
        if not candidate:
            return None
        candidate.status = "rejected"
        await db.flush()
        await db.refresh(candidate)
        return candidate
