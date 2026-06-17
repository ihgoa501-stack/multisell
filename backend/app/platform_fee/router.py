"""平台费用 - 路由"""

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.models import User
from app.common import Result, PageResult
from app.platform_fee.schemas import (
    PlatformFeeRuleCreate,
    PlatformFeeRuleUpdate,
    PlatformFeeRuleVO,
    PlatformFeeRuleQuery,
    PlatformFeeCalculateRequest,
    PlatformFeeCalculateResponse,
)
from app.platform_fee.service import PlatformFeeService
from app.operation_log.service import OperationLogService

router = APIRouter(prefix="/platform-fee", tags=["平台费用"])


def rule_to_vo(r) -> PlatformFeeRuleVO:
    return PlatformFeeRuleVO(
        id=r.id,
        platform_id=r.platform_id,
        country_code=r.country_code,
        category_id=r.category_id,
        fee_type=r.fee_type,
        fee_rate_pct=float(r.fee_rate_pct) if r.fee_rate_pct else 0,
        fixed_amount=float(r.fixed_amount) if r.fixed_amount else 0,
        min_amount=float(r.min_amount) if r.min_amount else None,
        max_amount=float(r.max_amount) if r.max_amount else None,
        currency=r.currency,
        effective_from=r.effective_from,
        effective_to=r.effective_to,
        priority=r.priority,
        status=r.status,
        remark=r.remark,
        created_at=r.created_at,
        updated_at=r.updated_at,
    )


@router.get("/rules", summary="查询费用规则列表")
async def list_rules(
    query: PlatformFeeRuleQuery = Depends(),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:view")),
):
    rules, total = await PlatformFeeService.list_rules(
        db,
        platform_id=query.platform_id,
        country_code=query.country_code,
        fee_type=query.fee_type,
        status=query.status,
        page=query.page,
        page_size=query.page_size,
    )
    return PageResult.ok(
        records=[rule_to_vo(r) for r in rules],
        total=total,
        page=query.page,
        page_size=query.page_size,
    )


@router.post("/rules", summary="创建费用规则")
async def create_rule(
    data: PlatformFeeRuleCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    rule = await PlatformFeeService.create_rule(db, data.model_dump())
    await OperationLogService.log(
        db,
        module="platform_fee",
        action="create_rule",
        resource_id=str(rule.id),
        content=f"创建费用规则: platform_id={rule.platform_id}, fee_type={rule.fee_type}",
        operator=current_user.username,
    )
    return Result.ok(rule_to_vo(rule))


@router.put("/rules/{rule_id}", summary="更新费用规则")
async def update_rule(
    rule_id: int,
    data: PlatformFeeRuleUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    rule = await PlatformFeeService.get_rule(db, rule_id)
    if not rule:
        return Result.not_found(f"费用规则不存在: id={rule_id}")

    update_data = data.model_dump(exclude_unset=True)
    if not update_data:
        return Result.bad_request("没有需要更新的字段")

    rule = await PlatformFeeService.update_rule(db, rule, update_data)
    await OperationLogService.log(
        db,
        module="platform_fee",
        action="update_rule",
        resource_id=str(rule.id),
        content=f"更新费用规则: {update_data}",
        operator=current_user.username,
    )
    return Result.ok(rule_to_vo(rule))


@router.delete("/rules/{rule_id}", summary="删除费用规则")
async def delete_rule(
    rule_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    rule = await PlatformFeeService.get_rule(db, rule_id)
    if not rule:
        return Result.not_found(f"费用规则不存在: id={rule_id}")

    await PlatformFeeService.delete_rule(db, rule)
    await OperationLogService.log(
        db,
        module="platform_fee",
        action="delete_rule",
        resource_id=str(rule_id),
        content=f"删除费用规则: platform_id={rule.platform_id}, fee_type={rule.fee_type}",
        operator=current_user.username,
    )
    return Result.ok(message="删除成功")


@router.post("/calculate", summary="计算平台费用")
async def calculate_fee(
    data: PlatformFeeCalculateRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:view")),
):
    result = await PlatformFeeService.calculate_fee(db, data)
    return Result.ok(result)
