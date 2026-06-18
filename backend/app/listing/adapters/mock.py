"""本地 mock 平台发布适配器。"""

from typing import Any

from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku


class MockListingAdapter:
    async def publish(
        self,
        *,
        product: Product,
        platform: Platform,
        skus: list[Sku],
        prices: dict[int, Price],
        inventories: dict[int, Inventory],
    ) -> PublishResult:
        if platform.code == "mockfail":
            raise RuntimeError("mock publish failed")

        platform_product_id = f"{platform.code}-{product.id}"
        platform_sku = f"{platform.code}-sku-{skus[0].id}" if skus else None
        published_skus: list[dict[str, Any]] = [
            {
                "sku_id": sku.id,
                "code": sku.code,
                "spec_desc": sku.spec_desc,
                "price": float(prices[sku.id].price),
                "quantity": inventories[sku.id].quantity,
            }
            for sku in skus
        ]

        return PublishResult(
            platform_product_id=platform_product_id,
            platform_sku=platform_sku,
            platform_url=f"https://{platform.code}.example.com/products/{product.id}",
            published_data={
                "name": product.name,
                "description": product.description or product.ai_description,
                "main_image": product.main_image,
                "sku_count": len(skus),
                "skus": published_skus,
            },
            sync_message="mock publish synced",
        )

    async def sync_status(
        self,
        *,
        listing_id: int,
        platform: Platform,
        platform_product_id: str,
    ) -> str:
        return "synced"

    async def validate_credentials(self, *, platform: Platform) -> bool:
        return platform.status == 1
