"""商品聚合详情 — 服务层"""

from datetime import datetime
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    Product, Sku, Price, Inventory, Brand,
    ProductSupplier, Supplier, ProductListing, Platform,
)
from app.core.service import ProductService, product_to_vo


class ProductDetailService:
    async def get_detail(db: AsyncSession, product_id: int) -> dict:
        """聚合返回商品详情 + SKU列表 + 价格 + 库存 + 供应商 + 发布状态"""
        product = await ProductService.get_by_id(db, product_id)
        if not product:
            raise ValueError("商品不存在")

        vo = product_to_vo(product)

        # 品牌名称
        if product.brand_id:
            brand = await db.get(Brand, product.brand_id)
            if brand:
                vo.brand_name = brand.name

        # SKU列表（带库存，库存从 Inventory 表读取）
        stmt = select(Sku).where(Sku.product_id == product_id)
        skus_result = await db.execute(stmt)
        skus = skus_result.scalars().all()
        # 批量查询库存
        sku_ids = [s.id for s in skus]
        inv_map = {}
        if sku_ids:
            inv_stmt = select(Inventory).where(Inventory.sku_id.in_(sku_ids))
            inv_res = await db.execute(inv_stmt)
            for inv in inv_res.scalars().all():
                inv_map[inv.sku_id] = inv

        sku_list = []
        for sku in skus:
            inv = inv_map.get(sku.id)
            s = {
                "id": sku.id,
                "code": sku.code,
                "barcode": sku.barcode,
                "spec_desc": sku.spec_desc,
                "price": float(sku.price) if sku.price else None,
                "market_price": float(sku.market_price) if sku.market_price else None,
                "stock": inv.quantity if inv else 0,
                "image": sku.image,
                "status": sku.status,
            }
            # 获取每个SKU的当前售价（按有效期过滤）
            now = datetime.utcnow()
            price_stmt = select(Price).where(
                Price.sku_id == sku.id,
                Price.price_type == "sale_price",
                Price.status == 1,
                (Price.start_time.is_(None) | (Price.start_time <= now)),
                (Price.end_time.is_(None) | (Price.end_time > now)),
            ).order_by(Price.created_at.desc()).limit(1)
            price_res = await db.execute(price_stmt)
            current_price = price_res.scalar_one_or_none()
            if current_price:
                s["sale_price"] = float(current_price.price)
            sku_list.append(s)

        # 库存信息（复用上面 inv_map 的查询结果）
        inv_list = []
        for inv in inv_map.values():
            inv_list.append({
                "id": inv.id,
                "sku_id": inv.sku_id,
                "warehouse": inv.warehouse,
                "quantity": inv.quantity,
                "safety_stock": inv.safety_stock,
            })

        # 供应商列表
        ps_stmt = select(ProductSupplier, Supplier.name).join(
            Supplier, ProductSupplier.supplier_id == Supplier.id
        ).where(ProductSupplier.product_id == product_id)
        ps_res = await db.execute(ps_stmt)
        suppliers = []
        for ps, name in ps_res.all():
            suppliers.append({
                "id": ps.id,
                "supplier_id": ps.supplier_id,
                "supplier_name": name,
                "supply_price": float(ps.supply_price) if ps.supply_price else None,
            })

        # 发布状态
        listing_stmt = (
            select(ProductListing, Platform.name, Platform.code)
            .join(Platform, ProductListing.platform_id == Platform.id)
            .where(ProductListing.product_id == product_id)
            .order_by(Platform.sort_order)
        )
        listing_res = await db.execute(listing_stmt)
        listings = []
        for listing, plat_name, plat_code in listing_res.all():
            listings.append({
                "id": listing.id,
                "platform_id": listing.platform_id,
                "platform_name": plat_name,
                "platform_code": plat_code,
                "platform_product_id": listing.platform_product_id,
                "status": listing.status,
                "platform_url": listing.platform_url,
                "last_sync_at": listing.last_sync_at.isoformat() if listing.last_sync_at else None,
            })

        return {
            "product": vo.model_dump(),
            "skus": sku_list,
            "inventory": inv_list,
            "suppliers": suppliers,
            "listings": listings,
        }
