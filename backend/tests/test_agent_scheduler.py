"""Agent 调度引擎测试"""
import pytest
from app.database import async_session_factory


class TestSchedulerConfig:
    """调度器配置管理"""

    @pytest.fixture
    def scheduler(self):
        from app.agent.scheduler import AgentScheduler
        return AgentScheduler()

    def test_default_schedules_all_agents(self, scheduler):
        """所有 10 个 Agent 都有默认调度配置"""
        schedules = scheduler.get_schedules()
        assert len(schedules) == 10
        for agent_id in ["G1", "G2", "G3", "A1", "A2", "A3", "A4", "A5", "A6", "A7"]:
            assert agent_id in schedules

    def test_default_schedules_enabled(self, scheduler):
        """默认所有 Agent 调度为启用状态"""
        for cfg in scheduler.get_schedules().values():
            assert cfg.get("enabled") is True

    def test_default_schedules_have_interval(self, scheduler):
        """每个调度都有合理的间隔"""
        for agent_id, cfg in scheduler.get_schedules().items():
            interval = cfg.get("interval", 0)
            assert interval > 0, f"{agent_id} 间隔必须大于 0"
            assert interval >= 60, f"{agent_id} 间隔不低于 60s（当前 {interval}s）"

    def test_get_schedule(self, scheduler):
        """查询单个 Agent 配置"""
        cfg = scheduler.get_schedule("A5")
        assert cfg is not None
        assert cfg["interval"] == 900

    def test_get_schedule_not_found(self, scheduler):
        """不存在的 Agent 返回 None"""
        assert scheduler.get_schedule("NONEXISTENT") is None

    def test_update_schedule(self, scheduler):
        """更新调度配置"""
        ok = scheduler.update_schedule("A5", {"interval": 600, "enabled": False})
        assert ok is True
        cfg = scheduler.get_schedule("A5")
        assert cfg["interval"] == 600
        assert cfg["enabled"] is False

    def test_update_schedule_not_found(self, scheduler):
        """更新不存在的 Agent 返回 False"""
        ok = scheduler.update_schedule("NONEXISTENT", {"interval": 600})
        assert ok is False

    def test_schedules_immutable_snapshot(self, scheduler):
        """get_schedules 返回快照，修改不影响原始数据"""
        schedules = scheduler.get_schedules()
        original = schedules["A5"]["interval"]
        schedules["A5"]["interval"] = 99999
        assert scheduler.get_schedule("A5")["interval"] == original

    def test_entropy_config_in_scheduler(self):
        """熵系统配置正确"""
        from app.agent.scheduler import ENTROPY_INTERVAL
        assert ENTROPY_INTERVAL > 0
        assert ENTROPY_INTERVAL == 21600  # 6h


class TestSchedulerContextBuilder:
    """上下文构建器"""

    @pytest.fixture
    def builder(self):
        from app.agent.scheduler import SchedulerContextBuilder
        return SchedulerContextBuilder()

    async def test_build_generic(self, builder, async_client):
        """通用上下文的字段结构"""
        async with async_session_factory() as session:
            ctx = await builder.build_generic_context(session)
        assert "product_count" in ctx
        assert "sku_count" in ctx
        assert "timestamp" in ctx

    async def test_build_customer_service_no_orders(self, builder, async_client):
        """无订单时返回空列表"""
        async with async_session_factory() as session:
            contexts = await builder.build_customer_service_context(session)
        assert isinstance(contexts, list)


class TestSchedulerAPI:
    """调度管理 API"""

    async def test_list_schedules(self, async_client):
        """GET /api/agents/schedules → 返回所有 Agent 调度配置"""
        resp = await async_client.get("/api/agents/schedules")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        schedules = data["data"]
        assert isinstance(schedules, dict)
        assert len(schedules) == 10

    async def test_get_schedule(self, async_client):
        """GET /api/agents/schedules/{agent_id} → 单个配置"""
        resp = await async_client.get("/api/agents/schedules/A5")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["interval"] == 900
        assert data["data"]["enabled"] is True

    async def test_get_schedule_not_found(self, async_client):
        """不存在的 Agent 返回错误"""
        resp = await async_client.get("/api/agents/schedules/NONEXISTENT")
        data = resp.json()
        assert data["code"] == 404

    async def test_update_schedule(self, async_client):
        """PUT /api/agents/schedules/{agent_id} → 更新配置"""
        resp = await async_client.put(
            "/api/agents/schedules/A5",
            json={"interval": 600, "enabled": False},
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["data"]["interval"] == 600
        assert data["data"]["enabled"] is False

    async def test_update_schedule_not_found(self, async_client):
        """更新不存在的 Agent 返回错误"""
        resp = await async_client.put(
            "/api/agents/schedules/NONEXISTENT",
            json={"interval": 600},
        )
        data = resp.json()
        assert data["code"] == 404

    async def test_trigger_schedule(self, async_client):
        """POST /api/agents/schedules/{agent_id}/trigger → 手动触发调度"""
        resp = await async_client.post("/api/agents/schedules/A5/trigger")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        result = data["data"]
        assert result["agent_id"] == "A5"
        assert "decisions_made" in result
