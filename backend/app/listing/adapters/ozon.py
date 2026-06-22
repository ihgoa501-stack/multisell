"""Ozon 平台发布适配器 — 真实 API 集成

对接 Ozon Seller API v4。
文档: https://docs.ozon.ru/api/seller/

关键端点:
  POST /v4/product/import          — 创建/更新商品
  POST /v4/product/info            — 查询商品状态
  POST /v1/product/pictures/upload — 上传本地图片流
  GET  /v1/ping                    — 健康检查
"""

import logging
import os
from datetime import datetime, timezone
from typing import Any, Optional

import httpx
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku, ExchangeRate
from app.common.rate_limiter import get_limiter_for_platform

logger = logging.getLogger(__name__)

OZON_API_BASE = "https://api-seller.ozon.ru"
DEFAULT_TIMEOUT = 30.0


class OzonListingAdapter:
    """Ozon 发布适配器 — 真实 Ozon Seller API"""

    API_VERSION = "v4"
    PLATFORM_CODE = "ozon"

    # ------------------------------------------------------------------ #
    #  HTTP 客户端
    # ------------------------------------------------------------------ #

    def _client(self, platform: Platform) -> httpx.AsyncClient:
        """构造带 Ozon 认证头部的 HTTP 客户端"""
        base = (platform.api_base_url or OZON_API_BASE).rstrip("/")
        return httpx.AsyncClient(
            base_url=base,
            headers={
                "Client-Id": platform.client_id or "",
                "Api-Key": platform.api_key or "",
                "Content-Type": "application/json",
            },
            timeout=DEFAULT_TIMEOUT,
            cookies={},
        )

    # ------------------------------------------------------------------ #
    #  图片上传与转换
    # ------------------------------------------------------------------ #

    async def _upload_local_image_if_needed(self, platform: Platform, image_url: str) -> str:
        """若是系统本地图片则先上传至 Ozon 并返回其平台 URL，否则直接返回"""
        from app.config import settings
        
        if image_url.startswith(settings.STATIC_URL):
            filename = image_url[len(settings.STATIC_URL):].lstrip("/")
            local_path = os.path.join(settings.UPLOAD_DIR, filename)
            if os.path.exists(local_path):
                # 限流控制
                limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
                await limiter.acquire()
                
                logger.info("Uploading local image to Ozon: %s", local_path)
                async with self._client(platform) as client:
                    with open(local_path, "rb") as f:
                        files = {"file": (os.path.basename(local_path), f, "image/png")}
                        resp = await client.post("/v1/product/pictures/upload", files=files)
                        body = self._parse_response(resp, "upload_image")
                        return body.get("result", {}).get("url", image_url)
        return image_url

    # ------------------------------------------------------------------ #
    #  数据映射与定价
    # ------------------------------------------------------------------ #

    def _build_name(self, product: Product) -> str:
        name = product.ai_title or product.name or ""
        return name[:500]

    def _build_description(self, product: Product) -> str:
        desc = product.ai_description or product.description or ""
        return desc[:5000]

    def _build_attributes(self, product: Product) -> list[dict]:
        """Ozon 属性结构"""
        attrs: list[dict] = []
        if product.brand_id:
            attrs.append({"attribute_id": 85, "value": str(product.brand_id)})
        if product.cargo_type == "battery":
            attrs.append({"attribute_id": 22231, "value": "1"})
        return attrs

    def _calculate_platform_price(self, cny_price: float, rate: float, extra: dict) -> float:
        """定价公式: 售价 = (内部售价 * (1 + 利润加成) + 固定物流) * 汇率 / (1 - 佣金率)"""
        markup = float(extra.get("markup_rate", 0.0))
        commission = float(extra.get("commission_rate", 0.0))
        shipping = float(extra.get("fixed_shipping_fee", 0.0))
        
        raw = (cny_price * (1.0 + markup) + shipping) * rate
        if commission < 1.0:
            raw = raw / (1.0 - commission)
        return float(round(raw))  # 卢布取整

    def _build_sku_data(self, sku: Sku, platform_price: float, inventory: Optional[Inventory]) -> dict:
        return {
            "offer_id": sku.code or f"sku-{sku.id}",
            "price": f"{platform_price:.2f}",
            "old_price": f"{platform_price * 1.2:.2f}",
            "currency_code": "RUB",
            "stock": {
                "present": max(int(inventory.quantity) - int(inventory.locked_quantity), 0) if inventory else 0,
                "reserved": int(inventory.locked_quantity) if inventory else 0,
            },
            "barcode": sku.barcode or "",
            "height": str(float(sku.sku_height_cm or 0)),
            "width": str(float(sku.sku_width_cm or 0)),
            "depth": str(float(sku.sku_length_cm or 0)),
            "weight": str(float(sku.sku_weight_kg or 0)),
        }

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
        if not skus:
            raise RuntimeError("Ozon 发布失败: 至少需要一个 SKU")
        if not product.package_length_cm or not product.package_weight_kg:
            raise RuntimeError("Ozon 发布失败: 缺少包装尺寸或重量数据")

        # 1. 自动双向绑定前置检查：若已存在则直接绑定 (仅在传入 db 且非单测时运行)
        offer_id = skus[0].code or f"sku-{skus[0].id}"
        if db is not None:
            exist_status = "unknown"
            try:
                exist_status = await self.sync_status(
                    listing_id=skus[0].id,
                    platform=platform,
                    platform_product_id=f"ozon-{offer_id}",
                )
            except Exception:
                pass

            if exist_status in ("synced", "pending", "in_progress"):
                logger.info("Ozon duplicate offer_id detected: %s. Performing auto-binding.", offer_id)
                return PublishResult(
                    platform_product_id=f"ozon-{offer_id}",
                    platform_sku=offer_id,
                    platform_url=f"https://www.ozon.ru/product/{offer_id}/",
                    published_data={"message": "Auto-bound existing platform product"},
                    sync_message=f"auto-bound existing product (status={exist_status})",
                )

        from app.exchange_rate.service import ExchangeRateService

        # 2. 汇率换算准备 (CNY -> RUB)
        rate_val = await ExchangeRateService.get_rate_or_fallback(db, "CNY", "RUB", 12.5)

        extra = platform.extra_config or {}

        # 3. 优先将本地图片上传到官方图床
        main_img = product.main_image
        if main_img:
            try:
                main_img = await self._upload_local_image_if_needed(platform, main_img)
            except Exception as e:
                logger.warning("Upload main image to Ozon failed: %s", e)

        resolved_images = [main_img] if main_img else []
        if product.images and isinstance(product.images, list):
            for img in product.images:
                if img == product.main_image:
                    continue
                try:
                    uploaded = await self._upload_local_image_if_needed(platform, img)
                    resolved_images.append(uploaded)
                except Exception as e:
                    logger.warning("Upload detail image %s to Ozon failed: %s", img, e)
                    resolved_images.append(img)

        # 4. 构建 SKU 列表及折算价格
        sku_data_list = []
        for sku in skus:
            price_cny = float(prices[sku.id].price) if prices.get(sku.id) else 0.0
            platform_price = self._calculate_platform_price(price_cny, rate_val, extra)
            sku_data_list.append(self._build_sku_data(
                sku, platform_price, inventories.get(sku.id),
            ))

        payload = {
            "items": [{
                "name": self._build_name(product),
                "description": self._build_description(product),
                "category_id": product.category_id or 0,
                "attributes": self._build_attributes(product),
                "images": resolved_images,
                "offer_id": offer_id,
                "currency_code": "RUB",
                "height": str(float(product.package_height_cm or 0)),
                "width": str(float(product.package_width_cm or 0)),
                "depth": str(float(product.package_length_cm or 0)),
                "weight": str(float(product.package_weight_kg or 0)),
                "sku_data": sku_data_list,
            }]
        }

        # 限流控制
        limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
        await limiter.acquire()

        logger.info("publishing to Ozon: product_id=%s, skus=%d", product.id, len(skus))
        async with self._client(platform) as client:
            resp = await client.post("/v4/product/import", json=payload)
            body = self._parse_response(resp, "publish")

        task_id = body.get("result", {}).get("task_id", "")
        platform_product_id = f"ozon-{offer_id}"

        return PublishResult(
            platform_product_id=platform_product_id,
            platform_sku=offer_id,
            platform_url=f"https://www.ozon.ru/product/{platform_product_id}/",
            published_data={"api_response": body, "request_payload": payload},
            sync_message=f"published to Ozon (task_id={task_id})",
        )

    async def sync_status(
        self,
        *,
        listing_id: int,
        platform: Platform,
        platform_product_id: str,
    ) -> str:
        """查询 Ozon 商品发布状态"""
        offer_id = platform_product_id
        if platform_product_id and platform_product_id.startswith("ozon-"):
            offer_id = platform_product_id[len("ozon-"):]

        payload = {"offer_id": offer_id, "sku": None}

        # 限流控制
        limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
        await limiter.acquire()

        async with self._client(platform) as client:
            resp = await client.post("/v4/product/info", json=payload)
            body = self._parse_response(resp, "sync_status")

        items = body.get("result", {}).get("items", [])
        if not items:
            return "unknown"

        state = items[0].get("state", "")
        
        # Ozon 状态映射
        state_map = {
            "imported": "synced",
            "processed": "synced",
            "processing": "in_progress",
            "created": "pending",
            "failed": "failed",
            "rejected": "failed",
        }
        return state_map.get(state, state)

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """用 Ozon Ping API 校验凭证"""
        if not platform.client_id or not platform.api_key:
            return False
        
        # 限流控制
        limiter = await get_limiter_for_platform(self.PLATFORM_CODE, platform.id)
        await limiter.acquire()

        try:
            async with self._client(platform) as client:
                resp = await client.get("/v1/ping")
                return resp.status_code == 200
        except Exception as exc:
            logger.warning("Ozon credential check failed: %s", exc)
            return False

    # ------------------------------------------------------------------ #
    #  辅助方法
    # ------------------------------------------------------------------ #

    @staticmethod
    def _parse_response(resp: httpx.Response, context: str) -> dict[str, Any]:
        """解析 Ozon API 响应，统一错误处理"""
        try:
            body = resp.json()
        except Exception:
            raise RuntimeError(f"Ozon {context} 响应非 JSON: {resp.status_code} {resp.text[:500]}")

        if resp.status_code >= 400:
            error = body.get("error", {}) or {}
            code = error.get("code", resp.status_code)
            message = error.get("message", body.get("message", resp.text[:300]))
            raise RuntimeError(f"Ozon {context} 失败 [{code}]: {message}")

        return body
