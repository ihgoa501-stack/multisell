"""Agent 基类与生命周期"""

from enum import Enum
from typing import Any, Optional
from uuid import uuid4


class EvolutionStage(str, Enum):
    OBSERVATION = "observation"
    SUGGESTION = "suggestion"
    SEMI_AUTONOMOUS = "semi_autonomous"
    FULL_AUTONOMOUS = "full_autonomous"


class AgentMetadata:
    agent_id: str
    name: str
    description: str
    decision_points: list[str]
    version: str = "1.0.0"


class BaseAgent:
    agent_id: str = ""
    name: str = ""
    description: str = ""
    decision_points: list[str] = []
    version: str = "1.0.0"

    def __init__(
        self, user_id: int, stage_override: Optional[dict[str, EvolutionStage]] = None
    ):
        self.user_id = user_id
        self.session_id = str(uuid4())
        self._stage_overrides = stage_override or {}

    def get_stage(self, decision_point: str) -> EvolutionStage:
        return self._stage_overrides.get(decision_point, EvolutionStage.OBSERVATION)

    def get_confidence_threshold(self, decision_point: str) -> float:
        stage = self.get_stage(decision_point)
        thresholds = {
            EvolutionStage.OBSERVATION: 0.0,
            EvolutionStage.SUGGESTION: 0.85,
            EvolutionStage.SEMI_AUTONOMOUS: 0.90,
            EvolutionStage.FULL_AUTONOMOUS: 0.95,
        }
        return thresholds[stage]

    async def decide(
        self, decision_point: str, context: dict[str, Any], db: Any = None
    ) -> dict[str, Any]:
        raise NotImplementedError

    def build_decision_record(
        self,
        decision_point: str,
        context: dict[str, Any],
        agent_output: dict[str, Any],
        final_decision: dict[str, Any],
        user_action: str = "ignored",
        user_overrides: Optional[dict] = None,
        user_feedback: Optional[str] = None,
        rules_applied: Optional[list] = None,
        rule_overrides: int = 0,
        confidence: Optional[float] = None,
        response_time_ms: Optional[int] = None,
        token_count: Optional[int] = None,
        episode_id: Optional[int] = None,
    ) -> dict[str, Any]:
        return {
            "user_id": self.user_id,
            "agent_id": self.agent_id,
            "decision_point": decision_point,
            "context_json": context,
            "agent_output": agent_output,
            "final_decision": final_decision,
            "user_action": user_action,
            "user_overrides": user_overrides,
            "user_feedback": user_feedback,
            "rules_applied": rules_applied,
            "rule_overrides": rule_overrides,
            "evolution_stage": self.get_stage(decision_point).value,
            "confidence": confidence,
            "response_time_ms": response_time_ms,
            "token_count": token_count,
            "session_id": self.session_id,
            "episode_id": episode_id,
        }
