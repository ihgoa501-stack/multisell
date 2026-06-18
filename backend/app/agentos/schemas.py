from datetime import datetime
from typing import Any

from pydantic import BaseModel, Field


class BusinessObjectVO(BaseModel):
    type: str | None = None
    id: str | None = None
    label: str | None = None


class WorkItemVO(BaseModel):
    id: str
    source_type: str
    source_id: str
    source_module: str | None = None
    business_object: BusinessObjectVO = Field(default_factory=BusinessObjectVO)
    squad: str
    agent_id: str | None = None
    title: str
    summary: str | None = None
    recommendation: str | None = None
    risk_level: str
    approval_required: bool = False
    status: str
    action_type: str | None = None
    context: dict[str, Any] = Field(default_factory=dict)
    audit_link: str | None = None
    created_at: datetime | None = None


class ControlCenterSummaryVO(BaseModel):
    sales_today: float = 0
    profit_today: float = 0
    inventory_risks: int = 0
    pending_approvals: int = 0
    active_work_items: int = 0
    agent_automation_rate: float = 0


class SquadVO(BaseModel):
    id: str
    name: str
    description: str
    agents: list[str]
    decision_count_7d: int = 0
    pending_approvals: int = 0
    risk_count: int = 0
    adoption_rate: float = 0
    autonomy_level: str = "suggestion"


class TemplateCardVO(BaseModel):
    id: str
    title: str
    squad: str
    description: str
    mode: str
    route: str
    phase: str = "phase_1"
