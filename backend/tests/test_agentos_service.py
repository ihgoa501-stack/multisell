from datetime import datetime, timezone
from types import SimpleNamespace

from app.agentos.service import AgentOSService, AGENT_SQUADS, TEMPLATE_CARDS


def test_normalize_agent_pending_action_to_work_item():
    row = SimpleNamespace(
        id=11,
        agent_id="A5",
        decision_id=7,
        action_type="replenish",
        status="pending",
        summary="SKU-001 建议补货 120 件",
        action_payload={"sku_id": 1, "amount": 1200},
        created_at=datetime(2026, 6, 18, 8, 0, tzinfo=timezone.utc),
    )

    item = AgentOSService.normalize_agent_pending_action(row)

    assert item["id"] == "agent_action:11"
    assert item["source_type"] == "agent_action"
    assert item["source_id"] == "11"
    assert item["squad"] == "fulfillment"
    assert item["agent_id"] == "A5"
    assert item["title"] == "SKU-001 建议补货 120 件"
    assert item["risk_level"] == "high"
    assert item["approval_required"] is True
    assert item["status"] == "pending"
    assert item["audit_link"] == "/agents/A5"


def test_normalize_exception_to_work_item():
    row = SimpleNamespace(
        id=21,
        source_module="settlement",
        source_type="settlement_item",
        source_id=9,
        severity="critical",
        status="open",
        title="结算金额差异",
        description="平台结算与订单金额不一致",
        recommended_action="检查平台费用规则",
        created_at=datetime(2026, 6, 18, 9, 0, tzinfo=timezone.utc),
    )

    item = AgentOSService.normalize_exception(row)

    assert item["id"] == "exception:21"
    assert item["source_type"] == "exception"
    assert item["business_object"]["type"] == "settlement_item"
    assert item["squad"] == "risk"
    assert item["risk_level"] == "critical"
    assert item["approval_required"] is False
    assert item["status"] == "open"


def test_normalize_notification_to_work_item():
    row = SimpleNamespace(
        id=31,
        alert_type="inventory_low_stock",
        title="SKU 2 库存预警",
        content="当前库存 3，安全库存 10",
        link_url="/inventory/2",
        severity="warning",
        is_read=False,
        source_id="sku=2",
        created_at=datetime(2026, 6, 18, 10, 0, tzinfo=timezone.utc),
    )

    item = AgentOSService.normalize_notification(row)

    assert item["id"] == "notification:31"
    assert item["source_type"] == "notification"
    assert item["squad"] == "fulfillment"
    assert item["risk_level"] == "medium"
    assert item["status"] == "unread"
    assert item["audit_link"] == "/inventory/2"


def test_squad_and_template_constants_are_phase_1_specific():
    assert [s["id"] for s in AGENT_SQUADS] == ["growth", "fulfillment", "risk"]
    assert {t["id"] for t in TEMPLATE_CARDS} >= {
        "pre_listing_decision",
        "listing_optimization",
        "inventory_replenishment",
        "profit_risk",
    }
