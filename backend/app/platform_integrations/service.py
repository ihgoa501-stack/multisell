"""平台集成 - 服务层"""

from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    PlatformIntegrationAccount,
    PlatformCategoryMapping,
    PlatformAttributeMapping,
    Platform,
)
from app.platform_integrations.adapter_registry import test_connection


def _mask_credentials(credentials: dict[str, str]) -> dict:
    """将明文凭据转为脱敏元信息。"""
    masked = {}
    keys = list(credentials.keys())
    for k, v in credentials.items():
        if len(v) <= 4:
            masked[k] = v
        else:
            masked[k] = v[:4] + "..." + v[-4:]
    return {
        "keys": keys,
        "masked": masked,
        "has_credentials": True,
    }


class PlatformIntegrationService:

    # ── Accounts ─────────────────────────────────────────────────────────

    @staticmethod
    async def create_account(
        db: AsyncSession,
        data: dict,
        operator: Optional[str] = None,
    ) -> PlatformIntegrationAccount:
        credential_metadata = None
        if data.get("credentials"):
            credential_metadata = _mask_credentials(data["credentials"])

        account = PlatformIntegrationAccount(
            platform_id=data["platform_id"],
            adapter_code=data["adapter_code"],
            account_name=data["account_name"],
            credential_metadata=credential_metadata,
            created_by=operator,
            updated_by=operator,
        )
        db.add(account)
        await db.flush()
        await db.refresh(account)
        return account

    @staticmethod
    async def update_account(
        db: AsyncSession,
        account_id: int,
        data: dict,
        operator: Optional[str] = None,
    ) -> Optional[PlatformIntegrationAccount]:
        account = await db.get(PlatformIntegrationAccount, account_id)
        if not account:
            return None

        if "account_name" in data and data["account_name"] is not None:
            account.account_name = data["account_name"]
        if "status" in data and data["status"] is not None:
            account.status = data["status"]
        if "credentials" in data and data["credentials"] is not None:
            account.credential_metadata = _mask_credentials(data["credentials"])

        account.updated_by = operator
        await db.flush()
        await db.refresh(account)
        return account

    @staticmethod
    async def get_account(
        db: AsyncSession, account_id: int
    ) -> Optional[PlatformIntegrationAccount]:
        return await db.get(PlatformIntegrationAccount, account_id)

    @staticmethod
    async def list_accounts(
        db: AsyncSession, adapter_code: Optional[str] = None, status: Optional[str] = None
    ) -> list[PlatformIntegrationAccount]:
        stmt = select(PlatformIntegrationAccount).order_by(
            PlatformIntegrationAccount.created_at.desc()
        )
        if adapter_code:
            stmt = stmt.where(PlatformIntegrationAccount.adapter_code == adapter_code)
        if status:
            stmt = stmt.where(PlatformIntegrationAccount.status == status)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def test_account_connection(
        db: AsyncSession,
        account: PlatformIntegrationAccount,
    ) -> tuple[bool, str]:
        platform = await db.get(Platform, account.platform_id)
        if not platform:
            return False, "关联平台不存在"
        return await test_connection(account.adapter_code, platform)

    # ── Category Mappings ────────────────────────────────────────────────

    @staticmethod
    async def create_category_mapping(
        db: AsyncSession,
        data: dict,
        operator: Optional[str] = None,
    ) -> PlatformCategoryMapping:
        mapping = PlatformCategoryMapping(
            platform_id=data["platform_id"],
            adapter_code=data["adapter_code"],
            local_category_id=data["local_category_id"],
            platform_category_id=data["platform_category_id"],
            platform_category_name=data.get("platform_category_name"),
            platform_category_path=data.get("platform_category_path"),
            created_by=operator,
            updated_by=operator,
        )
        db.add(mapping)
        await db.flush()
        await db.refresh(mapping)
        return mapping

    @staticmethod
    async def list_category_mappings(
        db: AsyncSession,
        platform_id: Optional[int] = None,
        adapter_code: Optional[str] = None,
    ) -> list[PlatformCategoryMapping]:
        stmt = select(PlatformCategoryMapping).order_by(
            PlatformCategoryMapping.id
        )
        if platform_id:
            stmt = stmt.where(PlatformCategoryMapping.platform_id == platform_id)
        if adapter_code:
            stmt = stmt.where(PlatformCategoryMapping.adapter_code == adapter_code)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    # ── Attribute Mappings ───────────────────────────────────────────────

    @staticmethod
    async def create_attribute_mapping(
        db: AsyncSession,
        data: dict,
        operator: Optional[str] = None,
    ) -> PlatformAttributeMapping:
        mapping = PlatformAttributeMapping(
            platform_id=data["platform_id"],
            adapter_code=data["adapter_code"],
            local_attribute=data["local_attribute"],
            platform_attribute=data["platform_attribute"],
            default_value=data.get("default_value"),
            created_by=operator,
            updated_by=operator,
        )
        db.add(mapping)
        await db.flush()
        await db.refresh(mapping)
        return mapping

    @staticmethod
    async def list_attribute_mappings(
        db: AsyncSession,
        platform_id: Optional[int] = None,
        adapter_code: Optional[str] = None,
    ) -> list[PlatformAttributeMapping]:
        stmt = select(PlatformAttributeMapping).order_by(
            PlatformAttributeMapping.id
        )
        if platform_id:
            stmt = stmt.where(PlatformAttributeMapping.platform_id == platform_id)
        if adapter_code:
            stmt = stmt.where(PlatformAttributeMapping.adapter_code == adapter_code)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    # ── Helpers ──────────────────────────────────────────────────────────

    @staticmethod
    async def _resolve_names(
        db: AsyncSession,
        account: PlatformIntegrationAccount,
    ) -> dict:
        platform = await db.get(Platform, account.platform_id)
        return {
            "platform_name": platform.name if platform else None,
        }
