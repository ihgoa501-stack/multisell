"""Inventory sync — after local stock changes, push to all listed platforms."""

import logging
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Platform, ProductListing, Sku
from app.listing.adapters import get_listing_adapter

logger = logging.getLogger(__name__)


async def sync_inventory_to_platforms(
    db: AsyncSession, sku_id: int, sku_code: str, quantity: int
) -> None:
    """Push new quantity to all platforms where this SKU's product is listed."""
    # Find the product this SKU belongs to
    sku = await db.get(Sku, sku_id)
    if not sku:
        logger.warning("SKU %s not found, cannot sync inventory", sku_id)
        return

    # Find all listings for the product
    rows = (
        await db.execute(
            select(ProductListing, Platform)
            .join(Platform, ProductListing.platform_id == Platform.id)
            .where(ProductListing.product_id == sku.product_id)
        )
    ).all()

    for listing, platform in rows:
        adapter = get_listing_adapter(platform.code)
        if not hasattr(adapter, "sync_inventory"):
            continue
        try:
            ok = await adapter.sync_inventory(
                platform=platform,
                sku_code=sku_code,
                platform_sku=listing.platform_sku or "",
                quantity=quantity,
            )
            if ok:
                logger.info(
                    "Inventory synced for SKU %s on %s (%d units)",
                    sku_code,
                    platform.code,
                    quantity,
                )
            else:
                logger.warning(
                    "Inventory sync failed (returned False) for SKU %s on %s",
                    sku_code,
                    platform.code,
                )
        except Exception:
            logger.exception(
                "Inventory sync error for SKU %s on %s",
                sku_code,
                platform.code,
            )
