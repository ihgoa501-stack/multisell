"""AgentOS action center tests."""

import pytest
from sqlalchemy import select

from app.agentos.models import (
    ActionProposal,
    ApprovalRequest,
    CommandExecution,
    OutcomeReview,
)


@pytest.mark.usefixtures("prepare_db")
class TestActionProposalAPI:
    async def test_create_action_proposal_returns_work_item(self, async_client):
        resp = await async_client.post(
            "/api/agentos/action-proposals",
            json={
                "source_type": "agent",
                "source_id": "A6",
                "agent_id": "A6",
                "squad_id": "risk",
                "action_type": "profit_review",
                "business_object_type": "sku",
                "business_object_id": "SKU-100",
                "title": "复核 SKU-100 利润",
                "description": "实际物流费高于预估，需要复核售价",
                "proposed_payload": {"sku_id": 100, "expected_margin": 0.18},
                "risk_level": "medium",
                "requires_approval": True,
                "confidence": 0.82,
            },
        )

        assert resp.status_code == 200, resp.text
        payload = resp.json()
        assert payload["code"] == 200
        data = payload["data"]
        assert data["id"].startswith("action_proposal:")
        assert data["source_type"] == "action_proposal"
        assert data["status"] == "pending"
        assert data["risk_level"] == "medium"
        assert data["requires_approval"] is True

    async def test_created_proposal_appears_in_work_items(self, async_client):
        create_resp = await async_client.post(
            "/api/agentos/action-proposals",
            json={
                "source_type": "agent",
                "source_id": "A2",
                "agent_id": "A2",
                "squad_id": "growth",
                "action_type": "listing_draft",
                "business_object_type": "product",
                "business_object_id": "P-200",
                "title": "生成 P-200 Listing 草稿",
                "description": "基于利润通过结果生成 Listing 草稿",
                "proposed_payload": {"product_id": 200, "platform": "ozon"},
                "risk_level": "low",
                "requires_approval": False,
                "confidence": 0.9,
            },
        )
        assert create_resp.status_code == 200

        list_resp = await async_client.get(
            "/api/agentos/work-items?source_type=action_proposal",
        )
        assert list_resp.status_code == 200
        records = list_resp.json()["records"]
        assert any(item["title"] == "生成 P-200 Listing 草稿" for item in records)


@pytest.mark.usefixtures("prepare_db")
class TestActionApprovalFlow:
    async def _create_proposal(self, async_client, risk_level="high", requires_approval=True):
        resp = await async_client.post(
            "/api/agentos/action-proposals",
            json={
                "source_type": "agent",
                "source_id": "A5",
                "agent_id": "A5",
                "squad_id": "fulfillment",
                "action_type": "inventory_allocate",
                "business_object_type": "sku",
                "business_object_id": "SKU-300",
                "title": "为 SKU-300 分配库存",
                "description": "库存 Agent 建议为 Ozon 分配 20 件库存",
                "proposed_payload": {"sku_id": 300, "platform": "ozon", "quantity": 20},
                "risk_level": risk_level,
                "requires_approval": requires_approval,
                "confidence": 0.77,
            },
        )
        assert resp.status_code == 200, resp.text
        return resp.json()["data"]["source_id"]

    async def test_approve_proposal_creates_approval_request(self, async_client):
        proposal_id = await self._create_proposal(async_client)

        resp = await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/approve",
            json={"comment": "同意执行"},
        )

        assert resp.status_code == 200, resp.text
        data = resp.json()["data"]
        assert data["ok"] is True
        assert data["proposal"]["status"] == "approved"
        assert data["approval"]["decision"] == "approved"

    async def test_reject_proposal_stores_reason(self, async_client):
        proposal_id = await self._create_proposal(async_client)

        resp = await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/reject",
            json={"comment": "库存不足，先不执行"},
        )

        assert resp.status_code == 200, resp.text
        data = resp.json()["data"]
        assert data["ok"] is True
        assert data["proposal"]["status"] == "rejected"
        assert data["approval"]["decision"] == "rejected"

    async def test_execute_requires_approved_or_low_risk_auto_action(self, async_client):
        proposal_id = await self._create_proposal(async_client)

        blocked = await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/execute",
            json={"executor": "test-runner"},
        )
        assert blocked.status_code == 200
        assert blocked.json()["code"] == 400

        approve = await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/approve",
            json={"comment": "批准执行"},
        )
        assert approve.status_code == 200

        executed = await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/execute",
            json={"executor": "test-runner"},
        )
        assert executed.status_code == 200, executed.text
        data = executed.json()["data"]
        assert data["ok"] is True
        assert data["proposal"]["status"] == "executed"
        assert data["execution"]["status"] == "succeeded"

    async def test_review_executed_proposal(self, async_client):
        proposal_id = await self._create_proposal(async_client, risk_level="low", requires_approval=False)

        executed = await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/execute",
            json={"executor": "test-runner"},
        )
        assert executed.status_code == 200

        reviewed = await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/review",
            json={
                "outcome": "positive",
                "business_metric": "manual_minutes_saved",
                "metric_delta": 12.0,
                "notes": "草稿生成节省了人工整理时间",
            },
        )
        assert reviewed.status_code == 200, reviewed.text
        data = reviewed.json()["data"]
        assert data["ok"] is True
        assert data["proposal"]["status"] == "reviewed"
        assert data["review"]["outcome"] == "positive"


@pytest.mark.usefixtures("prepare_db")
class TestActionCenterPersistence:
    async def test_database_rows_are_created(self, async_client):
        create = await async_client.post(
            "/api/agentos/action-proposals",
            json={
                "source_type": "agent",
                "source_id": "G1",
                "agent_id": "G1",
                "squad_id": "risk",
                "action_type": "daily_report",
                "business_object_type": "dashboard",
                "business_object_id": "daily",
                "title": "生成每日经营日报",
                "description": "总控 Agent 生成经营日报",
                "proposed_payload": {"period": "today"},
                "risk_level": "low",
                "requires_approval": False,
                "confidence": 0.91,
            },
        )
        proposal_id = int(create.json()["data"]["source_id"])

        await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/execute",
            json={"executor": "test-runner"},
        )
        await async_client.post(
            f"/api/agentos/action-proposals/{proposal_id}/review",
            json={
                "outcome": "positive",
                "business_metric": "report_generated",
                "metric_delta": 1,
                "notes": "日报已生成",
            },
        )

        from app.database import async_session_factory

        async with async_session_factory() as db:
            proposal = await db.get(ActionProposal, proposal_id)
            assert proposal is not None
            assert proposal.status == "reviewed"

            approvals = (
                await db.execute(
                    select(ApprovalRequest).where(ApprovalRequest.proposal_id == proposal_id)
                )
            ).scalars().all()
            assert approvals == []

            executions = (
                await db.execute(
                    select(CommandExecution).where(CommandExecution.proposal_id == proposal_id)
                )
            ).scalars().all()
            assert len(executions) == 1

            reviews = (
                await db.execute(
                    select(OutcomeReview).where(OutcomeReview.proposal_id == proposal_id)
                )
            ).scalars().all()
            assert len(reviews) == 1
