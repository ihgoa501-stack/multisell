"""发布管理 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result
from app.models import ProductListing, Product, Platform

router = APIRouter(tags=["发布管理"])


@router.post("/products/{product_id}/publish/{platform_id}", summary="发布商品到平台")
async def publish_to_platform(product_id: int, platform_id: int, db: AsyncSession = Depends(get_db)):
    """模拟发布商品到指定平台"""
    # 检查商品和平台是否存在
    product = await db.get(Product, product_id)
    if not product:
        return Result.not_found("商品不存在")

    platform = await db.get(Platform, platform_id)
    if not platform:
        return Result.not_found("平台不存在")

    # 检查是否已有发布记录
    stmt = select(ProductListing).where(
        ProductListing.product_id == product_id,
        ProductListing.platform_id == platform_id,
    )
    result = await db.execute(stmt)
    listing = result.scalar_one_or_none()

    if listing:
        # 更新已有记录
        listing.status = "synced"
        listing.sync_message = None
    else:
        # 新建发布记录
        listing = ProductListing(
            product_id=product_id,
            platform_id=platform_id,
            platform_product_id=f"{platform.code}_{product_id}_{product.name[:10]}",
            status="synced",
            platform_url=f"https://{platform.code}.example.com/product/{product_id}",
            published_data={
                "name": product.name,
                "description": product.description or product.ai_description,
            },
        )
        db.add(listing)

    # 更新商品的多平台状态
    if not product.platform_statuses:
        product.platform_statuses = {}
    product.platform_statuses[str(platform_id)] = "synced"

    await db.flush()
    await db.refresh(listing)

    return Result.ok({
        "id": listing.id,
        "product_id": listing.product_id,
        "platform_id": listing.platform_id,
        "platform_product_id": listing.platform_product_id,
        "status": listing.status,
        "platform_url": listing.platform_url,
    })


@router.get("/products/{product_id}/listings", summary="查询商品在各平台的发布状态")
async def get_product_listings(product_id: int, db: AsyncSession = Depends(get_db)):
    stmt = (
        select(ProductListing, Platform.name, Platform.code)
        .join(Platform, ProductListing.platform_id == Platform.id)
        .where(ProductListing.product_id == product_id)
        .order_by(Platform.sort_order)
    )
    result = await db.execute(stmt)
    listings = []
    for listing, platform_name, platform_code in result.all():
        listings.append({
            "id": listing.id,
            "product_id": listing.product_id,
            "platform_id": listing.platform_id,
            "platform_name": platform_name,
            "platform_code": platform_code,
            "platform_product_id": listing.platform_product_id,
            "platform_sku": listing.platform_sku,
            "status": listing.status,
            "platform_url": listing.platform_url,
            "sync_message": listing.sync_message,
            "last_sync_at": listing.last_sync_at.isoformat() if listing.last_sync_at else None,
            "created_at": listing.created_at.isoformat() if listing.created_at else None,
        })
    return Result.ok(listings)


@router.get("/listings", summary="全局发布状态概览")
async def get_all_listings(db: AsyncSession = Depends(get_db)):
    """获取所有商品在各平台的发布状态"""
    stmt = (
        select(ProductListing, Product.name, Platform.name, Platform.code)
        .join(Product, ProductListing.product_id == Product.id)
        .join(Platform, ProductListing.platform_id == Platform.id)
        .order_by(ProductListing.created_at.desc())
    )
    result = await db.execute(stmt)
    items = []
    for listing, product_name, platform_name, platform_code in result.all():
        items.append({
            "id": listing.id,
            "product_id": listing.product_id,
            "product_name": product_name,
            "platform_id": listing.platform_id,
            "platform_name": platform_name,
            "platform_code": platform_code,
            "platform_product_id": listing.platform_product_id,
            "status": listing.status,
            "platform_url": listing.platform_url,
            "sync_message": listing.sync_message,
            "last_sync_at": listing.last_sync_at.isoformat() if listing.last_sync_at else None,
        })
    return Result.ok(items)
