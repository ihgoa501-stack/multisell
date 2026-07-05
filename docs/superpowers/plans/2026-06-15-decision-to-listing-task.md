> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Decision To Listing Task Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert approved pre-listing decision results into auditable listing tasks so operators can move from "this SKU is worth listing" to "this product is ready to publish".

**Architecture:** Add a `listing_task` table and a focused service/router layer under the existing `backend/app/listing/` module. Listing tasks sit between decision results and direct platform publishing: they store the decision snapshot, check publish readiness with existing `ListingService.validate_publish_ready(...)`, and only allow `ready` tasks to call the existing publish adapter.

**Tech Stack:** FastAPI, Pydantic v2, SQLAlchemy async, Alembic, PostgreSQL, pytest, Vue 3, TypeScript, Vite, Naive UI.

---

## Starting Point

This plan implements Stage 1 from:

```text
docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md
```

Expected existing modules:

- `backend/app/decision/` has single and batch pre-listing decision APIs.
- `backend/app/listing/` has direct publish APIs and `ListingService.validate_publish_ready(...)`.
- `backend/app/models.py` has `ProductListing`.
- `frontend/src/views/decision/BatchPreListingDecision.vue` can display batch decision results.
- `frontend/src/views/listing/ListingManage.vue` can display existing platform listing records.

Create a new branch:

```bash
git switch main
git pull
git switch -c codex/decision-to-listing-task
```

If current work is still on the Excel batch decision branch, branch from that completed branch instead:

```bash
git switch codex/excel-batch-prelisting-decision
git switch -c codex/decision-to-listing-task
```

## Scope

In scope:

- `listing_task` database table.
- Create listing tasks from approved decision results.
- Readiness check using existing publish validation.
- Listing task queue APIs.
- Publish a ready listing task through existing `ListingService.publish(...)`.
- Frontend action from batch decision results: "生成上架任务".
- Frontend listing task queue.
- Permissions and audit logs.

Out of scope:

- Real marketplace API integration.
- New platform adapter.
- Platform category mapping.
- Logistics reconciliation.
- WMS.
- BI dashboards.
- Agent task queue.

## Business Rules

Task statuses:

```text
ready      Product/platform can be published now.
blocked    Product/platform is missing required data.
published  Task was published through existing ListingService.
failed     Publish attempt failed.
cancelled  Operator cancelled task.
```

Creation rules:

- Only decision results with `recommendation == "approve"` can create tasks.
- Each approved decision row creates or reuses one task for `(product_id, platform_id)`.
- `sku_id` is used to find `product_id`.
- If the same `(product_id, platform_id)` already has a task in `ready`, `blocked`, or `failed`, return it instead of creating a duplicate.
- If the product fails readiness validation, create/update task as `blocked` with `missing_requirements`.
- If the product passes readiness validation, create/update task as `ready`.

Publish rules:

- Only `ready` tasks can be published.
- Publishing calls existing `ListingService.publish(db, product, platform)`.
- Successful publish marks task `published` and links `product_listing_id`.
- Failed publish marks task `failed` and stores `last_error`.
- Direct old publish endpoint remains available.

Permissions:

```text
listing:view          View tasks and listing records.
listing:task_manage   Create/cancel/recheck listing tasks.
listing:publish       Publish ready listing task.
```

## File Structure

Create:

- `backend/app/listing/task_schemas.py` - Pydantic schemas for listing task APIs.
- `backend/app/listing/task_service.py` - task creation, readiness check, list, cancel, publish.
- `backend/alembic/versions/20260615_04_add_listing_tasks.py` - migration.
- `backend/tests/test_listing_tasks.py` - backend behavior, permissions, audit.
- `frontend/src/api/modules/listing.ts` - listing task API client.
- `frontend/src/views/listing/ListingTaskQueue.vue` - queue page.

Modify:

- `backend/app/models.py` - add `ListingTask`.
- `backend/app/listing/router.py` - add task endpoints.
- `backend/seed.py` - add `listing:task_manage`.
- `frontend/src/views/decision/BatchPreListingDecision.vue` - add generate task action.
- `frontend/src/router/modules/listing.ts` or the current listing route module - add task queue route.
- `frontend/src/views/listing/ListingManage.vue` - add link to task queue if route module structure makes this easier.
- `docs/PROJECT_STATUS.md` - document Stage 1.
- `docs/ROADMAP.md` - next recommended task becomes shipping bill reconciliation.

## API Contract

Create from decisions:

```text
POST /api/listing-tasks/from-decisions
```

Request:

```json
{
  "items": [
    {
      "item_key": "row-1",
      "sku_id": 123,
      "platform_id": 1,
      "decision_result": {
        "sku_id": 123,
        "destination_country": "RU",
        "target_sale_price": 5000,
        "product_cost": 500,
        "shipping_fee": 60,
        "platform_fee": 500,
        "payment_fee": 150,
        "fixed_fee": 0,
        "advertising_fee": 0,
        "other_fee": 100,
        "profit_amount": 3690,
        "profit_margin": 73.8,
        "recommendation": "approve",
        "blocking_reasons": [],
        "warnings": [],
        "platform_fee_source": "manual"
      }
    }
  ]
}
```

Response:

```json
{
  "created_count": 1,
  "reused_count": 0,
  "skipped_count": 0,
  "tasks": [
    {
      "id": 1,
      "product_id": 10,
      "platform_id": 1,
      "status": "ready",
      "missing_requirements": [],
      "source_item_key": "row-1"
    }
  ],
  "skipped": []
}
```

List:

```text
GET /api/listing-tasks?status=ready&platform_id=1
```

Recheck:

```text
POST /api/listing-tasks/{task_id}/recheck
```

Publish:

```text
POST /api/listing-tasks/{task_id}/publish
```

Cancel:

```text
POST /api/listing-tasks/{task_id}/cancel
```

## Task 1: Database Model And Migration

