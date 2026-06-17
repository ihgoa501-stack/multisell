"""Shopee 平台模拟发布适配器

模拟 Shopee 真实 API 的数据形状和行为。
参考 Shopee Product API v2 /api/v2/product/add 接口规范。
"""

import logging
from datetime import datetime, timezone

from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku

logger = logging.getLogger(__name__)


class ShopeeListingAdapter:
    """Shopee 发布适配器 — 模拟真实 API 行为"""

    API_VERSION = "v2"
    PLATFORM_CODE = "shopee"

    def _build_name(self, product: Product) -> str:
        """Shopee 商品名限 120 字符"""
        name = product.ai_title or product.name
        return name[:120]

    def _build_description(self, product: Product) -> str:
        """Shopee 描述"""
        desc = product.ai_description or product.description or ""
        return desc[:30000]

    def _build_variations(self, skus: list[Sku], prices: dict[int, Price],
                          inventories: dict[int, Inventory]) -> list[dict]:
        """Shopee 变体结构 (variations)"""
        variations = []
        for sku in skus:
            price = prices.get(sku.id)
            inventory = inventories.get(sku.id)
            specs = sku.spec_values or {}

            variation = {
                "name": sku.spec_desc or sku.code or f"Var-{sku.id}",
                "sku_code": sku.code or f"shopee-sku-{sku.id}",
                "price": float(price.price) if price else 0.0,
                "stock": max(inventory.quantity - inventory.locked_quantity, 0) if inventory else 0,
                "weight": float(sku.sku_weight_kg or 0),
                "length": float(sku.sku_length_cm or 0),
                "width": float(sku.sku_width_cm or 0),
                "height": float(sku.sku_height_cm or 0),
            }

            # Shopee 变体颜色/尺寸分隔
            if isinstance(specs, dict):
                for spec_key, spec_val in specs.items():
                    variation.setdefault("variation_options", {})[spec_key] = spec_val

            variations.append(variation)

        return variations

    def _build_logistics(self, product: Product) -> list[dict]:
        """Shopee 物流信息"""
        logistics = [{
            "logistic_id": 0,  # 模拟: 使用平台默认物流
            "enabled": True,
            "shipping_fee": 0,
            "is_free": False,
            "estimated_shipping_fee": None,
        }]
        if product.package_weight_kg:
            logistics[0]["weight"] = float(product.package_weight_kg)
        if product.package_length_cm and product.package_width_cm and product.package_height_cm:
            logistics[0]["package_length"] = float(product.package_length_cm)
            logistics[0]["package_width"] = float(product.package_width_cm)
            logistics[0]["package_height"] = float(product.package_height_cm)
        return logistics

    def _build_wholesales(self) -> list[dict]:
        """Shopee 批发价设置（模拟）"""
        return [
            {"min_count": 10, "price": 0.95},
            {"min_count": 50, "price": 0.90},
        ]

    async def publish(
        self,
        *,
        product: Product,
        platform: Platform,
        skus: list[Sku],
        prices: dict[int, Price],
        inventories: dict[int, Inventory],
    ) -> PublishResult:
        """模拟发布到 Shopee"""
        if not skus:
            raise RuntimeError("Shopee 发布失败: 至少需要一个 SKU")

        # Shopee 要求包装数据
        if not product.package_weight_kg:
            raise RuntimeError("Shopee 发布失败: 缺少包装重量")

        variations = self._build_variations(skus, prices, inventories)

        payload = {
            "api_version": self.API_VERSION,
            "name": self._build_name(product),
            "description": self._build_description(product),
            "category_id": product.category_id or 0,
            "brand": {"brand_id": product.brand_id or 0},
            "images": [product.main_image] if product.main_image else [],
            "variations": variations,
            "logistics": self._build_logistics(product),
            "wholesales": self._build_wholesales(),
            "cargo_type": product.cargo_type or "normal",
            "condition": "new",
        }

        platform_product_id = f"{self.PLATFORM_CODE}-{product.id}-{datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S')}"

        # 模拟 Shopee 发布响应
        return PublishResult(
            platform_product_id=platform_product_id,
            platform_sku=skus[0].code or f"shopee-sku-{skus[0].id}",
            platform_url=f"https://shopee.ph/product/{platform_product_id}/",
            published_data={
                "response": {
                    "item_id": int(datetime.now().timestamp() * 1000000) % 1000000,
                    "warning": "商品创建成功，等待审核" if product.category_id else "品类未指定，请手动设置",
                },
                "request_payload": payload,
            },
            sync_message="published to Shopee (simulated)",
        )

    async def sync_status(self, *, listing_id: int) -> str:
        """模拟 Shopee 状态同步"""
        return "synced"

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """模拟 Shopee 凭证校验"""
        return bool(platform.client_id and platform.api_key and platform.status == 1)
