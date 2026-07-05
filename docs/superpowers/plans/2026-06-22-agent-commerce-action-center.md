> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Agent Commerce Action Center Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Phase 2 of LingMirror Agent Commerce OS: a persistent action center that turns human operations and Agent proposals into auditable WorkItems with approval, execution, and outcome review.

**Architecture:** Keep the current AgentOS aggregation layer, but add persistent action-center models under `backend/app/agentos/`. Existing sources (`ExceptionItem`, `Notification`, `ListingTask`, `AgentAction`) remain valid WorkItem inputs; new `ActionProposal` records become the canonical path for Agent-originated commerce actions. Approval and execution never bypass policy checks, operation logs, or source-object state transitions.

**Tech Stack:** FastAPI, SQLAlchemy 2.0 async, PostgreSQL JSON columns, Alembic migrations, Pydantic schemas, Vue 3, TypeScript, Naive UI, pytest/httpx ASGI tests.

---

## Current Context

Existing useful pieces:

- `backend/app/agentos/service.py` already normalizes `agent_action`, `exception`, `notification`, and `listing_task` into `AgentOSWorkItem`.
- `backend/app/agentos/router.py` already exposes `/api/agentos/work-items`, status mutation, approve, reject, operation logs, and Agent detail endpoints.
- `backend/app/agentos/models.py` currently only persists `AgentOSOperationLog`.
- `backend/app/models.py` has a legacy `agent_action` table for simple action proposals and a separate Hermes `agent_pending_action` model under `backend/app/agent/models.py`.
- `frontend/src/views/agentos/WorkItems.vue` and `frontend/src/components/agentos/WorkItemCard.vue` already render WorkItems and call approve/reject/status APIs.

Design decision for this phase:

```text
Do not delete or rewrite existing AgentAction tables.
Add ActionProposal as the new canonical commerce action model.
Keep adapters so old AgentAction, exceptions, notifications, and listing tasks still appear in the task center.
```

---

## File Structure

Create:

- `backend/app/agentos/action_center_service.py` — persistent action proposal, approval, execution, review service.
- `backend/tests/test_agentos_action_center.py` — action center API and service tests.
- `backend/alembic/versions/20260622_01_add_agentos_action_center.py` — migration for new tables.

Modify:

- `backend/app/agentos/models.py` — add persistent action-center models.
- `backend/app/agentos/schemas.py` — add action-center enums, payload schemas, and response models.
- `backend/app/agentos/router.py` — add proposal, approval, execution, and review endpoints.
- `backend/app/agentos/service.py` — include persistent action proposals in WorkItems and delegate approve/reject for `action_proposal:*`.
- `frontend/src/api/modules/agentos.ts` — add action-center types and API methods.
- `frontend/src/components/agentos/WorkItemCard.vue` — show proposal metadata and call review/execution flows.
- `frontend/src/views/agentos/WorkItems.vue` — add source/risk filters and refresh behavior for persistent proposals.
- `docs/INDEX.md` — already links this plan from the Development section.

Do not modify:

- Existing business modules unless a task explicitly calls for a source adapter.
- `.kilo/worktrees/`.
- Existing unrelated dirty files.

---

### Task 1: Add Failing Backend Tests For The Persistent Action Center

**Files:**
- Create: `backend/tests/test_agentos_action_center.py`

- [ ] **Step 1: Create tests for proposal creation, listing, approval, rejection, execution, and review**

Create `backend/tests/test_agentos_action_center.py`:

```python
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
```

- [ ] **Step 2: Run the new tests and verify they fail before implementation**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py -q
```

Expected: FAIL with `ImportError` for `ActionProposal`, `ApprovalRequest`, `CommandExecution`, or `OutcomeReview`.

- [ ] **Step 3: Commit the failing test if using TDD checkpoints**

```bash
git add backend/tests/test_agentos_action_center.py
git commit -m "test: define agentos action center contract"
```

---

### Task 2: Add Persistent Action Center Models And Migration

**Files:**
- Modify: `backend/app/agentos/models.py`
- Create: `backend/alembic/versions/20260622_01_add_agentos_action_center.py`

- [ ] **Step 1: Extend AgentOS models**

Replace `backend/app/agentos/models.py` with this model set, preserving the existing `AgentOSOperationLog` table:

```python
"""AgentOS 持久化模型"""
from sqlalchemy import (
    BigInteger,
    Boolean,
    CheckConstraint,
    Column,
    DateTime,
    ForeignKey,
    Integer,
    Numeric,
    String,
    Text,
)
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy import JSON
from sqlalchemy import func as sa_func
from sqlalchemy.orm import relationship

from app.database import Base


JSONType = JSON().with_variant(JSONB, "postgresql")


ACTION_PROPOSAL_STATUS_FLOW = {
    "suggested": {"pending_approval", "approved", "executing", "rejected", "expired", "blocked_by_policy"},
    "pending_approval": {"approved", "rejected", "expired", "blocked_by_policy"},
    "approved": {"executing", "cancelled", "blocked_by_policy"},
    "executing": {"executed", "failed"},
    "executed": {"reviewed"},
    "reviewed": set(),
    "rejected": set(),
    "expired": set(),
    "blocked_by_policy": set(),
    "failed": {"executing", "cancelled"},
    "cancelled": set(),
}


class AgentOSOperationLog(Base):
    """AgentOS 操作审计日志"""
    __tablename__ = "agentos_operation_log"

    id = Column(Integer, primary_key=True, autoincrement=True)
    user_id = Column(Integer, nullable=False, index=True)
    item_id = Column(String(128), nullable=False, comment="WorkItem ID (e.g. exception:42)")
    action = Column(String(32), nullable=False, comment="approve / reject / status_update")
    source_type = Column(String(32), nullable=True, comment="agent_action / exception / notification / listing_task")
    previous_status = Column(String(32), nullable=True)
    new_status = Column(String(32), nullable=True)
    comment = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)


