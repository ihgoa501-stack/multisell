"""平台费用规则管理 API 测试"""
import pytest
from uuid import uuid4
from httpx import AsyncClient

pytestmark = [pytest.mark.asyncio]


def _uc(name: str) -> str:
    return f"{name}_{uuid4().hex[:6]}"


async def _get_platform_id(async_client, name=None):
    name = name or f"Ozon_{uuid4().hex[:6]}"
    resp = await async_client.post("/api/platforms", json={"name": name, "code": name.lower()})
    assert resp.status_code == 200, f"Create platform failed: {resp.text}"
    return resp.json()["data"]["id"]


async def _ensure_category(async_client, name="测试类目"):
    resp = await async_client.get("/api/categories")
    cats = resp.json().get("data", [])
    if cats:
        return cats[0]["id"]
    resp = await async_client.post("/api/categories", json={"name": name})
    assert resp.status_code == 200, f"Create category failed: {resp.text}"
    return resp.json()["data"]["id"]


async def _create_rule(async_client, platform_id: int, **overrides):
    payload = {
        "platform_id": platform_id,
        "fee_type": "commission",
        "fee_rate_pct": 5.0,
        "currency": "CNY",
        "status": "active",
        "priority": 0,
    }
    payload.update(overrides)
    resp = await async_client.post("/api/platform-fee/rules", json=payload)
    assert resp.status_code == 200
    assert resp.json()["code"] == 200
    return resp.json()["data"]


class TestRuleCRUD:
    """费用规则 CRUD"""

    async def test_create_commission_rule(self, async_client):
        pid = await _get_platform_id(async_client)
        rule = await _create_rule(async_client, pid, country_code="RU", fee_type="commission", fee_rate_pct=5.0)
        assert rule["fee_type"] == "commission"
        assert rule["fee_rate_pct"] == 5.0
        assert rule["country_code"] == "RU"
        assert rule["platform_id"] == pid
        assert rule["status"] == "active"
        assert "id" in rule

    async def test_create_rule_with_null_country(self, async_client):
        pid = await _get_platform_id(async_client)
        rule = await _create_rule(async_client, pid, country_code=None, fee_type="fixed", fixed_amount=10.0)
        assert rule["country_code"] is None
        assert rule["fee_type"] == "fixed"
        assert rule["fixed_amount"] == 10.0

    async def test_list_rules_with_filters(self, async_client):
        pid = await _get_platform_id(async_client)
        await _create_rule(async_client, pid, country_code="RU", fee_type="commission", fee_rate_pct=3.0)
        await _create_rule(async_client, pid, country_code="RU", fee_type="fixed", fixed_amount=5.0)

        resp = await async_client.get(f"/api/platform-fee/rules?platform_id={pid}&country_code=RU")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert len(data["records"]) >= 2

        resp = await async_client.get("/api/platform-fee/rules?fee_type=commission&status=active")
        assert resp.status_code == 200
        for r in resp.json()["records"]:
            assert r["fee_type"] == "commission"

    async def test_update_rule(self, async_client):
        pid = await _get_platform_id(async_client)
        rule = await _create_rule(async_client, pid, country_code="RU", fee_rate_pct=3.0)
        rule_id = rule["id"]

        resp = await async_client.put(f"/api/platform-fee/rules/{rule_id}", json={"fee_rate_pct": 8.5})
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["fee_rate_pct"] == 8.5

    async def test_delete_rule(self, async_client):
        pid = await _get_platform_id(async_client)
        rule = await _create_rule(async_client, pid, country_code="RU")
        rule_id = rule["id"]

        resp = await async_client.delete(f"/api/platform-fee/rules/{rule_id}")
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

        resp = await async_client.get(f"/api/platform-fee/rules?platform_id={pid}")
        ids = [r["id"] for r in resp.json()["records"]]
        assert rule_id not in ids

    async def test_get_single_rule(self, async_client):
        pid = await _get_platform_id(async_client)
        rule = await _create_rule(async_client, pid, country_code="RU", fee_type="payment", fee_rate_pct=2.0, remark="test get")
        rule_id = rule["id"]

        resp = await async_client.get(f"/api/platform-fee/rules?platform_id={pid}&country_code=RU")
        records = resp.json()["records"]
        matched = [r for r in records if r["id"] == rule_id]
        assert len(matched) == 1
        assert matched[0]["fee_type"] == "payment"
        assert matched[0]["remark"] == "test get"
        assert matched[0]["fee_rate_pct"] == 2.0


