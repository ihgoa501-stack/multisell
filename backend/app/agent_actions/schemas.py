"""Agent 动作提案与审批 - Pydantic Schema"""

from typing import Optional

from pydantic import BaseModel, Field


class AgentActionCreate(BaseModel):
    source_module: Optional[str] = None
    source_type: Optional[str] = None
    source_id: Optional[int] = None
    exception_id: Optional[int] = None
    action_type: str = Field(..., max_length=100)
    title: str = Field(..., min_length=1, max_length=300)
    description: Optional[str] = None
    proposed_payload: Optional[dict] = None
    before_snapshot: Optional[dict] = None


class AgentActionAfterSnapshot(BaseModel):
    after_snapshot: Optional[dict] = None


class AgentActionReject(BaseModel):
    rejection_reason: Optional[str] = None
