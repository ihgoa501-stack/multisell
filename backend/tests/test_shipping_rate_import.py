"""物流报价表导入和区域级报价规则测试。"""

from io import BytesIO
from uuid import uuid4

import openpyxl
import pytest

from app.config import settings


pytestmark = [pytest.mark.asyncio]


@pytest.fixture(autouse=True)
def _auth_disabled():
    original = settings.AUTH_ENABLED
    settings.AUTH_ENABLED = False
    yield
    settings.AUTH_ENABLED = original


def _uc(prefix: str) -> str:
    return f"{prefix}_{uuid4().hex[:8]}"


async def _seed_product_sku(async_client) -> int:
    resp = await async_client.post("/api/products", json={
        "name": f"区域报价测试商品-{uuid4().hex[:8]}",
        "package_length_cm": 30,
        "package_width_cm": 20,
        "package_height_cm": 10,
        "package_weight_kg": 0.8,
        "cargo_type": "normal",
    })
    assert resp.status_code == 200, resp.text
    product_id = resp.json()["data"]["id"]
    resp = await async_client.post(
        f"/api/products/{product_id}/specs",
        json={"specs": [{"name": "颜色", "values": ["标准"]}]},
    )
    assert resp.status_code == 200, resp.text
    resp = await async_client.post(f"/api/products/{product_id}/skus/generate")
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]["skus"][0]["id"]


async def _seed_channel(async_client) -> int:
    resp = await async_client.post("/api/shipping/providers", json={
        "name": f"区域报价物流商-{uuid4().hex[:6]}",
        "code": _uc("zone_provider"),
    })
    assert resp.status_code == 200, resp.text
    provider_id = resp.json()["data"]["id"]
    resp = await async_client.post("/api/shipping/channels", json={
        "provider_id": provider_id,
        "name": f"同渠道分国家报价-{uuid4().hex[:6]}",
        "code": _uc("zone_channel"),
        "volumetric_divisor": 6000,
        "cargo_types": ["normal"],
        "currency": "CNY",
    })
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]["id"]


async def _create_zone(async_client, channel_id: int, country_code: str) -> int:
    resp = await async_client.post(
        f"/api/shipping/channels/{channel_id}/zones",
        json={"country_code": country_code},
    )
    assert resp.status_code == 200, resp.text
    return resp.json()["data"]["id"]


async def test_calculation_uses_zone_specific_rule_for_same_channel(async_client):
    sku_id = await _seed_product_sku(async_client)
    channel_id = await _seed_channel(async_client)
    us_zone_id = await _create_zone(async_client, channel_id, "US")
    de_zone_id = await _create_zone(async_client, channel_id, "DE")

    resp = await async_client.post(f"/api/shipping/channels/{channel_id}/rules", json={
        "zone_id": us_zone_id,
        "rule_type": "fixed_plus_per_kg",
        "fixed_fee": 8,
        "per_kg_price": 42,
        "rounding_increment": 0.1,
    })
    assert resp.status_code == 200, resp.text
    resp = await async_client.post(f"/api/shipping/channels/{channel_id}/rules", json={
        "zone_id": de_zone_id,
        "rule_type": "fixed_plus_per_kg",
        "fixed_fee": 15,
        "per_kg_price": 60,
        "rounding_increment": 0.1,
    })
    assert resp.status_code == 200, resp.text

    us_resp = await async_client.post("/api/shipping/calculate", json={
        "sku_id": sku_id,
        "quantity": 1,
        "destination_country": "US",
        "cargo_type": "normal",
    })
    de_resp = await async_client.post("/api/shipping/calculate", json={
        "sku_id": sku_id,
        "quantity": 1,
        "destination_country": "DE",
        "cargo_type": "normal",
    })

    assert us_resp.status_code == 200
    assert de_resp.status_code == 200
    us_result = us_resp.json()["data"]["results"][0]
    de_result = de_resp.json()["data"]["results"][0]
    assert us_result["total_shipping_fee"] == 50.0
    assert de_result["total_shipping_fee"] == 75.0


def _xlsx_bytes(rows: list[dict]) -> bytes:
    wb = openpyxl.Workbook()
    ws = wb.active
    headers = list(rows[0].keys())
    ws.append(headers)
    for row in rows:
        ws.append([row.get(header) for header in headers])
    stream = BytesIO()
    wb.save(stream)
    return stream.getvalue()


async def test_import_xlsx_creates_provider_channel_zone_and_rule(async_client):
    channel_name = f"美国专线-{uuid4().hex[:6]}"
    content = _xlsx_bytes([
        {
            "provider_name": "云途测试导入",
            "provider_code": _uc("yuntu_import"),
            "channel_name": channel_name,
            "channel_code": _uc("us_line"),
            "country_code": "US",
            "rule_type": "fixed_plus_per_kg",
            "fixed_fee": 8,
            "per_kg_price": 42,
            "minimum_charge": 25,
            "rounding_increment": 0.1,
            "volumetric_divisor": 6000,
            "cargo_types": "normal,battery",
            "currency": "CNY",
        }
    ])

    resp = await async_client.post(
        "/api/shipping/import-rules",
        files={
            "file": (
                "shipping-rates.xlsx",
                content,
                "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            )
        },
    )

    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["code"] == 200
    assert body["data"]["imported_rows"] == 1
    assert body["data"]["error_rows"] == 0
    assert body["data"]["created_providers"] == 1
    assert body["data"]["created_channels"] == 1
    assert body["data"]["created_zones"] == 1
    assert body["data"]["created_rules"] == 1

    channels_resp = await async_client.get("/api/shipping/channels")
    imported_channel = next(
        channel for channel in channels_resp.json()["data"]
        if channel["name"] == channel_name
    )
    rules_resp = await async_client.get(f"/api/shipping/channels/{imported_channel['id']}/rules")
    rules = rules_resp.json()["data"]
    assert len(rules) == 1
    assert rules[0]["country_code"] == "US"
    assert rules[0]["fixed_fee"] == 8.0
    assert rules[0]["per_kg_price"] == 42.0


async def test_import_reports_row_errors_without_creating_rule(async_client):
    content = _xlsx_bytes([
        {
            "provider_name": "错误报价物流商",
            "channel_name": "错误报价渠道",
            "country_code": "",
            "rule_type": "fixed_plus_per_kg",
            "fixed_fee": 8,
            "per_kg_price": 42,
        }
    ])

    resp = await async_client.post(
        "/api/shipping/import-rules",
        files={
            "file": (
                "bad-rates.xlsx",
                content,
                "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            )
        },
    )

    assert resp.status_code == 200
    body = resp.json()
    assert body["code"] == 200
    assert body["data"]["imported_rows"] == 0
    assert body["data"]["error_rows"] == 1
    assert body["data"]["errors"][0]["row"] == 2
    assert "country_code" in body["data"]["errors"][0]["message"]
