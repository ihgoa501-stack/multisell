"""Agent 协作流水线测试"""
import pytest


class TestChainConditions:
    """链触发条件"""

    @pytest.fixture
    def conditions(self):
        from app.agent.pipeline import (
            _chain_stock_red_to_a5,
            _chain_discount_block,
            _chain_a6_loss,
        )
        return {
            "stock_red": _chain_stock_red_to_a5,
            "discount_block": _chain_discount_block,
            "a6_loss": _chain_a6_loss,
        }

    def test_stock_red_triggers(self, conditions):
        """A5 红色预警 → 触发链"""
        result = {
            "agent_id": "A5",
            "decision_point": "stock_alert",
            "agent_output": {"stock_status": "red", "sellable_days": 2},
        }
        assert conditions["stock_red"](result) is True

    def test_stock_yellow_does_not_trigger(self, conditions):
        """A5 黄色预警 → 不触发"""
        result = {
            "agent_output": {"stock_status": "yellow"},
            "final_decision": {"stock_status": "yellow"},
        }
        assert conditions["stock_red"](result) is False

    def test_stock_green_does_not_trigger(self, conditions):
        """A5 正常 → 不触发"""
        result = {"agent_output": {"stock_status": "green"}}
        assert conditions["stock_red"](result) is False

    def test_discount_block_triggers(self, conditions):
        """G3 阻断 → 触发链"""
        result = {
            "final_decision": {"action": "block", "reason": "折扣过深"},
        }
        assert conditions["discount_block"](result) is True

    def test_discount_warn_does_not_trigger(self, conditions):
        """G3 预警 → 不触发"""
        result = {"final_decision": {"action": "warn"}}
        assert conditions["discount_block"](result) is False

    def test_a6_loss_triggers(self, conditions):
        """A6 亏损 → 触发链"""
        result = {"final_decision": {"is_loss": True, "margin_pct": -5}}
        assert conditions["a6_loss"](result) is True

    def test_a6_below_threshold_triggers(self, conditions):
        """A6 低于阈值 → 触发链"""
        result = {"final_decision": {"below_threshold": True}}
        assert conditions["a6_loss"](result) is True

    def test_a6_profitable_does_not_trigger(self, conditions):
        """A6 正常 → 不触发"""
        result = {"final_decision": {"is_loss": False, "below_threshold": False}}
        assert conditions["a6_loss"](result) is False


class TestChainContextMappers:
    """链上下文映射器"""

    def test_stock_context_has_required_fields(self):
        from app.agent.pipeline import _chain_stock_ctx
        result = {
            "final_decision": {
                "sku_code": "SKU001",
                "sellable_stock": 10,
                "sellable_days": 3,
                "suggested_replenish_qty": 200,
            },
            "decision_id": 42,
        }
        ctx = _chain_stock_ctx(result)
        assert ctx["sku_code"] == "SKU001"
        assert ctx["sellable_days"] == 3
        assert ctx["source_decision_id"] == 42

    def test_discount_context_has_required_fields(self):
        from app.agent.pipeline import _chain_discount_ctx
        result = {
            "final_decision": {
                "sku_code": "SKU001",
                "original_price": 100,
                "proposed_discount": 0.3,
                "reason": "利润率不足",
            },
            "decision_id": 43,
        }
        ctx = _chain_discount_ctx(result)
        assert ctx["sku_code"] == "SKU001"
        assert ctx["block_reason"] == "利润率不足"

    def test_a6_loss_context_has_required_fields(self):
        from app.agent.pipeline import _chain_a6_loss_ctx
        result = {
            "final_decision": {
                "sku_code": "SKU001",
                "margin_pct": -3.5,
                "anomaly_reason": "成本上涨",
            },
        }
        ctx = _chain_a6_loss_ctx(result)
        assert ctx["sku_code"] == "SKU001"
        assert ctx["anomaly_reason"] == "成本上涨"


class TestChainDefinitions:
    """链规则定义完整性"""

    def test_all_chains_defined(self):
        from app.agent.pipeline import CHAINS
        assert "A5" in CHAINS
        assert "G3" in CHAINS
        assert "A6" in CHAINS

    def test_chains_have_valid_targets(self):
        """链目标 Agent 必须注册"""
        from app.agent.pipeline import CHAINS
        from app.agent.registry import AgentRegistry
        for src, dps in CHAINS.items():
            for dp, rules in dps.items():
                for rule in rules:
                    assert AgentRegistry.get_agent_class(rule.target_agent) is not None, \
                        f"链 {src}[{dp}] → {rule.target_agent} 目标未注册"

    def test_all_chains_have_description(self):
        from app.agent.pipeline import CHAINS
        for src, dps in CHAINS.items():
            for dp, rules in dps.items():
                for rule in rules:
                    assert rule.description, f"{src}[{dp}] → {rule.target_agent} 缺少描述"


class TestEvaluateChains:
    """链执行"""

    async def test_no_chains_for_unknown_agent(self, async_client):
        """无链规则的 Agent → 空结果"""
        from app.agent.pipeline import evaluate_chains
        from app.database import async_session_factory
        async with async_session_factory() as db:
            result = await evaluate_chains(
                "A1", "product_scout", {"decision_id": 1}, 1, db,
            )
        assert result == []

    async def test_no_chains_when_condition_not_met(self, async_client):
        """条件不满足时 → 空结果"""
        from app.agent.pipeline import evaluate_chains
        from app.database import async_session_factory
        result_data = {
            "agent_id": "A5",
            "decision_point": "stock_alert",
            "decision_id": 1,
            "agent_output": {"stock_status": "green"},
            "final_decision": {"stock_status": "green"},
        }
        async with async_session_factory() as db:
            result = await evaluate_chains(
                "A5", "stock_alert", result_data, 1, db,
            )
        assert result == []
