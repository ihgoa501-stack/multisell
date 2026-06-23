#!/usr/bin/env python3
"""
AI Agent 演示数据生成脚本

为 Agent 系统生成演示用的：
1. 商品物流数据（重量/尺寸）
2. 库存明细（可售/锁定/在途/安全库存）
3. 激活的促销和折扣
4. 通过 API 调用 Agent 决策生成 agent_decision 记录
5. 个人规则种子

用法:
    cd backend
    python seed_agent_demo.py
"""
import asyncio
import sys
import os
import httpx

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from sqlalchemy import select
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

from app.config import settings
from app.models import (
    Product, Sku, Inventory, PersonalRule,
)

API_BASE = "http://localhost:8001/api"


# ── 更新商品物流数据 ──

PRODUCT_LOGISTICS = {
    "TechPro 无线蓝牙耳机 Pro": {
        "package_length_cm": 12, "package_width_cm": 10, "package_height_cm": 5,
        "package_weight_kg": 0.35, "cargo_type": "normal",
    },
    "TechPro 智能手表 S3": {
        "package_length_cm": 15, "package_width_cm": 10, "package_height_cm": 6,
        "package_weight_kg": 0.30, "cargo_type": "normal",
    },
    "StyleWear 男士休闲夹克": {
        "package_length_cm": 35, "package_width_cm": 25, "package_height_cm": 5,
        "package_weight_kg": 0.50, "cargo_type": "normal",
    },
    "NatureHome 不锈钢保温杯 500ml": {
        "package_length_cm": 10, "package_width_cm": 10, "package_height_cm": 28,
        "package_weight_kg": 0.40, "cargo_type": "normal",
    },
    "BeautyGlow 玻尿酸保湿精华液 30ml": {
        "package_length_cm": 8, "package_width_cm": 6, "package_height_cm": 15,
        "package_weight_kg": 0.25, "cargo_type": "sensitive",
    },
    "FreshFood 混合坚果礼盒 1kg": {
        "package_length_cm": 30, "package_width_cm": 20, "package_height_cm": 12,
        "package_weight_kg": 1.10, "cargo_type": "normal",
    },
    "SportMax 瑜伽垫 TPE双面防滑": {
        "package_length_cm": 65, "package_width_cm": 15, "package_height_cm": 15,
        "package_weight_kg": 1.50, "cargo_type": "normal",
    },
}

