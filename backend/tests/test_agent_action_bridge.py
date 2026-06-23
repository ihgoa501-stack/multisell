"""Agent → 动作中枢桥接测试"""
import pytest
from unittest.mock import patch

from app.agent.base import EvolutionStage
from app.agentos.schemas import RiskLevel


class TestAgentActionMapping:
    """辅助函数单元测试"""

    def test_map_replenish_to_inventory_allocate(self):
        from app.agent.service import AgentService
        assert AgentService._map_action_type("replenish") == "inventory_allocate"

    def test_map_price_review_to_profit_review(self):
        from app.agent.service import AgentService
        assert AgentService._map_action_type("price_review") == "profit_review"

    def test_map_discount_review_to_profit_review(self):
        from app.agent.service import AgentService
        assert AgentService._map_action_type("discount_review") == "profit_review"

    def test_map_ad_action_to_notify(self):
        from app.agent.service import AgentService
        assert AgentService._map_action_type("ad_action") == "notify"

    def test_map_unknown_preserves(self):
        from app.agent.service import AgentService
        assert AgentService._map_action_type("unknown_type") == "unknown_type"


class TestActionRiskDerivation:
    """风险等级推导测试"""

    def test_full_autonomous_low_risk(self):
        from app.agent.service import AgentService
        risk, need_approval = AgentService._derive_action_risk(
            "replenish", {}, EvolutionStage.FULL_AUTONOMOUS
        )
        assert risk == RiskLevel.LOW
        assert need_approval is False

    def test_suggestion_medium_with_approval(self):
        from app.agent.service import AgentService
        risk, need_approval = AgentService._derive_action_risk(
            "price_review", {}, EvolutionStage.SUGGESTION
        )
        assert risk == RiskLevel.MEDIUM
        assert need_approval is True

    def test_semi_autonomous_urgent_replenish_high_risk(self):
        from app.agent.service import AgentService
        risk, need_approval = AgentService._derive_action_risk(
            "replenish", {"urgency": "urgent"}, EvolutionStage.SEMI_AUTONOMOUS
        )
        assert risk == RiskLevel.HIGH
        assert need_approval is True

    def test_semi_autonomous_discount_review_high_risk(self):
        from app.agent.service import AgentService
        risk, need_approval = AgentService._derive_action_risk(
            "discount_review", {}, EvolutionStage.SEMI_AUTONOMOUS
        )
        assert risk == RiskLevel.HIGH
        assert need_approval is True

    def test_semi_autonomous_price_review_medium_risk(self):
        from app.agent.service import AgentService
        risk, need_approval = AgentService._derive_action_risk(
            "price_review", {}, EvolutionStage.SEMI_AUTONOMOUS
        )
        assert risk == RiskLevel.MEDIUM
        assert need_approval is True

    def test_semi_autonomous_ad_critical(self):
        from app.agent.service import AgentService
        risk, need_approval = AgentService._derive_action_risk(
            "ad_action", {"status": "critical"}, EvolutionStage.SEMI_AUTONOMOUS
        )
        assert risk == RiskLevel.CRITICAL
        assert need_approval is False

    def test_semi_autonomous_ad_warning(self):
        from app.agent.service import AgentService
        risk, need_approval = AgentService._derive_action_risk(
            "ad_action", {"status": "warning"}, EvolutionStage.SEMI_AUTONOMOUS
        )
        assert risk == RiskLevel.MEDIUM
        assert need_approval is False


