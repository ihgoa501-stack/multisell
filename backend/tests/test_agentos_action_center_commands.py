"""Action center business command adapter tests."""

import pytest

from app.agentos.action_center_service import resolve_command_adapter


def test_low_risk_internal_commands_are_supported():
    adapter = resolve_command_adapter("daily_report")
    assert adapter["command_name"] == "daily_report"
    assert adapter["execution_mode"] == "record_only"


def test_high_risk_unknown_command_is_blocked():
    with pytest.raises(ValueError) as exc:
        resolve_command_adapter("delete_platform_credentials")
    assert "not registered" in str(exc.value)


def test_listing_draft_is_registered():
    adapter = resolve_command_adapter("listing_draft")
    assert adapter["command_name"] == "listing_draft"


def test_profit_review_is_registered():
    adapter = resolve_command_adapter("profit_review")
    assert adapter["command_name"] == "profit_review"


def test_inventory_allocate_is_registered():
    adapter = resolve_command_adapter("inventory_allocate")
    assert adapter["command_name"] == "inventory_allocate"


def test_notify_is_registered():
    adapter = resolve_command_adapter("notify")
    assert adapter["command_name"] == "notify"


def test_all_adapters_are_record_only():
    for name in ["daily_report", "listing_draft", "profit_review", "inventory_allocate", "notify"]:
        adapter = resolve_command_adapter(name)
        assert adapter["execution_mode"] == "record_only"
