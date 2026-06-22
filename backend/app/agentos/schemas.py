"""AgentOS 数据模型 — 聚合层对前端暴露的数据契约"""

from datetime import datetime
from enum import Enum
from typing import Any, Literal, Optional

from pydantic import BaseModel, Field


class AutonomyLevel(str, Enum):
    """自治等级（沿用现有 Hermes Agent 阶段）"""
    OBSERVATION = "OBSERVATION"
    SUGGESTION = "SUGGESTION"
    SEMI_AUTONOMOUS = "SEMI_AUTONOMOUS"
    FULL_AUTONOMOUS = "FULL_AUTONOMOUS"

    @property
    def label(self) -> str:
        return _AUTONOMY_LABELS[self]

    @property
    def level(self) -> int:
        return _AUTONOMY_LEVELS[self]


_AUTONOMY_LABELS = {
    AutonomyLevel.OBSERVATION: "观察",
    AutonomyLevel.SUGGESTION: "建议",
    AutonomyLevel.SEMI_AUTONOMOUS: "半自主",
    AutonomyLevel.FULL_AUTONOMOUS: "全自主",
}

_AUTONOMY_LEVELS = {
    AutonomyLevel.OBSERVATION: 0,
    AutonomyLevel.SUGGESTION: 1,
    AutonomyLevel.SEMI_AUTONOMOUS: 2,
    AutonomyLevel.FULL_AUTONOMOUS: 3,
}


