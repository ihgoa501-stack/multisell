> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# LingMirror AgentOS Phase 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add approval audit log persistence, autonomy level upgrade/downgrade rules engine, and corresponding frontend management UI.

**Architecture:** Create `backend/app/agentos/models.py` for the `AgentOSOperationLog` SQLAlchemy table (new migration). Add autonomy upgrade rules as pure functions in a new `backend/app/agentos/autonomy_service.py`, reusable by tests without DB. Extend the existing router and service for log queries and upgrade/downgrade actions. Add a frontend autonomy management page and a small upgrade-candidate panel on the Agent Squads page.

**Tech Stack:** FastAPI, async SQLAlchemy 2.0, Alembic, PostgreSQL, pytest, Vue 3, TypeScript, Naive UI.

**Prerequisites:** Phase 1 (aggregation shell) and Phase 2 (status write + approve/reject) are deployed and all 30 tests pass.

---

## Scope Guard

This plan implements the Phase 3 vertical slice only:

- AgentOSOperationLog table + migration
- Operation log auto-write on approve/reject/status-update
- Operation log query API
- Autonomy upgrade rule engine (pure functions)
- Autonomy upgrade/downgrade API
- Frontend autonomy management view
- Frontend upgrade candidate panel on Squad page

This plan does NOT:
- Introduce real LLM calls
- Implement a full workflow engine
- Add real-time notifications
- Create a desktop or IM client
- Add a new database table for "WorkItem" (still uses aggregation pattern)

## File Structure

### Create
- `backend/app/agentos/models.py` — SQLAlchemy model for AgentOSOperationLog
- `backend/app/agentos/autonomy_service.py` — pure-function upgrade/downgrade rules
- `backend/alembic/versions/20260618_03_add_agentos_operation_log.py` — Alembic migration
- `frontend/src/views/agentos/AutonomyManagement.vue` — autonomy management page
- `frontend/src/components/agentos/AutonomyUpgradeCard.vue` — upgrade suggestion card

### Modify
- `backend/app/agentos/schemas.py` — add AgentOSOperationLogVO, AutonomyUpgradeRequest, AutonomyCandidateVO
- `backend/app/agentos/service.py` — wire operation log writes into approve/reject/status-update; add get_operations, get_upgrade_candidates, execute_upgrade, execute_downgrade
- `backend/app/agentos/router.py` — add operation log and autonomy upgrade endpoints
- `frontend/src/api/modules/agentos.ts` — add operation log and autonomy upgrade API methods
- `frontend/src/router/modules/agentos.ts` — add autonomy management route
- `frontend/src/views/agentos/Squads.vue` — add upgrade candidate panel
- `backend/tests/test_agentos_phase1.py` — add Phase 3 test classes

---

## Task 1: Operation Log Table + Migration

**Files:**
- Create: `backend/app/agentos/models.py`
- Create: `backend/alembic/versions/20260618_03_add_agentos_operation_log.py`
- Test: auto-verified by running upgrade heads

- [ ] **Step 1: Create the SQLAlchemy model**

Create `backend/app/agentos/models.py`:

```python
"""AgentOS 持久化模型"""
from datetime import datetime

from sqlalchemy import Column, DateTime, Integer, String, Text
from sqlalchemy import func as sa_func

from app.database import Base


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
```

- [ ] **Step 2: Generate and inspect the Alembic migration**

Run:

```bash
cd backend && .venv/bin/alembic revision --autogenerate -m "add agentos_operation_log"
```

Open the generated file in `backend/alembic/versions/` and verify it contains:

```python
def upgrade() -> None:
    op.create_table('agentos_operation_log',
        sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('item_id', sa.String(length=128), nullable=False),
        sa.Column('action', sa.String(length=32), nullable=False),
        sa.Column('source_type', sa.String(length=32), nullable=True),
        sa.Column('previous_status', sa.String(length=32), nullable=True),
        sa.Column('new_status', sa.String(length=32), nullable=True),
        sa.Column('comment', sa.Text(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa_func.now(), nullable=False),
        sa.PrimaryKeyConstraint('id'),
    )
    op.create_index(op.f('ix_agentos_operation_log_user_id'), 'agentos_operation_log', ['user_id'], unique=False)

def downgrade() -> None:
    op.drop_index(op.f('ix_agentos_operation_log_user_id'), table_name='agentos_operation_log')
    op.drop_table('agentos_operation_log')
```

If migration is empty, hand-edit it to match the above.

- [ ] **Step 3: Run the migration**

```bash
cd backend && .venv/bin/alembic upgrade heads
```

Expected: no errors. Verify with:

```bash
cd backend && .venv/bin/alembic current --verbose
```

- [ ] **Step 4: Add the Pydantic response schema**

In `backend/app/agentos/schemas.py`, before the `# ─── 请求模型` section, add:

