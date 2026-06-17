"""G1 运营驾驶舱 Agent (Phase 3 MVP)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.5
- 汇总 A5/A6/G3 风险
- 展示 Agent 决策概览
- 规则健康概览

G1 的 dashboard 数据通过专用 API 端点获取，
这里只定义 Agent 元数据和决策点。
"""
from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent


@register_agent
class G1DashboardAgent(BaseAgent):
    agent_id = "G1"
    name = "运营驾驶舱 Agent"
    description = "汇总库存、利润、折扣风险，展示决策概览和规则健康状态"
    decision_points = ["dashboard_overview"]
    version = "1.0.0"

    DEFAULT_STAGES = {
        "dashboard_overview": EvolutionStage.OBSERVATION,
    }

    async def decide(self, decision_point: str, context: dict[str, Any]) -> dict[str, Any]:
        # G1 dashboard data is provided via dedicated API endpoint
        # This method is a fallback for future extensibility
        if decision_point == "dashboard_overview":
            return {
                "message": "请使用 /api/agents/dashboard 端点获取驾驶舱数据",
                "confidence": 0.0,
            }
        return {"action": "unknown", "confidence": 0.0}
