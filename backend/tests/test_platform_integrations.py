"""Stage 10A — Platform Adapter Core Tests"""

from uuid import uuid4

import pytest
from sqlalchemy import select

from app.database import async_session_factory
from app.models import (
    PlatformIntegrationAccount,
    PlatformCategoryMapping,
    PlatformAttributeMapping,
    OperationLog,
    Platform,
    Category,
)
from tests.auth_helpers import register_and_login, grant_permission


def _code(prefix: str = "test") -> str:
    return f"{prefix}_{uuid4().hex[:8]}"


async def _create_platform(session) -> Platform:
    code = _code("plat")
    p = Platform(name=code, code=code, status=1)
    session.add(p)
    await session.flush()
    return p


async def _create_category(session) -> Category:
    name = _code("cat")
    c = Category(name=name, status=1)
    session.add(c)
    await session.flush()
    return c


# ── Adapter Registry ───────────────────────────────────────────────────


class TestAdapterRegistry:
    async def test_adapter_registry_returns_mock_adapters(self, async_client):
        resp = await async_client.get("/api/platform-integrations/adapters")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        adapters = data["data"]
        assert isinstance(adapters, list)
        assert len(adapters) >= 5

        codes = {a["adapter_code"] for a in adapters}
        assert "amazon" in codes
        assert "tiktok" in codes
        assert "temu" in codes
        assert "shopify" in codes

        amazon = next(a for a in adapters if a["adapter_code"] == "amazon")
        assert amazon["display_name"] == "Amazon"
        assert amazon["supports_listing_publish"] is True
        assert amazon["supports_order_import"] is True
        assert amazon["supports_settlement_import"] is True
        assert amazon["supports_tracking_sync"] is True
        assert amazon["auth_type"] == "oauth2"

    async def test_adapter_detail_fields(self, async_client):
        resp = await async_client.get("/api/platform-integrations/adapters")
        data = resp.json()["data"]
        for a in data:
            assert "adapter_code" in a
            assert "display_name" in a
            assert "supports_listing_publish" in a
            assert "supports_order_import" in a
            assert "supports_settlement_import" in a
            assert "supports_tracking_sync" in a
            assert "auth_type" in a


# ── Accounts CRUD ──────────────────────────────────────────────────────


