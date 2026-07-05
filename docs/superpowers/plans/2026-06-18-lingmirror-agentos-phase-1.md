> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# LingMirror AgentOS Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Phase 1 AgentOS shell: a cross-border e-commerce AI workbench with unified work items, control-center data APIs, Agent squad views, autonomy visibility, approval surfaces, and built-in template entry points.

**Architecture:** Add a focused `backend/app/agentos/` module that aggregates existing Agent decisions, pending actions, exceptions, and notifications into a `WorkItem` view model without introducing a new database table in Phase 1. Add frontend AgentOS routes and pages that consume the new APIs while leaving existing product, order, inventory, settlement, exception, notification, and Agent pages intact as drill-down surfaces.

**Tech Stack:** FastAPI, async SQLAlchemy 2.0, Pydantic, PostgreSQL, pytest/httpx ASGI transport, Vue 3, TypeScript, Naive UI, Vite.

---

## Scope Guard

This plan implements the Phase 1 vertical slice only:

- AgentOS control center
- Unified task/work-item API and UI
- Agent squad overview
- Built-in template cards
- Autonomy and safety visibility

This plan does not implement a desktop client, IM two-way approval, Skill marketplace, generic workflow canvas, deep external platform automation, or full Trust Score automation.

The worktree is currently dirty. Before editing any touched file, read its current contents and preserve unrelated changes.

## File Structure

Create:

- `backend/app/agentos/__init__.py` - exports the AgentOS router for app auto-discovery.
- `backend/app/agentos/schemas.py` - Pydantic response models for summaries, work items, squads, and templates.
- `backend/app/agentos/service.py` - aggregation and normalization logic; no mutations.
- `backend/app/agentos/router.py` - `/agentos/*` API endpoints.
- `backend/tests/test_agentos_service.py` - pure normalization and squad/template tests.
- `backend/tests/test_agentos_router.py` - API contract tests through ASGI client.
- `frontend/src/api/modules/agentos.ts` - typed AgentOS API client.
- `frontend/src/router/modules/agentos.ts` - route definitions for AgentOS pages.
- `frontend/src/views/agentos/ControlCenter.vue` - main AgentOS workbench.
- `frontend/src/views/agentos/TaskCenter.vue` - unified task list.
- `frontend/src/views/agentos/AgentSquads.vue` - squad and autonomy overview.
- `frontend/src/views/agentos/components/WorkItemCard.vue` - reusable work-item card.
- `frontend/src/views/agentos/components/SquadCard.vue` - reusable squad card.
- `frontend/src/views/agentos/components/TemplateCard.vue` - reusable template card.

Modify:

- `frontend/src/components/Layout.vue` - add icon mappings only if the new route meta icons are not already supported.
- `frontend/src/api/index.ts` - no change expected if modules are auto-merged; verify after adding `agentos.ts`.

## Task 1: Backend AgentOS Schemas And Pure Normalizers

**Files:**

- Create: `backend/app/agentos/__init__.py`
- Create: `backend/app/agentos/schemas.py`
- Create: `backend/app/agentos/service.py`
- Test: `backend/tests/test_agentos_service.py`

- [ ] **Step 1: Write failing service tests**

Create `backend/tests/test_agentos_service.py`:

```python
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_service.py -q
```

Expected: fail with `ModuleNotFoundError: No module named 'app.agentos'`.

- [ ] **Step 3: Create AgentOS module export**

Create `backend/app/agentos/__init__.py`:

```python
from app.agentos.router import router

__all__ = ["router"]
```

- [ ] **Step 4: Create response schemas**

Create `backend/app/agentos/schemas.py`:

```python
from datetime import datetime
from typing import Any

from pydantic import BaseModel, Field


class BusinessObjectVO(BaseModel):
    type: str | None = None
    id: str | None = None
    label: str | None = None


class WorkItemVO(BaseModel):
    id: str
    source_type: str
    source_id: str
    source_module: str | None = None
    business_object: BusinessObjectVO = Field(default_factory=BusinessObjectVO)
    squad: str
    agent_id: str | None = None
    title: str
    summary: str | None = None
    recommendation: str | None = None
    risk_level: str
    approval_required: bool = False
    status: str
    action_type: str | None = None
    context: dict[str, Any] = Field(default_factory=dict)
    audit_link: str | None = None
    created_at: datetime | None = None


class ControlCenterSummaryVO(BaseModel):
    sales_today: float = 0
    profit_today: float = 0
    inventory_risks: int = 0
    pending_approvals: int = 0
    active_work_items: int = 0
    agent_automation_rate: float = 0


class SquadVO(BaseModel):
    id: str
    name: str
    description: str
    agents: list[str]
    decision_count_7d: int = 0
    pending_approvals: int = 0
    risk_count: int = 0
    adoption_rate: float = 0
    autonomy_level: str = "suggestion"


class TemplateCardVO(BaseModel):
    id: str
    title: str
    squad: str
    description: str
    mode: str
    route: str
    phase: str = "phase_1"
```

- [ ] **Step 5: Create pure normalizers and constants**

Create `backend/app/agentos/service.py`:

