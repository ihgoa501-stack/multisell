"""AgentOS action center service."""

from __future__ import annotations

from datetime import datetime, timezone
from decimal import Decimal
from typing import Any

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.agentos.models import (
    ACTION_PROPOSAL_STATUS_FLOW,
    ActionProposal,
    AgentOSOperationLog,
    ApprovalRequest,
    CommandExecution,
    OutcomeReview,
)
from app.agentos.schemas import (
    ActionProposalCreate,
    AgentOSWorkItem,
    AutonomyLevel,
    RiskLevel,
    WorkItemPriority,
    WorkItemStatus,
)
from sqlalchemy import func as sa_func

from app.operation_log.service import OperationLogService
from app.agentos.command_handlers import HANDLER_MAP


COMMAND_ADAPTERS: dict[str, dict[str, str]] = {
    "daily_report": {"command_name": "daily_report", "execution_mode": "integrated"},
    "listing_draft": {"command_name": "listing_draft", "execution_mode": "integrated"},
    "profit_review": {"command_name": "profit_review", "execution_mode": "integrated"},
    "inventory_allocate": {
        "command_name": "inventory_allocate",
        "execution_mode": "integrated",
    },
    "notify": {"command_name": "notify", "execution_mode": "integrated"},
}


def resolve_command_adapter(action_type: str) -> dict[str, str]:
    adapter = COMMAND_ADAPTERS.get(action_type)
    if adapter is None:
        raise ValueError(f"Business command adapter not registered: {action_type}")
    return adapter


RISK_TO_PRIORITY = {
    "low": WorkItemPriority.LOW,
    "medium": WorkItemPriority.MEDIUM,
    "high": WorkItemPriority.HIGH,
    "critical": WorkItemPriority.CRITICAL,
}


PROPOSAL_STATUS_TO_WORK_STATUS = {
    "suggested": WorkItemStatus.PENDING,
    "pending_approval": WorkItemStatus.PENDING,
    "approved": WorkItemStatus.IN_PROGRESS,
    "executing": WorkItemStatus.IN_PROGRESS,
    "executed": WorkItemStatus.COMPLETED,
    "reviewed": WorkItemStatus.COMPLETED,
    "rejected": WorkItemStatus.CANCELLED,
    "expired": WorkItemStatus.CANCELLED,
    "blocked_by_policy": WorkItemStatus.BLOCKED,
    "failed": WorkItemStatus.FAILED,
    "cancelled": WorkItemStatus.CANCELLED,
}


