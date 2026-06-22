"""AgentOS 聚合路由"""

from fastapi import APIRouter, Depends, HTTPException, Path

from app.auth import require_permission
from app.common.schemas import PageResult, Result
from app.database import get_db
from app.models import User

from .action_center_service import ActionCenterService
from .schemas import (
    ActionApprovalPayload,
    ActionExecutionPayload,
    ActionProposalCreate,
    ActionReviewPayload,
    PolicyCreate,
    WorkItemApproval,
    WorkItemStatusUpdate,
)
from .service import AgentOSService

router = APIRouter(tags=["AgentOS"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.get("/agentos/control-center", summary="AgentOS 总控台")
async def control_center(
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回全局概览、团队状态、优先任务、指标和最近活动"""
    data = await AgentOSService.get_control_center(db)
    return Result.ok(data)


@router.get("/agentos/work-items", summary="AgentOS 任务列表")
async def list_work_items(
    status: str | None = None,
    priority: str | None = None,
    squad: str | None = None,
    agent_id: str | None = None,
    source_type: str | None = None,
    requires_approval: bool | None = None,
    limit: int = 20,
    offset: int = 0,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回分页 WorkItem 列表"""
    data = await AgentOSService.get_work_items(
        db,
        status=status,
        priority=priority,
        squad=squad,
        agent_id=agent_id,
        source_type=source_type,
        requires_approval=requires_approval,
        limit=limit,
        offset=offset,
    )
    return PageResult.ok(
        records=data["items"],
        total=data["total"],
        page=(offset // limit) + 1,
        page_size=limit,
    )


@router.get("/agentos/squads", summary="AgentOS 团队列表")
async def list_squads(
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回 Agent 团队列表及摘要"""
    data = await AgentOSService.get_squads(db)
    return Result.ok(data)


@router.get("/agentos/templates", summary="AgentOS 模板列表")
async def list_templates(
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回内置模板列表"""
    data = await AgentOSService.get_templates(db)
    return Result.ok(data)


@router.get("/agentos/operations", summary="AgentOS 操作审计日志")
async def list_operations(
    item_id: str | None = None,
    action: str | None = None,
    source_type: str | None = None,
    user_id: int | None = None,
    limit: int = 20,
    offset: int = 0,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """查询审批/状态变更操作日志"""
    data = await AgentOSService.get_operations(
        db, item_id=item_id, action=action, source_type=source_type,
        user_id=user_id, limit=limit, offset=offset,
    )
    return PageResult.ok(
        records=data["records"],
        total=data["total"],
        page=(offset // limit) + 1,
        page_size=limit,
    )


# ── Phase 2: Action Proposal 端点 ──────────────────────────


@router.post("/agentos/action-proposals", summary="创建 AgentOS 动作提案")
async def create_action_proposal(
    body: ActionProposalCreate,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    item = await ActionCenterService.create_proposal(
        db, body, operator=_operator(current_user),
    )
    return Result.ok(item)


@router.post("/agentos/action-proposals/{proposal_id}/approve", summary="审批通过动作提案")
async def approve_action_proposal(
    proposal_id: int,
    body: ActionApprovalPayload,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    try:
        result = await ActionCenterService.approve(
            db,
            proposal_id,
            operator=_operator(current_user),
            user_id=current_user.id,
            comment=body.comment,
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))
    if result is None:
        return Result.not_found("ActionProposal not found")
    return Result.ok(result)


@router.post("/agentos/action-proposals/{proposal_id}/reject", summary="拒绝动作提案")
async def reject_action_proposal(
    proposal_id: int,
    body: ActionApprovalPayload,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    try:
        result = await ActionCenterService.reject(
            db,
            proposal_id,
            operator=_operator(current_user),
            user_id=current_user.id,
            comment=body.comment,
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))
    if result is None:
        return Result.not_found("ActionProposal not found")
    return Result.ok(result)


@router.post("/agentos/action-proposals/{proposal_id}/execute", summary="执行动作提案")
async def execute_action_proposal(
    proposal_id: int,
    body: ActionExecutionPayload,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    try:
        result = await ActionCenterService.execute(
            db,
            proposal_id,
            operator=_operator(current_user),
            user_id=current_user.id,
            executor=body.executor,
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))
    if result is None:
        return Result.not_found("ActionProposal not found")
    return Result.ok(result)


@router.post("/agentos/action-proposals/{proposal_id}/review", summary="复盘动作结果")
async def review_action_proposal(
    proposal_id: int,
    body: ActionReviewPayload,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    try:
        result = await ActionCenterService.review(
            db,
            proposal_id,
            operator=_operator(current_user),
            user_id=current_user.id,
            outcome=body.outcome,
            business_metric=body.business_metric,
            metric_delta=body.metric_delta,
            notes=body.notes,
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))
    if result is None:
        return Result.not_found("ActionProposal not found")
    return Result.ok(result)


@router.post("/agentos/action-proposals/{proposal_id}/undo", summary="撤销已执行的动作提案")
async def undo_action_proposal(
    proposal_id: int,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:action")),
):
    """Create a compensation proposal for an executed action."""
    result = await ActionCenterService.create_compensation(
        db,
        proposal_id,
        operator=_operator(current_user),
        user_id=current_user.id,
    )
    if result is None:
        raise HTTPException(status_code=404, detail="Action proposal not found or not undoable")
    return Result.ok(result)


# ── Phase 2: Mutation 操作 ──────────────────────────────


@router.patch("/agentos/work-items/{item_id}/status", summary="更新 WorkItem 状态")
async def update_work_item_status(
    item_id: str = Path(..., description="WorkItem ID (e.g. exception:42)"),
    body: WorkItemStatusUpdate = ...,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    """标记 WorkItem 为已读/处理中/已完成"""
    result = await AgentOSService.update_work_item_status(
        db, item_id, current_user.id, body.status.value,
    )
    if not result["ok"]:
        if result.get("error") == "not_found":
            return Result.not_found("WorkItem not found")
        return Result.bad_request(result["error"])
    return Result.ok(result)


@router.post("/agentos/work-items/{item_id}/approve", summary="审批通过 WorkItem")
async def approve_work_item(
    item_id: str = Path(..., description="WorkItem ID"),
    body: WorkItemApproval = ...,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    """审批通过并触发底层执行"""
    result = await AgentOSService.approve_work_item(
        db, item_id, current_user.id, body.comment,
    )
    if not result["ok"]:
        if result.get("error") == "not_found":
            return Result.not_found("WorkItem not found")
        return Result.bad_request(result["error"])
    return Result.ok(result)


@router.post("/agentos/work-items/{item_id}/reject", summary="拒绝 WorkItem")
async def reject_work_item(
    item_id: str = Path(..., description="WorkItem ID"),
    body: WorkItemApproval = ...,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    """拒绝 WorkItem 并记录理由"""
    result = await AgentOSService.reject_work_item(
        db, item_id, current_user.id, body.comment,
    )
    if not result["ok"]:
        if result.get("error") == "not_found":
            return Result.not_found("WorkItem not found")
        return Result.bad_request(result["error"])
    return Result.ok(result)


# ── Phase 3: Autonomy Upgrade ────────────────────────────


@router.get("/agentos/agents/upgrade-candidates", summary="自治等级升级候选")
async def list_upgrade_candidates(
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回建议升级/降级的 Agent 候选列表"""
    result = await AgentOSService.get_upgrade_candidates(db, current_user.id)
    return Result.ok(result)


@router.post("/agentos/agents/{agent_id}/upgrade", summary="执行自治等级升级")
async def upgrade_agent_level(
    agent_id: str,
    target_level: str,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    """将 Agent 升级到目标自治等级"""
    result = await AgentOSService.execute_upgrade(
        db, current_user.id, agent_id, target_level,
    )
    return Result.ok(result)


@router.post("/agentos/agents/{agent_id}/downgrade", summary="执行自治等级降级")
async def downgrade_agent_level(
    agent_id: str,
    target_level: str,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    """将 Agent 降级到目标自治等级"""
    result = await AgentOSService.execute_downgrade(
        db, current_user.id, agent_id, target_level,
    )
    return Result.ok(result)


# ── Phase 4 Finale: Agent Detail ──────────────────────────


@router.get("/agentos/agents/{agent_id}/detail", summary="Agent 详情")
async def get_agent_detail(
    agent_id: str,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回单个 Agent 的详情、WorkItem 列表和操作记录"""
    data = await AgentOSService.get_agent_detail(db, current_user.id, agent_id)
    return Result.ok(data)


# ── Phase 6: Policy Matrix CRUD (§2.2) ──────────────────────────


@router.get("/agentos/policies", summary="策略矩阵列表")
async def list_policies(
    agent_id: str | None = None,
    page: int = 1,
    page_size: int = 20,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回当前用户的分页冲突仲裁策略列表"""
    from .policy_service import PolicyService

    records, total = await PolicyService.list_policies(
        db, current_user.id, agent_id=agent_id, page=page, page_size=page_size,
    )

    def _policy_to_dict(p):
        return {
            "id": p.id,
            "agent_id_a": p.agent_id_a,
            "agent_id_b": p.agent_id_b,
            "decision_point": p.decision_point,
            "condition": p.condition,
            "winner": p.winner,
            "reason": p.reason,
            "priority": p.priority,
            "created_at": p.created_at.isoformat() if p.created_at else None,
            "updated_at": p.updated_at.isoformat() if p.updated_at else None,
        }

    return PageResult.ok(
        records=[_policy_to_dict(r) for r in records],
        total=total,
        page=page,
        page_size=page_size,
    )


@router.post("/agentos/policies", summary="创建冲突仲裁策略")
async def create_policy(
    body: PolicyCreate,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    """创建一条冲突仲裁策略"""
    from .policy_service import PolicyService

    try:
        policy = await PolicyService.create_policy(
            db,
            current_user.id,
            data=body.model_dump(exclude_unset=True),
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))

    return Result.ok({
        "id": policy.id,
        "agent_id_a": policy.agent_id_a,
        "agent_id_b": policy.agent_id_b,
        "decision_point": policy.decision_point,
        "condition": policy.condition,
        "winner": policy.winner,
        "reason": policy.reason,
        "priority": policy.priority,
        "created_at": policy.created_at.isoformat() if policy.created_at else None,
        "updated_at": policy.updated_at.isoformat() if policy.updated_at else None,
    })


@router.delete("/agentos/policies/{policy_id}", summary="删除冲突仲裁策略")
async def delete_policy(
    policy_id: int,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    """删除一条冲突仲裁策略"""
    from .policy_service import PolicyService

    ok = await PolicyService.delete_policy(db, policy_id, current_user.id)
    if not ok:
        return Result.not_found("Policy not found")
    return Result.ok({"deleted": True})
