"""仪表盘 & 报表测试"""

import pytest


class TestDashboard:
    """仪表盘"""

    async def test_dashboard_stats(self, async_client):
        """GET /api/dashboard/stats → 仪表盘统计"""
        resp = await async_client.get("/api/dashboard/stats")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        stats = data["data"]
        # 验证关键字段
        assert "products" in stats
        assert "inventory" in stats
        assert "brands" in stats
        assert "suppliers" in stats
        assert "platforms" in stats
        assert "orders" in stats
        assert "finance" in stats
        # products 字段（skus 移入 products 内部）
        assert "total" in stats["products"]
        assert "on_shelf" in stats["products"]
        assert "draft" in stats["products"]
        assert "skus" in stats["products"]
        # platforms 字段
        assert "published" in stats["platforms"]
        assert "detail" in stats["platforms"]
        # 近期动态
        assert "recent_logs" in stats


class TestReports:
    """报表统计"""

    async def test_product_stats(self, async_client):
        """GET /api/reports/product-stats → 商品统计报表"""
        resp = await async_client.get("/api/reports/product-stats")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert "total" in data["data"]
        assert "on_shelf" in data["data"]
        assert "draft" in data["data"]
        assert "off_shelf" in data["data"]
        assert "category_distribution" in data["data"]

    async def test_platform_stats(self, async_client):
        """GET /api/reports/platform-stats → 平台发布统计"""
        resp = await async_client.get("/api/reports/platform-stats")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert "items" in data["data"]
        if data["data"]["items"]:
            item = data["data"]["items"][0]
            assert "platform_name" in item
            assert "published" in item
            assert "pending" in item
            assert "failed" in item
