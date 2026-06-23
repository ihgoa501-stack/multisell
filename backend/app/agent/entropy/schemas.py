"""熵管理 Pydantic Schema"""

from datetime import datetime
from typing import Optional, Any
from pydantic import BaseModel, ConfigDict


class RuleMarkChangeVO(BaseModel):
    id: int
    target_type: str
    target_id: int
    field_path: str
    old_value: Optional[Any] = None
    new_value: Any
    source_type: str
    source_id: Optional[str] = None
    change_summary: str
    parent_change_id: Optional[int] = None
    related_decision_ids: Optional[list] = None
    context_json: Optional[Any] = None
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class RuleHealthVO(BaseModel):
    rule_id: int
    rule_name: str
    rule_type: str
    agent_id: str
    decision_point: str
    score: float
    status: str
    times_applied: int
    times_overridden: int
    override_rate: float
    days_since_last_applied: Optional[int] = None
    confidence: float
    risk_level: str


class EntropySummaryVO(BaseModel):
    total_rules: int
    active_rules: int
    shadow_rules: int
    retired_rules: int
    avg_health_score: float
    unhealthy_rule_count: int
    pending_merge_count: int
    recent_changes_count: int
    system_entropy_index: float


class SpcStatusVO(BaseModel):
    agent_id: str
    decision_point: str
    metric_name: str
    current_value: float
    baseline_mean: float
    ucl: float
    lcl: float
    uwl: float
    lwl: float
    consecutive_same_side: int
    is_out_of_control: bool
    is_warning: bool
    last_breach_at: Optional[datetime] = None
    next_recalc_at: Optional[datetime] = None


class DefenseActionVO(BaseModel):
    action: str
    description: str
    affected_rules: list[dict]
    mark_changes: list[dict]


class DefenseSummaryVO(BaseModel):
    expired_count: int
    budget_exceeded_count: int
    decay_applied_count: int
    merged_count: int
    regret_actions: int
    total_affected: int
