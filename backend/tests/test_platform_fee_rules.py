"""平台费用规则 API、匹配、权限与审计测试。"""

import pytest
from uuid import uuid4

from sqlalchemy import select

from app.database import async_session_factory
from app.models import OperationLog, PlatformFeeRule, Platform
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
    async def test_platform_fee_rule_model_is_mapped(self, async_client):
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


class TestPlatformFeeRuleService:
    async def test_match_prefers_category_rule_over_site_and_global(self, async_client):
        from app.platform_fee.schemas import PlatformFeeRuleCreate, PlatformFeeRuleMatchRequest
        from app.platform_fee.service import PlatformFeeRuleService

        platform_id = await _create_platform(async_client, "pf_match")
        async with async_session_factory() as session:
            from app.models import Category
            cat = Category(name="test-match-cat", status=1)
            session.add(cat)
            await session.flush()
            category_id = cat.id

            await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id,
                site_code=None,
                category_id=None,
                commission_pct=5,
                payment_fee_pct=1,
                fixed_fee=1,
                advertising_pct=0,
                other_reserve_fee=0,
                priority=0,
                remark="global",
            ))
            await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id,
                site_code="RU",
                category_id=None,
                commission_pct=10,
                payment_fee_pct=2,
                fixed_fee=2,
                advertising_pct=1,
                other_reserve_fee=0,
                priority=0,
                remark="site",
            ))
            category_rule = await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id,
                site_code="RU",
                category_id=category_id,
                commission_pct=15,
                payment_fee_pct=3,
                fixed_fee=3,
                advertising_pct=2,
                other_reserve_fee=1,
                priority=0,
                remark="category",
            ))
            await session.commit()

        async with async_session_factory() as session:
            matched = await PlatformFeeRuleService.match(
                session,
                PlatformFeeRuleMatchRequest(
                    platform_id=platform_id,
                    site_code="RU",
                    category_id=category_id,
                ),
            )
        assert matched is not None
        assert matched["id"] == category_rule["id"]
        assert matched["commission_pct"] == 15

    async def test_match_falls_back_to_site_rule(self, async_client):
        from app.platform_fee.schemas import PlatformFeeRuleCreate, PlatformFeeRuleMatchRequest
        from app.platform_fee.service import PlatformFeeRuleService

        platform_id = await _create_platform(async_client, "pf_site")
        async with async_session_factory() as session:
            await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id, site_code=None, category_id=None,
                commission_pct=5, payment_fee_pct=1, fixed_fee=1,
                advertising_pct=0, other_reserve_fee=0, priority=0, remark="global",
            ))
            site_rule = await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id, site_code="RU", category_id=None,
                commission_pct=10, payment_fee_pct=2, fixed_fee=2,
                advertising_pct=1, other_reserve_fee=0, priority=0, remark="site",
            ))
            await session.commit()

        async with async_session_factory() as session:
            matched = await PlatformFeeRuleService.match(
                session,
                PlatformFeeRuleMatchRequest(platform_id=platform_id, site_code="RU"),
            )
        assert matched["id"] == site_rule["id"]

    async def test_match_falls_back_to_global_rule(self, async_client):
        from app.platform_fee.schemas import PlatformFeeRuleCreate, PlatformFeeRuleMatchRequest
        from app.platform_fee.service import PlatformFeeRuleService

        platform_id = await _create_platform(async_client, "pf_global")
        async with async_session_factory() as session:
            global_rule = await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id, site_code=None, category_id=None,
                commission_pct=5, payment_fee_pct=1, fixed_fee=1,
                advertising_pct=0, other_reserve_fee=0, priority=0, remark="global",
            ))
            await session.commit()

        async with async_session_factory() as session:
            matched = await PlatformFeeRuleService.match(
                session,
                PlatformFeeRuleMatchRequest(platform_id=platform_id, site_code="BR"),
            )
        assert matched["id"] == global_rule["id"]

    async def test_match_returns_none_when_no_rule_matches(self, async_client):
        from app.platform_fee.schemas import PlatformFeeRuleCreate, PlatformFeeRuleMatchRequest
        from app.platform_fee.service import PlatformFeeRuleService

        platform_id = await _create_platform(async_client, "pf_none")
        async with async_session_factory() as session:
            await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id, site_code="US", category_id=None,
                commission_pct=5, payment_fee_pct=1, fixed_fee=1,
                advertising_pct=0, other_reserve_fee=0, priority=0, remark="us",
            ))
            await session.commit()

        async with async_session_factory() as session:
            matched = await PlatformFeeRuleService.match(
                session,
                PlatformFeeRuleMatchRequest(platform_id=88888, site_code="BR"),
            )
        assert matched is None

    async def test_match_ignores_disabled_rules(self, async_client):
        from app.platform_fee.schemas import PlatformFeeRuleCreate, PlatformFeeRuleMatchRequest
        from app.platform_fee.service import PlatformFeeRuleService

        platform_id = await _create_platform(async_client, "pf_disabled")
        async with async_session_factory() as session:
            await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id, site_code="RU", category_id=None,
                commission_pct=10, payment_fee_pct=2, fixed_fee=2,
                advertising_pct=1, other_reserve_fee=0, priority=0, status=0,
                remark="disabled",
            ))
            await session.commit()

        async with async_session_factory() as session:
            matched = await PlatformFeeRuleService.match(
                session,
                PlatformFeeRuleMatchRequest(platform_id=platform_id, site_code="RU"),
            )
        assert matched is None


