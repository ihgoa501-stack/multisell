"""发布管理 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.common import Result
from app.models import ProductListing, Product, Platform, User
from app.listing.service import (
    ListingService,
    PublishFailedError,
    PublishValidationError,
    listing_to_dict,
)
from app.operation_log.service import OperationLogService

router = APIRouter(tags=["发布管理"])


@router.post("/products/{product_id}/publish/{platform_id}", summary="发布商品到平台")
async def publish_to_platform(
    product_id: int,
    platform_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:publish")),
):
    """发布商品到指定平台。"""
    product = await db.get(Product, product_id)
    if not product:
        return Result.not_found("商品不存在")

    platform = await db.get(Platform, platform_id)
    if not platform:
        return Result.not_found("平台不存在")

    try:
        listing = await ListingService.publish(db, product, platform)
    except PublishValidationError as exc:
        message = "商品信息不完整"
        if "logistics" in exc.missing_requirements:
            message = "物流数据不完整：请填写包装长宽高和包装重量"
        return Result(
            code=400,
            message=message,
            data={"missing_requirements": exc.missing_requirements},
        )
    except PublishFailedError as exc:
        await OperationLogService.log(
            db,
            module="listing",
            action="publish_failed",
            resource_id=f"product={product_id},platform={platform_id}",
            content=f"发布失败: product_id={product_id}, platform_id={platform_id}, error={exc.listing.sync_message}",
            operator=current_user.username,
        )
        return Result(code=500, message="发布失败", data=listing_to_dict(exc.listing))

    await OperationLogService.log(
        db,
        module="listing",
        action="publish",
        resource_id=str(listing.id),
        content=f"发布商品到平台: product_id={product_id}, platform_id={platform_id}",
        operator=current_user.username,
    )
    return Result.ok(listing_to_dict(listing))


@router.get("/products/{product_id}/listings", summary="查询商品在各平台的发布状态")
async def get_product_listings(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:view")),
):
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
async def get_all_listings(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:view")),
):
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
