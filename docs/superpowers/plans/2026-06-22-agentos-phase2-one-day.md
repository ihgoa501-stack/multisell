# AgentOS Phase 2 One-Day Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the AgentOS Phase 2 operator loop in one day: create proposal, approve/reject, execute, review, refresh WorkItems, and document the completed scope.

**Architecture:** Keep the existing backend Action Center as the source of truth. Add only the missing frontend controls and narrow tests around the user-facing loop; do not add new database tables or new business domains. WorkItem remains the aggregation model, while ActionProposal owns executable command lifecycle.

**Tech Stack:** FastAPI, SQLAlchemy async, PostgreSQL test database, Pytest, Vue 3, TypeScript, Naive UI, Vite.

---

## One-Day Scope

### Must Finish Today

- Existing backend Action Center endpoints stay intact:
  - `POST /api/agentos/action-proposals`
  - `POST /api/agentos/action-proposals/{proposal_id}/approve`
  - `POST /api/agentos/action-proposals/{proposal_id}/reject`
  - `POST /api/agentos/action-proposals/{proposal_id}/execute`
  - `POST /api/agentos/action-proposals/{proposal_id}/review`
  - `PATCH /api/agentos/work-items/{item_id}/status`
  - `POST /api/agentos/work-items/{item_id}/approve`
  - `POST /api/agentos/work-items/{item_id}/reject`
- Frontend task cards support the full action proposal lifecycle:
  - Pending proposal: approve/reject.
  - Approved or auto-suggested proposal: execute.
  - Executed proposal: review.
  - Non-approval basic WorkItem: mark complete.
- Task list refreshes after each mutation.
- Focused backend and frontend checks pass.
- `docs/TIMELINE.md` reflects Phase 2 completion status.

### Explicitly Out Of Scope Today

- Real Ozon/Shopee/Wildberries production API publishing.
- New database schema unless tests reveal an existing migration gap.
- New Agent autonomy upgrade policy.
- Full visual redesign of AgentOS.
- General dashboard/reporting expansion.

## Current Code Map

- Modify: `frontend/src/components/agentos/WorkItemCard.vue`
  - Owns per-task actions. Add execute/review buttons and review modal for `source_type === "action_proposal"`.
- Modify: `frontend/src/views/agentos/WorkItems.vue`
  - Owns list loading, filtering, create proposal modal, and refresh callbacks. Keep it as the page coordinator.
- Modify: `frontend/src/api/modules/agentos.ts`
  - Existing functions already include `executeActionProposal()` and `reviewActionProposal()`. Only adjust types if the UI needs stricter metadata helpers.
- Test/verify: `backend/tests/test_agentos_action_center.py`
  - Already covers create, approve, reject, execute, review, and persistence.
- Test/verify: `backend/tests/test_agentos_phase1.py`
  - Already covers WorkItem mutation endpoints.
- Modify: `docs/TIMELINE.md`
  - Add a 2026-06-22 Phase 2 completion entry and move current P0 status forward.
- Create or modify only if frontend test tooling already exists: `frontend/src/components/agentos/WorkItemCard.spec.ts`
  - Add a minimal component test for button visibility and emitted refresh. If no frontend test harness exists, skip component test and rely on `npm run build` plus manual browser verification.

## Day Schedule

- 09:00-09:30: Baseline checks.
- 09:30-11:00: Frontend task card execute/review UI.
- 11:00-12:00: Task list refresh and edge state cleanup.
- 13:00-14:30: Backend focused tests and fix regressions only.
- 14:30-16:00: Frontend build and manual browser verification.
- 16:00-17:00: Documentation and final smoke test.
- 17:00-18:00: Buffer for one blocking issue.

## Task 1: Establish Baseline

**Files:**
- Read only: `backend/tests/test_agentos_action_center.py`
- Read only: `backend/tests/test_agentos_phase1.py`
- Read only: `frontend/src/components/agentos/WorkItemCard.vue`
- Read only: `frontend/src/views/agentos/WorkItems.vue`

- [ ] **Step 1: Confirm clean working tree**

Run:

```bash
git status --short
```

Expected: no output, or only intentional local files from this plan.