class TestPlatformFeeCRUD:
    async def test_create_rule_with_permission(self, async_client):
        headers = await _auth(async_client, "crud_create", "platform_fee:manage")
        platform_id = await _create_platform(async_client, "crud_pf")

        resp = await async_client.post("/api/platform-fee-rules", json={
            "platform_id": platform_id,
            "site_code": "RU",
            "commission_pct": 12,
            "payment_fee_pct": 3,
            "fixed_fee": 5,
            "advertising_pct": 2,
            "other_reserve_fee": 1,
            "priority": 0,
            "status": 1,
            "remark": "test rule",
        }, headers=headers)
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["platform_id"] == platform_id
        assert data["site_code"] == "RU"

    async def test_create_rule_fails_without_permission(self, async_client):
        headers = await _auth(async_client, "crud_no_perm")
        platform_id = await _create_platform(async_client, "crud_np")

        resp = await async_client.post("/api/platform-fee-rules", json={
            "platform_id": platform_id,
            "site_code": "RU",
            "commission_pct": 12,
            "payment_fee_pct": 3,
            "fixed_fee": 5,
            "advertising_pct": 2,
            "other_reserve_fee": 1,
            "priority": 0,
            "status": 1,
        }, headers=headers)
        assert resp.status_code == 403

    async def test_list_rules_with_permission(self, async_client):
        headers = await _auth(async_client, "crud_list", "platform_fee:view")
        resp = await async_client.get("/api/platform-fee-rules", headers=headers)
        assert resp.status_code == 200
        assert isinstance(resp.json()["data"], list)

    async def test_get_rule_detail(self, async_client):
        headers = await _auth(async_client, "crud_detail", "platform_fee:view")
        platform_id = await _create_platform(async_client, "crud_det")
        create_headers = await _auth(async_client, "crud_detail_mgr", "platform_fee:manage")
        create_resp = await async_client.post("/api/platform-fee-rules", json={
            "platform_id": platform_id,
            "site_code": "MY",
            "commission_pct": 8,
            "payment_fee_pct": 2,
            "fixed_fee": 3,
            "advertising_pct": 1,
            "other_reserve_fee": 0,
            "priority": 0,
        }, headers=create_headers)
        rule_id = create_resp.json()["data"]["id"]

        resp = await async_client.get(f"/api/platform-fee-rules/{rule_id}", headers=headers)
        assert resp.status_code == 200
        assert resp.json()["data"]["site_code"] == "MY"

    async def test_update_rule_with_audit_log(self, async_client):
        headers = await _auth(async_client, "crud_upd", "platform_fee:manage")
        platform_id = await _create_platform(async_client, "crud_upd_pf")
        create_resp = await async_client.post("/api/platform-fee-rules", json={
            "platform_id": platform_id,
            "site_code": "RU",
            "commission_pct": 10,
            "payment_fee_pct": 2,
            "fixed_fee": 3,
            "advertising_pct": 1,
            "other_reserve_fee": 0,
            "priority": 0,
        }, headers=headers)
        rule_id = create_resp.json()["data"]["id"]

        resp = await async_client.put(f"/api/platform-fee-rules/{rule_id}", json={
            "commission_pct": 15,
            "remark": "updated",
        }, headers=headers)
        assert resp.status_code == 200
        assert resp.json()["data"]["commission_pct"] == 15

        # Verify audit log via direct session
        async with async_session_factory() as session:
            result = await session.execute(
                select(OperationLog).where(
                    OperationLog.module == "platform_fee",
                    OperationLog.action == "update",
                    OperationLog.resource_id == str(rule_id),
                )
            )
            logs = result.scalars().all()
        assert len(logs) >= 1

    async def test_delete_rule_with_audit_log(self, async_client):
        headers = await _auth(async_client, "crud_del", "platform_fee:manage")
        platform_id = await _create_platform(async_client, "crud_del_pf")
        create_resp = await async_client.post("/api/platform-fee-rules", json={
            "platform_id": platform_id,
            "site_code": "RU",
            "commission_pct": 10,
            "payment_fee_pct": 2,
            "fixed_fee": 3,
            "advertising_pct": 1,
            "other_reserve_fee": 0,
            "priority": 0,
        }, headers=headers)
        rule_id = create_resp.json()["data"]["id"]

        resp = await async_client.delete(f"/api/platform-fee-rules/{rule_id}", headers=headers)
        assert resp.status_code == 200

        # Verify audit log via direct session
        async with async_session_factory() as session:
            result = await session.execute(
                select(OperationLog).where(
                    OperationLog.module == "platform_fee",
                    OperationLog.action == "delete",
                    OperationLog.resource_id == str(rule_id),
                )
            )
            logs = result.scalars().all()
        assert len(logs) >= 1

    async def test_match_endpoint_returns_correct_rule(self, async_client):
        headers = await _auth(async_client, "match_ep", "platform_fee:calculate")
        platform_id = await _create_platform(async_client, "match_ep_pf")
        manage_headers = await _auth(async_client, "match_ep_mgr", "platform_fee:manage")
        await async_client.post("/api/platform-fee-rules", json={
            "platform_id": platform_id,
            "site_code": None,
            "category_id": None,
            "commission_pct": 5,
            "payment_fee_pct": 1,
            "fixed_fee": 1,
            "advertising_pct": 0,
            "other_reserve_fee": 0,
            "priority": 0,
            "remark": "global",
        }, headers=manage_headers)
        site_resp = await async_client.post("/api/platform-fee-rules", json={
            "platform_id": platform_id,
            "site_code": "RU",
            "category_id": None,
            "commission_pct": 10,
            "payment_fee_pct": 2,
            "fixed_fee": 2,
            "advertising_pct": 1,
            "other_reserve_fee": 0,
            "priority": 0,
            "remark": "site",
        }, headers=manage_headers)
        site_rule_id = site_resp.json()["data"]["id"]

        resp = await async_client.post("/api/platform-fee-rules/match", json={
            "platform_id": platform_id,
            "site_code": "RU",
        }, headers=headers)
        assert resp.status_code == 200
        assert resp.json()["data"]["id"] == site_rule_id

    async def test_match_endpoint_returns_404_when_no_match(self, async_client):
        headers = await _auth(async_client, "match_404", "platform_fee:calculate")
        resp = await async_client.post("/api/platform-fee-rules/match", json={
            "platform_id": 99999,
            "site_code": "ZZ",
        }, headers=headers)
        assert resp.status_code == 200
        assert resp.json()["data"] is None


