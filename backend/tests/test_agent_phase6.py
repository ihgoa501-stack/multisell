"""Phase 6 — 剩余 Agent（A1/A2/A4/A7/G2）测试"""

import pytest


class TestA1ProductScout:
    @pytest.fixture
    def agent(self):
        from app.agent.agents.product_scout import A1ProductScoutAgent

        return A1ProductScoutAgent(user_id=1)

    async def test_scout_with_candidates(self, agent):
        ctx = {
            "category": "Baby Feeding",
            "marketplace": "US",
            "candidates": [
                {
                    "name": "Silicone Plate",
                    "price": 19.99,
                    "cost": 4.5,
                    "search_volume": 15000,
                    "trend_growth": 120,
                    "review_count": 200,
                },
                {
                    "name": "Baby Spoon Set",
                    "price": 12.99,
                    "cost": 3.0,
                    "search_volume": 8000,
                    "trend_growth": 80,
                    "review_count": 500,
                },
            ],
        }
        result = await agent.decide("product_scout", ctx)
        assert "candidates" in result
        assert len(result["candidates"]) == 2
        assert result["candidates"][0]["score"] >= result["candidates"][1]["score"]

    async def test_scout_insufficient(self, agent):
        result = await agent.decide("product_scout", {"category": "test"})
        assert result["status"] == "insufficient_data"

    async def test_market_analysis(self, agent):
        result = await agent.decide(
            "market_analysis", {"category": "Toys", "trend": "growing"}
        )
        assert result["trend_direction"] == "growing"


class TestA2ListingOptimizer:
    @pytest.fixture
    def agent(self):
        from app.agent.agents.listing_optimizer import A2ListingOptimizerAgent

        return A2ListingOptimizerAgent(user_id=1)

    async def test_optimize_with_keywords(self, agent):
        ctx = {
            "product_name": "Silicone Baby Plate",
            "marketplace": "US",
            "features": [
                "Suction base",
                "Food-grade silicone",
                "BPA free",
                "3 compartments",
            ],
            "keywords": [
                {"word": "baby plate", "volume": 15000},
                {"word": "silicone plate", "volume": 12000},
                {"word": "suction plate", "volume": 8000},
            ],
        }
        result = await agent.decide("listing_optimize", ctx)
        assert "baby plate" in result["title"].lower()
        assert len(result["bullets"]) >= 3
        assert len(result["search_terms"]) > 0

    async def test_optimize_insufficient(self, agent):
        result = await agent.decide("listing_optimize", {"product_name": "test"})
        assert result["status"] == "insufficient_data"

    async def test_keyword_research(self, agent):
        result = await agent.decide(
            "keyword_research", {"seed_keywords": ["baby", "plate"]}
        )
        assert result["total_found"] > 0


class TestA4CustomerService:
    @pytest.fixture
    def agent(self):
        from app.agent.agents.customer_service import A4CustomerServiceAgent

        return A4CustomerServiceAgent(user_id=1)

    async def test_auto_reply_shipping(self, agent):
        result = await agent.decide(
            "auto_reply",
            {
                "message": "Where is my order?",
                "language": "en",
                "order_context": {"estimated_delivery_days": "3-5"},
            },
        )
        assert result["action"] == "auto_reply"
        assert result["auto_reply"] is not None

    async def test_auto_reply_high_risk(self, agent):
        result = await agent.decide(
            "auto_reply",
            {
                "message": "I want to file a trademark lawsuit",
                "language": "en",
            },
        )
        assert result["action"] == "escalate"

    async def test_intent_classify(self, agent):
        result = await agent.decide(
            "intent_classify", {"message": "How to return this item", "language": "en"}
        )
        assert result["intent"] in ("return", "unknown")

    async def test_insufficient(self, agent):
        result = await agent.decide("auto_reply", {"language": "en"})
        assert result["status"] == "insufficient_data"


class TestA7Compliance:
    @pytest.fixture
    def agent(self):
        from app.agent.agents.compliance import A7ComplianceGuardAgent

        return A7ComplianceGuardAgent(user_id=1)

    async def test_compliance_electronics_us(self, agent):
        result = await agent.decide(
            "compliance_check",
            {
                "product_name": "Wireless Earbuds",
                "category": "electronics",
                "target_country": "US",
                "target_platform": "amazon",
            },
        )
        assert "FCC" in result["required_certifications"]
        assert result["risk_level"] == "medium"

    async def test_compliance_baby_eu(self, agent):
        result = await agent.decide(
            "compliance_check",
            {
                "product_name": "Baby Toy",
                "category": "baby",
                "target_country": "EU",
                "target_platform": "amazon",
            },
        )
        assert "CE" in result["required_certifications"]
        assert result["risk_level"] == "high"

    async def test_compliance_insufficient(self, agent):
        result = await agent.decide("compliance_check", {"product_name": "test"})
        assert result["status"] == "insufficient_data"

    async def test_certification_lookup(self, agent):
        result = await agent.decide(
            "certification_lookup", {"certification": "FCC", "country": "US"}
        )
        assert "FCC" in result["description"]


class TestG2WarehouseCustoms:
    @pytest.fixture
    def agent(self):
        from app.agent.agents.warehouse_customs import G2WarehouseCustomsAgent

        return G2WarehouseCustomsAgent(user_id=1)

    async def test_clearance_electronics_us(self, agent):
        result = await agent.decide(
            "customs_clearance",
            {
                "product_name": "Bluetooth Speaker",
                "destination_country": "US",
                "cargo_type": "electronics",
                "declared_value": 1500,
                "weight_kg": 0.8,
            },
        )
        assert "Commercial Invoice" in result["required_documents"]
        assert result["hs_code"] != "需要人工归类"

    async def test_clearance_insufficient(self, agent):
        result = await agent.decide("customs_clearance", {"product_name": "test"})
        assert result["status"] == "insufficient_data"

    async def test_warehouse_high_volume(self, agent):
        result = await agent.decide(
            "warehouse_advice",
            {
                "destination_country": "US",
                "monthly_sales_volume": 1000,
            },
        )
        assert "双轨" in result["strategy"]

    async def test_warehouse_low_volume(self, agent):
        result = await agent.decide(
            "warehouse_advice",
            {
                "destination_country": "US",
                "monthly_sales_volume": 50,
            },
        )
        assert "直发" in result["strategy"] or "自配送" in result["note"]
