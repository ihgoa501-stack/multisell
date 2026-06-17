"""订单导入 + 库存分配 — 功能测试"""


def _ok(resp):
    assert resp.status_code == 200, f"HTTP {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body.get("code") == 200, f"业务错误: {body}"
    return body.get("data")


class TestOrderImportAPI:

    async def test_order_import_mock(self, async_client):
        # 创建平台（确保平台存在）
        import uuid
        code = f"imp{uuid.uuid4().hex[:8]}"
        r = await async_client.post("/api/platforms", json={
            "name": f"Imp-{code}", "code": code, "api_key": "k",
        })
        pid = _ok(r)["id"]

        # 需要已有商品+SKU
        r = await async_client.post("/api/products", json={
            "name": "ImpTest", "unit": "件", "status": 1,
            "package_length_cm": 10, "package_width_cm": 10, "package_height_cm": 10,
            "package_weight_kg": 0.5, "cargo_type": "normal",
            "main_image": "https://ex.com/i.jpg",
        })
        prod = _ok(r)
        r = await async_client.post(f"/api/products/{prod['id']}/specs", json={
            "specs": [{"name": "C", "values": ["A"]}],
        })
        _ok(r)
        r = await async_client.post(f"/api/products/{prod['id']}/skus/generate")
        _ok(r)

        r = await async_client.post(f"/api/order-import/mock?platform_id={pid}&count=3")
        result = _ok(r)
        assert result["total"] == 3
        assert result["success"] == 3
        assert len(result["orders"]) == 3

    async def test_import_record_list(self, async_client):
        r = await async_client.get("/api/order-imports")
        body = r.json()
        records = body.get("data", {}).get("records") or body.get("records", [])
        assert isinstance(records, list)


class TestAllocationAPI:

    async def test_warehouse_and_rules(self, async_client):
        r = await async_client.post("/api/warehouses/mock")
        result = _ok(r)
        assert result["warehouses_created"] >= 1

        r = await async_client.get("/api/warehouses")
        warehouses = _ok(r)
        assert len(warehouses) >= 1

        r = await async_client.get("/api/allocation-rules")
        rules = _ok(r)
        assert len(rules) >= 1