class ActionCenterService:
    @staticmethod
    def _operator(user: Any) -> str:
        if user is None:
            return "system"
        return getattr(user, "username", None) or str(getattr(user, "id", "system"))

    @staticmethod
    def _user_id(user: Any) -> int:
        if user is None:
            return 0
        return int(getattr(user, "id", 0) or 0)

    @staticmethod
    def _decimal_to_float(value: Any) -> float | None:
        if value is None:
            return None
        if isinstance(value, Decimal):
            return float(value)
        return float(value)

    @staticmethod
    def _proposal_to_dict(row: ActionProposal) -> dict[str, Any]:
        return {
            "id": row.id,
            "source_type": row.source_type,
            "source_id": row.source_id,
            "agent_id": row.agent_id,
            "squad_id": row.squad_id,
            "action_type": row.action_type,
            "business_object_type": row.business_object_type,
            "business_object_id": row.business_object_id,
            "title": row.title,
            "description": row.description,
            "proposed_payload": row.proposed_payload or {},
            "before_snapshot": row.before_snapshot,
            "after_snapshot": row.after_snapshot,
            "risk_level": row.risk_level,
            "requires_approval": bool(row.requires_approval),
            "status": row.status,
            "confidence": ActionCenterService._decimal_to_float(row.confidence),
            "proposed_by": row.proposed_by,
            "approved_by": row.approved_by,
            "rejected_by": row.rejected_by,
            "rejection_reason": row.rejection_reason,
            "created_at": row.created_at.isoformat() if row.created_at else None,
            "updated_at": row.updated_at.isoformat() if row.updated_at else None,
        }

    @staticmethod
    def proposal_to_work_item(row: ActionProposal) -> AgentOSWorkItem:
        risk = RiskLevel(row.risk_level)
        return AgentOSWorkItem(
            id=f"action_proposal:{row.id}",
            source_type="action_proposal",
            source_id=str(row.id),
            title=row.title,
            description=row.description,
            priority=RISK_TO_PRIORITY.get(row.risk_level, WorkItemPriority.MEDIUM),
            status=PROPOSAL_STATUS_TO_WORK_STATUS.get(
                row.status, WorkItemStatus.PENDING
            ),
            risk_level=risk,
            agent_id=row.agent_id,
            agent_name=None,
            squad_id=row.squad_id,
            squad_name=None,
            autonomy_level=AutonomyLevel.SUGGESTION,
            requires_approval=bool(
                row.requires_approval
                and row.status in {"suggested", "pending_approval"}
            ),
            created_at=row.created_at,
            updated_at=row.updated_at,
            action_url=f"/agentos/work-items?action_proposal={row.id}",
            metadata={
                "action_type": row.action_type,
                "business_object_type": row.business_object_type,
                "business_object_id": row.business_object_id,
                "payload": row.proposed_payload or {},
                "confidence": ActionCenterService._decimal_to_float(row.confidence),
                "proposal_status": row.status,
            },
        )

    @staticmethod
    async def create_proposal(
        db: AsyncSession,
        payload: ActionProposalCreate,
        operator: str,
    ) -> AgentOSWorkItem:
        status = "pending_approval" if payload.requires_approval else "suggested"
        proposal = ActionProposal(
            source_type=payload.source_type,
            source_id=payload.source_id,
            agent_id=payload.agent_id,
            squad_id=payload.squad_id,
            action_type=payload.action_type,
            business_object_type=payload.business_object_type,
            business_object_id=payload.business_object_id,
            title=payload.title,
            description=payload.description,
            proposed_payload=payload.proposed_payload,
            before_snapshot=payload.before_snapshot,
            risk_level=payload.risk_level.value,
            requires_approval=payload.requires_approval,
            status=status,
            confidence=payload.confidence,
            proposed_by=operator,
        )
        db.add(proposal)
        await db.flush()
        await db.refresh(proposal)
        await OperationLogService.log(
            db,
            module="agentos_action_center",
            action="propose",
            resource_id=str(proposal.id),
            content=f"创建动作提案: {proposal.action_type} - {proposal.title}",
            operator=operator,
        )
        return ActionCenterService.proposal_to_work_item(proposal)

    @staticmethod
    async def list_proposals(
        db: AsyncSession,
        status: str | None = None,
        risk_level: str | None = None,
        squad_id: str | None = None,
        limit: int = 50,
        offset: int = 0,
    ) -> tuple[list[ActionProposal], int]:
        stmt = select(ActionProposal)
        if status:
            stmt = stmt.where(ActionProposal.status == status)
        if risk_level:
            stmt = stmt.where(ActionProposal.risk_level == risk_level)
        if squad_id:
            stmt = stmt.where(ActionProposal.squad_id == squad_id)
        stmt = stmt.order_by(ActionProposal.created_at.desc())
        total_q = select(sa_func.count()).select_from(stmt.subquery())
        total = (await db.execute(total_q)).scalar() or 0
        rows = (await db.execute(stmt.offset(offset).limit(limit))).scalars().all()
        return list(rows), total

    @staticmethod
    def _ensure_transition(current: str, target: str) -> None:
        if target not in ACTION_PROPOSAL_STATUS_FLOW.get(current, set()):
            raise ValueError(f"状态 {current} 不允许流转到 {target}")

    @staticmethod
    async def approve(
        db: AsyncSession,
        proposal_id: int,
        operator: str,
        user_id: int,
        comment: str | None,
    ) -> dict[str, Any] | None:
        proposal = await db.get(ActionProposal, proposal_id)
        if proposal is None:
            return None
        ActionCenterService._ensure_transition(proposal.status, "approved")
        previous = proposal.status
        proposal.status = "approved"
        proposal.approved_by = operator
        proposal.approved_at = datetime.now(timezone.utc)
        approval = ApprovalRequest(
            proposal_id=proposal.id,
            requester=proposal.proposed_by,
            approver=operator,
            decision="approved",
            comment=comment,
            decided_at=datetime.now(timezone.utc),
        )
        db.add(approval)
        db.add(
            AgentOSOperationLog(
                user_id=user_id,
                item_id=f"action_proposal:{proposal.id}",
                action="approve",
                source_type="action_proposal",
                previous_status=previous,
                new_status="approved",
                comment=comment,
            )
        )
        await db.flush()
        await db.refresh(proposal)
        await db.refresh(approval)
        return {
            "ok": True,
            "proposal": ActionCenterService._proposal_to_dict(proposal),
            "approval": {
                "id": approval.id,
                "proposal_id": approval.proposal_id,
                "decision": approval.decision,
                "comment": approval.comment,
            },
        }

    @staticmethod
    async def reject(
        db: AsyncSession,
        proposal_id: int,
        operator: str,
        user_id: int,
        comment: str | None,
    ) -> dict[str, Any] | None:
        proposal = await db.get(ActionProposal, proposal_id)
        if proposal is None:
            return None
        ActionCenterService._ensure_transition(proposal.status, "rejected")
        previous = proposal.status
        proposal.status = "rejected"
        proposal.rejected_by = operator
        proposal.rejected_at = datetime.now(timezone.utc)
        proposal.rejection_reason = comment
        approval = ApprovalRequest(
            proposal_id=proposal.id,
            requester=proposal.proposed_by,
            approver=operator,
            decision="rejected",
            comment=comment,
            decided_at=datetime.now(timezone.utc),
        )
        db.add(approval)
        db.add(
            AgentOSOperationLog(
                user_id=user_id,
                item_id=f"action_proposal:{proposal.id}",
                action="reject",
                source_type="action_proposal",
                previous_status=previous,
                new_status="rejected",
                comment=comment,
            )
        )
        await db.flush()
        await db.refresh(proposal)
        await db.refresh(approval)
        return {
            "ok": True,
            "proposal": ActionCenterService._proposal_to_dict(proposal),
            "approval": {
                "id": approval.id,
                "proposal_id": approval.proposal_id,
                "decision": approval.decision,
                "comment": approval.comment,
            },
        }

    @staticmethod
    async def execute(
        db: AsyncSession,
        proposal_id: int,
        operator: str,
        user_id: int,
        executor: str | None,
    ) -> dict[str, Any] | None:
        proposal = await db.get(ActionProposal, proposal_id)
        if proposal is None:
            return None
        if proposal.requires_approval and proposal.status != "approved":
            raise ValueError("需要审批的动作必须先审批通过")

        # 状态流转: current -> "executing"
        ActionCenterService._ensure_transition(proposal.status, "executing")
        target_from = proposal.status
        proposal.status = "executing"
        await db.flush()

        # 安全检查 + 解析适配器
        adapter = resolve_command_adapter(proposal.action_type)

        # 创建执行记录（状态 start）
        execution = CommandExecution(
            proposal_id=proposal.id,
            command_name=adapter["command_name"],
            executor=executor or operator,
            status="started",
            input_payload=proposal.proposed_payload or {},
        )
        db.add(execution)
        await db.flush()

        # 调度到真实 handler
        handler = HANDLER_MAP.get(proposal.action_type)
        if handler is None:
            # 没有注册 handler 时回退到 record_only
            execution.status = "succeeded"
            execution.result_payload = {
                "mode": "record_only",
                "message": "No business command handler registered for this action type.",
            }
            proposal.status = "executed"
        else:
            try:
                result = await handler(db, proposal.proposed_payload or {})
                execution.status = "succeeded"
                execution.result_payload = result
                proposal.status = "executed"
                proposal.after_snapshot = result
            except Exception as exc:
                execution.status = "failed"
                execution.error_message = str(exc)
                proposal.status = "failed"
                proposal.after_snapshot = {"error": str(exc)}

        execution.finished_at = datetime.now(timezone.utc)
        db.add(
            AgentOSOperationLog(
                user_id=user_id,
                item_id=f"action_proposal:{proposal.id}",
                action="execute",
                source_type="action_proposal",
                previous_status=target_from,
                new_status=proposal.status,
            )
        )
        await db.flush()
        await db.refresh(proposal)
        await db.refresh(execution)
        return {
            "ok": True,
            "proposal": ActionCenterService._proposal_to_dict(proposal),
            "execution": {
                "id": execution.id,
                "proposal_id": execution.proposal_id,
                "status": execution.status,
                "command_name": execution.command_name,
                "error_message": execution.error_message,
            },
        }

    @staticmethod
    async def review(
        db: AsyncSession,
        proposal_id: int,
        operator: str,
        user_id: int,
        outcome: str,
        business_metric: str | None,
        metric_delta: float | None,
        notes: str | None,
    ) -> dict[str, Any] | None:
        proposal = await db.get(ActionProposal, proposal_id)
        if proposal is None:
            return None
        ActionCenterService._ensure_transition(proposal.status, "reviewed")
        previous = proposal.status
        proposal.status = "reviewed"
        review = OutcomeReview(
            proposal_id=proposal.id,
            outcome=outcome,
            business_metric=business_metric,
            metric_delta=metric_delta,
            notes=notes,
            reviewed_by=operator,
        )
        db.add(review)
        db.add(
            AgentOSOperationLog(
                user_id=user_id,
                item_id=f"action_proposal:{proposal.id}",
                action="review",
                source_type="action_proposal",
                previous_status=previous,
                new_status="reviewed",
                comment=notes,
            )
        )
        await db.flush()
        await db.refresh(proposal)
        await db.refresh(review)
        return {
            "ok": True,
            "proposal": ActionCenterService._proposal_to_dict(proposal),
            "review": {
                "id": review.id,
                "proposal_id": review.proposal_id,
                "outcome": review.outcome,
                "business_metric": review.business_metric,
                "metric_delta": ActionCenterService._decimal_to_float(
                    review.metric_delta
                ),
                "notes": review.notes,
            },
        }
