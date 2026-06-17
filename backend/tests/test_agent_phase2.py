"""Phase 2-5 Agent 增强测试

覆盖范围：
1. A6 利润监控 — profit_check / cost_optimization
2. G1 运营驾驶舱 — dashboard aggregation
3. A3 广告建议 — acos_analysis / ad_optimization
4. Nudge/Shadow/熵规则健康
"""
import json
import pytest

from app.agent.agents.profit_watch import A6ProfitWatchAgent


# ================================================================
#  G1 运营驾驶舱 测试
# ================================================================

class TestG1Dashboard:

    async def test_dashboard_empty(self, async_client):
        """空数据时返回默认值（或无数据时正常响应）"""
        resp = await async_client.get("/api/agents/dashboard")
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert "summary" in data
        assert "recent_risks" in data
        assert "rule_health" in data

    async def test_dashboard_with_agent_decisions(self, async_client):
        """有决策数据时驾驶舱正常聚合"""
        # 创建一个 A5 红色预警决策
        await async_client.post("/api/agents/A5/decide", json={
            "decision_point": "stock_alert",
            "context": {
                "sku_code": "DASH-SKU-001",
                "sellable_stock": 5,
                "sales_7d": 14,
                "lead_time_days": 20,
                "safety_stock_days": 14,
            },
        })
        # 创建一个 G3 阻断决策
        await async_client.post("/api/agents/G3/decide", json={
            "decision_point": "discount_check",
            "context": {
                "sku_code": "DASH-SKU-002",
                "selling_price": 100,
                "cost_price": 90,
                "active_discounts": [{"type": "coupon", "value": 15}],
                "platform": "amazon",
            },
        })
        # 创建一个 A6 亏损决策
        await async_client.post("/api/agents/A6/decide", json={
            "decision_point": "profit_check",
            "context": {
                "sku_code": "DASH-SKU-003",
                "selling_price": 100,
                "cost_price": 95,
                "platform_fee_rate": 15,
                "shipping_fee": 20,
            },
        })

        resp = await async_client.get("/api/agents/dashboard")
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert data["summary"]["total_decisions_7d"] >= 3
        assert data["summary"]["active_risks"] >= 3  # A5 red + G3 block + A6 loss
        assert len(data["recent_risks"]) >= 3

    async def test_dashboard_g1_listed(self, async_client):
        """G1 出现在 Agent 列表"""
        resp = await async_client.get("/api/agents")
        assert resp.status_code == 200
        agents = resp.json().get("data", [])
        ids = [a["agent_id"] for a in agents]
        assert "G1" in ids


# ================================================================
#  A6 利润监控 Agent 测试
# ================================================================

