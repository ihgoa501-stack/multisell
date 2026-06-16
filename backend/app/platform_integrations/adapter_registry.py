"""平台 Adapter 能力注册表 — 静态注册，不依赖真实 API。"""

from dataclasses import dataclass, field
from typing import Optional


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


def test_connection(adapter_code: str, credential_metadata: dict) -> tuple[bool, str]:
    """Mock: 检查 adapter_code 是否存在，存在即返回 success。"""
    if adapter_code in ADAPTERS:
        return True, "连接测试通过（mock）"
    return False, f"未知适配器: {adapter_code}"