# ── 库存更新数据（每个 SKU 对应） ──
# key 是 SKU spec_desc 匹配前缀
INVENTORY_UPDATES = {
    # 耳机黑色：低库存 + 有在途
    "黑色":       {"locked": 5,  "in_transit": 100, "safety_stock": 30, "lead_time": 25, "sales_7d": 28, "moq": 200},
    # 耳机白色：库存充足
    "白色":       {"locked": 3,  "in_transit": 50,  "safety_stock": 20, "lead_time": 25, "sales_7d": 14, "moq": 200},
    # 手表午夜黑-运动款：库存正常
    "午夜黑-运动款": {"locked": 8,  "in_transit": 200, "safety_stock": 20, "lead_time": 30, "sales_7d": 21, "moq": 100},
    # 手表午夜黑-皮质款：库存偏低
    "午夜黑-皮质款": {"locked": 2,  "in_transit": 80,  "safety_stock": 15, "lead_time": 30, "sales_7d": 18, "moq": 100},
    # 手表星光银-运动款：库存充足
    "星光银-运动款": {"locked": 5,  "in_transit": 150, "safety_stock": 20, "lead_time": 30, "sales_7d": 14, "moq": 100},
    # 手表星光银-皮质款：库存不足 + 预警
    "星光银-皮质款": {"locked": 1,  "in_transit": 40,  "safety_stock": 10, "lead_time": 30, "sales_7d": 20, "moq": 100},
    # 夹克深蓝-M：畅销 + 库存快卖完
    "深蓝-M":     {"locked": 10, "in_transit": 0,   "safety_stock": 30, "lead_time": 20, "sales_7d": 35, "moq": 150},
    # 夹克深蓝-L
    "深蓝-L":     {"locked": 8,  "in_transit": 0,   "safety_stock": 25, "lead_time": 20, "sales_7d": 28, "moq": 150},
    # 夹克卡其-L
    "卡其-L":     {"locked": 5,  "in_transit": 0,   "safety_stock": 20, "lead_time": 20, "sales_7d": 14, "moq": 150},
    # 保温杯极光白：大量库存
    "极光白":     {"locked": 20, "in_transit": 500, "safety_stock": 50, "lead_time": 15, "sales_7d": 42, "moq": 500},
    # 保温杯曜石黑
    "曜石黑":     {"locked": 15, "in_transit": 300, "safety_stock": 40, "lead_time": 15, "sales_7d": 35, "moq": 500},
    # 保湿精华30ml：畅销
    "30ml":       {"locked": 12, "in_transit": 200, "safety_stock": 30, "lead_time": 20, "sales_7d": 25, "moq": 200},
    # 保湿精华50ml
    "50ml":       {"locked": 8,  "in_transit": 100, "safety_stock": 20, "lead_time": 20, "sales_7d": 15, "moq": 200},
    # 坚果1kg：爆款 + 有在途
    "1kg":        {"locked": 30, "in_transit": 1000, "safety_stock": 80, "lead_time": 10, "sales_7d": 120, "moq": 1000},
    # 瑜伽垫深紫-6mm
    "深紫-6mm":   {"locked": 20, "in_transit": 600, "safety_stock": 60, "lead_time": 18, "sales_7d": 50, "moq": 500},
}

# ── Agent 决策演示上下文 ──

AGENT_DEMO_DECISIONS = {
    "A5": [
        # 红色预警用例
        {"decision_point": "stock_alert", "context": {"sku_code": "A5-DEMO-RED", "sellable_stock": 5, "locked_stock": 2, "in_transit_stock": 0, "sales_7d": 21, "lead_time_days": 20, "moq": 200, "safety_stock_days": 14}},
        # 黄色预警用例
        {"decision_point": "stock_alert", "context": {"sku_code": "A5-DEMO-YELLOW", "sellable_stock": 50, "locked_stock": 5, "in_transit_stock": 30, "sales_7d": 28, "lead_time_days": 25, "moq": 200, "safety_stock_days": 14}},
        # 绿色正常
        {"decision_point": "stock_alert", "context": {"sku_code": "A5-DEMO-GREEN", "sellable_stock": 500, "locked_stock": 20, "in_transit_stock": 200, "sales_7d": 35, "lead_time_days": 20, "moq": 100, "safety_stock_days": 14}},
    ],
    "G3": [
        # 阻断（低于成本）
        {"decision_point": "discount_check", "context": {"sku_code": "G3-DEMO-BLOCK", "selling_price": 100, "cost_price": 85, "active_discounts": [{"type": "coupon", "value": 10}, {"type": "promotion", "value": 10}], "platform": "amazon"}},
        # 预警（毛利率不足10%）
        {"decision_point": "discount_check", "context": {"sku_code": "G3-DEMO-WARN", "selling_price": 100, "cost_price": 85, "active_discounts": [{"type": "coupon", "value": 8}], "platform": "shopify", "min_margin_threshold": 10}},
        # 放行
        {"decision_point": "discount_check", "context": {"sku_code": "G3-DEMO-ALLOW", "selling_price": 200, "cost_price": 80, "active_discounts": [{"type": "coupon", "value": 15}], "platform": "ozon"}},
    ],
    "A6": [
        # 亏损
        {"decision_point": "profit_check", "context": {"sku_code": "A6-DEMO-LOSS", "selling_price": 80, "cost_price": 60, "platform_fee_rate": 15, "shipping_fee": 18, "ad_cost_per_unit": 10}},
        # 低毛利
        {"decision_point": "profit_check", "context": {"sku_code": "A6-DEMO-LOW", "selling_price": 150, "cost_price": 100, "platform_fee_rate": 12, "shipping_fee": 15, "fixed_fee": 3, "min_margin_threshold": 20}},
        # 正常
        {"decision_point": "profit_check", "context": {"sku_code": "A6-DEMO-OK", "selling_price": 299, "cost_price": 120, "platform_fee_rate": 10, "shipping_fee": 12, "ad_cost_per_unit": 5, "min_margin_threshold": 15}},
    ],
    "A3": [
        # ACOS 过高
        {"decision_point": "acos_analysis", "context": {"campaign_id": "CAM-DEMO-HIGH", "spend": 500, "sales": 1000, "clicks": 200, "impressions": 8000, "conversions": 8, "gross_margin": 30, "target_acos": 25}},
        # 正常
        {"decision_point": "acos_analysis", "context": {"campaign_id": "CAM-DEMO-OK", "spend": 200, "sales": 1200, "clicks": 150, "impressions": 6000, "conversions": 12, "gross_margin": 40, "target_acos": 30}},
    ],
}

