# Platform Fee Rules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a local platform fee rules engine so pre-listing decisions can automatically calculate platform commission, payment fees, fixed fees, advertising reserve, and other reserve fees by platform, destination country, and category.

**Architecture:** Add a focused backend module `backend/app/platform_fee/` with schemas, service, and router, plus a `PlatformFeeRule` SQLAlchemy model and Alembic migration. Integrate this module into `PreListingDecisionService` while preserving current manual fee fallback behavior when no platform rule matches.

**Tech Stack:** Python 3.11+, FastAPI, Pydantic v2, SQLAlchemy 2.0 async, Alembic, PostgreSQL, pytest, Vue 3, TypeScript, Vite, Naive UI.

---

## Starting Point

Current baseline:

- Branch `main` is clean and pushed.
- Backend full suite currently passes: `217 passed`.
- Frontend build passes.
- Existing single-SKU decision endpoint: `POST /api/decisions/prelisting`.
- Existing request currently uses manual `platform_fee_pct`, `payment_fee_pct`, and `other_fee`.
- Existing planning doc: `docs/platform-fee-rules-plan.md`.

Create an isolated branch before implementing:

```bash
git switch main
git pull
git switch -c codex/platform-fee-rules
```

## Scope

In scope:

- `platform_fee_rule` database table.
- Backend CRUD and match APIs.
- Permissions: `platform_fee:view`, `platform_fee:manage`, `platform_fee:calculate`.
- Audit logs for platform fee rule create/update/delete.
- Pre-listing decision integration with automatic rule matching.
- Minimal frontend integration in the existing decision page.
- Docs and tests.

Out of scope for this task:

- Real platform API fee sync.
- Full platform fee management UI.
- Excel import/export for platform fee rules.
- VAT/tax reconciliation.
- Cross-platform category mapping table.
- Batch pre-listing decision.

## File Structure

Create:

- `backend/app/platform_fee/__init__.py` - exports router for automatic registration.
- `backend/app/platform_fee/router.py` - FastAPI endpoints and permission/audit wiring.
- `backend/app/platform_fee/schemas.py` - Pydantic request/response models.
- `backend/app/platform_fee/service.py` - CRUD, soft delete, and rule matching.
- `backend/alembic/versions/20260615_03_add_platform_fee_rules.py` - table migration.
- `backend/tests/test_platform_fee_rules.py` - CRUD, matching, auth, and audit coverage.

Modify:

- `backend/app/models.py` - add `PlatformFeeRule`.
- `backend/app/decision/schemas.py` - add `platform_id`, `category_id`, applied-rule fields.
- `backend/app/decision/service.py` - use `PlatformFeeRuleService.match` when `platform_id` is provided.
- `backend/tests/test_prelisting_decision.py` - cover rule integration and fallback.
- `backend/seed.py` - seed new permission codes.
- `docs/PERMISSIONS_AND_AUDIT.md` - document permissions and audit coverage.
- `docs/PROJECT_STATUS.md` - document completed first version.
- `docs/ROADMAP.md` - mark platform fee rules first version done and recommend batch decision next.
- `frontend/src/api/modules/decision.ts` - add `platform_id`, `category_id`, and applied-rule response fields.
- `frontend/src/views/decision/PreListingDecision.vue` - add platform selector and rule-source display.

## Matching Semantics

`PlatformFeeRuleService.match` must choose exactly one active rule using this priority:

1. Same `platform_id`, same `site_code`, same `category_id`.
2. Same `platform_id`, same `site_code`, `category_id IS NULL`.
3. Same `platform_id`, `site_code IS NULL`, `category_id IS NULL`.

Tie-breaker:

- Smaller `priority` wins.
- If priority ties, newer `id` does not win; smaller `id` wins for deterministic behavior.

Do not match disabled rules where `status != 1`.

`site_code` is stored uppercase. Empty string input should be normalized to `None`.

## Task 1: Backend Model And Migration

**Files:**
- Modify: `backend/app/models.py`
- Create: `backend/alembic/versions/20260615_03_add_platform_fee_rules.py`
- Test: `backend/tests/test_platform_fee_rules.py`

- [ ] **Step 1: Write failing model/migration smoke test**

Create `backend/tests/test_platform_fee_rules.py` with this initial content:

```python
"""平台费用规则 API、匹配、权限与审计测试。"""

import pytest
from uuid import uuid4

from sqlalchemy import select

from app.database import async_session_factory
from app.models import OperationLog, PlatformFeeRule
from tests.auth_helpers import enable_auth, grant_permission, register_and_login


pytestmark = pytest.mark.usefixtures("enable_auth")


def _code(prefix: str) -> str:
    return f"{prefix}_{uuid4().hex[:6]}"


async def _auth(async_client, username_prefix: str, permission: str | None = None):
    uid, token = await register_and_login(async_client, username_prefix)
    if permission:
        await grant_permission(uid, permission)
    return {"Authorization": f"Bearer {token}"}


async def _create_platform(async_client, code_prefix: str = "pf"):
    headers = await _auth(async_client, f"{code_prefix}_platform", "platform:create")
    payload = {
        "name": f"平台-{uuid4().hex[:6]}",
        "code": _code(code_prefix),
        "api_base_url": "https://example.com",
        "status": 1,
    }
    resp = await async_client.post("/api/platforms", json=payload, headers=headers)
    assert resp.status_code == 200
    return resp.json()["data"]["id"]


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


class TestPlatformFeeModel:
    async def test_platform_fee_rule_model_is_mapped(self):
        async with async_session_factory() as session:
            rule = PlatformFeeRule(
                platform_id=1,
                site_code="RU",
                category_id=None,
                commission_pct=12,
                payment_fee_pct=3,
                fixed_fee=5,
                advertising_pct=2,
                other_reserve_fee=1,
                priority=0,
                status=1,
                remark="model smoke test",
            )
            session.add(rule)
            await session.flush()
            assert rule.id is not None
            await session.rollback()
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_platform_fee_rules.py::TestPlatformFeeModel::test_platform_fee_rule_model_is_mapped -q
```

