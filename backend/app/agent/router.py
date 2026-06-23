"""Agent API 路由

注意：路由注册顺序很重要。
/agents/decisions, /agents/rules, /agents/profile, /agents/episodes
等静态路径必须定义在 /agents/{agent_id} 之前，
否则 FastAPI 会将 "decisions/rules/profile/episodes" 匹配为 {agent_id}。
"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.auth import require_permission
from app.common import Result, PageResult
from app.models import User
from app.agent.schemas import (
    DecisionLogVO,
    PersonalRuleVO,
    PersonalRuleCreate,
    PersonalRuleUpdate,
    HonchoProfileVO,
    HonchoProfileUpdate,
    AgentDecisionRequest,
    FeedbackRequest,
    EpisodeVO,
    StageChangeRequest,
    NudgeRespondRequest,
)
from app.agent.registry import AgentRegistry
from app.agent.service import AgentService
from app.agent.action_service import AgentActionService
from app.agent.evolution_service import EvolutionService
from app.agent.base import EvolutionStage
from app.agent.scheduler import scheduler as _agent_scheduler
from app.agent.pipeline import evaluate_chains
from app.agent.event_bus import event_bus as _agent_event_bus

router = APIRouter(tags=["AI Agent 系统"])


# ── 静态路径（必须放在 /agents/{agent_id} 之前） ──


@router.get("/agents", summary="Agent列表")
async def list_agents():
    return Result.ok(AgentRegistry.list_agents())


@router.get("/agents/decisions", summary="决策日志列表")
async def list_decisions(
    agent_id: str = Query(None),
    decision_point: str = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    logs, total = await AgentService.get_decision_logs(
        db, current_user.id, agent_id, decision_point, page, page_size
    )
    return PageResult.ok(
        records=[DecisionLogVO.model_validate(rec) for rec in logs],
        total=total,
        page=page,
        page_size=page_size,
    )


@router.post("/agents/decisions/{decision_id}/feedback", summary="提交决策反馈")
async def submit_feedback(
    decision_id: int,
    req: FeedbackRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    decision = await AgentService.update_decision_feedback(
        db,
        decision_id,
        req.user_action,
        user_overrides=req.user_overrides,
        user_feedback=req.user_feedback,
    )
    if not decision:
        return Result.not_found("决策记录不存在")
    return Result.ok(DecisionLogVO.model_validate(decision))


@router.get("/agents/actions", summary="待执行操作列表")
async def list_actions(
    agent_id: str = Query(None),
    status: str = Query("pending"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    actions, total = await AgentActionService.list_pending(
        db, current_user.id, agent_id, status, page, page_size
    )
    return PageResult.ok(
        records=[
            {
                "id": a.id,
                "agent_id": a.agent_id,
                "decision_id": a.decision_id,
                "action_type": a.action_type,
                "status": a.status,
                "summary": a.summary,
                "action_payload": a.action_payload,
                "execution_result": a.execution_result,
                "created_at": a.created_at.isoformat() if a.created_at else None,
            }
            for a in actions
        ],
        total=total,
        page=page,
        page_size=page_size,
    )


@router.post("/agents/actions/{action_id}/execute", summary="确认并执行操作")
async def execute_action(
    action_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    action = await AgentActionService.execute_action(db, action_id, current_user.id)
    if not action:
        return Result.not_found("操作不存在或无权执行")
    return Result.ok(
        {
            "id": action.id,
            "status": action.status,
            "execution_result": action.execution_result,
        }
    )


@router.post("/agents/actions/{action_id}/reject", summary="拒绝操作")
async def reject_action(
    action_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    action = await AgentActionService.reject_action(db, action_id, current_user.id)
    if not action:
        return Result.not_found("操作不存在或无权拒绝")
    return Result.ok({"id": action.id, "status": action.status})


@router.get("/agents/dashboard", summary="运营驾驶舱")
async def get_dashboard(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    dashboard = await AgentService.get_dashboard(db, current_user.id)
    return Result.ok(dashboard)


@router.get("/agents/rules", summary="个人规则列表")
async def list_rules(
    agent_id: str = Query(None),
    decision_point: str = Query(None),
    status: str = Query(None),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    rules = await AgentService.list_rules(
        db, current_user.id, agent_id, decision_point, status
    )
    return Result.ok([PersonalRuleVO.model_validate(r) for r in rules])


@router.post("/agents/rules", summary="创建个人规则")
async def create_rule(
    data: PersonalRuleCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    rule = await AgentService.create_rule(db, current_user.id, data.model_dump())
    return Result.ok(PersonalRuleVO.model_validate(rule))


@router.put("/agents/rules/{rule_id}", summary="更新个人规则")
async def update_rule(
    rule_id: int,
    data: PersonalRuleUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    rule = await AgentService.update_rule(
        db, rule_id, current_user.id, data.model_dump(exclude_none=True)
    )
    if not rule:
        return Result.not_found("规则不存在或无权修改")
    return Result.ok(PersonalRuleVO.model_validate(rule))


@router.delete("/agents/rules/{rule_id}", summary="删除个人规则")
async def delete_rule(
    rule_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    ok = await AgentService.delete_rule(db, rule_id, current_user.id)
    if not ok:
        return Result.not_found("规则不存在或无权删除")
    return Result.ok(message="规则已删除")


@router.get("/agents/profile", summary="Honcho用户模型")
async def get_honcho_profile(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    profile = await AgentService.get_or_create_honcho_profile(db, current_user.id)
    return Result.ok(HonchoProfileVO.model_validate(profile))


@router.put("/agents/profile", summary="更新Honcho用户模型")
async def update_honcho_profile(
    data: HonchoProfileUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    profile = await AgentService.update_honcho_profile(
        db, current_user.id, data.model_dump(exclude_none=True)
    )
    return Result.ok(HonchoProfileVO.model_validate(profile))


@router.get("/agents/episodes", summary="Episode列表")
async def list_episodes(
    agent_id: str = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    episodes, total = await AgentService.list_episodes(
        db, current_user.id, agent_id, page, page_size
    )
    return PageResult.ok(
        records=[EpisodeVO.model_validate(e) for e in episodes],
        total=total,
        page=page,
        page_size=page_size,
    )


# ── 事件总线（放在动态路径之前） ──


@router.get("/agents/events/routes", summary="事件路由列表")
async def list_event_routes(
    current_user: User = Depends(require_permission("agent:view")),
):
    return Result.ok(_agent_event_bus.get_routes())


@router.post("/agents/events/emit", summary="手动触发事件")
async def emit_event(
    event_type: str = Query(..., description="事件类型"),
    payload: dict = {},
    source: str = Query("manual", description="来源"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    matched = await _agent_event_bus.emit(event_type, payload, source)
    return Result.ok({"event_type": event_type, "matched_routes": matched})


# ── 调度管理（放在动态路径之前） ──


@router.get("/agents/schedules", summary="所有 Agent 调度配置")
async def list_schedules(
    current_user: User = Depends(require_permission("agent:view")),
):
    return Result.ok(_agent_scheduler.get_schedules())


@router.get("/agents/schedules/{agent_id}", summary="单个 Agent 调度配置")
async def get_schedule(
    agent_id: str,
    current_user: User = Depends(require_permission("agent:view")),
):
    cfg = _agent_scheduler.get_schedule(agent_id)
    if not cfg:
        return Result.not_found(f"Agent {agent_id} 未配置调度")
    return Result.ok(cfg)


@router.put("/agents/schedules/{agent_id}", summary="更新调度配置")
async def update_schedule(
    agent_id: str,
    config: dict,
    current_user: User = Depends(require_permission("agent:manage")),
):
    ok = _agent_scheduler.update_schedule(agent_id, config)
    if not ok:
        return Result.not_found(f"Agent {agent_id} 未配置调度")
    return Result.ok(_agent_scheduler.get_schedule(agent_id))


@router.post("/agents/schedules/{agent_id}/trigger", summary="手动触发调度")
async def trigger_schedule(
    agent_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    result = await _agent_scheduler.trigger_now(agent_id, db)
    if "error" in result:
        return Result.bad_request(result["error"])
    return Result.ok(result)


# ── 进化/自治等级控制（放在动态路径之前） ──


@router.get("/agents/evolution/overview", summary="Agent自治等级总览")
async def evolution_overview(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    data = await EvolutionService.get_overview(db, current_user.id)
    return Result.ok(data)


@router.get("/agents/evolution/nudge/pending", summary="待处理Nudge列表")
async def list_pending_nudges(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    nudges = await EvolutionService.get_pending_nudges(db, current_user.id)
    return Result.ok(nudges)


@router.post("/agents/evolution/nudge/{nudge_id}/respond", summary="响应Nudge提示")
async def respond_nudge(
    nudge_id: int,
    req: NudgeRespondRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    result = await EvolutionService.respond_nudge(
        db, current_user.id, nudge_id, req.response
    )
    if not result.get("success"):
        return Result.bad_request(result.get("message", "操作失败"))
    return Result.ok(result)


@router.post("/agents/evolution/generate-nudges", summary="手动触发Nudge生成")
async def generate_nudges(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    nudges = await EvolutionService.generate_nudges(db, current_user.id)
    return Result.ok({"generated": len(nudges), "nudges": nudges})


# ── 进化动态路径 ──


@router.get("/agents/evolution/{agent_id}", summary="Agent进化详情与等级控制")
async def evolution_agent_detail(
    agent_id: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    data = await EvolutionService.get_agent_detail(db, current_user.id, agent_id)
    if not data:
        return Result.not_found(f"Agent {agent_id} 不存在")
    return Result.ok(data)


@router.put("/agents/evolution/{agent_id}/stage", summary="变更Agent自治阶段")
async def evolution_change_stage(
    agent_id: str,
    req: StageChangeRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    result = await EvolutionService.change_stage(
        db,
        current_user.id,
        agent_id,
        req.decision_point,
        req.target_stage,
    )
    if not result.get("success"):
        return Result.bad_request(result.get("message", "操作失败"))
    return Result.ok(result)


@router.get("/agents/{agent_id}", summary="Agent详情")
async def get_agent(agent_id: str):
    meta = AgentRegistry.get_metadata(agent_id)
    if not meta:
        return Result.not_found(f"Agent {agent_id} 不存在")
    return Result.ok(meta)


@router.post("/agents/{agent_id}/decide", summary="Agent决策")
async def agent_decide(
    agent_id: str,
    req: AgentDecisionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    agent_cls = AgentRegistry.get_agent_class(agent_id)
    if not agent_cls:
        return Result.not_found(f"Agent {agent_id} 不存在")

    # 从 DB 加载进化阶段配置
    from app.agent.evolution_service import EvolutionService as EvoSvc

    config = await EvoSvc.get_or_create_config(
        db,
        current_user.id,
        agent_id,
        req.decision_point,
    )
    stage_override = {req.decision_point: EvolutionStage(config.current_stage)}

    agent = agent_cls(user_id=current_user.id, stage_override=stage_override)
    result = await AgentService.execute_decision(
        db, agent, req.decision_point, req.context, dry_run=req.dry_run
    )

    # ── 自动触发协作链 ──
    if not req.dry_run and result.get("decision_id"):
        chain_results = await evaluate_chains(
            agent_id,
            req.decision_point,
            result,
            current_user.id,
            db,
        )
        result["chain_triggered"] = len(chain_results)
        if chain_results:
            result["chain_results"] = [
                {
                    "agent_id": r["agent_id"],
                    "decision_point": r["decision_point"],
                    "decision_id": r.get("decision_id"),
                    "chain_source": r.get("chain_source"),
                }
                for r in chain_results
            ]

    return Result.ok(result)
