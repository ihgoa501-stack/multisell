# Product Loop Action Gate Audit

Date: 2026-07-10
Branch: feat/agentos-sandbox-safety-guards
Scope: backend only — approval/loop/owner/candidate/ai/listingtask domains

---

## 1. Complete Audit Table

Each row is a mutation path through the Product Loop. Checks are for AgentAction-gated, has explicit Approval, has RBAC on the HTTP route, has Audit logging, has Idempotency, and supports Dry-run/Sandbox.

| # | Mutation Path | AgentAction Gated | Has Approval | Has RBAC | Has Audit | Idempotency | Dry-Run/Sandbox | Risk |
|---|---|---|---|---|---|---|---|---|
| 1 | Loop Evaluate: creates listing_task + approval_request | NO (direct DB) | YES (approval_request created) | NO (protected, no perm check) | YES (oplog) | NO (no idempotency key) | YES (via listing task DecisionSnapshot) | HIGH |
| 2 | Owner RecordFeedback: adopt -> listing_task status update | NO (direct db.Model().Update) | NO (creates approval but listing task goes directly to pending_approval) | NO (protected, no perm check) | YES (oplog) | NO | NO | HIGH |
| 3 | Owner RecordFeedback: reject -> listing_task status update | NO (direct db.Model().Update) | NO | NO | YES (oplog) | NO | NO | MEDIUM |
| 4 | POST /approval create | NO | N/A (creates the approval) | NO (protected only) | NO (silent) | NO | NO | MEDIUM |
| 5 | PUT /approval/:id/review approve/reject | NO (bypassed if entity_type=unified_action -- writes unified_action table directly) | YES (approval record exists) | NO (protected only) | YES (oplog) | NO (concurrent review overwrites) | NO | HIGH |
| 6 | POST /listing-tasks (API create) | NO | NO | listing.read (WRONG) | NO (silent) | NO | NO | MEDIUM |
| 7 | PUT /listing-tasks/:id (API update) | NO | NO | listing.read (WRONG) | NO (silent) | NO | NO | HIGH |
| 8 | DELETE /listing-tasks/:id (API delete) | NO | NO | listing.read (WRONG) | NO (silent) | NO | NO | MEDIUM |
| 9 | POST /ai/actions/:id/execute (AI action execute) | YES | YES | ai.action | YES | YES (idempotency key) | YES (dry_run/sandbox) | LOW (properly gated) |
| 10 | EventBus approval.approved.listing_task -> ExecuteTask | NO (system action) | YES (state machine + approval ID check) | N/A (eventbus) | YES (MutationGuard + writeAudit) | YES (completed/executing checks) | YES (DecisionSnapshot mode) | LOW (well gated) |
| 11 | PUT /candidates/:id (candidate update) | NO | NO | NO | NO | NO | NO | LOW |
| 12 | POST /listing-task/:task_id/execute (manual execute) | NO | YES (state machine + approval check) | YES (listing_task:execute) | YES | PARTIAL (completed -> no-op) | YES (via DecisionSnapshot) | LOW (adequate internal gates) |
| 13 | PUT /ai/actions/:id/approve (AI action approve) | YES | YES | ai.action | YES | NO (concurrent approve overwrites) | YES | LOW |

---

## 2. Must Fix (high-confidence bypasses)

### M1. `listing.read` RBAC gates listing task mutations

**Location:** `/Users/lc/multisell/backend-go/internal/httpx/router.go:715`

```go
listingRoutes := protected.Group("", middleware.RequirePermission(db, "listing.read"))
```

**Problem:** The same `listing.read` permission gates both:
- GET /listing-tasks (read)
- POST /listing-tasks (create)
- PUT /listing-tasks/:id (update -- can change status, approval_id)
- DELETE /listing-tasks/:id (delete)

A user with read-only listing access can create, update, and delete listing tasks -- including changing their status from blocked to pending_approval or approved.

**Fix:** Add `middleware.RequirePermission(db, "listing.write")` to mutation routes. Shortest fix: split the route group into read-only and write.

### M2. Approval review bypasses UnifiedAction lifecycle when entity_type=unified_action

**Location:** `/Users/lc/multisell/backend-go/internal/domain/approval/service.go:139-140`

```go
uaUpdates := map[string]interface{}{}
if req.EntityType == "unified_action" && req.EntityID > 0 {
    uaUpdates["status"] = status
    uaUpdates["updated_at"] = now
    ...
}
if len(uaUpdates) > 0 {
    if err := tx.Table("unified_action").Where("id = ?", req.EntityID).Updates(uaUpdates).Error; err != nil {
```

**Problem:** The approval service directly writes to the `unified_action` table when an approval's entity_type is "unified_action". This bypasses:
- The `ai.action` RBAC permission required by `POST /ai/actions/:id/approve`
- The AI service's `ApproveAction`/`RejectAction` methods (which set timestamps, operator, and broadcast events)
- The status transition validation in `transitionAction`

**Impact:** An authenticated user can go through `PUT /approval/:id/review` (which has NO RBAC beyond JWT auth) to approve/reject a unified action, bypassing the `ai.action` RBAC gate entirely.

### M3. Approval routes have no RBAC

**Location:** `/Users/lc/multisell/backend-go/internal/domain/approval/routes.go:17-26`

```go
g.POST("", h.CreateApproval)
g.PUT("/:id/review", h.ReviewApproval)
```

**Problem:** Any authenticated user can create and review approvals. The handler correctly sets Requester/Reviewer from JWT, preventing impersonation, but there is no permission check. A team member with no approval authority can approve listing tasks.

