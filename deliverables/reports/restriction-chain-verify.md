# Restriction Chain Audit Report

Date: 2026-07-06<br>
Scope: backend-go/ — all 6 restriction mechanisms<br>
Method: Source read + caller trace + test coverage + migration data<br>
Verdict: **4 WORKS, 1 HAS_GAP, 1 HAS_GAP** (DispatchSafe gap, MutationGuard coverage gap)

---

## 1. ActionCatalog — WORKS (with caveat)

### Source
`backend-go/internal/platform/actioncatalog/catalog.go`

### Evidence
- **ValidateProduction** correctly returns three error types: `ErrUnknownAction` (unknown type), `ErrAutonomousBlocked` (L4, `AutonomousBlocked=true`), `ErrApprovalRequired` (L3 with `RequireApproval=true` but no approval).
- **Default catalog** registers 23 entries across all four levels (L1=5, L2=4, L3=12, L4=1). Every `price_update`, `order_cancel`, `refund_issue`, `sync_inventory`, `credential_change`, `permission_change`, `destructive_data_change` is L3 with `RequireApproval=true`. `auto_publish` is L4 with `AutonomousBlocked=true`.
- **Two system actions** (`system.inventory.receive`, `system.inventory.aftersale_restock`) are registered as L3 with `RequireApproval=false` — documented as deterministic system-internal state transitions.
- **Tests** (`catalog_test.go`, 12 tests) cover: all known actions present, L1 no approval, L2 no approval, L3 requires approval, L4 blocked, unknown action rejected, L3 approval required, L3 with approval passes, L1 passes, price_review requires approval, duplicate entries panic, HasAction.
- **7 callers** across `command.go`, `ai/service.go`, tests.

### Caveat
`ValidateProduction` accepts a `riskLevel int` parameter but **never uses it** in the function body. The spec's `AutonomousBlocked` and `RequireApproval` fields drive the checks exclusively. The unused parameter is misleading and could create a false sense of risk-based gating. This is a design smell but not a functional gap — the catalog entries themselves are self-consistent.

---

## 2. DispatchSafe — HAS_GAP

### Source
`backend-go/internal/platform/command/command.go:128`

### Evidence
- **DispatchSafe exists** and is well-implemented: validates structural fields (identity, mode, risk), enforces dry-run (validate only, never mutate), enforces catalog validation in production mode (`ValidateProduction`), enforces approval for high-risk actions (`RiskLevel >= RiskHigh || ApprovalRequired`), and calls the optional `PolicyChecker.IsApproved` and `AuditRecorder`.
- **Tests** (`action_test.go`, 7 DispatchSafe tests) cover: dry-run no execution, dry-run unregistered handler rejected, production high-risk requires approval, production high-risk with approval passes, production approval not in policy rejected, sandbox allows execution, low-risk no approval needed, structural validation (missing fields), default approval_required for high-risk.

### The Gap
**DispatchSafe is never called in production code.** The production execution flow is:

```
orchestrator.runWithTimeout
  → aiSvc.ApproveAction / aiSvc.RejectAction / aiSvc.ExecuteAction
    → ExecuteAction (ai/service.go:258)
      → manually calls s.cat.ValidateProduction (line 291)
      → manually checks RequiresApproval + status (line 399)
      → manually runs guardrails check (line 419)
      → calls s.cmd.Dispatch(...) (line 491) -- NOT DispatchSafe
```

`ExecuteAction` bypasses `DispatchSafe` and calls raw `Dispatch` instead. It re-implements catalog validation and approval checking internally — but:
1. **No `PolicyChecker` used** — approval ID is validated only by action status (`Status == "approved"`), not by a policy-level `IsApproved` check.
2. **No `AuditRecorder` wired** — the `Dispatcher` in `router.go:167` is created as `command.NewDispatcher(logger, command.WithCatalog(cat))` — no `WithAuditRecorder`. Even if DispatchSafe were called, audit records would be nil.
3. **The `PolicyChecker` interface** (used by DispatchSafe) has zero implementations in the codebase — it exists for dependency-injection but nothing implements `IsApproved`.

---

## 3. forbidden_action Seeds — WORKS

### Source
- Migration: `backend-go/migrations/000050_forbidden_actions.up.sql`
- Logic: `backend-go/internal/domain/actionpolicy/forbidden.go`

