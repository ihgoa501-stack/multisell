"""平台费用规则 - 路由"""

from typing import Optional

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.database import get_db
from app.common import Result
from app.models import User
from app.operation_log.service import OperationLogService
from app.platform_fee.schemas import (
    PlatformFeeRuleCreate,
    PlatformFeeRuleUpdate,
    PlatformFeeRuleMatchRequest,
)
from app.platform_fee.service import PlatformFeeRuleService

router = APIRouter(tags=["平台费用规则"])


@router.post("/platform-fee-rules", summary="创建平台费用规则")
async def create_rule(
    data: PlatformFeeRuleCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    rule = await PlatformFeeRuleService.create(db, data)
    await OperationLogService.log(
        db,
        module="platform_fee",
        action="create",
        resource_id=str(rule["id"]),
        content=f"创建平台费用规则: 平台ID={rule['platform_id']}, 站点={rule.get('site_code') or '全局'}",
        operator=current_user.username,
    )
    return Result.ok(rule)


@router.get("/platform-fee-rules", summary="平台费用规则列表")
async def list_rules(
    platform_id: Optional[int] = Query(None, description="按平台筛选"),
    site_code: Optional[str] = Query(None, description="按站点筛选"),
    category_id: Optional[int] = Query(None, description="按类目筛选"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:view")),
):
    rules = await PlatformFeeRuleService.list(db, platform_id, site_code, category_id)
    return Result.ok(rules)


@router.get("/platform-fee-rules/{rule_id}", summary="平台费用规则详情")
async def get_rule(
    rule_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:view")),
):
    rule = await PlatformFeeRuleService.get_by_id(db, rule_id)
    if not rule:
        return Result.not_found("平台费用规则不存在")
    return Result.ok(rule)


@router.put("/platform-fee-rules/{rule_id}", summary="更新平台费用规则")
async def update_rule(
    rule_id: int,
    data: PlatformFeeRuleUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    rule = await PlatformFeeRuleService.update(db, rule_id, data)
    if not rule:
        return Result.not_found("平台费用规则不存在")
    await OperationLogService.log(
        db,
        module="platform_fee",
        action="update",
        resource_id=str(rule_id),
        content=f"更新平台费用规则: 平台ID={rule['platform_id']}",
        operator=current_user.username,
    )
    return Result.ok(rule)


@router.delete("/platform-fee-rules/{rule_id}", summary="删除平台费用规则")
async def delete_rule(
    rule_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    ok = await PlatformFeeRuleService.delete(db, rule_id)
    if not ok:
        return Result.not_found("平台费用规则不存在")
    await OperationLogService.log(
        db,
        module="platform_fee",
        action="delete",
        resource_id=str(rule_id),
        content=f"删除平台费用规则: {rule_id}",
        operator=current_user.username,
    )
    return Result.ok(message="删除成功")


@router.post("/platform-fee-rules/match", summary="匹配平台费用规则")
async def match_rule(
    data: PlatformFeeRuleMatchRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:calculate")),
):
    rule = await PlatformFeeRuleService.match(db, data)
    return Result.ok(rule)
