"""平台集成 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.database import get_db
from app.common import Result
from app.models import User, Platform, Category
from app.platform_integrations.adapter_registry import list_adapters
from app.platform_integrations.schemas import (
    AdapterCapabilityResponse,
    PlatformIntegrationAccountCreate,
    PlatformIntegrationAccountUpdate,
    PlatformIntegrationAccountResponse,
    PlatformCategoryMappingCreate,
    PlatformCategoryMappingResponse,
    PlatformAttributeMappingCreate,
    PlatformAttributeMappingResponse,
    TestConnectionResponse,
)
from app.platform_integrations.service import PlatformIntegrationService
from app.operation_log.service import OperationLogService

router = APIRouter(tags=["平台集成"])


# ── Adapter 注册表 ─────────────────────────────────────────────────────


@router.get("/platform-integrations/adapters", summary="列出所有平台适配器")
async def get_adapters(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:view")),
):
    adapters = list_adapters()
    return Result.ok([
        AdapterCapabilityResponse(
            adapter_code=a.adapter_code,
            display_name=a.display_name,
            supports_listing_publish=a.supports_listing_publish,
            supports_order_import=a.supports_order_import,
            supports_settlement_import=a.supports_settlement_import,
            supports_tracking_sync=a.supports_tracking_sync,
            auth_type=a.auth_type,
        )
        for a in adapters
    ])


# ── 平台账号 ───────────────────────────────────────────────────────────


@router.post("/platform-integrations/accounts", summary="创建平台集成账号")
async def create_account(
    data: PlatformIntegrationAccountCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:manage")),
):
    platform = await db.get(Platform, data.platform_id)
    if not platform:
        return Result.not_found("平台不存在")

    account = await PlatformIntegrationService.create_account(
        db, data.model_dump(), operator=current_user.username,
    )
    platform_name = platform.name if platform else None

    await OperationLogService.log(
        db,
        module="platform_integration",
        action="create_account",
        resource_id=str(account.id),
        content=f"创建平台集成账号: {account.account_name} (adapter={account.adapter_code})",
        operator=current_user.username,
    )

    return Result.ok(PlatformIntegrationAccountResponse(
        id=account.id,
        platform_id=account.platform_id,
        platform_name=platform_name,
        adapter_code=account.adapter_code,
        account_name=account.account_name,
        status=account.status,
        credential_metadata=account.credential_metadata,
        created_by=account.created_by,
        updated_by=account.updated_by,
        created_at=account.created_at,
        updated_at=account.updated_at,
    ))


@router.get("/platform-integrations/accounts", summary="列出平台集成账号")
async def list_accounts(
    adapter_code: str | None = None,
    status: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:view")),
):
    accounts = await PlatformIntegrationService.list_accounts(
        db, adapter_code=adapter_code, status=status,
    )
    items = []
    for acc in accounts:
        platform = await db.get(Platform, acc.platform_id)
        items.append(PlatformIntegrationAccountResponse(
            id=acc.id,
            platform_id=acc.platform_id,
            platform_name=platform.name if platform else None,
            adapter_code=acc.adapter_code,
            account_name=acc.account_name,
            status=acc.status,
            credential_metadata=acc.credential_metadata,
            created_by=acc.created_by,
            updated_by=acc.updated_by,
            created_at=acc.created_at,
            updated_at=acc.updated_at,
        ))
    return Result.ok(items)


@router.get("/platform-integrations/accounts/{account_id}", summary="查询平台集成账号详情")
async def get_account(
    account_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:view")),
):
    account = await PlatformIntegrationService.get_account(db, account_id)
    if not account:
        return Result.not_found("账号不存在")
    platform = await db.get(Platform, account.platform_id)
    return Result.ok(PlatformIntegrationAccountResponse(
        id=account.id,
        platform_id=account.platform_id,
        platform_name=platform.name if platform else None,
        adapter_code=account.adapter_code,
        account_name=account.account_name,
        status=account.status,
        credential_metadata=account.credential_metadata,
        created_by=account.created_by,
        updated_by=account.updated_by,
        created_at=account.created_at,
        updated_at=account.updated_at,
    ))


@router.put("/platform-integrations/accounts/{account_id}", summary="更新平台集成账号")
async def update_account(
    account_id: int,
    data: PlatformIntegrationAccountUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:manage")),
):
    account = await PlatformIntegrationService.update_account(
        db, account_id, data.model_dump(exclude_unset=True), operator=current_user.username,
    )
    if not account:
        return Result.not_found("账号不存在")

    await OperationLogService.log(
        db,
        module="platform_integration",
        action="update_account",
        resource_id=str(account.id),
        content=f"更新平台集成账号: {account.account_name} (status={account.status})",
        operator=current_user.username,
    )

    platform = await db.get(Platform, account.platform_id)
    return Result.ok(PlatformIntegrationAccountResponse(
        id=account.id,
        platform_id=account.platform_id,
        platform_name=platform.name if platform else None,
        adapter_code=account.adapter_code,
        account_name=account.account_name,
        status=account.status,
        credential_metadata=account.credential_metadata,
        created_by=account.created_by,
        updated_by=account.updated_by,
        created_at=account.created_at,
        updated_at=account.updated_at,
    ))


@router.post("/platform-integrations/accounts/{account_id}/test", summary="测试平台连接")
async def test_account(
    account_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:test")),
):
    account = await PlatformIntegrationService.get_account(db, account_id)
    if not account:
        return Result.not_found("账号不存在")

    success, message = await PlatformIntegrationService.test_account_connection(db, account)

    await OperationLogService.log(
        db,
        module="platform_integration",
        action="test_account",
        resource_id=str(account.id),
        content=f"测试平台连接: {account.account_name} -> {message}",
        operator=current_user.username,
    )

    return Result.ok(TestConnectionResponse(success=success, message=message))


# ── 类目映射 ───────────────────────────────────────────────────────────


@router.post("/platform-integrations/category-mappings", summary="创建平台类目映射")
async def create_category_mapping(
    data: PlatformCategoryMappingCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:manage")),
):
    platform = await db.get(Platform, data.platform_id)
    if not platform:
        return Result.not_found("平台不存在")
    category = await db.get(Category, data.local_category_id)
    if not category:
        return Result.not_found("本地类目不存在")

    mapping = await PlatformIntegrationService.create_category_mapping(
        db, data.model_dump(), operator=current_user.username,
    )

    await OperationLogService.log(
        db,
        module="platform_integration",
        action="save_mapping",
        resource_id=f"category_mapping:{mapping.id}",
        content=f"保存平台类目映射: {category.name} -> {data.platform_category_name or data.platform_category_id}",
        operator=current_user.username,
    )

    return Result.ok(PlatformCategoryMappingResponse(
        id=mapping.id,
        platform_id=mapping.platform_id,
        platform_name=platform.name,
        adapter_code=mapping.adapter_code,
        local_category_id=mapping.local_category_id,
        local_category_name=category.name,
        platform_category_id=mapping.platform_category_id,
        platform_category_name=mapping.platform_category_name,
        platform_category_path=mapping.platform_category_path,
        created_by=mapping.created_by,
        created_at=mapping.created_at,
    ))


@router.get("/platform-integrations/category-mappings", summary="列出平台类目映射")
async def list_category_mappings(
    platform_id: int | None = None,
    adapter_code: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:view")),
):
    mappings = await PlatformIntegrationService.list_category_mappings(
        db, platform_id=platform_id, adapter_code=adapter_code,
    )
    items = []
    for m in mappings:
        platform = await db.get(Platform, m.platform_id)
        category = await db.get(Category, m.local_category_id)
        items.append(PlatformCategoryMappingResponse(
            id=m.id,
            platform_id=m.platform_id,
            platform_name=platform.name if platform else None,
            adapter_code=m.adapter_code,
            local_category_id=m.local_category_id,
            local_category_name=category.name if category else None,
            platform_category_id=m.platform_category_id,
            platform_category_name=m.platform_category_name,
            platform_category_path=m.platform_category_path,
            created_by=m.created_by,
            created_at=m.created_at,
        ))
    return Result.ok(items)


# ── 属性映射 ───────────────────────────────────────────────────────────


@router.post("/platform-integrations/attribute-mappings", summary="创建平台属性映射")
async def create_attribute_mapping(
    data: PlatformAttributeMappingCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:manage")),
):
    platform = await db.get(Platform, data.platform_id)
    if not platform:
        return Result.not_found("平台不存在")

    mapping = await PlatformIntegrationService.create_attribute_mapping(
        db, data.model_dump(), operator=current_user.username,
    )

    await OperationLogService.log(
        db,
        module="platform_integration",
        action="save_mapping",
        resource_id=f"attribute_mapping:{mapping.id}",
        content=f"保存平台属性映射: {data.local_attribute} -> {data.platform_attribute}",
        operator=current_user.username,
    )

    return Result.ok(PlatformAttributeMappingResponse(
        id=mapping.id,
        platform_id=mapping.platform_id,
        platform_name=platform.name,
        adapter_code=mapping.adapter_code,
        local_attribute=mapping.local_attribute,
        platform_attribute=mapping.platform_attribute,
        default_value=mapping.default_value,
        created_by=mapping.created_by,
        created_at=mapping.created_at,
    ))


@router.get("/platform-integrations/attribute-mappings", summary="列出平台属性映射")
async def list_attribute_mappings(
    platform_id: int | None = None,
    adapter_code: str | None = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_integration:view")),
):
    mappings = await PlatformIntegrationService.list_attribute_mappings(
        db, platform_id=platform_id, adapter_code=adapter_code,
    )
    items = []
    for m in mappings:
        platform = await db.get(Platform, m.platform_id)
        items.append(PlatformAttributeMappingResponse(
            id=m.id,
            platform_id=m.platform_id,
            platform_name=platform.name if platform else None,
            adapter_code=m.adapter_code,
            local_attribute=m.local_attribute,
            platform_attribute=m.platform_attribute,
            default_value=m.default_value,
            created_by=m.created_by,
            created_at=m.created_at,
        ))
    return Result.ok(items)