### Evidence
- **Migration** creates the `forbidden_action` table and seeds 7 rows (all `agent_id='*'`, `risk_level='high'`):
  - `price_update`, `inventory_update`, `order_cancel`, `platform_publish`, `credential_change`, `permission_change`, `data_delete`
- **CheckForbidden** (`forbidden.go:27-45`):
  - Gracefully returns `nil` if the DB table doesn't exist (no panic in dev).
  - Queries with wildcard matching: `(action_type = ? OR action_type = '*') AND (agent_id = ? OR agent_id = '' OR agent_id = '*')`.
  - Also matches `risk_level` wildcard.
  - Returns a clean error if count > 0.
- **Wired** in `orchestrator.go:366`: called after `persistAction()` creates the action. If blocked, the action is rejected via `aiSvc.RejectAction`.
- **Mock seed** in `router.go:716` runs `mock.NewService().SeedMockData()` — worth a quick check that mock data doesn't insert conflicting forbidden_action rows. Investigation shows the mock service does NOT touch the `forbidden_action` table (confirmed by checking `backend-go/internal/domain/mock/` — no references to forbidden_action), so there's no interference.

### Test Coverage
- `actionpolicy_test.go` tests handler CRUD and `matches()` logic, but **does NOT test `CheckForbidden`**.

---

## 4. Approval Policy (Evaluate) — WORKS

### Source
- Code: `backend-go/internal/domain/actionpolicy/service.go`
- Migration: `backend-go/migrations/000004_approval_policy.up.sql`

### Evidence
- **Service.Evaluate** (`service.go:51-78`) iterates all enabled `approval_policy_rule` rows, matches by `RiskLevel`, `ActionType`, `SquadID`, `AgentID`, `BusinessObjectType`, `MaxAmount`, `MaxQuantity`, `MinConfidence` (all with wildcard support).
- **High-risk gate** (lines 67-75): any action with `RiskLevel=="high"` that would otherwise be `auto_approve` is forcibly escalated to `escalate`. This prevents unintended auto-approval of actually-high-risk actions.
- **10 seed rules** cover: L3 auto-approve for low-risk, A5 stock alert auto, G3 small discount auto, A2 listing draft auto, medium risk >1000 escalate, high-risk always escalate, critical risk block, A7 compliance manual, A6 low-confidence escalate, batch ops >10 SKUs escalate.
- **Wired** in `orchestrator.go:381-430`: runs after action creation, can auto-approve + auto-execute, block the action, or leave it for human review.
- **Wired** via HTTP: `POST /policy/evaluate` and CRUD endpoints in `routes.go`.
- **Tests** cover: ListRules, CreateAndGetRule, Evaluate with no match, wildcard matching, amount boundary.

### Note
Policy evaluation runs **after** action creation (the action is already persisted in "suggested" state). This is deliberate — the action is the record, and policy then decides its fate. The audit trail captures the complete lifecycle.

---

## 5. ToolBridge — WORKS (no gap expected)

### Source
`backend-go/internal/platform/toolbridge/bridge.go`

### Evidence
- ToolBridge is purely a **data-collection routing layer**: `FetchPage(ctx, url) → *PageData, error`. It collects structured data from URLs via registered `ToolDriver` implementations.
- **No mutation surface**: no business state changes, no DB writes, no action execution.
- **Drivers**: `PluginDriver` (browser extension) — registered in `router.go:818`.
- **Used by**: A12 Collection Agent and sourcing for web page data gathering.
- **No bypass of action/approval system** — ToolBridge has no access to the dispatcher, catalog, or policy engine. It's a read-only data pipeline.

This is the correct design: read-side infrastructure should not participate in the action approval chain. No action needed.

---

## 6. MutationGuard — WORKS (coverage gap in scope)

### Source
- Guard: `backend-go/internal/platform/eventbus/guard.go`
- Wiring: `backend-go/internal/httpx/router.go:339-340`
- Integration: `guard_test.go` (6 unit tests), `guard_integration_test.go` (1 integration test)

### Evidence
- **MutationGuard** wraps event bus handlers with structured audit logging (start = "pending", end = "executed" / "failed").
- Uses `MutationAuditLogger.LogStructured` for fire-and-forget audit writes.
- Falls through safely on nil guard or nil audit logger.
- **Wired to 2 handlers** in `router.go`:
  1. `supplychain.order.received` → `system.inventory.receive`
  2. `supplychain.aftersale.completed` → `system.inventory.aftersale_restock`
