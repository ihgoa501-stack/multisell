"""熵管理 API 路由"""
from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.auth import require_permission
from app.common import Result, PageResult
from app.models import User
from app.agent.entropy.schemas import RuleMarkChangeVO, RuleHealthVO, SpcStatusVO, DefenseActionVO, EntropySummaryVO
from app.agent.entropy.service import EntropyService

router = APIRouter(tags=["熵管理系统"])
entropy_service = EntropyService()


@router.get("/entropy/dashboard", summary="熵管理驾驶舱")
async def get_dashboard(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    dashboard = await entropy_service.get_dashboard(db, current_user.id)
    return Result.ok(dashboard)


@router.post("/entropy/defend", summary="执行防守动作 (TTL+Budget+Decay+Merge+Regret)")
async def run_defenses(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    result = await entropy_service.run_defenses(db, current_user.id)
    return Result.ok(result)


@router.get("/entropy/health", summary="规则健康评分列表")
async def get_health_scores(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    scores = await entropy_service.get_health_scores(db, current_user.id)
    return Result.ok(scores)


@router.get("/entropy/spc", summary="SPC 控制状态")
async def get_spc_status(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    status = await entropy_service.get_spc_status(db, current_user.id)
    return Result.ok(status)


@router.get("/entropy/changes", summary="变更日志")
async def get_change_log(
    source_type: str = Query(None, description="筛选来源类型"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    changes, total = await entropy_service.get_change_log(
        db, current_user.id, source_type, page, page_size,
    )
    return PageResult.ok(
        records=[RuleMarkChangeVO.model_validate(c) for c in changes],
        total=total, page=page, page_size=page_size,
    )
