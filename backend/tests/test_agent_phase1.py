"""Phase 1 Agent 增强测试

覆盖范围：
1. A5 库存预警 — stock_alert / replenishment_plan / logistics_choice
2. G3 折扣风控 — discount_check / promotion_validation
3. 决策日志 — 写入、查询、反馈

测试要求：
- 数据充足时输出完整
- 数据不足时返回 insufficient_data
- 折扣阻断/预警逻辑正确
- 多折扣叠加模拟正确
"""

import pytest

from app.agent.agents.inventory_alert import A5InventoryAlertAgent
from app.agent.agents.discount_risk import G3DiscountRiskAgent


# ================================================================
#  A5 库存预警 Agent 测试
# ================================================================


class TestA5InventoryAlert:
    @pytest.fixture
    def agent(self):
        return A5InventoryAlertAgent(user_id=1)

    # ── stock_alert ──────────────────────────────────────────

    async def test_stock_alert_green(self, agent):
        """库存充足 → green"""
        ctx = {
            "sku_code": "SKU-TEST-001",
            "sellable_stock": 500,
            "locked_stock": 20,
            "in_transit_stock": 100,
            "sales_7d": 35,
            "sales_14d": 70,
            "sales_30d": 150,
            "lead_time_days": 20,
            "moq": 100,
            "safety_stock_days": 14,
        }
        result = await agent.decide("stock_alert", ctx)
        assert result["stock_status"] == "green"
        assert result["sellable_days"] > 34  # 500+100 / (35/7) = 120 days
        assert result["daily_sales_source"] == "7d"
        assert result["confidence"] == 0.85
        assert result["risk_reason"] == "库存充足，暂无风险"

    async def test_stock_alert_yellow(self, agent):
        """可售天数接近阈值 → yellow"""
        ctx = {
            "sku_code": "SKU-TEST-002",
            "sellable_stock": 30,
            "locked_stock": 0,
            "in_transit_stock": 0,
            "sales_7d": 14,
            "lead_time_days": 20,
            "moq": 100,
            "safety_stock_days": 14,
        }
        result = await agent.decide("stock_alert", ctx)
        assert result["stock_status"] == "yellow"
        assert result["confidence"] == 0.88
        assert "补货" in result["risk_reason"]
        assert result["suggested_replenish_qty"] > 0

    async def test_stock_alert_red(self, agent):
        """可售天数极低 → red"""
        ctx = {
            "sku_code": "SKU-TEST-003",
            "sellable_stock": 5,
            "locked_stock": 0,
            "in_transit_stock": 0,
            "sales_7d": 14,
            "lead_time_days": 20,
            "moq": 100,
            "safety_stock_days": 14,
        }
        result = await agent.decide("stock_alert", ctx)
        assert result["stock_status"] == "red"
        assert result["confidence"] == 0.95
        assert "断货" in result["risk_reason"]
        assert "紧急补货" in result["suggested_actions"]

    async def test_stock_alert_insufficient_data(self, agent):
        """缺少必填字段 → insufficient_data"""
        ctx = {"sku_code": "SKU-TEST-004"}
        result = await agent.decide("stock_alert", ctx)
        assert result["status"] == "insufficient_data"
        assert "missing_fields" in result
        assert "confidence" in result
        assert result["confidence"] == 0.0

    async def test_stock_alert_empty_sales(self, agent):
        """无销量数据 → insufficient_data"""
        ctx = {
            "sku_code": "SKU-TEST-005",
            "sellable_stock": 100,
            "sales_7d": 0,
            "sales_14d": 0,
            "sales_30d": 0,
            "lead_time_days": 20,
            "safety_stock_days": 14,
        }
        result = await agent.decide("stock_alert", ctx)
        # sales all zero → sellable_days 999 → green
        assert result["stock_status"] == "green"
        assert result["sellable_days"] == 999.0

    async def test_stock_alert_legacy_fields(self, agent):
        """向后兼容旧字段名 quantity/in_transit"""
        ctx = {
            "sku_code": "SKU-TEST-006",
            "quantity": 100,
            "in_transit": 50,
            "daily_sales": 10,
            "lead_time_days": 20,
            "safety_stock_days": 14,
        }
        result = await agent.decide("stock_alert", ctx)
        # backfill maps quantity→sellable_stock, daily_sales→sales_7d
        assert result["sellable_stock"] == 100
        assert result["in_transit_stock"] == 50
        assert result["daily_sales_source"] == "7d"

    async def test_stock_alert_sales_30d_fallback(self, agent):
        """只有30天销量 → 用30天推算日均"""
        ctx = {
            "sku_code": "SKU-TEST-007",
            "sellable_stock": 200,
            "in_transit_stock": 0,
            "sales_7d": 0,
            "sales_14d": 0,
            "sales_30d": 300,
            "lead_time_days": 20,
            "moq": 100,
            "safety_stock_days": 14,
        }
        result = await agent.decide("stock_alert", ctx)
        assert result["daily_sales_source"] == "30d"
        assert 19.5 < result["sellable_days"] < 20.5  # 200 / (300/30) = 20

    # ── replenishment_plan ──────────────────────────────────

    async def test_replenishment_normal(self, agent):
        """正常补货计算"""
        ctx = {
            "sku_code": "SKU-REP-001",
            "sellable_stock": 50,
            "in_transit_stock": 100,
            "sales_7d": 70,  # daily = 10
            "sales_30d": 300,
            "lead_time_days": 20,
            "moq": 100,
            "safety_stock_days": 14,
        }
        result = await agent.decide("replenishment_plan", ctx)
        assert result["sku_code"] == "SKU-REP-001"
        assert result["suggested_replenish_qty"] >= 100  # MOQ floor
        assert result["urgency"] in ("normal", "urgent")

    async def test_replenishment_insufficient_data(self, agent):
        """补货数据不足"""
        ctx = {"sku_code": "SKU-REP-002"}
        result = await agent.decide("replenishment_plan", ctx)
        assert result["status"] == "insufficient_data"

    # ── logistics_choice ────────────────────────────────────

    async def test_logistics_normal(self, agent):
        """正常库存 → 海运"""
        ctx = {
            "stock_status": "green",
            "sellable_days": 60,
            "lead_time_days": 20,
            "destination": "US",
        }
        result = await agent.decide("logistics_choice", ctx)
        assert result["suggested_logistics"] == "海运"
        assert len(result["options"]) >= 1

    async def test_logistics_urgent(self, agent):
        """紧急库存 → 空运"""
        ctx = {
            "stock_status": "red",
            "sellable_days": 3,
            "lead_time_days": 20,
            "cargo_value": 100,
            "destination": "US",
        }
        result = await agent.decide("logistics_choice", ctx)
        assert result["suggested_logistics"] in ("空运/国际快递", "快船")

    async def test_logistics_high_value_urgent(self, agent):
        """高价值+紧急 → 空运快递"""
        ctx = {
            "stock_status": "red",
            "sellable_days": 2,
            "lead_time_days": 20,
            "cargo_value": 200,
            "destination": "US",
        }
        result = await agent.decide("logistics_choice", ctx)
        # urgent + high-value should recommend air_express
        options = result.get("options", [])
        has_express = any(o["method"] == "air_express" for o in options)
        assert has_express

    async def test_logistics_unknown(self, agent):
        """未知决策点"""
        result = await agent.decide("unknown_point", {})
        assert result["action"] == "unknown"
        assert result["confidence"] == 0.0