Expected:

- FAIL during import with `ImportError` or `AttributeError` because `PlatformFeeRule` does not exist.

- [ ] **Step 3: Add SQLAlchemy model**

In `backend/app/models.py`, add this class after `Platform` and before `ProductListing`:

```python
class PlatformFeeRule(Base):
    """平台费用规则"""
    __tablename__ = "platform_fee_rule"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    site_code = Column(String(10), comment="站点/国家代码，空表示平台全局")
    category_id = Column(BigInteger, ForeignKey("category.id"), comment="本地类目ID，空表示站点/平台通用")
    commission_pct = Column(Numeric(8, 4), default=0, nullable=False, comment="平台佣金比例")
    payment_fee_pct = Column(Numeric(8, 4), default=0, nullable=False, comment="支付手续费比例")
    fixed_fee = Column(Numeric(10, 2), default=0, nullable=False, comment="固定交易费")
    advertising_pct = Column(Numeric(8, 4), default=0, nullable=False, comment="广告/营销预留比例")
    other_reserve_fee = Column(Numeric(10, 2), default=0, nullable=False, comment="其他固定预留费用")
    priority = Column(Integer, default=0, nullable=False, comment="优先级，值小优先")
    status = Column(SmallInteger, default=1, nullable=False, comment="状态: 0-禁用, 1-启用")
    remark = Column(Text, comment="备注")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    platform = relationship("Platform", lazy="selectin")
    category = relationship("Category", lazy="selectin")
```

- [ ] **Step 4: Add Alembic migration**

Create `backend/alembic/versions/20260615_03_add_platform_fee_rules.py`:

```python
"""add platform fee rules

Revision ID: 20260615_03
Revises: 20260615_02
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa


revision = "20260615_03"
down_revision = "20260615_02"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "platform_fee_rule",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("platform_id", sa.BigInteger(), sa.ForeignKey("platform.id"), nullable=False),
        sa.Column("site_code", sa.String(length=10), nullable=True),
        sa.Column("category_id", sa.BigInteger(), sa.ForeignKey("category.id"), nullable=True),
        sa.Column("commission_pct", sa.Numeric(8, 4), nullable=False, server_default="0"),
        sa.Column("payment_fee_pct", sa.Numeric(8, 4), nullable=False, server_default="0"),
        sa.Column("fixed_fee", sa.Numeric(10, 2), nullable=False, server_default="0"),
        sa.Column("advertising_pct", sa.Numeric(8, 4), nullable=False, server_default="0"),
        sa.Column("other_reserve_fee", sa.Numeric(10, 2), nullable=False, server_default="0"),
        sa.Column("priority", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("status", sa.SmallInteger(), nullable=False, server_default="1"),
        sa.Column("remark", sa.Text(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_platform_fee_rule_platform", "platform_fee_rule", ["platform_id"])
    op.create_index("ix_platform_fee_rule_match", "platform_fee_rule", ["platform_id", "site_code", "category_id", "status"])


def downgrade() -> None:
    op.drop_index("ix_platform_fee_rule_match", table_name="platform_fee_rule")
    op.drop_index("ix_platform_fee_rule_platform", table_name="platform_fee_rule")
    op.drop_table("platform_fee_rule")
```

- [ ] **Step 5: Run migration on local dev database**

Run:

```bash
cd backend
.venv/bin/alembic upgrade head
```

Expected:

- Migration applies successfully.

