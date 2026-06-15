"""发布管理服务层。"""

from dataclasses import dataclass

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.service import is_product_logistics_complete
from app.listing.adapters import get_listing_adapter
from app.models import Inventory, Platform, Price, Product, ProductListing, Sku


@dataclass(frozen=True)
class PublishValidationError(Exception):
    missing_requirements: list[str]


class PublishFailedError(Exception):
    def __init__(self, listing: ProductListing):
        self.listing = listing
        super().__init__(listing.sync_message)


def listing_to_dict(listing: ProductListing) -> dict:
    return {
        "id": listing.id,
        "product_id": listing.product_id,
        "platform_id": listing.platform_id,
        "platform_product_id": listing.platform_product_id,
        "platform_sku": listing.platform_sku,
        "status": listing.status,
        "platform_url": listing.platform_url,
        "sync_message": listing.sync_message,
        "last_sync_at": listing.last_sync_at.isoformat() if listing.last_sync_at else None,
        "created_at": listing.created_at.isoformat() if listing.created_at else None,
    }


class ListingService:
    @staticmethod
    async def _get_or_create_listing(
        db: AsyncSession,
        product_id: int,
        platform_id: int,
    ) -> ProductListing:
        existing = await db.execute(
            select(ProductListing).where(
                ProductListing.product_id == product_id,
                ProductListing.platform_id == platform_id,
            )
        )
        listing = existing.scalar_one_or_none()
        if listing is None:
            listing = ProductListing(product_id=product_id, platform_id=platform_id)
            db.add(listing)
        return listing

    @staticmethod
    async def _active_skus(db: AsyncSession, product_id: int) -> list[Sku]:
        result = await db.execute(
            select(Sku)
            .where(Sku.product_id == product_id, Sku.status == 1)
            .order_by(Sku.id)
        )
        return list(result.scalars().all())

    @staticmethod
    async def _sale_prices(db: AsyncSession, sku_ids: list[int]) -> dict[int, Price]:
        if not sku_ids:
            return {}

        result = await db.execute(
            select(Price)
            .where(
                Price.sku_id.in_(sku_ids),
                Price.price_type == "sale_price",
                Price.status == 1,
            )
            .order_by(Price.updated_at.desc(), Price.id.desc())
        )
        prices: dict[int, Price] = {}
        for price in result.scalars().all():
            prices.setdefault(price.sku_id, price)
        return prices

    @staticmethod
    async def _inventories(db: AsyncSession, sku_ids: list[int]) -> dict[int, Inventory]:
        if not sku_ids:
            return {}

        result = await db.execute(
            select(Inventory).where(Inventory.sku_id.in_(sku_ids))
        )
        return {inventory.sku_id: inventory for inventory in result.scalars().all()}

    @staticmethod
    async def validate_publish_ready(
        db: AsyncSession,
        product: Product,
        platform: Platform,
    ) -> tuple[list[str], list[Sku], dict[int, Price], dict[int, Inventory]]:
        missing: list[str] = []

        if platform.status != 1:
            missing.append("platform")
        if not product.name:
            missing.append("name")
        if not product.main_image:
            missing.append("main_image")
        if not is_product_logistics_complete(product):
            missing.append("logistics")

        skus = await ListingService._active_skus(db, product.id)
        sku_ids = [sku.id for sku in skus]
        prices = await ListingService._sale_prices(db, sku_ids)
        inventories = await ListingService._inventories(db, sku_ids)

        if not skus:
            missing.extend(["sku", "price", "inventory"])
        else:
            if any(sku.id not in prices for sku in skus):
                missing.append("price")
            if any(
                sku.id not in inventories or inventories[sku.id].quantity <= 0
                for sku in skus
            ):
                missing.append("inventory")

        return list(dict.fromkeys(missing)), skus, prices, inventories

    @staticmethod
    async def publish(db: AsyncSession, product: Product, platform: Platform) -> ProductListing:
        missing, skus, prices, inventories = await ListingService.validate_publish_ready(
            db, product, platform
        )
        if missing:
            raise PublishValidationError(missing)

        adapter = get_listing_adapter(platform.code)
        listing = await ListingService._get_or_create_listing(db, product.id, platform.id)

        try:
            result = await adapter.publish(
                product=product,
                platform=platform,
                skus=skus,
                prices=prices,
                inventories=inventories,
            )
        except Exception as exc:
            listing.status = "failed"
            listing.sync_message = str(exc)

            platform_statuses = dict(product.platform_statuses or {})
            platform_statuses[str(platform.id)] = "failed"
            product.platform_statuses = platform_statuses

            await db.flush()
            await db.refresh(listing)
            raise PublishFailedError(listing) from exc

        listing.platform_product_id = result.platform_product_id
        listing.platform_sku = result.platform_sku
        listing.status = "synced"
        listing.platform_url = result.platform_url
        listing.sync_message = result.sync_message
        listing.published_data = result.published_data

        platform_statuses = dict(product.platform_statuses or {})
        platform_statuses[str(platform.id)] = "synced"
        product.platform_statuses = platform_statuses

        await db.flush()
        await db.refresh(listing)
        return listing