```python
# ─── Phase 3 模型 ─────────────────────────────────────────


class AgentOSOperationLogVO(BaseModel):
    """操作审计日志"""
    id: int
    user_id: int
    item_id: str
    action: str
    source_type: Optional[str] = None
    previous_status: Optional[str] = None
    new_status: Optional[str] = None
    comment: Optional[str] = None
    created_at: Optional[datetime] = None
```

After the `WorkItemApproval` class, add:

```python
class OperationLogQuery(BaseModel):
    item_id: Optional[str] = None
    action: Optional[str] = None
    source_type: Optional[str] = None
    limit: int = 20
    offset: int = 0
```

- [ ] **Step 5: Commit Task 1**

```bash
git add backend/app/agentos/models.py backend/alembic/versions/20260618_03_add_agentos_operation_log.py backend/app/agentos/schemas.py
git commit -m "feat(agentos): add operation log table and migration"
```

---

## Task 2: Wire Operation Log Writes Into Service

**Files:**
- Modify: `backend/app/agentos/service.py`

- [ ] **Step 1: Add _write_operation_log helper**

Inside `AgentOSService`, add before the `# ── Phase 2` section:

```python
    # ── Phase 3: Operation Log ──────────────────────────

    @staticmethod
    def _extract_source_type(item_id: str) -> str:
        return item_id.split(":")[0] if ":" in item_id else "unknown"

    @staticmethod
    async def _write_operation_log(
        db: AsyncSession,
        user_id: int,
        item_id: str,
        action: str,
        previous_status: str | None = None,
        new_status: str | None = None,
        comment: str | None = None,
    ) -> None:
        """写入操作审计日志"""
        from app.agentos.models import AgentOSOperationLog

        log = AgentOSOperationLog(
            user_id=user_id,
            item_id=item_id,
            action=action,
            source_type=AgentOSService._extract_source_type(item_id),
            previous_status=previous_status,
            new_status=new_status,
            comment=comment,
        )
        db.add(log)
```

- [ ] **Step 2: Wire into update_work_item_status**

Add these lines at the end of `update_work_item_status`, just before the `return {"ok": True, "new_status": new_status}` line:

```python
        await AgentOSService._write_operation_log(
            db, user_id, item_id, "status_update",
            previous_status=None, new_status=new_status,
        )
```

Note: the method signature already includes `status` so we can compute `previous_status` from the row before mutation. Update the method to capture the old status before changing it. In each branch, before `row.status = ...`, capture `old = str(row.status)` and pass it to `_write_operation_log`.

- [ ] **Step 3: Wire into approve_work_item**

Add at the end, before the final return:

```python
        await AgentOSService._write_operation_log(
            db, user_id, item_id, "approve",
            previous_status="pending", new_status="in_progress",
            comment=comment,
        )
```

- [ ] **Step 4: Wire into reject_work_item**

Add at the end, before the final return:

```python
        await AgentOSService._write_operation_log(
            db, user_id, item_id, "reject",
            previous_status="pending", new_status="cancelled",
            comment=comment,
        )
```

- [ ] **Step 5: Run Phase 1+2 tests to verify no regression**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py -q
```

Expected: 30 passed.

- [ ] **Step 6: Commit Task 2**

```bash
git add backend/app/agentos/service.py
git commit -m "feat(agentos): wire operation log writes into mutations"
```

---

## Task 3: Operation Log Query API

**Files:**
- Modify: `backend/app/agentos/service.py`
- Modify: `backend/app/agentos/router.py`

- [ ] **Step 1: Add get_operations to service**

Inside `AgentOSService`, add after `_write_operation_log`:

```python
    @staticmethod
    async def get_operations(
        db: AsyncSession,
        item_id: str | None = None,
        action: str | None = None,
        source_type: str | None = None,
        user_id: int | None = None,
        limit: int = 20,
        offset: int = 0,
    ) -> dict[str, Any]:
        """查询操作审计日志"""
        from app.agentos.models import AgentOSOperationLog

        query = select(AgentOSOperationLog)
        if item_id:
            query = query.where(AgentOSOperationLog.item_id == item_id)
        if action:
            query = query.where(AgentOSOperationLog.action == action)
        if source_type:
            query = query.where(AgentOSOperationLog.source_type == source_type)
        if user_id:
            query = query.where(AgentOSOperationLog.user_id == user_id)

        count_q = select(sa_func.count()).select_from(query.subquery())
        total = (await db.execute(count_q)).scalar() or 0

        rows = (
            await db.execute(
                query.order_by(AgentOSOperationLog.created_at.desc())
                .offset(offset)
                .limit(limit)
            )
        ).scalars().all()

        return {
            "records": [AgentOSOperationLogVO.model_validate(r) for r in rows],
            "total": total,
            "limit": limit,
            "offset": offset,
        }
```

Add import for `AgentOSOperationLogVO` from schemas at top of service:

```python
from app.agentos.schemas import (
    ...,
    AgentOSOperationLogVO,
)
```

- [ ] **Step 2: Add endpoint to router**

Before the `# ── Phase 2` section comment, add:

