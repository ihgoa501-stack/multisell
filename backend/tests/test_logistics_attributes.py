"""物流基础字段第一阶段测试"""


async def _create_product(async_client, **overrides):
    payload = {
        "name": "物流测试商品",
        "unit": "件",
        "status": 0,
    }
    payload.update(overrides)
    resp = await async_client.post("/api/products", json=payload)
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]


async def _create_product_with_sku(async_client, **product_overrides):
    product = await _create_product(async_client, **product_overrides)
    specs_resp = await async_client.post(
        f"/api/products/{product['id']}/specs",
        json={"specs": [{"name": "颜色", "values": ["黑色"]}]},
    )
    assert specs_resp.status_code == 200
    assert specs_resp.json()["code"] == 200

    sku_resp = await async_client.post(f"/api/products/{product['id']}/skus/generate")
    assert sku_resp.status_code == 200
    sku_data = sku_resp.json()
    assert sku_data["code"] == 200
    return product, sku_data["data"]["skus"][0]


class TestLogisticsAttributes:
    async def test_create_product_returns_logistics_fields_and_complete_status(self, async_client):
        resp = await async_client.post(
            "/api/products",
            json={
                "name": "带物流字段商品",
                "unit": "件",
                "product_length_cm": 11.2,
                "product_width_cm": 6.3,
                "product_height_cm": 3.4,
                "product_weight_kg": 0.45,
                "package_length_cm": 12.2,
                "package_width_cm": 7.3,
                "package_height_cm": 4.4,
                "package_weight_kg": 0.6,
                "cargo_type": "battery",
            },
        )

        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        product = data["data"]
        assert product["product_length_cm"] == 11.2
        assert product["product_width_cm"] == 6.3
        assert product["product_height_cm"] == 3.4
        assert product["product_weight_kg"] == 0.45
        assert product["package_length_cm"] == 12.2
        assert product["package_width_cm"] == 7.3
        assert product["package_height_cm"] == 4.4
        assert product["package_weight_kg"] == 0.6
        assert product["cargo_type"] == "battery"
        assert product["logistics_status"] == "complete"
        assert product["logistics_status_name"] == "物流完整"

    async def test_update_product_logistics_fields_changes_logistics_status(self, async_client):
        product = await _create_product(
            async_client,
            product_length_cm=10.0,
            product_width_cm=8.0,
            product_height_cm=5.0,
            product_weight_kg=0.7,
        )

        detail_resp = await async_client.get(f"/api/products/{product['id']}")
        assert detail_resp.status_code == 200
        detail = detail_resp.json()["data"]
        assert detail["logistics_status"] == "incomplete"
        assert detail["logistics_status_name"] == "物流不完整"

        update_resp = await async_client.put(
            f"/api/products/{product['id']}",
            json={
                "package_length_cm": 16.0,
                "package_width_cm": 10.0,
                "package_height_cm": 7.0,
                "package_weight_kg": 1.1,
            },
        )
        assert update_resp.status_code == 200
        updated = update_resp.json()["data"]
        assert updated["product_length_cm"] == 10.0
        assert updated["product_width_cm"] == 8.0
        assert updated["product_height_cm"] == 5.0
        assert updated["product_weight_kg"] == 0.7
        assert updated["logistics_status"] == "complete"
        assert updated["logistics_status_name"] == "物流完整"

    async def test_product_list_returns_logistics_status(self, async_client):
        incomplete = await _create_product(
            async_client,
            name="物流不完整商品",
            package_length_cm=20.0,
            package_width_cm=10.0,
            package_height_cm=8.0,
        )
        complete = await _create_product(
            async_client,
            name="物流完整商品",
            package_length_cm=21.0,
            package_width_cm=11.0,
            package_height_cm=9.0,
            package_weight_kg=1.5,
        )

        resp = await async_client.get("/api/products", params={"page": 1, "page_size": 50})

        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        records = {item["id"]: item for item in data["records"]}
        assert records[incomplete["id"]]["logistics_status"] == "incomplete"
        assert records[incomplete["id"]]["logistics_status_name"] == "物流不完整"
        assert records[complete["id"]]["logistics_status"] == "complete"
        assert records[complete["id"]]["logistics_status_name"] == "物流完整"

    async def test_sku_returns_package_override_fields_and_keeps_weight_semantics(self, async_client):
        product, sku = await _create_product_with_sku(
            async_client,
            package_length_cm=25.0,
            package_width_cm=16.0,
            package_height_cm=9.0,
            package_weight_kg=1.8,
        )

        update_resp = await async_client.put(
            f"/api/skus/{sku['id']}",
            json={
                "weight": 0.9,
                "sku_length_cm": 26.0,
                "sku_width_cm": 17.0,
                "sku_height_cm": 10.0,
                "sku_weight_kg": 2.1,
            },
        )

        assert update_resp.status_code == 200
        update_data = update_resp.json()
        assert update_data["code"] == 200
        updated = update_data["data"]
        assert updated["weight"] == 0.9
        assert updated["sku_length_cm"] == 26.0
        assert updated["sku_width_cm"] == 17.0
        assert updated["sku_height_cm"] == 10.0
        assert updated["sku_weight_kg"] == 2.1

        detail_resp = await async_client.get(f"/api/skus/{sku['id']}")
        assert detail_resp.status_code == 200
        detail = detail_resp.json()["data"]
        assert detail["weight"] == 0.9
        assert detail["sku_weight_kg"] == 2.1

        list_resp = await async_client.get(f"/api/products/{product['id']}/skus")
        assert list_resp.status_code == 200
        list_data = {item["id"]: item for item in list_resp.json()["data"]}
        assert list_data[sku["id"]]["sku_length_cm"] == 26.0
        assert list_data[sku["id"]]["sku_width_cm"] == 17.0
        assert list_data[sku["id"]]["sku_height_cm"] == 10.0
        assert list_data[sku["id"]]["sku_weight_kg"] == 2.1
        assert list_data[sku["id"]]["weight"] == 0.9

        weight_only_resp = await async_client.put(
            f"/api/skus/{sku['id']}",
            json={"weight": 1.2},
        )
        assert weight_only_resp.status_code == 200
        weight_only = weight_only_resp.json()["data"]
        assert weight_only["weight"] == 1.2
        assert weight_only["sku_weight_kg"] == 2.1

        sku_weight_only_resp = await async_client.put(
            f"/api/skus/{sku['id']}",
            json={"sku_weight_kg": 2.4},
        )
        assert sku_weight_only_resp.status_code == 200
        sku_weight_only = sku_weight_only_resp.json()["data"]
        assert sku_weight_only["weight"] == 1.2
        assert sku_weight_only["sku_weight_kg"] == 2.4
