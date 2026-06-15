"""平台发布适配器注册入口。"""

from app.listing.adapters.base import ListingAdapter
from app.listing.adapters.mock import MockListingAdapter


def get_listing_adapter(platform_code: str) -> ListingAdapter:
    """按平台代码获取发布适配器。

    当前真实平台尚未接入 API，先统一走 mock 适配器，保证业务流程可验证。
    """
    return MockListingAdapter()