class WorkItemPriority(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"

    @property
    def label(self) -> str:
        return _PRIORITY_LABELS[self]


_PRIORITY_LABELS = {
    WorkItemPriority.LOW: "低",
    WorkItemPriority.MEDIUM: "中",
    WorkItemPriority.HIGH: "高",
    WorkItemPriority.CRITICAL: "紧急",
}


class WorkItemStatus(str, Enum):
    PENDING = "pending"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    FAILED = "failed"
    BLOCKED = "blocked"
    CANCELLED = "cancelled"

    @property
    def label(self) -> str:
        return _STATUS_LABELS[self]


_STATUS_LABELS = {
    WorkItemStatus.PENDING: "待处理",
    WorkItemStatus.IN_PROGRESS: "处理中",
    WorkItemStatus.COMPLETED: "已完成",
    WorkItemStatus.FAILED: "失败",
    WorkItemStatus.BLOCKED: "已阻塞",
    WorkItemStatus.CANCELLED: "已取消",
}


class RiskLevel(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"

    @property
    def label(self) -> str:
        return _RISK_LABELS[self]


_RISK_LABELS = {
    RiskLevel.LOW: "低风险",
    RiskLevel.MEDIUM: "中风险",
    RiskLevel.HIGH: "高风险",
    RiskLevel.CRITICAL: "严重",
}


# ─── 核心数据模型 ─────────────────────────────────────────────


class AgentOSAgent(BaseModel):
    """Agent 团队成员"""
    id: str
    name: str
    role: str
    squad_id: str
    status: str = "idle"
    autonomy_level: AutonomyLevel = AutonomyLevel.SUGGESTION
    current_workload: int = 0
    success_rate: float = 0.0
    last_activity_at: Optional[datetime] = None
    risk_level: RiskLevel = RiskLevel.LOW


class AgentOSSquad(BaseModel):
    """Agent 小队"""
    id: str
    name: str
    description: str = ""
    domain: str = ""
    status: str = "active"
    autonomy_level: AutonomyLevel = AutonomyLevel.SUGGESTION
    agents: list[AgentOSAgent] = Field(default_factory=list)
    active_work_items: int = 0
    pending_approvals: int = 0
    risk_level: RiskLevel = RiskLevel.LOW
    health_score: float = 0.0


class AgentOSWorkItem(BaseModel):
    """统一任务模型"""
    id: str
    source_type: str
    source_id: str
    title: str
    description: Optional[str] = None
    priority: WorkItemPriority = WorkItemPriority.MEDIUM
    status: WorkItemStatus = WorkItemStatus.PENDING
    risk_level: RiskLevel = RiskLevel.LOW
    agent_id: Optional[str] = None
    agent_name: Optional[str] = None
    squad_id: Optional[str] = None
    squad_name: Optional[str] = None
    autonomy_level: AutonomyLevel = AutonomyLevel.SUGGESTION
    requires_approval: bool = False
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    due_at: Optional[datetime] = None
    action_url: Optional[str] = None
    metadata: dict[str, Any] = Field(default_factory=dict)


class AgentOSOverview(BaseModel):
    """全局概览"""
    health_score: float = 0.0
    active_agents: int = 0
    pending_approvals: int = 0
    critical_items: int = 0


class AgentOSMetric(BaseModel):
    """业务指标"""
    key: str
    label: str
    value: float = 0.0
    trend: Optional[str] = None  # up / down / stable
    unit: str = ""


class AgentOSTemplate(BaseModel):
    """内置模板"""
    id: str
    title: str
    description: str = ""
    squad: str = ""
    mode: str = "Agent"
    route: str = ""
    phase: str = "phase_1"


# ─── 响应模型 ─────────────────────────────────────────────


class ControlCenterResponse(BaseModel):
    """总控台响应"""
    overview: AgentOSOverview = Field(default_factory=AgentOSOverview)
    squads: list[AgentOSSquad] = Field(default_factory=list)
    priority_work_items: list[AgentOSWorkItem] = Field(default_factory=list)
    metrics: list[AgentOSMetric] = Field(default_factory=list)
    recent_activity: list[AgentOSWorkItem] = Field(default_factory=list)


class WorkItemsResponse(BaseModel):
    """任务中心响应"""
    items: list[AgentOSWorkItem] = Field(default_factory=list)
    total: int = 0
    limit: int = 20
    offset: int = 0


class SquadsResponse(BaseModel):
    """团队页响应"""
    squads: list[AgentOSSquad] = Field(default_factory=list)
    summary: Optional[AgentOSOverview] = None


class TemplatesResponse(BaseModel):
    """模板响应"""
    templates: list[AgentOSTemplate] = Field(default_factory=list)


# ─── 请求模型（Phase 2 写入）─────────────────────────────────────


class WorkItemStatusUpdate(BaseModel):
    """更新 WorkItem 状态请求"""
    status: WorkItemStatus
    comment: Optional[str] = None


class WorkItemApproval(BaseModel):
    """审批 WorkItem 请求"""
    action: Literal["approve", "reject"] = "approve"
    comment: Optional[str] = None


# ─── Phase 3 模型 ─────────────────────────────────────────


class AgentOSOperationLogVO(BaseModel):
    """操作审计日志"""
    id: int
    user_id: int
    item_id: str
    action: str
    source_type: Optional[str] = None
    previous_status: Optional[str] = None
    new_status: Optional[str] = None
    comment: Optional[str] = None
    created_at: Optional[datetime] = None


class OperationLogQuery(BaseModel):
    item_id: Optional[str] = None
    action: Optional[str] = None
    source_type: Optional[str] = None
    limit: int = 20
    offset: int = 0


class AutonomyCandidateVO(BaseModel):
    """自治等级升级候选"""
    agent_id: str
    agent_name: str
    squad_id: str
    current_level: str
    suggested: bool
    direction: Optional[str] = None  # upgrade / downgrade / None
    target_level: Optional[str] = None
    confidence: float = 0
    reason: str = ""


class AgentDetailResponse(BaseModel):
    """Agent 详情响应"""
    agent: AgentOSAgent
    squad_name: str = ""
    current_work_items: list[AgentOSWorkItem] = Field(default_factory=list)
    recent_operations: list[AgentOSOperationLogVO] = Field(default_factory=list)
    decision_count_7d: int = 0
    adoption_rate_7d: float = 0.0


# ─── Phase 2: Action Center Schemas ────────────────────────────


class ActionProposalStatus(str, Enum):
    SUGGESTED = "suggested"
    PENDING_APPROVAL = "pending_approval"
    APPROVED = "approved"
    EXECUTING = "executing"
    EXECUTED = "executed"
    REVIEWED = "reviewed"
    REJECTED = "rejected"
    EXPIRED = "expired"
    BLOCKED_BY_POLICY = "blocked_by_policy"
    FAILED = "failed"
    CANCELLED = "cancelled"


class ActionProposalCreate(BaseModel):
    source_type: str = Field(min_length=1, max_length=50)
    source_id: Optional[str] = Field(default=None, max_length=100)
    agent_id: Optional[str] = Field(default=None, max_length=20)
    squad_id: Optional[str] = Field(default=None, max_length=50)
    action_type: str = Field(min_length=1, max_length=100)
    business_object_type: Optional[str] = Field(default=None, max_length=50)
    business_object_id: Optional[str] = Field(default=None, max_length=100)
    title: str = Field(min_length=1, max_length=300)
    description: Optional[str] = None
    proposed_payload: dict[str, Any] = Field(default_factory=dict)
    before_snapshot: Optional[dict[str, Any]] = None
    risk_level: RiskLevel = RiskLevel.MEDIUM
    requires_approval: bool = True
    confidence: Optional[float] = Field(default=None, ge=0.0, le=1.0)
    store_id: Optional[int] = Field(default=None, description="店铺ID")
    approval_deadline: Optional[datetime] = Field(default=None, description="审批截止时间")
    escalation_level: int = Field(default=0, description="审批升级层级")
    auto_decision: str = Field(default="reject", description="超时后动作: reject/auto_execute")


class ActionProposalVO(BaseModel):
    id: int
    source_type: str
    source_id: Optional[str] = None
    agent_id: Optional[str] = None
    squad_id: Optional[str] = None
    action_type: str
    business_object_type: Optional[str] = None
    business_object_id: Optional[str] = None
    title: str
    description: Optional[str] = None
    proposed_payload: dict[str, Any] = Field(default_factory=dict)
    before_snapshot: Optional[dict[str, Any]] = None
    after_snapshot: Optional[dict[str, Any]] = None
    risk_level: RiskLevel
    requires_approval: bool
    status: ActionProposalStatus
    confidence: Optional[float] = None
    proposed_by: Optional[str] = None
    approved_by: Optional[str] = None
    rejected_by: Optional[str] = None
    rejection_reason: Optional[str] = None
    store_id: Optional[int] = None
    approval_deadline: Optional[datetime] = None
    escalation_level: int = 0
    auto_decision: str = "reject"
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


class ActionApprovalPayload(BaseModel):
    comment: Optional[str] = None


class ActionExecutionPayload(BaseModel):
    executor: Optional[str] = None


class ActionReviewPayload(BaseModel):
    outcome: Literal["positive", "neutral", "negative"]
    business_metric: Optional[str] = Field(default=None, max_length=100)
    metric_delta: Optional[float] = None
    notes: Optional[str] = None


class ApprovalRequestVO(BaseModel):
    id: int
    proposal_id: int
    requester: Optional[str] = None
    approver: Optional[str] = None
    decision: str
    comment: Optional[str] = None
    created_at: Optional[datetime] = None
    decided_at: Optional[datetime] = None


class CommandExecutionVO(BaseModel):
    id: int
    proposal_id: int
    command_name: str
    executor: Optional[str] = None
    status: str
    input_payload: dict[str, Any] = Field(default_factory=dict)
    result_payload: Optional[dict[str, Any]] = None
    error_message: Optional[str] = None
    compensation: Optional[dict[str, Any]] = None
    compensated_by: Optional[int] = None
    started_at: Optional[datetime] = None
    finished_at: Optional[datetime] = None


class OutcomeReviewVO(BaseModel):
    id: int
    proposal_id: int
    outcome: str
    business_metric: Optional[str] = None
    metric_delta: Optional[float] = None
    notes: Optional[str] = None
    reviewed_by: Optional[str] = None
    created_at: Optional[datetime] = None
