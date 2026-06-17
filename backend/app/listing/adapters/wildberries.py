"""Wildberries 平台模拟发布适配器

模拟 Wildberries 真实 API 的数据形状和行为。
参考 Wildberries Content API /api/v3/cards/upload 接口规范。
"""

import logging
from datetime import datetime, timezone

from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku

logger = logging.getLogger(__name__)


class WildberriesListingAdapter:
    """Wildberries 发布适配器 — 模拟真实 API 行为"""

    API_VERSION = "v3"
    PLATFORM_CODE = "wb"

    def _build_name(self, product: Product) -> str:
        """WB 标题限 100 字符"""
        name = product.ai_title or product.name
        return name[:100]

    def _build_description(self, product: Product) -> str:
        """WB 描述"""
        desc = product.ai_description or product.description or ""
        return desc[:5000]

    def _build_characteristics(self, product: Product) -> dict:
        """WB 特征结构 (characteristics)"""
        chars = {}
        if product.brand_id:
            chars["Бренд"] = str(product.brand_id)  # Brand in Russian
        if product.cargo_type:
            chars["Тип товара"] = {
                "normal": "Обычный",
                "battery": "С батарейкой",
                "liquid": "Жидкость",
                "sensitive": "Чувствительный",
            }.get(product.cargo_type, product.cargo_type)
        if product.seo_keywords and isinstance(product.seo_keywords, list):
            chars["Ключевые слова"] = ", ".join(product.seo_keywords[:5])
        return chars

    def _build_sizes(self, skus: list[Sku], prices: dict[int, Price],
                     inventories: dict[int, Inventory]) -> list[dict]:
        """WB 尺寸/价格结构"""
        sizes = []
        for sku in skus:
            price = prices.get(sku.id)
            inventory = inventories.get(sku.id)

            size = {
                "sku": sku.code or f"wb-sku-{sku.id}",
                "price": float(price.price) if price else 0.0,
                "stock": max(inventory.quantity - inventory.locked_quantity, 0) if inventory else 0,
                "barcode": sku.barcode or "",
            }

            if sku.sku_weight_kg:
                size["weight_kg"] = float(sku.sku_weight_kg)

            # WB 使用 tech_size 来表示规格
            if sku.spec_desc:
                size["tech_size"] = sku.spec_desc[:50]

            sizes.append(size)

        return sizes

    async def publish(
        self,
        *,
        product: Product,
        platform: Platform,
        skus: list[Sku],
        prices: dict[int, Price],
        inventories: dict[int, Inventory],
    ) -> PublishResult:
        """模拟发布到 Wildberries"""
        if not skus:
            raise RuntimeError("WB 发布失败: 至少需要一个 SKU")

        if not product.main_image:
            raise RuntimeError("WB 发布失败: 缺少主图")

        sizes = self._build_sizes(skus, prices, inventories)

        payload = {
            "api_version": self.API_VERSION,
            "imt": {
                "name": self._build_name(product),
                "description": self._build_description(product),
                "category_id": product.category_id or 0,
                "brand": str(product.brand_id) if product.brand_id else "",
                "images": [product.main_image],
                "characteristics": self._build_characteristics(product),
            },
            "sizes": sizes,
        }

        platform_product_id = f"{self.PLATFORM_CODE}-{product.id}-{datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S')}"

        return PublishResult(
            platform_product_id=platform_product_id,
            platform_sku=skus[0].code or f"wb-sku-{skus[0].id}",
            platform_url=f"https://www.wildberries.ru/catalog/{platform_product_id}/detail.aspx",
            published_data={
                "data": {
                    "vendor_code": skus[0].code or f"wb-{product.id}",
                    "sizes_count": len(sizes),
                    "imt_id": int(datetime.now().timestamp() * 1000000) % 1000000,
                    "nm_id": int(datetime.now().timestamp() * 10000) % 100000000,
                },
                "request_payload": payload,
            },
            sync_message="published to Wildberries (simulated)",
        )

    async def sync_status(self, *, listing_id: int) -> str:
        """模拟 WB 状态同步"""
        return "synced"

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """模拟 WB 凭证校验"""
        return bool(platform.api_key and platform.status == 1)
