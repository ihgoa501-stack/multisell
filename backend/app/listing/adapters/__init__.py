"""平台发布适配器注册入口。

通过平台代码自动选择对应的适配器。
模拟 Ozon / Shopee / Wildberries 的真实数据形状和行为，
后续接入真实 API 时只需替换对应适配器实现即可。
"""

import logging
from typing import Optional

from app.listing.adapters.base import ListingAdapter
from app.listing.adapters.mock import MockListingAdapter
from app.listing.adapters.ozon import OzonListingAdapter
from app.listing.adapters.shopee import ShopeeListingAdapter
from app.listing.adapters.wildberries import WildberriesListingAdapter

logger = logging.getLogger(__name__)

# 适配器注册表: platform_code -> adapter_class
_ADAPTER_REGISTRY: dict[str, type] = {
    "ozon": OzonListingAdapter,
    "shopee": ShopeeListingAdapter,
    "wb": WildberriesListingAdapter,
    "wildberries": WildberriesListingAdapter,
}


def get_listing_adapter(platform_code: str) -> ListingAdapter:
    """按平台代码获取发布适配器。

    支持平台:
      - ozon: Ozon 模拟适配器
      - shopee: Shopee 模拟适配器
      - wb / wildberries: Wildberries 模拟适配器
      - 其他: 回退到通用 Mock 适配器

    真实平台 API 接入时，替换对应 adapter 的实现即可，
    无需修改此工厂函数之外的代码。
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
