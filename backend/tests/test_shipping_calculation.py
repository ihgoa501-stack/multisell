"""运费计算核心逻辑测试"""

import pytest
from uuid import uuid4

from app.config import settings


pytestmark = [pytest.mark.asyncio]


# ── fixtures ──────────────────────────────────────────────────────────────


@pytest.fixture(autouse=True)
def _auth_disabled():
    original = settings.AUTH_ENABLED
    settings.AUTH_ENABLED = False
    yield
    settings.AUTH_ENABLED = original


_COUNTER = 0


def _unique_country() -> str:
    """每个测试函数使用唯一的目的地国家代码，避免数据干扰。"""
    global _COUNTER
    _COUNTER += 1
    return f"Z{_COUNTER:03d}"


async def _ensure_seed(async_client, country: str):
    """创建基础种子数据（含固定费+每公斤规则）。"""
    uid = uuid4().hex[:6]
    resp = await async_client.post(
        "/api/products",
        json={
            "name": f"Test_{uid}",
            "package_length_cm": 30,
            "package_width_cm": 20,
            "package_height_cm": 10,
            "package_weight_kg": 0.8,
            "cargo_type": "normal",
        },
    )
    assert resp.status_code == 200
    pid = resp.json()["data"]["id"]

    await async_client.post(
        f"/api/products/{pid}/specs",
        json={"specs": [{"name": "颜色", "values": ["红"]}]},
    )
    resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    sid = resp.json()["data"]["skus"][0]["id"]

    resp = await async_client.post(
        "/api/shipping/providers",
        json={
            "name": f"P_{uid}",
            "code": f"p_{uid}",
        },
    )
    provid = resp.json()["data"]["id"]
    resp = await async_client.post(
        "/api/shipping/channels",
        json={
            "provider_id": provid,
            "name": f"C_{uid}",
            "code": f"c_{uid}",
            "volumetric_divisor": 6000,
            "cargo_types": ["normal"],
        },
    )
    cid = resp.json()["data"]["id"]
    await async_client.post(
        f"/api/shipping/channels/{cid}/zones", json={"country_code": country}
    )
    await async_client.post(
        f"/api/shipping/channels/{cid}/rules",
        json={
            "rule_type": "fixed_plus_per_kg",
            "fixed_fee": 8,
            "per_kg_price": 42,
            "minimum_charge": 25,
            "rounding_increment": 0.1,
        },
    )
    return {"sku_id": sid, "provider_id": provid, "channel_id": cid}


async def _seed_channel(async_client, country: str, rule_data: dict) -> tuple:
    """创建供应商+渠道+区域+规则，返回 (provider_id, channel_id)。"""
    uid = uuid4().hex[:6]
    resp = await async_client.post(
        "/api/shipping/providers",
        json={
            "name": f"S_{uid}",
            "code": f"s_{uid}",
        },
    )
    assert resp.status_code == 200, f"create provider failed: {resp.text}"
    pid = resp.json()["data"]["id"]

    resp = await async_client.post(
        "/api/shipping/channels",
        json={
            "provider_id": pid,
            "name": f"Ch_{uid}",
            "code": f"ch_{uid}",
            "volumetric_divisor": rule_data.get("volumetric_divisor", 6000),
            "cargo_types": rule_data.get("cargo_types", ["normal"]),
        },
    )
    assert resp.status_code == 200, f"create channel failed: {resp.text}"
    cid = resp.json()["data"]["id"]

    await async_client.post(
        f"/api/shipping/channels/{cid}/zones", json={"country_code": country}
    )

    rule = {
        k: v
        for k, v in rule_data.items()
        if k not in ("volumetric_divisor", "cargo_types")
    }
    rule.setdefault("rule_type", "fixed_plus_per_kg")
    rule.setdefault("rounding_increment", 0.1)
    resp = await async_client.post(f"/api/shipping/channels/{cid}/rules", json=rule)
    assert resp.status_code == 200, f"create rule failed: {resp.text}"
    return pid, cid


