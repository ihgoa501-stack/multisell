"""Agent 事件总线测试"""
import pytest


class TestEventBusRoutes:
    """事件路由定义"""

    @pytest.fixture
    def bus(self):
        from app.agent.event_bus import AgentEventBus
        return AgentEventBus()

    def test_default_routes_loaded(self, bus):
        """默认路由全部加载"""
        routes = bus.get_routes()
        assert len(routes) >= 15  # 至少有 15+ 条默认路由

    def test_inventory_event_routes(self, bus):
        """库存事件路由存在"""
        routes = bus.get_routes()
        types = {r["event_type"] for r in routes}
        assert "inventory.low_stock" in types
        assert "inventory.out_of_stock" in types

    def test_discount_event_routes(self, bus):
        """折扣事件路由存在"""
        routes = bus.get_routes()
        types = {r["event_type"] for r in routes}
        assert "discount.proposed" in types
        assert "price.changed" in types

    def test_custom_route_can_be_added(self, bus):
        """可以动态添加路由"""
        bus.add_route("custom.test", "A5", "stock_alert")
        routes = bus.get_routes()
        assert any(r["event_type"] == "custom.test" for r in routes)

    async def test_emit_no_match(self, bus):
        """未匹配路由 → 返回 0"""
        matched = await bus.emit("nonexistent.event", {})
        assert matched == 0


class TestEventBusMatch:
    """事件匹配逻辑"""

    @pytest.fixture
    def bus(self):
        from app.agent.event_bus import AgentEventBus
        return AgentEventBus()

    def test_exact_match(self, bus):
        """精确匹配"""
        from app.agent.event_bus import _match_event
        assert _match_event("inventory.low_stock", "inventory.low_stock") is True

    def test_prefix_match(self, bus):
        """前缀匹配"""
        from app.agent.event_bus import _match_event
        assert _match_event("inventory.low_stock", "inventory.") is True

    def test_no_match(self, bus):
        """不匹配"""
        from app.agent.event_bus import _match_event
        assert _match_event("order.created", "inventory") is False

    def test_wildcard_suffix(self, bus):
        """后缀通配"""
        from app.agent.event_bus import _match_event
        assert _match_event("inventory.low_stock", "inventory.*") is True

    def test_partial_match(self, bus):
        """部分前缀匹配"""
        from app.agent.event_bus import _match_event
        assert _match_event("inventory.low_stock.stockout", "inventory.low_stock") is True


class TestEventBusAPI:
    """事件总线 API"""

    async def test_list_routes(self, async_client):
        """GET /api/agents/events/routes → 事件路由列表"""
        resp = await async_client.get("/api/agents/events/routes")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert len(data["data"]) >= 15
        # 验证关键字段
        first = data["data"][0]
        assert "event_type" in first
        assert "agent_id" in first
        assert "decision_point" in first

    async def test_emit_event(self, async_client):
        """POST /api/agents/events/emit → 手动触发事件"""
        resp = await async_client.post(
            "/api/agents/events/emit",
            params={"event_type": "test.event", "source": "test"},
            json={},
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        # 未匹配路由，matched_routes 应为 0
        assert data["data"]["matched_routes"] == 0


class TestEventIntegration:
    """事件集成入口"""

    async def test_emit_helper(self, async_client):
        """from app.events import emit_agent_event"""
        from app.events import emit_agent_event
        matched = await emit_agent_event("test.integration", {"key": "value"}, "test")
        assert matched == 0  # 未匹配路由
