"""平台发布适配器注册入口。

通过平台代码自动选择对应的适配器。

当前适配器状态:
  - ozon:  真实 Ozon Seller API v4 (发布/状态同步/凭证校验)
  - shopee: 真实 Shopee Open Platform API v2 (发布/状态同步/凭证校验)
  - wb/wildberries: 真实 Wildberries Content API v3 (发布/状态同步/凭证校验)
  - amazon: 真实 Amazon SP-API (发布/状态同步/凭证校验)
  - tiktok: 真实 TikTok Shop API (发布/状态同步/凭证校验)
  - 其他: 回退到通用 Mock 适配器
"""

import logging

from app.listing.adapters.base import ListingAdapter
from app.listing.adapters.mock import MockListingAdapter
from app.listing.adapters.ozon import OzonListingAdapter
from app.listing.adapters.shopee import ShopeeListingAdapter
from app.listing.adapters.wildberries import WildberriesListingAdapter
from app.listing.adapters.amazon import AmazonListingAdapter
from app.listing.adapters.tiktok import TikTokShopListingAdapter

logger = logging.getLogger(__name__)

# 适配器注册表: platform_code -> adapter_class
_ADAPTER_REGISTRY: dict[str, type] = {
    "ozon": OzonListingAdapter,
    "shopee": ShopeeListingAdapter,
    "wb": WildberriesListingAdapter,
    "wildberries": WildberriesListingAdapter,
    "amazon": AmazonListingAdapter,
    "tiktok": TikTokShopListingAdapter,
}


def get_listing_adapter(platform_code: str) -> ListingAdapter:
    """按平台代码获取发布适配器。

    支持平台:
      - ozon: Ozon 真实 Seller API v4
      - shopee: Shopee 真实 Open Platform API v2
      - wb / wildberries: Wildberries 真实 Content API v3
      - 其他: 回退到通用 Mock 适配器

    添加新平台只需在 _ADAPTER_REGISTRY 注册对应 adapter 类。
    """
    normalized = platform_code.lower().strip()
    adapter_cls = _ADAPTER_REGISTRY.get(normalized)

    if adapter_cls is None:
        logger.info("平台 %s 无专用适配器，使用 Mock", platform_code)
        return MockListingAdapter()

    logger.info("使用适配器 %s 发布到平台 %s", adapter_cls.__name__, platform_code)
    return adapter_cls()


def register_adapter(platform_code: str, adapter_cls: type) -> None:
    """注册自定义发布适配器（供插件/扩展使用）"""
    _ADAPTER_REGISTRY[platform_code.lower().strip()] = adapter_cls
    logger.info("已注册适配器 %s -> %s", platform_code, adapter_cls.__name__)