# ── 个人规则种子 ──

DEMO_PERSONAL_RULES = [
    {
        "agent_id": "A5", "decision_point": "stock_alert",
        "rule_type": "threshold", "rule_name": "库存红色预警阈值",
        "rule_condition": {"field": "sellable_days", "op": "lt", "value": 5},
        "rule_action": {"override": {"action": "escalate", "notify": "purchase"}},
        "priority": 100, "source": "template",
    },
    {
        "agent_id": "A5", "decision_point": "stock_alert",
        "rule_type": "strategy", "rule_name": "紧急时优先空运",
        "rule_condition": {"field": "stock_status", "op": "eq", "value": "red"},
        "rule_action": {"override": {"suggested_logistics": "空运/国际快递"}},
        "priority": 80, "source": "template",
    },
    {
        "agent_id": "G3", "decision_point": "discount_check",
        "rule_type": "veto", "rule_name": "低于成本绝对阻断",
        "rule_condition": {"field": "action", "op": "eq", "value": "block"},
        "rule_action": {"override": {"blocked": True, "confidence": 1.0}},
        "priority": 100, "source": "template",
    },
    {
        "agent_id": "G3", "decision_point": "discount_check",
        "rule_type": "threshold", "rule_name": "最低毛利率15%",
        "rule_condition": {"field": "gross_margin", "op": "lt", "value": 15},
        "rule_action": {"override": {"action": "warn"}},
        "priority": 90, "source": "template",
    },
    {
        "agent_id": "A6", "decision_point": "profit_check",
        "rule_type": "threshold", "rule_name": "最低毛利率阈值20%",
        "rule_condition": {"field": "gross_margin", "op": "lt", "value": 20},
        "rule_action": {"override": {"action": "warn"}},
        "priority": 90, "source": "template",
    },
    {
        "agent_id": "A6", "decision_point": "profit_check",
        "rule_type": "strategy", "rule_name": "亏损时自动标记",
        "rule_condition": {"field": "is_loss", "op": "eq", "value": True},
        "rule_action": {"override": {"profit_check_status": "block"}},
        "priority": 80, "source": "template",
    },
    {
        "agent_id": "A3", "decision_point": "acos_analysis",
        "rule_type": "threshold", "rule_name": "ACoS 超过毛利率时标记亏损",
        "rule_condition": {"field": "acos_abnormal", "op": "eq", "value": True},
        "rule_action": {"override": {"status": "critical"}},
        "priority": 90, "source": "template",
    },
]


async def seed_product_logistics(session):
    """更新商品物流数据"""
    print("  └─ 更新商品物流数据...")
    for prod_name, log_data in PRODUCT_LOGISTICS.items():
        stmt = select(Product).where(Product.name == prod_name)
        result = await session.execute(stmt)
        product = result.scalar_one_or_none()
        if not product:
            print(f"     → 商品「{prod_name}」不存在，跳过")
            continue
        for k, v in log_data.items():
            setattr(product, k, v)
        print(f"     ✔ {prod_name}")


