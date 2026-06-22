"""Unit tests for order sync worker — mapping function only (no DB)."""

from decimal import Decimal
from app.order_import.sync_worker import map_platform_order


def test_map_platform_order_to_local():
    platform_order = {
        "order_sn": "OZON-123",
        "status": "delivered",
        "total_amount": "29.99",
        "shipping_fee": "5.00",
        "paid_at": "2026-06-20T10:00:00Z",
        "recipient_name": "Alice",
        "recipient_phone": "123456",
        "shipping_address": "Moscow, Red Square 1",
        "items": [{"sku_code": "SKU001", "quantity": 2, "unit_price": "14.995"}],
    }
    local = map_platform_order(platform_order, platform_id=1)
    assert local["order_no"] == "OZON-123"
    assert local["status"] == "delivered"
    assert local["total_amount"] == Decimal("29.99")
    assert local["shipping_fee"] == Decimal("5.00")
    assert local["pay_amount"] == Decimal("34.99")
    assert local["recipient_name"] == "Alice"
    assert local["recipient_phone"] == "123456"
    assert local["shipping_address"] == "Moscow, Red Square 1"
    assert len(local["items"]) == 1
    assert local["items"][0]["sku_code"] == "SKU001"
    assert local["items"][0]["quantity"] == 2
    assert local["items"][0]["unit_price"] == Decimal("14.995")
    assert local["items"][0]["subtotal"] == Decimal("29.990")


def test_map_platform_order_status_mapping():
    cases = [
        ("delivered", "delivered"),
        ("shipped", "shipped"),
        ("ready_to_ship", "paid"),
        ("cancelled", "cancelled"),
        ("unknown_status", "pending"),
        ("", "pending"),
    ]
    for platform_status, expected in cases:
        local = map_platform_order({"order_sn": "T1", "status": platform_status, "items": []}, platform_id=1)
        assert local["status"] == expected, f"{platform_status} -> {expected}"


def test_map_platform_order_no_items():
    local = map_platform_order({"order_sn": "EMPTY", "items": []}, platform_id=1)
    assert local["total_amount"] == Decimal("0")
    assert local["pay_amount"] == Decimal("0")
    assert local["items"] == []


def test_map_platform_order_paid_at_none():
    local = map_platform_order({"order_sn": "NO-PAY", "items": []}, platform_id=1)
    assert local["paid_at"] is None