class TestAccountsCRUD:
    async def test_create_account_success(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "My Amazon Store",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        account = data["data"]
        assert account["account_name"] == "My Amazon Store"
        assert account["adapter_code"] == "amazon"
        assert account["status"] == "draft"
        assert account["id"] > 0

    async def test_create_account_does_not_return_secrets(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Secret Store",
                "credentials": {
                    "api_key": "AKIA1234567890EXAMPLE",
                    "api_secret": "supersecretkey12345",
                },
            },
        )
        assert resp.status_code == 200
        data = resp.json()["data"]

        # credential_metadata should NOT contain plaintext secrets
        meta = data["credential_metadata"]
        assert meta["has_credentials"] is True
        assert "keys" in meta
        assert "api_key" in meta["keys"]
        assert "api_secret" in meta["keys"]

        # Plaintext values should NOT appear in metadata
        assert "AKIA1234567890EXAMPLE" not in str(meta["masked"].values())
        assert "supersecretkey12345" not in str(meta["masked"].values())

        # Masked values should be truncated with ...
        assert "..." in meta["masked"]["api_key"]
        assert "..." in meta["masked"]["api_secret"]

        # The response should NOT contain any "credentials" field at the top level
        assert "credentials" not in data

    async def test_create_account_without_credentials(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "No Creds",
            },
        )
        assert resp.status_code == 200
        meta = resp.json()["data"]["credential_metadata"]
        assert meta is None

    async def test_list_accounts(self, async_client):
        async with async_session_factory() as session:
            p1 = await _create_platform(session)
            p2 = await _create_platform(session)
            await session.commit()

        await async_client.post(
            "/api/platform-integrations/accounts",
            json={"platform_id": p1.id, "adapter_code": "amazon", "account_name": "Acc1"},
        )
        await async_client.post(
            "/api/platform-integrations/accounts",
            json={"platform_id": p2.id, "adapter_code": "tiktok", "account_name": "Acc2"},
        )

        resp = await async_client.get("/api/platform-integrations/accounts")
        assert resp.status_code == 200
        items = resp.json()["data"]
        names = {i["account_name"] for i in items}
        assert "Acc1" in names
        assert "Acc2" in names

    async def test_list_accounts_filter_by_adapter(self, async_client):
        async with async_session_factory() as session:
            p = await _create_platform(session)
            await session.commit()

        ac1 = _code("acct")
        ac2 = _code("acct")
        await async_client.post(
            "/api/platform-integrations/accounts",
            json={"platform_id": p.id, "adapter_code": ac1, "account_name": "FilterA"},
        )
        await async_client.post(
            "/api/platform-integrations/accounts",
            json={"platform_id": p.id, "adapter_code": ac2, "account_name": "FilterB"},
        )

        resp = await async_client.get(
            f"/api/platform-integrations/accounts?adapter_code={ac1}"
        )
        items = resp.json()["data"]
        assert len(items) == 1
        assert items[0]["adapter_code"] == ac1

    async def test_get_account_by_id(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        create_resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "shopify",
                "account_name": "My Shop",
            },
        )
        account_id = create_resp.json()["data"]["id"]

        resp = await async_client.get(
            f"/api/platform-integrations/accounts/{account_id}"
        )
        assert resp.status_code == 200
        account = resp.json()["data"]
        assert account["account_name"] == "My Shop"
        assert account["adapter_code"] == "shopify"
        assert account["id"] == account_id

    async def test_get_account_not_found(self, async_client):
        resp = await async_client.get("/api/platform-integrations/accounts/99999")
        assert resp.status_code == 200
        assert resp.json()["code"] == 404

    async def test_update_account_name_and_status(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        create_resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Old Name",
            },
        )
        account_id = create_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/platform-integrations/accounts/{account_id}",
            json={
                "account_name": "New Name",
                "status": "active",
            },
        )
        assert resp.status_code == 200
        account = resp.json()["data"]
        assert account["account_name"] == "New Name"
        assert account["status"] == "active"

    async def test_update_account_not_found(self, async_client):
        resp = await async_client.put(
            "/api/platform-integrations/accounts/99999",
            json={"account_name": "Nope"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 404

    async def test_update_credentials_masks_them(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        create_resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Cred Update",
            },
        )
        account_id = create_resp.json()["data"]["id"]

        resp = await async_client.put(
            f"/api/platform-integrations/accounts/{account_id}",
            json={"credentials": {"new_key": "new_secret_value_123"}},
        )
        meta = resp.json()["data"]["credential_metadata"]
        assert "new_key" in meta["keys"]
        assert "new_secret_value_123" not in str(meta["masked"].values())
        assert "..." in meta["masked"]["new_key"]

    async def test_platform_not_found_on_create(self, async_client):
        resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": 99999,
                "adapter_code": "amazon",
                "account_name": "No Platform",
            },
        )
        assert resp.json()["code"] == 404


# ── Test Connection ────────────────────────────────────────────────────


class TestAccountConnection:
    async def test_test_account_returns_success(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        create_resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Test Conn",
                "credentials": {"api_key": "test"},
            },
        )
        account_id = create_resp.json()["data"]["id"]

        resp = await async_client.post(
            f"/api/platform-integrations/accounts/{account_id}/test"
        )
        assert resp.status_code == 200
        result = resp.json()["data"]
        assert result["success"] is True
        assert "mock" in result["message"].lower()

    async def test_test_account_not_found(self, async_client):
        resp = await async_client.post(
            "/api/platform-integrations/accounts/99999/test"
        )
        assert resp.json()["code"] == 404

    async def test_test_account_unknown_adapter_fails(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        create_resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "nonexistent_adapter",
                "account_name": "Fake",
            },
        )
        account_id = create_resp.json()["data"]["id"]

        resp = await async_client.post(
            f"/api/platform-integrations/accounts/{account_id}/test"
        )
        result = resp.json()["data"]
        assert result["success"] is False
        assert "未知适配器" in result["message"]


# ── Category Mappings ──────────────────────────────────────────────────


