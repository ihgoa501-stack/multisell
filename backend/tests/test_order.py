"""订单管理 API 测试"""


async def _create_sellable_sku(async_client):
    product_resp = await async_client.post(
        "/api/products",
        json={"name": "订单测试商品", "unit": "件", "status": 1},
    )
    assert product_resp.status_code == 200
    product = product_resp.json()["data"]

    specs_resp = await async_client.post(
        f"/api/products/{product['id']}/specs",
        json={"specs": [{"name": "颜色", "values": ["黑色"]}]},
    )
    assert specs_resp.status_code == 200
    assert specs_resp.json()["code"] == 200

    sku_resp = await async_client.post(f"/api/products/{product['id']}/skus/generate")
    assert sku_resp.status_code == 200
    sku = sku_resp.json()["data"]["skus"][0]

    price_resp = await async_client.post(
        "/api/prices",
        json={"sku_id": sku["id"], "price_type": "sale_price", "price": 99.5},
    )
    assert price_resp.status_code == 200
    assert price_resp.json()["code"] == 200

    # 确保有库存
    inv_resp = await async_client.put(
        f"/api/inventory/{sku['id']}",
        json={"quantity": 20},
    )
    assert inv_resp.status_code == 200, inv_resp.text

    return product, sku


async def _create_order(async_client, sku_id: int, quantity: int = 2):
    resp = await async_client.post(
        "/api/orders",
        json={
            "recipient_name": "张三",
            "recipient_phone": "13800138000",
            "shipping_address": "上海市测试路 1 号",
            "payment_method": "mock",
            "shipping_fee": 8,
            "remark": "测试订单",
            "items": [{"sku_id": sku_id, "quantity": quantity}],
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]


class TestOrders:
    async def test_create_order_snapshots_items_and_totals(self, async_client):
        product, sku = await _create_sellable_sku(async_client)

        order = await _create_order(async_client, sku["id"], quantity=2)

        assert order["order_no"].startswith("MS")
        assert order["status"] == "pending"
        assert order["product_name"] == product["name"]
        assert order["quantity"] == 2
        assert order["total_amount"] == 199.0
        assert order["shipping_fee"] == 8.0
        assert order["pay_amount"] == 207.0
        assert order["items"][0]["sku_id"] == sku["id"]
        assert order["items"][0]["product_name"] == product["name"]
        assert order["items"][0]["unit_price"] == 99.5
        assert order["items"][0]["subtotal"] == 199.0

    async def test_list_and_detail_orders_for_frontend_contract(self, async_client):
        _product, sku = await _create_sellable_sku(async_client)
        created = await _create_order(async_client, sku["id"], quantity=1)

        list_resp = await async_client.get("/api/orders", params={"page": 1, "page_size": 20})

        assert list_resp.status_code == 200
        listing = list_resp.json()
        assert listing["code"] == 200
        assert listing["total"] >= 1
        row = next(item for item in listing["records"] if item["id"] == created["id"])
        assert set(["order_no", "product_name", "quantity", "total_amount", "status", "created_at"]).issubset(row)

        detail_resp = await async_client.get(f"/api/orders/{created['id']}")

        assert detail_resp.status_code == 200
        detail = detail_resp.json()["data"]
        assert detail["id"] == created["id"]
        assert detail["recipient_name"] == "张三"
        assert len(detail["items"]) == 1
        assert detail["status_logs"][0]["to_status"] == "pending"

    async def test_order_status_flow_updates_inventory(self, async_client):
        """订单状态流转正确影响库存"""
        product, sku = await _create_sellable_sku(async_client)
        order = await _create_order(async_client, sku["id"], quantity=2)

        inv_resp = await async_client.get(f"/api/inventory/{sku['id']}")
        assert inv_resp.json()["data"]["locked_quantity"] == 2

        await async_client.put(f"/api/orders/{order['id']}/status", json={"status": "paid"})
        inv_resp = await async_client.get(f"/api/inventory/{sku['id']}")
        assert inv_resp.json()["data"]["quantity"] == 18
        assert inv_resp.json()["data"]["locked_quantity"] == 0

    async def test_order_status_transitions_are_validated(self, async_client):
        _product, sku = await _create_sellable_sku(async_client)
        order = await _create_order(async_client, sku["id"], quantity=1)

        invalid_resp = await async_client.put(
            f"/api/orders/{order['id']}/status",
            json={"status": "shipped"},
        )
        assert invalid_resp.status_code == 200
        assert invalid_resp.json()["code"] == 400

        paid_resp = await async_client.put(
            f"/api/orders/{order['id']}/status",
            json={"status": "paid"},
        )
        assert paid_resp.status_code == 200
        assert paid_resp.json()["code"] == 200
        assert paid_resp.json()["data"]["status"] == "paid"

        shipped_resp = await async_client.put(
            f"/api/orders/{order['id']}/status",
            json={"status": "shipped"},
        )
        assert shipped_resp.status_code == 200
        assert shipped_resp.json()["code"] == 200
        assert shipped_resp.json()["data"]["status"] == "shipped"

        detail_resp = await async_client.get(f"/api/orders/{order['id']}")
        logs = detail_resp.json()["data"]["status_logs"]
        assert [log["to_status"] for log in logs] == ["pending", "paid", "shipped"]
