"""TikTok Shop 平台发布适配器 — 真实 API 集成

对接 TikTok Shop API v2 (Product API 202309)。
文档: https://partner.tiktokshop.com/doc/page/63

认证方式:
  OAuth2 access_token（通过 x-tts-access-token 请求头传递）

关键端点:
  POST /product/202309/products       — 创建商品
  GET  /product/202309/products/{id}  — 查询商品状态
  GET  /product/202309/shop           — 店铺信息（凭证校验）
"""

import hashlib
import logging
from datetime import datetime, timezone
from typing import Any

import httpx

from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku

logger = logging.getLogger(__name__)

TIKTOK_API_BASE = "https://open-api.tiktokglobalshop.com"
DEFAULT_TIMEOUT = 30.0


class TikTokShopListingAdapter:
    """TikTok Shop 发布适配器 — 真实 TikTok Shop API"""

    API_VERSION = "202309"
    PLATFORM_CODE = "tiktok"

    # ------------------------------------------------------------------ #
    #  HTTP 客户端
    # ------------------------------------------------------------------ #

    def _client(self, platform: Platform) -> httpx.AsyncClient:
        """构造带 access_token 的 HTTP 客户端。"""
        extra = platform.extra_config or {}
        access_token = extra.get("access_token", "")
        base = (platform.api_base_url or TIKTOK_API_BASE).rstrip("/")

        headers = {
            "Content-Type": "application/json",
            "x-tts-access-token": access_token,
        }

        return httpx.AsyncClient(
            base_url=base,
            headers=headers,
            timeout=DEFAULT_TIMEOUT,
            cookies={},
        )

    @staticmethod
    def _build_common_params(platform: Platform) -> dict:
        """构造 TikTok Shop 公共参数（app_key, shop_id, timestamp, sign）。"""
        extra = platform.extra_config or {}
        app_key = platform.client_id or ""
        app_secret = platform.api_key or ""
        shop_id = extra.get("shop_id", "")
        timestamp = int(datetime.now(timezone.utc).timestamp())

        # 签名: app_secret + app_key + timestamp + shop_id
        raw = f"{app_secret}{app_key}{timestamp}{shop_id}"
        sign = hashlib.sha256(raw.encode("utf-8")).hexdigest()

        return {
            "app_key": app_key,
            "shop_id": str(shop_id),
            "timestamp": str(timestamp),
            "sign": sign,
        }

    # ------------------------------------------------------------------ #
    #  数据映射
    # ------------------------------------------------------------------ #

    def _build_product_name(self, product: Product) -> str:
        """TikTok 标题限 255 字符"""
        name = product.ai_title or product.name or ""
        return name[:255]

    def _build_description(self, product: Product) -> str:
        """TikTok 描述（HTML 格式）"""
        desc = product.ai_description or product.description or ""
        return desc[:5000]

    def _build_skus(self, skus: list[Sku], prices: dict[int, Price],
                    inventories: dict[int, Inventory]) -> list[dict]:
        """TikTok SKU 变体结构"""
        sku_list = []
        for sku in skus:
            price = prices.get(sku.id)
            inventory = inventories.get(sku.id)

            sales_attrs: list[dict] = []
            spec_values = sku.spec_values or {}
            if isinstance(spec_values, dict):
                for spec_key, spec_val in spec_values.items():
                    sales_attrs.append({
                        "name": spec_key,
                        "value": str(spec_val),
                    })

            sku_item: dict[str, Any] = {
                "id": sku.code or f"tt-sku-{sku.id}",
                "price": float(price.price) if price else 0.0,
                "quantity": max(int(inventory.quantity) - int(inventory.locked_quantity), 0) if inventory else 0,
                "sales_attributes": sales_attrs if sales_attrs else [{"name": "Default", "value": "Default"}],
            }

            if sku.barcode:
                sku_item["barcode"] = sku.barcode
            if sku.sku_weight_kg:
                sku_item["weight"] = float(sku.sku_weight_kg)

            sku_list.append(sku_item)

        return sku_list

    def _build_images(self, product: Product) -> list[str]:
        """TikTok 图片列表（主图优先）"""
        images: list[str] = []
        if product.main_image:
            images.append(product.main_image)
        if product.images and isinstance(product.images, list):
            for img in product.images:
                if img not in images and len(images) < 15:
                    images.append(img)
        return images

    def _build_package_dimensions(self, product: Product) -> dict:
        """TikTok 包装尺寸"""
        dims: dict[str, float] = {}
        if product.package_length_cm:
            dims["length"] = float(product.package_length_cm)
        if product.package_width_cm:
            dims["width"] = float(product.package_width_cm)
        if product.package_height_cm:
            dims["height"] = float(product.package_height_cm)
        return dims

    def _build_delivery_info(self, product: Product) -> dict:
        """TikTok 配送信息"""
        info: dict[str, Any] = {}
        if product.package_weight_kg:
            info["weight"] = float(product.package_weight_kg)
        dims = self._build_package_dimensions(product)
        if dims:
            info["package_dimensions"] = dims
        return info

    def _build_category_id(self, product: Product) -> int:
        """TikTok 类目 ID"""
        return product.category_id or 0

    def _build_brand_id(self, product: Product) -> int:
        """TikTok 品牌 ID"""
        return product.brand_id or 0

    # ------------------------------------------------------------------ #
    #  核心接口
    # ------------------------------------------------------------------ #

    async def publish(
        self,
        *,
        product: Product,
        platform: Platform,
        skus: list[Sku],
        prices: dict[int, Price],
        inventories: dict[int, Inventory],
    ) -> PublishResult:
        """发布商品到 TikTok Shop（真实 API）。"""
        if not skus:
            raise RuntimeError("TikTok 发布失败: 至少需要一个 SKU")

        common = self._build_common_params(platform)
        sku_list = self._build_skus(skus, prices, inventories)

        payload = {
            "common": common,
            "product": {
                "product_name": self._build_product_name(product),
                "description": self._build_description(product),
                "category_id": self._build_category_id(product),
                "brand_id": self._build_brand_id(product),
                "images": self._build_images(product),
                "skus": sku_list,
                "delivery_info": self._build_delivery_info(product),
            },
        }

        logger.info(
            "publishing to TikTok Shop: product_id=%s, skus=%d",
            product.id, len(skus),
        )

        async with self._client(platform) as client:
            resp = await client.post("/product/202309/products", json=payload)
            body = self._parse_response(resp, "publish")

        data = body.get("data", {}) or {}
        product_id = data.get("product_id", "")
        platform_product_id = f"tt-{product_id}" if product_id else (
            f"tt-{product.id}-{int(datetime.now(timezone.utc).timestamp())}"
        )

        return PublishResult(
            platform_product_id=platform_product_id,
            platform_sku=skus[0].code or f"tt-sku-{skus[0].id}",
            platform_url=(
                f"https://shop.tiktok.com/product/{product_id}" if product_id else ""
            ),
            published_data={"api_response": body, "request_payload": payload},
            sync_message=f"published to TikTok Shop (product_id={product_id})" if product_id else
                         "published to TikTok Shop",
        )

    async def sync_status(
        self,
        *,
        listing_id: int,
        platform: Platform,
        platform_product_id: str,
    ) -> str:
        """查询 TikTok Shop 商品状态。

        通过 GET /product/202309/products/{id} 查询。
        platform_product_id 格式: tt-{product_id}
        """
        product_id = platform_product_id.replace("tt-", "", 1) if platform_product_id.startswith("tt-") else ""

        if not product_id:
            return "unknown"

        common = self._build_common_params(platform)
        params = {"common": common}

        async with self._client(platform) as client:
            resp = await client.get(f"/product/202309/products/{product_id}", params=params)
            body = self._parse_response(resp, "sync_status")

        data = body.get("data", {}) or {}
        status = data.get("status", "")

        # TikTok 状态: DRAFT / PUBLISHED / UNDER_REVIEW / REJECTED / UNLIST
        status_map = {
            "PUBLISHED": "synced",
            "UNDER_REVIEW": "in_progress",
            "DRAFT": "pending",
            "REJECTED": "failed",
            "UNLIST": "unlisted",
        }
        return status_map.get(status, status)

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """用 TikTok Shop 店铺信息 API 校验凭证。"""
        extra = platform.extra_config or {}
        if not extra.get("access_token") or not platform.client_id or not platform.api_key:
            return False

        common = self._build_common_params(platform)
        params = {"common": common}

        try:
            async with self._client(platform) as client:
                resp = await client.get("/product/202309/shop", params=params)
                body = resp.json()
                code = body.get("code", -1)
                return code == 0
        except Exception as exc:
            logger.warning("TikTok Shop credential check failed: %s", exc)
            return False

    # ------------------------------------------------------------------ #
    #  辅助方法
    # ------------------------------------------------------------------ #

    @staticmethod
    def _parse_response(resp: httpx.Response, context: str) -> dict[str, Any]:
        """解析 TikTok Shop API 响应，统一错误处理。"""
        try:
            body = resp.json()
        except Exception:
            raise RuntimeError(f"TikTok {context} 响应非 JSON: {resp.status_code} {resp.text[:500]}")

        if resp.status_code >= 400:
            msg = body.get("message", body.get("error", resp.text[:300]))
            raise RuntimeError(f"TikTok {context} 失败 [{resp.status_code}]: {msg}")

        # TikTok API 返回 code=0 表示成功
        code = body.get("code", -1)
        if code != 0:
            msg = body.get("message", f"error_code={code}")
            raise RuntimeError(f"TikTok {context} 业务错误 [code={code}]: {msg}")

        return body