```python
@router.get("/agentos/operations", summary="AgentOS 操作审计日志")
async def list_operations(
    item_id: str | None = None,
    action: str | None = None,
    source_type: str | None = None,
    user_id: int | None = None,
    limit: int = 20,
    offset: int = 0,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """查询审批/状态变更操作日志"""
    data = await AgentOSService.get_operations(
        db, item_id=item_id, action=action, source_type=source_type,
        user_id=user_id, limit=limit, offset=offset,
    )
    return PageResult.ok(
        records=data["records"],
        total=data["total"],
        page=(offset // limit) + 1,
        page_size=limit,
    )
```

- [ ] **Step 3: Run tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py -q
```

Expected: 30 passed.

- [ ] **Step 4: Commit Task 3**

```bash
git add backend/app/agentos/service.py backend/app/agentos/router.py
git commit -m "feat(agentos): add operation log query API"
```

---

## Task 4: Autonomy Upgrade Rules Engine (Pure Functions)

**Files:**
- Create: `backend/app/agentos/autonomy_service.py`
- Test: Phase 3 test section in `backend/tests/test_agentos_phase1.py`

- [ ] **Step 1: Write the failing test**

Add a new test section in `test_agentos_phase1.py` after the Phase 2 tests:

```python
# ─── Phase 3: Autonomy Upgrade Tests ──────────────────────


def test_suggest_upgrade_high_success_rate():
    """成功率 > 90% 应建议升级"""
    from app.agentos.autonomy_service import suggest_upgrade

    result = suggest_upgrade(
        agent_id="A5",
        current_level="SUGGESTION",
        success_rate=0.95,
        adoption_rate=0.85,
        recent_risk_levels=["low", "low", "medium"],
        total_decisions=50,
        recent_errors=1,
    )
    assert result["suggested"] is True
    assert result["target_level"] == "SEMI_AUTONOMOUS"
    assert result["confidence"] >= 0.7


def test_suggest_upgrade_low_adoption():
    """采纳率 < 30% 不应建议升级"""
    from app.agentos.autonomy_service import suggest_upgrade

    result = suggest_upgrade(
        agent_id="A3",
        current_level="SUGGESTION",
        success_rate=0.85,
        adoption_rate=0.20,
        recent_risk_levels=["medium", "high"],
        total_decisions=20,
        recent_errors=3,
    )
    assert result["suggested"] is False


def test_suggest_downgrade_high_errors():
    """错误率高应建议降级"""
    from app.agentos.autonomy_service import suggest_upgrade

    result = suggest_upgrade(
        agent_id="A6",
        current_level="SEMI_AUTONOMOUS",
        success_rate=0.50,
        adoption_rate=0.40,
        recent_risk_levels=["critical", "high", "high"],
        total_decisions=10,
        recent_errors=5,
    )
    # 当前为 SEMI_AUTONOMOUS 且错误率高 → 建议降级
    assert result["suggested"] is True
    assert result["target_level"] == "SUGGESTION"
    assert result["direction"] == "downgrade"


def test_suggest_upgrade_full_autonomous_blocked():
    """FULL_AUTONOMOUS 不应再建议升级"""
    from app.agentos.autonomy_service import suggest_upgrade

    result = suggest_upgrade(
        agent_id="G1",
        current_level="FULL_AUTONOMOUS",
        success_rate=0.99,
        adoption_rate=0.95,
        recent_risk_levels=["low"],
        total_decisions=200,
        recent_errors=0,
    )
    assert result["suggested"] is False
    assert result["reason"] == "already_at_max"


def test_suggest_upgrade_insufficient_data():
    """决策量 < 10 条时不建议升级"""
    from app.agentos.autonomy_service import suggest_upgrade

    result = suggest_upgrade(
        agent_id="A1",
        current_level="OBSERVATION",
        success_rate=1.0,
        adoption_rate=1.0,
        recent_risk_levels=["low"],
        total_decisions=3,
        recent_errors=0,
    )
    assert result["suggested"] is False
    assert "insufficient" in result.get("reason", "").lower()
```

- [ ] **Step 2: Run tests to see failures**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py::test_suggest_upgrade_high_success_rate -v
```

Expected: FAIL with `ModuleNotFoundError: No module named 'app.agentos.autonomy_service'`.

- [ ] **Step 3: Implement the upgrade rules engine**

Create `backend/app/agentos/autonomy_service.py`:

