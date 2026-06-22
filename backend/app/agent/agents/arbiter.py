"""G0 Arbiter Agent — conflict arbitration (non-LLM, uses Policy Matrix then heuristic)"""
from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent


@register_agent
class G0ArbiterAgent(BaseAgent):
    """Agent间冲突仲裁 — 裁决多个Agent之间的矛盾决策

    G0 doesn't call LLM. It uses the Policy Matrix first,
    then falls back to confidence-based heuristic,
    then escalates to human.
    """

    agent_id = "G0"
    name = "Arbiter"
    description = "冲突仲裁 — 裁决多个Agent之间的矛盾决策"
    decision_points = ["arbitrate"]
    version = "1.0.0"

    DEFAULT_STAGES = {
        "arbitrate": EvolutionStage.SUGGESTION,
    }

    async def decide(
        self, decision_point: str, context: dict[str, Any], db: Any = None
    ) -> dict[str, Any]:
        """Decide arbitration outcome.

        Context expects:
            agent_id_a, agent_id_b, decision_point,
            confidence_a (float), confidence_b (float)

        Returns:
            {"verdict": str, "reason": str, "confidence": float, "method": str}
        """
        from app.agentos.policy_service import PolicyService

        agent_id_a = context.get("agent_id_a", "")
        agent_id_b = context.get("agent_id_b", "")
        dp = context.get("decision_point", "")

        # Step 1: Try Policy Matrix
        if db is not None:
            matrix_result = await PolicyService.resolve_conflict(
                db, agent_id_a, agent_id_b, dp, context
            )
            if matrix_result:
                return {
                    "verdict": matrix_result["winner"],
                    "reason": matrix_result["reason"],
                    "confidence": 0.95,
                    "method": "policy_matrix",
                }

        # Step 2: Fall back to confidence-based heuristic
        conf_a = context.get("confidence_a", 0.0)
        conf_b = context.get("confidence_b", 0.0)

        if abs(conf_a - conf_b) > 0.15:
            winner = agent_id_a if conf_a > conf_b else agent_id_b
            return {
                "verdict": "adopt_" + winner,
                "reason": (
                    f"基于置信度裁决: {agent_id_a}({conf_a}) vs "
                    f"{agent_id_b}({conf_b})"
                ),
                "confidence": max(conf_a, conf_b),
                "method": "confidence_heuristic",
            }

        # Step 3: Escalate to human
        return {
            "verdict": "escalate",
            "reason": (
                f"Policy Matrix 未覆盖且置信度接近 "
                f"({conf_a} vs {conf_b})"
            ),
            "confidence": 0.5,
            "method": "escalate",
        }
