"""新模块冒烟测试 — 验证路由注册和基本功能"""


def _check_endpoint(client, path: str, method: str = "GET") -> tuple[int, dict]:
    """请求端点并返回状态码和 JSON"""
    if method == "GET":
        resp = client.get(path)
    elif method == "POST":
        resp = client.post(path, json={})
    elif method == "PUT":
        resp = client.put(path, json={})
    else:
        resp = client.get(path)
    return resp.status_code, resp.json()


class TestAllocationModule:
    """库存分配模块"""

    async def test_list_warehouses(self, async_client):
        resp = await async_client.get("/api/warehouses")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] in (200, 404)

    async def test_list_allocation_rules(self, async_client):
        resp = await async_client.get("/api/allocation-rules")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200


class TestFinanceModule:
    """财务管理模块"""

    async def test_list_accounts(self, async_client):
        resp = await async_client.get("/api/finance/accounts")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_list_transactions(self, async_client):
        resp = await async_client.get("/api/finance/transactions")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200


class TestNotificationModule:
    """通知模块"""

    async def test_list_notifications(self, async_client):
        resp = await async_client.get("/api/notifications")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200


class TestOrderImportModule:
    """订单导入模块"""

    async def test_list_imports(self, async_client):
        resp = await async_client.get("/api/order-imports")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200


class TestPlatformFeeModule:
    """平台费用模块"""

    async def test_list_fee_rules(self, async_client):
        resp = await async_client.get("/api/platform-fee/rules")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200


class TestSettlementModule:
    """结算模块"""

    async def test_list_settlements(self, async_client):
        resp = await async_client.get("/api/settlements")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200


class TestImportBatchModule:
    """导入批次模块"""

    async def test_list_import_batches(self, async_client):
        resp = await async_client.get("/api/import/batches")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200


class TestExceptionModule:
    """异常工作台模块"""

    async def test_list_exceptions(self, async_client):
        resp = await async_client.get("/api/exceptions")
        assert resp.status_code == 200
        data = resp.json()
        assert "code" in data


class TestListingTaskModule:
    """多平台上架任务模块"""

    async def test_list_listing_tasks(self, async_client):
        resp = await async_client.get("/api/listing-tasks")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