```python
from __future__ import annotations

from datetime import datetime, timezone, timedelta
from typing import Any

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.agent.models import AgentAction as PendingAgentAction
from app.agent.models import AgentDecision
from app.models import ExceptionItem, Notification


AGENT_TO_SQUAD = {
    "A1": "growth",
    "A2": "growth",
    "A3": "growth",
    "A4": "growth",
    "A5": "fulfillment",
    "G2": "fulfillment",
    "A6": "risk",
    "A7": "risk",
    "G3": "risk",
    "G1": "risk",
}

MODULE_TO_SQUAD = {
    "product": "growth",
    "listing": "growth",
    "listing_task": "growth",
    "inventory": "fulfillment",
    "order": "fulfillment",
    "shipping": "fulfillment",
    "settlement": "risk",
    "finance": "risk",
    "platform_fee": "risk",
    "compliance": "risk",
}

ALERT_TO_SQUAD = {
    "inventory_low_stock": "fulfillment",
    "inventory_out_of_stock": "fulfillment",
    "order_pending": "fulfillment",
    "listing_failed": "growth",
    "settlement_pending": "risk",
    "settlement_discrepancy": "risk",
}

AGENT_SQUADS = [
    {
        "id": "growth",
        "name": "增长小队",
        "description": "负责选品、Listing、广告建议和上新质量。",
        "agents": ["A1", "A2", "A3", "A4"],
    },
    {
        "id": "fulfillment",
        "name": "履约小队",
        "description": "负责库存、订单、仓储、海关和物流闭环。",
        "agents": ["A5", "G2"],
    },
    {
        "id": "risk",
        "name": "风控小队",
        "description": "负责利润、折扣、合规和平台安全红线。",
        "agents": ["A6", "A7", "G3", "G1"],
    },
]

TEMPLATE_CARDS = [
    {
        "id": "pre_listing_decision",
        "title": "上架前经营决策",
        "squad": "risk",
        "description": "串联商品、物流、平台费和利润，判断是否值得上架。",
        "mode": "Agent",
        "route": "/decisions/prelisting",
    },
    {
        "id": "listing_optimization",
        "title": "Listing 优化",
        "squad": "growth",
        "description": "生成标题、描述、关键词和平台适配建议。",
        "mode": "Agent",
        "route": "/listings/ai-workbench",
    },
    {
        "id": "inventory_replenishment",
        "title": "库存补货",
        "squad": "fulfillment",
        "description": "识别低库存和断货风险，生成补货建议。",
        "mode": "Agent",
        "route": "/inventory/alerts",
    },
    {
        "id": "profit_risk",
        "title": "利润风控",
        "squad": "risk",
        "description": "识别毛利率下降、费用异常和折扣风险。",
        "mode": "Ask",
        "route": "/finance",
    },
    {
        "id": "customer_service_draft",
        "title": "客服草稿",
        "squad": "growth",
        "description": "生成多语言客服回复草稿，不自动回复真实客户。",
        "mode": "Ask",
        "route": "/agents/A4",
    },
]


class AgentOSService:
    @staticmethod
    def _iso(dt: Any) -> Any:
        return dt if not hasattr(dt, "isoformat") else dt

    @staticmethod
    def _squad_for_agent(agent_id: str | None) -> str:
        return AGENT_TO_SQUAD.get(agent_id or "", "risk")

    @staticmethod
    def _risk_from_severity(severity: str | None) -> str:
        mapping = {"critical": "critical", "error": "high", "warning": "medium", "info": "low"}
        return mapping.get(severity or "", "medium")

    @staticmethod
    def _risk_from_action(action_type: str | None, payload: dict[str, Any] | None = None) -> str:
        payload = payload or {}
        amount = abs(float(payload.get("amount") or payload.get("total_amount") or 0))
        sku_count = len(payload.get("sku_codes") or [])
        if amount >= 500 or sku_count > 20:
            return "high"
        if action_type in {"replenish", "price_adjust", "discount_review", "ad_action"}:
            return "medium"
        return "low"

    @staticmethod
    def _approval_required(risk_level: str, status: str) -> bool:
        return status == "pending" and risk_level in {"high", "critical"}

    @staticmethod
    def normalize_agent_pending_action(row: PendingAgentAction) -> dict[str, Any]:
        risk = AgentOSService._risk_from_action(row.action_type, row.action_payload)
        return {
            "id": f"agent_action:{row.id}",
            "source_type": "agent_action",
            "source_id": str(row.id),
            "source_module": "agent",
            "business_object": {"type": "decision", "id": str(row.decision_id) if row.decision_id else None},
            "squad": AgentOSService._squad_for_agent(row.agent_id),
            "agent_id": row.agent_id,
            "title": row.summary,
            "summary": row.summary,
            "recommendation": row.summary,
            "risk_level": risk,
            "approval_required": AgentOSService._approval_required(risk, row.status),
            "status": row.status,
            "action_type": row.action_type,
            "context": row.action_payload or {},
            "audit_link": f"/agents/{row.agent_id}",
            "created_at": AgentOSService._iso(row.created_at),
        }

    @staticmethod
    def normalize_exception(row: ExceptionItem) -> dict[str, Any]:
        squad = MODULE_TO_SQUAD.get(row.source_module or "", "risk")
        risk = AgentOSService._risk_from_severity(row.severity)
        return {
            "id": f"exception:{row.id}",
            "source_type": "exception",
            "source_id": str(row.id),
            "source_module": row.source_module,
            "business_object": {"type": row.source_type, "id": str(row.source_id) if row.source_id else None},
            "squad": squad,
            "agent_id": None,
            "title": row.title,
            "summary": row.description,
            "recommendation": row.recommended_action,
            "risk_level": risk,
            "approval_required": False,
            "status": row.status,
            "action_type": None,
            "context": {},
            "audit_link": f"/exceptions",
            "created_at": AgentOSService._iso(row.created_at),
        }

    @staticmethod
    def normalize_notification(row: Notification) -> dict[str, Any]:
        risk = AgentOSService._risk_from_severity(row.severity)
        return {
            "id": f"notification:{row.id}",
            "source_type": "notification",
            "source_id": str(row.id),
            "source_module": row.alert_type,
            "business_object": {"type": row.alert_type, "id": row.source_id},
            "squad": ALERT_TO_SQUAD.get(row.alert_type, "risk"),
            "agent_id": None,
            "title": row.title,
            "summary": row.content,
            "recommendation": "查看并处理该预警",
            "risk_level": risk,
            "approval_required": False,
            "status": "read" if row.is_read else "unread",
            "action_type": None,
            "context": {"link_url": row.link_url},
            "audit_link": row.link_url or "/notifications",
            "created_at": AgentOSService._iso(row.created_at),
        }
```

