"""商品管理 - 服务层"""

from typing import Optional
from sqlalchemy import select, func, or_
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Product, ProductListing, ProductSupplier, Sku
from app.core.schemas import ProductCreate, ProductUpdate, ProductQuery, ProductVO
from app.common import ProductStatus


def _numeric_to_float(value):
    return float(value) if value is not None else None


def missing_logistics_fields(product: Product) -> list[str]:
    """返回缺失的包装物流字段中文标签列表"""
    missing = []
    if not product.package_length_cm or float(product.package_length_cm) <= 0:
        missing.append("包装长")
    if not product.package_width_cm or float(product.package_width_cm) <= 0:
        missing.append("包装宽")
    if not product.package_height_cm or float(product.package_height_cm) <= 0:
        missing.append("包装高")
    if not product.package_weight_kg or float(product.package_weight_kg) <= 0:
        missing.append("包装重量")
    return missing


def package_volume_weight_kg(product: Product) -> Optional[float]:
    """计算包装体积重 (长*宽*高/6000)"""
    values = [
        product.package_length_cm,
        product.package_width_cm,
        product.package_height_cm,
    ]
    if not all(value is not None and float(value) > 0 for value in values):
        return None
    return round(
        float(product.package_length_cm)
        * float(product.package_width_cm)
        * float(product.package_height_cm)
        / 6000,
        3,
    )


def is_product_logistics_complete(product: Product) -> bool:
    package_values = [
        product.package_length_cm,
        product.package_width_cm,
        product.package_height_cm,
        product.package_weight_kg,
    ]
    return all(value is not None and float(value) > 0 for value in package_values)


def product_to_vo(product) -> ProductVO:
    """商品模型转VO（共享函数，router 和 detail_service 复用）"""
    status_name = ProductStatus.STATUS_MAP.get(product.status, "未知")
    category_name = product.category.name if product.category else None
    logistics_complete = is_product_logistics_complete(product)
    pkg_missing = missing_logistics_fields(product)
    pkg_vol_weight = package_volume_weight_kg(product)
    return ProductVO(
        id=product.id,
        name=product.name,
        subtitle=product.subtitle,
        description=product.description,
        brand_id=product.brand_id,
        category_id=product.category_id,
        category_name=category_name,
        brand_name=None,
        unit=product.unit,
        status=product.status,
        status_name=status_name,
        main_image=product.main_image,
        images=product.images,
        product_length_cm=_numeric_to_float(product.product_length_cm),
        product_width_cm=_numeric_to_float(product.product_width_cm),
        product_height_cm=_numeric_to_float(product.product_height_cm),
        product_weight_kg=_numeric_to_float(product.product_weight_kg),
        package_length_cm=_numeric_to_float(product.package_length_cm),
        package_width_cm=_numeric_to_float(product.package_width_cm),
        package_height_cm=_numeric_to_float(product.package_height_cm),
        package_weight_kg=_numeric_to_float(product.package_weight_kg),
        cargo_type=product.cargo_type or "normal",
        missing_logistics_fields=pkg_missing,
        package_volume_weight_kg=pkg_vol_weight,
        logistics_status="complete" if logistics_complete else "incomplete",
        logistics_status_name="物流完整" if logistics_complete else "物流不完整",
        ai_status=product.ai_status,
        platform_statuses=product.platform_statuses,
        created_at=product.created_at,
        updated_at=product.updated_at,
    )


class ProductService:
    @staticmethod
    async def create(db: AsyncSession, data: ProductCreate) -> Product:
        product = Product(**data.model_dump())
        db.add(product)
        await db.flush()
        await db.refresh(product)
        return product

    @staticmethod
    async def update(
        db: AsyncSession, product_id: int, data: ProductUpdate
    ) -> Optional[Product]:
        product = await db.get(Product, product_id)
        if not product:
            return None
        update_data = data.model_dump(exclude_unset=True)
        for key, value in update_data.items():
            setattr(product, key, value)
        await db.flush()
        await db.refresh(product)
        return product

    @staticmethod
    async def get_by_id(db: AsyncSession, product_id: int) -> Optional[Product]:
        return await db.get(Product, product_id)

    @staticmethod
    async def delete(db: AsyncSession, product_id: int) -> bool:
        product = await db.get(Product, product_id)
        if not product:
            return False
        dependency_checks = [
            (Sku, "商品存在SKU，不能删除"),
            (ProductSupplier, "商品已绑定供应商，不能删除"),
            (ProductListing, "商品存在平台发布记录，不能删除"),
        ]
        for model, message in dependency_checks:
            count_stmt = select(func.count()).where(model.product_id == product_id)
            if (await db.scalar(count_stmt) or 0) > 0:
                raise ValueError(message)
        await db.delete(product)
        await db.flush()
        return True

    @staticmethod
    async def list_products(
        db: AsyncSession, query: ProductQuery
    ) -> tuple[list[Product], int]:
        """分页查询商品列表"""
        stmt = select(Product)

        # 筛选条件
        if query.name:
            stmt = stmt.where(Product.name.like(f"%{query.name}%"))
        if query.category_id:
            stmt = stmt.where(Product.category_id == query.category_id)
        if query.status is not None:
            stmt = stmt.where(Product.status == query.status)
        if query.brand_id:
            stmt = stmt.where(Product.brand_id == query.brand_id)

        if query.cargo_type:
            stmt = stmt.where(Product.cargo_type == query.cargo_type)

        if query.logistics_status == "complete":
            stmt = stmt.where(
                Product.package_length_cm.is_not(None),
                Product.package_width_cm.is_not(None),
                Product.package_height_cm.is_not(None),
                Product.package_weight_kg.is_not(None),
                Product.package_length_cm > 0,
                Product.package_width_cm > 0,
                Product.package_height_cm > 0,
                Product.package_weight_kg > 0,
            )
        elif query.logistics_status == "incomplete":
            stmt = stmt.where(
                or_(
                    Product.package_length_cm.is_(None),
                    Product.package_width_cm.is_(None),
                    Product.package_height_cm.is_(None),
                    Product.package_weight_kg.is_(None),
                    Product.package_length_cm <= 0,
                    Product.package_width_cm <= 0,
                    Product.package_height_cm <= 0,
                    Product.package_weight_kg <= 0,
                )
            )

        # 先查总数
        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        # 分页
        offset = (query.page - 1) * query.page_size
        stmt = (
            stmt.order_by(Product.created_at.desc())
            .offset(offset)
            .limit(query.page_size)
        )

        result = await db.execute(stmt)
        products = result.scalars().all()

        return list(products), total
