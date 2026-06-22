"""AgentOS 持久化模型"""
from sqlalchemy import (
    BigInteger,
    Boolean,
    CheckConstraint,
    Column,
    DateTime,
    ForeignKey,
    Integer,
    Numeric,
    String,
    Text,
)
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy import JSON
from sqlalchemy import func as sa_func
from sqlalchemy.orm import relationship

from app.database import Base


JSONType = JSON().with_variant(JSONB, "postgresql")


ACTION_PROPOSAL_STATUS_FLOW = {
    "suggested": {"pending_approval", "approved", "executing", "rejected", "expired", "blocked_by_policy"},
    "pending_approval": {"approved", "rejected", "expired", "blocked_by_policy"},
    "approved": {"executing", "cancelled", "blocked_by_policy"},
    "executing": {"executed", "failed"},
    "executed": {"reviewed"},
    "reviewed": set(),
    "rejected": set(),
    "expired": set(),
    "blocked_by_policy": set(),
    "failed": {"executing", "cancelled"},
    "cancelled": set(),
}


class AgentOSOperationLog(Base):
    """AgentOS 操作审计日志"""
    __tablename__ = "agentos_operation_log"

    id = Column(Integer, primary_key=True, autoincrement=True)
    user_id = Column(Integer, nullable=False, index=True)
    item_id = Column(String(128), nullable=False, comment="WorkItem ID (e.g. exception:42)")
    action = Column(String(32), nullable=False, comment="approve / reject / status_update")
    source_type = Column(String(32), nullable=True, comment="agent_action / exception / notification / listing_task")
    previous_status = Column(String(32), nullable=True)
    new_status = Column(String(32), nullable=True)
    comment = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)


class ActionProposal(Base):
    """AgentOS 统一动作提案"""
    __tablename__ = "agentos_action_proposal"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    source_type = Column(String(50), nullable=False, index=True)
    source_id = Column(String(100), nullable=True, index=True)
    agent_id = Column(String(20), nullable=True, index=True)
    squad_id = Column(String(50), nullable=True, index=True)
    action_type = Column(String(100), nullable=False, index=True)
    business_object_type = Column(String(50), nullable=True, index=True)
    business_object_id = Column(String(100), nullable=True, index=True)
    title = Column(String(300), nullable=False)
    description = Column(Text, nullable=True)
    proposed_payload = Column(JSONType, nullable=False, default=dict)
    before_snapshot = Column(JSONType, nullable=True)
    after_snapshot = Column(JSONType, nullable=True)
    risk_level = Column(String(20), nullable=False, default="medium", index=True)
    requires_approval = Column(Boolean, nullable=False, default=True)
    status = Column(String(30), nullable=False, default="suggested", index=True)
    confidence = Column(Numeric(5, 4), nullable=True)
    proposed_by = Column(String(100), nullable=True)
    approved_by = Column(String(100), nullable=True)
    approved_at = Column(DateTime(timezone=True), nullable=True)
    rejected_by = Column(String(100), nullable=True)
    rejected_at = Column(DateTime(timezone=True), nullable=True)
    rejection_reason = Column(Text, nullable=True)

    store_id = Column(Integer, nullable=True, comment="店铺ID（null=全店铺通用）")
    approval_deadline = Column(DateTime(timezone=True), nullable=True, comment="审批截止时间")
    escalation_level = Column(Integer, default=0, comment="当前升级层级 (0=初始)")
    auto_decision = Column(String(20), default="reject", comment="超时后动作: reject/auto_execute")

    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=sa_func.now(), onupdate=sa_func.now(), nullable=False)

    approvals = relationship("ApprovalRequest", back_populates="proposal", lazy="selectin")
    executions = relationship("CommandExecution", back_populates="proposal", lazy="selectin")
    reviews = relationship("OutcomeReview", back_populates="proposal", lazy="selectin")

    __table_args__ = (
        CheckConstraint(
            "risk_level in ('low', 'medium', 'high', 'critical')",
            name="ck_agentos_action_proposal_risk",
        ),
        CheckConstraint(
            "status in ('suggested', 'pending_approval', 'approved', 'executing', 'executed', 'reviewed', 'rejected', 'expired', 'blocked_by_policy', 'failed', 'cancelled')",
            name="ck_agentos_action_proposal_status",
        ),
    )


class ApprovalRequest(Base):
    """ActionProposal 审批记录"""
    __tablename__ = "agentos_approval_request"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    proposal_id = Column(BigInteger, ForeignKey("agentos_action_proposal.id"), nullable=False, index=True)
    requester = Column(String(100), nullable=True)
    approver = Column(String(100), nullable=True)
    decision = Column(String(30), nullable=False, default="pending")
    comment = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)
    decided_at = Column(DateTime(timezone=True), nullable=True)

    proposal = relationship("ActionProposal", back_populates="approvals", lazy="selectin")

    __table_args__ = (
        CheckConstraint(
            "decision in ('pending', 'approved', 'rejected')",
            name="ck_agentos_approval_request_decision",
        ),
    )


class CommandExecution(Base):
    """业务命令执行记录"""
    __tablename__ = "agentos_command_execution"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    proposal_id = Column(BigInteger, ForeignKey("agentos_action_proposal.id"), nullable=False, index=True)
    command_name = Column(String(100), nullable=False)
    executor = Column(String(100), nullable=True)
    status = Column(String(30), nullable=False, default="started")
    input_payload = Column(JSONType, nullable=False, default=dict)
    result_payload = Column(JSONType, nullable=True)
    error_message = Column(Text, nullable=True)

    compensation = Column(JSON, nullable=True, comment="补偿操作定义")
    compensated_by = Column(BigInteger, ForeignKey("agentos_command_execution.id"), nullable=True, comment="补偿记录ID（自引用）")

    started_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)
    finished_at = Column(DateTime(timezone=True), nullable=True)

    proposal = relationship("ActionProposal", back_populates="executions", lazy="selectin")

    __table_args__ = (
        CheckConstraint(
            "status in ('started', 'succeeded', 'failed')",
            name="ck_agentos_command_execution_status",
        ),
    )


class OutcomeReview(Base):
    """动作执行后的经营结果复盘"""
    __tablename__ = "agentos_outcome_review"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    proposal_id = Column(BigInteger, ForeignKey("agentos_action_proposal.id"), nullable=False, index=True)
    outcome = Column(String(30), nullable=False)
    business_metric = Column(String(100), nullable=True)
    metric_delta = Column(Numeric(14, 4), nullable=True)
    notes = Column(Text, nullable=True)
    reviewed_by = Column(String(100), nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)

    proposal = relationship("ActionProposal", back_populates="reviews", lazy="selectin")

    __table_args__ = (
        CheckConstraint(
            "outcome in ('positive', 'neutral', 'negative')",
            name="ck_agentos_outcome_review_outcome",
        ),
    )
