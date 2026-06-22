"""测试售后管理全流程"""

import pytest

pytestmark = pytest.mark.asyncio


async def _create_sellable_sku(async_client):
    """创建可销售SKU（含库存）。"""
    product_resp = await async_client.post(
        "/api/products",
        json={"name": "售后测试商品", "unit": "件", "status": 1},
    )
    assert product_resp.status_code == 200
    product = product_resp.json()["data"]

    specs_resp = await async_client.post(
        f"/api/products/{product['id']}/specs",
        json={"specs": [{"name": "颜色", "values": ["黑色"]}]},
    )
    assert specs_resp.status_code == 200

    sku_resp = await async_client.post(f"/api/products/{product['id']}/skus/generate")
    assert sku_resp.status_code == 200
    sku = sku_resp.json()["data"]["skus"][0]

    price_resp = await async_client.post(
        "/api/prices",
        json={"sku_id": sku["id"], "price_type": "sale_price", "price": 99.5},
    )
    assert price_resp.status_code == 200

    inv_resp = await async_client.put(
        f"/api/inventory/{sku['id']}",
        json={"quantity": 20},
    )
    assert inv_resp.status_code == 200

    return product, sku


async def _create_order(async_client, sku_id):
    """创建测试订单。"""
    resp = await async_client.post(
        "/api/orders",
        json={
            "recipient_name": "张三",
            "recipient_phone": "13800138000",
            "shipping_address": "上海市测试路 1 号",
            "payment_method": "mock",
            "shipping_fee": 8,
            "remark": "测试订单",
            "product_cost": 50,
            "items": [{"sku_id": sku_id, "quantity": 2}],
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]


async def _pay_order(async_client, order_id):
    """付款：将订单从 pending 变为 paid。"""
    resp = await async_client.put(
        f"/api/orders/{order_id}/status",
        json={"status": "paid"},
    )
    assert resp.status_code == 200, resp.text
    data = resp.json()
    assert data["code"] == 200, resp.text
    return data["data"]


async def _create_paid_order(async_client, sku_id):
    """创建订单并付款。"""
    order = await _create_order(async_client, sku_id)
    await _pay_order(async_client, order["id"])
    return order


class TestAfterSalesAPI:

    async def test_create_return_request(self, async_client):
        """创建售后单"""
        _, sku = await _create_sellable_sku(async_client)
        order = await _create_paid_order(async_client, sku["id"])

        resp = await async_client.post(
            "/api/aftersales",
            json={
                "order_id": order["id"],
                "sku_id": sku["id"],
                "item_id": order["items"][0]["id"],
                "return_quantity": 1,
                "reason": "customer_return",
                "refund_amount": 99.5,
            },
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200, resp.text
        after = data["data"]
        assert after["status"] == "requested"
        assert after["order_id"] == order["id"]
        assert after["sku_id"] == sku["id"]
        assert after["return_quantity"] == 1
        assert after["refund_amount"] == 99.5

    async def test_approve_return(self, async_client):
        """审批售后单"""
        _, sku = await _create_sellable_sku(async_client)
        order = await _create_paid_order(async_client, sku["id"])

        resp = await async_client.post(
            "/api/aftersales",
            json={
                "order_id": order["id"],
                "sku_id": sku["id"],
                "return_quantity": 1,
                "reason": "defective",
                "refund_amount": 50,
            },
        )
        after = resp.json()["data"]

        resp = await async_client.post(
            f"/api/aftersales/{after['id']}/approve",
            json={"inspection_result": "外观完好"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200, resp.text
        assert data["data"]["status"] == "approved"
        assert data["data"]["inspection_result"] == "外观完好"

    async def test_reject_return(self, async_client):
        """驳回售后单"""
        _, sku = await _create_sellable_sku(async_client)
        order = await _create_paid_order(async_client, sku["id"])

        resp = await async_client.post(
            "/api/aftersales",
            json={
                "order_id": order["id"],
                "sku_id": sku["id"],
                "return_quantity": 1,
                "reason": "defective",
                "refund_amount": 50,
            },
        )
        after = resp.json()["data"]

        resp = await async_client.post(
            f"/api/aftersales/{after['id']}/reject",
            json={"rejection_reason": "不符合退货条件"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200, resp.text
        assert data["data"]["status"] == "rejected"

    async def test_full_lifecycle(self, async_client):
        """完整生命周期: create -> approve -> receive -> refund"""
        _, sku = await _create_sellable_sku(async_client)
        order = await _create_paid_order(async_client, sku["id"])

        # 1. Create
        resp = await async_client.post(
            "/api/aftersales",
            json={
                "order_id": order["id"],
                "sku_id": sku["id"],
                "return_quantity": 1,
                "reason": "customer_return",
                "refund_amount": 50,
            },
        )
        assert resp.status_code == 200
        after = resp.json()["data"]
        after_id = after["id"]

        # 2. Approve
        resp = await async_client.post(
            f"/api/aftersales/{after_id}/approve",
            json={},
        )
        assert resp.json()["code"] == 200
        assert resp.json()["data"]["status"] == "approved"

        # 3. Receive (restock)
        resp = await async_client.post(
            f"/api/aftersales/{after_id}/receive",
            json={"inspection_result": "验货通过"},
        )
        assert resp.json()["code"] == 200
        assert resp.json()["data"]["status"] == "received"

        # 4. Refund
        resp = await async_client.post(
            f"/api/aftersales/{after_id}/refund",
            json={"note": "已退款"},
        )
        assert resp.json()["code"] == 200
        assert resp.json()["data"]["status"] == "refunded"

    async def test_list_and_detail(self, async_client):
        """列表和详情"""
        _, sku = await _create_sellable_sku(async_client)
        order = await _create_paid_order(async_client, sku["id"])

        resp = await async_client.post(
            "/api/aftersales",
            json={
                "order_id": order["id"],
                "sku_id": sku["id"],
                "return_quantity": 1,
                "reason": "customer_return",
                "refund_amount": 50,
            },
        )
        after = resp.json()["data"]

        # List
        resp = await async_client.get("/api/aftersales")
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200, resp.text
        ids = [r["id"] for r in data["records"]]
        assert after["id"] in ids

        # Detail
        resp = await async_client.get(f"/api/aftersales/{after['id']}")
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200, resp.text
        assert data["data"]["id"] == after["id"]

    async def test_status_transition_validation(self, async_client):
        """状态流转校验"""
        _, sku = await _create_sellable_sku(async_client)
        order = await _create_paid_order(async_client, sku["id"])

        resp = await async_client.post(
            "/api/aftersales",
            json={
                "order_id": order["id"],
                "sku_id": sku["id"],
                "return_quantity": 1,
                "reason": "defective",
                "refund_amount": 50,
            },
        )
        after = resp.json()["data"]

        # Cannot reject rejected -> rejected
        # First reject
        resp = await async_client.post(
            f"/api/aftersales/{after['id']}/reject",
            json={"rejection_reason": "不符合条件"},
        )
        assert resp.json()["code"] == 200

        # Then try to approve — should fail
        resp = await async_client.post(
            f"/api/aftersales/{after['id']}/approve",
            json={},
        )
        assert resp.json()["code"] != 200  # bad request