- **HTTP audit middleware** (`httpx/middleware/audit.go`) is globally active (line 126 in router.go) and records all mutation HTTP requests (POST/PUT/PATCH/DELETE) plus sensitive GETs.

### Gap
Only 2 out of ~22 event bus subscription handlers are wrapped with MutationGuard. Many subscription handlers that perform mutations are **not guarded**:
- `sourcing.recommend` (triggers A2 listing_optimize for high-score — a mutation path)
- `approval.approved.listing_task` (executes listing tasks with side effects)
- Various `scheduler.tick.*` handlers that trigger agent runs (agents may then create actions)
- `supplychain.aftersale.returned` (creates reverse logistics, modifies state)

The guard is correctly designed but underapplied. However, the **HTTP audit middleware** covers direct HTTP mutations, and the **agent decision audit** subscribers (`agent.decided.*`) log all agent decisions. So the audit gap is specifically for event-bus-triggered state mutations that bypass both HTTP and the agent path.

---

## Cross-Cutting Issue: Merge Conflicts

**3 files** have unresolved `<<<<<<< HEAD` merge conflict markers:
1. `backend-go/internal/ai/orchestrator.go` (lines 369-373, 399-406, 416-419)
2. `backend-go/internal/ai/service.go` (lines 192-201, 215-225, 243-258, 300-395, 399-439)
3. `backend-go/internal/ai/handler.go` (lines 211-215, 235-239, 259-266)
4. `backend-go/internal/ai/routes.go` (lines 18-25)

These files **will not compile** in their current state. The conflict is between two versions of function signatures (`(id, operator, reason, userID)` vs `(id, operator, userID, reason)` parameter ordering across `ApproveAction`, `RejectAction`, `ExecuteAction`). This indicates the restriction chain cannot be compiled or tested until resolved.

---

## Summary

| # | Mechanism | Verdict | Key Evidence |
|---|-----------|---------|-------------|
| 1 | ActionCatalog | **WORKS** | 23 entries, all levels correctly labelled, ValidateProduction rejects L4 + requires L3 approval. Unused `riskLevel` param is dead code. |
| 2 | DispatchSafe | **HAS_GAP** | Never called in production. `ExecuteAction` calls raw `Dispatch`, not `DispatchSafe`. No `PolicyChecker` implementation found. No `AuditRecorder` wired on Dispatcher. |
| 3 | forbidden_action | **WORKS** | 7 seed entries, `CheckForbidden` with wildcard matching, wired into orchestrator. No test for CheckForbidden itself. |
| 4 | Approval Policy | **WORKS** | 10 seed rules, Evaluate with high-risk gate, wired to orchestrator + HTTP endpoint. Tests cover matching + boundaries. |
| 5 | ToolBridge | **WORKS** | Read-only data collection, no mutation surface, no access to action pipeline. Correct by design. |
| 6 | MutationGuard | **WORKS (coverage gap)** | Guard correctly wraps + audits 2 handlers. ~20 other mutation handlers are unguarded. HTTP audit middleware provides partial coverage. |

### Overall Chain Flow

```
Agent proposes action
  → CheckForbidden (forbidden_action table)   ← WORKS
  → CreateAction (persisted as "suggested")    ← always happens
  → Policy Evaluate (auto_approve / block / escalate)  ← WORKS
    → auto_approve path: ApproveAction + ExecuteAction  ← HAS_GAP (bypasses DispatchSafe)
    → escalate path: approval_request created           ← WORKS
    → block path: RejectAction                          ← WORKS
  → ExecuteAction
    → ValidateProduction (catalog)                     ← WORKS (manual, not via DispatchSafe)
    → Guardrails (L4 execution guard)                  ← WORKS
    → cmd.Dispatch (NOT DispatchSafe)                  ← THE GAP
```

### Recommended Fix Priority (not requested, for awareness)

1. Resolve merge conflicts in 4 files (orchestrator.go, service.go, handler.go, routes.go)
2. Either wire `DispatchSafe` into `ExecuteAction`, or accept the manual-gate pattern as intentional and document it
3. Wire `AuditRecorder` onto the Dispatcher in router.go
4. Add more MutationGuard wrappers on service handlers that mutate state
