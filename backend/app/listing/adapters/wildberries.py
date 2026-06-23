"""Wildberries 平台发布适配器 — 真实 API 集成

对接 Wildberries Content API v3。
文档: https://openapi.wildberries.ru/

认证方式:
  Bearer Token（从 Platform.api_key 获取）

关键端点:
  POST /api/v3/cards/upload             — 创建/更新商品卡片
  POST /api/v3/cards/filter             — 按 vendor code 查询卡片
  GET  /api/v3/cards/cursor/list        — 列出卡片（凭证校验）
"""

import logging
from datetime import datetime, timezone
from typing import Any, Optional
from sqlalchemy.ext.asyncio import AsyncSession

import httpx

from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku

logger = logging.getLogger(__name__)

WB_API_BASE = "https://content-api.wildberries.ru"
DEFAULT_TIMEOUT = 30.0


class WildberriesListingAdapter:
    """Wildberries 发布适配器 — 真实 Wildberries Content API"""

    API_VERSION = "v3"
    PLATFORM_CODE = "wb"

    # ------------------------------------------------------------------ #
    #  HTTP 客户端
    # ------------------------------------------------------------------ #

    def _client(self, platform: Platform) -> httpx.AsyncClient:
        """构造带 Bearer Token 认证的 HTTP 客户端。"""
        base = (platform.api_base_url or WB_API_BASE).rstrip("/")
        return httpx.AsyncClient(
            base_url=base,
            headers={
                "Authorization": f"Bearer {platform.api_key or ''}",
                "Content-Type": "application/json",
            },
            timeout=DEFAULT_TIMEOUT,
            cookies={},
        )

    # ------------------------------------------------------------------ #
    #  数据映射
    # ------------------------------------------------------------------ #

    def _build_name(self, product: Product) -> str:
        """WB 标题限 100 字符"""
        name = product.ai_title or product.name or ""
        return name[:100]

    def _build_description(self, product: Product) -> str:
        """WB 描述"""
        desc = product.ai_description or product.description or ""
        return desc[:5000]

    def _build_characteristics_list(self, product: Product) -> list[dict]:
        """WB 特征列表 (characteristics 为数组)"""
        chars: list[dict] = []
        if product.brand_id:
            chars.append({"Бренд": str(product.brand_id)})
        if product.cargo_type:
            type_map = {
                "normal": "Обычный",
                "battery": "С батарейкой",
                "liquid": "Жидкость",
                "sensitive": "Чувствительный",
            }
            chars.append({"Тип товара": type_map.get(product.cargo_type, product.cargo_type)})
        if product.seo_keywords and isinstance(product.seo_keywords, list):
            chars.append({"Ключевые слова": ", ".join(product.seo_keywords[:5])})
        return chars

    def _build_sizes(self, skus: list[Sku], prices: dict[int, Price],
                     inventories: dict[int, Inventory]) -> list[dict]:
        """WB 尺寸/价格结构"""
        sizes = []
        for sku in skus:
            price = prices.get(sku.id)
            inventories.get(sku.id)

            size: dict[str, Any] = {
                "price": float(price.price) if price else 0.0,
                "skus": [sku.barcode] if sku.barcode else [sku.code or f"wb-sku-{sku.id}"],
            }

            if sku.spec_desc:
                size["techSize"] = sku.spec_desc[:50]

            if sku.sku_weight_kg:
                size["weight"] = float(sku.sku_weight_kg)

            sizes.append(size)

        return sizes

    def _build_pictures(self, product: Product) -> list[dict]:
        """WB 图片列表"""
        pictures: list[dict] = []
        if product.main_image:
            pictures.append({"url": product.main_image})
        if product.images and isinstance(product.images, list):
            seen = {product.main_image} if product.main_image else set()
            for img in product.images:
                if img not in seen and len(pictures) < 10:
                    pictures.append({"url": img})
                    seen.add(img)
        return pictures

    def _build_dimensions(self, product: Product) -> dict:
        """WB 包装尺寸（单位: cm, kg）"""
        dims: dict[str, float] = {}
        if product.package_length_cm:
            dims["length"] = float(product.package_length_cm)
        if product.package_width_cm:
            dims["width"] = float(product.package_width_cm)
        if product.package_height_cm:
            dims["height"] = float(product.package_height_cm)
        if product.package_weight_kg:
            dims["weight"] = float(product.package_weight_kg)
        return dims

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
        """发布商品到 Wildberries（真实 API）。"""
        if not skus:
            raise RuntimeError("WB 发布失败: 至少需要一个 SKU")
        if not product.main_image:
            raise RuntimeError("WB 发布失败: 缺少主图")

        sizes = self._build_sizes(skus, prices, inventories)
        first_code = skus[0].code or f"sku-{skus[0].id}"

        card = {
            "vendorCode": first_code,
            "brand": str(product.brand_id) if product.brand_id else "",
            "title": self._build_name(product),
            "description": self._build_description(product),
            "pictures": self._build_pictures(product),
            "characteristics": self._build_characteristics_list(product),
            "sizes": sizes,
        }

        dims = self._build_dimensions(product)
        if dims:
            card["dimensions"] = dims

        payload = [card]

        logger.info(
            "publishing to Wildberries: product_id=%s, skus=%d, vendor_code=%s",
            product.id, len(skus), first_code,
        )

        async with self._client(platform) as client:
            resp = await client.post("/api/v3/cards/upload", json=payload)
            body = self._parse_response(resp, "publish")

        data_list = body.get("data", [])
        nm_id = ""
        if data_list:
            nm_id = str(data_list[0].get("nmID", ""))

        platform_product_id = (
            f"wb-{nm_id}" if nm_id
            else f"wb-{product.id}-{int(datetime.now(timezone.utc).timestamp())}"
        )

        return PublishResult(
            platform_product_id=platform_product_id,
            platform_sku=first_code,
            platform_url=(
                f"https://www.wildberries.ru/catalog/{nm_id}/detail.aspx"
                if nm_id else ""
            ),
            published_data={"api_response": body, "request_payload": payload},
            sync_message=f"published to Wildberries (nmID={nm_id})" if nm_id else
                         "published to Wildberries",
        )

    async def sync_status(
        self,
        *,
        listing_id: int,
        platform: Platform,
        platform_product_id: str,
    ) -> str:
        """查询 Wildberries 商品发布状态。

        通过 /api/v3/cards/filter 按 vendor code 查询。
        platform_product_id 格式: wb-{nmId} 或 wb-{product.id}-{timestamp}
        """
        # 从 platform_product_id 提取 vendor code（fallback 到 listing_id）
        offer_id = f"sku-{listing_id}"

        async with self._client(platform) as client:
            resp = await client.post("/api/v3/cards/filter", json={
                "vendorCodes": [offer_id],
                "allowedCategories": [],
            })
            body = self._parse_response(resp, "sync_status")

        data_list = body.get("data", [])
        if not data_list:
            return "unknown"

        state = data_list[0].get("status", {}).get("state", "")

        # WB 状态映射:
        # LOADED = 已加载, READY = 已准备好, IN_PROCESS = 处理中
        # REJECTED = 被拒绝, DISABLED = 已禁用
        state_map = {
            "LOADED": "synced",
            "READY": "synced",
            "IN_PROCESS": "in_progress",
            "PROCESSING": "in_progress",
            "REJECTED": "failed",
            "DISABLED": "disabled",
        }
        return state_map.get(state, state)

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """用 WB Cursor List API 校验 Token。"""
        if not platform.api_key:
            return False
        try:
            async with self._client(platform) as client:
                resp = await client.get("/api/v3/cards/cursor/list", params={"limit": 1})
                return resp.status_code == 200
        except Exception as exc:
            logger.warning("Wildberries credential check failed: %s", exc)
            return False

    # ------------------------------------------------------------------ #
    #  辅助方法
    # ------------------------------------------------------------------ #

    @staticmethod
    def _parse_response(resp: httpx.Response, context: str) -> dict[str, Any]:
        """解析 Wildberries API 响应，统一错误处理。"""
        try:
            body = resp.json()
        except Exception:
            raise RuntimeError(f"WB {context} 响应非 JSON: {resp.status_code} {resp.text[:500]}")

        if resp.status_code >= 400:
            msg = body.get("error", body.get("message", resp.text[:300]))
            if isinstance(msg, dict):
                msg = str(msg)
            raise RuntimeError(f"WB {context} 失败 [{resp.status_code}]: {msg}")

        # WB API 可能返回 error 字段表示业务错误
        error = body.get("error")
        if error:
            msg = body.get("errorText", str(error))
            raise RuntimeError(f"WB {context} 业务错误: {msg}")

        return body
