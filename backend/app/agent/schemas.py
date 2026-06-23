"""Agent Pydantic Schema"""

from datetime import datetime
from typing import Optional, Any
from pydantic import BaseModel, ConfigDict, Field


class AgentMetadataVO(BaseModel):
    agent_id: str
    name: str
    description: str
    decision_points: list[str]
    version: str


class DecisionLogVO(BaseModel):
    id: int
    user_id: int
    agent_id: str
    decision_point: str
    context_json: Any
    agent_output: Any
    final_decision: Any
    user_action: str
    user_overrides: Optional[Any] = None
    user_feedback: Optional[str] = None
    rules_applied: Optional[list] = None
    evolution_stage: str
    confidence: Optional[float] = None
    response_time_ms: Optional[int] = None
    token_count: Optional[int] = None
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class PersonalRuleVO(BaseModel):
    id: int
    user_id: int
    agent_id: str
    decision_point: str
    rule_type: str
    rule_name: str
    rule_condition: Any
    rule_action: Any
    priority: int
    source: str
    status: str
    confidence: float = 0
    times_applied: int = 0
    times_overridden: int = 0
    last_applied_at: Optional[datetime] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class PersonalRuleCreate(BaseModel):
    agent_id: str
    decision_point: str
    rule_type: str = Field(..., pattern="^(threshold|strategy|style|veto)$")
    rule_name: str
    rule_condition: dict
    rule_action: dict
    priority: int = 100
    source: str = "manual"


class PersonalRuleUpdate(BaseModel):
    rule_name: Optional[str] = None
    rule_condition: Optional[dict] = None
    rule_action: Optional[dict] = None
    priority: Optional[int] = None
    status: Optional[str] = Field(None, pattern="^(active|shadow|paused|retired)$")


class HonchoProfileVO(BaseModel):
    id: int
    user_id: int
    risk_tolerance: str
    communication_style: str
    notification_prefs: Optional[Any] = None
    agent_profiles: Any
    hypothesis_count: int = 0
    confirmed_count: int = 0
    last_dialectic_at: Optional[datetime] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class HonchoProfileUpdate(BaseModel):
    risk_tolerance: Optional[str] = Field(
        None, pattern="^(conservative|moderate|aggressive)$"
    )
    communication_style: Optional[str] = Field(
        None, pattern="^(concise|balanced|detailed)$"
    )
    notification_prefs: Optional[dict] = None
    agent_profiles: Optional[dict] = None


class AgentDecisionRequest(BaseModel):
    decision_point: str
    context: dict
    dry_run: bool = False


class FeedbackRequest(BaseModel):
    user_action: str = Field(..., pattern="^(accepted|modified|rejected|ignored)$")
    user_overrides: Optional[dict] = None
    user_feedback: Optional[str] = None


class AgentDecisionResponse(BaseModel):
    agent_id: str
    decision_point: str
    decision: dict
    stage: str
    confidence: float
    rules_applied: list[int] = []
    decision_id: Optional[int] = None


class EpisodeVO(BaseModel):
    id: int
    user_id: int
    agent_id: str
    episode_number: int
    decision_count: int
    episode_summary: Optional[str] = None
    key_insights: Optional[Any] = None
    acceptance_rate: Optional[float] = None
    avg_confidence: Optional[float] = None
    nudge_triggered: int = 0
    started_at: datetime
    ended_at: datetime
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


# ── 进化/等级控制相关 Pydantic 模型 ──────────────────────

STAGE_PATTERN = r"^(observation|suggestion|semi_autonomous|full_autonomous)$"


class StageChangeRequest(BaseModel):
    decision_point: str
    target_stage: str = Field(..., pattern=STAGE_PATTERN)


class NudgeRespondRequest(BaseModel):
    response: str = Field(..., pattern=r"^(accept|dismiss)$")
