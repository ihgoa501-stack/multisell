"""全链路集成测试 — 商品→发布→订单→结算→财务

验证核心业务管线端到端可用性。
"""


def _ok(resp):
    """断言 HTTP 200 + 业务 code 200"""
    assert resp.status_code == 200, f"HTTP {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body.get("code") == 200, f"业务错误: {body}"
    return body.get("data")


class TestFullPipeline:
    _platform_id = None
    _category_id = None

    @classmethod
    async def _ensure_ref_data(cls, client):
        """确保参考数据（平台/分类）存在"""
        if cls._platform_id is not None:
            return

        import uuid

        code = f"pipe{uuid.uuid4().hex[:8]}"

        # 创建平台
        resp = await client.post(
            "/api/platforms",
            json={
                "name": f"PipeTest-{code}",
                "code": code,
                "api_base_url": "https://mock.example.com",
                "client_id": "test_client",
                "api_key": "test_key",
                "status": 1,
            },
        )
        data = _ok(resp)
        cls._platform_id = data["id"]

        # 创建分类
        resp = await client.post(
            "/api/categories",
            json={
                "name": "测试分类",
                "parent_id": 0,
                "level": 0,
                "sort_order": 1,
            },
        )
        data = _ok(resp)
        cls._category_id = data["id"]

    async def _create_product(self, client) -> dict:
        """创建基础商品"""
        await self._ensure_ref_data(client)
        resp = await client.post(
            "/api/products",
            json={
                "name": "全链路测试商品",
                "subtitle": "集成测试用",
                "unit": "件",
                "status": 1,
                "category_id": self._category_id,
                "product_length_cm": 20,
                "product_width_cm": 15,
                "product_height_cm": 10,
                "product_weight_kg": 0.5,
                "package_length_cm": 22,
                "package_width_cm": 17,
                "package_height_cm": 12,
                "package_weight_kg": 0.6,
                "cargo_type": "normal",
                "main_image": "https://example.com/test.jpg",
            },
        )
        product = _ok(resp)
        assert product["id"] > 0
        return product

    async def _create_sku(self, client, product_id: int) -> dict:
        """创建规格 + 生成SKU + 设置所有SKU的价格+库存"""
        # 定义规格
        resp = await client.post(
            f"/api/products/{product_id}/specs",
            json={
                "specs": [{"name": "颜色", "values": ["黑色", "白色"]}],
            },
        )
        _ok(resp)

        # 生成 SKU
        resp = await client.post(f"/api/products/{product_id}/skus/generate")
        data = _ok(resp)
        skus = data["skus"]
        assert len(skus) == 2
        sku = skus[0]

        # 为所有 SKU 设置价格 + 库存（publish 会校验所有 SKU）
        for s in skus:
            resp = await client.post(
                "/api/prices",
                json={
                    "sku_id": s["id"],
                    "price_type": "sale_price",
                    "price": 199.0,
                },
            )
            _ok(resp)
            resp = await client.post(
                "/api/prices",
                json={
                    "sku_id": s["id"],
                    "price_type": "cost_price",
                    "price": 120.0,
                },
            )
            _ok(resp)
            resp = await client.put(f"/api/inventory/{s['id']}", json={"quantity": 100})
            _ok(resp)

        return sku

    async def _create_order(self, client, sku_id: int) -> dict:
        """创建订单"""
        resp = await client.post(
            "/api/orders",
            json={
                "recipient_name": "测试用户",
                "recipient_phone": "13800138000",
                "shipping_address": "测试地址",
                "payment_method": "card",
                "shipping_fee": 15.0,
                "items": [{"sku_id": sku_id, "quantity": 2}],
            },
        )
        order = _ok(resp)
        assert order["order_no"].startswith("MS")
        assert order["status"] == "pending"
        assert order["total_amount"] == 398.0
        assert order["pay_amount"] == 413.0

        # 更新为已支付
        resp = await client.put(
            f"/api/orders/{order['id']}/status", json={"status": "paid"}
        )
        _ok(resp)

        return order

    # ═══════════════════════════════════════════════════════════
    # 测试用例
    # ═══════════════════════════════════════════════════════════

    async def test_full_pipeline_product_to_listing(self, async_client):
        """阶段1: 商品 → 发布到平台"""
        product = await self._create_product(async_client)
        await self._create_sku(async_client, product["id"])

        # 发布到平台 (mock adapter)
        resp = await async_client.post(
            f"/api/products/{product['id']}/publish/{self._platform_id}"
        )
        listing = _ok(resp)
        assert listing["status"] == "synced"

        # 检查发布状态
        resp = await async_client.get(f"/api/products/{product['id']}/listings")
        listings = _ok(resp)
        assert len(listings) >= 1
        assert listings[0]["platform_id"] == self._platform_id

    async def test_full_pipeline_order_flow(self, async_client):
        """阶段2: 商品 → 订单 → 运费计算"""
        product = await self._create_product(async_client)
        sku = await self._create_sku(async_client, product["id"])
        order = await self._create_order(async_client, sku["id"])

        # 检查订单详情
        resp = await async_client.get(f"/api/orders/{order['id']}")
        detail = _ok(resp)
        assert len(detail["items"]) == 1
        assert detail["items"][0]["product_name"] == product["name"]

        # 订单列表
        resp = await async_client.get("/api/orders")
        body = resp.json()
        assert body["code"] == 200
        records = body.get("data", {}).get("records") or body.get("records", [])
        assert len(records) >= 1

    async def test_full_pipeline_settlement(self, async_client):
        """阶段3: 结算模块 — 模拟数据生成 → 对账"""
        # 先有订单
        product = await self._create_product(async_client)
        sku = await self._create_sku(async_client, product["id"])
        await self._create_order(async_client, sku["id"])

        # 生成模拟结算
        resp = await async_client.post(
            f"/api/settlements/mock?platform_id={self._platform_id}&count=3"
        )
        mock_result = _ok(resp)
        assert mock_result["id"] > 0
        assert mock_result["settlement_no"].startswith("STL-")

        # 检查结算单列表
        resp = await async_client.get("/api/settlements")
        body = resp.json()
        settlements = body.get("data", {}).get("records") or body.get("records", [])
        assert len(settlements) >= 1

        settlement_id = settlements[0]["id"]

        # 检查明细
        resp = await async_client.get(f"/api/settlements/{settlement_id}/items")
        body = resp.json()
        items = body.get("data", {}).get("records") or body.get("records", [])
        assert len(items) >= 1

        # 执行对账
        resp = await async_client.post(
            f"/api/settlements/{settlement_id}/reconcile",
            json={"auto_match": True, "strategy": "by_order_no"},
        )
        result = _ok(resp)
        assert result["settlement_id"] == settlement_id
        assert result["total"] > 0

    async def test_full_pipeline_finance(self, async_client):
        """阶段4: 财务模块 — 利润汇总"""
        # 先有订单
        product = await self._create_product(async_client)
        sku = await self._create_sku(async_client, product["id"])
        await self._create_order(async_client, sku["id"])

        # 生成模拟财务账户
        resp = await async_client.post("/api/finance/mock")
        accounts_result = _ok(resp)
        assert accounts_result["accounts_created"] >= 0

        # 利润汇总
        resp = await async_client.get("/api/finance/profit-summary")
        summary = _ok(resp)
        assert summary["order_count"] >= 1
        assert summary["total_revenue"] > 0
        assert summary["total_profit"] is not None
        assert summary["profit_margin"] is not None

    async def test_full_pipeline_allocation(self, async_client):
        """阶段5: 库存分配"""
        product = await self._create_product(async_client)
        sku = await self._create_sku(async_client, product["id"])

        # 生成模拟仓库
        resp = await async_client.post("/api/warehouses/mock")
        mock_result = _ok(resp)
        assert mock_result["warehouses_created"] >= 1
        assert mock_result["rules_created"] >= 1

        # 查询仓库列表
        resp = await async_client.get("/api/warehouses")
        warehouses = _ok(resp)
        assert len(warehouses) >= 1

        # 查询SKU在仓库的库存分布
        resp = await async_client.get(f"/api/inventory/warehouse/{sku['id']}")
        inv_data = _ok(resp)
        assert isinstance(inv_data, list)

    async def test_full_pipeline_order_import(self, async_client):
        """阶段6: 订单导入 — 模拟生成"""
        # 先有商品 + SKU
        product = await self._create_product(async_client)
        await self._create_sku(async_client, product["id"])

        # 生成模拟导入订单
        resp = await async_client.post(
            f"/api/order-import/mock?platform_id={self._platform_id}&count=3"
        )
        result = _ok(resp)
        assert result["success"] >= 1
        assert result["total"] == 3
        assert len(result["orders"]) >= 1

        # 检查导入记录
        resp = await async_client.get("/api/order-imports")
        body = resp.json()
        records = body.get("data", {}).get("records") or body.get("records", [])
        assert len(records) >= 1