@pytest.mark.asyncio
async def test_prelisting_decision_uses_fee_rule_when_platform_provided(async_client):
    from tests.auth_helpers import grant_permission, register_and_login
    from uuid import uuid4

    uid, token = await register_and_login(async_client, "decision_fee_rule")
    await grant_permission(uid, "platform:create")
    await grant_permission(uid, "platform_fee:manage")
    await grant_permission(uid, "decision:calculate")
    await grant_permission(uid, "product:create")
    await grant_permission(uid, "sku:create")
    await grant_permission(uid, "sku:update")
    await grant_permission(uid, "shipping:manage")
    headers = {"Authorization": f"Bearer {token}"}

    platform_resp = await async_client.post(
        "/api/platforms",
        json={"name": f"Ozon-{uuid4().hex[:6]}", "code": f"ozon_{uuid4().hex[:6]}"},
        headers=headers,
    )
    assert platform_resp.status_code == 200
    platform_id = platform_resp.json()["data"]["id"]

    # Create a platform fee rule
    rule_resp = await async_client.post(
        "/api/platform-fee-rules",
        json={
            "platform_id": platform_id,
            "site_code": "RU",
            "commission_pct": 20,
            "payment_fee_pct": 5,
            "fixed_fee": 3,
            "advertising_pct": 2,
            "other_reserve_fee": 4,
            "priority": 0,
        },
        headers=headers,
    )
    assert rule_resp.status_code == 200, f"rule create failed: {rule_resp.text}"
    rule_id = rule_resp.json()["data"]["id"]

    # Create SKU + shipping test data with auth
    sku_id = await _create_auth_test_data(async_client, headers)

    resp = await async_client.post(
        "/api/decisions/prelisting",
        json={
            "sku_id": sku_id,
            "destination_country": "RU",
            "target_sale_price": 200,
            "platform_id": platform_id,
            "platform_fee_pct": 10,  # manual fallback — should be overridden by rule
            "payment_fee_pct": 3,
            "other_fee": 2,
            "minimum_margin_pct": 10,
            "cargo_type": "normal",
        },
        headers=headers,
    )
    assert resp.status_code == 200, f"decision failed: {resp.text}"
    data = resp.json()["data"]
    assert data["applied_platform_fee_rule_id"] == rule_id
    assert data["platform_fee_source"] == "rule"
    assert data["platform_fee"] == 40.0  # 200 * 20%
    assert data["payment_fee"] == 10.0  # 200 * 5%
    assert data["advertising_fee"] == 4.0  # 200 * 2%
    assert data["fixed_fee"] == 3.0
    assert data["other_fee"] == 4.0  # overridden by rule's other_reserve_fee


