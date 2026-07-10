# Owner Cockpit UI Audit — 2026-07-10

## Pages Audited

1. `/owner` — Owner Cockpit
2. `/actions` — Action Center list
3. `/actions/[id]` — Action detail / review room
4. `/listing-tasks` — Listing tasks list (generic CRUD)
5. `/listing-tasks/[id]` — Listing task detail

## Components Audited

- `components/ui/HighRiskConfirmDialog` — shared high-risk confirmation dialog
- `components/actions/ActionRiskConfirmDialog` — wrapper for action confirmations
- `components/actions/ActionConfirmModal` — older action detail modal (view-only mode)

---

## Evaluation Against 7 Questions

Source: `docs/CURRENT_DIRECTION_AND_PRIORITIES.md` section 2 "Owner Control Before Autonomy".

| # | Question | Owner | Actions List | Actions Detail | Listing Tasks List | Listing Tasks Detail |
|---|----------|-------|-------------|----------------|-------------------|---------------------|
| 1 | What happened? | YES — product_title, suggestion | YES — title, action_type | YES — title, description, payload | PARTIAL — ID/product_id only | YES — full task info |
| 2 | Why important? | PARTIAL — confidence/risk, no profit impact | PARTIAL — risk/confidence, no business impact | YES — risk_reason + description | NO — no risk/business info | PARTIAL — no business impact |
| 3 | Agent recommendation? | YES — suggestion + decision columns | YES — title links to detail | YES — full detail | NO — generic CRUD | NO — no agent context |
| 4 | Approve consequence? | YES — HighRiskConfirmDialog | YES — ActionRiskConfirmDialog | NO — fires immediately | N/A | YES — HighRiskConfirmDialog |
| 5 | Reject/wait consequence? | YES — reject dialog shows consequence | YES — ActionRiskConfirmDialog | PARTIAL — reject has reason modal, no consequence text | N/A | NO — no reject option |
| 6 | Mode badge? | YES — execution_mode column + dialog | PARTIAL — in dialog only, not in table | NO — no mode badge | NO — no mode column | YES — execution_mode badge |
| 7 | Audit trail? | PARTIAL — dialog mentions operation_log, no link | PARTIAL — trace_id in dialog, not linked | YES — audit timeline + trace link | NO — no audit reference | PARTIAL — approval info shown |

---

## Gap Classification

### Frontend-Only Gaps (backend already returns the data)

| Gap | Page | File | Data Available? | Fix Complexity |
|-----|------|------|----------------|----------------|
| Missing execution_mode column | Actions list | `actions/page.tsx` | YES — `UnifiedAction.execution_mode?: string` | ~5 lines |
| Missing mode badge | Action detail | `actions/[id]/page.tsx` | Likely YES — same API as list | ~5 lines |
| Missing execution_mode column | Listing tasks list | `listing-tasks/page.tsx` | YES — same API as detail | ~5 lines |
| Hardcoded environmentMode="production" | Listing task detail | `listing-tasks/[id]/page.tsx` | YES — `task.execution_mode` available | ~3 lines |
| Approve/execute fires without confirmation | Action detail | `actions/[id]/page.tsx` | YES — can wrap in dialog | ~15 lines (bigger change) |

### Backend-Dependent Gaps (need API changes)

| Gap | Page | Reason |
|-----|------|--------|
| No profit/business impact in decision queue | Owner page | Backend doesn't return profit impact in decision queue API |
| No clickable audit trail link | All pages | No frontend audit page exists (`/audit` or similar) |
| No agent recommendation context | Listing tasks list | Generic CRUD — no agent fields returned |
| No reject option | Listing task detail | Backend doesn't expose reject endpoint for listing tasks |

---

## Minimum Frontend-Only Fixes (implemented below)

### Fix 1: Actions list — add execution_mode column

**File:** `frontend-next/src/app/(main)/actions/page.tsx`

**Change:** Insert a column between "confidence" and "status" showing the execution mode with a color-coded tag.

