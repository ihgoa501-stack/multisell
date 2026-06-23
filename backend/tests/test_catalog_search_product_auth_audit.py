"""分类、品牌、商品读取、搜索的权限与审计覆盖。"""

from typing import Optional
from uuid import uuid4

import pytest
from sqlalchemy import select

from app.database import async_session_factory
from app.models import OperationLog
from tests.auth_helpers import grant_permission, register_and_login


pytestmark = pytest.mark.usefixtures("enable_auth")


async def _count_logs(module: str, action: str, resource_id: Optional[str] = None) -> int:
    async with async_session_factory() as session:
        stmt = select(OperationLog).where(
            OperationLog.module == module,
            OperationLog.action == action,
        )
        if resource_id is not None:
            stmt = stmt.where(OperationLog.resource_id == resource_id)
        result = await session.execute(stmt)
        return len(result.scalars().all())


class TestProductReadAuth:
    async def test_product_list_requires_product_view(self, async_client):
        _uid, token = await register_and_login(async_client, "prod_view_no")

        resp = await async_client.get(
            "/api/products",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403

    async def test_product_list_with_product_view_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "prod_view_ok")
        await grant_permission(uid, "product:view")

        resp = await async_client.get(
            "/api/products",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 200

    async def test_product_detail_requires_product_view_before_not_found(self, async_client):
        _uid, token = await register_and_login(async_client, "prod_detail_no")

        resp = await async_client.get(
            "/api/products/999999",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403


class TestCategoryAuthAudit:
    async def test_category_tree_requires_category_view(self, async_client):
        _uid, token = await register_and_login(async_client, "cat_view_no")

        resp = await async_client.get(
            "/api/categories/tree",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403

    async def test_category_tree_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "cat_view_ok")
        await grant_permission(uid, "category:view")

        resp = await async_client.get(
            "/api/categories/tree",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 200

    async def test_create_category_requires_category_create(self, async_client):
        _uid, token = await register_and_login(async_client, "cat_create_no")

        resp = await async_client.post(
            "/api/categories",
            json={"name": f"分类-{uuid4().hex[:6]}"},
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403

    async def test_create_category_with_permission_logs_audit(self, async_client):
        uid, token = await register_and_login(async_client, "cat_create_ok")
        await grant_permission(uid, "category:create")
        name = f"分类-{uuid4().hex[:6]}"

        resp = await async_client.post(
            "/api/categories",
            json={"name": name},
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 200
        category_id = str(resp.json()["data"]["id"])
        assert await _count_logs("category", "create", category_id) == 1

    async def test_update_category_requires_category_update(self, async_client):
        _uid, token = await register_and_login(async_client, "cat_update_no")

        resp = await async_client.put(
            "/api/categories/1",
            json={"name": "无权限更新"},
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403

    async def test_delete_category_requires_category_delete(self, async_client):
        _uid, token = await register_and_login(async_client, "cat_delete_no")

        resp = await async_client.delete(
            "/api/categories/1",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403


class TestBrandAuthAudit:
    async def test_brand_list_requires_brand_view(self, async_client):
        _uid, token = await register_and_login(async_client, "brand_view_no")

        resp = await async_client.get(
            "/api/brands",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403

    async def test_brand_list_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "brand_view_ok")
        await grant_permission(uid, "brand:view")

        resp = await async_client.get(
            "/api/brands",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 200

    async def test_create_brand_requires_brand_create(self, async_client):
        _uid, token = await register_and_login(async_client, "brand_create_no")

        resp = await async_client.post(
            "/api/brands",
            json={"name": f"品牌-{uuid4().hex[:6]}"},
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403

    async def test_create_brand_with_permission_logs_audit(self, async_client):
        uid, token = await register_and_login(async_client, "brand_create_ok")
        await grant_permission(uid, "brand:create")
        name = f"品牌-{uuid4().hex[:6]}"

        resp = await async_client.post(
            "/api/brands",
            json={"name": name},
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 200
        brand_id = str(resp.json()["data"]["id"])
        assert await _count_logs("brand", "create", brand_id) == 1

    async def test_update_brand_requires_brand_update(self, async_client):
        _uid, token = await register_and_login(async_client, "brand_update_no")

        resp = await async_client.put(
            "/api/brands/1",
            json={"name": "无权限更新"},
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403

    async def test_delete_brand_requires_brand_delete(self, async_client):
        _uid, token = await register_and_login(async_client, "brand_delete_no")

        resp = await async_client.delete(
            "/api/brands/1",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403


class TestSearchAuth:
    async def test_search_requires_search_view(self, async_client):
        _uid, token = await register_and_login(async_client, "search_no")

        resp = await async_client.get(
            "/api/search",
            params={"q": "test"},
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 403

    async def test_search_with_permission_succeeds(self, async_client):
        uid, token = await register_and_login(async_client, "search_ok")
        await grant_permission(uid, "search:view")

        resp = await async_client.get(
            "/api/search",
            params={"q": "test"},
            headers={"Authorization": f"Bearer {token}"},
        )

        assert resp.status_code == 200