- [ ] **Step 6: Run service tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_service.py -q
```

Expected: all tests pass.

- [ ] **Step 7: Commit Task 1**

```bash
git add backend/app/agentos backend/tests/test_agentos_service.py
git commit -m "feat(agentos): add work item normalizers"
```

## Task 2: Backend AgentOS Aggregation APIs

**Files:**

- Modify: `backend/app/agentos/service.py`
- Create: `backend/app/agentos/router.py`
- Test: `backend/tests/test_agentos_router.py`

- [ ] **Step 1: Write failing API tests**

Create `backend/tests/test_agentos_router.py`:

```python
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
    records = body["data"]["records"]
    assert {item["source_type"] for item in records} >= {
        "agent_action",
        "exception",
        "notification",
    }
    assert body["data"]["total"] >= 3


@pytest.mark.asyncio
async def test_agentos_control_center_shape(async_client):
    res = await async_client.get("/api/agentos/control-center")
    assert res.status_code == 200
    data = res.json()["data"]
    assert set(data.keys()) == {"summary", "work_items", "squads", "templates"}
    assert {"id", "name", "agents"} <= set(data["squads"][0].keys())
    assert {"id", "title", "route"} <= set(data["templates"][0].keys())
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_router.py -q
```

Expected: fail with 404 for `/api/agentos/work-items` and `/api/agentos/control-center`.

- [ ] **Step 3: Add service aggregation methods**

Append these methods inside `AgentOSService` in `backend/app/agentos/service.py`:

```python
    @staticmethod
    def _sort_key(item: dict[str, Any]) -> datetime:
        value = item.get("created_at")
        if isinstance(value, datetime):
            return value
        return datetime.min.replace(tzinfo=timezone.utc)

    @staticmethod
    async def list_work_items(
        db: AsyncSession,
        user_id: int,
        source_type: str | None = None,
        squad: str | None = None,
        status: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict[str, Any]], int]:
        items: list[dict[str, Any]] = []

        if source_type in (None, "agent_action"):
            action_stmt = select(PendingAgentAction).where(PendingAgentAction.user_id == user_id)
            if status:
                action_stmt = action_stmt.where(PendingAgentAction.status == status)
            action_result = await db.execute(action_stmt.order_by(PendingAgentAction.created_at.desc()).limit(100))
            items.extend(AgentOSService.normalize_agent_pending_action(row) for row in action_result.scalars().all())

        if source_type in (None, "exception"):
            exception_stmt = select(ExceptionItem)
            if status:
                exception_stmt = exception_stmt.where(ExceptionItem.status == status)
            exception_result = await db.execute(exception_stmt.order_by(ExceptionItem.created_at.desc()).limit(100))
            items.extend(AgentOSService.normalize_exception(row) for row in exception_result.scalars().all())

        if source_type in (None, "notification"):
            notification_stmt = select(Notification).where(Notification.user_id == user_id)
            if status == "unread":
                notification_stmt = notification_stmt.where(Notification.is_read == 0)
            notification_result = await db.execute(notification_stmt.order_by(Notification.created_at.desc()).limit(100))
            items.extend(AgentOSService.normalize_notification(row) for row in notification_result.scalars().all())

        if squad:
            items = [item for item in items if item["squad"] == squad]

        items.sort(key=AgentOSService._sort_key, reverse=True)
        total = len(items)
        start = (page - 1) * page_size
        return items[start:start + page_size], total

    @staticmethod
    async def get_squads(db: AsyncSession, user_id: int) -> list[dict[str, Any]]:
        seven_days_ago = datetime.now(timezone.utc) - timedelta(days=7)
        squads = [dict(squad) for squad in AGENT_SQUADS]

        for squad in squads:
            agents = squad["agents"]
            decisions = await db.scalar(
                select(func.count()).select_from(AgentDecision).where(
                    AgentDecision.user_id == user_id,
                    AgentDecision.agent_id.in_(agents),
                    AgentDecision.created_at >= seven_days_ago,
                )
            ) or 0
            accepted = await db.scalar(
                select(func.count()).select_from(AgentDecision).where(
                    AgentDecision.user_id == user_id,
                    AgentDecision.agent_id.in_(agents),
                    AgentDecision.user_action == "accepted",
                    AgentDecision.created_at >= seven_days_ago,
                )
            ) or 0
            pending = await db.scalar(
                select(func.count()).select_from(PendingAgentAction).where(
                    PendingAgentAction.user_id == user_id,
                    PendingAgentAction.agent_id.in_(agents),
                    PendingAgentAction.status == "pending",
                )
            ) or 0
            squad["decision_count_7d"] = int(decisions)
            squad["pending_approvals"] = int(pending)
            squad["risk_count"] = int(pending)
            squad["adoption_rate"] = round(accepted / decisions, 3) if decisions else 0
            squad["autonomy_level"] = "semi_autonomous" if pending else "suggestion"

        return squads

    @staticmethod
    async def get_summary(db: AsyncSession, user_id: int) -> dict[str, Any]:
        work_items, total = await AgentOSService.list_work_items(db, user_id, page=1, page_size=200)
        pending_approvals = sum(1 for item in work_items if item["approval_required"])
        inventory_risks = sum(1 for item in work_items if item["squad"] == "fulfillment")
        executed = sum(1 for item in work_items if item["status"] in {"executed", "read", "resolved"})
        return {
            "sales_today": 0,
            "profit_today": 0,
            "inventory_risks": inventory_risks,
            "pending_approvals": pending_approvals,
            "active_work_items": total,
            "agent_automation_rate": round(executed / total, 3) if total else 0,
        }

    @staticmethod
    async def get_control_center(db: AsyncSession, user_id: int) -> dict[str, Any]:
        work_items, _ = await AgentOSService.list_work_items(db, user_id, page=1, page_size=8)
        return {
            "summary": await AgentOSService.get_summary(db, user_id),
            "work_items": work_items,
            "squads": await AgentOSService.get_squads(db, user_id),
            "templates": TEMPLATE_CARDS,
        }