- [ ] **Step 6: Run model smoke test**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_platform_fee_rules.py::TestPlatformFeeModel::test_platform_fee_rule_model_is_mapped -q
```

Expected:

- PASS.

- [ ] **Step 7: Commit model and migration**

Run:

```bash
git add backend/app/models.py backend/alembic/versions/20260615_03_add_platform_fee_rules.py backend/tests/test_platform_fee_rules.py
git commit -m "feat: add platform fee rule model"
```

## Task 2: Schemas, Service, And Matching Rules

**Files:**
- Create: `backend/app/platform_fee/__init__.py`
- Create: `backend/app/platform_fee/schemas.py`
- Create: `backend/app/platform_fee/service.py`
- Modify: `backend/tests/test_platform_fee_rules.py`

- [ ] **Step 1: Add failing service tests**

Append to `backend/tests/test_platform_fee_rules.py`:

```python
class TestPlatformFeeRuleService:
    async def test_match_prefers_category_rule_over_site_and_global(self, async_client):
        from app.platform_fee.schemas import PlatformFeeRuleCreate, PlatformFeeRuleMatchRequest
        from app.platform_fee.service import PlatformFeeRuleService

        platform_id = await _create_platform(async_client, "pf_match")
        async with async_session_factory() as session:
            global_rule = await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id,
                site_code=None,
                category_id=None,
                commission_pct=5,
                payment_fee_pct=1,
                fixed_fee=1,
                advertising_pct=0,
                other_reserve_fee=0,
                priority=0,
                remark="global",
            ))
            site_rule = await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id,
                site_code="RU",
                category_id=None,
                commission_pct=10,
                payment_fee_pct=2,
                fixed_fee=2,
                advertising_pct=1,
                other_reserve_fee=0,
                priority=0,
                remark="site",
            ))
            category_rule = await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id,
                site_code="RU",
                category_id=12345,
                commission_pct=15,
                payment_fee_pct=3,
                fixed_fee=3,
                advertising_pct=2,
                other_reserve_fee=1,
                priority=0,
                remark="category",
            ))
            await session.commit()

            matched = await PlatformFeeRuleService.match(session, PlatformFeeRuleMatchRequest(
                platform_id=platform_id,
                site_code="ru",
                category_id=12345,
            ))

        assert global_rule["id"] != site_rule["id"]
        assert matched is not None
        assert matched["id"] == category_rule["id"]
        assert matched["site_code"] == "RU"
        assert matched["commission_pct"] == 15.0

    async def test_match_falls_back_to_site_rule_when_category_missing(self, async_client):
        from app.platform_fee.schemas import PlatformFeeRuleCreate, PlatformFeeRuleMatchRequest
        from app.platform_fee.service import PlatformFeeRuleService

        platform_id = await _create_platform(async_client, "pf_site")
        async with async_session_factory() as session:
            site_rule = await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id,
                site_code="BR",
                category_id=None,
                commission_pct=11,
                payment_fee_pct=2,
                fixed_fee=4,
                advertising_pct=1,
                other_reserve_fee=0,
                priority=0,
                remark="site fallback",
            ))
            await session.commit()

            matched = await PlatformFeeRuleService.match(session, PlatformFeeRuleMatchRequest(
                platform_id=platform_id,
                site_code="BR",
                category_id=99999,
            ))

        assert matched is not None
        assert matched["id"] == site_rule["id"]

    async def test_match_ignores_disabled_rules(self, async_client):
        from app.platform_fee.schemas import PlatformFeeRuleCreate, PlatformFeeRuleMatchRequest, PlatformFeeRuleUpdate
        from app.platform_fee.service import PlatformFeeRuleService

        platform_id = await _create_platform(async_client, "pf_disabled")
        async with async_session_factory() as session:
            rule = await PlatformFeeRuleService.create(session, PlatformFeeRuleCreate(
                platform_id=platform_id,
                site_code="MY",
                category_id=None,
                commission_pct=20,
                payment_fee_pct=2,
                fixed_fee=1,
                advertising_pct=0,
                other_reserve_fee=0,
                priority=0,
                remark="disabled",
            ))
            await PlatformFeeRuleService.update(session, rule["id"], PlatformFeeRuleUpdate(status=0))
            await session.commit()

            matched = await PlatformFeeRuleService.match(session, PlatformFeeRuleMatchRequest(
                platform_id=platform_id,
                site_code="MY",
                category_id=None,
            ))

        assert matched is None
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_platform_fee_rules.py::TestPlatformFeeRuleService -q
```

Expected:

- FAIL because `app.platform_fee` module does not exist.

- [ ] **Step 3: Create schemas**

Create `backend/app/platform_fee/schemas.py`:

```python
"""平台费用规则 - Pydantic Schema"""

from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field, model_validator


class PlatformFeeRuleCreate(BaseModel):
    platform_id: int = Field(..., description="平台ID")
    site_code: Optional[str] = Field(None, max_length=10, description="站点/国家代码；空表示平台全局")
    category_id: Optional[int] = Field(None, description="本地类目ID；空表示通用")
    commission_pct: float = Field(default=0, ge=0, le=100, description="平台佣金比例")
    payment_fee_pct: float = Field(default=0, ge=0, le=100, description="支付手续费比例")
    fixed_fee: float = Field(default=0, ge=0, description="固定交易费")
    advertising_pct: float = Field(default=0, ge=0, le=100, description="广告/营销预留比例")
    other_reserve_fee: float = Field(default=0, ge=0, description="其他固定预留费用")
    priority: int = Field(default=0, description="优先级，值小优先")
    remark: Optional[str] = Field(None, description="备注")

    @model_validator(mode="after")
    def normalize_site_code(self):
        if self.site_code is not None:
            normalized = self.site_code.strip().upper()
            self.site_code = normalized or None
        return self


class PlatformFeeRuleUpdate(BaseModel):
    site_code: Optional[str] = Field(None, max_length=10)
    category_id: Optional[int] = None
    commission_pct: Optional[float] = Field(None, ge=0, le=100)
    payment_fee_pct: Optional[float] = Field(None, ge=0, le=100)
    fixed_fee: Optional[float] = Field(None, ge=0)
    advertising_pct: Optional[float] = Field(None, ge=0, le=100)
    other_reserve_fee: Optional[float] = Field(None, ge=0)
    priority: Optional[int] = None
    status: Optional[int] = Field(None, ge=0, le=1)
    remark: Optional[str] = None

    @model_validator(mode="after")
    def normalize_site_code(self):
        if self.site_code is not None:
            normalized = self.site_code.strip().upper()
            self.site_code = normalized or None
        return self


class PlatformFeeRuleMatchRequest(BaseModel):
    platform_id: int
    site_code: Optional[str] = Field(None, max_length=10)
    category_id: Optional[int] = None

    @model_validator(mode="after")
    def normalize_site_code(self):
        if self.site_code is not None:
            normalized = self.site_code.strip().upper()
            self.site_code = normalized or None
        return self


class PlatformFeeRuleVO(BaseModel):
    id: int
    platform_id: int
    platform_name: Optional[str] = None
    site_code: Optional[str] = None
    category_id: Optional[int] = None
    category_name: Optional[str] = None
    commission_pct: float
    payment_fee_pct: float
    fixed_fee: float
    advertising_pct: float
    other_reserve_fee: float
    priority: int
    status: int
    remark: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
```

- [ ] **Step 4: Create service**

Create `backend/app/platform_fee/service.py`:

```python
"""平台费用规则 - 服务层"""

from decimal import Decimal
from typing import Optional

