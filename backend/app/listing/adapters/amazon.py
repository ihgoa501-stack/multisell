"""Amazon 平台发布适配器 — 真实 API 集成

对接 Amazon Selling Partner API (SP-API)。
文档: https://developer-docs.amazon.com/sp-api/

认证方式:
  Layer 1 — LWA (Login With Amazon): client_id + client_secret + refresh_token → access_token
  Layer 2 — AWS Signature V4: IAM access_key + secret_key 签名请求

关键端点:
  PUT /listings/2021-08-01/items/{sellerId}/{sku}  — 创建/更新商品
  GET /listings/2021-08-01/items/{sellerId}/{sku}   — 查询商品
  GET /sellers/v1/marketplaceParticipations          — 凭证校验
"""

import hashlib
import hmac
import logging
from datetime import datetime, timezone
from typing import Any, Optional

import httpx

from app.listing.adapters.base import PublishResult
from app.models import Inventory, Platform, Price, Product, Sku

logger = logging.getLogger(__name__)

LWA_TOKEN_URL = "https://api.amazon.com/auth/o2/token"
SP_API_BASE = "https://sellingpartnerapi-eu.amazon.com"
DEFAULT_TIMEOUT = 30.0

# AWS Signature V4 常量
AWS_SERVICE = "execute-api"
AWS_REGION = "eu-west-1"  # 默认欧盟, 可在 extra_config 中覆盖