```

- [ ] **Step 4: Create router endpoints**

Create `backend/app/agentos/router.py`:

```python
from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import PageResult, Result
from app.database import get_db
from app.models import User
from app.agentos.schemas import (
    ControlCenterSummaryVO,
    SquadVO,
    TemplateCardVO,
    WorkItemVO,
)
from app.agentos.service import AgentOSService, TEMPLATE_CARDS

router = APIRouter(tags=["AgentOS 工作台"])


@router.get("/agentos/work-items", summary="AgentOS 统一工作项")
async def list_work_items(
    source_type: str | None = Query(None),
    squad: str | None = Query(None),
    status: str | None = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    items, total = await AgentOSService.list_work_items(
        db,
        current_user.id,
        source_type=source_type,
        squad=squad,
        status=status,
        page=page,
        page_size=page_size,
    )
    return PageResult.ok(
        records=[WorkItemVO.model_validate(item) for item in items],
        total=total,
        page=page,
        page_size=page_size,
    )


@router.get("/agentos/control-center", summary="AgentOS 总控台")
async def get_control_center(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    return Result.ok(await AgentOSService.get_control_center(db, current_user.id))


@router.get("/agentos/squads", summary="AgentOS Agent 小队")
async def list_squads(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("agent:view")),
):
    squads = await AgentOSService.get_squads(db, current_user.id)
    return Result.ok([SquadVO.model_validate(squad) for squad in squads])


@router.get("/agentos/templates", summary="AgentOS 内置模板")
async def list_templates(
    current_user: User = Depends(require_permission("agent:view")),
):
    return Result.ok([TemplateCardVO.model_validate(template) for template in TEMPLATE_CARDS])
```

- [ ] **Step 5: Run router tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_router.py -q
```

Expected: all tests pass.

- [ ] **Step 6: Run Agent-related backend tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_service.py tests/test_agentos_router.py tests/test_agent_phase6.py -q
```

Expected: all selected tests pass.

- [ ] **Step 7: Commit Task 2**

```bash
git add backend/app/agentos backend/tests/test_agentos_router.py
git commit -m "feat(agentos): add control center APIs"
```

## Task 3: Frontend AgentOS API Client And Types

**Files:**

- Create: `frontend/src/api/modules/agentos.ts`

- [ ] **Step 1: Create typed API module**

Create `frontend/src/api/modules/agentos.ts`:

```typescript
import http from '@/api/http'

export interface AgentOSBusinessObject {
  type?: string | null
  id?: string | null
  label?: string | null
}

export interface AgentOSWorkItem {
  id: string
  source_type: string
  source_id: string
  source_module?: string | null
  business_object: AgentOSBusinessObject
  squad: string
  agent_id?: string | null
  title: string
  summary?: string | null
  recommendation?: string | null
  risk_level: string
  approval_required: boolean
  status: string
  action_type?: string | null
  context: Record<string, any>
  audit_link?: string | null
  created_at?: string | null
}

export interface AgentOSSquad {
  id: string
  name: string
  description: string
  agents: string[]
  decision_count_7d: number
  pending_approvals: number
  risk_count: number
  adoption_rate: number
  autonomy_level: string
}

export interface AgentOSTemplate {
  id: string
  title: string
  squad: string
  description: string
  mode: string
  route: string
  phase: string
}

export interface AgentOSSummary {
  sales_today: number
  profit_today: number
  inventory_risks: number
  pending_approvals: number
  active_work_items: number
  agent_automation_rate: number
}

export interface AgentOSControlCenter {
  summary: AgentOSSummary
  work_items: AgentOSWorkItem[]
  squads: AgentOSSquad[]
  templates: AgentOSTemplate[]
}

export const agentosApi = {
  getControlCenter() {
    return http.get('/agentos/control-center')
  },
  getWorkItems(params?: {
    source_type?: string
    squad?: string
    status?: string
    page?: number
    page_size?: number
  }) {
    return http.get('/agentos/work-items', { params })
  },
  getSquads() {
    return http.get('/agentos/squads')
  },
  getTemplates() {
    return http.get('/agentos/templates')
  },
}
```

- [ ] **Step 2: Run TypeScript build to catch client syntax errors**

Run:

```bash
cd frontend && npm run build
```

Expected: build either passes or fails only on unrelated pre-existing TypeScript errors. If it fails on `agentos.ts`, fix the exact error before continuing.

- [ ] **Step 3: Commit Task 3**

```bash
git add frontend/src/api/modules/agentos.ts
git commit -m "feat(agentos): add frontend API client"
```

## Task 4: Frontend Routes And Shared AgentOS Cards

**Files:**

- Create: `frontend/src/router/modules/agentos.ts`
- Create: `frontend/src/views/agentos/components/WorkItemCard.vue`
- Create: `frontend/src/views/agentos/components/SquadCard.vue`
- Create: `frontend/src/views/agentos/components/TemplateCard.vue`
- Modify: `frontend/src/components/Layout.vue` if route icon names are missing

- [ ] **Step 1: Create AgentOS routes**

Create `frontend/src/router/modules/agentos.ts`:

```typescript
import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'agentos',
    name: 'AgentOSControlCenter',
    component: () => import('@/views/agentos/ControlCenter.vue'),
    meta: { title: 'AgentOS 总控台', icon: 'analytics', menu: true, perm: 'agent:view' },
  },
  {
    path: 'agentos/tasks',
    name: 'AgentOSTaskCenter',
    component: () => import('@/views/agentos/TaskCenter.vue'),
    meta: { title: '任务中心', icon: 'checkmark-circle', menu: true, perm: 'agent:view' },
  },
  {
    path: 'agentos/squads',
    name: 'AgentOSSquads',
    component: () => import('@/views/agentos/AgentSquads.vue'),
    meta: { title: 'Agent 小队', icon: 'cube', menu: true, perm: 'agent:view' },
  },
]
```

- [ ] **Step 2: Create WorkItemCard**

Create `frontend/src/views/agentos/components/WorkItemCard.vue`:

```vue
<template>
  <n-card size="small" class="work-item-card">
    <template #header>
      <n-space justify="space-between" align="center">
        <n-space align="center">
          <n-tag :type="riskType" size="small">{{ riskLabel }}</n-tag>
          <strong>{{ item.title }}</strong>
        </n-space>
        <n-tag size="small" :type="statusType">{{ statusLabel }}</n-tag>
      </n-space>
    </template>

    <n-space vertical size="small">
      <div class="muted">{{ item.summary || item.recommendation || '暂无说明' }}</div>
      <n-space size="small">
        <n-tag size="small" :bordered="false">{{ squadLabel }}</n-tag>
        <n-tag v-if="item.agent_id" size="small" :bordered="false">{{ item.agent_id }}</n-tag>
        <n-tag v-if="item.approval_required" size="small" type="warning">需审批</n-tag>
      </n-space>
      <n-space>
        <n-button size="tiny" type="primary" ghost @click="$emit('inspect', item)">查看</n-button>
        <n-button v-if="item.approval_required" size="tiny" type="success" ghost @click="$emit('approve', item)">批准</n-button>
        <n-button v-if="item.approval_required" size="tiny" type="warning" ghost @click="$emit('reject', item)">拒绝</n-button>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AgentOSWorkItem } from '@/api/modules/agentos'

