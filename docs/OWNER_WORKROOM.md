# LingMirror Owner Workroom

Last updated: 2026-06-30

## What the Owner Can Do Now

1. **View decision queue** at `GET /owner/decision-queue` — see candidate products, their completeness scores, profit margins, agent recommendations, listing task status, approval status, and blocking reasons
2. **Approve/reject agent suggestions** through the approval module at `POST /approval`
3. **Track agent feedback** — whether recommendations were accepted or rejected, recorded via `POST /listing-task/:task_id/feedback`
4. **See blocking reasons** — if a task can't execute, the system shows why (no approval, blocked status, missing requirements, etc.)

## Sandbox / Mock Boundaries

| Action | Status | Notes |
|--------|--------|-------|
| Loop Evaluate | Mock | Generates recommendations using local rule engine, no AI API |
| ListingTask Execute | Local | Does not call external platform APIs; Prism is config-driven |
| Platform Sync | Mock | Uses `mock_sync_status` table, no real platform connections |
| Price / Inventory changes | Blocked | No API for real mutation; all gated |
| External publishing | Blocked | No production platform adapters |

## How High-Risk Actions Are Blocked

1. **Auth/JWT** — all APIs require JWT (middleware)
2. **RBAC** — sensitive modules gated by `RequirePermission`
3. **State Machine** — ListingTask status transitions validated; invalid transitions rejected
4. **Approval Gate** — ExecuteTask requires an approved `approval_request`; without it, execution fails with clear error
5. **Idempotency** — completed tasks cannot execute again
6. **Audit** — all key state transitions write to `operationlog`

## How to Verify the First Closed Loop

1. Create a candidate product (`POST /candidates`)
2. Run loop evaluate (`POST /loop/evaluate/:productId`) — generates completeness check, profit calculation, recommendation, and blocked listing task
3. Check owner decision queue (`GET /owner/decision-queue`) — see recommendation and task
4. Create an approval (`POST /approval`) — action: approve
5. Execute the listing task (`POST /listing-task/:task_id/execute`) — should succeed
6. Submit feedback (`POST /listing-task/:task_id/feedback`) — record acceptance
7. Verify audit trail (`GET /operationlog?module=listingtask`)
