"""平台发布适配器基础类型。"""

from dataclasses import dataclass
from typing import Optional, Protocol
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Inventory, Platform, Price, Product, Sku


@dataclass(frozen=True)
class PublishResult:
    platform_product_id: str
    platform_sku: Optional[str]
    platform_url: str
    published_data: dict
    sync_message: Optional[str] = None


class ListingAdapter(Protocol):
    async def publish(
        self,
        *,
        product: Product,
        platform: Platform,
        skus: list[Sku],
        prices: dict[int, Price],
        inventories: dict[int, Inventory],
        db: Optional[AsyncSession] = None,
    ) -> PublishResult:
        """发布商品到平台并返回平台侧同步结果。"""

    async def sync_status(
        self,
        *,
        listing_id: int,
        platform: Platform,
        platform_product_id: str,
    ) -> str:
        """同步平台发布状态。"""

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """校验平台凭证是否可用。"""
