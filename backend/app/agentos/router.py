from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import PageResult, Result
from app.database import get_db
from app.models import User
from app.agentos.schemas import (
    ControlCenterSummaryVO,
    SquadVO,
    TemplateCardVO,
    WorkItemVO,
)
from app.agentos.service import AgentOSService, TEMPLATE_CARDS

router = APIRouter(tags=["AgentOS 工作台"])


@router.get("/agentos/work-items", summary="AgentOS 统一工作项")
async def list_work_items(
    source_type: str | None = Query(None),
    squad: str | None = Query(None),
    status: str | None = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    items, total = await AgentOSService.list_work_items(
        db,
        current_user.id,
        source_type=source_type,
        squad=squad,
        status=status,
        page=page,
        page_size=page_size,
    )
    return PageResult.ok(
        records=[WorkItemVO.model_validate(item) for item in items],
        total=total,
        page=page,
        page_size=page_size,
    )


@router.get("/agentos/control-center", summary="AgentOS 总控台")
async def get_control_center(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    return Result.ok(await AgentOSService.get_control_center(db, current_user.id))


@router.get("/agentos/squads", summary="AgentOS Agent 小队")
async def list_squads(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    squads = await AgentOSService.get_squads(db, current_user.id)
    return Result.ok([SquadVO.model_validate(squad) for squad in squads])


@router.get("/agentos/templates", summary="AgentOS 内置模板")
async def list_templates(
    current_user: User = Depends(require_permission("agent:view")),
):
    return Result.ok([TemplateCardVO.model_validate(template) for template in TEMPLATE_CARDS])