from sqlalchemy import and_, or_, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models import PlatformFeeRule
from app.platform_fee.schemas import (
    PlatformFeeRuleCreate,
    PlatformFeeRuleMatchRequest,
    PlatformFeeRuleUpdate,
)


def _money(value) -> float:
    return float(value or 0)


def _to_dict(rule: PlatformFeeRule) -> dict:
    return {
        "id": rule.id,
        "platform_id": rule.platform_id,
        "platform_name": rule.platform.name if rule.platform else None,
        "site_code": rule.site_code,
        "category_id": rule.category_id,
        "category_name": rule.category.name if rule.category else None,
        "commission_pct": _money(rule.commission_pct),
        "payment_fee_pct": _money(rule.payment_fee_pct),
        "fixed_fee": _money(rule.fixed_fee),
        "advertising_pct": _money(rule.advertising_pct),
        "other_reserve_fee": _money(rule.other_reserve_fee),
        "priority": rule.priority or 0,
        "status": rule.status or 0,
        "remark": rule.remark,
        "created_at": rule.created_at,
        "updated_at": rule.updated_at,
    }


class PlatformFeeRuleService:
    @staticmethod
    async def list(
        db: AsyncSession,
        platform_id: Optional[int] = None,
        site_code: Optional[str] = None,
        category_id: Optional[int] = None,
        status: Optional[int] = None,
    ) -> list[dict]:
        stmt = select(PlatformFeeRule).options(
            selectinload(PlatformFeeRule.platform),
            selectinload(PlatformFeeRule.category),
        )
        if platform_id is not None:
            stmt = stmt.where(PlatformFeeRule.platform_id == platform_id)
        if site_code is not None:
            stmt = stmt.where(PlatformFeeRule.site_code == (site_code.strip().upper() or None))
        if category_id is not None:
            stmt = stmt.where(PlatformFeeRule.category_id == category_id)
        if status is not None:
            stmt = stmt.where(PlatformFeeRule.status == status)
        stmt = stmt.order_by(PlatformFeeRule.platform_id, PlatformFeeRule.priority, PlatformFeeRule.id)
        result = await db.execute(stmt)
        return [_to_dict(rule) for rule in result.scalars().all()]

    @staticmethod
    async def get_by_id(db: AsyncSession, rule_id: int) -> Optional[dict]:
        result = await db.execute(
            select(PlatformFeeRule)
            .options(selectinload(PlatformFeeRule.platform), selectinload(PlatformFeeRule.category))
            .where(PlatformFeeRule.id == rule_id)
        )
        rule = result.scalar_one_or_none()
        return _to_dict(rule) if rule else None

    @staticmethod
    async def create(db: AsyncSession, data: PlatformFeeRuleCreate) -> dict:
        rule = PlatformFeeRule(
            platform_id=data.platform_id,
            site_code=data.site_code,
            category_id=data.category_id,
            commission_pct=Decimal(str(data.commission_pct)),
            payment_fee_pct=Decimal(str(data.payment_fee_pct)),
            fixed_fee=Decimal(str(data.fixed_fee)),
            advertising_pct=Decimal(str(data.advertising_pct)),
            other_reserve_fee=Decimal(str(data.other_reserve_fee)),
            priority=data.priority,
            remark=data.remark,
        )
        db.add(rule)
        await db.flush()
        await db.refresh(rule, ["platform", "category"])
        return _to_dict(rule)

    @staticmethod
    async def update(db: AsyncSession, rule_id: int, data: PlatformFeeRuleUpdate) -> Optional[dict]:
        result = await db.execute(
            select(PlatformFeeRule)
            .options(selectinload(PlatformFeeRule.platform), selectinload(PlatformFeeRule.category))
            .where(PlatformFeeRule.id == rule_id)
        )
        rule = result.scalar_one_or_none()
        if not rule:
            return None
        update_data = data.model_dump(exclude_unset=True)
        for key, value in update_data.items():
            if key in {"commission_pct", "payment_fee_pct", "fixed_fee", "advertising_pct", "other_reserve_fee"}:
                value = Decimal(str(value))
            setattr(rule, key, value)
        await db.flush()
        await db.refresh(rule)
        return _to_dict(rule)

    @staticmethod
    async def delete(db: AsyncSession, rule_id: int) -> bool:
        result = await db.execute(select(PlatformFeeRule).where(PlatformFeeRule.id == rule_id))
        rule = result.scalar_one_or_none()
        if not rule:
            return False
        rule.status = 0
        await db.flush()
        return True

    @staticmethod
    async def match(db: AsyncSession, data: PlatformFeeRuleMatchRequest) -> Optional[dict]:
        site_code = data.site_code.strip().upper() if data.site_code else None
        site_code = site_code or None
        category_id = data.category_id

        candidate_conditions = [
            and_(
                PlatformFeeRule.platform_id == data.platform_id,
                PlatformFeeRule.site_code == site_code,
                PlatformFeeRule.category_id == category_id,
            ),
            and_(
                PlatformFeeRule.platform_id == data.platform_id,
                PlatformFeeRule.site_code == site_code,
                PlatformFeeRule.category_id.is_(None),
            ),
            and_(
                PlatformFeeRule.platform_id == data.platform_id,
                PlatformFeeRule.site_code.is_(None),
                PlatformFeeRule.category_id.is_(None),
            ),
        ]

        result = await db.execute(
            select(PlatformFeeRule)
            .options(selectinload(PlatformFeeRule.platform), selectinload(PlatformFeeRule.category))
            .where(PlatformFeeRule.status == 1)
            .where(or_(*candidate_conditions))
            .order_by(
                PlatformFeeRule.priority,
                PlatformFeeRule.id,
            )
        )
        rules = list(result.scalars().all())
        if not rules:
            return None

        def rank(rule: PlatformFeeRule) -> int:
            if rule.site_code == site_code and rule.category_id == category_id:
                return 0
            if rule.site_code == site_code and rule.category_id is None:
                return 1
            return 2

        rules.sort(key=lambda rule: (rank(rule), rule.priority or 0, rule.id))
        return _to_dict(rules[0])
