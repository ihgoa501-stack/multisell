"""预上架决策 API 测试"""

from uuid import uuid4

import pytest
from httpx import AsyncClient


@pytest.mark.asyncio
async def test_prelisting_decision_approves_when_margin_high(
    async_client: AsyncClient,
):
    """利润率≥最低利润率 → approve"""
    sku_id = await _create_test_data(async_client)

    resp = await async_client.post(
        "/api/decisions/prelisting",
        json={
            "sku_id": sku_id,
            "destination_country": "RU",
            "target_sale_price": 5000,
            "platform_fee_pct": 10,
            "payment_fee_pct": 3,
            "other_fee": 100,
            "minimum_margin_pct": 20,
            "cargo_type": "normal",
        },
    )
    assert resp.status_code == 200, f"决策请求失败: {resp.text}"
    data = resp.json()["data"]
    assert data["recommendation"] == "approve", f"期望 approve, 实际: {data}"
    assert data["profit_margin"] >= 20, f"利润率不足: {data['profit_margin']}%"
    assert data["blocking_reasons"] == []


@pytest.mark.asyncio
async def test_prelisting_decision_blocks_when_shipping_unavailable(
    async_client: AsyncClient,
):
    """无可用物流渠道 → needs_data"""
    sku_id = await _create_test_data(async_client)

    resp = await async_client.post(
        "/api/decisions/prelisting",
        json={
            "sku_id": sku_id,
            "destination_country": "ZZ",  # 只有 RU 有报价
            "target_sale_price": 5000,
            "platform_fee_pct": 10,
            "payment_fee_pct": 3,
            "other_fee": 100,
            "minimum_margin_pct": 20,
            "cargo_type": "normal",
        },
    )
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data["recommendation"] == "needs_data", f"期望 needs_data, 实际: {data}"


@pytest.mark.asyncio
async def test_prelisting_decision_rejects_when_margin_too_low(
    async_client: AsyncClient,
):
    """利润率<最低利润率 → reject"""
    sku_id = await _create_test_data(async_client)

    resp = await async_client.post(
        "/api/decisions/prelisting",
        json={
            "sku_id": sku_id,
            "destination_country": "RU",
            "target_sale_price": 800,
            "platform_fee_pct": 10,
            "payment_fee_pct": 3,
            "other_fee": 100,
            "minimum_margin_pct": 20,
            "cargo_type": "normal",
        },
    )
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data["recommendation"] == "reject", f"期望 reject, 实际: {data}"
    assert data["profit_margin"] < 20


@pytest.mark.asyncio
async def test_prelisting_decision_404_for_nonexistent_sku(
    async_client: AsyncClient,
):
    """不存在的 SKU → 404"""
    resp = await async_client.post(
        "/api/decisions/prelisting",
        json={
            "sku_id": 99999,
            "destination_country": "RU",
            "target_sale_price": 5000,
            "platform_fee_pct": 10,
            "payment_fee_pct": 3,
            "other_fee": 100,
            "minimum_margin_pct": 20,
            "cargo_type": "normal",
        },
    )
    assert resp.status_code == 404


@pytest.mark.skip(reason="endpoint POST /api/decisions/prelisting/batch not implemented yet")
@pytest.mark.asyncio
async def test_batch_prelisting_decision_returns_summary_and_item_results(
    async_client: AsyncClient,
):
    sku_id = await _create_test_data(async_client)

    resp = await async_client.post(
        "/api/decisions/prelisting/batch",
        json={
            "items": [
                {
                    "item_key": "approve-row",
                    "sku_id": sku_id,
                    "destination_country": "RU",
                    "target_sale_price": 5000,
                    "platform_fee_pct": 10,
                    "payment_fee_pct": 3,
                    "other_fee": 100,
                    "minimum_margin_pct": 20,
                    "cargo_type": "normal",
                },
                {
                    "item_key": "needs-data-row",
                    "sku_id": sku_id,
                    "destination_country": "ZZ",
                    "target_sale_price": 5000,
                    "platform_fee_pct": 10,
                    "payment_fee_pct": 3,
                    "other_fee": 100,
                    "minimum_margin_pct": 20,
                    "cargo_type": "normal",
                },
                {
                    "item_key": "missing-sku-row",
                    "sku_id": 999999,
                    "destination_country": "RU",
                    "target_sale_price": 5000,
                    "platform_fee_pct": 10,
                    "payment_fee_pct": 3,
                    "other_fee": 100,
                    "minimum_margin_pct": 20,
                    "cargo_type": "normal",
                },
            ]
        },
    )

    assert resp.status_code == 200, resp.text
    data = resp.json()["data"]
    assert data["summary"]["total_items"] == 3
    assert data["summary"]["success_count"] == 2
    assert data["summary"]["error_count"] == 1
    assert data["summary"]["approve_count"] == 1
    assert data["summary"]["needs_data_count"] == 1
    assert data["summary"]["reject_count"] == 0
    assert data["summary"]["average_profit_margin"] > 0

    items = data["items"]
    assert items[0]["index"] == 0
    assert items[0]["item_key"] == "approve-row"
    assert items[0]["status"] == "success"
    assert items[0]["result"]["recommendation"] == "approve"
    assert items[0]["error_message"] is None

    assert items[1]["status"] == "success"
    assert items[1]["result"]["recommendation"] == "needs_data"

    assert items[2]["status"] == "error"
    assert items[2]["result"] is None
    assert "SKU不存在" in items[2]["error_message"]


@pytest.mark.skip(reason="endpoint POST /api/decisions/prelisting/batch not implemented yet")
@pytest.mark.asyncio
async def test_batch_prelisting_decision_rejects_empty_items(async_client: AsyncClient):
    resp = await async_client.post(
        "/api/decisions/prelisting/batch",
        json={"items": []},
    )
    assert resp.status_code == 422


async def _create_test_data(async_client: AsyncClient) -> int:
    """创建商品+SKU+物流渠道种子数据，返回 sku_id"""
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
    )
    assert resp.status_code == 200, f"创建商品失败: {resp.text}"
    pid = resp.json()["data"]["id"]

    # 2. 定义规格 + 生成 SKU（SKU 成本价在 generate 后通过 PUT 更新）
    await async_client.post(
        f"/api/products/{pid}/specs",
        json={"specs": [{"name": "颜色", "values": ["标准"]}]},
    )
    resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    assert resp.status_code == 200, f"生成 SKU 失败: {resp.text}"
    sku_id = resp.json()["data"]["skus"][0]["id"]

    # 设置成本价
    await async_client.put(
        f"/api/skus/{sku_id}",
        json={"cost_price": 500, "code": f"SKU-{uid}"},
    )

    # 3. 创建物流供应商
    resp = await async_client.post(
        "/api/shipping/providers",
        json={"name": f"P_{uid}", "code": f"p_{uid}"},
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
    )
    assert resp.status_code == 200, f"创建物流渠道失败: {resp.text}"
    cid = resp.json()["data"]["id"]

    # 5. 创建区域
    await async_client.post(
        f"/api/shipping/channels/{cid}/zones",
        json={"country_code": "RU"},
    )

    # 6. 创建报价规则（固定费 50 + 每公斤 20）
    await async_client.post(
        f"/api/shipping/channels/{cid}/rules",
        json={
            "rule_type": "fixed_plus_per_kg",
            "fixed_fee": 50,
            "per_kg_price": 20,
            "minimum_charge": 25,
            "rounding_increment": 0.1,
        },
    )

    return sku_id