async def seed_inventory_details(session):
    """更新库存明细（锁定库存、安全库存）"""
    print("  └─ 更新库存明细...")
    skus = (await session.execute(select(Sku))).scalars().all()
    for sku in skus:
        spec_desc = sku.spec_desc or ""
        # 找到匹配的库存更新数据
        matched = None
        for key, data in INVENTORY_UPDATES.items():
            if spec_desc.startswith(key):
                matched = data
                break
        if not matched:
            continue

        inv_stmt = select(Inventory).where(Inventory.sku_id == sku.id)
        inv = (await session.execute(inv_stmt)).scalar_one_or_none()
        if not inv:
            continue

        inv.locked_quantity = matched["locked"]
        inv.safety_stock = matched["safety_stock"]
        print(f"     ✔ SKU {sku.code}: 锁定={inv.locked_quantity}, 安全库存={inv.safety_stock}")


async def create_agent_decisions():
    """通过 API 调用 Agent 决策生成日志"""
    print("  └─ 生成 Agent 决策记录...")
    async with httpx.AsyncClient(base_url=API_BASE, timeout=10) as client:
        for agent_id, decisions in AGENT_DEMO_DECISIONS.items():
            for dec in decisions:
                try:
                    resp = await client.post(
                        f"/agents/{agent_id}/decide",
                        json={
                            "decision_point": dec["decision_point"],
                            "context": dec["context"],
                        },
                    )
                    if resp.status_code == 200:
                        data = resp.json().get("data", {})
                        decision_id = data.get("decision_id")
                        print(f"     ✔ {agent_id}/{dec['decision_point']} → id={decision_id}")
                    else:
                        print(f"     ⚠ {agent_id}/{dec['decision_point']} → {resp.status_code}")
                except Exception as e:
                    print(f"     ❌ {agent_id}/{dec['decision_point']} → {e}")


async def seed_personal_rules(session):
    """创建个人规则种子"""
    print("  └─ 创建个人规则...")
    for rule_data in DEMO_PERSONAL_RULES:
        existing = (await session.execute(
            select(PersonalRule).where(
                PersonalRule.rule_name == rule_data["rule_name"],
            )
        )).scalar_one_or_none()
        if existing:
            print(f"     → 规则「{rule_data['rule_name']}」已存在，跳过")
            continue
        rule = PersonalRule(
            user_id=1,
            **rule_data,
        )
        session.add(rule)
        print(f"     ✔ {rule_data['rule_name']} ({rule_data['rule_type']})")


async def main():
    """主流程"""
    engine = create_async_engine(settings.DATABASE_URL, echo=False)
    session_factory = async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)

    print(f"\n{'='*50}")
    print("  AI Agent 演示数据初始化")
    print(f"{'='*50}\n")
    print(f"📦 数据库: {settings.DATABASE_URL}")
    print()

    async with session_factory() as session:
        try:
            print("📋 第1步：更新商品物流数据")
            await seed_product_logistics(session)

            print("\n📋 第2步：更新库存明细")
            await seed_inventory_details(session)

            print("\n📋 第3步：创建个人规则")
            await seed_personal_rules(session)

            await session.commit()
            print("\n✔ 数据库更新完成")

        except Exception as e:
            await session.rollback()
            print(f"\n❌ 数据库更新失败: {e}")
            raise

    print("\n📋 第4步：通过 API 生成 Agent 决策日志（需要后端运行中）")
    print("   提示：如果后端未运行可跳过此步")
    try:
        await create_agent_decisions()
    except Exception as e:
        print(f"    ⚠ 跳过: {e}")

    print(f"\n{'='*50}")
    print("  ✅ AI Agent 演示数据就绪！")
    print(f"{'='*50}\n")

    await engine.dispose()


if __name__ == "__main__":
    asyncio.run(main())