class ActionProposal(Base):
    """AgentOS 统一动作提案"""
    __tablename__ = "agentos_action_proposal"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    source_type = Column(String(50), nullable=False, index=True)
    source_id = Column(String(100), nullable=True, index=True)
    agent_id = Column(String(20), nullable=True, index=True)
    squad_id = Column(String(50), nullable=True, index=True)
    action_type = Column(String(100), nullable=False, index=True)
    business_object_type = Column(String(50), nullable=True, index=True)
    business_object_id = Column(String(100), nullable=True, index=True)
    title = Column(String(300), nullable=False)
    description = Column(Text, nullable=True)
    proposed_payload = Column(JSONType, nullable=False, default=dict)
    before_snapshot = Column(JSONType, nullable=True)
    after_snapshot = Column(JSONType, nullable=True)
    risk_level = Column(String(20), nullable=False, default="medium", index=True)
    requires_approval = Column(Boolean, nullable=False, default=True)
    status = Column(String(30), nullable=False, default="suggested", index=True)
    confidence = Column(Numeric(5, 4), nullable=True)
    proposed_by = Column(String(100), nullable=True)
    approved_by = Column(String(100), nullable=True)
    approved_at = Column(DateTime(timezone=True), nullable=True)
    rejected_by = Column(String(100), nullable=True)
    rejected_at = Column(DateTime(timezone=True), nullable=True)
    rejection_reason = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=sa_func.now(), onupdate=sa_func.now(), nullable=False)

    approvals = relationship("ApprovalRequest", back_populates="proposal", lazy="selectin")
    executions = relationship("CommandExecution", back_populates="proposal", lazy="selectin")
    reviews = relationship("OutcomeReview", back_populates="proposal", lazy="selectin")

    __table_args__ = (
        CheckConstraint(
            "risk_level in ('low', 'medium', 'high', 'critical')",
            name="ck_agentos_action_proposal_risk",
        ),
        CheckConstraint(
            "status in ('suggested', 'pending_approval', 'approved', 'executing', 'executed', 'reviewed', 'rejected', 'expired', 'blocked_by_policy', 'failed', 'cancelled')",
            name="ck_agentos_action_proposal_status",
        ),
    )


class ApprovalRequest(Base):
    """ActionProposal 审批记录"""
    __tablename__ = "agentos_approval_request"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    proposal_id = Column(BigInteger, ForeignKey("agentos_action_proposal.id"), nullable=False, index=True)
    requester = Column(String(100), nullable=True)
    approver = Column(String(100), nullable=True)
    decision = Column(String(30), nullable=False, default="pending")
    comment = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)
    decided_at = Column(DateTime(timezone=True), nullable=True)

    proposal = relationship("ActionProposal", back_populates="approvals", lazy="selectin")

    __table_args__ = (
        CheckConstraint(
            "decision in ('pending', 'approved', 'rejected')",
            name="ck_agentos_approval_request_decision",
        ),
    )


class CommandExecution(Base):
    """业务命令执行记录"""
    __tablename__ = "agentos_command_execution"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    proposal_id = Column(BigInteger, ForeignKey("agentos_action_proposal.id"), nullable=False, index=True)
    command_name = Column(String(100), nullable=False)
    executor = Column(String(100), nullable=True)
    status = Column(String(30), nullable=False, default="started")
    input_payload = Column(JSONType, nullable=False, default=dict)
    result_payload = Column(JSONType, nullable=True)
    error_message = Column(Text, nullable=True)
    started_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)
    finished_at = Column(DateTime(timezone=True), nullable=True)

    proposal = relationship("ActionProposal", back_populates="executions", lazy="selectin")

    __table_args__ = (
        CheckConstraint(
            "status in ('started', 'succeeded', 'failed')",
            name="ck_agentos_command_execution_status",
        ),
    )


class OutcomeReview(Base):
    """动作执行后的经营结果复盘"""
    __tablename__ = "agentos_outcome_review"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    proposal_id = Column(BigInteger, ForeignKey("agentos_action_proposal.id"), nullable=False, index=True)
    outcome = Column(String(30), nullable=False)
    business_metric = Column(String(100), nullable=True)
    metric_delta = Column(Numeric(14, 4), nullable=True)
    notes = Column(Text, nullable=True)
    reviewed_by = Column(String(100), nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=sa_func.now(), nullable=False)

    proposal = relationship("ActionProposal", back_populates="reviews", lazy="selectin")

    __table_args__ = (
        CheckConstraint(
            "outcome in ('positive', 'neutral', 'negative')",
            name="ck_agentos_outcome_review_outcome",
        ),
    )
