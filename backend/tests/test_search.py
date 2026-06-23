"""搜索接口测试"""



class TestSearch:
    """全局搜索"""

    async def test_search_products(self, async_client):
        """GET /api/search?q=xxx → 搜索商品"""
        resp = await async_client.get("/api/search", params={"q": "跑鞋"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200

    async def test_search_with_empty_query(self, async_client):
        """GET /api/search?q= → 空搜索词应返回 422"""
        resp = await async_client.get("/api/search", params={"q": ""})
        assert resp.status_code == 422