async def _create_auth_test_data(async_client, headers) -> int:
    """Create product+SKU+shipping seed data with auth headers, return sku_id"""
    uid = uuid4().hex[:6]

    # 1. 创建商品（含物流包装字段）
    resp = await async_client.post(
        "/api/products",
        json={
            "name": f"Test_{uid}",
            "package_length_cm": 30,
            "package_width_cm": 20,
            "package_height_cm": 10,
            "package_weight_kg": 0.5,
            "cargo_type": "normal",
        },
        headers=headers,
    )
    assert resp.status_code == 200, f"创建商品失败: {resp.text}"
    pid = resp.json()["data"]["id"]

    # 2. 定义规格 + 生成 SKU
    await async_client.post(
        f"/api/products/{pid}/specs",
        json={"specs": [{"name": "颜色", "values": ["标准"]}]},
        headers=headers,
    )
    resp = await async_client.post(f"/api/products/{pid}/skus/generate", headers=headers)
    assert resp.status_code == 200, f"生成 SKU 失败: {resp.text}"
    sku_id = resp.json()["data"]["skus"][0]["id"]

    # 设置成本价
    await async_client.put(
        f"/api/skus/{sku_id}",
        json={"cost_price": 500, "code": f"SKU-{uid}"},
        headers=headers,
    )

    # 3. 创建物流供应商
    resp = await async_client.post(
        "/api/shipping/providers",
        json={"name": f"P_{uid}", "code": f"p_{uid}"},
        headers=headers,
    )
    assert resp.status_code == 200, f"创建物流供应商失败: {resp.text}"
    provid = resp.json()["data"]["id"]

    # 4. 创建物流渠道
    resp = await async_client.post(
        "/api/shipping/channels",
        json={
            "provider_id": provid,
            "name": f"C_{uid}",
            "code": f"c_{uid}",
            "cargo_types": ["normal"],
        },
        headers=headers,
    )
    assert resp.status_code == 200, f"创建物流渠道失败: {resp.text}"
    cid = resp.json()["data"]["id"]

    # 5. 创建区域
    await async_client.post(
        f"/api/shipping/channels/{cid}/zones",
        json={"country_code": "RU"},
        headers=headers,
    )

    # 6. 创建报价规则
    await async_client.post(
        f"/api/shipping/channels/{cid}/rules",
        json={
            "rule_type": "fixed_plus_per_kg",
            "fixed_fee": 50,
            "per_kg_price": 20,
            "minimum_charge": 25,
            "rounding_increment": 0.1,
        },
        headers=headers,
    )

    return sku_id


@pytest.mark.asyncio
async def test_prelisting_decision_falls_back_to_manual_fee_when_no_rule_matches(async_client):
    from tests.auth_helpers import grant_permission, register_and_login
    from uuid import uuid4

    uid, token = await register_and_login(async_client, "decision_fee_fallback")
    await grant_permission(uid, "platform:create")
    await grant_permission(uid, "decision:calculate")
    await grant_permission(uid, "product:create")
    await grant_permission(uid, "sku:create")
    await grant_permission(uid, "sku:update")
    await grant_permission(uid, "shipping:manage")
    headers = {"Authorization": f"Bearer {token}"}

    platform_resp = await async_client.post(
        "/api/platforms",
        json={"name": f"Shopee-{uuid4().hex[:6]}", "code": f"shopee_{uuid4().hex[:6]}"},
        headers=headers,
    )
    assert platform_resp.status_code == 200
    platform_id = platform_resp.json()["data"]["id"]

    sku_id = await _create_auth_test_data(async_client, headers)

    resp = await async_client.post(
        "/api/decisions/prelisting",
        json={
            "sku_id": sku_id,
            "destination_country": "MY",
            "target_sale_price": 200,
            "platform_id": platform_id,
            "platform_fee_pct": 10,
            "payment_fee_pct": 3,
            "other_fee": 2,
            "minimum_margin_pct": 10,
        },
        headers=headers,
    )
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data["applied_platform_fee_rule_id"] is None
    assert data["platform_fee_source"] == "manual"
    assert data["platform_fee"] == 20.0  # 200 * 10%
    assert data["payment_fee"] == 6.0  # 200 * 3%
    assert "未匹配到平台费用规则" in "".join(data["warnings"])
