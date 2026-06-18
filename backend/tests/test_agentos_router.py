import pytest

from app.database import async_session_factory
from app.agent.models import AgentAction
from app.models import ExceptionItem, Notification


@pytest.mark.asyncio
async def test_agentos_work_items_returns_unified_sources(async_client):
    async with async_session_factory() as db:
        db.add(AgentAction(
            user_id=1,
            agent_id="A5",
            action_type="replenish",
            status="pending",
            summary="SKU-001 建议补货 120 件",
            action_payload={"amount": 1200},
        ))
        db.add(ExceptionItem(
            source_module="settlement",
            source_type="settlement_item",
            source_id=9,
            severity="critical",
            status="open",
            title="结算金额差异",
            description="平台结算与订单金额不一致",
            recommended_action="检查平台费用规则",
        ))
        db.add(Notification(
            user_id=1,
            alert_type="inventory_low_stock",
            title="SKU 2 库存预警",
            content="当前库存 3，安全库存 10",
            severity="warning",
            is_read=0,
            source_id="sku=2",
            link_url="/inventory/2",
        ))
        await db.commit()

    res = await async_client.get("/api/agentos/work-items")
    assert res.status_code == 200
    body = res.json()
    assert body["code"] == 200
    records = body["records"]
    assert {item["source_type"] for item in records} >= {
        "agent_action",
        "exception",
        "notification",
    }
    assert body["total"] >= 3


@pytest.mark.asyncio
async def test_agentos_control_center_shape(async_client):
    res = await async_client.get("/api/agentos/control-center")
    assert res.status_code == 200
    data = res.json()["data"]
    assert set(data.keys()) == {"summary", "work_items", "squads", "templates"}
    assert {"id", "name", "agents"} <= set(data["squads"][0].keys())
    assert {"id", "title", "route"} <= set(data["templates"][0].keys())