class TestCategoryMappings:
    async def test_create_category_mapping(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            category = await _create_category(session)
            await session.commit()

        resp = await async_client.post(
            "/api/platform-integrations/category-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "local_category_id": category.id,
                "platform_category_id": "amz_cat_123",
                "platform_category_name": "Electronics",
                "platform_category_path": "Electronics > Audio",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        mapping = data["data"]
        assert mapping["platform_category_id"] == "amz_cat_123"
        assert mapping["platform_category_name"] == "Electronics"
        assert mapping["local_category_id"] == category.id

    async def test_create_category_mapping_platform_not_found(self, async_client):
        async with async_session_factory() as session:
            category = await _create_category(session)
            await session.commit()

        resp = await async_client.post(
            "/api/platform-integrations/category-mappings",
            json={
                "platform_id": 99999,
                "adapter_code": "amazon",
                "local_category_id": category.id,
                "platform_category_id": "c1",
            },
        )
        assert resp.json()["code"] == 404

    async def test_list_category_mappings(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            cat1 = await _create_category(session)
            cat2 = await _create_category(session)
            await session.commit()

        await async_client.post(
            "/api/platform-integrations/category-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "local_category_id": cat1.id,
                "platform_category_id": "amz_1",
            },
        )
        await async_client.post(
            "/api/platform-integrations/category-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": "tiktok",
                "local_category_id": cat2.id,
                "platform_category_id": "tt_1",
            },
        )

        resp = await async_client.get(
            f"/api/platform-integrations/category-mappings?platform_id={platform.id}"
        )
        items = resp.json()["data"]
        assert len(items) == 2

    async def test_list_category_mappings_filter_by_adapter(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            cat = await _create_category(session)
            await session.commit()

        ac1 = _code("cmap")
        ac2 = _code("cmap")
        await async_client.post(
            "/api/platform-integrations/category-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": ac1,
                "local_category_id": cat.id,
                "platform_category_id": "fp_1",
            },
        )
        await async_client.post(
            "/api/platform-integrations/category-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": ac2,
                "local_category_id": cat.id,
                "platform_category_id": "fp_2",
            },
        )

        resp = await async_client.get(
            f"/api/platform-integrations/category-mappings?adapter_code={ac1}"
        )
        items = resp.json()["data"]
        assert len(items) == 1
        assert items[0]["adapter_code"] == ac1


# ── Attribute Mappings ─────────────────────────────────────────────────


class TestAttributeMappings:
    async def test_create_attribute_mapping(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        resp = await async_client.post(
            "/api/platform-integrations/attribute-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "local_attribute": "brand",
                "platform_attribute": "brand_name",
                "default_value": "My Brand",
            },
        )
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["local_attribute"] == "brand"
        assert data["platform_attribute"] == "brand_name"
        assert data["default_value"] == "My Brand"

    async def test_create_attribute_mapping_platform_not_found(self, async_client):
        resp = await async_client.post(
            "/api/platform-integrations/attribute-mappings",
            json={
                "platform_id": 99999,
                "adapter_code": "amazon",
                "local_attribute": "brand",
                "platform_attribute": "brand_name",
            },
        )
        assert resp.json()["code"] == 404

    async def test_list_attribute_mappings(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        await async_client.post(
            "/api/platform-integrations/attribute-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "local_attribute": "brand",
                "platform_attribute": "brand_name",
            },
        )
        await async_client.post(
            "/api/platform-integrations/attribute-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": "tiktok",
                "local_attribute": "color",
                "platform_attribute": "colour",
            },
        )

        resp = await async_client.get(
            f"/api/platform-integrations/attribute-mappings?platform_id={platform.id}"
        )
        items = resp.json()["data"]
        assert len(items) == 2

    async def test_list_attribute_mappings_filter_by_adapter(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        ac1 = _code("amap")
        ac2 = _code("amap")
        await async_client.post(
            "/api/platform-integrations/attribute-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": ac1,
                "local_attribute": "brand",
                "platform_attribute": "brand_name",
            },
        )
        await async_client.post(
            "/api/platform-integrations/attribute-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": ac2,
                "local_attribute": "color",
                "platform_attribute": "colour",
            },
        )

        resp = await async_client.get(
            f"/api/platform-integrations/attribute-mappings?adapter_code={ac1}"
        )
        items = resp.json()["data"]
        assert len(items) == 1
        assert items[0]["adapter_code"] == ac1


# ── Audit ──────────────────────────────────────────────────────────────


