"""平台费用规则 API、匹配、权限与审计测试。"""

import pytest
from uuid import uuid4

from sqlalchemy import select

from app.database import async_session_factory
from app.models import OperationLog, PlatformFeeRule
from tests.auth_helpers import enable_auth, grant_permission, register_and_login


pytestmark = pytest.mark.usefixtures("enable_auth")


def _code(prefix: str) -> str:
    return f"{prefix}_{uuid4().hex[:6]}"


async def _auth(async_client, username_prefix: str, permission: str | None = None):
    uid, token = await register_and_login(async_client, username_prefix)
    if permission:
        await grant_permission(uid, permission)
    return {"Authorization": f"Bearer {token}"}


async def _create_platform(async_client, code_prefix: str = "pf"):
    headers = await _auth(async_client, f"{code_prefix}_platform", "platform:create")
    payload = {
        "name": f"平台-{uuid4().hex[:6]}",
        "code": _code(code_prefix),
        "api_base_url": "https://example.com",
        "status": 1,
    }
    resp = await async_client.post("/api/platforms", json=payload, headers=headers)
    assert resp.status_code == 200
    return resp.json()["data"]["id"]


async def _count_logs(module: str, action: str, resource_id: str) -> int:
    async with async_session_factory() as session:
        result = await session.execute(
            select(OperationLog).where(
                OperationLog.module == module,
                OperationLog.action == action,
                OperationLog.resource_id == resource_id,
            )
        )
        return len(result.scalars().all())


class TestPlatformFeeModel:
    async def test_platform_fee_rule_model_is_mapped(self):
        async with async_session_factory() as session:
            from app.models import Platform
            # Create a valid platform for the FK
            platform = Platform(
                name="model-test-platform",
                code=f"mtp_{uuid4().hex[:6]}",
                api_base_url="https://example.com",
                status=1,
            )
            session.add(platform)
            await session.flush()
            platform_id = platform.id

            rule = PlatformFeeRule(
                platform_id=platform_id,
                site_code="RU",
                category_id=None,
                commission_pct=12,
                payment_fee_pct=3,
                fixed_fee=5,
                advertising_pct=2,
                other_reserve_fee=1,
                priority=0,
                status=1,
                remark="model smoke test",
            )
            session.add(rule)
            await session.flush()
            assert rule.id is not None
            await session.rollback()