const props = defineProps<{ item: AgentOSWorkItem }>()
defineEmits<{
  inspect: [item: AgentOSWorkItem]
  approve: [item: AgentOSWorkItem]
  reject: [item: AgentOSWorkItem]
}>()

const riskType = computed(() => {
  const map: Record<string, string> = { critical: 'error', high: 'error', medium: 'warning', low: 'info' }
  return map[props.item.risk_level] || 'default'
})
const riskLabel = computed(() => {
  const map: Record<string, string> = { critical: '严重', high: '高风险', medium: '中风险', low: '低风险' }
  return map[props.item.risk_level] || props.item.risk_level
})
const statusType = computed(() => {
  const map: Record<string, string> = { pending: 'warning', unread: 'warning', executed: 'success', resolved: 'success', rejected: 'default' }
  return map[props.item.status] || 'default'
})
const statusLabel = computed(() => {
  const map: Record<string, string> = { pending: '待处理', unread: '未读', read: '已读', executed: '已执行', resolved: '已解决', rejected: '已拒绝' }
  return map[props.item.status] || props.item.status
})
const squadLabel = computed(() => {
  const map: Record<string, string> = { growth: '增长小队', fulfillment: '履约小队', risk: '风控小队' }
  return map[props.item.squad] || props.item.squad
})
</script>

<style scoped>
.work-item-card { margin-bottom: 10px; }
.muted { color: #666; font-size: 13px; line-height: 1.5; }
</style>
```

- [ ] **Step 3: Create SquadCard**

Create `frontend/src/views/agentos/components/SquadCard.vue`:

```vue
<template>
  <n-card size="small">
    <template #header>
      <n-space justify="space-between" align="center">
        <strong>{{ squad.name }}</strong>
        <n-tag :type="autonomyType" size="small">{{ autonomyLabel }}</n-tag>
      </n-space>
    </template>
    <n-space vertical size="small">
      <div class="muted">{{ squad.description }}</div>
      <n-space>
        <n-tag v-for="agent in squad.agents" :key="agent" size="small">{{ agent }}</n-tag>
      </n-space>
      <n-grid :cols="3" :x-gap="8">
        <n-grid-item><div class="metric">{{ squad.decision_count_7d }}</div><div class="label">7天决策</div></n-grid-item>
        <n-grid-item><div class="metric">{{ Math.round(squad.adoption_rate * 100) }}%</div><div class="label">采纳率</div></n-grid-item>
        <n-grid-item><div class="metric">{{ squad.pending_approvals }}</div><div class="label">待审批</div></n-grid-item>
      </n-grid>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AgentOSSquad } from '@/api/modules/agentos'