### M4. Owner RecordFeedback has no RBAC

**Location:** `/Users/lc/multisell/backend-go/internal/domain/owner/service.go:271`

**Problem:** `RecordFeedback` can adopt or reject a listing recommendation, which directly mutates `listing_recommendation.feedback_status` and `listing_task.status` via raw DB calls. The route is under `protected` (JWT auth only), no RBAC check.

---

## 3. Should Fix

### S1. Owner RecordFeedback should go through listingtask Service

**Location:** `/Users/lc/multisell/backend-go/internal/domain/owner/service.go:311`

```go
s.db.Model(&task).Update("status", "pending_approval")
```

This direct DB mutation bypasses listingtask.Service.Update, which handles the status transition via the state machine. If the state machine rejects the transition, the direct DB Update silently succeeds, creating an inconsistent state. Use `listingtask.Service.Update()` instead of raw `db.Model().Update()`.

### S2. Candidate route mutations lack RBAC

**Location:** `/Users/lc/multisell/backend-go/internal/domain/candidate/routes.go:16-30`

All candidate mutation routes (POST, PUT, DELETE, FillFields, SkipField, Rescrape) are under `protected` with JWT only. Low risk because candidate products are work-in-progress data, but inconsistent with the rest of the system.

### S3. Listing task Create/Update API routes lack audit

Creating or updating a listing task via the API (`POST /listing-tasks`, `PUT /listing-tasks/:id`) does not write an audit log entry. The listing_task schema has a `created_by`/`updated_by` field on the model, so identity is recorded, but there is no structured audit log.

### S4. Loop Evaluate creates listing task bypassing UnifiedAction

**Location:** `/Users/lc/multisell/backend-go/internal/domain/loop/service.go:83-109`

The loop creates listing tasks and approval requests directly in a transaction, bypassing the UnifiedAction system entirely. This is a deliberate design choice (the Product Loop has its own simplified state machine), but it means:
- No actioncatalog validation of the action type
- No execution guardrails check
- No idempotency key on the listing task
- No sandbox execution mode on the listing task (only via DecisionSnapshot)

**When to fix:** When the Product Loop needs to respect execution guardrails (L4 checks).

---

## 4. Won't Fix Yet

### W1. Product Loop bypassing UnifiedAction entirely

**Rationale:** This is an intentional architectural decision. The Product Loop has its own state machine (`listing_task: blocked -> pending_approval -> approved -> executing -> completed/failed`) with its own approval linkage. Integrating with the UnifiedAction system would add complexity without immediate benefit, because:
- The listing task already has approval ID verification at execution time
- Audit logging is implemented via operationlog
- The EventBus mutation guard wraps the auto-execution path
- The same flow goes through a state machine transition check

The Product Loop is an independent workflow with its own safety gates, not a bypass of existing ones.

### W2. Direct DB mutations in RecordFeedback

**Rationale:** The owner cockpit pattern intentionally does direct DB mutations for status transitions because:
- It is a tightly controlled flow (adopt -> pending_approval + create_approval, reject -> rejected)
- It creates an approval request in the same call, maintaining the approval chain
- The approval execution gate is the actual safety barrier

**Upgrade path:** Move to listingtask.Service.Update when the listing task state machine needs to reject transitions from the owner cockpit path.

### W3. Candidate routes lack RBAC

**Rationale:** Candidate products are work-in-progress/suggestion data. They do not affect platform state, pricing, or inventory. Adding RBAC here is premature optimization. Any user who can log in can work on candidates.

### W4. Loop Evaluate creates listing task outside UnifiedAction

**Rationale:** See W1. The Product Loop workflow is intentionally decoupled from the UnifiedAction system. Adding an AgentAction step between Evaluate and listing task creation would add latency, require another RBAC check, and create two parallel tracking systems for the same workflow.

---

## 5. Code Changes Made

No code changes were made in this audit cycle. This document records the findings. The must-fix items (M1-M4) should be addressed in a follow-up PR.

### Diff Summary for each Must Fix

**M1:** In `/Users/lc/multisell/backend-go/internal/domain/listingtask/routes.go`, split the route group:
- Move POST, PUT, DELETE routes to a sub-group with `middleware.RequirePermission(db, "listing.write")`
- Keep GET routes on the `listing.read` group

**M2:** In `/Users/lc/multisell/backend-go/internal/domain/approval/service.go`, either:
- Remove the unified_action auto-sync from `Review()` (forces callers to use the AI service for unified_action transitions), or
- Add an RBAC check inside the sync path

**M3:** In `/Users/lc/multisell/backend-go/internal/domain/approval/routes.go`, add RBAC middleware:
```go
g.POST("", middleware.RequirePermission(db, "approval.create"), h.CreateApproval)
g.PUT("/:id/review", middleware.RequirePermission(db, "approval.review"), h.ReviewApproval)
```

**M4:** Add a route-level RBAC middleware or a permission check in the handler for the feedback route.

---

## 6. Summary

| Category | Count | Items |
|---|---|---|
| Must fix | 4 | M1-M4 -- all high-confidence, concrete bypasses |
| Should fix | 4 | S1-S4 -- real improvement, not urgent |
| Won't fix yet | 4 | W1-W4 -- intentional design or low risk |

The UnifiedAction system in `/ai` is well-gated (path #9) -- it has idempotency, RBAC, approval, audit, guardrails, and execution modes. The bypasses are in the **Product Loop's custom workflow** (loop -> owner -> approval -> listing_task -> ExecuteTask), which has its own safety mechanisms but lacks RBAC on several entry points and directly syncs unified_action state without going through the AI service lifecycle.