```

- [ ] **Step 2: Add Alembic migration**

Create `backend/alembic/versions/20260622_01_add_agentos_action_center.py`:

```python
"""add agentos action center

Revision ID: 20260622_01
Revises: 72181db29a25
Create Date: 2026-06-22
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql


revision: str = "20260622_01"
down_revision: Union[str, Sequence[str], None] = "72181db29a25"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


json_type = postgresql.JSONB(astext_type=sa.Text())


def upgrade() -> None:
    op.create_table(
        "agentos_action_proposal",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("source_type", sa.String(length=50), nullable=False),
        sa.Column("source_id", sa.String(length=100), nullable=True),
        sa.Column("agent_id", sa.String(length=20), nullable=True),
        sa.Column("squad_id", sa.String(length=50), nullable=True),
        sa.Column("action_type", sa.String(length=100), nullable=False),
        sa.Column("business_object_type", sa.String(length=50), nullable=True),
        sa.Column("business_object_id", sa.String(length=100), nullable=True),
        sa.Column("title", sa.String(length=300), nullable=False),
        sa.Column("description", sa.Text(), nullable=True),
        sa.Column("proposed_payload", json_type, nullable=False, server_default=sa.text("'{}'::jsonb")),
        sa.Column("before_snapshot", json_type, nullable=True),
        sa.Column("after_snapshot", json_type, nullable=True),
        sa.Column("risk_level", sa.String(length=20), nullable=False, server_default="medium"),
        sa.Column("requires_approval", sa.Boolean(), nullable=False, server_default=sa.text("true")),
        sa.Column("status", sa.String(length=30), nullable=False, server_default="suggested"),
        sa.Column("confidence", sa.Numeric(5, 4), nullable=True),
        sa.Column("proposed_by", sa.String(length=100), nullable=True),
        sa.Column("approved_by", sa.String(length=100), nullable=True),
        sa.Column("approved_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("rejected_by", sa.String(length=100), nullable=True),
        sa.Column("rejected_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("rejection_reason", sa.Text(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.CheckConstraint(
            "risk_level in ('low', 'medium', 'high', 'critical')",
            name="ck_agentos_action_proposal_risk",
        ),
        sa.CheckConstraint(
            "status in ('suggested', 'pending_approval', 'approved', 'executing', 'executed', 'reviewed', 'rejected', 'expired', 'blocked_by_policy', 'failed', 'cancelled')",
            name="ck_agentos_action_proposal_status",
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_agentos_action_proposal_source_type", "agentos_action_proposal", ["source_type"])
    op.create_index("ix_agentos_action_proposal_source_id", "agentos_action_proposal", ["source_id"])
    op.create_index("ix_agentos_action_proposal_agent_id", "agentos_action_proposal", ["agent_id"])
    op.create_index("ix_agentos_action_proposal_squad_id", "agentos_action_proposal", ["squad_id"])
    op.create_index("ix_agentos_action_proposal_action_type", "agentos_action_proposal", ["action_type"])
    op.create_index("ix_agentos_action_proposal_business_object_type", "agentos_action_proposal", ["business_object_type"])
    op.create_index("ix_agentos_action_proposal_business_object_id", "agentos_action_proposal", ["business_object_id"])
    op.create_index("ix_agentos_action_proposal_risk_level", "agentos_action_proposal", ["risk_level"])
    op.create_index("ix_agentos_action_proposal_status", "agentos_action_proposal", ["status"])

    op.create_table(
        "agentos_approval_request",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("proposal_id", sa.BigInteger(), nullable=False),
        sa.Column("requester", sa.String(length=100), nullable=True),
        sa.Column("approver", sa.String(length=100), nullable=True),
        sa.Column("decision", sa.String(length=30), nullable=False, server_default="pending"),
        sa.Column("comment", sa.Text(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.Column("decided_at", sa.DateTime(timezone=True), nullable=True),
        sa.CheckConstraint("decision in ('pending', 'approved', 'rejected')", name="ck_agentos_approval_request_decision"),
        sa.ForeignKeyConstraint(["proposal_id"], ["agentos_action_proposal.id"]),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_agentos_approval_request_proposal_id", "agentos_approval_request", ["proposal_id"])

    op.create_table(
        "agentos_command_execution",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("proposal_id", sa.BigInteger(), nullable=False),
        sa.Column("command_name", sa.String(length=100), nullable=False),
        sa.Column("executor", sa.String(length=100), nullable=True),
        sa.Column("status", sa.String(length=30), nullable=False, server_default="started"),
        sa.Column("input_payload", json_type, nullable=False, server_default=sa.text("'{}'::jsonb")),
        sa.Column("result_payload", json_type, nullable=True),
        sa.Column("error_message", sa.Text(), nullable=True),
        sa.Column("started_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.Column("finished_at", sa.DateTime(timezone=True), nullable=True),
        sa.CheckConstraint("status in ('started', 'succeeded', 'failed')", name="ck_agentos_command_execution_status"),
        sa.ForeignKeyConstraint(["proposal_id"], ["agentos_action_proposal.id"]),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_agentos_command_execution_proposal_id", "agentos_command_execution", ["proposal_id"])

    op.create_table(
        "agentos_outcome_review",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("proposal_id", sa.BigInteger(), nullable=False),
        sa.Column("outcome", sa.String(length=30), nullable=False),
        sa.Column("business_metric", sa.String(length=100), nullable=True),
        sa.Column("metric_delta", sa.Numeric(14, 4), nullable=True),
        sa.Column("notes", sa.Text(), nullable=True),
        sa.Column("reviewed_by", sa.String(length=100), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.CheckConstraint("outcome in ('positive', 'neutral', 'negative')", name="ck_agentos_outcome_review_outcome"),
        sa.ForeignKeyConstraint(["proposal_id"], ["agentos_action_proposal.id"]),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_agentos_outcome_review_proposal_id", "agentos_outcome_review", ["proposal_id"])


def downgrade() -> None:
    op.drop_index("ix_agentos_outcome_review_proposal_id", table_name="agentos_outcome_review")
    op.drop_table("agentos_outcome_review")
    op.drop_index("ix_agentos_command_execution_proposal_id", table_name="agentos_command_execution")
    op.drop_table("agentos_command_execution")
    op.drop_index("ix_agentos_approval_request_proposal_id", table_name="agentos_approval_request")
    op.drop_table("agentos_approval_request")
    op.drop_index("ix_agentos_action_proposal_status", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_risk_level", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_business_object_id", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_business_object_type", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_action_type", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_squad_id", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_agent_id", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_source_id", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_source_type", table_name="agentos_action_proposal")
    op.drop_table("agentos_action_proposal")
```

- [ ] **Step 3: Run migration and model tests**

Run:

```bash
cd backend && .venv/bin/alembic upgrade heads
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py -q
```

Expected after only this task: tests still fail because service and routes do not exist, but import errors for the new models are gone.

- [ ] **Step 4: Commit models and migration**

```bash
git add backend/app/agentos/models.py backend/alembic/versions/20260622_01_add_agentos_action_center.py
git commit -m "feat: add agentos action center models"
```

---

### Task 3: Add Action Center Schemas

**Files:**
- Modify: `backend/app/agentos/schemas.py`

- [ ] **Step 1: Add status enums and payload schemas**

Append these schemas after the existing WorkItem approval models:

```python
class ActionProposalStatus(str, Enum):
    SUGGESTED = "suggested"
    PENDING_APPROVAL = "pending_approval"
    APPROVED = "approved"
    EXECUTING = "executing"
    EXECUTED = "executed"
    REVIEWED = "reviewed"
    REJECTED = "rejected"
    EXPIRED = "expired"
    BLOCKED_BY_POLICY = "blocked_by_policy"
    FAILED = "failed"
    CANCELLED = "cancelled"


class ActionProposalCreate(BaseModel):
    source_type: str = Field(min_length=1, max_length=50)
    source_id: Optional[str] = Field(default=None, max_length=100)
    agent_id: Optional[str] = Field(default=None, max_length=20)
    squad_id: Optional[str] = Field(default=None, max_length=50)
    action_type: str = Field(min_length=1, max_length=100)
    business_object_type: Optional[str] = Field(default=None, max_length=50)
    business_object_id: Optional[str] = Field(default=None, max_length=100)
    title: str = Field(min_length=1, max_length=300)
    description: Optional[str] = None
    proposed_payload: dict[str, Any] = Field(default_factory=dict)
    before_snapshot: Optional[dict[str, Any]] = None
    risk_level: RiskLevel = RiskLevel.MEDIUM
    requires_approval: bool = True
    confidence: Optional[float] = Field(default=None, ge=0.0, le=1.0)


class ActionProposalVO(BaseModel):
    id: int
    source_type: str
    source_id: Optional[str] = None
    agent_id: Optional[str] = None
    squad_id: Optional[str] = None
    action_type: str
    business_object_type: Optional[str] = None
    business_object_id: Optional[str] = None
    title: str
    description: Optional[str] = None
    proposed_payload: dict[str, Any] = Field(default_factory=dict)
    before_snapshot: Optional[dict[str, Any]] = None
    after_snapshot: Optional[dict[str, Any]] = None
    risk_level: RiskLevel
    requires_approval: bool
    status: ActionProposalStatus
    confidence: Optional[float] = None
    proposed_by: Optional[str] = None
    approved_by: Optional[str] = None
    rejected_by: Optional[str] = None
    rejection_reason: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


class ActionApprovalPayload(BaseModel):
    comment: Optional[str] = None


class ActionExecutionPayload(BaseModel):
    executor: Optional[str] = None


class ActionReviewPayload(BaseModel):
    outcome: Literal["positive", "neutral", "negative"]
    business_metric: Optional[str] = Field(default=None, max_length=100)
    metric_delta: Optional[float] = None
    notes: Optional[str] = None


class ApprovalRequestVO(BaseModel):
    id: int
    proposal_id: int
    requester: Optional[str] = None
    approver: Optional[str] = None
    decision: str
    comment: Optional[str] = None
    created_at: Optional[datetime] = None
    decided_at: Optional[datetime] = None


class CommandExecutionVO(BaseModel):
    id: int
    proposal_id: int
    command_name: str
    executor: Optional[str] = None
    status: str
    input_payload: dict[str, Any] = Field(default_factory=dict)
    result_payload: Optional[dict[str, Any]] = None
    error_message: Optional[str] = None
    started_at: Optional[datetime] = None
    finished_at: Optional[datetime] = None


class OutcomeReviewVO(BaseModel):
    id: int
    proposal_id: int
    outcome: str
    business_metric: Optional[str] = None
    metric_delta: Optional[float] = None
    notes: Optional[str] = None
    reviewed_by: Optional[str] = None
    created_at: Optional[datetime] = None
```

- [ ] **Step 2: Run schema import check**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python - <<'PY'
from app.agentos.schemas import ActionProposalCreate, ActionProposalVO, ActionProposalStatus
print(ActionProposalStatus.SUGGESTED.value)
print(ActionProposalCreate(source_type="agent", action_type="notify", title="x").model_dump()["risk_level"])
PY
```

Expected:

```text
suggested
RiskLevel.MEDIUM
```

- [ ] **Step 3: Commit schemas**

```bash
git add backend/app/agentos/schemas.py
git commit -m "feat: add agentos action center schemas"
```

---

### Task 4: Implement Action Center Service

**Files:**
- Create: `backend/app/agentos/action_center_service.py`

- [ ] **Step 1: Create service with conversion helpers and state transitions**

Create `backend/app/agentos/action_center_service.py`:

```python
"""AgentOS action center service."""
from __future__ import annotations

from datetime import datetime, timezone
from decimal import Decimal
from typing import Any

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.agentos.models import (
    ACTION_PROPOSAL_STATUS_FLOW,
    ActionProposal,
    AgentOSOperationLog,
    ApprovalRequest,
    CommandExecution,
    OutcomeReview,
)
from app.agentos.schemas import (
    ActionProposalCreate,
    AgentOSWorkItem,
    AutonomyLevel,
    RiskLevel,
    WorkItemPriority,
    WorkItemStatus,
)
from app.operation_log.service import OperationLogService


RISK_TO_PRIORITY = {
    "low": WorkItemPriority.LOW,
    "medium": WorkItemPriority.MEDIUM,
    "high": WorkItemPriority.HIGH,
    "critical": WorkItemPriority.CRITICAL,
}


PROPOSAL_STATUS_TO_WORK_STATUS = {
    "suggested": WorkItemStatus.PENDING,
    "pending_approval": WorkItemStatus.PENDING,
    "approved": WorkItemStatus.IN_PROGRESS,
    "executing": WorkItemStatus.IN_PROGRESS,
    "executed": WorkItemStatus.COMPLETED,
    "reviewed": WorkItemStatus.COMPLETED,
    "rejected": WorkItemStatus.CANCELLED,
    "expired": WorkItemStatus.CANCELLED,
    "blocked_by_policy": WorkItemStatus.BLOCKED,
    "failed": WorkItemStatus.FAILED,
    "cancelled": WorkItemStatus.CANCELLED,
}


class ActionCenterService:
    @staticmethod
    def _operator(user: Any) -> str:
        if user is None:
            return "system"
        return getattr(user, "username", None) or str(getattr(user, "id", "system"))

    @staticmethod
    def _user_id(user: Any) -> int:
        if user is None:
            return 0
        return int(getattr(user, "id", 0) or 0)

    @staticmethod
    def _decimal_to_float(value: Any) -> float | None:
        if value is None:
            return None
        if isinstance(value, Decimal):
            return float(value)
        return float(value)

    @staticmethod
    def _proposal_to_dict(row: ActionProposal) -> dict[str, Any]:
        return {
            "id": row.id,
            "source_type": row.source_type,
            "source_id": row.source_id,
            "agent_id": row.agent_id,
            "squad_id": row.squad_id,
            "action_type": row.action_type,
            "business_object_type": row.business_object_type,
            "business_object_id": row.business_object_id,
            "title": row.title,
            "description": row.description,
            "proposed_payload": row.proposed_payload or {},
            "before_snapshot": row.before_snapshot,
            "after_snapshot": row.after_snapshot,
            "risk_level": row.risk_level,
            "requires_approval": bool(row.requires_approval),
            "status": row.status,
            "confidence": ActionCenterService._decimal_to_float(row.confidence),
            "proposed_by": row.proposed_by,
            "approved_by": row.approved_by,
            "rejected_by": row.rejected_by,
            "rejection_reason": row.rejection_reason,
            "created_at": row.created_at.isoformat() if row.created_at else None,
            "updated_at": row.updated_at.isoformat() if row.updated_at else None,
        }

    @staticmethod
    def proposal_to_work_item(row: ActionProposal) -> AgentOSWorkItem:
        risk = RiskLevel(row.risk_level)
        return AgentOSWorkItem(
            id=f"action_proposal:{row.id}",
            source_type="action_proposal",
            source_id=str(row.id),
            title=row.title,
            description=row.description,
            priority=RISK_TO_PRIORITY.get(row.risk_level, WorkItemPriority.MEDIUM),
            status=PROPOSAL_STATUS_TO_WORK_STATUS.get(row.status, WorkItemStatus.PENDING),
            risk_level=risk,
            agent_id=row.agent_id,
            agent_name=None,
            squad_id=row.squad_id,
            squad_name=None,
            autonomy_level=AutonomyLevel.SUGGESTION,
            requires_approval=bool(row.requires_approval and row.status in {"suggested", "pending_approval"}),
            created_at=row.created_at,
            updated_at=row.updated_at,
            action_url=f"/agentos/work-items?action_proposal={row.id}",
            metadata={
                "action_type": row.action_type,
                "business_object_type": row.business_object_type,
                "business_object_id": row.business_object_id,
                "payload": row.proposed_payload or {},
                "confidence": ActionCenterService._decimal_to_float(row.confidence),
                "proposal_status": row.status,
            },
        )

    @staticmethod
    async def create_proposal(
        db: AsyncSession,
        payload: ActionProposalCreate,
        operator: str,
    ) -> AgentOSWorkItem:
        status = "pending_approval" if payload.requires_approval else "suggested"
        proposal = ActionProposal(
            source_type=payload.source_type,
            source_id=payload.source_id,
            agent_id=payload.agent_id,
            squad_id=payload.squad_id,
            action_type=payload.action_type,
            business_object_type=payload.business_object_type,
            business_object_id=payload.business_object_id,
            title=payload.title,
            description=payload.description,
            proposed_payload=payload.proposed_payload,
            before_snapshot=payload.before_snapshot,
            risk_level=payload.risk_level.value,
            requires_approval=payload.requires_approval,
            status=status,
            confidence=payload.confidence,
            proposed_by=operator,
        )
        db.add(proposal)
        await db.flush()
        await db.refresh(proposal)
        await OperationLogService.log(
            db,
            module="agentos_action_center",
            action="propose",
            resource_id=str(proposal.id),
            content=f"创建动作提案: {proposal.action_type} - {proposal.title}",
            operator=operator,
        )
        return ActionCenterService.proposal_to_work_item(proposal)

    @staticmethod
    async def list_proposals(
        db: AsyncSession,
        status: str | None = None,
        risk_level: str | None = None,
        squad_id: str | None = None,
        limit: int = 50,
        offset: int = 0,
    ) -> tuple[list[ActionProposal], int]:
        stmt = select(ActionProposal)
        if status:
            stmt = stmt.where(ActionProposal.status == status)
        if risk_level:
            stmt = stmt.where(ActionProposal.risk_level == risk_level)
        if squad_id:
            stmt = stmt.where(ActionProposal.squad_id == squad_id)
        stmt = stmt.order_by(ActionProposal.created_at.desc())
        rows = (await db.execute(stmt.offset(offset).limit(limit))).scalars().all()
        total = len((await db.execute(stmt)).scalars().all())
        return list(rows), total

    @staticmethod
    def _ensure_transition(current: str, target: str) -> None:
        if target not in ACTION_PROPOSAL_STATUS_FLOW.get(current, set()):
            raise ValueError(f"状态 {current} 不允许流转到 {target}")

    @staticmethod
    async def approve(
        db: AsyncSession,
        proposal_id: int,
        operator: str,
        user_id: int,
        comment: str | None,
    ) -> dict[str, Any] | None:
        proposal = await db.get(ActionProposal, proposal_id)
        if proposal is None:
            return None
        ActionCenterService._ensure_transition(proposal.status, "approved")
        previous = proposal.status
        proposal.status = "approved"
        proposal.approved_by = operator
        proposal.approved_at = datetime.now(timezone.utc)
        approval = ApprovalRequest(
            proposal_id=proposal.id,
            requester=proposal.proposed_by,
            approver=operator,
            decision="approved",
            comment=comment,
            decided_at=datetime.now(timezone.utc),
        )
        db.add(approval)
        db.add(AgentOSOperationLog(
            user_id=user_id,
            item_id=f"action_proposal:{proposal.id}",
            action="approve",
            source_type="action_proposal",
            previous_status=previous,
            new_status="approved",
            comment=comment,
        ))
        await db.flush()
        await db.refresh(proposal)
        await db.refresh(approval)
        return {
            "ok": True,
            "proposal": ActionCenterService._proposal_to_dict(proposal),
            "approval": {
                "id": approval.id,
                "proposal_id": approval.proposal_id,
                "decision": approval.decision,
                "comment": approval.comment,
            },
        }

    @staticmethod
    async def reject(
        db: AsyncSession,
        proposal_id: int,
        operator: str,
        user_id: int,
        comment: str | None,
    ) -> dict[str, Any] | None:
        proposal = await db.get(ActionProposal, proposal_id)
        if proposal is None:
            return None
        ActionCenterService._ensure_transition(proposal.status, "rejected")
        previous = proposal.status
        proposal.status = "rejected"
        proposal.rejected_by = operator
        proposal.rejected_at = datetime.now(timezone.utc)
        proposal.rejection_reason = comment
        approval = ApprovalRequest(
            proposal_id=proposal.id,
            requester=proposal.proposed_by,
            approver=operator,
            decision="rejected",
            comment=comment,
            decided_at=datetime.now(timezone.utc),
        )
        db.add(approval)
        db.add(AgentOSOperationLog(
            user_id=user_id,
            item_id=f"action_proposal:{proposal.id}",
            action="reject",
            source_type="action_proposal",
            previous_status=previous,
            new_status="rejected",
            comment=comment,
        ))
        await db.flush()
        await db.refresh(proposal)
        await db.refresh(approval)
        return {
            "ok": True,
            "proposal": ActionCenterService._proposal_to_dict(proposal),
            "approval": {
                "id": approval.id,
                "proposal_id": approval.proposal_id,
                "decision": approval.decision,
                "comment": approval.comment,
            },
        }

    @staticmethod
    async def execute(
        db: AsyncSession,
        proposal_id: int,
        operator: str,
        user_id: int,
        executor: str | None,
    ) -> dict[str, Any] | None:
        proposal = await db.get(ActionProposal, proposal_id)
        if proposal is None:
            return None
        if proposal.requires_approval and proposal.status != "approved":
            raise ValueError("需要审批的动作必须先审批通过")
        target_from = proposal.status
        target = "executing"
        if target in ACTION_PROPOSAL_STATUS_FLOW.get(target_from, set()):
            proposal.status = target
            await db.flush()
        ActionCenterService._ensure_transition(proposal.status, "executed")
        execution = CommandExecution(
            proposal_id=proposal.id,
            command_name=proposal.action_type,
            executor=executor or operator,
            status="succeeded",
            input_payload=proposal.proposed_payload or {},
            result_payload={
                "mode": "simulated",
                "message": "Action center recorded execution; business command adapter will replace this in the next phase.",
            },
            finished_at=datetime.now(timezone.utc),
        )
        proposal.status = "executed"
        proposal.after_snapshot = execution.result_payload
        db.add(execution)
        db.add(AgentOSOperationLog(
            user_id=user_id,
            item_id=f"action_proposal:{proposal.id}",
            action="execute",
            source_type="action_proposal",
            previous_status=target_from,
            new_status="executed",
            comment=None,
        ))
        await db.flush()
        await db.refresh(proposal)
        await db.refresh(execution)
        return {
            "ok": True,
            "proposal": ActionCenterService._proposal_to_dict(proposal),
            "execution": {
                "id": execution.id,
                "proposal_id": execution.proposal_id,
                "status": execution.status,
                "command_name": execution.command_name,
            },
        }

    @staticmethod
    async def review(
        db: AsyncSession,
        proposal_id: int,
        operator: str,
        user_id: int,
        outcome: str,
        business_metric: str | None,
        metric_delta: float | None,
        notes: str | None,
    ) -> dict[str, Any] | None:
        proposal = await db.get(ActionProposal, proposal_id)
        if proposal is None:
            return None
        ActionCenterService._ensure_transition(proposal.status, "reviewed")
        previous = proposal.status
        proposal.status = "reviewed"
        review = OutcomeReview(
            proposal_id=proposal.id,
            outcome=outcome,
            business_metric=business_metric,
            metric_delta=metric_delta,
            notes=notes,
            reviewed_by=operator,
        )
        db.add(review)
        db.add(AgentOSOperationLog(
            user_id=user_id,
            item_id=f"action_proposal:{proposal.id}",
            action="review",
            source_type="action_proposal",
            previous_status=previous,
            new_status="reviewed",
            comment=notes,
        ))
        await db.flush()
        await db.refresh(proposal)
        await db.refresh(review)
        return {
            "ok": True,
            "proposal": ActionCenterService._proposal_to_dict(proposal),
            "review": {
                "id": review.id,
                "proposal_id": review.proposal_id,
                "outcome": review.outcome,
                "business_metric": review.business_metric,
                "metric_delta": ActionCenterService._decimal_to_float(review.metric_delta),
                "notes": review.notes,
            },
        }
```

- [ ] **Step 2: Run action-center tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py -q
```

Expected after this task: tests still fail because routes have not been wired.

- [ ] **Step 3: Commit service**

```bash
git add backend/app/agentos/action_center_service.py
git commit -m "feat: implement agentos action center service"
```

---

### Task 5: Add Action Center Routes

**Files:**
- Modify: `backend/app/agentos/router.py`

- [ ] **Step 1: Add imports**

Extend the imports in `backend/app/agentos/router.py`:

```python
from .action_center_service import ActionCenterService
from .schemas import (
    ActionApprovalPayload,
    ActionExecutionPayload,
    ActionProposalCreate,
    ActionReviewPayload,
    WorkItemApproval,
    WorkItemStatusUpdate,
)
```

- [ ] **Step 2: Add operator helper**

Insert after `router = APIRouter(tags=["AgentOS"])`:

```python
def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"
```

- [ ] **Step 3: Add proposal endpoints before existing mutation routes**

Insert these endpoints before the `# ── Phase 2: Mutation 操作` block:

```python
@router.post("/agentos/action-proposals", summary="创建 AgentOS 动作提案")
async def create_action_proposal(
    body: ActionProposalCreate,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    item = await ActionCenterService.create_proposal(
        db, body, operator=_operator(current_user),
    )
    return Result.ok(item)


@router.post("/agentos/action-proposals/{proposal_id}/approve", summary="审批通过动作提案")
async def approve_action_proposal(
    proposal_id: int,
    body: ActionApprovalPayload,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    try:
        result = await ActionCenterService.approve(
            db,
            proposal_id,
            operator=_operator(current_user),
            user_id=current_user.id,
            comment=body.comment,
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))
    if result is None:
        return Result.not_found("ActionProposal not found")
    return Result.ok(result)


@router.post("/agentos/action-proposals/{proposal_id}/reject", summary="拒绝动作提案")
async def reject_action_proposal(
    proposal_id: int,
    body: ActionApprovalPayload,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    try:
        result = await ActionCenterService.reject(
            db,
            proposal_id,
            operator=_operator(current_user),
            user_id=current_user.id,
            comment=body.comment,
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))
    if result is None:
        return Result.not_found("ActionProposal not found")
    return Result.ok(result)


@router.post("/agentos/action-proposals/{proposal_id}/execute", summary="执行动作提案")
async def execute_action_proposal(
    proposal_id: int,
    body: ActionExecutionPayload,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    try:
        result = await ActionCenterService.execute(
            db,
            proposal_id,
            operator=_operator(current_user),
            user_id=current_user.id,
            executor=body.executor,
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))
    if result is None:
        return Result.not_found("ActionProposal not found")
    return Result.ok(result)


@router.post("/agentos/action-proposals/{proposal_id}/review", summary="复盘动作结果")
async def review_action_proposal(
    proposal_id: int,
    body: ActionReviewPayload,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:operate")),
):
    try:
        result = await ActionCenterService.review(
            db,
            proposal_id,
            operator=_operator(current_user),
            user_id=current_user.id,
            outcome=body.outcome,
            business_metric=body.business_metric,
            metric_delta=body.metric_delta,
            notes=body.notes,
        )
    except ValueError as exc:
        return Result.bad_request(str(exc))
    if result is None:
        return Result.not_found("ActionProposal not found")
    return Result.ok(result)
```

- [ ] **Step 4: Run action-center tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py -q
```

Expected after this task: proposal creation, approval, rejection, execution, and review tests pass except the work-items integration test.

- [ ] **Step 5: Commit routes**

```bash
git add backend/app/agentos/router.py
git commit -m "feat: expose agentos action proposal endpoints"
```

---

### Task 6: Integrate Action Proposals Into AgentOS WorkItems

**Files:**
- Modify: `backend/app/agentos/service.py`
- Modify: `backend/app/agentos/router.py`
- Modify: `backend/app/agentos/schemas.py`

- [ ] **Step 1: Add source_type query support**

In `backend/app/agentos/schemas.py`, extend `WorkItemQuery` only if a backend query model exists. If no backend query model exists, keep this in the router query params only.

In `backend/app/agentos/router.py`, update `list_work_items` signature:

```python
@router.get("/agentos/work-items", summary="AgentOS 任务列表")
async def list_work_items(
    status: str | None = None,
    priority: str | None = None,
    squad: str | None = None,
    agent_id: str | None = None,
    source_type: str | None = None,
    requires_approval: bool | None = None,
    limit: int = 20,
    offset: int = 0,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    data = await AgentOSService.get_work_items(
        db,
        status=status,
        priority=priority,
        squad=squad,
        agent_id=agent_id,
        source_type=source_type,
        requires_approval=requires_approval,
        limit=limit,
        offset=offset,
    )
    return PageResult.ok(
        records=data["items"],
        total=data["total"],
        page=(offset // limit) + 1,
        page_size=limit,
    )
```

- [ ] **Step 2: Import the new model and service**

At the top of `backend/app/agentos/service.py`, add:

```python
from app.agentos.action_center_service import ActionCenterService
from app.agentos.models import ActionProposal
```

Keep the existing `AgentOSOperationLog` import from `app.agentos.models`.

- [ ] **Step 3: Include action proposals in the WorkItem aggregation**

Find the method that builds `items` inside `AgentOSService.get_work_items`. Add this query before sorting and pagination:

```python
proposal_stmt = select(ActionProposal).order_by(ActionProposal.created_at.desc()).limit(100)
proposal_result = await db.execute(proposal_stmt)
proposal_items = [
    ActionCenterService.proposal_to_work_item(row)
    for row in proposal_result.scalars().all()
]
items.extend(proposal_items)
```

- [ ] **Step 4: Apply source_type filter**

In the in-memory filter section of `AgentOSService.get_work_items`, add:

```python
if source_type:
    items = [it for it in items if it.source_type == source_type]
```

Also update the method signature to accept `source_type: str | None = None`.

- [ ] **Step 5: Delegate legacy WorkItem approval for action proposals**

In `AgentOSService.approve_work_item`, add this branch after parsing `source_type` and `source_id`:

```python
if source_type == "action_proposal":
    result = await ActionCenterService.approve(
        db,
        int(source_id),
        operator=str(user_id),
        user_id=user_id,
        comment=comment,
    )
    if result is None:
        return {"ok": False, "error": "not_found"}
    return result
```

In `AgentOSService.reject_work_item`, add:

```python
if source_type == "action_proposal":
    result = await ActionCenterService.reject(
        db,
        int(source_id),
        operator=str(user_id),
        user_id=user_id,
        comment=comment,
    )
    if result is None:
        return {"ok": False, "error": "not_found"}
    return result
```

- [ ] **Step 6: Run AgentOS regression tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py tests/test_agentos_action_center.py -q
```

Expected: all AgentOS tests pass.

- [ ] **Step 7: Commit WorkItem integration**

```bash
git add backend/app/agentos/service.py backend/app/agentos/router.py backend/app/agentos/schemas.py
git commit -m "feat: surface action proposals as agentos work items"
```

---

### Task 7: Upgrade Frontend API Types And WorkItem UI

**Files:**
- Modify: `frontend/src/api/modules/agentos.ts`
- Modify: `frontend/src/components/agentos/WorkItemCard.vue`
- Modify: `frontend/src/views/agentos/WorkItems.vue`

- [ ] **Step 1: Add `source_type` filter and proposal APIs**

In `frontend/src/api/modules/agentos.ts`, update `WorkItemQuery`:

```typescript
export interface WorkItemQuery {
  status?: string
  priority?: string
  squad?: string
  agent_id?: string
  source_type?: string
  requires_approval?: boolean
  limit?: number
  offset?: number
}
```

Add proposal types and methods before the object export:

```typescript
export interface ActionProposalCreatePayload {
  source_type: string
  source_id?: string | null
  agent_id?: string | null
  squad_id?: string | null
  action_type: string
  business_object_type?: string | null
  business_object_id?: string | null
  title: string
  description?: string | null
  proposed_payload?: Record<string, any>
  before_snapshot?: Record<string, any> | null
  risk_level: RiskLevel
  requires_approval: boolean
  confidence?: number | null
}

export interface ActionApprovalPayload {
  comment?: string | null
}

export interface ActionExecutionPayload {
  executor?: string | null
}

export interface ActionReviewPayload {
  outcome: 'positive' | 'neutral' | 'negative'
  business_metric?: string | null
  metric_delta?: number | null
  notes?: string | null
}

export function createActionProposal(payload: ActionProposalCreatePayload) {
  return http.post('/agentos/action-proposals', payload)
}

export function approveActionProposal(proposalId: number, payload: ActionApprovalPayload) {
  return http.post(`/agentos/action-proposals/${proposalId}/approve`, payload)
}

export function rejectActionProposal(proposalId: number, payload: ActionApprovalPayload) {
  return http.post(`/agentos/action-proposals/${proposalId}/reject`, payload)
}

export function executeActionProposal(proposalId: number, payload: ActionExecutionPayload) {
  return http.post(`/agentos/action-proposals/${proposalId}/execute`, payload)
}

export function reviewActionProposal(proposalId: number, payload: ActionReviewPayload) {
  return http.post(`/agentos/action-proposals/${proposalId}/review`, payload)
}
```

Add these methods to `agentosApi`.

- [ ] **Step 2: Add source filter to WorkItems view**

In `frontend/src/views/agentos/WorkItems.vue`, extend `filters`:

```typescript
const filters = reactive({
  status: null as string | null,
  priority: null as string | null,
  squad: null as string | null,
  sourceType: null as string | null,
  requiresApproval: false,
})
```

Add source options:

```typescript
const sourceOptions = [
  { label: '动作提案', value: 'action_proposal' },
  { label: 'Agent 动作', value: 'agent_action' },
  { label: '异常', value: 'exception' },
  { label: '通知', value: 'notification' },
  { label: '上架任务', value: 'listing_task' },
]
```

Add this filter control next to the squad filter:

```vue
<div class="filter-group">
  <span class="filter-label">来源</span>
  <n-select
    v-model:value="filters.sourceType"
    clearable
    :options="sourceOptions"
    style="width: 130px;"
    placeholder="全部来源"
  />
</div>
```

Update `fetchItems`:

```typescript
if (filters.sourceType) query.source_type = filters.sourceType
```

Update `resetFilters`:

```typescript
filters.sourceType = null
```

- [ ] **Step 3: Show action proposal metadata on cards**

In `frontend/src/components/agentos/WorkItemCard.vue`, add `action_proposal` to `sourceLabel`:

```typescript
const sourceLabel = computed(() => {
  const map: Record<string, string> = {
    action_proposal: '动作提案',
    agent_action: 'Agent 动作',
    exception: '异常',
    notification: '通知',
    listing_task: '上架任务',
  }
  return map[props.item.source_type] || props.item.source_type
})
```

Add this block below the existing description:

```vue
<n-space v-if="item.metadata?.action_type" size="small" style="margin-top: 6px;">
  <n-tag size="small" :bordered="false">{{ item.metadata.action_type }}</n-tag>
  <n-tag v-if="item.metadata?.business_object_type" size="small" :bordered="false">
    {{ item.metadata.business_object_type }}:{{ item.metadata.business_object_id || '-' }}
  </n-tag>
  <n-tag v-if="item.metadata?.confidence !== undefined" size="small" :bordered="false">
    置信度 {{ Math.round(Number(item.metadata.confidence || 0) * 100) }}%
  </n-tag>
</n-space>
```

- [ ] **Step 4: Run frontend checks**

Run:

```bash
cd frontend && npm run build
```

Expected: build succeeds.

- [ ] **Step 5: Commit frontend integration**

```bash
git add frontend/src/api/modules/agentos.ts frontend/src/components/agentos/WorkItemCard.vue frontend/src/views/agentos/WorkItems.vue
git commit -m "feat: expose action proposals in agentos UI"
```

---

### Task 8: Add Business Command Adapter Boundary

**Files:**
- Modify: `backend/app/agentos/action_center_service.py`
- Create: `backend/tests/test_agentos_action_center_commands.py`

- [ ] **Step 1: Add tests for command adapter allowlist**

Create `backend/tests/test_agentos_action_center_commands.py`:

```python
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
```

- [ ] **Step 2: Implement adapter registry**

Add this registry near the top of `backend/app/agentos/action_center_service.py`:

```python
COMMAND_ADAPTERS: dict[str, dict[str, str]] = {
    "daily_report": {"command_name": "daily_report", "execution_mode": "record_only"},
    "listing_draft": {"command_name": "listing_draft", "execution_mode": "record_only"},
    "profit_review": {"command_name": "profit_review", "execution_mode": "record_only"},
    "inventory_allocate": {"command_name": "inventory_allocate", "execution_mode": "record_only"},
    "notify": {"command_name": "notify", "execution_mode": "record_only"},
}


def resolve_command_adapter(action_type: str) -> dict[str, str]:
    adapter = COMMAND_ADAPTERS.get(action_type)
    if adapter is None:
        raise ValueError(f"Business command adapter not registered: {action_type}")
    return adapter
```

In `ActionCenterService.execute`, replace `command_name=proposal.action_type` with:

```python
adapter = resolve_command_adapter(proposal.action_type)
```

Then use:

```python
command_name=adapter["command_name"],
result_payload={
    "mode": adapter["execution_mode"],
    "message": "Action center recorded execution; business command adapter will replace record-only behavior per command.",
},
```

- [ ] **Step 3: Run command adapter tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center_commands.py tests/test_agentos_action_center.py -q
```

Expected: all pass.

- [ ] **Step 4: Commit command adapter boundary**

```bash
git add backend/app/agentos/action_center_service.py backend/tests/test_agentos_action_center_commands.py
git commit -m "feat: add agentos command adapter boundary"
```

---

### Task 9: Full Verification

**Files:**
- No source files changed in this task.

- [ ] **Step 1: Run targeted backend tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest \
  tests/test_agentos_phase1.py \
  tests/test_agentos_action_center.py \
  tests/test_agentos_action_center_commands.py \
  tests/test_agent_action_audit.py \
  tests/test_exception_workbench.py \
  tests/test_listing_tasks.py \
  -q
```

Expected: all selected tests pass.

- [ ] **Step 2: Run full backend tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest -q
```

Expected: all backend tests pass.

- [ ] **Step 3: Run frontend build**

```bash
cd frontend && npm run build
```

Expected: build succeeds.

- [ ] **Step 4: Verify migrations**

```bash
cd backend && .venv/bin/alembic current --verbose
cd backend && .venv/bin/alembic upgrade heads
```

Expected: current revision includes `20260622_01`; `upgrade heads` finishes without migration errors.

- [ ] **Step 5: Manual smoke check**

Start services:

```bash
cd backend && .venv/bin/uvicorn app.main:app --reload --host 127.0.0.1 --port 8001
cd frontend && npm run dev -- --host 127.0.0.1 --port 3001
```

Smoke flow:

1. Open `http://127.0.0.1:3001/agentos/work-items`.
2. Create a proposal through API:

```bash
curl -s -X POST http://127.0.0.1:8001/api/agentos/action-proposals \
  -H 'Content-Type: application/json' \
  -d '{"source_type":"agent","source_id":"A6","agent_id":"A6","squad_id":"risk","action_type":"profit_review","business_object_type":"sku","business_object_id":"SKU-SMOKE","title":"Smoke 利润复核","description":"验证动作中枢","proposed_payload":{"sku_id":"SKU-SMOKE"},"risk_level":"medium","requires_approval":true,"confidence":0.8}'
```

3. Refresh the task center.
4. Filter source as `动作提案`.
5. Approve the card.
6. Confirm the task updates and backend returns `code: 200`.

- [ ] **Step 6: Final commit**

```bash
git status --short
git add backend/app/agentos/models.py \
  backend/app/agentos/schemas.py \
  backend/app/agentos/action_center_service.py \
  backend/app/agentos/router.py \
  backend/app/agentos/service.py \
  backend/alembic/versions/20260622_01_add_agentos_action_center.py \
  backend/tests/test_agentos_action_center.py \
  backend/tests/test_agentos_action_center_commands.py \
  frontend/src/api/modules/agentos.ts \
  frontend/src/components/agentos/WorkItemCard.vue \
  frontend/src/views/agentos/WorkItems.vue
git commit -m "feat: add agent commerce action center"
```

---

## Implementation Notes

- `ActionProposal` is the canonical model for new Agent Commerce OS actions.
- Existing `AgentAction` stays in place for backward compatibility and continues to appear in AgentOS WorkItems.
- `CommandExecution` is record-only in this phase. Real business command adapters should be implemented in the next phase after this boundary is stable.
- Approval and execution routes return `Result.ok()`, `Result.bad_request()`, or `Result.not_found()` following project conventions.
- `AUTH_ENABLED=False` in tests provides mock admin access, so tests should not manually create auth tokens.
- Use `alembic upgrade heads`, because this repository can have multiple Alembic heads.

## Follow-Up Plan

After this plan is implemented and verified, write the next plan:

```text
docs/superpowers/plans/YYYY-MM-DD-agent-commerce-business-commands.md
```

That plan should replace record-only command execution with real adapters for:

- `listing_draft`
- `profit_review`
- `inventory_allocate`
- `daily_report`
- `notify`
