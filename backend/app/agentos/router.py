"""AgentOS 聚合路由"""

from fastapi import APIRouter, Depends, Path

from app.auth import require_permission
from app.common.schemas import PageResult, Result
from app.database import get_db
from app.models import User

from .schemas import WorkItemApproval, WorkItemStatusUpdate
from .service import AgentOSService

router = APIRouter(tags=["AgentOS"])


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