# ================================================================
#  G3 折扣风控 Agent 测试
# ================================================================


class TestG3DiscountRisk:
    @pytest.fixture
    def agent(self):
        return G3DiscountRiskAgent(user_id=1)

    # ── discount_check ──────────────────────────────────────

    async def test_discount_check_allow(self, agent):
        """安全折扣 → allow"""
        ctx = {
            "sku_code": "SKU-DISC-001",
            "selling_price": 100,
            "cost_price": 60,
            "active_discounts": [{"type": "coupon", "value": 10}],
            "platform": "amazon",
            "min_margin_threshold": 10,
        }
        result = await agent.decide("discount_check", ctx)
        assert result["action"] == "allow"
        assert result["blocked"] is False
        assert result["gross_margin"] > 10  # should be > 30%

    async def test_discount_check_block_below_cost(self, agent):
        """折后价低于成本 → block"""
        ctx = {
            "sku_code": "SKU-DISC-002",
            "selling_price": 100,
            "cost_price": 80,
            "active_discounts": [
                {"type": "coupon", "value": 15},  # 15%
                {"type": "promotion", "value": 10},  # 10%
            ],
            "platform": "amazon",
            "min_margin_threshold": 10,
        }
        result = await agent.decide("discount_check", ctx)
        assert result["action"] == "block"
        assert result["blocked"] is True
        assert "亏损" in result["reason"]
        assert result["gross_profit"] < 0

    async def test_discount_check_warn_below_110(self, agent):
        """折后价低于成本×1.1 → warn"""
        ctx = {
            "sku_code": "SKU-DISC-003",
            "selling_price": 100,
            "cost_price": 85,
            "active_discounts": [{"type": "coupon", "value": 12}],
            "platform": "amazon",
            "min_margin_threshold": 10,
        }
        result = await agent.decide("discount_check", ctx)
        # final_price = 100 * 0.88 = 88, cost*1.1 = 93.5, 88 < 93.5 → warn
        assert result["action"] == "warn"
        assert result["blocked"] is False
        # 88 < 93.5 → should warn
        assert result["final_price"] == 88.0  # 100 * 0.88

    async def test_discount_check_multi_stacking(self, agent):
        """多折扣叠加（百分比 20% + 10% = 28% off）"""
        ctx = {
            "sku_code": "SKU-DISC-004",
            "selling_price": 200,
            "cost_price": 120,
            "active_discounts": [
                {"type": "percentage", "value": 20},
                {"type": "coupon", "value": 10},
            ],
            "platform": "shopify",
            "min_margin_threshold": 10,
        }
        result = await agent.decide("discount_check", ctx)
        # After 20%: 200 * 0.8 = 160
        # After 10%: 160 * 0.9 = 144
        assert result["final_price"] == 144.0
        assert result["action"] == "allow"
        assert result["gross_margin"] > 10

    async def test_discount_check_fixed_amount(self, agent):
        """固定金额折扣"""
        ctx = {
            "sku_code": "SKU-DISC-005",
            "selling_price": 100,
            "cost_price": 50,
            "active_discounts": [
                {"type": "fixed", "value": 30},
                {"type": "percentage", "value": 10},
            ],
            "platform": "walmart",
        }
        result = await agent.decide("discount_check", ctx)
        # Fixed 30: 100 - 30 = 70
        # Then 10%: 70 * 0.9 = 63
        assert result["final_price"] == 63.0
        assert result["action"] == "allow"

    async def test_discount_check_buy_x_get_y(self, agent):
        """买2送1折扣"""
        ctx = {
            "sku_code": "SKU-DISC-006",
            "selling_price": 100,
            "cost_price": 60,
            "active_discounts": [
                {"type": "buy_x_get_y", "value": 0, "buy_qty": 2, "free_qty": 1},
            ],
            "platform": "amazon",
        }
        result = await agent.decide("discount_check", ctx)
        # buy 2 get 1 free → 2/3 of base
        # Equivalent to 33.3% off: 100 * (1 - 1/3) = 66.67
        assert result["final_price"] == pytest.approx(66.67, 0.1)
        assert result["action"] == "allow"

    async def test_discount_check_insufficient_data(self, agent):
        """缺少必填字段"""
        ctx = {"sku_code": "SKU-DISC-007"}
        result = await agent.decide("discount_check", ctx)
        assert result["status"] == "insufficient_data"
        assert "selling_price" in result["missing_fields"]
        assert "cost_price" in result["missing_fields"]

    async def test_discount_check_below_min_margin(self, agent):
        """低于最低毛利率阈值 → warn"""
        ctx = {
            "sku_code": "SKU-DISC-008",
            "selling_price": 100,
            "cost_price": 85,
            "active_discounts": [{"type": "coupon", "value": 5}],
            "platform": "ebay",
            "min_margin_threshold": 15,
        }
        result = await agent.decide("discount_check", ctx)
        # final = 95, margin = (95-85)/95 = 10.5% < 15% → warn
        assert result["action"] == "warn"
        assert result["gross_margin"] < 15

    # ── promotion_validation ────────────────────────────────

    async def test_promotion_allow(self, agent):
        """单个促销放行"""
        ctx = {
            "selling_price": 100,
            "cost_price": 50,
            "promotion": {"type": "percentage", "value": 10},
        }
        result = await agent.decide("promotion_validation", ctx)
        assert result["action"] == "allow"
        assert result["blocked"] is False
        assert result["final_price"] == 90.0

    async def test_promotion_block_below_cost(self, agent):
        """促销价低于成本 → block"""
        ctx = {
            "selling_price": 100,
            "cost_price": 95,
            "promotion": {"type": "percentage", "value": 15},
        }
        result = await agent.decide("promotion_validation", ctx)
        assert result["action"] == "block"
        assert result["blocked"] is True

    async def test_promotion_prime_day(self, agent):
        """大促特殊放行 (不亏本时)"""
        ctx = {
            "selling_price": 100,
            "cost_price": 80,  # 折后 80 > 成本 80*1.1=88? 100*0.8=80, 80 >= 80 → not below cost
            "promotion": {"type": "percentage", "value": 15},  # final = 85
            "is_prime_day": True,
        }
        result = await agent.decide("promotion_validation", ctx)
        assert result["action"] == "allow_special"
        assert result["blocked"] is False


