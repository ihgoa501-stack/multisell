"""系统配置 API 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.auth import require_permission
from app.common import Result
from app.models import User
from app.agent.config_service import ConfigService, CONFIG_DEFS

router = APIRouter(tags=["系统配置"])


@router.get("/settings/llm", summary="获取 LLM 配置")
async def get_llm_settings(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    configs = await ConfigService.get_all(db)
    # 附上配置定义（前端表单渲染用）
    return Result.ok(
        {
            "configs": configs,
            "definitions": CONFIG_DEFS,
        }
    )


@router.put("/settings/llm", summary="更新 LLM 配置")
async def update_llm_settings(
    data: dict,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:execute")),
):
    allowed = set(CONFIG_DEFS.keys())
    for key, value in data.items():
        if key in allowed:
            await ConfigService.set(db, key, value, current_user.id)
    await db.commit()
    return Result.ok(message="配置已更新")