class TestAuditLog:
    async def _count_logs(self, module: str, action: str, resource_id: str = None) -> int:
        async with async_session_factory() as session:
            stmt = select(OperationLog).where(
                OperationLog.module == module,
                OperationLog.action == action,
            )
            if resource_id:
                stmt = stmt.where(OperationLog.resource_id == resource_id)
            result = await session.execute(stmt)
            return len(result.all())

    async def test_create_account_audit(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Audit Test",
            },
        )
        count = await self._count_logs("platform_integration", "create_account")
        assert count > 0

    async def test_update_account_audit(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        create_resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Audit Update",
            },
        )
        account_id = create_resp.json()["data"]["id"]

        await async_client.put(
            f"/api/platform-integrations/accounts/{account_id}",
            json={"status": "active"},
        )
        count = await self._count_logs("platform_integration", "update_account")
        assert count > 0

    async def test_test_account_audit(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        create_resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Audit Test Conn",
                "credentials": {"key": "val"},
            },
        )
        account_id = create_resp.json()["data"]["id"]

        await async_client.post(
            f"/api/platform-integrations/accounts/{account_id}/test"
        )
        count = await self._count_logs("platform_integration", "test_account")
        assert count > 0

    async def test_save_category_mapping_audit(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            category = await _create_category(session)
            await session.commit()

        await async_client.post(
            "/api/platform-integrations/category-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "local_category_id": category.id,
                "platform_category_id": "audit_cat",
            },
        )
        count = await self._count_logs("platform_integration", "save_mapping")
        assert count > 0

    async def test_save_attribute_mapping_audit(self, async_client):
        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        await async_client.post(
            "/api/platform-integrations/attribute-mappings",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "local_attribute": "size",
                "platform_attribute": "size_name",
            },
        )
        count = await self._count_logs("platform_integration", "save_mapping")
        assert count > 0


# ── Permissions ────────────────────────────────────────────────────────


class TestPermissions:
    @pytest.fixture(autouse=True)
    def _enable_auth(self):
        from app.config import settings
        original = settings.AUTH_ENABLED
        settings.AUTH_ENABLED = True
        yield
        settings.AUTH_ENABLED = original

    async def test_view_adapters_requires_view_permission(self, async_client):
        user_id, token = await register_and_login(async_client, "pi_view")
        await grant_permission(user_id, "platform_integration:view")

        resp = await async_client.get(
            "/api/platform-integrations/adapters",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200

    async def test_view_adapters_denied_without_permission(self, async_client):
        _uid, token = await register_and_login(async_client, "pi_no_perm")

        resp = await async_client.get(
            "/api/platform-integrations/adapters",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_create_account_requires_manage_permission(self, async_client):
        user_id, token = await register_and_login(async_client, "pi_mgmt")
        await grant_permission(user_id, "platform_integration:manage")

        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Perm Test",
            },
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200

    async def test_create_account_denied_without_manage(self, async_client):
        _uid, token = await register_and_login(async_client, "pi_no_mgmt")

        resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={"platform_id": 1, "adapter_code": "amazon", "account_name": "Nope"},
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_test_account_requires_test_permission(self, async_client):
        user_id, token = await register_and_login(async_client, "pi_test")
        await grant_permission(user_id, "platform_integration:test")
        await grant_permission(user_id, "platform_integration:manage")

        async with async_session_factory() as session:
            platform = await _create_platform(session)
            await session.commit()

        create_resp = await async_client.post(
            "/api/platform-integrations/accounts",
            json={
                "platform_id": platform.id,
                "adapter_code": "amazon",
                "account_name": "Perm Test Conn",
            },
            headers={"Authorization": f"Bearer {token}"},
        )
        account_id = create_resp.json()["data"]["id"]

        resp = await async_client.post(
            f"/api/platform-integrations/accounts/{account_id}/test",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200

    async def test_test_account_denied_without_permission(self, async_client):
        _uid, token = await register_and_login(async_client, "pi_no_test")

        resp = await async_client.post(
            "/api/platform-integrations/accounts/1/test",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403

    async def test_list_accounts_requires_view(self, async_client):
        user_id, token = await register_and_login(async_client, "pi_list")
        await grant_permission(user_id, "platform_integration:view")

        resp = await async_client.get(
            "/api/platform-integrations/accounts",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 200

    async def test_list_accounts_denied_without_view(self, async_client):
        _uid, token = await register_and_login(async_client, "pi_no_list")

        resp = await async_client.get(
            "/api/platform-integrations/accounts",
            headers={"Authorization": f"Bearer {token}"},
        )
        assert resp.status_code == 403
