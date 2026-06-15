"""物流运费 - 路由"""

from fastapi import APIRouter, Depends, File, Query, UploadFile
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import get_current_user, require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.operation_log.service import OperationLogService
from app.shipping.schemas import (
    ProviderCreate, ProviderUpdate,
    ChannelCreate, ChannelUpdate,
    ZoneCreate, RuleCreate, RuleUpdate,
    CalculateRequest,
)
from app.shipping.service import (
    ProviderService, ChannelService, ZoneService, RuleService,
    CalculateService, ImportService,
)

router = APIRouter(tags=["物流运费"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


# ── Provider ──────────────────────────────────────────────────────────────

@router.get("/shipping/providers", summary="物流供应商列表")
async def list_providers(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:view")),
):
    providers = await ProviderService.list(db)
    return Result.ok(providers)


@router.post("/shipping/providers", summary="创建物流供应商")
async def create_provider(
    data: ProviderCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    provider = await ProviderService.create(db, data)
    await OperationLogService.log(
        db, module="shipping_provider", action="create",
        resource_id=str(provider["id"]),
        content=f"创建物流供应商: {provider['name']}",
        operator=_operator(current_user),
    )
    return Result.ok(provider)


@router.put("/shipping/providers/{provider_id}", summary="更新物流供应商")
async def update_provider(
    provider_id: int,
    data: ProviderUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    provider = await ProviderService.update(db, provider_id, data)
    if not provider:
        return Result.not_found("物流供应商不存在")
    await OperationLogService.log(
        db, module="shipping_provider", action="update",
        resource_id=str(provider_id),
        content=f"更新物流供应商: {provider['name']}",
        operator=_operator(current_user),
    )
    return Result.ok(provider)


@router.delete("/shipping/providers/{provider_id}", summary="禁用物流供应商")
async def delete_provider(
    provider_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    ok = await ProviderService.delete(db, provider_id)
    if not ok:
        return Result.not_found("物流供应商不存在")
    await OperationLogService.log(
        db, module="shipping_provider", action="delete",
        resource_id=str(provider_id),
        content=f"禁用物流供应商 ID={provider_id}",
        operator=_operator(current_user),
    )
    return Result.ok()


# ── Channel ───────────────────────────────────────────────────────────────

@router.get("/shipping/channels", summary="物流渠道列表")
async def list_channels(
    provider_id: int = Query(None, description="供应商ID筛选"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:view")),
):
    channels = await ChannelService.list(db, provider_id)
    return Result.ok(channels)


@router.post("/shipping/channels", summary="创建物流渠道")
async def create_channel(
    data: ChannelCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    channel = await ChannelService.create(db, data)
    await OperationLogService.log(
        db, module="shipping_channel", action="create",
        resource_id=str(channel["id"]),
        content=f"创建物流渠道: {channel['name']} (供应商ID={data.provider_id})",
        operator=_operator(current_user),
    )
    return Result.ok(channel)


@router.put("/shipping/channels/{channel_id}", summary="更新物流渠道")
async def update_channel(
    channel_id: int,
    data: ChannelUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    channel = await ChannelService.update(db, channel_id, data)
    if not channel:
        return Result.not_found("物流渠道不存在")
    await OperationLogService.log(
        db, module="shipping_channel", action="update",
        resource_id=str(channel_id),
        content=f"更新物流渠道: {channel['name']}",
        operator=_operator(current_user),
    )
    return Result.ok(channel)


@router.delete("/shipping/channels/{channel_id}", summary="禁用物流渠道")
async def delete_channel(
    channel_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    ok = await ChannelService.delete(db, channel_id)
    if not ok:
        return Result.not_found("物流渠道不存在")
    await OperationLogService.log(
        db, module="shipping_channel", action="delete",
        resource_id=str(channel_id),
        content=f"禁用物流渠道 ID={channel_id}",
        operator=_operator(current_user),
    )
    return Result.ok()


# ── Zone ──────────────────────────────────────────────────────────────────

@router.get("/shipping/channels/{channel_id}/zones", summary="物流区域列表")
async def list_zones(
    channel_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:view")),
):
    zones = await ZoneService.list_by_channel(db, channel_id)
    return Result.ok(zones)


@router.post("/shipping/channels/{channel_id}/zones", summary="创建物流区域")
async def create_zone(
    channel_id: int,
    data: ZoneCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    zone = await ZoneService.create(db, channel_id, data)
    await OperationLogService.log(
        db, module="shipping_zone", action="create",
        resource_id=str(zone["id"]),
        content=f"创建物流区域: {zone['country_code']} (渠道ID={channel_id})",
        operator=_operator(current_user),
    )
    return Result.ok(zone)


@router.delete("/shipping/zones/{zone_id}", summary="删除物流区域")
async def delete_zone(
    zone_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    ok = await ZoneService.delete(db, zone_id)
    if not ok:
        return Result.not_found("物流区域不存在")
    await OperationLogService.log(
        db, module="shipping_zone", action="delete",
        resource_id=str(zone_id),
        content=f"删除物流区域 ID={zone_id}",
        operator=_operator(current_user),
    )
    return Result.ok()


# ── Rules ─────────────────────────────────────────────────────────────────

@router.get("/shipping/channels/{channel_id}/rules", summary="报价规则列表")
async def list_rules(
    channel_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:view")),
):
    rules = await RuleService.list_by_channel(db, channel_id)
    return Result.ok(rules)


@router.post("/shipping/channels/{channel_id}/rules", summary="创建报价规则")
async def create_rule(
    channel_id: int,
    data: RuleCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    rule = await RuleService.create(db, channel_id, data)
    await OperationLogService.log(
        db, module="shipping_quote_rule", action="create",
        resource_id=str(rule["id"]),
        content=f"创建报价规则: {rule['rule_type']} (渠道ID={channel_id})",
        operator=_operator(current_user),
    )
    return Result.ok(rule)


@router.put("/shipping/rules/{rule_id}", summary="更新报价规则")
async def update_rule(
    rule_id: int,
    data: RuleUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    rule = await RuleService.update(db, rule_id, data)
    if not rule:
        return Result.not_found("报价规则不存在")
    await OperationLogService.log(
        db, module="shipping_quote_rule", action="update",
        resource_id=str(rule_id),
        content=f"更新报价规则 ID={rule_id}",
        operator=_operator(current_user),
    )
    return Result.ok(rule)


@router.delete("/shipping/rules/{rule_id}", summary="删除报价规则")
async def delete_rule(
    rule_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    ok = await RuleService.delete(db, rule_id)
    if not ok:
        return Result.not_found("报价规则不存在")
    await OperationLogService.log(
        db, module="shipping_quote_rule", action="delete",
        resource_id=str(rule_id),
        content=f"删除报价规则 ID={rule_id}",
        operator=_operator(current_user),
    )
    return Result.ok()


# ── Import ────────────────────────────────────────────────────────────────

@router.post("/shipping/import-rules", summary="导入物流报价表")
async def import_shipping_rules(
    file: UploadFile = File(...),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:manage")),
):
    try:
        content = await file.read()
        result = await ImportService.import_rules(db, file.filename or "", content)
    except ValueError as e:
        return Result.bad_request(str(e))
    await OperationLogService.log(
        db,
        module="shipping_quote_rule",
        action="import",
        resource_id=None,
        content=f"导入物流报价表: {file.filename}, 成功{result['imported_rows']}行, 错误{result['error_rows']}行",
        operator=_operator(current_user),
    )
    return Result.ok(result)


# ── Calculate ─────────────────────────────────────────────────────────────

@router.post("/shipping/calculate", summary="运费计算")
async def calculate_shipping(
    data: CalculateRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("shipping:calculate")),
):
    try:
        result = await CalculateService.calculate(db, data)
        return Result.ok(result.model_dump())
    except ValueError as e:
        return Result.bad_request(str(e))