async def _seed_product_sku(async_client, pkg: dict) -> int:
    """创建商品+SKU，返回 SKU ID。"""
    uid = uuid4().hex[:6]
    resp = await async_client.post("/api/products", json={"name": f"Prod_{uid}", **pkg})
    assert resp.status_code == 200, f"create product failed: {resp.text}"
    pid = resp.json()["data"]["id"]
    await async_client.post(
        f"/api/products/{pid}/specs",
        json={"specs": [{"name": "颜色", "values": ["标准"]}]},
    )
    resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    assert resp.status_code == 200, f"generate sku failed: {resp.text}"
    return resp.json()["data"]["skus"][0]["id"]


# ── 测试用例 ─────────────────────────────────────────────────────────────


class TestPackageResolution:
    async def test_product_package_fallback(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        body = resp.json()
        assert body["data"]["package"]["source"] == "product"
        assert body["data"]["package"]["weight_kg"] == 0.8

    async def test_sku_override(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        await async_client.put(
            f"/api/skus/{d['sku_id']}",
            json={
                "sku_length_cm": 25,
                "sku_width_cm": 15,
                "sku_height_cm": 8,
                "sku_weight_kg": 0.5,
            },
        )
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        assert resp.json()["data"]["package"]["source"] == "sku"
        assert resp.json()["data"]["package"]["weight_kg"] == 0.5

    async def test_missing_package_data_blocks_calculation(self, async_client):
        sid = await _seed_product_sku(async_client, {})
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": "US",
                "cargo_type": "normal",
            },
        )
        assert resp.json()["code"] == 400
        assert "物流数据不完整" in resp.json()["message"]