```python
"""自治等级升级规则引擎 — 纯函数，无数据库依赖"""

from __future__ import annotations

from typing import Any

LEVEL_ORDER = ["OBSERVATION", "SUGGESTION", "SEMI_AUTONOMOUS", "FULL_AUTONOMOUS"]

RISK_SCORE = {"low": 1, "medium": 2, "high": 3, "critical": 4}


def suggest_upgrade(
    agent_id: str,
    current_level: str,
    success_rate: float,
    adoption_rate: float,
    recent_risk_levels: list[str],
    total_decisions: int,
    recent_errors: int,
) -> dict[str, Any]:
    """
    建议自治等级升降。

    返回:
        suggested: bool — 是否建议变更
        direction: "upgrade" | "downgrade" | None
        target_level: str — 目标等级
        confidence: float — 置信度 (0-1)
        reason: str — 理由说明
    """
    # 基础数据量检查
    if total_decisions < 10:
        return {
            "suggested": False,
            "direction": None,
            "target_level": current_level,
            "confidence": 0,
            "reason": "insufficient_data",
        }

    current_idx = LEVEL_ORDER.index(current_level) if current_level in LEVEL_ORDER else -1
    if current_idx == -1:
        return {
            "suggested": False,
            "direction": None,
            "target_level": current_level,
            "confidence": 0,
            "reason": "unknown_level",
        }

    # 计算风险得分
    risk_sum = sum(RISK_SCORE.get(r, 2) for r in recent_risk_levels)
    risk_avg = risk_sum / max(len(recent_risk_levels), 1)
    error_rate = recent_errors / max(total_decisions, 1)

    # 降级判断：高风险 + 低成功率
    if current_idx > 0 and (error_rate > 0.3 or (success_rate < 0.6 and risk_avg > 2.5)):
        target_idx = current_idx - 1
        return {
            "suggested": True,
            "direction": "downgrade",
            "target_level": LEVEL_ORDER[target_idx],
            "confidence": round(min(0.95, 0.5 + error_rate), 2),
            "reason": f"高风险(avg={risk_avg:.1f})/低成功率({success_rate:.0%})",
        }

    # 已达最高等级
    if current_idx >= len(LEVEL_ORDER) - 1:
        return {
            "suggested": False,
            "direction": None,
            "target_level": current_level,
            "confidence": 1.0,
            "reason": "already_at_max",
        }

    # 升级判断
    upgrade_signals = 0
    total_signals = 0

    # 信号1: 成功率 > 90%
    total_signals += 1
    if success_rate >= 0.9:
        upgrade_signals += 1

    # 信号2: 采纳率 > 70%
    total_signals += 1
    if adoption_rate >= 0.7:
        upgrade_signals += 1

    # 信号3: 最近无高风险
    total_signals += 1
    if risk_avg < 2.0:
        upgrade_signals += 1

    # 信号4: 错误率低
    total_signals += 1
    if error_rate < 0.05:
        upgrade_signals += 1

    confidence = upgrade_signals / max(total_signals, 1)

    if confidence >= 0.75 and current_idx < len(LEVEL_ORDER) - 1:
        target_idx = current_idx + 1
        return {
            "suggested": True,
            "direction": "upgrade",
            "target_level": LEVEL_ORDER[target_idx],
            "confidence": round(confidence, 2),
            "reason": f"高成功率({success_rate:.0%})/高采纳率({adoption_rate:.0%})",
        }

    return {
        "suggested": False,
        "direction": None,
        "target_level": current_level,
        "confidence": round(confidence, 2),
        "reason": "condition_not_met",
    }


def batch_suggest_upgrades(
    agents: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    """批量计算多个 Agent 的升级建议"""
    return [
        {
            "agent_id": a.get("id", ""),
            "current_level": a.get("autonomy_level", "SUGGESTION"),
            **suggest_upgrade(
                agent_id=a.get("id", ""),
                current_level=a.get("autonomy_level", "SUGGESTION"),
                success_rate=a.get("success_rate", 0),
                adoption_rate=a.get("adoption_rate", 0),
                recent_risk_levels=a.get("recent_risk_levels", []),
                total_decisions=a.get("total_decisions", 0),
                recent_errors=a.get("recent_errors", 0),
            ),
        }
        for a in agents
    ]
```

- [ ] **Step 4: Run upgrade rule tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py -k "test_suggest" -v
```

Expected: all 5 upgrade rule tests pass.

- [ ] **Step 5: Add AutonomyCandidateVO schema**

In `backend/app/agentos/schemas.py`, after `AgentOSOperationLogVO`, add:

```python
class AutonomyCandidateVO(BaseModel):
    """自治等级升级候选"""
    agent_id: str
    agent_name: str
    squad_id: str
    current_level: str
    suggested: bool
    direction: Optional[str] = None  # upgrade / downgrade / None
    target_level: Optional[str] = None
    confidence: float = 0
    reason: str = ""
