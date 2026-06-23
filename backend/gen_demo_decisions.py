"""通过 API 生成 Agent 演示决策"""

import httpx
import asyncio

API = "http://localhost:8001/api"

DECISIONS = {
    "A5": [
        {
            "dp": "stock_alert",
            "ctx": {
                "sku_code": "DEMO-RED",
                "sellable_stock": 5,
                "locked_stock": 2,
                "in_transit_stock": 0,
                "sales_7d": 21,
                "lead_time_days": 20,
                "moq": 200,
                "safety_stock_days": 14,
            },
        },
        {
            "dp": "stock_alert",
            "ctx": {
                "sku_code": "DEMO-YELLOW",
                "sellable_stock": 50,
                "locked_stock": 5,
                "in_transit_stock": 30,
                "sales_7d": 28,
                "lead_time_days": 25,
                "moq": 200,
                "safety_stock_days": 14,
            },
        },
        {
            "dp": "stock_alert",
            "ctx": {
                "sku_code": "DEMO-GREEN",
                "sellable_stock": 500,
                "locked_stock": 20,
                "in_transit_stock": 200,
                "sales_7d": 35,
                "lead_time_days": 20,
                "moq": 100,
                "safety_stock_days": 14,
            },
        },
    ],
    "G3": [
        {
            "dp": "discount_check",
            "ctx": {
                "sku_code": "DEMO-BLOCK",
                "selling_price": 100,
                "cost_price": 85,
                "active_discounts": [
                    {"type": "coupon", "value": 10},
                    {"type": "promotion", "value": 10},
                ],
                "platform": "amazon",
            },
        },
        {
            "dp": "discount_check",
            "ctx": {
                "sku_code": "DEMO-WARN",
                "selling_price": 100,
                "cost_price": 85,
                "active_discounts": [{"type": "coupon", "value": 8}],
                "platform": "shopify",
                "min_margin_threshold": 10,
            },
        },
        {
            "dp": "discount_check",
            "ctx": {
                "sku_code": "DEMO-ALLOW",
                "selling_price": 200,
                "cost_price": 80,
                "active_discounts": [{"type": "coupon", "value": 15}],
                "platform": "ozon",
            },
        },
    ],
    "A6": [
        {
            "dp": "profit_check",
            "ctx": {
                "sku_code": "DEMO-LOSS",
                "selling_price": 80,
                "cost_price": 60,
                "platform_fee_rate": 15,
                "shipping_fee": 18,
                "ad_cost_per_unit": 10,
            },
        },
        {
            "dp": "profit_check",
            "ctx": {
                "sku_code": "DEMO-LOW",
                "selling_price": 150,
                "cost_price": 100,
                "platform_fee_rate": 12,
                "shipping_fee": 15,
                "fixed_fee": 3,
                "min_margin_threshold": 20,
            },
        },
        {
            "dp": "profit_check",
            "ctx": {
                "sku_code": "DEMO-OK",
                "selling_price": 299,
                "cost_price": 120,
                "platform_fee_rate": 10,
                "shipping_fee": 12,
                "ad_cost_per_unit": 5,
                "min_margin_threshold": 15,
            },
        },
    ],
    "A3": [
        {
            "dp": "acos_analysis",
            "ctx": {
                "campaign_id": "CAM-DEMO-HIGH",
                "spend": 500,
                "sales": 1000,
                "clicks": 200,
                "impressions": 8000,
                "conversions": 8,
                "gross_margin": 30,
                "target_acos": 25,
            },
        },
        {
            "dp": "acos_analysis",
            "ctx": {
                "campaign_id": "CAM-DEMO-OK",
                "spend": 200,
                "sales": 1200,
                "clicks": 150,
                "impressions": 6000,
                "conversions": 12,
                "gross_margin": 40,
                "target_acos": 30,
            },
        },
    ],
}


async def main():
    async with httpx.AsyncClient(base_url=API, timeout=10) as c:
        for agent_id, items in DECISIONS.items():
            for item in items:
                try:
                    r = await c.post(
                        f"/agents/{agent_id}/decide",
                        json={
                            "decision_point": item["dp"],
                            "context": item["ctx"],
                        },
                    )
                    js = r.json()
                    data = js.get("data", js)
                    did = data.get("decision_id", "?")
                    print(f"  {agent_id}/{item['dp']} → id={did}")
                except Exception as e:
                    print(f"  {agent_id}/{item['dp']} → ERROR: {e}")


asyncio.run(main())