class TestWeightFormula:
    async def test_volumetric_weight_greater_than_actual(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        r = resp.json()["data"]["results"][0]
        assert r["volumetric_weight_kg"] == 1.0
        assert r["actual_weight_kg"] == 0.8
        assert r["chargeable_weight_kg"] == 1.0

    async def test_actual_weight_greater_than_volumetric(self, async_client):
        c = _unique_country()
        _, cid = await _seed_channel(
            async_client,
            c,
            {
                "volumetric_divisor": 6000,
                "fixed_fee": 5,
                "per_kg_price": 20,
            },
        )
        sid = await _seed_product_sku(
            async_client,
            {
                "package_length_cm": 10,
                "package_width_cm": 10,
                "package_height_cm": 10,
                "package_weight_kg": 5.0,
            },
        )
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        r = [x for x in resp.json()["data"]["results"] if x["channel_id"] == cid]
        assert len(r) == 1, f"channel {cid} not found in results: {resp.json()}"
        r = r[0]
        assert r["actual_weight_kg"] == 5.0
        assert r["volumetric_weight_kg"] == pytest.approx(0.167, rel=0.01)
        assert r["chargeable_weight_kg"] == 5.0

    async def test_quantity_multiplies_weight_and_volume(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 3,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        r = resp.json()["data"]["results"][0]
        assert r["actual_weight_kg"] == 2.4
        assert r["volumetric_weight_kg"] == 3.0
        assert r["chargeable_weight_kg"] == 3.0

    async def test_rounding(self, async_client):
        c = _unique_country()
        _, cid = await _seed_channel(
            async_client,
            c,
            {
                "volumetric_divisor": 5000,
                "fixed_fee": 10,
                "per_kg_price": 30,
                "rounding_increment": 0.5,
            },
        )
        sid = await _seed_product_sku(
            async_client,
            {
                "package_length_cm": 20,
                "package_width_cm": 15,
                "package_height_cm": 10,
                "package_weight_kg": 0.6,
            },
        )
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        r = [x for x in resp.json()["data"]["results"] if x["channel_id"] == cid]
        assert len(r) == 1, f"channel {cid} not found"
        assert r[0]["chargeable_weight_kg"] == 1.0


class TestQuoteRuleTypes:
    async def test_fixed_plus_per_kg(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        r = resp.json()["data"]["results"][0]
        assert r["total_shipping_fee"] == 50.0

    async def test_first_weight_plus_increment(self, async_client):
        c = _unique_country()
        _, cid = await _seed_channel(
            async_client,
            c,
            {
                "volumetric_divisor": 8000,
                "rule_type": "first_weight_plus_increment",
                "first_kg": 0.1,
                "first_price": 20,
                "additional_kg": 0.1,
                "additional_price": 5,
            },
        )
        sid = await _seed_product_sku(
            async_client,
            {
                "package_length_cm": 20,
                "package_width_cm": 15,
                "package_height_cm": 5,
                "package_weight_kg": 0.35,
            },
        )
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        r = [x for x in resp.json()["data"]["results"] if x["channel_id"] == cid]
        assert len(r) == 1, f"channel {cid} not found"
        r = r[0]
        assert r["chargeable_weight_kg"] == 0.4
        assert r["total_shipping_fee"] == 35.0

    async def test_tiered_weight(self, async_client):
        c = _unique_country()
        _, cid = await _seed_channel(
            async_client,
            c,
            {
                "volumetric_divisor": 5000,
                "rule_type": "tiered_weight",
                "tier_config": [
                    {"min_kg": 0, "max_kg": 0.5, "price": 60},
                    {"min_kg": 0.5, "max_kg": 1, "price": 80},
                    {"min_kg": 1, "max_kg": 2, "price": 120},
                ],
            },
        )
        sid = await _seed_product_sku(
            async_client,
            {
                "package_length_cm": 25,
                "package_width_cm": 20,
                "package_height_cm": 15,
                "package_weight_kg": 1.2,
            },
        )
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        r = [x for x in resp.json()["data"]["results"] if x["channel_id"] == cid]
        assert len(r) == 1, f"channel {cid} not found"
        r = r[0]
        assert r["chargeable_weight_kg"] == 1.5
        assert r["total_shipping_fee"] == 120.0


class TestSurchargesAndMinimum:
    async def test_minimum_charge_applied(self, async_client):
        c = _unique_country()
        _, cid = await _seed_channel(
            async_client,
            c,
            {
                "fixed_fee": 8,
                "per_kg_price": 42,
                "minimum_charge": 25,
            },
        )
        sid = await _seed_product_sku(
            async_client,
            {
                "package_length_cm": 5,
                "package_width_cm": 5,
                "package_height_cm": 3,
                "package_weight_kg": 0.05,
            },
        )
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        r = [x for x in resp.json()["data"]["results"] if x["channel_id"] == cid]
        assert len(r) == 1, f"channel {cid} not found"
        assert r[0]["minimum_applied"] is True
        assert r[0]["total_shipping_fee"] == 25.0

    async def test_fixed_surcharge(self, async_client):
        c = _unique_country()
        _, cid = await _seed_channel(
            async_client,
            c,
            {
                "cargo_types": ["battery"],
                "fixed_fee": 10,
                "per_kg_price": 50,
                "minimum_charge": 30,
                "surcharge_fixed": 15,
                "fuel_surcharge_pct": 15,
            },
        )
        sid = await _seed_product_sku(
            async_client,
            {
                "package_length_cm": 20,
                "package_width_cm": 15,
                "package_height_cm": 10,
                "package_weight_kg": 0.5,
            },
        )
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "battery",
            },
        )
        r = [x for x in resp.json()["data"]["results"] if x["channel_id"] == cid]
        assert len(r) == 1, f"channel {cid} not found"
        r = r[0]
        assert r["surcharge_fee"] == 15.0
        assert r["fuel_surcharge_fee"] == 7.5
        assert r["total_shipping_fee"] == 57.5


class TestFiltering:
    async def test_inactive_provider_excluded(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        await async_client.put(
            f"/api/shipping/providers/{d['provider_id']}", json={"status": 0}
        )
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        assert len(resp.json()["data"]["results"]) == 0

    async def test_cargo_type_mismatch_excluded(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "battery",
            },
        )
        assert len(resp.json()["data"]["results"]) == 0

    async def test_destination_country_mismatch_excluded(self, async_client):
        c = _unique_country()
        unmatched_country = _unique_country()
        d = await _ensure_seed(async_client, c)
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": unmatched_country,
                "cargo_type": "normal",
            },
        )
        assert len(resp.json()["data"]["results"]) == 0

    async def test_results_sorted_by_total_fee(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        # 再建一个更便宜的渠道（同一个供应商）
        uid = uuid4().hex[:6]
        resp = await async_client.post(
            "/api/shipping/channels",
            json={
                "provider_id": d["provider_id"],
                "name": f"Eco_{uid}",
                "code": f"eco_{uid}",
                "volumetric_divisor": 8000,
                "cargo_types": ["normal"],
            },
        )
        assert resp.status_code == 200
        cid2 = resp.json()["data"]["id"]
        await async_client.post(
            f"/api/shipping/channels/{cid2}/zones", json={"country_code": c}
        )
        await async_client.post(
            f"/api/shipping/channels/{cid2}/rules",
            json={
                "rule_type": "fixed_plus_per_kg",
                "fixed_fee": 5,
                "per_kg_price": 30,
                "rounding_increment": 0.1,
            },
        )

        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        results = resp.json()["data"]["results"]
        assert len(results) == 2, (
            f"expected 2 results, got {len(results)}: {resp.json()}"
        )
        assert results[0]["total_shipping_fee"] <= results[1]["total_shipping_fee"]


class TestManualCalculation:
    async def test_manual_package_calculation(self, async_client):
        country = _unique_country()
        _, cid = await _seed_channel(
            async_client,
            country,
            {
                "volumetric_divisor": 6000,
                "fixed_fee": 8,
                "per_kg_price": 42,
                "minimum_charge": 25,
                "cargo_types": ["normal"],
            },
        )

        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "mode": "manual",
                "quantity": 1,
                "destination_country": country,
                "cargo_type": "normal",
                "package": {
                    "length_cm": 30,
                    "width_cm": 20,
                    "height_cm": 10,
                    "weight_kg": 0.8,
                },
            },
        )

        body = resp.json()
        assert body["code"] == 200
        assert body["data"]["mode"] == "manual"
        assert body["data"]["sku_id"] is None
        assert body["data"]["package"]["source"] == "manual"
        result = [
            item for item in body["data"]["results"] if item["channel_id"] == cid
        ][0]
        assert result["actual_weight_kg"] == 0.8
        assert result["volumetric_weight_kg"] == 1.0
        assert result["chargeable_weight_kg"] == 1.0
        assert result["total_shipping_fee"] == 50.0

    async def test_manual_package_requires_positive_dimensions(self, async_client):
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "mode": "manual",
                "quantity": 1,
                "destination_country": "US",
                "cargo_type": "normal",
                "package": {
                    "length_cm": 0,
                    "width_cm": 20,
                    "height_cm": 10,
                    "weight_kg": 0.8,
                },
            },
        )
        assert resp.status_code in (200, 422)
        if resp.status_code == 200:
            assert resp.json()["code"] == 400

    async def test_manual_mode_must_have_package(self, async_client):
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "mode": "manual",
                "quantity": 1,
                "destination_country": "US",
                "cargo_type": "normal",
            },
        )
        assert resp.status_code in (200, 422)
        if resp.status_code == 200:
            assert resp.json()["code"] == 400

    async def test_backward_compatible_sku_mode_without_mode_field(self, async_client):
        c = _unique_country()
        d = await _ensure_seed(async_client, c)
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": d["sku_id"],
                "quantity": 1,
                "destination_country": c,
                "cargo_type": "normal",
            },
        )
        assert resp.status_code == 200
        body = resp.json()
        assert body["data"]["mode"] == "sku"
        assert body["data"]["sku_id"] == d["sku_id"]