class TestCalculateFee:
    """费用计算"""

    async def test_exact_match(self, async_client):
        pid = await _get_platform_id(async_client)
        cid = await _ensure_category(async_client)
        await _create_rule(async_client, pid, country_code="RU", category_id=cid, fee_type="commission", fee_rate_pct=5.0)

        resp = await async_client.post("/api/platform-fee/calculate", json={
            "platform_id": pid,
            "country_code": "RU",
            "category_id": cid,
            "sale_price": 200,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        result = data["data"]
        assert result["total_fee"] == 10.0
        assert result["rules_matched"] == 1
        assert len(result["items"]) == 1
        assert result["items"][0]["fee_type"] == "commission"

    async def test_fallback_to_platform_default(self, async_client):
        pid = await _get_platform_id(async_client)
        await _create_rule(async_client, pid, country_code="RU", fee_type="commission", fee_rate_pct=5.0)
        await _create_rule(async_client, pid, country_code=None, fee_type="commission", fee_rate_pct=10.0)

        resp = await async_client.post("/api/platform-fee/calculate", json={
            "platform_id": pid,
            "country_code": "US",
            "sale_price": 100,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        result = data["data"]
        assert result["total_fee"] == 10.0
        assert result["rules_matched"] == 1

    async def test_multiple_fee_types(self, async_client):
        pid = await _get_platform_id(async_client)
        await _create_rule(async_client, pid, country_code="RU", fee_type="commission", fee_rate_pct=5.0)
        await _create_rule(async_client, pid, country_code="RU", fee_type="fixed", fixed_amount=3.0, fee_rate_pct=0)

        resp = await async_client.post("/api/platform-fee/calculate", json={
            "platform_id": pid,
            "country_code": "RU",
            "sale_price": 200,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        result = data["data"]
        fee_types = {item["fee_type"] for item in result["items"]}
        assert "commission" in fee_types
        assert "fixed" in fee_types
        assert result["total_fee"] == 13.0
        assert result["rules_matched"] == 2

    async def test_min_amount_respected(self, async_client):
        pid = await _get_platform_id(async_client)
        await _create_rule(async_client, pid, country_code="RU", fee_type="commission", fee_rate_pct=5.0, min_amount=50.0)

        resp = await async_client.post("/api/platform-fee/calculate", json={
            "platform_id": pid,
            "country_code": "RU",
            "sale_price": 100,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        result = data["data"]
        assert result["total_fee"] == 50.0
        assert result["items"][0]["amount"] == 50.0

    async def test_only_active_rules(self, async_client):
        pid = await _get_platform_id(async_client)
        await _create_rule(async_client, pid, country_code="RU", fee_type="commission", fee_rate_pct=5.0)
        await _create_rule(async_client, pid, country_code="RU", fee_type="fixed", fixed_amount=10.0, status="inactive")

        resp = await async_client.post("/api/platform-fee/calculate", json={
            "platform_id": pid,
            "country_code": "RU",
            "sale_price": 100,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        result = data["data"]
        assert result["rules_matched"] == 1
        assert result["items"][0]["fee_type"] == "commission"
        assert result["total_fee"] == 5.0

    async def test_no_rules_matched(self, async_client):
        pid = await _get_platform_id(async_client)

        resp = await async_client.post("/api/platform-fee/calculate", json={
            "platform_id": pid,
            "country_code": "RU",
            "sale_price": 100,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        result = data["data"]
        assert result["total_fee"] == 0.0
        assert result["items"] == []
        assert result["rules_matched"] == 0
