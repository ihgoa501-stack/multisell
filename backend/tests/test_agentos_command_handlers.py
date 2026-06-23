"""Unit tests for AgentOS command handlers — services mocked at the real module path."""

from unittest.mock import AsyncMock, patch

import pytest


class TestHandleProfitReview:
    async def test_success(self):
        mock_resp = type(
            "R",
            (),
            {
                "model_dump": lambda s: {
                    "profit_amount": 850.0,
                    "recommendation": "approve",
                }
            },
        )()
        with patch(
            "app.decision.service.PreListingDecisionService.calculate",
            new=AsyncMock(return_value=mock_resp),
        ):
            from app.agentos.command_handlers import handle_profit_review

            result = await handle_profit_review(
                None, {"sku_id": 1, "target_sale_price": 5000.0}
            )
        assert result["profit_amount"] == 850.0
        assert result["recommendation"] == "approve"

    async def test_missing_sku_id_raises(self):
        from app.agentos.command_handlers import handle_profit_review

        with pytest.raises(KeyError):
            await handle_profit_review(None, {"target_sale_price": 100})


class TestHandleInventoryAllocate:
    async def test_manual(self):
        with patch(
            "app.allocation.service.AllocationService.allocate",
            new=AsyncMock(return_value={"sku_id": 1, "allocated_qty": 50}),
        ):
            from app.agentos.command_handlers import handle_inventory_allocate

            result = await handle_inventory_allocate(
                None, {"sku_id": 1, "warehouse_id": 2, "quantity": 50}
            )
        assert result["mode"] == "manual"
        assert result["allocations"][0]["allocated_qty"] == 50

    async def test_auto(self):
        with patch(
            "app.allocation.service.AllocationService.auto_allocate",
            new=AsyncMock(return_value=[{"sku_id": 1, "allocated_qty": 30}]),
        ):
            from app.agentos.command_handlers import handle_inventory_allocate

            result = await handle_inventory_allocate(None, {"sku_id": 1})
        assert result["mode"] == "auto"

    async def test_missing_sku_id_raises(self):
        from app.agentos.command_handlers import handle_inventory_allocate

        with pytest.raises(KeyError):
            await handle_inventory_allocate(None, {})


class TestHandleListingDraft:
    async def test_success(self):
        mock_listing = type("L", (), {"id": 10, "published_data": None})()
        mock_db = type(
            "DB",
            (),
            {
                "get": AsyncMock(
                    return_value=type("P", (), {"id": 1, "name": "测试"})()
                ),
                "flush": AsyncMock(),
            },
        )()

        with (
            patch(
                "app.agent.agents.listing_optimizer.A2ListingOptimizerAgent.decide",
                new=AsyncMock(
                    return_value={
                        "title": "Test",
                        "bullets": ["B1"],
                        "search_terms": ["K1"],
                    }
                ),
            ),
            patch(
                "app.listing.service.ListingService._get_or_create_listing",
                new=AsyncMock(return_value=mock_listing),
            ),
        ):
            from app.agentos.command_handlers import handle_listing_draft

            result = await handle_listing_draft(
                mock_db, {"product_id": 1, "platform_id": 5}
            )
        assert result["optimization"]["title"] == "Test"
        assert result["listing_id"] == 10

    async def test_no_product_id(self):
        from app.agentos.command_handlers import handle_listing_draft

        with pytest.raises(ValueError, match="product_id is required"):
            await handle_listing_draft(None, {})

    async def test_product_not_found(self):
        db = type("DB", (), {"get": AsyncMock(return_value=None)})()
        from app.agentos.command_handlers import handle_listing_draft

        with pytest.raises(ValueError, match="not found"):
            await handle_listing_draft(db, {"product_id": 999})


class TestHandleDailyReport:
    async def test_success(self):
        with patch(
            "app.dashboard.service.DashboardService.get_dashboard",
            new=AsyncMock(return_value={"products": {"total": 7}}),
        ):
            from app.agentos.command_handlers import handle_daily_report

            result = await handle_daily_report(None, {})
        assert result["products"]["total"] == 7


class TestHandleNotify:
    async def test_success(self):
        with (
            patch(
                "app.notification.service.NotificationService.check_and_create_alerts",
                new=AsyncMock(return_value={"low_stock": 2}),
            ),
            patch(
                "app.notification.service.NotificationService.get_unread_count",
                new=AsyncMock(return_value={"total": 5}),
            ),
        ):
            from app.agentos.command_handlers import handle_notify

            result = await handle_notify(None, {})
        assert result["alerts_created"]["low_stock"] == 2
        assert result["unread_summary"]["total"] == 5


class TestHandlerMap:
    def test_all_registered(self):
        from app.agentos.command_handlers import HANDLER_MAP

        for name in [
            "profit_review",
            "inventory_allocate",
            "listing_draft",
            "daily_report",
            "notify",
        ]:
            assert name in HANDLER_MAP
            assert callable(HANDLER_MAP[name])