@pytest.mark.usefixtures("prepare_db")
class TestAgentActionBridgeAPI:
    """通过 API 触发 Agent 决策后验证 ActionProposal 被桥接创建"""

    async def _set_agent_stage(self, agent_id: str, decision_point: str, stage: str = "suggestion"):
        """设置 Agent 自治等级，确保桥接代码执行"""
        from app.database import async_session_factory
        from app.agent.models import AgentEvolutionConfig
        from sqlalchemy import select

        async with async_session_factory() as db:
            stmt = select(AgentEvolutionConfig).where(
                AgentEvolutionConfig.user_id == 1,
                AgentEvolutionConfig.agent_id == agent_id,
                AgentEvolutionConfig.decision_point == decision_point,
            )
            config = (await db.execute(stmt)).scalar_one_or_none()
            if not config:
                config = AgentEvolutionConfig(
                    user_id=1, agent_id=agent_id,
                    decision_point=decision_point,
                    current_stage=stage,
                )
                db.add(config)
            else:
                config.current_stage = stage
            await db.commit()

    async def _count_proposals(self, db) -> int:
        from sqlalchemy import select, func
        from app.agentos.models import ActionProposal
        return (await db.execute(select(func.count()).select_from(ActionProposal))).scalar() or 0

    async def test_agent_decision_creates_action_proposal(self, async_client):
        """执行 A5 决策后验证至少有一个 ActionProposal 被创建"""
        await self._set_agent_stage("A5", "stock_alert")
        from app.database import async_session_factory

        before = 0
        async with async_session_factory() as db:
            before = await self._count_proposals(db)

        resp = await async_client.post(
            "/api/agents/A5/decide",
            json={
                "decision_point": "stock_alert",
                "context": {
                    "sku_code": "SKU-BRIDGE-001",
                    "sellable_stock": 5,
                    "locked_stock": 0,
                    "in_transit_stock": 0,
                    "sales_7d": 30,
                    "lead_time_days": 20,
                    "safety_stock_days": 14,
                },
            },
        )
        assert resp.status_code == 200, resp.text

        async with async_session_factory() as db:
            after = await self._count_proposals(db)
        assert after > before, "Agent 决策后应该创建了 ActionProposal"

    async def test_proposal_has_correct_source_type(self, async_client):
        """验证桥接创建的 ActionProposal source_type 正确"""
        await self._set_agent_stage("A6", "profit_watch")
        from app.database import async_session_factory
        from sqlalchemy import select
        from app.agentos.models import ActionProposal

        resp = await async_client.post(
            "/api/agents/A6/decide",
            json={
                "decision_point": "profit_watch",
                "context": {
                    "sku_code": "SKU-PROFIT-001",
                    "selling_price": 500,
                    "cost_price": 600,
                },
            },
        )
        assert resp.status_code == 200, resp.text

        async with async_session_factory() as db:
            stmt = select(ActionProposal).order_by(ActionProposal.id.desc()).limit(1)
            proposal = (await db.execute(stmt)).scalars().first()
            if proposal:
                assert proposal.source_type in ("agent_action", "agent_decision")
                assert proposal.agent_id in ("A6",)

    async def test_bridge_failure_does_not_block_decision(self, async_client):
        """桥接异常时主决策流程不应中断"""
        await self._set_agent_stage("A5", "stock_alert")
        with patch("app.agentos.action_center_service.ActionCenterService.create_proposal") as mock:
            mock.side_effect = RuntimeError("Bridge failed")
            resp = await async_client.post(
                "/api/agents/A5/decide",
                json={
                    "decision_point": "stock_alert",
                    "context": {
                        "sku_code": "SKU-FAIL-001",
                        "sellable_stock": 5,
                        "locked_stock": 0,
                        "sales_7d": 30,
                        "lead_time_days": 20,
                        "safety_stock_days": 14,
                    },
                },
            )
        assert resp.status_code == 200, "桥接失败应返回 200"
        data = resp.json().get("data", {})
        assert data.get("decision_id") is not None, "决策应正常创建"

    async def test_proposal_appears_in_work_items(self, async_client):
        """桥接创建的 ActionProposal 应在 WorkItems 中可见"""
        await self._set_agent_stage("A5", "stock_alert")

        # 触发 A5 决策
        await async_client.post(
            "/api/agents/A5/decide",
            json={
                "decision_point": "stock_alert",
                "context": {
                    "sku_code": "SKU-WI-001",
                    "sellable_stock": 3,
                    "locked_stock": 0,
                    "in_transit_stock": 0,
                    "sales_7d": 30,
                    "lead_time_days": 20,
                    "safety_stock_days": 14,
                },
            },
        )

        # 在 WorkItems 中检查
        resp = await async_client.get("/api/agentos/work-items?source_type=action_proposal")
        assert resp.status_code == 200
        records = resp.json().get("records", [])
        assert any(
            item.get("source_type") == "action_proposal" for item in records
        ), "WorkItems 应包含 action_proposal"