```

- [ ] **Step 5: Export router placeholder**

Create `backend/app/platform_fee/__init__.py`:

```python
from .router import router
```

At this point `router.py` does not exist yet, so imports will fail in full app startup. Continue immediately to Task 3 before running full tests.

## Task 3: Backend CRUD And Match API

**Files:**
- Create: `backend/app/platform_fee/router.py`
- Modify: `backend/tests/test_platform_fee_rules.py`

- [ ] **Step 1: Add failing API tests**

Append to `backend/tests/test_platform_fee_rules.py`:

```python
class TestPlatformFeeRuleApi:
    async def test_list_rules_requires_platform_fee_view(self, async_client):
        headers = await _auth(async_client, "pf_view_no")
        resp = await async_client.get("/api/platform-fee-rules", headers=headers)
        assert resp.status_code == 403

    async def test_create_rule_requires_platform_fee_manage(self, async_client):
        platform_id = await _create_platform(async_client, "pf_create_no")
        headers = await _auth(async_client, "pf_manage_no")
        resp = await async_client.post(
            "/api/platform-fee-rules",
            json={"platform_id": platform_id, "site_code": "RU", "commission_pct": 10},
            headers=headers,
        )
        assert resp.status_code == 403

    async def test_create_list_update_delete_rule_with_permission(self, async_client):
        platform_id = await _create_platform(async_client, "pf_crud")
        headers = await _auth(async_client, "pf_crud_ok", "platform_fee:manage")
        view_headers = await _auth(async_client, "pf_view_ok", "platform_fee:view")

        create_resp = await async_client.post(
            "/api/platform-fee-rules",
            json={
                "platform_id": platform_id,
                "site_code": "ru",
                "commission_pct": 12,
                "payment_fee_pct": 3,
                "fixed_fee": 5,
                "advertising_pct": 2,
                "other_reserve_fee": 1,
                "remark": "CRUD test",
            },
            headers=headers,
        )
        assert create_resp.status_code == 200
        rule = create_resp.json()["data"]
        assert rule["site_code"] == "RU"
        assert rule["commission_pct"] == 12.0
        assert await _count_logs("platform_fee_rule", "create", str(rule["id"])) == 1

        list_resp = await async_client.get(
            f"/api/platform-fee-rules?platform_id={platform_id}",
            headers=view_headers,
        )
        assert list_resp.status_code == 200
        assert any(item["id"] == rule["id"] for item in list_resp.json()["data"])

        update_resp = await async_client.put(
            f"/api/platform-fee-rules/{rule['id']}",
            json={"commission_pct": 14, "remark": "updated"},
            headers=headers,
        )
        assert update_resp.status_code == 200
        assert update_resp.json()["data"]["commission_pct"] == 14.0
        assert await _count_logs("platform_fee_rule", "update", str(rule["id"])) == 1

        delete_resp = await async_client.delete(
            f"/api/platform-fee-rules/{rule['id']}",
            headers=headers,
        )
        assert delete_resp.status_code == 200
        assert await _count_logs("platform_fee_rule", "delete", str(rule["id"])) == 1

    async def test_match_endpoint_requires_platform_fee_calculate(self, async_client):
        platform_id = await _create_platform(async_client, "pf_match_no")
        headers = await _auth(async_client, "pf_calc_no")
        resp = await async_client.post(
            "/api/platform-fee-rules/match",
            json={"platform_id": platform_id, "site_code": "RU"},
            headers=headers,
        )
        assert resp.status_code == 403

    async def test_match_endpoint_returns_best_rule(self, async_client):
        platform_id = await _create_platform(async_client, "pf_match_api")
        manage_headers = await _auth(async_client, "pf_match_manage", "platform_fee:manage")
        calc_headers = await _auth(async_client, "pf_match_calc", "platform_fee:calculate")
        await async_client.post(
            "/api/platform-fee-rules",
            json={"platform_id": platform_id, "site_code": "RU", "commission_pct": 9},
            headers=manage_headers,
        )
        await async_client.post(
            "/api/platform-fee-rules",
            json={"platform_id": platform_id, "site_code": "RU", "category_id": 777, "commission_pct": 18},
            headers=manage_headers,
        )

        resp = await async_client.post(
            "/api/platform-fee-rules/match",
            json={"platform_id": platform_id, "site_code": "RU", "category_id": 777},
            headers=calc_headers,
        )

        assert resp.status_code == 200
        assert resp.json()["data"]["commission_pct"] == 18.0
```

- [ ] **Step 2: Run API tests to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_platform_fee_rules.py::TestPlatformFeeRuleApi -q
```

Expected:

- FAIL because routes do not exist.

- [ ] **Step 3: Create router**

Create `backend/app/platform_fee/router.py`:

```python
"""平台费用规则 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result
from app.database import get_db
from app.models import User
from app.operation_log.service import OperationLogService
from app.platform_fee.schemas import (
    PlatformFeeRuleCreate,
    PlatformFeeRuleMatchRequest,
    PlatformFeeRuleUpdate,
)
from app.platform_fee.service import PlatformFeeRuleService


router = APIRouter(tags=["平台费用规则"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.get("/platform-fee-rules", summary="平台费用规则列表")
async def list_platform_fee_rules(
    platform_id: int = Query(None, description="平台ID"),
    site_code: str = Query(None, description="站点/国家代码"),
    category_id: int = Query(None, description="本地类目ID"),
    status: int = Query(None, description="状态"),
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("platform_fee:view")),
):
    rules = await PlatformFeeRuleService.list(db, platform_id, site_code, category_id, status)
    return Result.ok(rules)


@router.get("/platform-fee-rules/{rule_id}", summary="平台费用规则详情")
async def get_platform_fee_rule(
    rule_id: int,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("platform_fee:view")),
):
    rule = await PlatformFeeRuleService.get_by_id(db, rule_id)
    if not rule:
        return Result.not_found("平台费用规则不存在")
    return Result.ok(rule)


@router.post("/platform-fee-rules", summary="创建平台费用规则")
async def create_platform_fee_rule(
    data: PlatformFeeRuleCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    rule = await PlatformFeeRuleService.create(db, data)
    await OperationLogService.log(
        db,
        module="platform_fee_rule",
        action="create",
        resource_id=str(rule["id"]),
        content=f"创建平台费用规则: platform_id={rule['platform_id']}, site_code={rule['site_code']}",
        operator=_operator(current_user),
    )
    return Result.ok(rule)


@router.put("/platform-fee-rules/{rule_id}", summary="更新平台费用规则")
async def update_platform_fee_rule(
    rule_id: int,
    data: PlatformFeeRuleUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    rule = await PlatformFeeRuleService.update(db, rule_id, data)
    if not rule:
        return Result.not_found("平台费用规则不存在")
    await OperationLogService.log(
        db,
        module="platform_fee_rule",
        action="update",
        resource_id=str(rule_id),
        content=f"更新平台费用规则: {rule_id}",
        operator=_operator(current_user),
    )
    return Result.ok(rule)


@router.delete("/platform-fee-rules/{rule_id}", summary="禁用平台费用规则")
async def delete_platform_fee_rule(
    rule_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("platform_fee:manage")),
):
    ok = await PlatformFeeRuleService.delete(db, rule_id)
    if not ok:
        return Result.not_found("平台费用规则不存在")
    await OperationLogService.log(
        db,
        module="platform_fee_rule",
        action="delete",
        resource_id=str(rule_id),
        content=f"禁用平台费用规则: {rule_id}",
        operator=_operator(current_user),
    )
    return Result.ok()


@router.post("/platform-fee-rules/match", summary="匹配平台费用规则")
async def match_platform_fee_rule(
    data: PlatformFeeRuleMatchRequest,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("platform_fee:calculate")),
):
    rule = await PlatformFeeRuleService.match(db, data)
    return Result.ok(rule)
```

- [ ] **Step 4: Run platform fee rule tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_platform_fee_rules.py -q
```

Expected:

- PASS.

- [ ] **Step 5: Commit backend API**

Run:

```bash
git add backend/app/platform_fee backend/tests/test_platform_fee_rules.py
git commit -m "feat: add platform fee rule api"
```

## Task 4: Integrate Platform Fee Rules Into Pre-Listing Decisions

**Files:**
- Modify: `backend/app/decision/schemas.py`
- Modify: `backend/app/decision/service.py`
- Modify: `backend/tests/test_prelisting_decision.py`

- [ ] **Step 1: Add failing decision integration tests**

Append to `backend/tests/test_prelisting_decision.py`:

```python
async def test_prelisting_decision_uses_platform_fee_rule(async_client):
    from tests.auth_helpers import grant_permission, register_and_login
    from uuid import uuid4

    uid, token = await register_and_login(async_client, "decision_fee_rule")
    await grant_permission(uid, "platform:create")
    await grant_permission(uid, "platform_fee:manage")
    await grant_permission(uid, "decision:calculate")
    headers = {"Authorization": f"Bearer {token}"}

    platform_resp = await async_client.post(
        "/api/platforms",
        json={"name": f"Ozon-{uuid4().hex[:6]}", "code": f"ozon_{uuid4().hex[:6]}"},
        headers=headers,
    )
    assert platform_resp.status_code == 200
    platform_id = platform_resp.json()["data"]["id"]

    await async_client.post(
        "/api/platform-fee-rules",
        json={
            "platform_id": platform_id,
            "site_code": "RU",
            "commission_pct": 20,
            "payment_fee_pct": 5,
            "fixed_fee": 3,
            "advertising_pct": 2,
            "other_reserve_fee": 4,
        },
        headers=headers,
    )

    sku_id = await _seed_decision_ready_sku(async_client, headers)
    await _seed_shipping_quote(async_client, headers, "RU")

    resp = await async_client.post(
        "/api/decisions/prelisting",
        json={
            "sku_id": sku_id,
            "destination_country": "RU",
            "target_sale_price": 200,
            "platform_id": platform_id,
            "platform_fee_pct": 1,
            "payment_fee_pct": 1,
            "other_fee": 0,
            "minimum_margin_pct": 10,
        },
        headers=headers,
    )

    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data["applied_platform_fee_rule_id"] is not None
    assert data["platform_fee_source"] == "rule"
    assert data["platform_fee"] == 40.0
    assert data["payment_fee"] == 10.0
    assert data["advertising_fee"] == 4.0
    assert data["fixed_fee"] == 3.0
    assert data["other_fee"] == 4.0