- [ ] **Step 2: Run focused backend AgentOS tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py tests/test_agentos_phase1.py -q
```

Expected: all tests pass. If they fail because the database is not running, start DB:

```bash
docker compose up -d db
```

Then rerun the same pytest command.

- [ ] **Step 3: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: TypeScript and Vite build complete without errors.

- [ ] **Step 4: Commit only if baseline required fixes**

If no fixes were made, do not commit. If baseline fixes were required:

```bash
git add backend frontend
git commit -m "fix: restore agentos phase2 baseline"
```

## Task 2: Add ActionProposal Execute And Review Controls

**Files:**
- Modify: `frontend/src/components/agentos/WorkItemCard.vue`
- Uses existing API from: `frontend/src/api/modules/agentos.ts`

- [ ] **Step 1: Import execute and review API functions**

In `frontend/src/components/agentos/WorkItemCard.vue`, change the API import from:

```ts
import { updateWorkItemStatus, approveWorkItem, rejectWorkItem } from '@/api/modules/agentos'
```

to:

```ts
import {
  updateWorkItemStatus,
  approveWorkItem,
  rejectWorkItem,
  executeActionProposal,
  reviewActionProposal,
} from '@/api/modules/agentos'
```

- [ ] **Step 2: Add proposal ID and status computed helpers**

Add these computed values below `const mutating = ref(false)`:

```ts
const proposalId = computed(() => {
  if (props.item.source_type !== 'action_proposal') return null
  const raw = props.item.source_id || props.item.id.replace('action_proposal:', '')
  const id = Number(raw)
  return Number.isFinite(id) ? id : null
})

const proposalStatus = computed(() => {
  return String(props.item.metadata?.proposal_status || '')
})

const canExecuteProposal = computed(() => {
  if (!proposalId.value) return false
  return ['suggested', 'approved'].includes(proposalStatus.value)
})

const canReviewProposal = computed(() => {
  if (!proposalId.value) return false
  return proposalStatus.value === 'executed'
})
```

- [ ] **Step 3: Add review modal state**

Add this state below the computed helpers:

```ts
const showReviewModal = ref(false)
const reviewForm = ref({
  outcome: 'positive' as 'positive' | 'neutral' | 'negative',
  business_metric: '',
  metric_delta: null as number | null,
  notes: '',
})
```

- [ ] **Step 4: Add execute and review handlers**

Add these functions below `handleReject()`:

```ts
async function handleExecuteProposal() {
  if (!proposalId.value) {
    message.error('动作提案 ID 无效')
    return
  }
  mutating.value = true
  try {
    await executeActionProposal(proposalId.value, { executor: 'operator' })
    message.success('已执行')
    emit('statusUpdated', props.item)
  } catch (e: any) {
    message.error(e?.response?.data?.message || e?.response?.data?.data?.error || '执行失败')
  } finally {
    mutating.value = false
  }
}

async function handleSubmitReview() {
  if (!proposalId.value) {
    message.error('动作提案 ID 无效')
    return
  }
  mutating.value = true
  try {
    await reviewActionProposal(proposalId.value, {
      outcome: reviewForm.value.outcome,
      business_metric: reviewForm.value.business_metric || null,
      metric_delta: reviewForm.value.metric_delta,
      notes: reviewForm.value.notes || null,
    })
    message.success('复盘已提交')
    showReviewModal.value = false
    emit('statusUpdated', props.item)
  } catch (e: any) {
    message.error(e?.response?.data?.message || e?.response?.data?.data?.error || '复盘失败')
  } finally {
    mutating.value = false
  }
}
```

- [ ] **Step 5: Add buttons to the card footer**

In the footer button group, after the reject button, add:

```vue
<n-button
  v-if="canExecuteProposal"
  size="tiny"
  type="primary"
  :loading="mutating"
  @click="handleExecuteProposal"
>执行</n-button>
<n-button
  v-if="canReviewProposal"
  size="tiny"
  secondary
  :loading="mutating"
  @click="showReviewModal = true"
>复盘</n-button>
```

- [ ] **Step 6: Add the review modal template**

Place this modal below the closing `</n-card>` and before `</template>`:

```vue
<n-modal v-model:show="showReviewModal" preset="card" title="动作复盘" style="width: 520px;">
  <n-form label-placement="top">
    <n-form-item label="结果">
      <n-select
        v-model:value="reviewForm.outcome"
        :options="[
          { label: '正向', value: 'positive' },
          { label: '中性', value: 'neutral' },
          { label: '负向', value: 'negative' },
        ]"
      />
    </n-form-item>
    <n-form-item label="业务指标">
      <n-input v-model:value="reviewForm.business_metric" placeholder="如 margin_delta / report_generated" />
    </n-form-item>
    <n-form-item label="指标变化">
      <n-input-number v-model:value="reviewForm.metric_delta" clearable style="width: 100%;" />
    </n-form-item>
    <n-form-item label="备注">
      <n-input v-model:value="reviewForm.notes" type="textarea" :rows="3" placeholder="记录执行后的业务反馈" />
    </n-form-item>
  </n-form>
  <template #footer>
    <n-space justify="end">
      <n-button @click="showReviewModal = false">取消</n-button>
      <n-button type="primary" :loading="mutating" @click="handleSubmitReview">提交复盘</n-button>
    </n-space>
  </template>