class AmazonListingAdapter:
    """Amazon 发布适配器 — 真实 Selling Partner API"""

    API_VERSION = "2021-08-01"
    PLATFORM_CODE = "amazon"

    # ------------------------------------------------------------------ #
    #  LWA Token 获取
    # ------------------------------------------------------------------ #

    async def _get_access_token(self, platform: Platform) -> str:
        """通过 LWA 获取 access_token。"""
        extra = platform.extra_config or {}
        refresh_token = extra.get("refresh_token", "")

        payload = {
            "grant_type": "refresh_token",
            "client_id": platform.client_id or "",
            "client_secret": platform.api_key or "",
            "refresh_token": refresh_token,
        }

        async with httpx.AsyncClient(timeout=DEFAULT_TIMEOUT) as client:
            resp = await client.post(LWA_TOKEN_URL, json=payload)
            if resp.status_code >= 400:
                raise RuntimeError(
                    f"Amazon LWA token 获取失败: {resp.status_code} {resp.text[:300]}"
                )
            body = resp.json()
            token = body.get("access_token", "")
            if not token:
                raise RuntimeError("Amazon LWA 响应缺少 access_token")
            return token

    # ------------------------------------------------------------------ #
    #  AWS Signature V4
    # ------------------------------------------------------------------ #

    @staticmethod
    def _sha256(data: bytes) -> str:
        return hashlib.sha256(data).hexdigest()

    @staticmethod
    def _hmac_sha256(key: bytes, msg: str) -> bytes:
        return hmac.new(key, msg.encode("utf-8"), hashlib.sha256).digest()

    def _get_signing_key(
        self, secret_key: str, date_stamp: str, region: str, service: str
    ) -> bytes:
        k_date = self._hmac_sha256(f"AWS4{secret_key}".encode("utf-8"), date_stamp)
        k_region = self._hmac_sha256(k_date, region)
        k_service = self._hmac_sha256(k_region, service)
        k_signing = self._hmac_sha256(k_service, "aws4_request")
        return k_signing

    def _sign_v4(
        self,
        method: str,
        url: str,
        payload: bytes,
        access_key: str,
        secret_key: str,
        region: str,
        token: str,
    ) -> dict[str, str]:
        """为 SP-API 请求生成 AWS Signature V4 认证头部。"""
        now = datetime.now(timezone.utc)
        amz_date = now.strftime("%Y%m%dT%H%M%SZ")
        date_stamp = now.strftime("%Y%m%d")

        parsed_url = httpx.URL(url)
        canonical_uri = parsed_url.path or "/"
        canonical_qs = parsed_url.query.decode("utf-8") if parsed_url.query else ""

        payload_hash = self._sha256(payload)

        # 计算 signed headers
        signed_headers_map = {
            "host": parsed_url.host or "",
            "x-amz-date": amz_date,
            "x-amz-access-token": token,
            "content-type": "application/json",
        }

        # 按 header name 排序
        sorted_headers = sorted(signed_headers_map.keys())
        canonical_headers = "".join(
            f"{h}:{signed_headers_map[h]}\n" for h in sorted_headers
        )
        signed_headers_str = ";".join(sorted_headers)

        # Canonical Request
        canonical_request = (
            f"{method}\n"
            f"{canonical_uri}\n"
            f"{canonical_qs}\n"
            f"{canonical_headers}\n"
            f"{signed_headers_str}\n"
            f"{payload_hash}"
        )

        # String to Sign
        algorithm = "AWS4-HMAC-SHA256"
        credential_scope = f"{date_stamp}/{region}/{AWS_SERVICE}/aws4_request"
        string_to_sign = (
            f"{algorithm}\n"
            f"{amz_date}\n"
            f"{credential_scope}\n"
            f"{self._sha256(canonical_request.encode('utf-8'))}"
        )

        # Sign
        signing_key = self._get_signing_key(secret_key, date_stamp, region, AWS_SERVICE)
        signature = hmac.new(
            signing_key, string_to_sign.encode("utf-8"), hashlib.sha256
        ).hexdigest()

        authorization = (
            f"{algorithm} Credential={access_key}/{credential_scope}, "
            f"SignedHeaders={signed_headers_str}, Signature={signature}"
        )

        return {
            "Authorization": authorization,
            "x-amz-date": amz_date,
            "x-amz-access-token": token,
            "Content-Type": "application/json",
        }

    # ------------------------------------------------------------------ #
    #  HTTP 客户端
    # ------------------------------------------------------------------ #

    async def _signed_request(
        self,
        method: str,
        path: str,
        platform: Platform,
        access_token: str,
        payload: Optional[dict] = None,
    ) -> httpx.Response:
        """发送带 AWS Signature V4 签名的请求。"""
        extra = platform.extra_config or {}
        region = extra.get("aws_region", AWS_REGION)

        base = (platform.api_base_url or SP_API_BASE).rstrip("/")
        url = f"{base}{path}"
        body_bytes = (
            b"{}"
            if payload is None
            else (
                httpx._models.to_bytes(payload)
                if hasattr(httpx._models, "to_bytes")
                else httpx._content.json_dumps(payload).encode("utf-8")
            )
        )

        # 对 dict 序列化
        if isinstance(payload, dict):
            import json

            body_bytes = json.dumps(
                payload, ensure_ascii=False, separators=(",", ":")
            ).encode("utf-8")

        aws_access_key = extra.get("aws_access_key", "")
        aws_secret_key = extra.get("aws_secret_key", "")

        headers = self._sign_v4(
            method=method,
            url=url,
            payload=body_bytes,
            access_key=aws_access_key,
            secret_key=aws_secret_key,
            region=region,
            token=access_token,
        )

        async with httpx.AsyncClient(
            base_url=base, timeout=DEFAULT_TIMEOUT, cookies={}
        ) as client:
            resp = await client.request(
                method, path, headers=headers, content=body_bytes
            )
            return resp

    # ------------------------------------------------------------------ #
    #  数据映射
    # ------------------------------------------------------------------ #

    def _build_title(self, product: Product) -> str:
        """Amazon 标题限 200 字符"""
        title = product.ai_title or product.name or ""
        return title[:200]

    def _build_description(self, product: Product) -> str:
        """Amazon 描述"""
        desc = product.ai_description or product.description or ""
        return desc[:2000]

    def _build_item_name(self, product: Product) -> str:
        return self._build_title(product)

    def _build_product_type(self, product: Product) -> str:
        """Amazon 商品类型（基于类目 ID 映射）"""
        # 简化为通用类型，后续可通过类目映射优化
        return "PRODUCT"

    def _build_attributes(self, product: Product, skus: list[Sku]) -> dict:
        """Amazon 属性结构"""
        attributes: dict[str, Any] = {
            "item_name": [{"value": self._build_item_name(product)}],
            "brand": [
                {"value": str(product.brand_id) if product.brand_id else "Generic"}
            ],
            "description": [{"value": self._build_description(product)}],
        }

        if product.main_image:
            attributes["main_image"] = [{"value": product.main_image}]

        if (
            product.images
            and isinstance(product.images, list)
            and len(product.images) > 1
        ):
            attributes["other_images"] = [{"value": img} for img in product.images[1:6]]

        if product.package_weight_kg:
            attributes["item_package_weight"] = [
                {
                    "value": float(product.package_weight_kg),
                    "unit": "kilograms",
                }
            ]

        dimensions = []
        if product.package_length_cm:
            dimensions.append(
                {
                    "name": "length",
                    "value": {
                        "value": float(product.package_length_cm),
                        "unit": "centimeters",
                    },
                }
            )
        if product.package_width_cm:
            dimensions.append(
                {
                    "name": "width",
                    "value": {
                        "value": float(product.package_width_cm),
                        "unit": "centimeters",
                    },
                }
            )
        if product.package_height_cm:
            dimensions.append(
                {
                    "name": "height",
                    "value": {
                        "value": float(product.package_height_cm),
                        "unit": "centimeters",
                    },
                }
            )
        if dimensions:
            attributes["item_package_dimensions"] = dimensions

        return attributes

    def _build_offers(
        self,
        skus: list[Sku],
        prices: dict[int, Price],
        inventories: dict[int, Inventory],
    ) -> list[dict]:
        """Amazon 报价结构"""
        offers = []
        for sku in skus:
            price = prices.get(sku.id)
            inventory = inventories.get(sku.id)
            offers.append(
                {
                    "sku": sku.code or f"amz-sku-{sku.id}",
                    "price": {
                        "currency_code": "USD",
                        "amount": float(price.price) if price else 0.0,
                    },
                    "quantity": max(
                        int(inventory.quantity) - int(inventory.locked_quantity), 0
                    )
                    if inventory
                    else 0,
                    "condition": "New",
                }
            )
        return offers

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
        """发布商品到 Amazon（真实 SP-API）。"""
        if not skus:
            raise RuntimeError("Amazon 发布失败: 至少需要一个 SKU")

        extra = platform.extra_config or {}
        seller_id = extra.get("seller_id", "")
        marketplace_id = extra.get("marketplace_id", "A1PA6795UKMFR9")  # 默认 EU
        if not seller_id:
            raise RuntimeError("Amazon 发布失败: 缺少 seller_id (extra_config)")

        first_sku = skus[0]
        sku_code = first_sku.code or f"sku-{first_sku.id}"

        # 获取 LWA access token
        access_token = await self._get_access_token(platform)

        # 构建请求体
        payload = {
            "productType": self._build_product_type(product),
            "attributes": self._build_attributes(product, skus),
            "offers": self._build_offers(skus, prices, inventories),
        }

        path = f"/listings/2021-08-01/items/{seller_id}/{sku_code}"
        params = f"marketplaceIds={marketplace_id}"

        logger.info(
            "publishing to Amazon: product_id=%s, sku=%s, marketplace=%s",
            product.id,
            sku_code,
            marketplace_id,
        )

        resp = await self._signed_request(
            "PUT", f"{path}?{params}", platform, access_token, payload
        )
        body = self._parse_response(resp, "publish")

        listing_id = body.get("sellingPartnerId", "")
        platform_product_id = f"amz-{listing_id}" if listing_id else f"amz-{sku_code}"

        return PublishResult(
            platform_product_id=platform_product_id,
            platform_sku=sku_code,
            platform_url=f"https://www.amazon.com/dp/{listing_id}"
            if listing_id
            else "",
            published_data={"api_response": body, "request_payload": payload},
            sync_message=f"published to Amazon (listing_id={listing_id})"
            if listing_id
            else "published to Amazon",
        )

    async def sync_status(
        self,
        *,
        listing_id: int,
        platform: Platform,
        platform_product_id: str,
    ) -> str:
        """查询 Amazon 商品状态。

        通过 GET /listings/2021-08-01/items/{sellerId}/{sku} 查询。
        """
        extra = platform.extra_config or {}
        seller_id = extra.get("seller_id", "")
        marketplace_id = extra.get("marketplace_id", "A1PA6795UKMFR9")
        if not seller_id:
            return "unknown"

        # 从 platform_product_id 推断 SKU
        sku_code = f"sku-{listing_id}"

        # 获取 LWA access token
        try:
            access_token = await self._get_access_token(platform)
        except Exception as exc:
            logger.warning("Amazon sync_status token 获取失败: %s", exc)
            return "unknown"

        path = f"/listings/2021-08-01/items/{seller_id}/{sku_code}"
        params = f"marketplaceIds={marketplace_id}"

        try:
            resp = await self._signed_request(
                "GET", f"{path}?{params}", platform, access_token
            )
            body = self._parse_response(resp, "sync_status")
        except Exception as exc:
            logger.warning("Amazon sync_status 查询失败: %s", exc)
            return "unknown"

        # Amazon 状态: BUYABLE / INACTIVE / UPLOAD_COMPLETE / SEARCH_SUPPRESSED
        status = body.get("status", [])
        if "BUYABLE" in status:
            return "synced"
        if "INACTIVE" in status:
            return "in_progress"
        if status:
            return "unknown"
        return "unknown"

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """用 Amazon Marketplace Participations API 校验凭证。"""
        extra = platform.extra_config or {}
        if (
            not extra.get("refresh_token")
            or not extra.get("aws_access_key")
            or not extra.get("aws_secret_key")
        ):
            return False
        if not platform.client_id or not platform.api_key:
            return False

        try:
            access_token = await self._get_access_token(platform)
            resp = await self._signed_request(
                "GET", "/sellers/v1/marketplaceParticipations", platform, access_token
            )
            body = resp.json()
            return resp.status_code == 200 and "payload" in body
        except Exception as exc:
            logger.warning("Amazon credential check failed: %s", exc)
            return False

    # ------------------------------------------------------------------ #
    #  辅助方法
    # ------------------------------------------------------------------ #

    @staticmethod
    def _parse_response(resp: httpx.Response, context: str) -> dict[str, Any]:
        """解析 Amazon SP-API 响应，统一错误处理。"""
        try:
            body = resp.json()
        except Exception:
            raise RuntimeError(
                f"Amazon {context} 响应非 JSON: {resp.status_code} {resp.text[:500]}"
            )

        if resp.status_code >= 400:
            msg = body.get("message", body.get("errors", resp.text[:300]))
            if isinstance(msg, list):
                msg = "; ".join(e.get("message", str(e)) for e in msg[:3])
            raise RuntimeError(f"Amazon {context} 失败 [{resp.status_code}]: {msg}")

        return body
