"""发布管理 API 测试"""


async def _create_platform(async_client, code: str = "mock"):
    resp = await async_client.post(
        "/api/platforms",
        json={
            "name": f"Mock {code}",
            "code": code,
            "api_base_url": f"https://{code}.example.com",
            "api_key": "secret-key",
            "client_id": "client-id",
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]


async def _create_product(async_client, **overrides):
    payload = {
        "name": "发布测试商品",
        "unit": "件",
        "status": 1,
    }
    payload.update(overrides)
    resp = await async_client.post("/api/products", json=payload)
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]


async def _make_product_publishable(async_client, product_id: int):
    specs_resp = await async_client.post(
        f"/api/products/{product_id}/specs",
        json={"specs": [{"name": "颜色", "values": ["黑色"]}]},
    )
    assert specs_resp.status_code == 200
    assert specs_resp.json()["code"] == 200

    sku_resp = await async_client.post(f"/api/products/{product_id}/skus/generate")
    assert sku_resp.status_code == 200
    sku = sku_resp.json()["data"]["skus"][0]

    price_resp = await async_client.post(
        "/api/prices",
        json={"sku_id": sku["id"], "price_type": "sale_price", "price": 129.9},
    )
    assert price_resp.status_code == 200
    assert price_resp.json()["code"] == 200

    inv_resp = await async_client.put(
        f"/api/inventory/{sku['id']}",
        json={"quantity": 5, "warehouse": "默认仓库", "safety_stock": 1},
    )
    assert inv_resp.status_code == 200
    assert inv_resp.json()["code"] == 200
    return sku


class TestListings:
    async def test_publish_rejects_product_with_incomplete_logistics_data(
        self, async_client
    ):
        platform = await _create_platform(async_client, code="mocklogistics")
        product = await _create_product(async_client, main_image="/static/demo.jpg")
        await _make_product_publishable(async_client, product["id"])

        resp = await async_client.post(
            f"/api/products/{product['id']}/publish/{platform['id']}"
        )

        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 400
        assert "物流数据不完整" in data["message"]
        missing = data["data"]["missing_requirements"]
        assert "logistics" in missing

    async def test_publish_rejects_incomplete_product_with_missing_requirements(
        self, async_client
    ):
        platform = await _create_platform(async_client, code="mockincomplete")
        product = await _create_product(async_client)

        resp = await async_client.post(
            f"/api/products/{product['id']}/publish/{platform['id']}"
        )

        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 400
        missing = data["data"]["missing_requirements"]
        assert "main_image" in missing
        assert "sku" in missing
        assert "price" in missing
        assert "inventory" in missing

    async def test_mock_publish_creates_synced_listing(self, async_client):
        platform = await _create_platform(async_client, code="mockpublish")
        product = await _create_product(
            async_client,
            main_image="/static/demo.jpg",
            package_length_cm=20.0,
            package_width_cm=12.0,
            package_height_cm=8.0,
            package_weight_kg=1.6,
        )
        await _make_product_publishable(async_client, product["id"])

        resp = await async_client.post(
            f"/api/products/{product['id']}/publish/{platform['id']}"
        )

        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        listing = data["data"]
        assert listing["status"] == "synced"
        assert listing["platform_product_id"].startswith("mockpublish-")
        assert (
            listing["platform_url"]
            == f"https://mockpublish.example.com/products/{product['id']}"
        )

        detail_resp = await async_client.get(f"/api/products/{product['id']}/listings")
        assert detail_resp.status_code == 200
        records = detail_resp.json()["data"]
        assert records[0]["status"] == "synced"

    async def test_publish_failure_records_failed_listing(self, async_client):
        platform = await _create_platform(async_client, code="mockfail")
        product = await _create_product(
            async_client,
            main_image="/static/demo.jpg",
            package_length_cm=20.0,
            package_width_cm=12.0,
            package_height_cm=8.0,
            package_weight_kg=1.6,
        )
        await _make_product_publishable(async_client, product["id"])

        resp = await async_client.post(
            f"/api/products/{product['id']}/publish/{platform['id']}"
        )

        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 500
        assert data["data"]["status"] == "failed"
        assert "mock publish failed" in data["data"]["sync_message"]

        detail_resp = await async_client.get(f"/api/products/{product['id']}/listings")
        assert detail_resp.status_code == 200
        records = detail_resp.json()["data"]
        assert records[0]["status"] == "failed"
        assert "mock publish failed" in records[0]["sync_message"]

    async def test_platform_api_key_is_never_returned(self, async_client):
        platform = await _create_platform(async_client, code="secretmock")

        list_resp = await async_client.get("/api/platforms")
        assert list_resp.status_code == 200
        assert "api_key" not in list_resp.json()["data"][0]

        detail_resp = await async_client.get(f"/api/platforms/{platform['id']}")
        assert detail_resp.status_code == 200
        assert "api_key" not in detail_resp.json()["data"]