async def test_prelisting_decision_falls_back_to_manual_fee_when_no_rule_matches(async_client):
    from tests.auth_helpers import grant_permission, register_and_login
    from uuid import uuid4

    uid, token = await register_and_login(async_client, "decision_fee_fallback")
    await grant_permission(uid, "platform:create")
    await grant_permission(uid, "decision:calculate")
    headers = {"Authorization": f"Bearer {token}"}

    platform_resp = await async_client.post(
        "/api/platforms",
        json={"name": f"Shopee-{uuid4().hex[:6]}", "code": f"shopee_{uuid4().hex[:6]}"},
        headers=headers,
    )
    assert platform_resp.status_code == 200
    platform_id = platform_resp.json()["data"]["id"]

    sku_id = await _seed_decision_ready_sku(async_client, headers)
    await _seed_shipping_quote(async_client, headers, "MY")

    resp = await async_client.post(
        "/api/decisions/prelisting",
        json={
            "sku_id": sku_id,
            "destination_country": "MY",
            "target_sale_price": 200,
            "platform_id": platform_id,
            "platform_fee_pct": 10,
            "payment_fee_pct": 3,
            "other_fee": 2,
            "minimum_margin_pct": 10,
        },
        headers=headers,
    )

    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data["applied_platform_fee_rule_id"] is None
    assert data["platform_fee_source"] == "manual"
    assert data["platform_fee"] == 20.0
    assert data["payment_fee"] == 6.0
    assert "未匹配到平台费用规则" in "".join(data["warnings"])
```

If helper functions `_seed_decision_ready_sku` or `_seed_shipping_quote` do not exist in the file, create them by reusing the setup code already present in `backend/tests/test_prelisting_decision.py`. Keep helpers local to that test file.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision.py -q
```

Expected:

- FAIL because decision schemas do not expose `platform_id` or applied rule fields.

- [ ] **Step 3: Extend decision schemas**

Modify `backend/app/decision/schemas.py`:

```python
from typing import Optional
```

Add fields to `PreListingDecisionRequest`:

```python
    platform_id: Optional[int] = Field(None, description="平台ID；填写后尝试匹配平台费用规则")
    category_id: Optional[int] = Field(None, description="本地类目ID；为空时从SKU所属商品读取")
```

Add fields to `PreListingDecisionResponse`:

```python
    fixed_fee: float = 0
    advertising_fee: float = 0
    applied_platform_fee_rule_id: Optional[int] = None
    platform_fee_source: str = "manual"  # manual / rule
    platform_fee_rule_summary: Optional[str] = None
```

- [ ] **Step 4: Update decision service**

Modify `backend/app/decision/service.py`:

```python
from app.platform_fee.schemas import PlatformFeeRuleMatchRequest
from app.platform_fee.service import PlatformFeeRuleService
```

After loading `sku`, load `product`:

```python
        product_stmt = select(Product).where(Product.id == sku.product_id)
        product_result = await db.execute(product_stmt)
        product = product_result.scalar_one_or_none()
        category_id = req.category_id if req.category_id is not None else (product.category_id if product else None)
```

Replace fee calculation with:

```python
        platform_fee_source = "manual"
        applied_platform_fee_rule_id = None
        platform_fee_rule_summary = None
        fixed_fee = 0.0
        advertising_fee = 0.0
        other_fee = req.other_fee

        platform_fee_pct = req.platform_fee_pct
        payment_fee_pct = req.payment_fee_pct

        if req.platform_id is not None:
            matched_rule = await PlatformFeeRuleService.match(
                db,
                PlatformFeeRuleMatchRequest(
                    platform_id=req.platform_id,
                    site_code=req.destination_country,
                    category_id=category_id,
                ),
            )
            if matched_rule:
                platform_fee_source = "rule"
                applied_platform_fee_rule_id = matched_rule["id"]
                platform_fee_rule_summary = (
                    f"{matched_rule.get('platform_name') or req.platform_id} "
                    f"{matched_rule.get('site_code') or 'GLOBAL'}"
                )
                platform_fee_pct = matched_rule["commission_pct"]
                payment_fee_pct = matched_rule["payment_fee_pct"]
                fixed_fee = matched_rule["fixed_fee"]
                advertising_fee = req.target_sale_price * matched_rule["advertising_pct"] / 100
                other_fee = matched_rule["other_reserve_fee"]
            else:
                warnings.append("未匹配到平台费用规则，使用手动输入费率")

        platform_fee = req.target_sale_price * platform_fee_pct / 100
        payment_fee = req.target_sale_price * payment_fee_pct / 100

        total_cost = product_cost + shipping_fee + platform_fee + payment_fee + fixed_fee + advertising_fee + other_fee
```

Return the new fields:

```python
            fixed_fee=round(fixed_fee, 2),
            advertising_fee=round(advertising_fee, 2),
            applied_platform_fee_rule_id=applied_platform_fee_rule_id,
            platform_fee_source=platform_fee_source,
            platform_fee_rule_summary=platform_fee_rule_summary,
```

- [ ] **Step 5: Run decision tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision.py tests/test_platform_fee_rules.py -q
```

Expected:

- PASS.

- [ ] **Step 6: Commit decision integration**

Run:

```bash
git add backend/app/decision/schemas.py backend/app/decision/service.py backend/tests/test_prelisting_decision.py
git commit -m "feat: apply platform fee rules to decisions"
```

## Task 5: Permissions, Seed, And Docs

**Files:**
- Modify: `backend/seed.py`
- Modify: `docs/PERMISSIONS_AND_AUDIT.md`
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Add permission seed**

In `backend/seed.py`, append these items to `SEED_PERMISSIONS`:

```python
    {"code": "platform_fee:view", "name": "查看平台费用规则", "module": "platform_fee"},
    {"code": "platform_fee:manage", "name": "管理平台费用规则", "module": "platform_fee"},
    {"code": "platform_fee:calculate", "name": "匹配平台费用规则", "module": "platform_fee"},
```

- [ ] **Step 2: Update permissions guide**

In `docs/PERMISSIONS_AND_AUDIT.md`, add this row to the permissions table:

```markdown
| 平台费用规则 | `platform_fee:view`, `platform_fee:manage`, `platform_fee:calculate` | 已覆盖 |
```

Add this row to the audit coverage table:

```markdown
| 平台费用规则 | create, update, delete |
```

- [ ] **Step 3: Update project status**

In `docs/PROJECT_STATUS.md`, add:

```markdown
### 平台费用规则