const props = defineProps<{ squad: AgentOSSquad }>()

const autonomyType = computed(() => props.squad.autonomy_level === 'semi_autonomous' ? 'success' : 'info')
const autonomyLabel = computed(() => {
  const map: Record<string, string> = {
    observation: '观察',
    suggestion: '建议',
    semi_autonomous: '半自主',
    full_autonomous: '全自主',
  }
  return map[props.squad.autonomy_level] || props.squad.autonomy_level
})
</script>

<style scoped>
.muted { color: #666; font-size: 13px; }
.metric { font-weight: 700; font-size: 18px; }
.label { color: #999; font-size: 12px; }
</style>
```

- [ ] **Step 4: Create TemplateCard**

Create `frontend/src/views/agentos/components/TemplateCard.vue`:

```vue
<template>
  <n-card size="small" hoverable @click="$emit('open', template)">
    <template #header>
      <n-space justify="space-between">
        <strong>{{ template.title }}</strong>
        <n-tag size="small" :type="modeType">{{ template.mode }}</n-tag>
      </n-space>
    </template>
    <p class="desc">{{ template.description }}</p>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AgentOSTemplate } from '@/api/modules/agentos'

const props = defineProps<{ template: AgentOSTemplate }>()
defineEmits<{ open: [template: AgentOSTemplate] }>()

const modeType = computed(() => props.template.mode === 'Agent' ? 'success' : 'info')
</script>

<style scoped>
.desc { color: #666; font-size: 13px; margin: 0; line-height: 1.5; }
</style>
```

- [ ] **Step 5: Verify route icons**

Open `frontend/src/components/Layout.vue`. If `analytics`, `checkmark-circle`, and `cube` already exist in `iconMap`, do not change the file. If one is missing, add it to `iconMap` using an existing imported Ionicon.

- [ ] **Step 6: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build passes, or any failure is unrelated to the new AgentOS route/components. Fix AgentOS-related errors before continuing.

- [ ] **Step 7: Commit Task 4**

```bash
git add frontend/src/router/modules/agentos.ts frontend/src/views/agentos/components frontend/src/components/Layout.vue
git commit -m "feat(agentos): add routes and shared cards"
```

## Task 5: AgentOS Control Center Page

**Files:**

- Create: `frontend/src/views/agentos/ControlCenter.vue`

- [ ] **Step 1: Create control center page**

Create `frontend/src/views/agentos/ControlCenter.vue`:

```vue
<template>
  <div>
    <n-page-header subtitle="AI 原生跨境电商经营工作台">
      <template #title>AgentOS 总控台</template>
      <template #extra>
        <n-space>
          <n-button size="small" secondary @click="fetchData">刷新</n-button>
          <n-button size="small" type="error" ghost>暂停全部 Agent</n-button>
        </n-space>
      </template>
    </n-page-header>

    <n-grid :cols="4" :x-gap="12" :y-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card size="small"><div class="metric">¥{{ fmt(summary.sales_today) }}</div><div class="label">今日销售</div></n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card size="small"><div class="metric">¥{{ fmt(summary.profit_today) }}</div><div class="label">今日利润</div></n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card size="small"><div class="metric">{{ summary.pending_approvals }}</div><div class="label">待审批动作</div></n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card size="small"><div class="metric">{{ Math.round(summary.agent_automation_rate * 100) }}%</div><div class="label">自动化率</div></n-card>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="24" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item :span="5">
        <n-card title="任务筛选" size="small">
          <n-space vertical>
            <n-button block :type="selectedSquad === '' ? 'primary' : 'default'" @click="selectedSquad = ''">全部任务</n-button>
            <n-button block :type="selectedSquad === 'growth' ? 'primary' : 'default'" @click="selectedSquad = 'growth'">增长小队</n-button>
            <n-button block :type="selectedSquad === 'fulfillment' ? 'primary' : 'default'" @click="selectedSquad = 'fulfillment'">履约小队</n-button>
            <n-button block :type="selectedSquad === 'risk' ? 'primary' : 'default'" @click="selectedSquad = 'risk'">风控小队</n-button>
          </n-space>
        </n-card>
      </n-grid-item>

      <n-grid-item :span="12">
        <n-card title="AI 任务工作台" size="small" :loading="loading">
          <n-empty v-if="filteredItems.length === 0" description="暂无待处理任务" />
          <work-item-card
            v-for="item in filteredItems"
            :key="item.id"
            :item="item"
            @inspect="inspectItem"
            @approve="inspectItem"
            @reject="inspectItem"
          />
        </n-card>
      </n-grid-item>

      <n-grid-item :span="7">
        <n-space vertical>
          <squad-card v-for="squad in squads" :key="squad.id" :squad="squad" />
        </n-space>
      </n-grid-item>
    </n-grid>

    <n-card title="内置电商模板" size="small" style="margin-top: 12px;">
      <n-grid :cols="3" :x-gap="12" :y-gap="12">
        <n-grid-item v-for="template in templates" :key="template.id">
          <template-card :template="template" @open="openTemplate" />
        </n-grid-item>
      </n-grid>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { agentosApi } from '@/api/modules/agentos'
import type { AgentOSSummary, AgentOSSquad, AgentOSTemplate, AgentOSWorkItem } from '@/api/modules/agentos'
import WorkItemCard from './components/WorkItemCard.vue'
import SquadCard from './components/SquadCard.vue'
import TemplateCard from './components/TemplateCard.vue'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const selectedSquad = ref('')
const summary = reactive<AgentOSSummary>({
  sales_today: 0,
  profit_today: 0,
  inventory_risks: 0,
  pending_approvals: 0,
  active_work_items: 0,
  agent_automation_rate: 0,
})
const workItems = ref<AgentOSWorkItem[]>([])
const squads = ref<AgentOSSquad[]>([])
const templates = ref<AgentOSTemplate[]>([])

const filteredItems = computed(() => {
  if (!selectedSquad.value) return workItems.value
  return workItems.value.filter(item => item.squad === selectedSquad.value)
})

function fmt(value: number) {
  return (value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 2 })
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await agentosApi.getControlCenter()
    const data = res?.data || {}
    Object.assign(summary, data.summary || {})
    workItems.value = data.work_items || []
    squads.value = data.squads || []
    templates.value = data.templates || []
  } catch (error: any) {
    message.error(error?.response?.data?.message || '加载 AgentOS 总控台失败')
  } finally {
    loading.value = false
  }
}