```

- [ ] **Step 6: Commit Task 4**

```bash
git add backend/app/agentos/autonomy_service.py backend/app/agentos/schemas.py backend/tests/test_agentos_phase1.py
git commit -m "feat(agentos): add autonomy upgrade rules engine"
```

---

## Task 5: Autonomy Upgrade API + Service Integration

**Files:**
- Modify: `backend/app/agentos/service.py`
- Modify: `backend/app/agentos/router.py`

- [ ] **Step 1: Add upgrade service methods**

Inside `AgentOSService`, add after `_write_operation_log`:

```python
    @staticmethod
    async def get_upgrade_candidates(
        db: AsyncSession,
        user_id: int,
    ) -> list[dict[str, Any]]:
        """获取自治等级升级候选列表"""
        from app.agentos.autonomy_service import batch_suggest_upgrades

        # 从 AgentDecision 统计每个 Agent 的表现
        from app.agent.models import AgentDecision, AgentAction as PendingAgentAction

        agents_data = []
        for agent_id, meta in AGENT_META.items():
            squad_id = AGENT_TO_SQUAD.get(agent_id, "governance")

            # 统计决策数和采纳率
            decisions_total = 0
            decisions_accepted = 0
            try:
                stmt = select(AgentDecision).where(AgentDecision.agent_id == agent_id)
                all_decisions = (await db.execute(stmt)).scalars().all()
                decisions_total = len(all_decisions)
                decisions_accepted = sum(
                    1 for d in all_decisions if d.user_action == "accepted"
                )
            except Exception:
                pass

            # 统计最近风险
            recent_risks = []
            try:
                stmt = (
                    select(PendingAgentAction)
                    .where(PendingAgentAction.agent_id == agent_id)
                    .order_by(PendingAgentAction.created_at.desc())
                    .limit(20)
                )
                actions = (await db.execute(stmt)).scalars().all()
                for a in actions:
                    if a.status == "failed":
                        recent_risks.append("high")
                if not actions:
                    recent_risks.append("low")
            except Exception:
                recent_risks.append("low")

            # 成功率 = (总 - 失败) / 总
            error_count = sum(1 for r in recent_risks if r == "high")
            success_rate = max(0, 1 - (error_count / max(decisions_total, 1)))

            adoption_rate = decisions_accepted / max(decisions_total, 1) if decisions_total else 0

            agents_data.append({
                "id": agent_id,
                "autonomy_level": AutonomyLevel.SUGGESTION.value,
                "success_rate": round(success_rate, 3),
                "adoption_rate": round(adoption_rate, 3),
                "recent_risk_levels": recent_risks,
                "total_decisions": decisions_total,
                "recent_errors": error_count,
            })

        candidates = batch_suggest_upgrades(agents_data)

        # 关联 Agent 元数据
        result = []
        for c in candidates:
            aid = c["agent_id"]
            meta = AGENT_META.get(aid, {})
            squad_id = AGENT_TO_SQUAD.get(aid, "governance")
            result.append({
                "agent_id": aid,
                "agent_name": meta.get("name", aid),
                "squad_id": squad_id,
                "squad_name": SQUAD_TO_NAME.get(squad_id, squad_id),
                "current_level": c.get("current_level", "SUGGESTION"),
                "suggested": c["suggested"],
                "direction": c["direction"],
                "target_level": c["target_level"],
                "confidence": c["confidence"],
                "reason": c["reason"],
            })
        return result

    @staticmethod
    async def execute_upgrade(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        target_level: str,
    ) -> dict[str, Any]:
        """执行自治等级升级（记录到 operation_log）"""
        # Phase 3 仅记录升级操作，不修改 Agent 模型（需要真实的 Agent 等级字段支持）
        await AgentOSService._write_operation_log(
            db, user_id, f"agent:{agent_id}", "autonomy_upgrade",
            previous_status=None, new_status=target_level,
            comment=f"自治等级升级至 {target_level}",
        )
        return {"ok": True, "agent_id": agent_id, "new_level": target_level}

    @staticmethod
    async def execute_downgrade(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        target_level: str,
    ) -> dict[str, Any]:
        """执行自治等级降级"""
        await AgentOSService._write_operation_log(
            db, user_id, f"agent:{agent_id}", "autonomy_downgrade",
            previous_status=None, new_status=target_level,
            comment=f"自治等级降级至 {target_level}",
        )
        return {"ok": True, "agent_id": agent_id, "new_level": target_level}
```

- [ ] **Step 2: Add router endpoints**

In `backend/app/agentos/router.py`, after the reject endpoint, add:

```python
# ── Phase 3: Autonomy Upgrade ────────────────────────────