class TestA6ProfitWatch:

    @pytest.fixture
    def agent(self):
        return A6ProfitWatchAgent(user_id=1)

    # ── profit_check ─────────────────────────────────────────

    async def test_profit_profitable(self, agent):
        """正常毛利 → allow"""
        ctx = {
            "sku_code": "SKU-PROFIT-001",
            "selling_price": 100,
            "cost_price": 40,
            "platform_fee_rate": 10,  # 10%
            "shipping_fee": 15,
            "fixed_fee": 2,
        }
        result = await agent.decide("profit_check", ctx)
        assert result["profit_check_status"] == "allow"
        assert result["is_loss"] is False
        assert result["below_threshold"] is False
        assert result["gross_margin"] > 15
        assert "fee_breakdown" in result

    async def test_profit_loss(self, agent):
        """亏损 → block"""
        ctx = {
            "sku_code": "SKU-PROFIT-002",
            "selling_price": 100,
            "cost_price": 90,
            "platform_fee_rate": 15,
            "shipping_fee": 20,
            "fixed_fee": 5,
        }
        result = await agent.decide("profit_check", ctx)
        assert result["profit_check_status"] == "block"
        assert result["is_loss"] is True
        assert result["profit_per_unit"] < 0
        assert "亏损" in result["anomaly_reason"]

    async def test_profit_below_threshold(self, agent):
        """低毛利低于阈值 → warn"""
        ctx = {
            "sku_code": "SKU-PROFIT-003",
            "selling_price": 100,
            "cost_price": 70,
            "platform_fee_rate": 15,
            "shipping_fee": 10,
            "fixed_fee": 2,
            "min_margin_threshold": 20,
        }
        result = await agent.decide("profit_check", ctx)
        # revenue=100, cost=70, pf=15, ship=10, fixed=2 → total fees=27
        # profit = 100-70-27 = 3, margin = 3%
        assert result["profit_check_status"] == "warn"
        assert result["is_loss"] is False
        assert result["below_threshold"] is True
        assert result["gross_margin"] < 20

    async def test_profit_with_discounts(self, agent):
        """含折扣的利润计算"""
        ctx = {
            "sku_code": "SKU-PROFIT-004",
            "selling_price": 100,
            "cost_price": 40,
            "platform_fee_rate": 10,
            "shipping_fee": 10,
            "discounts": [
                {"type": "coupon", "value": 10},
                {"type": "promotion", "value": 5},
            ],
        }
        result = await agent.decide("profit_check", ctx)
        # discount = 15%, effective_revenue = 85
        # pf = 10, ship = 10, discount = 15, total fees = 35
        # profit = 85 - 40 - 35 = 10
        assert result["effective_revenue"] == 85.0
        assert result["discount_rate"] == 15.0
        assert result["profit_per_unit"] == 10.0

    async def test_profit_with_ad_cost(self, agent):
        """含广告成本的利润计算"""
        ctx = {
            "sku_code": "SKU-PROFIT-005",
            "selling_price": 100,
            "cost_price": 50,
            "platform_fee_rate": 10,
            "shipping_fee": 10,
            "ad_cost_per_unit": 15,
        }
        result = await agent.decide("profit_check", ctx)
        # profit = 100 - 50 - 10(pf) - 10(ship) - 15(ad) = 15
        assert result["profit_per_unit"] == 15.0
        assert "ad_cost" in result["fee_breakdown"]
        assert result["fee_breakdown"]["ad_cost"] == 15.0

    async def test_profit_with_refund(self, agent):
        """含退款成本的利润计算"""
        ctx = {
            "sku_code": "SKU-PROFIT-006",
            "selling_price": 100,
            "cost_price": 50,
            "platform_fee_rate": 10,
            "shipping_fee": 10,
            "refund_rate": 5,
        }
        result = await agent.decide("profit_check", ctx)
        # refund = 100 * 0.05 = 5
        assert result["fee_breakdown"]["refund_cost"] == 5.0

    async def test_profit_fee_breakdown(self, agent):
        """费用拆分完整性"""
        ctx = {
            "sku_code": "SKU-PROFIT-007",
            "selling_price": 200,
            "cost_price": 80,
            "platform_fee_rate": 12,
            "shipping_fee": 25,
            "fixed_fee": 3,
            "ad_cost_per_unit": 10,
            "refund_rate": 3,
            "discounts": [{"type": "coupon", "value": 8}],
        }
        result = await agent.decide("profit_check", ctx)
        fees = result["fee_breakdown"]
        assert "platform_fee" in fees
        assert "shipping_fee" in fees
        assert "fixed_fee" in fees
        assert "discount" in fees
        assert "ad_cost" in fees
        assert "refund_cost" in fees
        assert "total" in fees
        # Verify total = sum of all fees
        expected_total = (fees["platform_fee"] + fees["shipping_fee"] + fees["fixed_fee"]
                          + fees["discount"] + fees["ad_cost"] + fees["refund_cost"])
        assert fees["total"] == round(expected_total, 2)

    async def test_profit_insufficient_data(self, agent):
        """缺少必填字段"""
        ctx = {"sku_code": "SKU-PROFIT-008"}
        result = await agent.decide("profit_check", ctx)
        assert result["status"] == "insufficient_data"
        assert "selling_price" in result["missing_fields"]
        assert "cost_price" in result["missing_fields"]

    async def test_profit_optimization_suggestions(self, agent):
        """亏损时有优化建议"""
        ctx = {
            "sku_code": "SKU-PROFIT-009",
            "selling_price": 50,
            "cost_price": 40,
            "platform_fee_rate": 15,
            "shipping_fee": 12,
        }
        result = await agent.decide("profit_check", ctx)
        assert "optimization_suggestions" in result
        assert len(result["optimization_suggestions"]) > 0

    async def test_profit_fee_warnings(self, agent):
        """费用占比过高时触发警告"""
        ctx = {
            "sku_code": "SKU-PROFIT-010",
            "selling_price": 100,
            "cost_price": 40,
            "platform_fee_rate": 10,
            "shipping_fee": 30,  # 占 30% 营收，超过 25% 阈值
        }
        result = await agent.decide("profit_check", ctx)
        assert "fee_warnings" in result
        # shipping_fee 30 > 100*0.25 = 25 → warning
        assert len(result["fee_warnings"]) > 0

    # ── cost_optimization ───────────────────────────────────

    async def test_cost_optimization_below_target(self, agent):
        """低于目标毛利率时有提价/降本建议"""
        ctx = {
            "sku_code": "SKU-OPT-001",
            "selling_price": 100,
            "cost_price": 85,
            "target_margin": 20,
        }
        result = await agent.decide("cost_optimization", ctx)
        assert result["current_margin"] == 15.0
        assert len(result["suggestions"]) > 0
        assert result["suggestions"][0]["type"] == "price_increase"

    async def test_cost_optimization_insufficient(self, agent):
        """缺少必填字段"""
        ctx = {}
        result = await agent.decide("cost_optimization", ctx)
        assert result["status"] == "insufficient_data"