function inspectItem(item: AgentOSWorkItem) {
  if (item.audit_link) router.push(item.audit_link)
}

function openTemplate(template: AgentOSTemplate) {
  router.push(template.route)
}

onMounted(fetchData)
</script>

<style scoped>
.metric { font-size: 22px; font-weight: 700; }
.label { color: #888; font-size: 12px; margin-top: 4px; }
</style>
```

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build passes. If TypeScript reports `res.data` shape errors, keep `res: any` in this page because `http` response wrapping is inconsistent across existing modules.

- [ ] **Step 3: Optional browser smoke test**

Run the frontend dev server:

```bash
cd frontend && npm run dev -- --host 127.0.0.1 --port 3001
```

Open `/agentos` and verify:

- Four summary cards render.
- Task column shows empty state or work items.
- Squad cards render.
- Template cards route to existing pages.

- [ ] **Step 4: Commit Task 5**

```bash
git add frontend/src/views/agentos/ControlCenter.vue
git commit -m "feat(agentos): add control center page"
```

## Task 6: Task Center Page With Filters

**Files:**

- Create: `frontend/src/views/agentos/TaskCenter.vue`

- [ ] **Step 1: Create task center page**

Create `frontend/src/views/agentos/TaskCenter.vue`:

```vue
<template>
  <div>
    <n-page-header subtitle="统一处理 Agent 建议、异常、通知和待审批动作">
      <template #title>任务中心</template>
    </n-page-header>

    <n-card size="small" style="margin-top: 12px;">
      <n-space>
        <span class="filter-label">来源</span>
        <n-select v-model:value="query.source_type" clearable style="width: 160px" :options="sourceOptions" />
        <span class="filter-label">小队</span>
        <n-select v-model:value="query.squad" clearable style="width: 160px" :options="squadOptions" />
        <span class="filter-label">状态</span>
        <n-select v-model:value="query.status" clearable style="width: 160px" :options="statusOptions" />
        <n-button type="primary" @click="fetchItems">筛选</n-button>
      </n-space>
    </n-card>

    <n-card size="small" style="margin-top: 12px;" :loading="loading">
      <n-empty v-if="items.length === 0" description="暂无任务" />
      <work-item-card
        v-for="item in items"
        :key="item.id"
        :item="item"
        @inspect="inspectItem"
        @approve="inspectItem"
        @reject="inspectItem"
      />
      <n-pagination
        v-if="total > query.page_size"
        v-model:page="query.page"
        :page-size="query.page_size"
        :item-count="total"
        @update:page="fetchItems"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { agentosApi } from '@/api/modules/agentos'
import type { AgentOSWorkItem } from '@/api/modules/agentos'
import WorkItemCard from './components/WorkItemCard.vue'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const items = ref<AgentOSWorkItem[]>([])
const total = ref(0)
const query = reactive({
  source_type: null as string | null,
  squad: null as string | null,
  status: null as string | null,
  page: 1,
  page_size: 20,
})

const sourceOptions = [
  { label: 'Agent 动作', value: 'agent_action' },
  { label: '异常', value: 'exception' },
  { label: '通知', value: 'notification' },
]
const squadOptions = [
  { label: '增长小队', value: 'growth' },
  { label: '履约小队', value: 'fulfillment' },
  { label: '风控小队', value: 'risk' },
]
const statusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '未读', value: 'unread' },
  { label: '已执行', value: 'executed' },
  { label: '已解决', value: 'resolved' },
]

async function fetchItems() {
  loading.value = true
  try {
    const res: any = await agentosApi.getWorkItems({
      source_type: query.source_type || undefined,
      squad: query.squad || undefined,
      status: query.status || undefined,
      page: query.page,
      page_size: query.page_size,
    })
    items.value = res?.records || res?.data?.records || []
    total.value = res?.total || res?.data?.total || 0
  } catch (error: any) {
    message.error(error?.response?.data?.message || '加载任务失败')
  } finally {
    loading.value = false
  }
}

function inspectItem(item: AgentOSWorkItem) {
  if (item.audit_link) router.push(item.audit_link)
}

onMounted(fetchItems)
</script>

