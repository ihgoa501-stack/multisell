# Project Stabilization And MVP Handoff Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current LingMirror / MultiSell workspace into a clean, verified baseline, then prepare the next MVP slice: pre-listing business decision support.

**Architecture:** Phase 1 is stabilization only: no new product behavior unless required to make existing tests and build pass. Phase 2 starts only after Phase 1 is green and adds a narrow decision workflow that reuses existing product, SKU, shipping, order profit, RBAC, and audit patterns.

**Tech Stack:** Python 3.11+, FastAPI, SQLAlchemy 2.0 async, Alembic, PostgreSQL, pytest, Vue 3, TypeScript, Vite, Naive UI.

---

## Current Situation

The repository is not a clean `main` snapshot. `git status --short --branch` currently shows many modified and untracked files across backend, frontend, docs, Alembic migrations, tests, and deleted example modules.

The codebase already contains substantial new work:

- Product logistics fields and logistics completeness display.
- Shipping providers, channels, zones, quote rules, rate import, and calculator.
- Order inventory locking and payment-time stock deduction.
- Order shipping snapshot and first-pass profit calculation.
- RBAC and operation-log coverage across most business modules.
- LingMirror branding and frontend route/module additions.

Do not start another feature before stabilizing this baseline.

## Scope

In scope for this handoff:

- Verify and fix the current backend and frontend baseline.
- Update stale documentation that contradicts implemented behavior.
- Organize current changes into reviewable commits or a reviewable branch.
- Define the next MVP slice around pre-listing decision support.

Out of scope for this handoff:

- Real Ozon, Shopee, Wildberries, AliExpress, or Temu API integration.
- Full after-sales workflow.
- Multi-warehouse allocation.
- Carrier label purchase, tracking, and reconciliation.
- Large UI redesign unrelated to the MVP slice.

## Phase 1 Files

Stabilization touches these files only if validation proves they need changes:

- `docs/PROJECT_STATUS.md` - remove stale status contradictions and align completed / pending lists.
- `docs/ROADMAP.md` - replace outdated recommended next step with the new baseline-first sequence.
- `docs/DEVELOPMENT_GUIDE.md` - update commands if they no longer match actual startup or test behavior.
- `docs/PERMISSIONS_AND_AUDIT.md` - align permission tables with current implemented modules.
- `backend/tests/*` - update only tests whose assertions are stale against intentional implemented behavior.
- `backend/app/*` - fix only real failing behavior found by tests.
- `frontend/src/*` - fix only typecheck/build/runtime errors found by `npm run build`.

## Phase 2 Candidate Files

Pre-listing decision support should be added as a narrow module, not mixed into existing shipping or order modules.