状态：已完成第一版。

已实现：
- 本地维护平台费用规则。
- 支持平台全局、平台+站点、平台+站点+类目三级匹配。
- 上架前经营决策可自动套用匹配到的平台费用规则。
- 无匹配规则时继续使用手动费率并返回 warning。
- 写操作接入权限和审计日志。

仍未实现：
- 真实平台费率 API 同步。
- 费用规则批量导入导出。
- 完整前端规则管理页。
- 跨平台类目映射。
```

- [ ] **Step 4: Update roadmap**

In `docs/ROADMAP.md`, update the recommended next task:

````markdown
最推荐继续做：

```text
批量上架前经营决策。
```

原因：

- 单 SKU 上架决策已经能结合商品成本、运费和平台费用规则。
- 下一步应让运营一次性评估多个商品/SKU。
- 批量结果可以直接驱动补物流数据、调价和上架准备。
````

- [ ] **Step 5: Commit seed and docs**

Run:

```bash
git add backend/seed.py docs/PERMISSIONS_AND_AUDIT.md docs/PROJECT_STATUS.md docs/ROADMAP.md
git commit -m "docs: document platform fee rules"
```

## Task 6: Minimal Frontend Decision Integration

**Files:**
- Modify: `frontend/src/api/modules/decision.ts`
- Modify: `frontend/src/views/decision/PreListingDecision.vue`

- [ ] **Step 1: Update frontend API types**

In `frontend/src/api/modules/decision.ts`, add these fields:

```ts
export interface PreListingDecisionRequest {
  sku_id: number
  destination_country: string
  target_sale_price: number
  platform_id?: number | null
  category_id?: number | null
  platform_fee_pct: number
  payment_fee_pct: number
  other_fee: number
  minimum_margin_pct: number
  cargo_type: string
}

export interface PreListingDecisionResponse {
  sku_id: number
  destination_country: string
  target_sale_price: number
  product_cost: number
  shipping_fee: number
  platform_fee: number
  payment_fee: number
  fixed_fee: number
  advertising_fee: number
  other_fee: number
  profit_amount: number
  profit_margin: number
  recommendation: string
  blocking_reasons: string[]
  warnings: string[]
  applied_platform_fee_rule_id?: number | null
  platform_fee_source: string
  platform_fee_rule_summary?: string | null
}
```

- [ ] **Step 2: Update decision page form**

In `frontend/src/views/decision/PreListingDecision.vue`:

- Add `platform_id` as a nullable numeric input or select.
- Keep manual fee inputs visible so fallback remains usable.
- Display `platform_fee_source`, `applied_platform_fee_rule_id`, and `platform_fee_rule_summary` in the result panel.
- Display `fixed_fee` and `advertising_fee` alongside existing cost items.

Use this result display text:

```vue
<n-descriptions-item label="费用规则来源">
  {{ result.platform_fee_source === 'rule' ? '规则库' : '手动输入' }}
  <span v-if="result.applied_platform_fee_rule_id">
    #{{ result.applied_platform_fee_rule_id }} {{ result.platform_fee_rule_summary || '' }}
  </span>
</n-descriptions-item>
<n-descriptions-item label="固定交易费">{{ result.fixed_fee }}</n-descriptions-item>
<n-descriptions-item label="广告预留">{{ result.advertising_fee }}</n-descriptions-item>
```

- [ ] **Step 3: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- PASS.

- [ ] **Step 4: Commit frontend integration**

Run:

```bash
git add frontend/src/api/modules/decision.ts frontend/src/views/decision/PreListingDecision.vue
git commit -m "feat: show platform fee rules in decision page"
```

## Task 7: Final Verification

**Files:**
- Read: repository root
- Modify: none unless verification reveals a real bug

- [ ] **Step 1: Run backend full suite**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest -q
```

Expected:

- All tests pass. Current baseline before this plan was `217 passed`; this task should add tests, so the final pass count should be higher.

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 3: Check git status and log**

Run:

```bash
git status --short --branch
git log --oneline --decorate -6
```

Expected:

- Working tree is clean.
- Recent commits correspond to the tasks above.

## Final Acceptance Criteria

The task is complete only when:

- `platform_fee_rule` table exists via Alembic migration.
- CRUD endpoints work and enforce permissions.
- Match endpoint returns the correct priority rule.
- Disabled rules do not match.
- Pre-listing decision uses matched rule when `platform_id` is provided.
- Pre-listing decision falls back to manual fee inputs when no rule matches.
- Response explains whether platform fees came from `rule` or `manual`.
- New permission codes are seeded and documented.
- Backend full tests pass.
- Frontend production build passes.

## Recommended Agent Prompt

Give this to the implementing agent:

```text
你接手的是 /Users/lc/multisell 的 LingMirror / MultiSell 项目。

先阅读：
- docs/platform-fee-rules-plan.md
- docs/superpowers/plans/2026-06-15-platform-fee-rules.md
- backend/app/decision/schemas.py
- backend/app/decision/service.py
- backend/app/shipping/service.py
- backend/app/platform/router.py
- docs/PERMISSIONS_AND_AUDIT.md

请在新分支 codex/platform-fee-rules 上按计划逐任务执行。严格 TDD：先写失败测试，再写实现。不要接真实平台 API，不要做完整管理后台 UI，不要改无关模块。

完成后必须运行：
- cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
- cd frontend && npm run build

交付时说明：
- 改了哪些文件
- 新增了哪些 API
- 匹配优先级如何工作
- 测试命令和结果
- 剩余限制
```
