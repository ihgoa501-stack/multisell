"""Shopee 平台发布适配器 — 真实 API 集成

对接 Shopee Open Platform API v2。
文档: https://open.shopee.com/documents/v2/

认证方式:
  Partner ID + Partner Key + Access Token + Shop ID → HMAC-SHA256 签名

关键端点:
  POST /api/v2/product/add               — 创建商品
  POST /api/v2/product/get_item_base_info — 查询商品状态
  GET  /api/v2/shop/get_shop_info         — 店铺信息（凭证校验）
"""

import hashlib
import hmac
import logging
import time
from datetime import datetime, timezone
from typing import Any, Optional

import httpx

from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku

logger = logging.getLogger(__name__)

SHOPEE_API_BASE = "https://partner.shopeemobile.com"
DEFAULT_TIMEOUT = 30.0


class ShopeeListingAdapter:
    """Shopee 发布适配器 — 真实 Shopee Open Platform API"""

    API_VERSION = "v2"
    PLATFORM_CODE = "shopee"

    # ------------------------------------------------------------------ #
    #  认证辅助
    # ------------------------------------------------------------------ #

    @staticmethod
    def _sign(api_key: str, partner_id: int, api_path: str, timestamp: int,
              access_token: str, shop_id: int) -> str:
        """生成 Shopee API 签名（HMAC-SHA256）。"""
        raw = f"{partner_id}|{api_path}|{timestamp}|{access_token}|{shop_id}"
        return hmac.new(
            api_key.encode("utf-8"),
            raw.encode("utf-8"),
            hashlib.sha256,
        ).hexdigest()

    def _build_auth_params(self, platform: Platform, api_path: str) -> dict[str, Any]:
        """构造公共认证查询参数。"""
        extra = platform.extra_config or {}
        partner_id = int(platform.client_id) if platform.client_id else 0
        shop_id = int(extra.get("shop_id", 0))
        access_token = extra.get("access_token", "")
        timestamp = int(time.time())

        sign = self._sign(
            api_key=platform.api_key or "",
            partner_id=partner_id,
            api_path=api_path,
            timestamp=timestamp,
            access_token=access_token,
            shop_id=shop_id,
        )

        return {
            "partner_id": partner_id,
            "timestamp": timestamp,
            "access_token": access_token,
            "shop_id": shop_id,
            "sign": sign,
        }

    # ------------------------------------------------------------------ #
    #  HTTP 客户端
    # ------------------------------------------------------------------ #

    def _client(self, platform: Platform) -> httpx.AsyncClient:
        """构造 Shopee HTTP 客户端。"""
        base = (platform.api_base_url or SHOPEE_API_BASE).rstrip("/")
        return httpx.AsyncClient(
            base_url=base,
            headers={"Content-Type": "application/json"},
            timeout=DEFAULT_TIMEOUT,
            cookies={},
        )

    # ------------------------------------------------------------------ #
    #  数据映射
    # ------------------------------------------------------------------ #

    def _build_name(self, product: Product) -> str:
        """Shopee 商品名限 120 字符"""
        name = product.ai_title or product.name or ""
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

            variation: dict[str, Any] = {
                "name": sku.spec_desc or sku.code or f"Var-{sku.id}",
                "sku_code": sku.code or f"shopee-sku-{sku.id}",
                "price": float(price.price) if price else 0.0,
                "stock": max(int(inventory.quantity) - int(inventory.locked_quantity), 0) if inventory else 0,
            }

            if sku.sku_weight_kg:
                variation["weight"] = float(sku.sku_weight_kg)
            if sku.sku_length_cm:
                variation["length"] = float(sku.sku_length_cm)
            if sku.sku_width_cm:
                variation["width"] = float(sku.sku_width_cm)
            if sku.sku_height_cm:
                variation["height"] = float(sku.sku_height_cm)

            # Shopee 变体颜色/尺寸分隔
            if isinstance(specs, dict):
                for spec_key, spec_val in specs.items():
                    variation.setdefault("variation_options", {})[spec_key] = spec_val

            variations.append(variation)

        return variations

    def _build_logistics(self, product: Product) -> list[dict]:
        """Shopee 物流信息"""
        logistic: dict[str, Any] = {
            "logistic_id": 0,  # 使用平台默认物流
            "enabled": True,
            "shipping_fee": 0,
            "is_free": False,
            "estimated_shipping_fee": 0,
        }
        if product.package_weight_kg:
            logistic["weight"] = float(product.package_weight_kg)
        if product.package_length_cm and product.package_width_cm and product.package_height_cm:
            logistic["package_length"] = float(product.package_length_cm)
            logistic["package_width"] = float(product.package_width_cm)
            logistic["package_height"] = float(product.package_height_cm)
        return [logistic]

    def _build_images(self, product: Product) -> list[str]:
        """Shopee 图片列表（最多 9 张）"""
        images: list[str] = []
        if product.main_image:
            images.append(product.main_image)
        if product.images and isinstance(product.images, list):
            for img in product.images:
                if len(images) >= 9:
                    break
                if img not in images:
                    images.append(img)
        return images

    def _build_wholesales(self) -> list[dict]:
        """Shopee 批发价设置"""
        return [
            {"min_count": 10, "price": 0.95},
            {"min_count": 50, "price": 0.90},
        ]

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
        """发布商品到 Shopee（真实 API）。"""
        if not skus:
            raise RuntimeError("Shopee 发布失败: 至少需要一个 SKU")
        if not product.package_weight_kg:
            raise RuntimeError("Shopee 发布失败: 缺少包装重量")

        variations = self._build_variations(skus, prices, inventories)

        payload = {
            "name": self._build_name(product),
            "description": self._build_description(product),
            "category_id": product.category_id or 0,
            "brand": {"brand_id": product.brand_id or 0},
            "images": self._build_images(product),
            "variations": variations,
            "logistics": self._build_logistics(product),
            "wholesales": self._build_wholesales(),
            "cargo_type": product.cargo_type or "normal",
            "condition": "new",
        }

        api_path = "/api/v2/product/add"
        params = self._build_auth_params(platform, api_path)

        logger.info(
            "publishing to Shopee: product_id=%s, skus=%d, shop_id=%s",
            product.id, len(skus), params.get("shop_id"),
        )

        async with self._client(platform) as client:
            resp = await client.post(api_path, params=params, json=payload)
            body = self._parse_response(resp, "publish")

        item_id = body.get("response", {}).get("item_id", "")
        platform_product_id = f"shopee-{item_id}" if item_id else (
            f"shopee-{product.id}-{int(datetime.now(timezone.utc).timestamp())}"
        )

        return PublishResult(
            platform_product_id=platform_product_id,
            platform_sku=skus[0].code or f"shopee-sku-{skus[0].id}",
            platform_url=f"https://shopee.ph/product/{platform_product_id}/",
            published_data={"api_response": body, "request_payload": payload},
            sync_message=f"published to Shopee (item_id={item_id})",
        )

    async def sync_status(self, *, listing_id: int, platform: Platform,
                          platform_product_id: str) -> str:
        """查询 Shopee 商品发布状态。

        通过 get_item_base_info 获取商品当前状态。
        """
        if not platform_product_id or not platform_product_id.startswith("shopee-"):
            return "unknown"

        # 从 platform_product_id 提取 item_id
        item_id = platform_product_id.replace("shopee-", "", 1)
        try:
            int(item_id)
        except (ValueError, TypeError):
            return "unknown"

        payload = {"item_id": int(item_id)}
        api_path = "/api/v2/product/get_item_base_info"
        params = self._build_auth_params(platform, api_path)

        async with self._client(platform) as client:
            resp = await client.post(api_path, params=params, json=payload)
            body = self._parse_response(resp, "sync_status")

        item_status = body.get("response", {}).get("item_status", "")
        # Shopee 状态映射: NORMAL=上架, UNLIST=下架, BANNED=违规
        status_map = {
            "NORMAL": "synced",
            "UNLIST": "unlisted",
            "BANNED": "banned",
            "DELETED": "deleted",
        }
        return status_map.get(item_status, "unknown")

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """用 Shopee GetShopInfo API 校验凭证。"""
        if not platform.client_id or not platform.api_key:
            return False
        extra = platform.extra_config or {}
        if not extra.get("access_token") or not extra.get("shop_id"):
            return False

        api_path = "/api/v2/shop/get_shop_info"
        params = self._build_auth_params(platform, api_path)

        try:
            async with self._client(platform) as client:
                resp = await client.get(api_path, params=params)
                body = resp.json()
                return resp.status_code == 200 and body.get("error") == 0
        except Exception as exc:
            logger.warning("Shopee credential check failed: %s", exc)
            return False

    # ------------------------------------------------------------------ #
    #  辅助方法
    # ------------------------------------------------------------------ #

    @staticmethod
    def _parse_response(resp: httpx.Response, context: str) -> dict[str, Any]:
        """解析 Shopee API 响应，统一错误处理。"""
        try:
            body = resp.json()
        except Exception:
            raise RuntimeError(f"Shopee {context} 响应非 JSON: {resp.status_code} {resp.text[:500]}")

        if resp.status_code >= 400:
            msg = body.get("message", body.get("error_description", resp.text[:300]))
            raise RuntimeError(f"Shopee {context} 失败 [{resp.status_code}]: {msg}")

        # Shopee API 返回 error=0 表示成功
        error_code = body.get("error")
        if error_code is not None and error_code != 0:
            msg = body.get("message", body.get("error_description", f"error_code={error_code}"))
            raise RuntimeError(f"Shopee {context} 失败 [error={error_code}]: {msg}")

        return body