</n-modal>
```

- [ ] **Step 7: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/components/agentos/WorkItemCard.vue
git commit -m "feat: complete agentos proposal card actions"
```

## Task 3: Tighten WorkItems Page Refresh And Form UX

**Files:**
- Modify: `frontend/src/views/agentos/WorkItems.vue`

- [ ] **Step 1: Make create proposal refresh awaitable**

In `handleCreateProposal()`, change:

```ts
fetchItems()
```

to:

```ts
await fetchItems()
```

- [ ] **Step 2: Reset review-sensitive filters after creating a proposal only when needed**

Keep the existing filters. Do not auto-clear filters; the user may intentionally be viewing `source_type=action_proposal`. Only ensure newly created proposals appear when no conflicting filter is active.

Add after `resetForm()`:

```ts
if (filters.sourceType && filters.sourceType !== 'action_proposal') {
  filters.sourceType = 'action_proposal'
}
```

- [ ] **Step 3: Fix total parsing to preserve zero**

Change:

```ts
total.value = res?.total || res?.data?.total || 0
```

to:

```ts
total.value = res?.total ?? res?.data?.total ?? 0
```

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/agentos/WorkItems.vue
git commit -m "fix: refresh agentos work items after mutations"
```

## Task 4: Backend Regression And Narrow Fixes

**Files:**
- Verify: `backend/app/agentos/router.py`
- Verify: `backend/app/agentos/action_center_service.py`
- Verify: `backend/app/agentos/service.py`
- Test: `backend/tests/test_agentos_action_center.py`
- Test: `backend/tests/test_agentos_phase1.py`

- [ ] **Step 1: Run focused backend tests**

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py tests/test_agentos_phase1.py -q
```

Expected: pass.

- [ ] **Step 2: If ActionProposal approval via WorkItem fails, add a regression test**

Add this test to `backend/tests/test_agentos_action_center.py` inside `TestActionApprovalFlow`:

```python
async def test_work_item_approve_path_approves_action_proposal(self, async_client):
    proposal_id = await self._create_proposal(async_client)

    resp = await async_client.post(
        f"/api/agentos/work-items/action_proposal:{proposal_id}/approve",
        json={"action": "approve", "comment": "从任务卡审批"},
    )

    assert resp.status_code == 200, resp.text
    data = resp.json()["data"]
    assert data["ok"] is True
    assert data["proposal"]["status"] == "approved"
```

Run:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py::TestActionApprovalFlow::test_work_item_approve_path_approves_action_proposal -q
```

Expected: pass. If it fails due to `ValueError`, fix `AgentOSService.approve_work_item()` to catch invalid `action_proposal` IDs and return `{"ok": False, "error": "invalid_id"}`.

- [ ] **Step 3: Commit only if code or tests changed**

```bash
git add backend/app/agentos backend/tests/test_agentos_action_center.py
git commit -m "test: cover agentos work item proposal approval"
```

## Task 5: Manual Browser Verification

**Files:**
- Verify runtime behavior in `frontend/src/views/agentos/WorkItems.vue`
- Verify runtime behavior in `frontend/src/components/agentos/WorkItemCard.vue`

- [ ] **Step 1: Start backend**

Run:

```bash
cd backend && .venv/bin/uvicorn app.main:app --reload --host 127.0.0.1 --port 8001
```

Expected: app starts and routers are discovered.

- [ ] **Step 2: Start frontend**

Run in another terminal:

```bash
cd frontend && npm run dev -- --host 127.0.0.1 --port 3001
```

Expected: Vite serves the app at `http://127.0.0.1:3001`.

- [ ] **Step 3: Verify full proposal loop**

Open:

```text
http://127.0.0.1:3001/agentos/work-items
```

Perform:

1. Click `新建提案`.
2. Create a low-risk `daily_report` proposal with `requires_approval = false`.
3. Confirm a task card appears with source `动作提案`.
4. Click `执行`.
5. Confirm the card refreshes to completed/executed state.
6. Click `复盘`.
7. Submit outcome `positive`, metric `report_generated`, delta `1`, notes `日报已生成`.
8. Confirm the card refreshes and no error toast appears.

- [ ] **Step 4: Verify approval loop**

Perform:

1. Create an `inventory_allocate` proposal with `requires_approval = true`.
2. Confirm card shows `需审批`.
3. Click `审批通过`.
4. Confirm card refreshes.
5. Click `执行`.
6. Confirm success or a clear business validation error from the command handler.

- [ ] **Step 5: Stop dev servers**

Stop both foreground commands with `Ctrl+C`.

## Task 6: Documentation Update

**Files:**
- Modify: `docs/TIMELINE.md`
- Optional create: `docs/features/agentos-phase2-operator-loop.md`

- [ ] **Step 1: Add Phase 2 completion entry to timeline**

Add this section near the top of `docs/TIMELINE.md`:

```markdown
## 2026-06-22：AgentOS Phase 2 操作闭环 ✅

完成 AgentOS 从建议到执行复盘的最小可用闭环：

| 维度 | 变化 |
|------|------|
| **动作中枢** | ActionProposal 支持创建、审批、拒绝、执行、复盘 |
| **WorkItem 写入** | 支持任务状态更新、审批、拒绝并记录操作日志 |
| **业务执行** | 接入 `daily_report`、`listing_draft`、`profit_review`、`inventory_allocate`、`notify` command handlers |
| **前端任务中心** | 支持新建提案、审批/拒绝、执行、复盘和列表刷新 |
| **测试** | Action Center 与 WorkItem mutation 聚焦测试通过 |
```

- [ ] **Step 2: Update P0 board**

Change P0 item `AgentOS Phase 2` from next priority to completed or replace it with:

```markdown
| 1 | 🔗 **AgentOS Phase 2 验收收尾** | 演示脚本、异常态文案、操作日志筛选体验 | `agentos`, `agent` |
```

- [ ] **Step 3: Commit docs**

```bash
git add docs/TIMELINE.md docs/features/agentos-phase2-operator-loop.md
git commit -m "docs: record agentos phase2 completion plan"
```

If the optional feature doc was not created, omit it from `git add`.

## Task 7: Final Verification

**Files:**
- All touched files.

- [ ] **Step 1: Run backend focused tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py tests/test_agentos_phase1.py -q
```

Expected: pass.

- [ ] **Step 2: Run frontend build**

```bash
cd frontend && npm run build
```

Expected: pass.

- [ ] **Step 3: Check final diff**

```bash
git status --short
git diff --stat
```

Expected: only intentional files changed.

- [ ] **Step 4: Final commit if needed**

If Task 2-6 were not committed independently:

```bash
git add frontend/src/components/agentos/WorkItemCard.vue frontend/src/views/agentos/WorkItems.vue docs/TIMELINE.md
git commit -m "feat: complete agentos phase2 operator loop"
```

## Acceptance Criteria

- Backend tests pass:

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_action_center.py tests/test_agentos_phase1.py -q
```

- Frontend build passes:

```bash
cd frontend && npm run build
```

- Manual loop works:
  - Create proposal.
  - Approve/reject when approval is required.
  - Execute approved or auto-suggested proposal.
  - Review executed proposal.
  - WorkItems list refreshes after every mutation.
- `docs/TIMELINE.md` no longer presents AgentOS Phase 2 as untouched future work.

## Risk Controls

- If backend tests fail because seed data is missing, keep using mocked handlers in `test_agentos_action_center.py`; do not broaden tests into full catalog/order fixtures.
- If a command handler returns a business validation error, surface the backend message in the UI and keep the proposal in `failed`; do not hide failures as success.
- If frontend component tests are not configured, do not spend the day adding a test framework. Use `npm run build` plus manual browser verification.
- If database migrations are out of sync, run `cd backend && .venv/bin/alembic upgrade heads`; do not create a new migration unless a model/table genuinely has no migration.

## Self-Review

- Spec coverage: The plan covers create, approve, reject, execute, review, refresh, tests, manual verification, and docs.
- Placeholder scan: No TBD or open-ended implementation placeholders remain.
- Type consistency: Frontend API names match `agentos.ts`; backend route names and statuses match `ActionCenterService` and `AgentOSService`.