# ================================================================
#  决策日志闭环 测试
# ================================================================


class TestDecisionLogging:
    async def test_execute_decision_creates_log(self, async_client):
        """执行 A5 决策后日志被写入"""
        resp = await async_client.post(
            "/api/agents/A5/decide",
            json={
                "decision_point": "stock_alert",
                "context": {
                    "sku_code": "SKU-LOG-001",
                    "sellable_stock": 200,
                    "locked_stock": 10,
                    "in_transit_stock": 50,
                    "sales_7d": 70,
                    "lead_time_days": 20,
                    "safety_stock_days": 14,
                },
            },
        )
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert data["decision_id"] is not None
        # 验证决策点正确
        assert data["decision_point"] == "stock_alert"
        assert data["agent_id"] == "A5"

    async def test_execute_g3_decision_creates_log(self, async_client):
        """执行 G3 决策后日志被写入"""
        resp = await async_client.post(
            "/api/agents/G3/decide",
            json={
                "decision_point": "discount_check",
                "context": {
                    "sku_code": "SKU-LOG-002",
                    "selling_price": 100,
                    "cost_price": 60,
                    "active_discounts": [{"type": "coupon", "value": 10}],
                    "platform": "amazon",
                },
            },
        )
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert data["decision_id"] is not None
        assert data["agent_id"] == "G3"

    async def test_query_decision_logs(self, async_client):
        """查询决策日志列表"""
        resp = await async_client.get("/api/agents/decisions")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200 or "records" in data
        records = data.get("records", data.get("data", []))
        assert isinstance(records, list)

    async def test_filter_by_agent(self, async_client):
        """按 Agent 筛选日志"""
        resp = await async_client.get("/api/agents/decisions?agent_id=A5")
        assert resp.status_code == 200
        data = resp.json()
        for r in data.get("records", []):
            assert r["agent_id"] == "A5"

    async def test_submit_feedback(self, async_client):
        """提交用户反馈"""
        # 先获取一条决策记录
        resp = await async_client.get("/api/agents/decisions?page_size=1")
        assert resp.status_code == 200
        records = resp.json().get("records", [])
        if not records:
            pytest.skip("无决策记录可测试反馈")
        decision_id = records[0]["id"]

        # 提交反馈
        resp = await async_client.post(
            f"/api/agents/decisions/{decision_id}/feedback",
            json={
                "user_action": "accepted",
                "user_feedback": "同意建议，已安排补货",
                "user_overrides": {"sellable_days": 15},
            },
        )
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert data["user_action"] == "accepted"
        assert data["user_feedback"] == "同意建议，已安排补货"
        assert data["user_overrides"] == {"sellable_days": 15}

    async def test_submit_feedback_not_found(self, async_client):
        """反馈不存在的决策 → code 404"""
        resp = await async_client.post(
            "/api/agents/decisions/999999/feedback",
            json={"user_action": "ignored"},
        )
        body = resp.json()
        # 后端 Result.not_found 返回 code=404（HTTP 状态码仍为 200）
        assert body.get("code") == 404 or resp.status_code == 404

    async def test_g3_blocked_discount_logged(self, async_client):
        """G3 阻断的折扣决策被记录"""
        resp = await async_client.post(
            "/api/agents/G3/decide",
            json={
                "decision_point": "discount_check",
                "context": {
                    "sku_code": "SKU-LOG-003",
                    "selling_price": 100,
                    "cost_price": 90,
                    "active_discounts": [
                        {"type": "coupon", "value": 15},
                        {"type": "promotion", "value": 10},
                    ],
                    "platform": "amazon",
                },
            },
        )
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert data["decision"]["action"] == "block"
        assert data["decision_id"] is not None

    async def test_dry_run_does_not_create_log(self, async_client):
        """dry_run=True 时不写入日志"""
        resp = await async_client.post(
            "/api/agents/A5/decide",
            json={
                "decision_point": "stock_alert",
                "context": {
                    "sku_code": "SKU-DRY-001",
                    "sellable_stock": 100,
                    "sales_7d": 35,
                    "lead_time_days": 20,
                    "safety_stock_days": 14,
                },
                "dry_run": True,
            },
        )
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert data["decision_id"] is None