**Files:**
- Modify: `backend/app/models.py`
- Create: `backend/alembic/versions/20260615_04_add_listing_tasks.py`
- Create: `backend/tests/test_listing_tasks.py`

- [ ] **Step 1: Add failing model smoke test**

Create `backend/tests/test_listing_tasks.py`:

```python
"""上架任务队列测试。"""

from uuid import uuid4

import pytest
from sqlalchemy import select

from app.database import async_session_factory
from app.models import ListingTask, OperationLog


def _code(prefix: str) -> str:
    return f"{prefix}_{uuid4().hex[:8]}"


async def _count_logs(module: str, action: str, resource_id: str) -> int:
    async with async_session_factory() as session:
        result = await session.execute(
            select(OperationLog).where(
                OperationLog.module == module,
                OperationLog.action == action,
                OperationLog.resource_id == resource_id,
            )
        )
        return len(result.scalars().all())


class TestListingTaskModel:
    async def test_listing_task_model_is_mapped(self):
        async with async_session_factory() as session:
            task = ListingTask(
                product_id=1,
                platform_id=1,
                source_type="decision",
                source_item_key="row-1",
                status="ready",
                missing_requirements=[],
                decision_snapshot={"recommendation": "approve"},
            )
            session.add(task)
            await session.flush()
            assert task.id is not None
            await session.rollback()
```

