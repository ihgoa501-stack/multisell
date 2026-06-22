"""Ozon, Shopee, Wildberries, Amazon & TikTok 真实 API 适配器单元测试"""

import json
from datetime import datetime
from unittest.mock import patch

import httpx
import pytest
from httpx import AsyncClient, MockTransport

from app.listing.adapters.ozon import OzonListingAdapter
from app.listing.adapters.shopee import ShopeeListingAdapter
from app.listing.adapters.wildberries import WildberriesListingAdapter
from app.listing.adapters.amazon import AmazonListingAdapter
from app.listing.adapters.tiktok import TikTokShopListingAdapter
from app.models import Inventory, Platform, Price, Product, Sku


# =================================================================== #
#  Fixtures
# =================================================================== #

@pytest.fixture
def platform() -> Platform:
    """共享 Ozon 测试平台（内存对象，不写 DB）"""
    return Platform(
        id=1,
        code="ozon",
        name="Ozon",
        api_base_url="https://api-seller.ozon.ru",
        client_id="test-client-id",
        api_key="test-api-key",
        extra_config={},
        status=1,
    )


@pytest.fixture
def shopee_platform() -> Platform:
    return Platform(
        id=2,
        code="shopee",
        name="Shopee",
        api_base_url="https://partner.shopeemobile.com",
        client_id="123456",
        api_key="test-partner-key",
        extra_config={"access_token": "test-access-token", "shop_id": 654321},
        status=1,
    )


@pytest.fixture
def product() -> Product:
    return Product(
        id=1001,
        name="测试商品",
        ai_title="测试商品 AI 标题",
        ai_description="AI 生成的描述文本",
        description="人工描述",
        category_id=12345,
        brand_id=678,
        main_image="https://example.com/img.jpg",
        images=["https://example.com/img1.jpg", "https://example.com/img2.jpg"],
        package_weight_kg=0.5,
        package_length_cm=20,
        package_width_cm=15,
        package_height_cm=10,
        cargo_type="normal",
    )


@pytest.fixture
def skus() -> list[Sku]:
    return [
        Sku(
            id=1,
            code="SKU-001",
            barcode="6901234567890",
            spec_desc="红色/L",
            spec_values={"color": "红色", "size": "L"},
            sku_weight_kg=0.3,
            sku_length_cm=10,
            sku_width_cm=8,
            sku_height_cm=5,
        ),
        Sku(
            id=2,
            code="SKU-002",
            barcode="6901234567891",
            spec_desc="蓝色/M",
            spec_values={"color": "蓝色", "size": "M"},
            sku_weight_kg=0.25,
            sku_length_cm=9,
            sku_width_cm=7,
            sku_height_cm=4,
        ),
    ]


@pytest.fixture
def prices(skus: list[Sku]) -> dict[int, Price]:
    return {
        skus[0].id: Price(id=1, sku_id=skus[0].id, price_type="sale_price", price=199.00),
        skus[1].id: Price(id=2, sku_id=skus[1].id, price_type="sale_price", price=149.00),
    }


@pytest.fixture
def inventories(skus: list[Sku]) -> dict[int, Inventory]:
    return {
        skus[0].id: Inventory(id=1, sku_id=skus[0].id, quantity=100, locked_quantity=5),
        skus[1].id: Inventory(id=2, sku_id=skus[1].id, quantity=50, locked_quantity=2),
    }


# =================================================================== #
#  Ozon Adapter Tests
# =================================================================== #

