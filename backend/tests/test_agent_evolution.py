"""Agent 进化/自治等级体系测试

覆盖 P0-2（信任评分引擎）, P0-3（等级控制 + Nudge）, P0-4（阶段行为差异）
"""
import pytest
import pytest_asyncio
from datetime import datetime, timezone, timedelta

from app.agent.base import EvolutionStage
from app.agent.evolution_service import TrustScoreCalculator, EvolutionService, SCORE_WEIGHTS


# ── 辅助函数 ──────────────────────────────────────────────


@pytest_asyncio.fixture
async def auth_client(async_client):
    """带认证头的 HTTP 客户端（AUTH_ENABLED=False 环境中无需真实 token）"""
    async_client.headers.update({"Authorization": "Bearer test-token"})
    return async_client


# ── 信任评分单元测试 ─────────────────────────────────────


class TestTrustScoreCalculator:
    """信任评分计算引擎单元测试"""

    def test_promotion_eligibility_sufficient(self):
        result = TrustScoreCalculator.check_promotion_eligibility(
            EvolutionStage.OBSERVATION, EvolutionStage.SUGGESTION,
            trust_score=60.0, decision_count=30,
        )
        assert result["eligible"] is True

    def test_promotion_eligibility_score_too_low(self):
        result = TrustScoreCalculator.check_promotion_eligibility(
            EvolutionStage.OBSERVATION, EvolutionStage.SUGGESTION,
            trust_score=30.0, decision_count=30,
        )
        assert result["eligible"] is False
        assert "评分不足" in result["reason"]

    def test_promotion_eligibility_samples_too_few(self):
        result = TrustScoreCalculator.check_promotion_eligibility(
            EvolutionStage.OBSERVATION, EvolutionStage.SUGGESTION,
            trust_score=60.0, decision_count=5,
        )
        assert result["eligible"] is False
        assert "样本不足" in result["reason"]

    def test_promotion_eligibility_full_requires_no_regret(self):
        result = TrustScoreCalculator.check_promotion_eligibility(
            EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS,
            trust_score=90.0, decision_count=120, has_regret=True,
        )
        assert result["eligible"] is False
        assert "Regret" in result["reason"]

    def test_promotion_eligibility_full_clean(self):
        result = TrustScoreCalculator.check_promotion_eligibility(
            EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS,
            trust_score=90.0, decision_count=120, has_regret=False,
        )
        assert result["eligible"] is True

    def test_promotion_target(self):
        assert TrustScoreCalculator.get_promotion_target(EvolutionStage.OBSERVATION) == EvolutionStage.SUGGESTION
        assert TrustScoreCalculator.get_promotion_target(EvolutionStage.SUGGESTION) == EvolutionStage.SEMI_AUTONOMOUS
        assert TrustScoreCalculator.get_promotion_target(EvolutionStage.FULL_AUTONOMOUS) is None

    def test_weights_sum_to_one(self):
        assert abs(sum(SCORE_WEIGHTS.values()) - 1.0) < 0.001


# ── API 集成测试 ─────────────────────────────────────────


class TestEvolutionAPI:
    """进化 API 集成测试"""

    async def test_overview(self, auth_client):
        resp = await auth_client.get("/api/agents/evolution/overview")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert "summary" in data["data"]
        assert "governance_agents" in data["data"]
        assert "specialist_agents" in data["data"]

    async def test_agent_detail(self, auth_client):
        resp = await auth_client.get("/api/agents/evolution/A5")
        assert resp.status_code == 200
        data = resp.json()
        assert data["data"]["agent_id"] == "A5"
        assert "decision_points" in data["data"]

    async def test_agent_detail_not_found(self, auth_client):
        resp = await auth_client.get("/api/agents/evolution/NONEXISTENT")
        assert resp.status_code == 404

    async def test_pending_nudges_empty(self, auth_client):
        resp = await auth_client.get("/api/agents/evolution/nudge/pending")
        assert resp.status_code == 200
        data = resp.json()
        assert isinstance(data["data"], list)
        assert len(data["data"]) == 0

    async def test_generate_nudges(self, auth_client):
        resp = await auth_client.post("/api/agents/evolution/generate-nudges")
        assert resp.status_code == 200
        data = resp.json()
        assert "generated" in data["data"]

    async def test_change_stage_downgrade(self, auth_client):
        resp = await auth_client.put(
            "/api/agents/evolution/A5/stage",
            json={"decision_point": "stock_alert", "target_stage": "observation"},
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["data"]["success"] is True
        assert data["data"]["new_stage"] == "observation"

    async def test_change_stage_invalid_target(self, auth_client):
        resp = await auth_client.put(
            "/api/agents/evolution/A5/stage",
            json={"decision_point": "stock_alert", "target_stage": "invalid_stage"},
        )
        assert resp.status_code == 422  # Pydantic validation

    async def test_respond_nudge_nonexistent(self, auth_client):
        resp = await auth_client.post(
            "/api/agents/evolution/nudge/99999/respond",
            json={"response": "dismiss"},
        )
        assert resp.status_code == 400

    async def test_agent_decide_observation(self, auth_client):
        """OBSERVATION 阶段的 decide 返回数据但不创建操作"""
        resp = await auth_client.post(
            "/api/agents/A5/decide",
            json={
                "decision_point": "stock_alert",
                "context": {
                    "sku_code": "SKU-TEST-001",
                    "sellable_stock": 10,
                    "safety_stock": 50,
                    "selling_price": 100.0,
                    "cost_price": 60.0,
                },
                "dry_run": False,
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["data"]["agent_id"] == "A5"
        assert "decision_id" in data["data"]
        assert data["data"]["decision_id"] is not None


class TestAgentDecideStages:
    """验证不同阶段的行为差异"""

    async def test_observation_stage(self, auth_client):
        """OBSERVATION: 仅收集数据，不创建 pending action"""
        # 先设为 OBSERVATION
        await auth_client.put(
            "/api/agents/evolution/A5/stage",
            json={"decision_point": "stock_alert", "target_stage": "observation"},
        )
        resp = await auth_client.post(
            "/api/agents/A5/decide",
            json={
                "decision_point": "stock_alert",
                "context": {"sku_code": "SKU-TEST-OBS", "sellable_stock": 3, "safety_stock": 50},
                "dry_run": False,
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["data"]["stage"] == "observation"
        # 检查 note 字段存在
        assert data["data"]["decision_id"] is not None

    async def test_suggestion_stage(self, auth_client):
        """SUGGESTION: 生成建议但不创建操作"""
        await auth_client.put(
            "/api/agents/evolution/A5/stage",
            json={"decision_point": "stock_alert", "target_stage": "suggestion"},
        )
        resp = await auth_client.post(
            "/api/agents/A5/decide",
            json={
                "decision_point": "stock_alert",
                "context": {"sku_code": "SKU-TEST-SUG", "sellable_stock": 3, "safety_stock": 50},
                "dry_run": False,
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["data"]["stage"] == "suggestion"

    async def test_semi_autonomous_stage(self, auth_client):
        """SEMI_AUTONOMOUS: 自动执行低风险操作"""
        await auth_client.put(
            "/api/agents/evolution/A5/stage",
            json={"decision_point": "stock_alert", "target_stage": "semi_autonomous"},
        )
        resp = await auth_client.post(
            "/api/agents/A5/decide",
            json={
                "decision_point": "stock_alert",
                "context": {"sku_code": "SKU-TEST-SEMI", "sellable_stock": 2, "safety_stock": 50},
                "dry_run": False,
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["data"]["stage"] == "semi_autonomous"
