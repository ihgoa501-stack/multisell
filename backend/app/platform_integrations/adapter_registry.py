"""平台 Adapter 能力注册表 + 凭据校验（调用真实 adapter 的 validate_credentials）。"""

from dataclasses import dataclass, field
from typing import Optional

from app import models


@dataclass(frozen=True)
class AdapterCapability:
    adapter_code: str
    display_name: str
    supports_listing_publish: bool = False
    supports_order_import: bool = False
    supports_settlement_import: bool = False
    supports_tracking_sync: bool = False
    auth_type: str = "api_key"  # api_key / oauth2 / client_credentials / none


ADAPTERS: dict[str, AdapterCapability] = {
    "amazon": AdapterCapability(
        adapter_code="amazon",
        display_name="Amazon",
        supports_listing_publish=True,
        supports_order_import=True,
        supports_settlement_import=True,
        supports_tracking_sync=True,
        auth_type="oauth2",
    ),
    "tiktok": AdapterCapability(
        adapter_code="tiktok",
        display_name="TikTok Shop",
        supports_listing_publish=True,
        supports_order_import=True,
        supports_settlement_import=True,
        supports_tracking_sync=True,
        auth_type="oauth2",
    ),
    "temu": AdapterCapability(
        adapter_code="temu",
        display_name="Temu",
        supports_listing_publish=True,
        supports_order_import=True,
        supports_settlement_import=True,
        supports_tracking_sync=False,
        auth_type="api_key",
    ),
    "shopify": AdapterCapability(
        adapter_code="shopify",
        display_name="Shopify",
        supports_listing_publish=True,
        supports_order_import=True,
        supports_settlement_import=False,
        supports_tracking_sync=False,
        auth_type="api_key",
    ),
    "ozon": AdapterCapability(
        adapter_code="ozon",
        display_name="Ozon",
        supports_listing_publish=True,
        supports_order_import=True,
        supports_settlement_import=True,
        supports_tracking_sync=True,
        auth_type="api_key",
    ),
    "wb": AdapterCapability(
        adapter_code="wb",
        display_name="Wildberries",
        supports_listing_publish=True,
        supports_order_import=True,
        supports_settlement_import=True,
        supports_tracking_sync=True,
        auth_type="api_key",
    ),
    "csv_order": AdapterCapability(
        adapter_code="csv_order",
        display_name="CSV订单导入",
        supports_listing_publish=False,
        supports_order_import=True,
        supports_settlement_import=False,
        supports_tracking_sync=False,
        auth_type="none",
    ),
}


def get_adapter(adapter_code: str) -> Optional[AdapterCapability]:
    return ADAPTERS.get(adapter_code)


def list_adapters() -> list[AdapterCapability]:
    return list(ADAPTERS.values())


async def test_connection(adapter_code: str, platform: models.Platform) -> tuple[bool, str]:
    """调用真实 adapter 的 validate_credentials() 校验凭据。"""
    if adapter_code not in ADAPTERS:
        return False, f"未知适配器: {adapter_code}"

    from app.listing.adapters import get_listing_adapter

    try:
        adapter = get_listing_adapter(adapter_code)
        if adapter is None:
            return False, f"适配器未注册: {adapter_code}"
        ok = await adapter.validate_credentials(platform=platform)
        return (ok, "验证通过" if ok else "凭证无效或API不可达")
    except Exception as exc:
        return (False, str(exc))