- Create: `backend/app/decision/__init__.py`
- Create: `backend/app/decision/router.py`
- Create: `backend/app/decision/schemas.py`
- Create: `backend/app/decision/service.py`
- Create: `backend/tests/test_prelisting_decision.py`
- Modify: `backend/app/models.py` only if persistence is required. Prefer stateless calculation first.
- Modify: `backend/seed.py` only if demo data is needed for manual UI verification.
- Create: `frontend/src/api/modules/decision.ts`
- Create: `frontend/src/router/modules/decision.ts`
- Create: `frontend/src/views/decision/PreListingDecision.vue`
- Modify: `frontend/src/components/Layout.vue` only if menu grouping cannot be handled by route metadata.
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`

## Phase 1: Stabilize Current Baseline

### Task 1: Record The Starting Point

**Files:**
- Read: repository root
- Modify: none

- [ ] **Step 1: Capture branch and file state**

Run:

```bash
git status --short --branch
git diff --stat
git log --oneline --decorate -5
```

Expected:

- Branch is `main` unless the user has moved the work.
- Working tree contains many existing changes.
- No command modifies files.

- [ ] **Step 2: Identify unrelated user work**

Run:

```bash
git diff --name-only
git ls-files --others --exclude-standard
```

Expected:

- Make a short note in the implementation summary grouping files by domain: backend, frontend, docs, migrations, tests.
- Do not revert or delete untracked files.

### Task 2: Verify Backend Baseline

**Files:**
- Read: `backend/pytest.ini`
- Read: `backend/tests/conftest.py`
- Modify only if tests fail: affected `backend/app/**` and affected `backend/tests/**`

- [ ] **Step 1: Start PostgreSQL**

Run:

```bash
docker compose up -d db
```

Expected:

- PostgreSQL container is healthy.
- Test database `product_management_test` exists through `backend/scripts/init-db.sql`.

- [ ] **Step 2: Install backend dependencies if needed**

Run:

```bash
cd backend
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

Expected:

- Requirements install without dependency resolution errors.

- [ ] **Step 3: Run full backend tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest -q
```

Expected:

- Target result is all tests passing, with the known xfail preserved if still present.
- If tests fail, fix the smallest behavior or stale assertion needed, then rerun the same command.

- [ ] **Step 4: Commit backend stabilization**

Only commit after the full backend command is green.

Run:

```bash
git status --short
git add backend docs
git commit -m "chore: stabilize backend baseline"
```

Expected:

- Commit contains only backend and docs changes required by backend validation.
- If frontend files are mixed in, stop and split the commit.

### Task 3: Verify Frontend Baseline

**Files:**
- Read: `frontend/package.json`
- Modify only if build fails: affected `frontend/src/**`, `frontend/index.html`, `frontend/vite.config.ts`, or generated type files

- [ ] **Step 1: Install frontend dependencies if needed**

Run:

```bash
cd frontend
npm install
```

Expected:

- `package-lock.json` remains consistent with `package.json`.

- [ ] **Step 2: Run production build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Vite build completes successfully.
- Vue type errors, missing module imports, and route import failures are fixed before moving on.

- [ ] **Step 3: Commit frontend stabilization**

Run:

```bash
git status --short
git add frontend
git commit -m "chore: stabilize frontend baseline"
```

Expected:

- Commit contains only frontend files and generated frontend type files.

### Task 4: Reconcile Project Documents

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`
- Modify: `docs/DEVELOPMENT_GUIDE.md`
- Modify: `docs/PERMISSIONS_AND_AUDIT.md`
- Modify: `README.md` only if startup, test, or product-state descriptions are stale

- [ ] **Step 1: Fix the stale order inventory limitation**

In `docs/PROJECT_STATUS.md`, remove or rewrite the section that says order inventory deduction is not implemented.

Required wording:

```markdown
### 订单库存闭环

状态：已完成第一版。

已实现：
- 创建订单锁定库存。
- `pending -> paid` 扣减实物库存并释放锁定库存。
- `pending -> cancelled` 释放锁定库存。
- 库存不足阻止创建订单。
- 库存变动写入库存日志。

仍未实现：
- `paid -> cancelled` 自动退库存。
- 售后退货入库。
- 多仓库分配。
- 并发压力测试。
```

- [ ] **Step 2: Update recommended next step**

In `docs/ROADMAP.md`, replace any recommendation that says the next task is "完成订单库存扣减闭环".

Required recommendation:

````markdown
## 推荐下一步任务

最推荐继续做：

```text
先完成仓库收口和全量验证，再进入上架前经营决策闭环。
```

原因：

- 当前工作区有大量未提交改动，需要先形成可信基线。
- 订单库存闭环、物流运费、权限审计等模块已经具备第一版能力。
- 下一阶段产品价值应聚焦商品上架前利润核算和是否建议上架。
````

- [ ] **Step 3: Align validation results**

In `docs/PROJECT_STATUS.md`, update "当前验证结果" with the exact command results from Task 2 and Task 3.

Required format:

````markdown
## 当前验证结果

最近一次验证（YYYY-MM-DD）：

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
cd frontend && npm run build
```

结果：后端填写 pytest 最后一行摘要，例如 `163 passed, 1 xfailed in 12.34s`；前端填写 `npm run build` 的成功摘要，例如 `built in 3.21s`。
````

- [ ] **Step 4: Commit documentation reconciliation**

Run:

```bash
git add README.md docs/PROJECT_STATUS.md docs/ROADMAP.md docs/DEVELOPMENT_GUIDE.md docs/PERMISSIONS_AND_AUDIT.md
git commit -m "docs: reconcile project status and roadmap"
```

Expected:

- Documentation no longer contradicts implemented order inventory behavior.
- Validation results match actual commands run by the worker.

### Task 5: Produce Reviewable Baseline Summary

**Files:**
- Create or update: `docs/PROJECT_STATUS.md`
- Modify: none elsewhere unless required by stale references

- [ ] **Step 1: Generate final status evidence**

Run:

```bash
git status --short --branch
git log --oneline --decorate -5
```

Expected:

- Working tree is either clean or contains only intentionally uncommitted files called out in the handoff summary.

- [ ] **Step 2: Write implementation summary**

The worker's final handoff must include:

```markdown
Backend verification:
- Command:
- Result:

Frontend verification:
- Command:
- Result:

Commits created:
- Copy the actual `git log --oneline` lines for commits created during this handoff.

Remaining uncommitted files:
- List each remaining file path and a one-line reason, or write `None`.

Known product limitations:
- Real platform adapters are not implemented.
- `paid -> cancelled` stock restoration is deferred to after-sales.
- Carrier label, tracking, and freight reconciliation are not implemented.
- Excel bulk import still needs production-grade row-level feedback for SKU / price / inventory.
```

## Phase 2: Next MVP Slice - Pre-Listing Business Decision

Start this phase only after Phase 1 is green and committed.

### Task 6: Add Backend Decision API Contract

**Files:**
- Create: `backend/app/decision/__init__.py`
- Create: `backend/app/decision/schemas.py`
- Create: `backend/app/decision/router.py`
- Create: `backend/app/decision/service.py`
- Create: `backend/tests/test_prelisting_decision.py`

- [ ] **Step 1: Write failing API tests**

Create `backend/tests/test_prelisting_decision.py` with tests for:

```python
async def test_prelisting_decision_recommends_approve_when_margin_is_high(async_client):
    ...
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data["recommendation"] == "approve"
    assert data["profit_margin"] >= 20
    assert data["blocking_reasons"] == []


async def test_prelisting_decision_blocks_when_shipping_data_missing(async_client):
    ...
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data["recommendation"] == "needs_data"
    assert "缺少物流数据" in data["blocking_reasons"][0]


async def test_prelisting_decision_rejects_when_profit_margin_is_too_low(async_client):
    ...
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data["recommendation"] == "reject"
    assert data["profit_margin"] < 10
```

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision.py -q
```

Expected:

- Tests fail because `/api/decisions/prelisting` does not exist.

- [ ] **Step 2: Implement schemas**

Create `backend/app/decision/schemas.py` with request and response models:

```python
from pydantic import BaseModel, Field


class PreListingDecisionRequest(BaseModel):
    sku_id: int
    destination_country: str = Field(min_length=2, max_length=10)
    target_sale_price: float = Field(gt=0)
    platform_fee_pct: float = Field(default=0, ge=0, le=100)
    payment_fee_pct: float = Field(default=0, ge=0, le=100)
    other_fee: float = Field(default=0, ge=0)
    minimum_margin_pct: float = Field(default=20, ge=0, le=100)
    cargo_type: str = "normal"


class PreListingDecisionResponse(BaseModel):
    sku_id: int
    destination_country: str
    target_sale_price: float
    product_cost: float
    shipping_fee: float
    platform_fee: float
    payment_fee: float
    other_fee: float
    profit_amount: float
    profit_margin: float
    recommendation: str
    blocking_reasons: list[str]
    warnings: list[str]
```

- [ ] **Step 3: Implement service using existing shipping calculator**

Create `backend/app/decision/service.py`.

Business rule:

- If SKU or product does not exist, raise `ValueError("SKU不存在")`.
- If shipping calculation returns no result because logistics data is incomplete or no channel matches, return `recommendation="needs_data"` with a blocking reason.
- Use the cheapest shipping quote from `CalculateService.calculate`.
- Profit formula: `target_sale_price - product_cost - shipping_fee - platform_fee - payment_fee - other_fee`.
- `approve` when `profit_margin >= minimum_margin_pct`.
- `reject` when data is complete but `profit_margin < minimum_margin_pct`.

- [ ] **Step 4: Implement router**

Create `backend/app/decision/router.py`:

```python
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.common.schemas import Result
from app.database import get_db
from app.decision.schemas import PreListingDecisionRequest
from app.decision.service import PreListingDecisionService
from app.models import User

router = APIRouter(prefix="/decisions", tags=["上架决策"])


@router.post("/prelisting", summary="上架前经营决策")
async def prelisting_decision(
    data: PreListingDecisionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("decision:calculate")),
):
    result = await PreListingDecisionService.calculate(db, data)
    return Result.ok(result)
```

- [ ] **Step 5: Export router**

Create `backend/app/decision/__init__.py`:

```python
from .router import router
```

- [ ] **Step 6: Run targeted tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision.py -q
```

Expected:

- New decision tests pass.

- [ ] **Step 7: Run full backend tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest -q
```

Expected:

- Full backend suite passes.

- [ ] **Step 8: Commit backend decision API**

Run:

```bash
git add backend/app/decision backend/tests/test_prelisting_decision.py
git commit -m "feat: add prelisting decision api"
```

### Task 7: Add Frontend Decision Page

**Files:**
- Create: `frontend/src/api/modules/decision.ts`
- Create: `frontend/src/router/modules/decision.ts`
- Create: `frontend/src/views/decision/PreListingDecision.vue`

- [ ] **Step 1: Add API module**

Create `frontend/src/api/modules/decision.ts`:

```ts
import http from '@/api/http'

export interface PreListingDecisionRequest {
  sku_id: number
  destination_country: string
  target_sale_price: number
  platform_fee_pct: number
  payment_fee_pct: number
  other_fee: number
  minimum_margin_pct: number
  cargo_type: string
}

export function calculatePreListingDecision(data: PreListingDecisionRequest) {
  return http.post('/decisions/prelisting', data)
}
```

- [ ] **Step 2: Add route module**

Create `frontend/src/router/modules/decision.ts`:

```ts
import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'decisions/prelisting',
    name: 'PreListingDecision',
    component: () => import('@/views/decision/PreListingDecision.vue'),
    meta: { title: '上架决策', icon: 'analytics', menu: true, perm: 'decision:calculate' },
  },
]
```

- [ ] **Step 3: Add page**

Create `frontend/src/views/decision/PreListingDecision.vue` with:

- SKU ID input.
- Destination country input.
- Target sale price input.
- Platform fee percent input.
- Payment fee percent input.
- Other fee input.
- Minimum margin percent input.
- Cargo type select.
- Calculate button.
- Result panel showing recommendation, profit amount, profit margin, shipping fee, blocking reasons, and warnings.

- [ ] **Step 4: Build frontend**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 5: Commit frontend decision page**

Run:

```bash
git add frontend/src/api/modules/decision.ts frontend/src/router/modules/decision.ts frontend/src/views/decision/PreListingDecision.vue
git commit -m "feat: add prelisting decision page"
```

### Task 8: Add Permission Seed And Docs For Decision Module

**Files:**
- Modify: `backend/seed.py`
- Modify: `docs/PERMISSIONS_AND_AUDIT.md`
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Add permission code**

In `backend/seed.py`, add:

```python
("上架决策计算", "decision:calculate", "decision"),
```

to the same permission seed structure that contains `shipping:calculate`, `order:view`, and `report:view`.

- [ ] **Step 2: Update permissions guide**

In `docs/PERMISSIONS_AND_AUDIT.md`, add:

```markdown
| 上架决策 | `decision:calculate` | 无写操作 |
```

- [ ] **Step 3: Update project status**

In `docs/PROJECT_STATUS.md`, add a section:

```markdown
### 上架前经营决策

状态：已完成第一版。

已实现：
- 根据 SKU、目的国、目标售价、平台费、支付费、其他费用测算利润。
- 复用现有运费计算结果选择最低可用报价。
- 利润率达到阈值时建议上架。
- 物流数据缺失或无可用渠道时返回需补数据。

仍未实现：
- 平台类目佣金规则库。
- 多平台批量比较。
- AI 自动生成上架建议说明。
- 人工审批流。
```

- [ ] **Step 4: Verify all**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest -q
cd ../frontend
npm run build
```

Expected:

- Backend tests pass.
- Frontend build passes.

- [ ] **Step 5: Commit seed and docs**

Run:

```bash
git add backend/seed.py docs/PERMISSIONS_AND_AUDIT.md docs/PROJECT_STATUS.md docs/ROADMAP.md
git commit -m "docs: document prelisting decision module"
```

## Execution Rules For Assigned Workers

- Work in a branch named with the `codex/` prefix unless the owner requests a different branch.
- Do not use `git reset --hard` or revert user changes.
- Do not mix stabilization commits and new feature commits.
- Run targeted tests after each task.
- Run full backend tests and frontend build before handoff.
- Update docs in the same task that changes product behavior.
- Stop and ask if test failures imply deleting or rewriting unrelated existing functionality.

## Final Acceptance Criteria

Phase 1 is complete when:

- Backend full test command passes.
- Frontend `npm run build` passes.
- Project docs no longer contradict implemented behavior.
- Current work is organized into reviewable commits or an explicit list of remaining uncommitted files.

Phase 2 is complete when:

- `/api/decisions/prelisting` returns `approve`, `reject`, or `needs_data` deterministically.
- The frontend has a usable pre-listing decision page.
- `decision:calculate` permission is seeded and documented.
- Backend full tests pass.
- Frontend build passes.

## Recommended Handoff Prompt

Use this prompt when assigning the work:

```text
你接手的是 /Users/lc/multisell 的 LingMirror / MultiSell 项目。

先阅读：
- docs/PROJECT_STATUS.md
- docs/ROADMAP.md
- docs/DEVELOPMENT_GUIDE.md
- docs/PERMISSIONS_AND_AUDIT.md
- docs/superpowers/plans/2026-06-15-project-stabilization-and-mvp-handoff.md

第一优先级：执行计划里的 Phase 1，先让当前仓库成为可验证、可提交、文档一致的基线。

不要先加新功能。不要 reset 或删除未跟踪文件。跑：
- backend 全量 pytest
- frontend npm run build

Phase 1 通过并提交后，再开始 Phase 2 的“上架前经营决策”模块。
```