# ================================================================
#  A6 决策日志集成测试
# ================================================================

class TestA6DecisionLogging:

    async def test_a6_decision_creates_log(self, async_client):
        """A6 利润决策写入 agent_decision"""
        resp = await async_client.post(
            "/api/agents/A6/decide",
            json={
                "decision_point": "profit_check",
                "context": {
                    "sku_code": "SKU-A6-LOG",
                    "selling_price": 100,
                    "cost_price": 50,
                    "platform_fee_rate": 10,
                    "shipping_fee": 10,
                },
            },
        )
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert data["decision_id"] is not None
        assert data["agent_id"] == "A6"

    async def test_a6_listed_as_agent(self, async_client):
        """A6 出现在 Agent 列表"""
        resp = await async_client.get("/api/agents")
        assert resp.status_code == 200
        agents = resp.json().get("data", [])
        ids = [a["agent_id"] for a in agents]
        assert "A6" in ids


# ================================================================
#  A3 广告建议 Agent 测试
# ================================================================

class TestA3AdAdvice:

    @pytest.fixture
    def agent(self):
        from app.agent.agents.ad_advice import A3AdAdviceAgent
        return A3AdAdviceAgent(user_id=1)

    # ── acos_analysis ────────────────────────────────────────

    async def test_acos_normal(self, agent):
        """ACoS 正常 → normal"""
        ctx = {
            "campaign_id": "CAM-001",
            "spend": 200,
            "sales": 1000,
            "clicks": 100,
            "impressions": 5000,
            "conversions": 10,
        }
        result = await agent.decide("acos_analysis", ctx)
        assert result["status"] == "normal"
        assert result["metrics"]["acos"] == 20.0
        assert result["acos_abnormal"] is False

    async def test_acos_critical(self, agent):
        """ACoS 超过毛利率 → critical"""
        ctx = {
            "campaign_id": "CAM-002",
            "spend": 600,
            "sales": 1000,
            "gross_margin": 25,
            "target_acos": 30,
        }
        result = await agent.decide("acos_analysis", ctx)
        assert result["status"] == "critical"
        assert result["acos_abnormal"] is True
        # ACOS=60% > gross_margin=25%
        assert "亏损" in result["alerts"][0]["message"]

    async def test_acos_warning(self, agent):
        """ACoS 超过目标但低于毛利率 → warning"""
        ctx = {
            "campaign_id": "CAM-003",
            "spend": 350,
            "sales": 1000,
            "gross_margin": 50,
            "target_acos": 30,
        }
        result = await agent.decide("acos_analysis", ctx)
        assert result["status"] == "warning"
        assert result["acos_abnormal"] is True
        # ACOS=35% > target=30%, but 35% < gross_margin=50%

    async def test_acos_budget_usage(self, agent):
        """预算使用率异常检测"""
        ctx = {
            "campaign_id": "CAM-004",
            "spend": 95,
            "sales": 300,
            "budget": 100,
        }
        result = await agent.decide("acos_analysis", ctx)
        assert result["metrics"]["budget_usage"] == 95.0
        # budget_usage > 90 → should have alert
        assert any("预算" in a["message"] for a in result["alerts"])

    async def test_acos_inventory_risk(self, agent):
        """库存不足时建议暂停广告"""
        ctx = {
            "campaign_id": "CAM-005",
            "spend": 100,
            "sales": 300,
            "inventory_status": "out_of_stock",
        }
        result = await agent.decide("acos_analysis", ctx)
        assert any("库存" in a["message"] for a in result["alerts"])
        assert any("暂停" in s for s in result["suggestions"])

    async def test_acos_bid_suggestion(self, agent):
        """ACoS 异常时有出价建议"""
        ctx = {
            "campaign_id": "CAM-006",
            "spend": 500,
            "sales": 1000,
            "clicks": 100,
            "target_acos": 30,
        }
        result = await agent.decide("acos_analysis", ctx)
        # ACOS=50% > target=30% → should have bid suggestion
        assert result["bid_suggestion"] is not None
        assert result["bid_suggestion"]["current_cpc"] == 5.0
        assert result["bid_suggestion"]["suggested_cpc"] < 5.0

    async def test_acos_insufficient_data(self, agent):
        """缺少必填字段"""
        ctx = {"campaign_id": "CAM-007"}
        result = await agent.decide("acos_analysis", ctx)
        assert result["status"] == "insufficient_data"
        assert "spend" in result["missing_fields"]
        assert "sales" in result["missing_fields"]

    # ── ad_optimization ─────────────────────────────────────

    async def test_ad_optimization_negative_keywords(self, agent):
        """搜索词分析 → 否定关键词建议"""
        ctx = {
            "campaign_id": "CAM-OPT-001",
            "spend": 500,
            "sales": 1000,
            "clicks": 100,
            "target_acos": 30,
            "search_terms": [
                {"keyword": "cheap product", "spend": 200, "sales": 100, "clicks": 50},
                {"keyword": "good deal", "spend": 50, "sales": 200, "clicks": 5},
            ],
        }
        result = await agent.decide("ad_optimization", ctx)
        # "cheap product": ACOS=200%, clicks=50 ≥ 10 → should be in negative keywords
        neg_items = [i for i in result["optimization_items"] if i["type"] == "negative_keyword"]
        assert len(neg_items) > 0

    async def test_ad_optimization_insufficient(self, agent):
        """缺少必填字段"""
        ctx = {}
        result = await agent.decide("ad_optimization", ctx)
        assert result["status"] == "insufficient_data"

    # ── 集成测试 ────────────────────────────────────────────

    async def test_a3_decision_creates_log(self, async_client):
        """A3 广告决策写入 agent_decision"""
        resp = await async_client.post(
            "/api/agents/A3/decide",
            json={
                "decision_point": "acos_analysis",
                "context": {
                    "campaign_id": "CAM-LOG-001",
                    "spend": 200,
                    "sales": 800,
                },
            },
        )
        assert resp.status_code == 200
        data = resp.json().get("data", {})
        assert data["decision_id"] is not None
        assert data["agent_id"] == "A3"

    async def test_a3_listed_as_agent(self, async_client):
        """A3 出现在 Agent 列表"""
        resp = await async_client.get("/api/agents")
        assert resp.status_code == 200
        agents = resp.json().get("data", [])
        ids = [a["agent_id"] for a in agents]
        assert "A3" in ids