@router.get("/agentos/agents/upgrade-candidates", summary="自治等级升级候选")
async def list_upgrade_candidates(
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回建议升级/降级的 Agent 候选列表"""
    result = await AgentOSService.get_upgrade_candidates(db, current_user.id)
    return Result.ok(result)


@router.post("/agentos/agents/{agent_id}/upgrade", summary="执行自治等级升级")
async def upgrade_agent_level(
    agent_id: str,
    target_level: str,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    """将 Agent 升级到目标自治等级"""
    result = await AgentOSService.execute_upgrade(
        db, current_user.id, agent_id, target_level,
    )
    return Result.ok(result)


@router.post("/agentos/agents/{agent_id}/downgrade", summary="执行自治等级降级")
async def downgrade_agent_level(
    agent_id: str,
    target_level: str,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:approve")),
):
    """将 Agent 降级到目标自治等级"""
    result = await AgentOSService.execute_downgrade(
        db, current_user.id, agent_id, target_level,
    )
    return Result.ok(result)
```

- [ ] **Step 3: Run all tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py -q
```

Expected: 35+ passed (30 Phase 1+2 + 5 Phase 3 upgrade rule tests).

- [ ] **Step 4: Commit Task 5**

```bash
git add backend/app/agentos/service.py backend/app/agentos/router.py
git commit -m "feat(agentos): add autonomy upgrade API"
```

---

## Task 6: Frontend API + Routes

**Files:**
- Modify: `frontend/src/api/modules/agentos.ts`
- Modify: `frontend/src/router/modules/agentos.ts`

- [ ] **Step 1: Add API methods**

In `frontend/src/api/modules/agentos.ts`, after the Phase 2 API section, add:

```typescript
// ── Phase 3: Operation Log ───────────────────────────────

export interface AgentOSOperationLog {
  id: number
  user_id: number
  item_id: string
  action: string
  source_type: string | null
  previous_status: string | null
  new_status: string | null
  comment: string | null
  created_at: string | null
}

export interface AutonomyCandidate {
  agent_id: string
  agent_name: string
  squad_id: string
  squad_name: string
  current_level: string
  suggested: boolean
  direction: string | null
  target_level: string | null
  confidence: number
  reason: string
}

export function getAgentOSOperations(params?: {
  item_id?: string
  action?: string
  source_type?: string
  limit?: number
  offset?: number
}) {
  return http.get('/agentos/operations', { params })
}

export function getAgentOSUpgradeCandidates() {
  return http.get('/agentos/agents/upgrade-candidates')
}

export function upgradeAgentLevel(agentId: string, targetLevel: string) {
  return http.post(`/agentos/agents/${agentId}/upgrade`, null, {
    params: { target_level: targetLevel },
  })
}

export function downgradeAgentLevel(agentId: string, targetLevel: string) {
  return http.post(`/agentos/agents/${agentId}/downgrade`, null, {
    params: { target_level: targetLevel },
  })
}
```

Update the `agentosApi` export object:

```typescript
export const agentosApi = {
  getControlCenter: getAgentOSControlCenter,
  getWorkItems: getAgentOSWorkItems,
  getSquads: getAgentOSSquads,
  getTemplates: getAgentOSTemplates,
  updateWorkItemStatus,
  approveWorkItem,
  rejectWorkItem,
  getAgentOSOperations,
  getAgentOSUpgradeCandidates,
  upgradeAgentLevel,
  downgradeAgentLevel,
}
```

- [ ] **Step 2: Add route**

In `frontend/src/router/modules/agentos.ts`, add a new child route:

```typescript
{
  path: 'autonomy',
  name: 'AgentOSAutonomy',
  component: () => import('@/views/agentos/AutonomyManagement.vue'),
  meta: {
    title: '自治管理',
    icon: 'shield',
    menu: true,
    perm: 'agentos:view',
  },
},
```

- [ ] **Step 3: Check icon availability**

Open `frontend/src/components/Layout.vue` and verify `shield` is in the `iconMap`. If not, add:

```typescript
import { ShieldOutline } from '@vicons/ionicons5'
// ... in iconMap
shield: ShieldOutline,
```

- [ ] **Step 4: Build frontend**

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 5: Commit Task 6**

```bash
git add frontend/src/api/modules/agentos.ts frontend/src/router/modules/agentos.ts frontend/src/components/Layout.vue
git commit -m "feat(agentos): add autonomy management API and route"
```

---

## Task 7: Autonomy Management Page

**Files:**
- Create: `frontend/src/views/agentos/AutonomyManagement.vue`

- [ ] **Step 1: Create the autonomy management page**

Create `frontend/src/views/agentos/AutonomyManagement.vue`:

```vue
<template>
  <div>
    <n-page-header subtitle="管理 Agent 自治等级升级与降级建议">
      <template #title>自治管理</template>
      <template #extra>
        <n-button size="small" @click="fetchCandidates" :loading="loading">刷新</n-button>
      </template>
    </n-page-header>

    <n-spin v-if="loading && !loaded" :show="true" style="margin-top: 40px;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </n-spin>

    <n-result v-else-if="error" status="error" title="加载失败" :description="error">
      <template #footer>
        <n-button @click="fetchCandidates">重试</n-button>
      </template>
    </n-result>

    <template v-else>
      <!-- 建议升级卡片 -->
      <n-card title="升级建议" size="small" style="margin-top: 12px;">
        <template v-if="upgradeCandidates.length === 0">
          <n-empty description="暂无升级建议" style="padding: 20px 0;" />
        </template>
        <n-grid :cols="3" :x-gap="12" :y-gap="12">
          <n-grid-item v-for="c in upgradeCandidates" :key="c.agent_id">
            <AutonomyUpgradeCard
              :candidate="c"
              :actioning="actioning === c.agent_id"
              @upgrade="handleUpgrade"
              @downgrade="handleDowngrade"
            />
          </n-grid-item>
        </n-grid>
      </n-card>

      <!-- 自治等级说明 -->
      <n-card title="自治等级体系" size="small" style="margin-top: 12px;">
        <n-grid :cols="4" :x-gap="12" :y-gap="8">
          <n-grid-item><n-alert type="default" :bordered="false"><template #header>L0 观察</template>只读数据，无建议</n-alert></n-grid-item>
          <n-grid-item><n-alert type="info" :bordered="false"><template #header>L1 建议</template>生成建议，人执行</n-alert></n-grid-item>
          <n-grid-item><n-alert type="success" :bordered="false"><template #header>L2 半自主</template>低风险自动，高风险审批</n-alert></n-grid-item>
          <n-grid-item><n-alert type="warning" :bordered="false"><template #header>L3 全自主</template>边界内自动执行</n-alert></n-grid-item>
        </n-grid>
      </n-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { getAgentOSUpgradeCandidates, upgradeAgentLevel, downgradeAgentLevel } from '@/api/modules/agentos'
import type { AutonomyCandidate } from '@/api/modules/agentos'
import AutonomyUpgradeCard from '@/components/agentos/AutonomyUpgradeCard.vue'

const message = useMessage()

const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)
const actioning = ref<string | null>(null)
const candidates = ref<AutonomyCandidate[]>([])

const upgradeCandidates = computed(() =>
  candidates.value.filter(c => c.suggested)
)

async function fetchCandidates() {
  loading.value = true
  error.value = null
  try {
    const res: any = await getAgentOSUpgradeCandidates()
    candidates.value = res?.data || []
    loaded.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || '加载失败'
    message.error('加载升级候选失败')
  } finally {
    loading.value = false
  }
}

async function handleUpgrade(candidate: AutonomyCandidate) {
  if (!candidate.target_level) return
  actioning.value = candidate.agent_id
  try {
    await upgradeAgentLevel(candidate.agent_id, candidate.target_level)
    message.success(`${candidate.agent_name} 已升级至 ${candidate.target_level}`)
    await fetchCandidates()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '升级失败')
  } finally {
    actioning.value = null
  }
}

async function handleDowngrade(candidate: AutonomyCandidate) {
  if (!candidate.target_level) return
  actioning.value = candidate.agent_id
  try {
    await downgradeAgentLevel(candidate.agent_id, candidate.target_level)
    message.success(`${candidate.agent_name} 已降级至 ${candidate.target_level}`)
    await fetchCandidates()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '降级失败')
  } finally {
    actioning.value = null
  }
}

onMounted(fetchCandidates)
</script>
```

- [ ] **Step 2: Build frontend to verify**

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 3: Commit Task 7**

```bash
git add frontend/src/views/agentos/AutonomyManagement.vue
git commit -m "feat(agentos): add autonomy management page"
```

---

## Task 8: AutonomyUpgradeCard Component

**Files:**
- Create: `frontend/src/components/agentos/AutonomyUpgradeCard.vue`

- [ ] **Step 1: Create component**

Create `frontend/src/components/agentos/AutonomyUpgradeCard.vue`:

```vue
<template>
  <n-card size="small" hoverable>
    <template #header>
      <n-space align="center" justify="space-between">
        <n-space align="center" size="small">
          <n-avatar :size="24" round>{{ candidate.agent_name.charAt(0) }}</n-avatar>
          <div>
            <div style="font-weight: 600; font-size: 14px;">{{ candidate.agent_name }}</div>
            <div style="color: #888; font-size: 12px;">{{ candidate.squad_name }}</div>
          </div>
        </n-space>
        <n-tag :type="directionType" size="small" :bordered="false">
          {{ directionLabel }}
        </n-tag>
      </n-space>
    </template>

    <n-space vertical size="small">
      <n-space justify="space-between">
        <span style="font-size: 13px; color: #666;">当前等级</span>
        <n-tag size="small">{{ currentLevelLabel }}</n-tag>
      </n-space>
      <n-space justify="space-between">
        <span style="font-size: 13px; color: #666;">目标等级</span>
        <n-tag size="small" :type="directionType">{{ targetLevelLabel }}</n-tag>
      </n-space>
      <n-space justify="space-between">
        <span style="font-size: 13px; color: #666;">置信度</span>
        <span style="font-weight: 600;">{{ (candidate.confidence * 100).toFixed(0) }}%</span>
      </n-space>
      <div style="font-size: 12px; color: #999; padding: 4px 0;">
        {{ candidate.reason || '-' }}
      </div>
    </n-space>

    <template #footer>
      <n-space justify="space-between">
        <n-button
          v-if="candidate.direction === 'upgrade'"
          size="tiny"
          type="primary"
          :loading="actioning && candidate.agent_id === actioning"
          @click="$emit('upgrade', candidate)"
        >执行升级</n-button>
        <n-button
          v-if="candidate.direction === 'downgrade'"
          size="tiny"
          type="warning"
          :loading="actioning && candidate.agent_id === actioning"
          @click="$emit('downgrade', candidate)"
        >执行降级</n-button>
        <span v-else style="color: #ccc; font-size: 12px;">无需变更</span>
      </n-space>
    </template>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AutonomyCandidate } from '@/api/modules/agentos'

const props = defineProps<{
  candidate: AutonomyCandidate
  actioning?: boolean
}>()

defineEmits<{
  upgrade: [candidate: AutonomyCandidate]
  downgrade: [candidate: AutonomyCandidate]
}>()

const levelLabels: Record<string, string> = {
  OBSERVATION: 'L0 观察',
  SUGGESTION: 'L1 建议',
  SEMI_AUTONOMOUS: 'L2 半自主',
  FULL_AUTONOMOUS: 'L3 全自主',
}

const currentLevelLabel = computed(() => levelLabels[props.candidate.current_level] || props.candidate.current_level)
const targetLevelLabel = computed(() => levelLabels[props.candidate.target_level || ''] || props.candidate.target_level || '-')

const directionType = computed(() => {
  if (props.candidate.direction === 'upgrade') return 'success'
  if (props.candidate.direction === 'downgrade') return 'warning'
  return 'default'
})

const directionLabel = computed(() => {
  if (props.candidate.direction === 'upgrade') return '建议升级'
  if (props.candidate.direction === 'downgrade') return '建议降级'
  return '稳定'
})
</script>
```

- [ ] **Step 2: Build frontend**

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 3: Commit Task 8**

```bash
git add frontend/src/components/agentos/AutonomyUpgradeCard.vue
git commit -m "feat(agentos): add autonomy upgrade card component"
```

---

## Task 9: Add Upgrade Candidate Panel to Squads Page

**Files:**
- Modify: `frontend/src/views/agentos/Squads.vue`

- [ ] **Step 1: Add candidate section to Squads.vue**

At the top of the `<script setup>` section, add imports:

```typescript
import { getAgentOSUpgradeCandidates } from '@/api/modules/agentos'
import type { AutonomyCandidate } from '@/api/modules/agentos'
```

After `const squads` definition, add:

```typescript
const upgradeCandidates = ref<AutonomyCandidate[]>([])
```

In `fetchSquads()`, after the main fetch, add:

```typescript
try {
  const candRes: any = await getAgentOSUpgradeCandidates()
  upgradeCandidates.value = (candRes?.data || []).filter((c: AutonomyCandidate) => c.suggested)
} catch (_e) {
  // 静默降级
}
```

After the squad grid and before the autonomy level explanation card, add:

```vue
<template v-if="upgradeCandidates.length > 0">
  <n-card title="自治等级升级建议" size="small" style="margin-top: 12px;">
    <n-space>
      <n-tag
        v-for="c in upgradeCandidates"
        :key="c.agent_id"
        :type="c.direction === 'upgrade' ? 'success' : 'warning'"
        style="cursor: pointer;"
        @click="router.push('/agentos/autonomy')"
      >
        {{ c.agent_name }} → {{ c.target_level }}
      </n-tag>
    </n-space>
  </n-card>
</template>
```

- [ ] **Step 2: Build frontend**

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 3: Commit Task 9**

```bash
git add frontend/src/views/agentos/Squads.vue
git commit -m "feat(agentos): add upgrade candidate panel to squads page"
```

---

## Task 10: Full Verification

**Files:**
- No new files

- [ ] **Step 1: Run all backend tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py -q
```

Expected: all Phase 3 tests pass (total > 35).

- [ ] **Step 2: Run full backend suite**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest -q
```

Expected: same pre-existing 4 failures only (test_agent_evolution.py).

- [ ] **Step 3: Run frontend build**

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 4: Check git status**

```bash
git status --short
```

Expected: only intended AgentOS files changed.

- [ ] **Step 5: Final commit if changes**

```bash
git add -A
git commit -m "feat(agentos): complete phase 3 autonomy management"
```

---

## Self-Review Checklist

- **Spec coverage:**
  - Operation log table + migration: Task 1
  - Operation log auto-write: Task 2
  - Operation log query API: Task 3
  - Autonomy upgrade rules engine: Task 4
  - Upgrade/downgrade API: Task 5
  - Frontend API + route: Task 6
  - Autonomy management page: Task 7
  - Upgrade card component: Task 8
  - Upgrade panel on squads page: Task 9
  - Full verification: Task 10

- **Placeholder scan:** All steps contain complete code, exact file paths, and exact commands.

- **Type consistency:**
  - Backend: `AgentOSOperationLogVO`, `AutonomyCandidateVO`, `autonomy_service.suggest_upgrade()`
  - Frontend: `AgentOSOperationLog`, `AutonomyCandidate`, `getAgentOSOperations()`, `getAgentOSUpgradeCandidates()`
  - API paths: `/operations`, `/agents/upgrade-candidates`, `/agents/{id}/upgrade`, `/agents/{id}/downgrade`
