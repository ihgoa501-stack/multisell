"""Agent 动作提案与审批 - 服务层"""

from datetime import datetime, timezone
from typing import Any, Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import AgentActionProposal as AgentAction, AGENT_ACTION_STATUS_FLOW
from app.operation_log.service import OperationLogService


ALLOWED_TRANSITIONS = AGENT_ACTION_STATUS_FLOW


class AgentActionService:

    @staticmethod
    async def create(
        db: AsyncSession,
        data: dict,
        operator: Optional[str] = None,
    ) -> dict:
        action = AgentAction(
            source_module=data.get("source_module"),
            source_type=data.get("source_type"),
            source_id=data.get("source_id"),
            exception_id=data.get("exception_id"),
            action_type=data["action_type"],
            title=data["title"],
            description=data.get("description"),
            proposed_payload=data.get("proposed_payload"),
            before_snapshot=data.get("before_snapshot"),
            status="proposed",
            proposed_by=operator,
        )
        db.add(action)
        await db.flush()
        await db.refresh(action)

        await OperationLogService.log(
            db, module="agent_action", action="propose",
            resource_id=str(action.id),
            content=f"提议动作: {action.action_type} - {action.title}",
            operator=operator or "system",
        )

        return AgentActionService._to_dict(action)

    @staticmethod
    async def list_actions(
        db: AsyncSession,
        exception_id: Optional[int] = None,
        status: Optional[str] = None,
    ) -> list[dict]:
        stmt = select(AgentAction).order_by(AgentAction.created_at.desc())
        if exception_id:
            stmt = stmt.where(AgentAction.exception_id == exception_id)
        if status:
            stmt = stmt.where(AgentAction.status == status)

        result = await db.execute(stmt)
        items = result.scalars().all()
        return [AgentActionService._to_dict(it) for it in items]

    @staticmethod
    async def get_action(db: AsyncSession, action_id: int) -> Optional[dict]:
        action = await db.get(AgentAction, action_id)
        if not action:
            return None
        return AgentActionService._to_dict(action)

    @staticmethod
    async def approve(
        db: AsyncSession,
        action_id: int,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        action = await db.get(AgentAction, action_id)
        if not action:
            return None
        if "approved" not in ALLOWED_TRANSITIONS.get(action.status, set()):
            raise ValueError(f"状态 {action.status} 不允许审批")

        action.status = "approved"
        action.approved_by = operator
        action.approved_at = datetime.now(timezone.utc)
        await db.flush()
        await db.refresh(action)

        await OperationLogService.log(
            db, module="agent_action", action="approve",
            resource_id=str(action.id),
            content=f"审批通过: {action.title}",
            operator=operator or "system",
        )
        return AgentActionService._to_dict(action)

    @staticmethod
    async def reject(
        db: AsyncSession,
        action_id: int,
        rejection_reason: Optional[str] = None,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        action = await db.get(AgentAction, action_id)
        if not action:
            return None
        if "rejected" not in ALLOWED_TRANSITIONS.get(action.status, set()):
            raise ValueError(f"状态 {action.status} 不允许驳回")

        action.status = "rejected"
        action.rejected_by = operator
        action.rejected_at = datetime.now(timezone.utc)
        action.rejection_reason = rejection_reason
        await db.flush()
        await db.refresh(action)

        await OperationLogService.log(
            db, module="agent_action", action="reject",
            resource_id=str(action.id),
            content=f"驳回动作: {action.title}",
            operator=operator or "system",
        )
        return AgentActionService._to_dict(action)

    @staticmethod
    async def mark_executed(
        db: AsyncSession,
        action_id: int,
        after_snapshot: Optional[dict] = None,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        action = await db.get(AgentAction, action_id)
        if not action:
            return None
        if "executed" not in ALLOWED_TRANSITIONS.get(action.status, set()):
            raise ValueError(f"状态 {action.status} 不允许标记执行")

        action.status = "executed"
        action.executed_by = operator
        action.executed_at = datetime.now(timezone.utc)
        if after_snapshot:
            action.after_snapshot = after_snapshot
        await db.flush()
        await db.refresh(action)

        await OperationLogService.log(
            db, module="agent_action", action="execute",
            resource_id=str(action.id),
            content=f"执行动作: {action.title}",
            operator=operator or "system",
        )
        return AgentActionService._to_dict(action)

    @staticmethod
    def _to_dict(action: AgentAction) -> dict:
        return {
            "id": action.id,
            "source_module": action.source_module,
            "source_type": action.source_type,
            "source_id": action.source_id,
            "exception_id": action.exception_id,
            "action_type": action.action_type,
            "title": action.title,
            "description": action.description,
            "proposed_payload": action.proposed_payload,
            "before_snapshot": action.before_snapshot,
            "after_snapshot": action.after_snapshot,
            "status": action.status,
            "proposed_by": action.proposed_by,
            "approved_by": action.approved_by,
            "approved_at": action.approved_at.isoformat() if action.approved_at else None,
            "rejected_by": action.rejected_by,
            "rejected_at": action.rejected_at.isoformat() if action.rejected_at else None,
            "rejection_reason": action.rejection_reason,
            "executed_by": action.executed_by,
            "executed_at": action.executed_at.isoformat() if action.executed_at else None,
            "created_at": action.created_at.isoformat() if action.created_at else None,
        }