class TestOzonListingAdapter:
    """Ozon 真实 API 适配器测试"""

    @pytest.fixture
    def adapter(self) -> OzonListingAdapter:
        return OzonListingAdapter()

    async def test_publish_success(self, adapter: OzonListingAdapter, platform: Platform,
                                    product: Product, skus: list[Sku],
                                    prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """publish() 应返回正确的 PublishResult"""
        # 构造 mock response
        def mock_handler(request: httpx.Request) -> httpx.Response:
            assert request.url.path.endswith("/v4/product/import")
            body = json.loads(request.read())
            assert body["items"][0]["name"] == "测试商品 AI 标题"
            assert body["items"][0]["sku_data"][0]["offer_id"] == "SKU-001"
            return httpx.Response(200, json={
                "result": {"task_id": "ozon-task-12345"},
            })

        # 注入 mock transport
        with patch.object(adapter, "_client") as mock_client_factory:
            mock_client_factory.return_value = AsyncClient(
                transport=MockTransport(mock_handler),
                base_url="https://api-seller.ozon.ru",
            )

            result = await adapter.publish(
                product=product,
                platform=platform,
                skus=skus,
                prices=prices,
                inventories=inventories,
            )

        assert result.platform_product_id.startswith("ozon-")
        assert result.platform_sku == "SKU-001"
        assert "ozon.ru" in result.platform_url
        assert "task_id" in result.sync_message
        assert result.published_data["api_response"]["result"]["task_id"] == "ozon-task-12345"
        assert "request_payload" in result.published_data

    async def test_publish_no_skus(self, adapter: OzonListingAdapter, platform: Platform,
                                    product: Product, prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """无 SKU 时应抛出 RuntimeError"""
        with pytest.raises(RuntimeError, match="至少需要一个 SKU"):
            await adapter.publish(
                product=product, platform=platform, skus=[],
                prices=prices, inventories=inventories,
            )

    async def test_publish_no_package_data(self, adapter: OzonListingAdapter,
                                            platform: Platform, product: Product,
                                            skus: list[Sku], prices: dict[int, Price],
                                            inventories: dict[int, Inventory]):
        """缺少包装数据时应抛出 RuntimeError"""
        product.package_length_cm = None
        product.package_weight_kg = None
        with pytest.raises(RuntimeError, match="缺少包装尺寸或重量"):
            await adapter.publish(
                product=product, platform=platform, skus=skus,
                prices=prices, inventories=inventories,
            )

    async def test_publish_api_error(self, adapter: OzonListingAdapter, platform: Platform,
                                      product: Product, skus: list[Sku],
                                      prices: dict[int, Price],
                                      inventories: dict[int, Inventory]):
        """API 返回错误时应抛出 RuntimeError"""
        def error_handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(400, json={
                "error": {"code": "INVALID_CATEGORY", "message": "无效类目 ID"},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(error_handler), base_url="https://api.example.com")

            with pytest.raises(RuntimeError, match="无效类目"):
                await adapter.publish(
                    product=product, platform=platform, skus=skus,
                    prices=prices, inventories=inventories,
                )

    async def test_sync_status_synced(self, adapter: OzonListingAdapter, platform: Platform):
        """sync_status 应返回 synced"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "result": {"items": [{"state": "imported"}]},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            status = await adapter.sync_status(
                listing_id=1001, platform=platform,
                platform_product_id="ozon-1001-1718000000",
            )

        assert status == "synced"

    async def test_sync_status_in_progress(self, adapter: OzonListingAdapter, platform: Platform):
        """sync_status 应返回 in_progress"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "result": {"items": [{"state": "processing"}]},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            status = await adapter.sync_status(
                listing_id=1001, platform=platform,
                platform_product_id="ozon-1001-1718000000",
            )

        assert status == "in_progress"

    async def test_sync_status_no_items(self, adapter: OzonListingAdapter, platform: Platform):
        """无 items 时应返回 unknown"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "result": {"items": []},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            status = await adapter.sync_status(
                listing_id=1001, platform=platform,
                platform_product_id="ozon-1001-1718000000",
            )

        assert status == "unknown"

    async def test_validate_credentials_success(self, adapter: OzonListingAdapter,
                                                 platform: Platform):
        """凭证有效时应返回 True"""
        def handler(request: httpx.Request) -> httpx.Response:
            assert request.url.path.endswith("/v1/ping")
            return httpx.Response(200)

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            result = await adapter.validate_credentials(platform=platform)

        assert result is True

    async def test_validate_credentials_failure(self, adapter: OzonListingAdapter,
                                                 platform: Platform):
        """凭证无效时应返回 False"""
        platform.client_id = ""
        platform.api_key = ""

        result = await adapter.validate_credentials(platform=platform)
        assert result is False

    async def test_validate_credentials_api_error(self, adapter: OzonListingAdapter,
                                                   platform: Platform):
        """API 返回非 200 时应返回 False"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(401, json={"error": {"code": "UNAUTHORIZED"}})

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            result = await adapter.validate_credentials(platform=platform)

        assert result is False

    async def test_fetch_orders(self, adapter: OzonListingAdapter, platform: Platform):
        """fetch_orders() 应返回映射后的订单数据"""
        call_count = 0

        def handler(request: httpx.Request) -> httpx.Response:
            nonlocal call_count
            call_count += 1
            assert request.url.path.endswith("/v3/posting/fbs/list")
            body = json.loads(request.read())
            assert "filter" in body
            assert "since" in body["filter"]
            assert body["dir"] == "ASC"
            assert body["limit"] == 100
            if call_count == 1:
                return httpx.Response(200, json={
                    "result": {
                        "postings": [{
                            "posting_number": "12345",
                            "status": "delivered",
                            "analytics_data": {"delivery_price": "5.00"},
                            "financial_data": {"products": [{
                                "sku": "SKU001", "quantity": 2, "price": "19.99",
                            }]},
                            "in_process_at": "2026-06-20T10:00:00Z",
                        }],
                        "count": 1,
                    }
                })
            return httpx.Response(200, json={
                "result": {"postings": [], "count": 0}
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.side_effect = lambda p=None: AsyncClient(
                transport=MockTransport(handler),
                base_url="https://api-seller.ozon.ru",
            )

            result = await adapter.fetch_orders(
                platform=platform,
                since=datetime(2026, 6, 19),
            )

        assert len(result) == 1
        assert result[0]["order_sn"] == "12345"
        assert result[0]["status"] == "delivered"
        assert result[0]["shipping_fee"] == "5.00"
        assert result[0]["paid_at"] == "2026-06-20T10:00:00Z"
        assert len(result[0]["items"]) == 1
        assert result[0]["items"][0]["sku_code"] == "SKU001"
        assert result[0]["items"][0]["quantity"] == 2
        assert result[0]["items"][0]["unit_price"] == "19.99"
        assert result[0]["total_amount"] == "39.98"
        assert call_count == 2  # one with data + one empty to break loop

    async def test_fetch_orders_pagination(self, adapter: OzonListingAdapter, platform: Platform):
        """fetch_orders() 应循环翻页直到无数据"""
        call_count = 0

        def handler(request: httpx.Request) -> httpx.Response:
            nonlocal call_count
            call_count += 1
            body = json.loads(request.read())
            page = body.get("page", 1)
            if page == 1:
                return httpx.Response(200, json={
                    "result": {"postings": [{
                        "posting_number": f"PAGE1-{page}",
                        "status": "delivered",
                        "analytics_data": {"delivery_price": "3.00"},
                        "financial_data": {"products": [{"sku": "A", "quantity": 1, "price": "10.00"}]},
                        "in_process_at": "2026-06-20T10:00:00Z",
                    }], "count": 1}
                })
            elif page == 2:
                return httpx.Response(200, json={
                    "result": {"postings": [{
                        "posting_number": f"PAGE2-{page}",
                        "status": "delivered",
                        "analytics_data": {"delivery_price": "4.00"},
                        "financial_data": {"products": [{"sku": "B", "quantity": 3, "price": "15.00"}]},
                        "in_process_at": "2026-06-21T10:00:00Z",
                    }], "count": 1}
                })
            else:
                return httpx.Response(200, json={
                    "result": {"postings": [], "count": 0}
                })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.side_effect = lambda p=None: AsyncClient(
                transport=MockTransport(handler),
                base_url="https://api-seller.ozon.ru",
            )

            result = await adapter.fetch_orders(
                platform=platform,
                since=datetime(2026, 6, 19),
            )

        assert len(result) == 2
        assert result[0]["order_sn"] == "PAGE1-1"
        assert result[1]["order_sn"] == "PAGE2-2"
        assert call_count == 3


# =================================================================== #
#  Shopee Adapter Tests
# =================================================================== #

class TestShopeeListingAdapter:
    """Shopee 真实 API 适配器测试"""

    @pytest.fixture
    def adapter(self) -> ShopeeListingAdapter:
        return ShopeeListingAdapter()

    async def test_publish_success(self, adapter: ShopeeListingAdapter,
                                    shopee_platform: Platform, product: Product,
                                    skus: list[Sku], prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """publish() 应返回正确的 PublishResult"""
        def handler(request: httpx.Request) -> httpx.Response:
            assert request.url.path.endswith("/api/v2/product/add")
            # 验证认证参数
            assert "partner_id" in str(request.url)
            assert "sign" in str(request.url)
            body = json.loads(request.read())
            assert body["name"] == "测试商品 AI 标题"
            assert len(body["variations"]) == 2
            assert body["variations"][0]["sku_code"] == "SKU-001"
            return httpx.Response(200, json={
                "error": 0,
                "request_id": "req-001",
                "response": {"item_id": 987654321},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://partner.shopeemobile.com",
            )

            result = await adapter.publish(
                product=product, platform=shopee_platform,
                skus=skus, prices=prices, inventories=inventories,
            )

        assert result.platform_product_id.startswith("shopee-")
        assert "shopee.ph" in result.platform_url
        assert "item_id" in result.sync_message

    async def test_publish_no_skus(self, adapter: ShopeeListingAdapter,
                                    shopee_platform: Platform, product: Product,
                                    prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """无 SKU 时应抛出 RuntimeError"""
        with pytest.raises(RuntimeError, match="至少需要一个 SKU"):
            await adapter.publish(
                product=product, platform=shopee_platform, skus=[],
                prices=prices, inventories=inventories,
            )

    async def test_publish_no_weight(self, adapter: ShopeeListingAdapter,
                                      shopee_platform: Platform, product: Product,
                                      skus: list[Sku], prices: dict[int, Price],
                                      inventories: dict[int, Inventory]):
        """缺少包装重量时应抛出 RuntimeError"""
        product.package_weight_kg = None
        with pytest.raises(RuntimeError, match="缺少包装重量"):
            await adapter.publish(
                product=product, platform=shopee_platform, skus=skus,
                prices=prices, inventories=inventories,
            )

    async def test_publish_api_error(self, adapter: ShopeeListingAdapter,
                                      shopee_platform: Platform, product: Product,
                                      skus: list[Sku], prices: dict[int, Price],
                                      inventories: dict[int, Inventory]):
        """API 返回 error != 0 时应抛出 RuntimeError"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "error": 1003,
                "message": "Invalid access token",
                "request_id": "req-001",
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            with pytest.raises(RuntimeError, match="Invalid access token"):
                await adapter.publish(
                    product=product, platform=shopee_platform, skus=skus,
                    prices=prices, inventories=inventories,
                )

    async def test_publish_http_error(self, adapter: ShopeeListingAdapter,
                                       shopee_platform: Platform, product: Product,
                                       skus: list[Sku], prices: dict[int, Price],
                                       inventories: dict[int, Inventory]):
        """HTTP 400 时应抛出 RuntimeError"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(400, json={
                "message": "Bad request: invalid category",
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            with pytest.raises(RuntimeError, match="Bad request"):
                await adapter.publish(
                    product=product, platform=shopee_platform, skus=skus,
                    prices=prices, inventories=inventories,
                )

    async def test_sync_status_synced(self, adapter: ShopeeListingAdapter,
                                       shopee_platform: Platform):
        """sync_status 应返回 synced"""
        def handler(request: httpx.Request) -> httpx.Response:
            assert request.url.path.endswith("/api/v2/product/get_item_base_info")
            return httpx.Response(200, json={
                "error": 0,
                "response": {"item_status": "NORMAL"},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            status = await adapter.sync_status(
                listing_id=1001, platform=shopee_platform,
                platform_product_id="shopee-987654321",
            )

        assert status == "synced"

    async def test_sync_status_unlisted(self, adapter: ShopeeListingAdapter,
                                         shopee_platform: Platform):
        """sync_status 应返回 unlisted"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "error": 0,
                "response": {"item_status": "UNLIST"},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            status = await adapter.sync_status(
                listing_id=1001, platform=shopee_platform,
                platform_product_id="shopee-987654321",
            )

        assert status == "unlisted"

    async def test_sync_status_unknown_item_id(
            self, adapter: ShopeeListingAdapter, shopee_platform: Platform):
        """无效的 platform_product_id 应返回 unknown"""
        status = await adapter.sync_status(
            listing_id=1001, platform=shopee_platform,
            platform_product_id="invalid-id",
        )

        assert status == "unknown"

    async def test_validate_credentials_success(self, adapter: ShopeeListingAdapter,
                                                 shopee_platform: Platform):
        """凭证有效时应返回 True"""
        def handler(request: httpx.Request) -> httpx.Response:
            assert request.url.path.endswith("/api/v2/shop/get_shop_info")
            return httpx.Response(200, json={
                "error": 0,
                "response": {"shop_id": 654321, "shop_name": "Test Shop"},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            result = await adapter.validate_credentials(platform=shopee_platform)

        assert result is True

    async def test_validate_credentials_missing_fields(
            self, adapter: ShopeeListingAdapter):
        """缺少字段时应返回 False"""
        platform = Platform(
            id=3, code="shopee", client_id="", api_key="",
            extra_config={}, status=1,
        )
        result = await adapter.validate_credentials(platform=platform)
        assert result is False

    async def test_validate_credentials_missing_access_token(
            self, adapter: ShopeeListingAdapter):
        """缺少 access_token 时应返回 False"""
        platform = Platform(
            id=3, code="shopee", client_id="123", api_key="key123",
            extra_config={"shop_id": 1}, status=1,
        )
        result = await adapter.validate_credentials(platform=platform)
        assert result is False

    async def test_validate_credentials_api_error(
            self, adapter: ShopeeListingAdapter, shopee_platform: Platform):
        """API 返回 error != 0 时应返回 False"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "error": 1001,
                "message": "Unauthorized",
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(transport=MockTransport(handler), base_url="https://api.example.com")

            result = await adapter.validate_credentials(platform=shopee_platform)

        assert result is False

    async def test_signature_generation(self, adapter: ShopeeListingAdapter):
        """_sign 应生成正确的 HMAC-SHA256"""
        # 使用已知的测试值
        sign = adapter._sign(
            api_key="test-key",
            partner_id=12345,
            api_path="/api/v2/product/add",
            timestamp=1718000000,
            access_token="test-token",
            shop_id=67890,
        )
        # 验证签名是 64 字符的 hex 字符串
        assert len(sign) == 64
        assert all(c in "0123456789abcdef" for c in sign)
        # 确认是确定性签名
        sign2 = adapter._sign(
            api_key="test-key",
            partner_id=12345,
            api_path="/api/v2/product/add",
            timestamp=1718000000,
            access_token="test-token",
            shop_id=67890,
        )
        assert sign == sign2

    async def test_build_auth_params(self, adapter: ShopeeListingAdapter,
                                      shopee_platform: Platform):
        """_build_auth_params 应包含所有必需字段"""
        params = adapter._build_auth_params(shopee_platform, "/api/v2/product/add")

        assert params["partner_id"] == 123456
        assert params["shop_id"] == 654321
        assert params["access_token"] == "test-access-token"
        assert isinstance(params["timestamp"], int)
        assert params["timestamp"] > 0
        assert len(params["sign"]) == 64


# =================================================================== #
#  Wildberries Adapter Tests
# =================================================================== #

class TestWildberriesListingAdapter:
    """Wildberries 真实 API 适配器测试"""

    @pytest.fixture
    def adapter(self) -> WildberriesListingAdapter:
        return WildberriesListingAdapter()

    @pytest.fixture
    def wb_platform(self) -> Platform:
        return Platform(
            id=3, code="wb", name="Wildberries",
            api_base_url="https://content-api.wildberries.ru",
            api_key="test-wb-token", client_id="", extra_config={}, status=1,
        )

    async def test_publish_success(self, adapter: WildberriesListingAdapter,
                                    wb_platform: Platform, product: Product,
                                    skus: list[Sku], prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """publish() 应返回正确的 PublishResult"""
        def handler(request: httpx.Request) -> httpx.Response:
            assert request.url.path.endswith("/api/v3/cards/upload")
            assert "Authorization" in request.headers
            assert request.headers["Authorization"] == "Bearer test-wb-token"
            body = json.loads(request.read())
            assert body[0]["vendorCode"] == "SKU-001"
            assert "sizes" in body[0]
            return httpx.Response(200, json={
                "data": [{"nmID": 12345678, "vendorCode": "SKU-001", "imtID": 999}],
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://content-api.wildberries.ru",
                headers={
                    "Authorization": "Bearer test-wb-token",
                    "Content-Type": "application/json",
                },
            )

            result = await adapter.publish(
                product=product, platform=wb_platform,
                skus=skus, prices=prices, inventories=inventories,
            )

        assert result.platform_product_id.startswith("wb-")
        assert "wildberries.ru" in result.platform_url
        assert "nmID" in result.sync_message

    async def test_publish_no_skus(self, adapter: WildberriesListingAdapter,
                                    wb_platform: Platform, product: Product,
                                    prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """无 SKU 时应抛出 RuntimeError"""
        with pytest.raises(RuntimeError, match="至少需要一个 SKU"):
            await adapter.publish(
                product=product, platform=wb_platform, skus=[],
                prices=prices, inventories=inventories,
            )

    async def test_publish_no_image(self, adapter: WildberriesListingAdapter,
                                     wb_platform: Platform, product: Product,
                                     skus: list[Sku], prices: dict[int, Price],
                                     inventories: dict[int, Inventory]):
        """缺少主图时应抛出 RuntimeError"""
        product.main_image = None
        with pytest.raises(RuntimeError, match="缺少主图"):
            await adapter.publish(
                product=product, platform=wb_platform, skus=skus,
                prices=prices, inventories=inventories,
            )

    async def test_publish_api_error(self, adapter: WildberriesListingAdapter,
                                      wb_platform: Platform, product: Product,
                                      skus: list[Sku], prices: dict[int, Price],
                                      inventories: dict[int, Inventory]):
        """API 返回错误时应抛出 RuntimeError"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(400, json={"error": "Invalid token"})

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://content-api.wildberries.ru",
            )
            with pytest.raises(RuntimeError, match="Invalid token"):
                await adapter.publish(
                    product=product, platform=wb_platform, skus=skus,
                    prices=prices, inventories=inventories,
                )

    async def test_sync_status_synced(self, adapter: WildberriesListingAdapter,
                                       wb_platform: Platform):
        """sync_status 应返回 synced"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "data": [{"status": {"state": "LOADED"}}],
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://content-api.wildberries.ru",
            )
            status = await adapter.sync_status(
                listing_id=1001, platform=wb_platform,
                platform_product_id="wb-12345678",
            )

        assert status == "synced"

    async def test_sync_status_in_progress(self, adapter: WildberriesListingAdapter,
                                            wb_platform: Platform):
        """sync_status 应返回 in_progress"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "data": [{"status": {"state": "IN_PROCESS"}}],
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://content-api.wildberries.ru",
            )
            status = await adapter.sync_status(
                listing_id=1001, platform=wb_platform,
                platform_product_id="wb-12345678",
            )
        assert status == "in_progress"

    async def test_sync_status_no_data(self, adapter: WildberriesListingAdapter,
                                        wb_platform: Platform):
        """无 data 时应返回 unknown"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={"data": []})

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://content-api.wildberries.ru",
            )
            status = await adapter.sync_status(
                listing_id=1001, platform=wb_platform,
                platform_product_id="wb-12345678",
            )
        assert status == "unknown"

    async def test_validate_credentials_success(self, adapter: WildberriesListingAdapter,
                                                 wb_platform: Platform):
        """凭证有效时应返回 True"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={"data": []})

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://content-api.wildberries.ru",
            )
            result = await adapter.validate_credentials(platform=wb_platform)
        assert result is True

    async def test_validate_credentials_missing_key(self, adapter: WildberriesListingAdapter):
        """缺少 api_key 时应返回 False"""
        platform = Platform(id=3, code="wb", name="WB", api_key="", status=1)
        result = await adapter.validate_credentials(platform=platform)
        assert result is False

    async def test_validate_credentials_api_error(self, adapter: WildberriesListingAdapter,
                                                   wb_platform: Platform):
        """API 返回非 200 时应返回 False"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(401, json={"error": "Unauthorized"})

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://content-api.wildberries.ru",
            )
            result = await adapter.validate_credentials(platform=wb_platform)
        assert result is False


# =================================================================== #
#  Amazon Adapter Tests
# =================================================================== #

class TestAmazonListingAdapter:
    """Amazon SP-API 适配器测试"""

    @pytest.fixture
    def adapter(self) -> AmazonListingAdapter:
        return AmazonListingAdapter()

    @pytest.fixture
    def amz_platform(self) -> Platform:
        return Platform(
            id=4, code="amazon", name="Amazon",
            api_base_url="https://sellingpartnerapi-eu.amazon.com",
            client_id="test-client-id",
            api_key="test-client-secret",
            extra_config={
                "refresh_token": "test-refresh-token",
                "aws_access_key": "test-aws-key",
                "aws_secret_key": "test-aws-secret",
                "seller_id": "TESTSELLER",
                "marketplace_id": "A1PA6795UKMFR9",
            },
            status=1,
        )

    async def test_publish_success(self, adapter: AmazonListingAdapter,
                                    amz_platform: Platform, product: Product,
                                    skus: list[Sku], prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """publish() 应返回正确的 PublishResult"""
        call_count = 0

        async def handler(request: httpx.Request) -> httpx.Response:
            nonlocal call_count
            call_count += 1
            if call_count == 1:
                # LWA token
                return httpx.Response(200, json={"access_token": "test-access-token", "expires_in": 3600})
            # SP-API request
            assert "x-amz-access-token" in request.headers
            assert "Authorization" in request.headers
            return httpx.Response(200, json={"sellingPartnerId": "AMZPROD001"})

        async def lwa_handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={"access_token": "test-access-token", "expires_in": 3600})

        # Patch LWA token fetch
        original_get_token = adapter._get_access_token

        async def mock_get_token(platform):
            return "test-access-token"

        adapter._get_access_token = mock_get_token

        # Patch _signed_request to use MockTransport
        original_signed = adapter._signed_request

        async def mock_signed(method, path, platform, token, payload=None):
            return httpx.Response(200, json={"sellingPartnerId": "AMZPROD001"})

        adapter._signed_request = mock_signed

        try:
            result = await adapter.publish(
                product=product, platform=amz_platform,
                skus=skus, prices=prices, inventories=inventories,
            )
        finally:
            adapter._get_access_token = original_get_token
            adapter._signed_request = original_signed

        assert result.platform_product_id.startswith("amz-")
        assert "AMZPROD001" in result.sync_message or "AMZPROD001" in result.platform_product_id

    async def test_publish_no_skus(self, adapter: AmazonListingAdapter,
                                    amz_platform: Platform, product: Product,
                                    prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """无 SKU 时应抛出 RuntimeError"""
        with pytest.raises(RuntimeError, match="至少需要一个 SKU"):
            await adapter.publish(
                product=product, platform=amz_platform, skus=[],
                prices=prices, inventories=inventories,
            )

    async def test_publish_no_seller_id(self, adapter: AmazonListingAdapter,
                                         amz_platform: Platform, product: Product,
                                         skus: list[Sku], prices: dict[int, Price],
                                         inventories: dict[int, Inventory]):
        """缺少 seller_id 时应抛出 RuntimeError"""
        amz_platform.extra_config = {}
        with pytest.raises(RuntimeError, match="缺少 seller_id"):
            await adapter.publish(
                product=product, platform=amz_platform, skus=skus,
                prices=prices, inventories=inventories,
            )

    async def test_sync_status_synced(self, adapter: AmazonListingAdapter,
                                       amz_platform: Platform):
        """sync_status 应返回 synced"""
        original_token = adapter._get_access_token
        original_signed = adapter._signed_request

        async def mock_token(p):
            return "test-token"

        async def mock_signed(method, path, platform, token, payload=None):
            return httpx.Response(200, json={
                "status": ["BUYABLE"],
                "sellingPartnerId": "AMZPROD001",
            })

        adapter._get_access_token = mock_token
        adapter._signed_request = mock_signed

        try:
            status = await adapter.sync_status(
                listing_id=1001, platform=amz_platform,
                platform_product_id="amz-AMZPROD001",
            )
        finally:
            adapter._get_access_token = original_token
            adapter._signed_request = original_signed

        assert status == "synced"

    async def test_sync_status_in_progress(self, adapter: AmazonListingAdapter,
                                            amz_platform: Platform):
        """sync_status 应返回 in_progress"""
        original_token = adapter._get_access_token
        original_signed = adapter._signed_request

        async def mock_token(p):
            return "test-token"

        async def mock_signed(method, path, platform, token, payload=None):
            return httpx.Response(200, json={
                "status": ["INACTIVE"],
                "sellingPartnerId": "AMZPROD001",
            })

        adapter._get_access_token = mock_token
        adapter._signed_request = mock_signed

        try:
            status = await adapter.sync_status(
                listing_id=1001, platform=amz_platform,
                platform_product_id="amz-AMZPROD001",
            )
        finally:
            adapter._get_access_token = original_token
            adapter._signed_request = original_signed

        assert status == "in_progress"

    async def test_validate_credentials_success(self, adapter: AmazonListingAdapter,
                                                 amz_platform: Platform):
        """凭证有效时应返回 True"""
        original_token = adapter._get_access_token
        original_signed = adapter._signed_request

        async def mock_token(p):
            return "test-token"

        async def mock_signed(method, path, platform, token, payload=None):
            return httpx.Response(200, json={"payload": [{"marketplaceId": "A1PA6795UKMFR9"}]})

        adapter._get_access_token = mock_token
        adapter._signed_request = mock_signed

        try:
            result = await adapter.validate_credentials(platform=amz_platform)
        finally:
            adapter._get_access_token = original_token
            adapter._signed_request = original_signed

        assert result is True

    async def test_validate_credentials_missing_fields(self, adapter: AmazonListingAdapter):
        """缺少必填字段时应返回 False"""
        platform = Platform(
            id=4, code="amazon", name="Amazon",
            extra_config={}, client_id="", api_key="", status=1,
        )
        result = await adapter.validate_credentials(platform=platform)
        assert result is False


# =================================================================== #
#  TikTok Shop Adapter Tests
# =================================================================== #

class TestTikTokShopListingAdapter:
    """TikTok Shop 真实 API 适配器测试"""

    @pytest.fixture
    def adapter(self) -> TikTokShopListingAdapter:
        return TikTokShopListingAdapter()

    @pytest.fixture
    def tt_platform(self) -> Platform:
        return Platform(
            id=5, code="tiktok", name="TikTok Shop",
            api_base_url="https://open-api.tiktokglobalshop.com",
            client_id="test-app-key",
            api_key="test-app-secret",
            extra_config={
                "access_token": "test-access-token",
                "shop_id": "test_shop_123",
            },
            status=1,
        )

    async def test_publish_success(self, adapter: TikTokShopListingAdapter,
                                    tt_platform: Platform, product: Product,
                                    skus: list[Sku], prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """publish() 应返回正确的 PublishResult"""
        def handler(request: httpx.Request) -> httpx.Response:
            assert request.url.path.endswith("/product/202309/products")
            assert "x-tts-access-token" in request.headers
            body = json.loads(request.read())
            assert "common" in body
            assert "product" in body
            assert body["product"]["product_name"] == "测试商品 AI 标题"
            return httpx.Response(200, json={
                "code": 0,
                "data": {"product_id": "tt-prod-001"},
                "message": "success",
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://open-api.tiktokglobalshop.com",
                headers={
                    "x-tts-access-token": "test-access-token",
                    "Content-Type": "application/json",
                },
            )

            result = await adapter.publish(
                product=product, platform=tt_platform,
                skus=skus, prices=prices, inventories=inventories,
            )

        assert result.platform_product_id.startswith("tt-")
        assert "tt-prod-001" in result.sync_message

    async def test_publish_no_skus(self, adapter: TikTokShopListingAdapter,
                                    tt_platform: Platform, product: Product,
                                    prices: dict[int, Price],
                                    inventories: dict[int, Inventory]):
        """无 SKU 时应抛出 RuntimeError"""
        with pytest.raises(RuntimeError, match="至少需要一个 SKU"):
            await adapter.publish(
                product=product, platform=tt_platform, skus=[],
                prices=prices, inventories=inventories,
            )

    async def test_publish_api_error(self, adapter: TikTokShopListingAdapter,
                                      tt_platform: Platform, product: Product,
                                      skus: list[Sku], prices: dict[int, Price],
                                      inventories: dict[int, Inventory]):
        """API 返回 code != 0 时应抛出 RuntimeError"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "code": 10001,
                "message": "Invalid access token",
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://open-api.tiktokglobalshop.com",
            )

            with pytest.raises(RuntimeError, match="Invalid access token"):
                await adapter.publish(
                    product=product, platform=tt_platform, skus=skus,
                    prices=prices, inventories=inventories,
                )

    async def test_publish_http_error(self, adapter: TikTokShopListingAdapter,
                                       tt_platform: Platform, product: Product,
                                       skus: list[Sku], prices: dict[int, Price],
                                       inventories: dict[int, Inventory]):
        """HTTP 400 时应抛出 RuntimeError"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(400, json={"message": "Bad request"})

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://open-api.tiktokglobalshop.com",
            )

            with pytest.raises(RuntimeError, match="Bad request"):
                await adapter.publish(
                    product=product, platform=tt_platform, skus=skus,
                    prices=prices, inventories=inventories,
                )

    async def test_sync_status_synced(self, adapter: TikTokShopListingAdapter,
                                       tt_platform: Platform):
        """sync_status 应返回 synced"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "code": 0,
                "data": {"status": "PUBLISHED"},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://open-api.tiktokglobalshop.com",
            )

            status = await adapter.sync_status(
                listing_id=1001, platform=tt_platform,
                platform_product_id="tt-tt-prod-001",
            )

        assert status == "synced"

    async def test_sync_status_in_progress(self, adapter: TikTokShopListingAdapter,
                                            tt_platform: Platform):
        """sync_status 应返回 in_progress"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "code": 0,
                "data": {"status": "UNDER_REVIEW"},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://open-api.tiktokglobalshop.com",
            )

            status = await adapter.sync_status(
                listing_id=1001, platform=tt_platform,
                platform_product_id="tt-tt-prod-001",
            )

        assert status == "in_progress"

    async def test_sync_status_rejected(self, adapter: TikTokShopListingAdapter,
                                         tt_platform: Platform):
        """sync_status 应返回 failed"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "code": 0,
                "data": {"status": "REJECTED"},
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://open-api.tiktokglobalshop.com",
            )

            status = await adapter.sync_status(
                listing_id=1001, platform=tt_platform,
                platform_product_id="tt-tt-prod-001",
            )

        assert status == "failed"

    async def test_sync_status_unknown_id(self, adapter: TikTokShopListingAdapter,
                                           tt_platform: Platform):
        """无效的 platform_product_id 应返回 unknown"""
        status = await adapter.sync_status(
            listing_id=1001, platform=tt_platform,
            platform_product_id="invalid-id",
        )
        assert status == "unknown"

    async def test_validate_credentials_success(self, adapter: TikTokShopListingAdapter,
                                                 tt_platform: Platform):
        """凭证有效时应返回 True"""
        def handler(request: httpx.Request) -> httpx.Response:
            assert request.url.path.endswith("/product/202309/shop")
            return httpx.Response(200, json={
                "code": 0,
                "data": {"shop_name": "Test Shop"},
                "message": "success",
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://open-api.tiktokglobalshop.com",
            )

            result = await adapter.validate_credentials(platform=tt_platform)

        assert result is True

    async def test_validate_credentials_missing_fields(self, adapter: TikTokShopListingAdapter):
        """缺少字段时应返回 False"""
        platform = Platform(
            id=5, code="tiktok", name="TikTok",
            extra_config={}, client_id="", api_key="", status=1,
        )
        result = await adapter.validate_credentials(platform=platform)
        assert result is False

    async def test_validate_credentials_api_error(self, adapter: TikTokShopListingAdapter,
                                                   tt_platform: Platform):
        """API 返回 code != 0 时应返回 False"""
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json={
                "code": 10001,
                "message": "Unauthorized",
            })

        with patch.object(adapter, "_client") as mock_client:
            mock_client.return_value = AsyncClient(
                transport=MockTransport(handler),
                base_url="https://open-api.tiktokglobalshop.com",
            )

            result = await adapter.validate_credentials(platform=tt_platform)

        assert result is False

    async def test_build_common_params(self, adapter: TikTokShopListingAdapter,
                                        tt_platform: Platform):
        """_build_common_params 应包含 sign"""
        params = adapter._build_common_params(tt_platform)
        assert params["app_key"] == "test-app-key"
        assert params["shop_id"] == "test_shop_123"
        assert "sign" in params
        assert len(params["sign"]) == 64
        assert "timestamp" in params