**Diff:**
```diff
+      {
+        title: '模式',
+        dataIndex: 'execution_mode',
+        key: 'execution_mode',
+        width: 90,
+        render: (v: string | undefined) => {
+          if (!v) return '-';
+          const mc: Record<string, string> = { dry_run: 'default', sandbox: 'orange', production: 'red' };
+          const ml: Record<string, string> = { dry_run: 'Dry-Run', sandbox: 'Sandbox', production: 'Production' };
+          return <Tag color={mc[v] || 'default'}>{ml[v] || v}</Tag>;
+        },
+      },
```

Insert after the `confidence` column (around line 192) and before the `status` column.

---

### Fix 2: Action detail — add execution_mode to type and display badge

**File:** `frontend-next/src/app/(main)/actions/[id]/page.tsx`

**2a:** Add `execution_mode` to the local `UnifiedAction` interface (around line 50):
```diff
   operator?: string;
+  execution_mode?: string;
```

**2b:** Add a mode badge in the header area, after the confidence tag (around line 267):
```diff
                    >
                      置信度 {(action.confidence * 100).toFixed(0)}%
                    </Tag>
+                    {action.execution_mode && (
+                      <Tag color={
+                        action.execution_mode === 'production' ? 'red' :
+                        action.execution_mode === 'sandbox' ? 'orange' : 'default'
+                      }>
+                        {action.execution_mode === 'dry_run' ? 'Dry-Run' :
+                         action.execution_mode === 'sandbox' ? 'Sandbox' :
+                         action.execution_mode === 'production' ? 'Production' : action.execution_mode}
+                      </Tag>
+                    )}
```

---

### Fix 3: Listing tasks list — add execution_mode column

**File:** `frontend-next/src/app/(main)/listing-tasks/page.tsx`

**Change:** Add a column between "platform ID" and "status" showing execution mode.

The `execution_mode` on listing tasks is a number (0-3). Add a column entry:
```diff
+        {
+          title: '执行模式',
+          dataIndex: 'execution_mode',
+          width: 100,
+          render: (v: unknown) => {
+            const labels: Record<number, string> = { 0: 'Dry-Run', 1: 'Sandbox', 2: '需审批', 3: '生产' };
+            const colors: Record<number, string> = { 0: 'default', 1: 'orange', 2: 'purple', 3: 'red' };
+            const num = Number(v);
+            if (isNaN(num)) return '-';
+            return <Tag color={colors[num] || 'default'}>{labels[num] || '未知'}</Tag>;
+          },
+        },
```

Insert after the `platform_id` column definition (around line 52), before the `status` column.

---

### Fix 4: Listing task detail — fix hardcoded environmentMode

**File:** `frontend-next/src/app/(main)/listing-tasks/[id]/page.tsx`

**Change:** Replace hardcoded `"production"` with a helper derived from `task.execution_mode`.

Add helper function before the component (around line 78):
```diff
+function executionModeToEnvironment(mode?: number): 'dry_run' | 'sandbox' | 'production' {
+  if (mode === 0) return 'dry_run';
+  if (mode === 1) return 'sandbox';
+  return 'production';
+}
```

Replace line 218:
```diff
-              environmentMode="production"
+              environmentMode={executionModeToEnvironment(task?.execution_mode)}
```

---

## Verification

These changes are frontend-only and safe:
- They only add display of data the backend already returns
- No new API calls, no new mutations, no new pages
- No behavior change for existing flows
- The execution_mode field is optional in all types, so missing data shows as `-`

## Skipped (not frontend-only or beyond minimum scope)

- **Profit impact in decision queue:** Backend doesn't return this data — backend gap.
- **Audit trail link:** No `/audit` page exists to link to — would need a new page first.
- **Approve/execute confirmation in action detail:** ~15 lines, more than "3-5 lines per component". The list page's `ActionRiskConfirmDialog` already provides the consequence preview. The detail page is meant for direct action.
- **Agent context in listing tasks:** Listing tasks are technical execution records, not decision records. The agent decision context belongs in `/owner` and `/actions`.