<style scoped>
.filter-label { color: #666; font-size: 13px; line-height: 32px; }
</style>
```

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 3: Commit Task 6**

```bash
git add frontend/src/views/agentos/TaskCenter.vue
git commit -m "feat(agentos): add task center"
```

## Task 7: Agent Squads Page

**Files:**

- Create: `frontend/src/views/agentos/AgentSquads.vue`

- [ ] **Step 1: Create Agent squads page**

Create `frontend/src/views/agentos/AgentSquads.vue`:

```vue
<template>
  <div>
    <n-page-header subtitle="按增长、履约、风控管理 AI 运营团队">
      <template #title>Agent 小队</template>
      <template #extra><n-button size="small" @click="fetchSquads">刷新</n-button></template>
    </n-page-header>

    <n-grid :cols="3" :x-gap="12" :y-gap="12" style="margin-top: 12px;">
      <n-grid-item v-for="squad in squads" :key="squad.id">
        <squad-card :squad="squad" />
        <n-card size="small" style="margin-top: 8px;">
          <n-space>
            <n-button size="small" ghost @click="router.push('/agents')">查看 Agent</n-button>
            <n-button size="small" ghost @click="router.push('/agents/rules')">规则</n-button>
            <n-button size="small" ghost @click="router.push('/agents/entropy')">熵管理</n-button>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-card title="自治等级说明" size="small" style="margin-top: 12px;">
      <n-grid :cols="4" :x-gap="12">
        <n-grid-item><n-alert type="default" title="观察">只读数据，生成报告。</n-alert></n-grid-item>
        <n-grid-item><n-alert type="info" title="建议">生成建议，人执行。</n-alert></n-grid-item>
        <n-grid-item><n-alert type="success" title="半自主">低风险自动，高风险审批。</n-alert></n-grid-item>
        <n-grid-item><n-alert type="warning" title="全自主">仅限高信任低风险链路。</n-alert></n-grid-item>
      </n-grid>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { agentosApi } from '@/api/modules/agentos'
import type { AgentOSSquad } from '@/api/modules/agentos'
import SquadCard from './components/SquadCard.vue'

const router = useRouter()
const message = useMessage()
const squads = ref<AgentOSSquad[]>([])

async function fetchSquads() {
  try {
    const res: any = await agentosApi.getSquads()
    squads.value = res?.data || []
  } catch (error: any) {
    message.error(error?.response?.data?.message || '加载 Agent 小队失败')
  }
}

onMounted(fetchSquads)
</script>
```

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 3: Commit Task 7**

```bash
git add frontend/src/views/agentos/AgentSquads.vue
git commit -m "feat(agentos): add squad overview"
```

## Task 8: Backend And Frontend Full Verification

**Files:**

- No source files unless verification finds AgentOS-specific defects.

- [ ] **Step 1: Run focused backend tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_service.py tests/test_agentos_router.py -q
```

Expected: all tests pass.

- [ ] **Step 2: Run broader Agent backend tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agent_phase1.py tests/test_agent_phase2.py tests/test_agent_phase3.py tests/test_agent_phase4.py tests/test_agent_phase5.py tests/test_agent_phase6.py -q
```

Expected: all selected Agent tests pass. If failures are caused by pre-existing dirty worktree changes, record the failing test names and rerun only AgentOS tests before final handoff.

- [ ] **Step 3: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 4: Run local backend and frontend for manual smoke**

Start backend:

```bash
cd backend && .venv/bin/uvicorn app.main:app --reload --host 127.0.0.1 --port 8001
```

Start frontend in a second terminal:

```bash
cd frontend && npm run dev -- --host 127.0.0.1 --port 3001
```

Open:

```text
http://127.0.0.1:3001/agentos
http://127.0.0.1:3001/agentos/tasks
http://127.0.0.1:3001/agentos/squads
```

Verify:

- AgentOS routes are visible in the menu for users with `agent:view`.
- `/agentos` renders summary cards, tasks, squads, and templates.
- `/agentos/tasks` filters by source, squad, and status.
- `/agentos/squads` shows three squads and links back to existing Agent pages.
- Clicking a work item with `audit_link` routes to an existing detail page.

- [ ] **Step 5: Final status check**

Run:

```bash
git status --short
git log --oneline -5
```

Expected:

- Only intended AgentOS files are changed or committed by this implementation.
- Pre-existing unrelated dirty files remain untouched.

- [ ] **Step 6: Final commit if verification fixes were made**

If Task 8 required fixes, commit them:

```bash
git add backend/app/agentos backend/tests/test_agentos_service.py backend/tests/test_agentos_router.py frontend/src/api/modules/agentos.ts frontend/src/router/modules/agentos.ts frontend/src/views/agentos frontend/src/components/Layout.vue
git commit -m "fix(agentos): polish phase 1 verification"
```

If no fixes were made, do not create an empty commit.

## Self-Review Checklist

- Spec coverage:
  - AgentOS Control Center is covered by Tasks 2, 3, and 5.
  - Unified WorkItem model is covered by Tasks 1, 2, and 6.
  - Agent Squads are covered by Tasks 1, 2, 4, and 7.
  - Templates and Skills entry points are covered by Tasks 1, 2, and 5.
  - Autonomy and safety visibility are covered by Tasks 2, 5, and 7.
  - Audit links are included in WorkItem normalization and UI drill-down.
  - Phase 2/3 items are explicitly excluded from this implementation.

- Red-flag scan:
  - The plan contains no open implementation markers or incomplete steps.
  - No task requires a generic low-code canvas, desktop client, IM workflow, or Skill marketplace.

- Type consistency:
  - Backend schema names use `WorkItemVO`, `SquadVO`, `TemplateCardVO`, and `ControlCenterSummaryVO`.
  - Frontend interfaces use `AgentOSWorkItem`, `AgentOSSquad`, `AgentOSTemplate`, and `AgentOSSummary`.
  - API paths are `/agentos/control-center`, `/agentos/work-items`, `/agentos/squads`, and `/agentos/templates`.
