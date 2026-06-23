"""健康检查 & 系统接口测试"""



class TestHealth:
    """系统健康检查"""

    async def test_health_check(self, async_client):
        """GET /api/health → 返回服务状态"""
        resp = await async_client.get("/api/health")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "ok"
        assert "service" in data
        assert "凌镜 LingMirror" in data["service"]
