"""
Test: A5 LLM integration — verify analyze() and fallback work correctly.

Run from backend/ directory with PYTHONPATH=$PWD.
"""

import asyncio
import os
import sys
import json

# Ensure .env is loaded
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from app.agent.llm_service import AgentLlmService


async def test_llm_analysis():
    """Test 1: LLM 分析能正常工作"""
    print("\n" + "=" * 60)
    print("🧪 Test 1: AgentLlmService.analyze() — DeepSeek")
    print("=" * 60)

    context = {
        "sku_code": "SKU-TEST-001",
        "sellable_stock": 15,
        "locked_stock": 2,
        "in_transit_stock": 0,
        "sales_7d": 42,
        "sales_14d": 85,
        "sales_30d": 180,
        "lead_time_days": 25,
        "safety_stock_days": 14,
        "moq": 100,
        "sellable_days": 2.5,
    }

    result = await AgentLlmService.analyze("A5", "stock_alert", context, db=None)

    if result is None:
        print("❌ analyze() returned None — LLM call failed or no API key")
        print("   Check that LLM_API_KEY is set in backend/.env")
        return False

    print("\n✅ analyze() succeeded!")
    print("\n📋 Raw LLM output:")
    print(json.dumps(result, ensure_ascii=False, indent=2))

    # Validate required fields
    checks = [
        ("risk_level", lambda v: v in ("low", "medium", "high")),
        ("risk_reason", lambda v: isinstance(v, str) and len(v) > 5),
        ("suggested_replenish_qty", lambda v: isinstance(v, (int, float)) and v >= 0),
        ("confidence", lambda v: 0 <= float(v) <= 1),
    ]

    print("\n📋 Field validation:")
    all_ok = True
    for field, check in checks:
        val = result.get(field)
        if val is not None and check(val):
            print("  ✅ {field} = {val}")
        else:
            print("  ❌ {field} = {val} (invalid or missing)")
            all_ok = False

    return all_ok


async def test_formula_fallback():
    """Test 2: A5 formula fallback (no LLM) still works"""
    print("\n" + "=" * 60)
    print("🧪 Test 2: Formula fallback — simulate no LLM")
    print("=" * 60)

    # Simulate what happens when analyze() returns None
    from app.agent.agents.inventory_alert import A5InventoryAlertAgent

    agent = A5InventoryAlertAgent(user_id=1)
    context = {
        "sku_code": "SKU-FALLBACK",
        "sellable_stock": 15,
        "locked_stock": 2,
        "in_transit_stock": 0,
        "sales_7d": 42,
        "lead_time_days": 25,
        "safety_stock_days": 14,
        "moq": 100,
    }

    # Call formula_check directly
    result = agent._formula_check(context)

    required_fields = [
        "stock_status",
        "sellable_days",
        "sellable_stock",
        "suggested_replenish_qty",
        "suggested_logistics",
        "risk_reason",
        "confidence",
    ]
    for f in required_fields:
        if f not in result:
            print("  ❌ Missing field: {f}")
            return False

    print("  ✅ stock_status = {result['stock_status']}")
    print("  ✅ sellable_days = {result['sellable_days']}")
    print("  ✅ suggested_replenish_qty = {result['suggested_replenish_qty']}")
    print("  ✅ suggested_logistics = {result['suggested_logistics']}")
    print("  ✅ confidence = {result['confidence']}")
    print("\n✅ Formula fallback works perfectly")

    return True


async def test_llm_validation():
    """Test 3: _validate_llm_output rejects bad data"""
    print("\n" + "=" * 60)
    print("🧪 Test 3: _validate_llm_output — rejects garbage")
    print("=" * 60)

    from app.agent.agents.inventory_alert import A5InventoryAlertAgent

    agent = A5InventoryAlertAgent(user_id=1)

    # Good output
    good = {
        "risk_level": "high",
        "risk_reason": "库存严重不足，需紧急补货",
        "suggested_replenish_qty": 200,
        "suggested_logistics": "air_freight",
        "confidence": 0.92,
    }
    assert agent._validate_llm_output(good), "Should accept valid output"
    print("  ✅ Good output accepted")

    # Bad risk_level
    assert not agent._validate_llm_output({"risk_level": "超紧急"}), (
        "Should reject invalid risk_level"
    )
    print("  ✅ Invalid risk_level rejected")

    # Bad logistics
    assert not agent._validate_llm_output(
        {
            "risk_level": "high",
            "risk_reason": "test",
            "suggested_logistics": "horse_drawn",
        }
    ), "Should reject invalid logistics"
    print("  ✅ Invalid logistics rejected")

    # Missing risk_reason
    assert not agent._validate_llm_output(
        {
            "risk_level": "high",
            "risk_reason": "",
        }
    ), "Should reject empty risk_reason"
    print("  ✅ Empty risk_reason rejected")

    print("\n✅ All validation tests passed")
    return True


async def test_integration_with_llm():
    """Test 4: Full _check_stock_alert with real LLM (requires API key)"""
    print("\n" + "=" * 60)
    print("🧪 Test 4: Full _check_stock_alert with DeepSeek")
    print("=" * 60)

    from app.agent.agents.inventory_alert import A5InventoryAlertAgent
    from app.agent.base import EvolutionStage

    # SEMI_AUTONOMOUS 阶段 -> 触发 LLM 分析
    agent = A5InventoryAlertAgent(
        user_id=1,
        stage_override={"stock_alert": EvolutionStage.SEMI_AUTONOMOUS},
    )
    context = {
        "sku_code": "SKU-INTEGRATION",
        "sellable_stock": 15,
        "locked_stock": 2,
        "in_transit_stock": 0,
        "sales_7d": 42,
        "sales_14d": 85,
        "sales_30d": 180,
        "lead_time_days": 25,
        "safety_stock_days": 14,
        "moq": 100,
    }

    result = await agent._check_stock_alert(context, db=None)

    print("\n📋 Full result:")
    print(
        json.dumps(
            {k: v for k, v in result.items() if k != "suggested_actions"},
            ensure_ascii=False,
            indent=2,
        )
    )

    checks = [
        ("llm_source", lambda v: isinstance(v, bool)),
        ("stock_status", lambda v: v in ("red", "yellow", "green")),
        ("sellable_days", lambda v: isinstance(v, (int, float))),
        ("risk_reason", lambda v: isinstance(v, str) and len(v) > 5),
        ("ai_explanation", lambda v: isinstance(v, str)),
    ]

    all_ok = True
    for field, check in checks:
        val = result.get(field)
        if check(val):
            print("  ✅ {field} = {val}")
        else:
            print("  ❌ {field} = {val}")
            all_ok = False

    if result.get("llm_source"):
        print("\n🎯 This result was GENERATED by LLM (not formula)")
    else:
        print("\n⚙️  This result fell back to formula (no LLM / API error)")

    return all_ok


async def main():
    tests = [
        ("Formula fallback", test_formula_fallback),
        ("LLM validation", test_llm_validation),
        ("LLM analysis (DeepSeek)", test_llm_analysis),
        ("Full integration", test_integration_with_llm),
    ]

    results = []
    for name, fn in tests:
        try:
            ok = await fn()
            results.append((name, "✅ PASS" if ok else "❌ FAIL"))
        except Exception as e:
            results.append((name, f"💥 ERROR: {e}"))

    print("\n" + "=" * 60)
    print("📊 Summary")
    print("=" * 60)
    for name, status in results:
        print("  {status} — {name}")


if __name__ == "__main__":
    asyncio.run(main())
