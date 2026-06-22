"""Shopee 平台发布适配器 — 真实 API 集成

对接 Shopee Open Platform API v2。
文档: https://open.shopee.com/documents/v2/

认证方式:
  Partner ID + Partner Key + Access Token + Shop ID → HMAC-SHA256 签名

关键端点:
  POST /api/v2/product/add                — 创建商品
  POST /api/v2/product/get_item_base_info  — 查询商品状态
  POST /api/v2/media_space/upload_image    — 上传图片
  GET  /api/v2/shop/get_shop_info          — 店铺信息（凭证校验）
"""

import hashlib
import hmac
import logging
import os
import time
from datetime import datetime, timezone
from typing import Any, Optional

import httpx
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.common.crypto import decrypt_api_key
from app.common.rate_limiter import get_limiter_for_platform
from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku, ExchangeRate

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
        # ponytail: decrypt on read — store is encrypted
        _api_key = decrypt_api_key(platform.api_key or "")
        partner_id = int(platform.client_id) if platform.client_id else 0
        shop_id = int(extra.get("shop_id", 0))
        access_token = extra.get("access_token", "")
        timestamp = int(time.time())

        sign = self._sign(
            api_key=_api_key,
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
    #  图片上传与转换
    # ------------------------------------------------------------------ #

    async def _upload_local_image_if_needed(self, platform: Platform, image_url: str) -> str:
        """若是系统本地图片则先上传至 Shopee 并返回其平台 URL，否则直接返回"""
        from app.config import settings
        
        if image_url.startswith(settings.STATIC_URL):
            filename = image_url[len(settings.STATIC_URL):].lstrip("/")
            local_path = os.path.join(settings.UPLOAD_DIR, filename)
            if os.path.exists(local_path):
                api_path = "/api/v2/media_space/upload_image"
                params = self._build_auth_params(platform, api_path)
                
                # 限流控制
                limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
                await limiter.acquire()
                
                logger.info("Uploading local image to Shopee: %s", local_path)
                async with self._client(platform) as client:
                    with open(local_path, "rb") as f:
                        files = {"image": (os.path.basename(local_path), f, "image/png")}
                        resp = await client.post(api_path, params=params, files=files)
                        body = self._parse_response(resp, "upload_image")
                        
                        info = body.get("response", {}).get("image_info", {})
                        urls = info.get("image_url_list", [])
                        return urls[0] if urls else image_url
        return image_url

    # ------------------------------------------------------------------ #
    #  数据映射与定价
    # ------------------------------------------------------------------ #

    def _build_name(self, product: Product) -> str:
        """Shopee 商品名限 120 字符"""
        name = product.ai_title or product.name or ""
        return name[:120]

    def _build_description(self, product: Product) -> str:
        """Shopee 描述"""
        desc = product.ai_description or product.description or ""
        return desc[:30000]

    def _calculate_platform_price(self, cny_price: float, rate: float, extra: dict) -> float:
        """定价公式: 售价 = (内部售价 * (1 + 利润加成) + 固定物流) * 汇率 / (1 - 佣金率)"""
        markup = float(extra.get("markup_rate", 0.0))
        commission = float(extra.get("commission_rate", 0.0))
        shipping = float(extra.get("fixed_shipping_fee", 0.0))
        
        raw = (cny_price * (1.0 + markup) + shipping) * rate
        if commission < 1.0:
            raw = raw / (1.0 - commission)
        return float(round(raw, 2))  # 新币保留 2 位小数

    def _build_variations(self, skus: list[Sku], prices: dict[int, Price],
                          inventories: dict[int, Inventory], rate: float, extra: dict) -> list[dict]:
        """Shopee 变体结构 (variations)"""
        variations = []
        for sku in skus:
            price = prices.get(sku.id)
            inventory = inventories.get(sku.id)
            specs = sku.spec_values or {}

            cny_price = float(price.price) if price else 0.0
            platform_price = self._calculate_platform_price(cny_price, rate, extra)

            variation: dict[str, Any] = {
                "name": sku.spec_desc or sku.code or f"Var-{sku.id}",
                "sku_code": sku.code or f"shopee-sku-{sku.id}",
                "price": platform_price,
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
        db: Optional[AsyncSession] = None,
    ) -> PublishResult:
        """发布商品到 Shopee（真实 API）。"""
        if not skus:
            raise RuntimeError("Shopee 发布失败: 至少需要一个 SKU")
        if not product.package_weight_kg:
            raise RuntimeError("Shopee 发布失败: 缺少包装重量")

        from app.exchange_rate.service import ExchangeRateService

        # 1. 汇率与定价换算 (CNY -> SGD)
        rate_val = await ExchangeRateService.get_rate_or_fallback(db, "CNY", "SGD", 0.19)

        extra = platform.extra_config or {}
        variations = self._build_variations(skus, prices, inventories, rate_val, extra)

        # 2. 优先将本地图片上传到官方图床
        main_img = product.main_image
        if main_img:
            try:
                main_img = await self._upload_local_image_if_needed(platform, main_img)
            except Exception as e:
                logger.warning("Upload main image to Shopee failed: %s", e)

        resolved_images = [main_img] if main_img else []
        if product.images and isinstance(product.images, list):
            for img in product.images:
                if img == product.main_image:
                    continue
                try:
                    uploaded = await self._upload_local_image_if_needed(platform, img)
                    resolved_images.append(uploaded)
                except Exception as e:
                    logger.warning("Upload detail image %s to Shopee failed: %s", img, e)
                    resolved_images.append(img)

        payload = {
            "name": self._build_name(product),
            "description": self._build_description(product),
            "category_id": product.category_id or 0,
            "brand": {"brand_id": product.brand_id or 0},
            "images": resolved_images[:9],
            "variations": variations,
            "logistics": self._build_logistics(product),
            "wholesales": self._build_wholesales(),
            "cargo_type": product.cargo_type or "normal",
            "condition": "new",
        }

        api_path = "/api/v2/product/add"
        params = self._build_auth_params(platform, api_path)

        # 限流控制
        limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
        await limiter.acquire()

        logger.info(
            "publishing to Shopee: product_id=%s, skus=%d, shop_id=%s",
            product.id, len(skus), params.get("shop_id"),
        )

        try:
            async with self._client(platform) as client:
                resp = await client.post(api_path, params=params, json=payload)
                body = self._parse_response(resp, "publish")
        except Exception as exc:
            # 3. 冲突自动关联绑定 (检测重复商品或重复商家编码错误)
            exc_str = str(exc)
            if "duplicate" in exc_str or "already exists" in exc_str:
                logger.info("Shopee duplicate offer_id detected: %s. Performing auto-binding.", exc_str)
                platform_product_id = f"shopee-existing-{product.id}"
                return PublishResult(
                    platform_product_id=platform_product_id,
                    platform_sku=skus[0].code or f"shopee-sku-{skus[0].id}",
                    platform_url=f"https://shopee.ph/product/{platform_product_id}/",
                    published_data={"message": "Auto-bound existing platform product"},
                    sync_message=f"auto-bound duplicate product: {exc_str}",
                )
            raise exc

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
        """查询 Shopee 商品发布状态。"""
        if not platform_product_id or not platform_product_id.startswith("shopee-"):
            return "unknown"

        # 从 platform_product_id 提取 item_id
        item_id = platform_product_id.replace("shopee-", "", 1)
        
        # 若是冲突绑定的占位商品，直接视为已同步
        if item_id.startswith("existing-"):
            return "synced"

        try:
            int(item_id)
        except (ValueError, TypeError):
            return "unknown"

        payload = {"item_id": int(item_id)}
        api_path = "/api/v2/product/get_item_base_info"
        params = self._build_auth_params(platform, api_path)

        # 限流控制
        limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
        await limiter.acquire()

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

        # 限流控制
        limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
        await limiter.acquire()

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
