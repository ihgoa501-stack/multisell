"""商品管理 API 测试"""



class TestProducts:
    """商品 CRUD"""

    async def test_list_products(self, async_client):
        """GET /api/products → 商品列表（分页）"""
        resp = await async_client.get("/api/products", params={"page": 1, "page_size": 10})
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert "records" in data
        assert "total" in data
        assert isinstance(data["records"], list)

    async def test_list_products_with_filter(self, async_client):
        """GET /api/products → 按名称筛选"""
        resp = await async_client.get("/api/products", params={"name": "跑鞋"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_create_product(self, async_client):
        """POST /api/products → 创建商品"""
        resp = await async_client.post("/api/products", json={
            "name": "测试商品 pytest",
            "subtitle": "自动化测试创建",
            "unit": "个",
            "status": 0,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        product = data["data"]
        assert product["name"] == "测试商品 pytest"
        assert "id" in product
        # 清理
        await async_client.delete(f"/api/products/{product['id']}")

    async def test_get_product_detail(self, async_client):
        """GET /api/products/{id} → 商品详情"""
        # 先创建一个
        create_resp = await async_client.post("/api/products", json={
            "name": "测试详情商品",
            "unit": "个",
        })
        pid = create_resp.json()["data"]["id"]

        resp = await async_client.get(f"/api/products/{pid}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["name"] == "测试详情商品"

        # 清理
        await async_client.delete(f"/api/products/{pid}")

    async def test_update_product(self, async_client):
        """PUT /api/products/{id} → 更新商品"""
        create_resp = await async_client.post("/api/products", json={
            "name": "更新前商品",
            "unit": "个",
        })
        pid = create_resp.json()["data"]["id"]

        resp = await async_client.put(f"/api/products/{pid}", json={
            "name": "更新后商品",
            "subtitle": "已更新",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["name"] == "更新后商品"

        await async_client.delete(f"/api/products/{pid}")

    async def test_delete_product(self, async_client):
        """DELETE /api/products/{id} → 删除商品"""
        create_resp = await async_client.post("/api/products", json={
            "name": "待删除商品",
            "unit": "个",
        })
        pid = create_resp.json()["data"]["id"]

        resp = await async_client.delete(f"/api/products/{pid}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_batch_update_status(self, async_client):
        """POST /api/products/batch/status → 批量修改状态"""
        resp = await async_client.post("/api/products/batch/status", json={
            "ids": [1],
            "status": 0,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_batch_delete(self, async_client):
        """POST /api/products/batch/delete → 批量删除（无ID则应返回错误）"""
        resp = await async_client.post("/api/products/batch/delete", json={
            "ids": []
        })
        data = resp.json()
        assert data["code"] != 200


class TestCategories:
    """分类管理"""

    async def test_list_categories(self, async_client):
        """GET /api/categories/tree → 分类树"""
        resp = await async_client.get("/api/categories/tree")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_create_category(self, async_client):
        """POST /api/categories → 创建分类"""
        import random
        suffix = random.randint(10000, 99999)
        resp = await async_client.post("/api/categories", json={
            "name": f"测试分类_{suffix}",
        })
        assert resp.status_code == 200
        data = resp.json()
        if data["code"] == 200:
            cid = data["data"]["id"]
            await async_client.delete(f"/api/categories/{cid}")


class TestBrands:
    """品牌管理"""

    async def test_list_brands(self, async_client):
        """GET /api/brands → 品牌列表"""
        resp = await async_client.get("/api/brands")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_create_brand(self, async_client):
        """POST /api/brands → 创建品牌"""
        import random
        suffix = random.randint(10000, 99999)
        resp = await async_client.post("/api/brands", json={
            "name": f"测试品牌_{suffix}",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert "id" in data["data"]


class TestSuppliers:
    """供应商管理"""

    async def test_list_suppliers(self, async_client):
        """GET /api/suppliers → 供应商列表"""
        resp = await async_client.get("/api/suppliers")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_create_supplier(self, async_client):
        """POST /api/suppliers → 创建供应商"""
        import random
        suffix = random.randint(10000, 99999)
        resp = await async_client.post("/api/suppliers", json={
            "name": f"测试供应商_{suffix}",
            "contact_person": "张三",
            "contact_phone": "13800138000",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
