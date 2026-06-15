"""Agent 动作提案与审批 - 路由"""

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.agent_actions.schemas import (
    AgentActionAfterSnapshot,
    AgentActionCreate,
    AgentActionReject,
)
from app.agent_actions.service import AgentActionService

router = APIRouter(tags=["Agent 动作"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.post("/agent-actions", summary="创建动作提案")
async def create_agent_action(
    data: AgentActionCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent_action:propose")),
):
    result = await AgentActionService.create(
        db, data.model_dump(exclude_unset=True), operator=_operator(current_user),
    )
    return Result.ok(result)


@router.get("/agent-actions", summary="动作提案列表")
async def list_agent_actions(
    exception_id: int | None = None,
    status: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent_action:view")),
):
    items = await AgentActionService.list_actions(db, exception_id=exception_id, status=status)
    return Result.ok(items)


@router.get("/agent-actions/{action_id}", summary="动作提案详情")
async def get_agent_action(
    action_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent_action:view")),
):
    item = await AgentActionService.get_action(db, action_id)
    if not item:
        raise HTTPException(status_code=404, detail="动作提案不存在")
    return Result.ok(item)


@router.post("/agent-actions/{action_id}/approve", summary="审批通过")
async def approve_agent_action(
    action_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent_action:approve")),
):
    try:
        item = await AgentActionService.approve(db, action_id, operator=_operator(current_user))
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    if not item:
        raise HTTPException(status_code=404, detail="动作提案不存在")
    return Result.ok(item)


@router.post("/agent-actions/{action_id}/reject", summary="驳回")
async def reject_agent_action(
    action_id: int,
    data: AgentActionReject,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent_action:approve")),
):
    try:
        item = await AgentActionService.reject(
            db, action_id, rejection_reason=data.rejection_reason, operator=_operator(current_user),
        )
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    if not item:
        raise HTTPException(status_code=404, detail="动作提案不存在")
    return Result.ok(item)


@router.post("/agent-actions/{action_id}/mark-executed", summary="标记执行")
async def mark_executed_agent_action(
    action_id: int,
    data: AgentActionAfterSnapshot,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent_action:execute")),
):
    try:
        item = await AgentActionService.mark_executed(
            db, action_id, after_snapshot=data.after_snapshot, operator=_operator(current_user),
        )
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    if not item:
        raise HTTPException(status_code=404, detail="动作提案不存在")
    return Result.ok(item)