- [ ] **Step 2: Run smoke test to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_listing_tasks.py::TestListingTaskModel::test_listing_task_model_is_mapped -q
```

Expected:

- FAIL because `ListingTask` does not exist.

- [ ] **Step 3: Add SQLAlchemy model**

In `backend/app/models.py`, add this class after `ProductListing`:

```python
class ListingTask(Base):
    """上架任务队列"""
    __tablename__ = "listing_task"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    sku_id = Column(BigInteger, ForeignKey("sku.id"), comment="来源SKU ID")
    product_listing_id = Column(BigInteger, ForeignKey("product_listing.id"), comment="发布记录ID")
    source_type = Column(String(50), default="decision", nullable=False, comment="来源: decision/manual")
    source_item_key = Column(String(100), comment="来源行标识")
    status = Column(String(50), default="blocked", nullable=False, comment="ready/blocked/published/failed/cancelled")
    missing_requirements = Column(JSON, default=list, nullable=False, comment="阻塞发布的缺失项")
    decision_snapshot = Column(JSON, comment="决策结果快照")
    target_sale_price = Column(Numeric(12, 2), comment="决策目标售价")
    target_profit_margin = Column(Numeric(8, 2), comment="决策利润率")
    destination_country = Column(String(10), comment="目的国")
    last_error = Column(Text, comment="最近错误")
    created_by = Column(String(100), comment="创建人")
    updated_by = Column(String(100), comment="更新人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    product = relationship("Product", lazy="selectin")
    platform = relationship("Platform", lazy="selectin")
    sku = relationship("Sku", lazy="selectin")
    product_listing = relationship("ProductListing", lazy="selectin")
```

- [ ] **Step 4: Add Alembic migration**

Create `backend/alembic/versions/20260615_04_add_listing_tasks.py`:

```python
"""add listing tasks

Revision ID: 20260615_04
Revises: 20260615_03
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSON


revision = "20260615_04"
down_revision = "20260615_03"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "listing_task",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("product_id", sa.BigInteger(), sa.ForeignKey("product.id"), nullable=False),
        sa.Column("platform_id", sa.BigInteger(), sa.ForeignKey("platform.id"), nullable=False),
        sa.Column("sku_id", sa.BigInteger(), sa.ForeignKey("sku.id"), nullable=True),
        sa.Column("product_listing_id", sa.BigInteger(), sa.ForeignKey("product_listing.id"), nullable=True),
        sa.Column("source_type", sa.String(length=50), nullable=False, server_default="decision"),
        sa.Column("source_item_key", sa.String(length=100), nullable=True),
        sa.Column("status", sa.String(length=50), nullable=False, server_default="blocked"),
        sa.Column("missing_requirements", JSON(), nullable=False, server_default="[]"),
        sa.Column("decision_snapshot", JSON(), nullable=True),
        sa.Column("target_sale_price", sa.Numeric(12, 2), nullable=True),
        sa.Column("target_profit_margin", sa.Numeric(8, 2), nullable=True),
        sa.Column("destination_country", sa.String(length=10), nullable=True),
        sa.Column("last_error", sa.Text(), nullable=True),
        sa.Column("created_by", sa.String(length=100), nullable=True),
        sa.Column("updated_by", sa.String(length=100), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_listing_task_status", "listing_task", ["status"])
    op.create_index("ix_listing_task_product_platform", "listing_task", ["product_id", "platform_id"])


def downgrade() -> None:
    op.drop_index("ix_listing_task_product_platform", table_name="listing_task")
    op.drop_index("ix_listing_task_status", table_name="listing_task")
    op.drop_table("listing_task")
```

- [ ] **Step 5: Run migration and smoke test**

Run:

```bash
cd backend
.venv/bin/alembic upgrade head
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_listing_tasks.py::TestListingTaskModel::test_listing_task_model_is_mapped -q
```

Expected:

- Migration applies.
- Smoke test passes.

- [ ] **Step 6: Commit model and migration**

Run:

```bash
git add backend/app/models.py backend/alembic/versions/20260615_04_add_listing_tasks.py backend/tests/test_listing_tasks.py
git commit -m "feat: add listing task model"
```

## Task 2: Schemas And Task Service

**Files:**
- Create: `backend/app/listing/task_schemas.py`
- Create: `backend/app/listing/task_service.py`
- Modify: `backend/tests/test_listing_tasks.py`

- [ ] **Step 1: Add failing service tests**

Append to `backend/tests/test_listing_tasks.py`:

```python
from app.listing.task_schemas import (
    ListingTaskCreateFromDecisionItem,
    ListingTaskCreateFromDecisionRequest,
)
from app.listing.task_service import ListingTaskService


async def _create_platform(async_client, code: str = "mock"):
    resp = await async_client.post(
        "/api/platforms",
        json={
            "name": f"Mock {code}",
            "code": code,
            "api_base_url": f"https://{code}.example.com",
            "api_key": "secret-key",
        },
    )
    assert resp.status_code == 200
    return resp.json()["data"]


async def _create_product_with_sku(async_client, publishable: bool = True):
    payload = {
        "name": f"上架任务商品-{uuid4().hex[:6]}",
        "unit": "件",
        "status": 1,
    }
    if publishable:
        payload.update({
            "main_image": "/static/demo.jpg",
            "package_length_cm": 20,
            "package_width_cm": 12,
            "package_height_cm": 8,
            "package_weight_kg": 1.2,
        })
    resp = await async_client.post("/api/products", json=payload)
    assert resp.status_code == 200
    product = resp.json()["data"]

    await async_client.post(
        f"/api/products/{product['id']}/specs",
        json={"specs": [{"name": "颜色", "values": ["黑色"]}]},
    )
    sku_resp = await async_client.post(f"/api/products/{product['id']}/skus/generate")
    sku = sku_resp.json()["data"]["skus"][0]
    await async_client.post(
        "/api/prices",
        json={"sku_id": sku["id"], "price_type": "sale_price", "price": 99},
    )
    await async_client.put(
        f"/api/inventory/{sku['id']}",
        json={"quantity": 5, "warehouse": "默认仓库", "safety_stock": 1},
    )
    return product, sku


def _decision_result(sku_id: int, recommendation: str = "approve") -> dict:
    return {
        "sku_id": sku_id,
        "destination_country": "RU",
        "target_sale_price": 5000,
        "product_cost": 500,
        "shipping_fee": 60,
        "platform_fee": 500,
        "payment_fee": 150,
        "fixed_fee": 0,
        "advertising_fee": 0,
        "other_fee": 100,
        "profit_amount": 3690,
        "profit_margin": 73.8,
        "recommendation": recommendation,
        "blocking_reasons": [],
        "warnings": [],
        "platform_fee_source": "manual",
    }


class TestListingTaskService:
    async def test_create_from_approved_decision_creates_ready_task(self, async_client):
        platform = await _create_platform(async_client, "mocktaskready")
        product, sku = await _create_product_with_sku(async_client, publishable=True)

        async with async_session_factory() as session:
            result = await ListingTaskService.create_from_decisions(
                session,
                ListingTaskCreateFromDecisionRequest(
                    items=[
                        ListingTaskCreateFromDecisionItem(
                            item_key="row-1",
                            sku_id=sku["id"],
                            platform_id=platform["id"],
                            decision_result=_decision_result(sku["id"]),
                        )
                    ]
                ),
                operator="tester",
            )
            await session.commit()

        assert result.created_count == 1
        assert result.reused_count == 0
        assert result.skipped_count == 0
        task = result.tasks[0]
        assert task.product_id == product["id"]
        assert task.platform_id == platform["id"]
        assert task.status == "ready"
        assert task.missing_requirements == []

    async def test_create_from_approved_decision_creates_blocked_task_when_product_incomplete(self, async_client):
        platform = await _create_platform(async_client, "mocktaskblocked")
        product, sku = await _create_product_with_sku(async_client, publishable=False)

        async with async_session_factory() as session:
            result = await ListingTaskService.create_from_decisions(
                session,
                ListingTaskCreateFromDecisionRequest(
                    items=[
                        ListingTaskCreateFromDecisionItem(
                            item_key="row-2",
                            sku_id=sku["id"],
                            platform_id=platform["id"],
                            decision_result=_decision_result(sku["id"]),
                        )
                    ]
                ),
                operator="tester",
            )
            await session.commit()

        task = result.tasks[0]
        assert task.product_id == product["id"]
        assert task.status == "blocked"
        assert "main_image" in task.missing_requirements
        assert "logistics" in task.missing_requirements

    async def test_create_from_reject_decision_skips_item(self, async_client):
        platform = await _create_platform(async_client, "mocktaskskip")
        _product, sku = await _create_product_with_sku(async_client, publishable=True)

        async with async_session_factory() as session:
            result = await ListingTaskService.create_from_decisions(
                session,
                ListingTaskCreateFromDecisionRequest(
                    items=[
                        ListingTaskCreateFromDecisionItem(
                            item_key="row-3",
                            sku_id=sku["id"],
                            platform_id=platform["id"],
                            decision_result=_decision_result(sku["id"], recommendation="reject"),
                        )
                    ]
                ),
                operator="tester",
            )

        assert result.created_count == 0
        assert result.skipped_count == 1
        assert result.skipped[0].reason == "recommendation_not_approve"
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_listing_tasks.py::TestListingTaskService -q
```

Expected:

- FAIL because task schemas/service do not exist.

- [ ] **Step 3: Create schemas**

Create `backend/app/listing/task_schemas.py`:

```python
"""上架任务队列 Schema。"""

from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field

from app.decision.schemas import PreListingDecisionResponse


class ListingTaskCreateFromDecisionItem(BaseModel):
    item_key: Optional[str] = Field(None, max_length=100)
    sku_id: int
    platform_id: int
    decision_result: PreListingDecisionResponse


class ListingTaskCreateFromDecisionRequest(BaseModel):
    items: list[ListingTaskCreateFromDecisionItem] = Field(..., min_length=1, max_length=100)


class ListingTaskVO(BaseModel):
    id: int
    product_id: int
    product_name: Optional[str] = None
    platform_id: int
    platform_name: Optional[str] = None
    sku_id: Optional[int] = None
    product_listing_id: Optional[int] = None
    source_type: str
    source_item_key: Optional[str] = None
    status: str
    missing_requirements: list[str]
    decision_snapshot: Optional[dict] = None
    target_sale_price: Optional[float] = None
    target_profit_margin: Optional[float] = None
    destination_country: Optional[str] = None
    last_error: Optional[str] = None
    created_by: Optional[str] = None
    updated_by: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


class ListingTaskSkippedItem(BaseModel):
    item_key: Optional[str] = None
    sku_id: Optional[int] = None
    platform_id: Optional[int] = None
    reason: str


class ListingTaskCreateFromDecisionResponse(BaseModel):
    created_count: int
    reused_count: int
    skipped_count: int
    tasks: list[ListingTaskVO]
    skipped: list[ListingTaskSkippedItem]
```

- [ ] **Step 4: Create service**

Create `backend/app/listing/task_service.py`:

```python
"""上架任务队列服务。"""

from decimal import Decimal
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.listing.service import ListingService, PublishFailedError, PublishValidationError
from app.listing.task_schemas import (
    ListingTaskCreateFromDecisionRequest,
    ListingTaskCreateFromDecisionResponse,
    ListingTaskSkippedItem,
    ListingTaskVO,
)
from app.models import ListingTask, Platform, Product, Sku


OPEN_STATUSES = {"ready", "blocked", "failed"}


def _to_float(value) -> Optional[float]:
    return float(value) if value is not None else None


def task_to_vo(task: ListingTask) -> ListingTaskVO:
    return ListingTaskVO(
        id=task.id,
        product_id=task.product_id,
        product_name=task.product.name if task.product else None,
        platform_id=task.platform_id,
        platform_name=task.platform.name if task.platform else None,
        sku_id=task.sku_id,
        product_listing_id=task.product_listing_id,
        source_type=task.source_type,
        source_item_key=task.source_item_key,
        status=task.status,
        missing_requirements=list(task.missing_requirements or []),
        decision_snapshot=task.decision_snapshot,
        target_sale_price=_to_float(task.target_sale_price),
        target_profit_margin=_to_float(task.target_profit_margin),
        destination_country=task.destination_country,
        last_error=task.last_error,
        created_by=task.created_by,
        updated_by=task.updated_by,
        created_at=task.created_at,
        updated_at=task.updated_at,
    )


class ListingTaskService:
    @staticmethod
    async def _existing_open_task(
        db: AsyncSession,
        product_id: int,
        platform_id: int,
    ) -> Optional[ListingTask]:
        result = await db.execute(
            select(ListingTask)
            .options(selectinload(ListingTask.product), selectinload(ListingTask.platform))
            .where(
                ListingTask.product_id == product_id,
                ListingTask.platform_id == platform_id,
                ListingTask.status.in_(OPEN_STATUSES),
            )
            .order_by(ListingTask.id.desc())
        )
        return result.scalars().first()

    @staticmethod
    async def _readiness_status(
        db: AsyncSession,
        product: Product,
        platform: Platform,
    ) -> tuple[str, list[str]]:
        missing, _skus, _prices, _inventories = await ListingService.validate_publish_ready(
            db,
            product,
            platform,
        )
        return ("blocked" if missing else "ready"), missing

    @staticmethod
    async def create_from_decisions(
        db: AsyncSession,
        req: ListingTaskCreateFromDecisionRequest,
        operator: str,
    ) -> ListingTaskCreateFromDecisionResponse:
        created_count = 0
        reused_count = 0
        tasks: list[ListingTaskVO] = []
        skipped: list[ListingTaskSkippedItem] = []

        for item in req.items:
            if item.decision_result.recommendation != "approve":
                skipped.append(
                    ListingTaskSkippedItem(
                        item_key=item.item_key,
                        sku_id=item.sku_id,
                        platform_id=item.platform_id,
                        reason="recommendation_not_approve",
                    )
                )
                continue

            sku = await db.get(Sku, item.sku_id)
            platform = await db.get(Platform, item.platform_id)
            if sku is None:
                skipped.append(ListingTaskSkippedItem(item_key=item.item_key, sku_id=item.sku_id, platform_id=item.platform_id, reason="sku_not_found"))
                continue
            if platform is None:
                skipped.append(ListingTaskSkippedItem(item_key=item.item_key, sku_id=item.sku_id, platform_id=item.platform_id, reason="platform_not_found"))
                continue
            product = await db.get(Product, sku.product_id)
            if product is None:
                skipped.append(ListingTaskSkippedItem(item_key=item.item_key, sku_id=item.sku_id, platform_id=item.platform_id, reason="product_not_found"))
                continue

            status, missing = await ListingTaskService._readiness_status(db, product, platform)
            task = await ListingTaskService._existing_open_task(db, product.id, platform.id)
            if task:
                reused_count += 1
            else:
                task = ListingTask(product_id=product.id, platform_id=platform.id)
                db.add(task)
                created_count += 1

            task.sku_id = item.sku_id
            task.source_type = "decision"
            task.source_item_key = item.item_key
            task.status = status
            task.missing_requirements = missing
            task.decision_snapshot = item.decision_result.model_dump()
            task.target_sale_price = Decimal(str(item.decision_result.target_sale_price))
            task.target_profit_margin = Decimal(str(item.decision_result.profit_margin))
            task.destination_country = item.decision_result.destination_country
            task.last_error = None
            task.updated_by = operator
            task.created_by = task.created_by or operator
            await db.flush()
            await db.refresh(task, ["product", "platform"])
            tasks.append(task_to_vo(task))

        return ListingTaskCreateFromDecisionResponse(
            created_count=created_count,
            reused_count=reused_count,
            skipped_count=len(skipped),
            tasks=tasks,
            skipped=skipped,
        )

    @staticmethod
    async def list(
        db: AsyncSession,
        status: Optional[str] = None,
        platform_id: Optional[int] = None,
    ) -> list[ListingTaskVO]:
        stmt = select(ListingTask).options(
            selectinload(ListingTask.product),
            selectinload(ListingTask.platform),
        )
        if status:
            stmt = stmt.where(ListingTask.status == status)
        if platform_id is not None:
            stmt = stmt.where(ListingTask.platform_id == platform_id)
        stmt = stmt.order_by(ListingTask.created_at.desc(), ListingTask.id.desc())
        result = await db.execute(stmt)
        return [task_to_vo(task) for task in result.scalars().all()]

    @staticmethod
    async def recheck(db: AsyncSession, task_id: int, operator: str) -> Optional[ListingTaskVO]:
        task = await db.get(ListingTask, task_id)
        if not task:
            return None
        product = await db.get(Product, task.product_id)
        platform = await db.get(Platform, task.platform_id)
        if not product or not platform:
            task.status = "blocked"
            task.missing_requirements = ["product" if not product else "platform"]
        else:
            task.status, task.missing_requirements = await ListingTaskService._readiness_status(db, product, platform)
        task.updated_by = operator
        await db.flush()
        await db.refresh(task, ["product", "platform"])
        return task_to_vo(task)

    @staticmethod
    async def cancel(db: AsyncSession, task_id: int, operator: str) -> Optional[ListingTaskVO]:
        task = await db.get(ListingTask, task_id)
        if not task:
            return None
        task.status = "cancelled"
        task.updated_by = operator
        await db.flush()
        await db.refresh(task, ["product", "platform"])
        return task_to_vo(task)

    @staticmethod
    async def publish(db: AsyncSession, task_id: int, operator: str) -> Optional[ListingTaskVO]:
        task = await db.get(ListingTask, task_id)
        if not task:
            return None
        if task.status != "ready":
            task.last_error = "只有 ready 状态的上架任务可以发布"
            await db.flush()
            await db.refresh(task, ["product", "platform"])
            return task_to_vo(task)

        product = await db.get(Product, task.product_id)
        platform = await db.get(Platform, task.platform_id)
        if not product or not platform:
            task.status = "blocked"
            task.missing_requirements = ["product" if not product else "platform"]
            task.updated_by = operator
            await db.flush()
            await db.refresh(task, ["product", "platform"])
            return task_to_vo(task)

        try:
            listing = await ListingService.publish(db, product, platform)
            task.status = "published"
            task.product_listing_id = listing.id
            task.last_error = None
        except PublishValidationError as exc:
            task.status = "blocked"
            task.missing_requirements = exc.missing_requirements
            task.last_error = "商品信息不完整"
        except PublishFailedError as exc:
            task.status = "failed"
            task.product_listing_id = exc.listing.id
            task.last_error = exc.listing.sync_message
        task.updated_by = operator
        await db.flush()
        await db.refresh(task, ["product", "platform"])
        return task_to_vo(task)
```

- [ ] **Step 5: Run service tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_listing_tasks.py::TestListingTaskService -q
```

Expected:

- PASS.

- [ ] **Step 6: Commit schemas and service**

Run:

```bash
git add backend/app/listing/task_schemas.py backend/app/listing/task_service.py backend/tests/test_listing_tasks.py
git commit -m "feat: create listing tasks from decisions"
```

## Task 3: Task APIs, Permissions, And Audit Logs

**Files:**
- Modify: `backend/app/listing/router.py`
- Modify: `backend/seed.py`
- Modify: `backend/tests/test_listing_tasks.py`

- [ ] **Step 1: Add failing API tests**

Append to `backend/tests/test_listing_tasks.py`:

```python
from tests.auth_helpers import enable_auth, grant_permission, register_and_login


pytestmark = pytest.mark.usefixtures("enable_auth")


async def _auth(async_client, prefix: str, permissions: list[str]):
    uid, token = await register_and_login(async_client, prefix)
    for permission in permissions:
        await grant_permission(uid, permission)
    return {"Authorization": f"Bearer {token}"}


class TestListingTaskApi:
    async def test_create_from_decisions_requires_task_manage_permission(self, async_client):
        headers = await _auth(async_client, "lt_no_manage", [])
        resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={"items": []},
            headers=headers,
        )
        assert resp.status_code == 403

    async def test_create_list_recheck_cancel_with_permissions(self, async_client):
        headers = await _auth(
            async_client,
            "lt_manage",
            ["listing:task_manage", "listing:view"],
        )
        platform = await _create_platform(async_client, "mocktaskapi")
        _product, sku = await _create_product_with_sku(async_client, publishable=True)

        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "row-api",
                        "sku_id": sku["id"],
                        "platform_id": platform["id"],
                        "decision_result": _decision_result(sku["id"]),
                    }
                ]
            },
            headers=headers,
        )
        assert create_resp.status_code == 200, create_resp.text
        data = create_resp.json()["data"]
        assert data["created_count"] == 1
        task = data["tasks"][0]
        assert task["status"] == "ready"
        assert await _count_logs("listing_task", "create_from_decision", str(task["id"])) == 1

        list_resp = await async_client.get("/api/listing-tasks?status=ready", headers=headers)
        assert list_resp.status_code == 200
        assert any(item["id"] == task["id"] for item in list_resp.json()["data"])

        recheck_resp = await async_client.post(f"/api/listing-tasks/{task['id']}/recheck", headers=headers)
        assert recheck_resp.status_code == 200
        assert recheck_resp.json()["data"]["status"] == "ready"

        cancel_resp = await async_client.post(f"/api/listing-tasks/{task['id']}/cancel", headers=headers)
        assert cancel_resp.status_code == 200
        assert cancel_resp.json()["data"]["status"] == "cancelled"

    async def test_publish_ready_task_requires_publish_permission(self, async_client):
        manage_headers = await _auth(async_client, "lt_publish_manage", ["listing:task_manage"])
        platform = await _create_platform(async_client, "mocktaskpublishperm")
        _product, sku = await _create_product_with_sku(async_client, publishable=True)
        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "row-publish",
                        "sku_id": sku["id"],
                        "platform_id": platform["id"],
                        "decision_result": _decision_result(sku["id"]),
                    }
                ]
            },
            headers=manage_headers,
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        no_publish_headers = await _auth(async_client, "lt_no_publish", [])
        resp = await async_client.post(f"/api/listing-tasks/{task_id}/publish", headers=no_publish_headers)
        assert resp.status_code == 403

    async def test_publish_ready_task_marks_published_and_logs(self, async_client):
        headers = await _auth(
            async_client,
            "lt_publish_ok",
            ["listing:task_manage", "listing:publish"],
        )
        platform = await _create_platform(async_client, "mocktaskpublish")
        _product, sku = await _create_product_with_sku(async_client, publishable=True)
        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "row-publish-ok",
                        "sku_id": sku["id"],
                        "platform_id": platform["id"],
                        "decision_result": _decision_result(sku["id"]),
                    }
                ]
            },
            headers=headers,
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        resp = await async_client.post(f"/api/listing-tasks/{task_id}/publish", headers=headers)

        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["status"] == "published"
        assert data["product_listing_id"] is not None
        assert await _count_logs("listing_task", "publish", str(task_id)) == 1
```

- [ ] **Step 2: Run API tests to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_listing_tasks.py::TestListingTaskApi -q
```

Expected:

- FAIL because routes do not exist.

- [ ] **Step 3: Add permission seed**

In `backend/seed.py`, append to `SEED_PERMISSIONS`:

```python
    {"code": "listing:task_manage", "name": "管理上架任务", "module": "listing"},
```

- [ ] **Step 4: Add task routes**

In `backend/app/listing/router.py`, add imports:

```python
from typing import Optional

from app.listing.task_schemas import ListingTaskCreateFromDecisionRequest
from app.listing.task_service import ListingTaskService
```

Add these routes before the product publish route:

```python
@router.post("/listing-tasks/from-decisions", summary="从决策结果生成上架任务")
async def create_listing_tasks_from_decisions(
    data: ListingTaskCreateFromDecisionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:task_manage")),
):
    result = await ListingTaskService.create_from_decisions(db, data, current_user.username)
    for task in result.tasks:
        await OperationLogService.log(
            db,
            module="listing_task",
            action="create_from_decision",
            resource_id=str(task.id),
            content=f"从决策结果生成上架任务: product_id={task.product_id}, platform_id={task.platform_id}, status={task.status}",
            operator=current_user.username,
        )
    return Result.ok(result)


@router.get("/listing-tasks", summary="上架任务列表")
async def list_listing_tasks(
    status: Optional[str] = None,
    platform_id: Optional[int] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:view")),
):
    return Result.ok(await ListingTaskService.list(db, status, platform_id))


@router.post("/listing-tasks/{task_id}/recheck", summary="重新检查上架任务")
async def recheck_listing_task(
    task_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:task_manage")),
):
    task = await ListingTaskService.recheck(db, task_id, current_user.username)
    if not task:
        return Result.not_found("上架任务不存在")
    await OperationLogService.log(
        db,
        module="listing_task",
        action="recheck",
        resource_id=str(task_id),
        content=f"重新检查上架任务: status={task.status}",
        operator=current_user.username,
    )
    return Result.ok(task)


@router.post("/listing-tasks/{task_id}/cancel", summary="取消上架任务")
async def cancel_listing_task(
    task_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:task_manage")),
):
    task = await ListingTaskService.cancel(db, task_id, current_user.username)
    if not task:
        return Result.not_found("上架任务不存在")
    await OperationLogService.log(
        db,
        module="listing_task",
        action="cancel",
        resource_id=str(task_id),
        content="取消上架任务",
        operator=current_user.username,
    )
    return Result.ok(task)


@router.post("/listing-tasks/{task_id}/publish", summary="发布上架任务")
async def publish_listing_task(
    task_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("listing:publish")),
):
    task = await ListingTaskService.publish(db, task_id, current_user.username)
    if not task:
        return Result.not_found("上架任务不存在")
    await OperationLogService.log(
        db,
        module="listing_task",
        action="publish",
        resource_id=str(task_id),
        content=f"发布上架任务: status={task.status}, product_listing_id={task.product_listing_id}",
        operator=current_user.username,
    )
    return Result.ok(task)
```

- [ ] **Step 5: Run API tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_listing_tasks.py -q
```

Expected:

- All listing task tests pass.

- [ ] **Step 6: Commit APIs**

Run:

```bash
git add backend/app/listing/router.py backend/seed.py backend/tests/test_listing_tasks.py
git commit -m "feat: add listing task api"
```

## Task 4: Frontend API Client And Task Queue Page

**Files:**
- Create: `frontend/src/api/modules/listing.ts`
- Create: `frontend/src/views/listing/ListingTaskQueue.vue`
- Modify: `frontend/src/router/modules/listing.ts`

- [ ] **Step 1: Create frontend API module**

Create `frontend/src/api/modules/listing.ts`:

```ts
import http from '@/api/http'
import type { PreListingDecisionResponse } from '@/api/modules/decision'

export interface ListingTaskCreateFromDecisionItem {
  item_key?: string | null
  sku_id: number
  platform_id: number
  decision_result: PreListingDecisionResponse
}

export interface ListingTask {
  id: number
  product_id: number
  product_name?: string | null
  platform_id: number
  platform_name?: string | null
  sku_id?: number | null
  product_listing_id?: number | null
  source_type: string
  source_item_key?: string | null
  status: string
  missing_requirements: string[]
  decision_snapshot?: Record<string, any> | null
  target_sale_price?: number | null
  target_profit_margin?: number | null
  destination_country?: string | null
  last_error?: string | null
  created_at?: string | null
  updated_at?: string | null
}

export interface ListingTaskCreateResponse {
  created_count: number
  reused_count: number
  skipped_count: number
  tasks: ListingTask[]
  skipped: Array<{
    item_key?: string | null
    sku_id?: number | null
    platform_id?: number | null
    reason: string
  }>
}

export function createListingTasksFromDecisions(items: ListingTaskCreateFromDecisionItem[]) {
  return http.post('/listing-tasks/from-decisions', { items })
}

export function getListingTasks(params?: { status?: string; platform_id?: number }) {
  return http.get('/listing-tasks', { params })
}

export function recheckListingTask(taskId: number) {
  return http.post(`/listing-tasks/${taskId}/recheck`)
}

export function cancelListingTask(taskId: number) {
  return http.post(`/listing-tasks/${taskId}/cancel`)
}

export function publishListingTask(taskId: number) {
  return http.post(`/listing-tasks/${taskId}/publish`)
}
```

- [ ] **Step 2: Create task queue page**

Create `frontend/src/views/listing/ListingTaskQueue.vue`:

```vue
<template>
  <div>
    <n-page-header subtitle="从上架决策生成的待发布任务">
      <template #title>上架任务队列</template>
    </n-page-header>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space style="margin-bottom: 12px;">
        <n-select
          v-model:value="status"
          :options="statusOptions"
          clearable
          placeholder="任务状态"
          style="width: 180px;"
        />
        <n-button type="primary" :loading="loading" @click="fetchTasks">查询</n-button>
      </n-space>

      <n-data-table :columns="columns" :data="tasks" :loading="loading" :pagination="{ pageSize: 20 }" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NSpace, NTag, useMessage } from 'naive-ui'
import {
  cancelListingTask,
  getListingTasks,
  publishListingTask,
  recheckListingTask,
  type ListingTask,
} from '@/api/modules/listing'

const message = useMessage()
const loading = ref(false)
const tasks = ref<ListingTask[]>([])
const status = ref<string | null>(null)

const statusOptions = [
  { label: '可发布', value: 'ready' },
  { label: '阻塞', value: 'blocked' },
  { label: '已发布', value: 'published' },
  { label: '失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' },
]

const statusTag: Record<string, { type: 'success' | 'warning' | 'error' | 'default'; text: string }> = {
  ready: { type: 'success', text: '可发布' },
  blocked: { type: 'warning', text: '阻塞' },
  published: { type: 'success', text: '已发布' },
  failed: { type: 'error', text: '失败' },
  cancelled: { type: 'default', text: '已取消' },
}

async function fetchTasks() {
  loading.value = true
  try {
    const resp = await getListingTasks({ status: status.value || undefined })
    tasks.value = resp.data || []
  } catch (err: any) {
    message.error(err?.message || '查询上架任务失败')
  } finally {
    loading.value = false
  }
}

async function handleRecheck(row: ListingTask) {
  try {
    await recheckListingTask(row.id)
    message.success('检查完成')
    await fetchTasks()
  } catch (err: any) {
    message.error(err?.message || '检查失败')
  }
}

async function handlePublish(row: ListingTask) {
  try {
    await publishListingTask(row.id)
    message.success('发布完成')
    await fetchTasks()
  } catch (err: any) {
    message.error(err?.message || '发布失败')
  }
}

async function handleCancel(row: ListingTask) {
  try {
    await cancelListingTask(row.id)
    message.success('已取消')
    await fetchTasks()
  } catch (err: any) {
    message.error(err?.message || '取消失败')
  }
}

const columns = [
  { title: '商品', key: 'product_name', ellipsis: { tooltip: true } },
  { title: '平台', key: 'platform_name', width: 120 },
  { title: '目的国', key: 'destination_country', width: 90 },
  { title: '目标售价', key: 'target_sale_price', width: 110 },
  { title: '利润率', key: 'target_profit_margin', width: 100, render: (row: ListingTask) => row.target_profit_margin == null ? '-' : `${row.target_profit_margin}%` },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: ListingTask) => {
      const meta = statusTag[row.status] || { type: 'default', text: row.status }
      return h(NTag, { type: meta.type, size: 'small' }, { default: () => meta.text })
    },
  },
  {
    title: '缺失项/错误',
    key: 'missing',
    ellipsis: { tooltip: true },
    render: (row: ListingTask) => row.last_error || row.missing_requirements.join('；') || '-',
  },
  {
    title: '操作',
    key: 'actions',
    width: 260,
    render: (row: ListingTask) =>
      h(NSpace, null, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => handleRecheck(row), disabled: row.status === 'published' || row.status === 'cancelled' }, { default: () => '重检' }),
          h(NButton, { size: 'small', type: 'primary', onClick: () => handlePublish(row), disabled: row.status !== 'ready' }, { default: () => '发布' }),
          h(NButton, { size: 'small', onClick: () => handleCancel(row), disabled: row.status === 'published' || row.status === 'cancelled' }, { default: () => '取消' }),
        ],
      }),
  },
]

onMounted(fetchTasks)
</script>
```

- [ ] **Step 3: Add route**

If `frontend/src/router/modules/listing.ts` exists, add:

```ts
{
  path: 'listing-tasks',
  name: 'ListingTaskQueue',
  component: () => import('@/views/listing/ListingTaskQueue.vue'),
  meta: {
    title: '上架任务',
    icon: 'list',
    menu: true,
    perm: 'listing:view',
  },
}
```

If the file does not exist, create `frontend/src/router/modules/listing.ts`:

```ts
import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'listings',
    name: 'ListingManage',
    component: () => import('@/views/listing/ListingManage.vue'),
    meta: {
      title: '发布管理',
      icon: 'send',
      menu: true,
      perm: 'listing:view',
    },
  },
  {
    path: 'listing-tasks',
    name: 'ListingTaskQueue',
    component: () => import('@/views/listing/ListingTaskQueue.vue'),
    meta: {
      title: '上架任务',
      icon: 'list',
      menu: true,
      perm: 'listing:view',
    },
  },
]
```

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 5: Commit frontend task queue**

Run:

```bash
git add frontend/src/api/modules/listing.ts frontend/src/views/listing/ListingTaskQueue.vue frontend/src/router/modules/listing.ts
git commit -m "feat: add listing task queue page"
```

## Task 5: Generate Listing Tasks From Batch Decision Page

**Files:**
- Modify: `frontend/src/views/decision/BatchPreListingDecision.vue`

- [ ] **Step 1: Extend imports**

In `frontend/src/views/decision/BatchPreListingDecision.vue`, add:

```ts
import { createListingTasksFromDecisions } from '@/api/modules/listing'
```

- [ ] **Step 2: Add toolbar button**

In the result summary/card actions area, add a button visible after calculation:

```vue
<n-button
  v-if="batchResult"
  type="primary"
  @click="handleCreateListingTasks"
>
  生成上架任务
</n-button>
```

If there is no actions area after calculation, place it beside the existing "导出结果" button.

- [ ] **Step 3: Add handler**

Add this function:

```ts
async function handleCreateListingTasks() {
  if (!batchResult.value) return
  const approved = batchResult.value.items.filter((item) => item.status === 'success' && item.result?.recommendation === 'approve')
  if (approved.length === 0) {
    message.warning('没有可生成上架任务的 approve 结果')
    return
  }

  const platformByKey = new Map(rows.map((row) => [row.item_key || row.key, row.platform_id]))
  const items = approved
    .map((item) => {
      const platformId = platformByKey.get(item.item_key || '')
      if (!item.result || !item.sku_id || !platformId) return null
      return {
        item_key: item.item_key,
        sku_id: item.sku_id,
        platform_id: platformId,
        decision_result: item.result,
      }
    })
    .filter(Boolean) as any[]

  if (items.length === 0) {
    message.warning('approve 结果缺少平台ID，无法生成上架任务')
    return
  }

  try {
    const resp = await createListingTasksFromDecisions(items)
    const data = resp.data
    message.success(`生成完成：新建 ${data.created_count}，复用 ${data.reused_count}，跳过 ${data.skipped_count}`)
  } catch (err: any) {
    message.error(err?.message || '生成上架任务失败')
  }
}
```

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 5: Commit decision page integration**

Run:

```bash
git add frontend/src/views/decision/BatchPreListingDecision.vue
git commit -m "feat: create listing tasks from batch decisions"
```

## Task 6: Docs And Final Verification

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`
- Modify: `docs/PERMISSIONS_AND_AUDIT.md`

- [ ] **Step 1: Update docs**

In `docs/PROJECT_STATUS.md`, add:

```markdown
### 决策到上架任务

状态：已完成第一版。

已实现：
- 批量上架决策 approve 结果可生成上架任务。
- 上架任务按商品和平台去重，避免重复任务。
- 上架任务复用发布前检查，缺商品图、SKU、价格、库存、物流数据时进入 blocked。
- ready 任务可调用现有发布 adapter 发布。
- 任务创建、重检、取消、发布接入权限和审计日志。

暂未实现：
- 真实平台 API。
- 平台类目映射。
- 平台属性映射。
- 发布失败自动重试队列。
```

In `docs/PERMISSIONS_AND_AUDIT.md`, add:

```markdown
| 上架任务 | `listing:view`, `listing:task_manage`, `listing:publish` | create_from_decision, recheck, cancel, publish |
```

In `docs/ROADMAP.md`, set recommended next task:

````markdown
最推荐继续做：

```text
物流账单导入与运费对账。
```

原因：

- 上架决策已经能进入上架任务队列。
- 下一步要补马帮 ERP 的物流成本优势：真实运费、账单导入、预估与实际差异。
- 运费对账完成后，订单利润才有真实成本基础。
````

- [ ] **Step 2: Run focused backend tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_listing_tasks.py tests/test_listing.py tests/test_platform_listing_auth_audit.py -q
```

Expected:

- Focused listing tests pass.

- [ ] **Step 3: Run backend full suite**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest -q
```

Expected:

- Full backend suite passes.

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 5: Check git state**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
```

Expected:

- Working tree is clean.
- Recent commits show model, service/API, frontend queue, decision page integration, docs.

## Final Acceptance Criteria

This stage is complete only when:

- `listing_task` table exists.
- Approved batch decision rows can create listing tasks.
- Reject/needs_data rows are skipped.
- Duplicate open tasks are reused.
- Incomplete products create `blocked` tasks with `missing_requirements`.
- Complete products create `ready` tasks.
- Ready tasks can publish through existing listing adapter.
- Non-ready tasks cannot publish.
- Task create/recheck/cancel/publish operations are permission-protected.
- Task create/recheck/cancel/publish operations are audited.
- Frontend can generate tasks from batch decision results.
- Frontend can view task queue, recheck, cancel, and publish tasks.
- Focused backend tests pass.
- Backend full suite passes.
- Frontend build passes.

## Recommended Agent Prompt

Give this to the implementing agent:

```text
你接手的是 /Users/lc/multisell 的 LingMirror / MultiSell 项目。

先阅读：
- docs/superpowers/plans/2026-06-15-decision-to-listing-task.md
- docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md
- backend/app/listing/service.py
- backend/app/listing/router.py
- backend/app/decision/schemas.py
- backend/app/models.py
- backend/tests/test_listing.py
- frontend/src/views/decision/BatchPreListingDecision.vue
- frontend/src/views/listing/ListingManage.vue

请在新分支 codex/decision-to-listing-task 上按计划逐任务执行。严格 TDD：先写失败测试，再写实现。只做 Stage 1：从批量决策 approve 结果生成上架任务。不要做真实平台 API，不要做物流账单对账，不要做 WMS，不要做 BI。

完成后必须运行：
- cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
- cd frontend && npm run build

交付时说明：
- 新增了哪些数据模型和 API
- listing task 状态流如何设计
- 如何从决策结果生成任务
- 如何阻止缺数据任务进入发布
- 测试命令和结果
- 剩余限制
```
