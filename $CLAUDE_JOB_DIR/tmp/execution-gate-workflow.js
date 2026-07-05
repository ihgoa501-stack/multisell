export const meta = {
  name: 'execution-gate-parallel',
  description: 'Parallel P0/P1 execution gate implementation — 3 parallel agents then verify',
  phases: [
    { title: 'AI Gate', detail: 'Execution gate hardening + model fields + guardrails + handler' },
    { title: 'Approval Wire', detail: 'User binding + event bus + scheduler health + closed-loop subscriber' },
    { title: 'Frontend', detail: 'ActionConfirmModal component + page integration' },
    { title: 'Integrate', detail: 'Merge branches + verify + create PR' },
  ],
}

// ── Phase 1: AI execution gate ──
phase('AI Gate')
const aiResult = await agent(`
You are implementing P0-2 (execution gate) and part of P0-3 (model fields) for the AIOS execution gate.

Your working directory is the git worktree root. The repo has backend-go/ and frontend-next/ as subdirectories.

MAKE THESE CHANGES:

1. **internal/ai/model.go** — Add ExecutionMode, IdempotencyKey, ApprovedByUserID, ExecutedByUserID to UnifiedAction struct. Add ExecutionMode and IdempotencyKey to CreateActionInput.

2. **internal/common/types.go** — Add UserIDFromCtx function:
\`\`\`go
func UserIDFromCtx(c *gin.Context) *int64 {
    v, ok := c.Get("user_id")
    if !ok { return nil }
    switch x := v.(type) {
    case int64: return &x
    case float64: n := int64(x); return &n
    }
    return nil
}
\`\`\`

3. **internal/ai/service.go** —
   - Add import: "github.com/lingmirror/backend-go/internal/aios/guardrails" + "fmt"
   - Add guard field to Service struct: \`\`\`guard  *guardrails.Chain\`\`\`
   - Add WithGuard method
   - Replace ExecuteAction with enhanced version that checks:
     a) Idempotency (if key set and already executed, return existing)
     b) Catalog validation
     c) Status transitions
     d) Approval required
     e) Guardrails L4 check via s.guard.Check()
     f) Dry-run mode
   - Accept userID *int64 param in ExecuteAction, store in executed_by_user_id
   - Accept userID *int64 param in ApproveAction (store in approved_by_user_id) and RejectAction

4. **internal/ai/handler.go** —
   - ExecuteAction: extract userID via userIDFromCtx(c), pass to service
   - ApproveAction: pass userIDFromCtx(c) to service
   - RejectAction: pass userIDFromCtx(c) to service

5. **internal/ai/routes.go** — Add guard *guardrails.Chain parameter to RegisterRoutes, chain WithGuard on service creation. Import "github.com/lingmirror/backend-go/internal/aios/guardrails".

6. **internal/ai/orchestrator.go** — Update all ApproveAction/RejectAction/ExecuteAction calls to pass nil userID where system-initiated.

7. **internal/ai/ai_test.go** — Update test calls to pass nil for userID param.

COMMIT with message: "P0-2: AI execution gate — model fields + guardrails + idempotency + mode"

When done, return the branch name you committed to.
`, {label: 'AI Gate', isolation: 'worktree', phase: 'AI Gate'})

// ── Phase 2: Approval + router wiring ──
phase('Approval Wire')
const approvalResult = await agent(`
You are implementing P0-3 (approval user binding) and P1-2 (closed-loop) for the AIOS execution gate.

Your working directory is the git worktree root.

MAKE THESE CHANGES:

1. **internal/domain/approval/model.go** —
   - Add RequesterUserID *int64 and ReviewerUserID *int64 to ApprovalRequest struct
   - Add RequesterUserID *int64 and ReviewerUserID *int64 to CreateApprovalInput
   - Add ReviewerUserID *int64 to ReviewApprovalInput

2. **internal/domain/approval/service.go** —
   - Add "context" and "github.com/lingmirror/backend-go/internal/platform/eventbus" imports
   - Add bus field *eventbus.Bus to Service struct
   - Add WithBus method
   - In Create method, set RequesterUserID from input
   - In Review method, add reviewer_user_id to updates map
   - After syncing unified_action in Review, call s.publishApprovalEvent() to publish lifecycle events
   - Add publishApprovalEvent method:
\`\`\`go
func (s *Service) publishApprovalEvent(req *ApprovalRequest, status string) {
    if s.bus == nil { return }
    topic := fmt.Sprintf("approval.%s.%s", status, req.RequestType)
    ctx := context.Background()
    s.bus.Publish(ctx, topic, "approval", map[string]interface{}{
        "approval_id": req.ID, "status": status, "request_type": req.RequestType,
        "entity_type": req.EntityType, "entity_id": req.EntityID,
        "product_id": req.ProductID, "reviewer": req.Reviewer,
        "reviewer_user_id": req.ReviewerUserID,
    })
}
\`\`\`

3. **internal/domain/approval/handler.go** —
   - In CreateApproval: set input.RequesterUserID = common.UserIDFromCtx(c)
   - In ReviewApproval: set input.ReviewerUserID = common.UserIDFromCtx(c)

4. **internal/httpx/router.go** —
   - Add after "go sched.Start(busCtx)":
\`\`\`go
r.GET("/api/v1/aios/scheduler/tasks", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"tasks": sched.TaskRunState()})
})
\`\`\`
   - Change second approvalSvc creation to add .WithBus(bus)
   - After loopSvc definition, add subscriber:
\`\`\`go
bus.Subscribe("approval.approved.listing_task", func(ctx context.Context, evt eventbus.Event) error {
    payload := evt.Payload
    if payload == nil { return nil }
    productID, _ := payload["product_id"].(float64)
    approvalID, _ := payload["approval_id"].(float64)
    approvalIDInt := int64(approvalID)
    ltSvc := listingtask.NewService(db, logger, prismSvc, prismStrict, approvalSvc, auditSvc, loopSvc)
    _, err := ltSvc.Create(&listingtask.CreateTaskInput{
        ProductID: int64(productID), PlatformID: 1, Status: "approved",
        ApprovalID: &approvalIDInt, CreatedBy: "system:approval_bridge",
    })
    if err != nil { logger.Warn("failed listing task from approval", zap.Error(err)); return err }
    return nil
})
\`\`\`
   - Then change:
\`\`\`go
ai.RegisterRoutes(protected, db, logger, hub, moaCoord, cmd)
\`\`\`
     to:
\`\`\`go
ai.RegisterRoutes(protected, db, logger, hub, moaCoord, cmd, aiosCfg.Guardrails)
\`\`\`

COMMIT with message: "P0-3/P1-2: Approval user binding + event bus + scheduler health + closed-loop"

When done, return the branch name you committed to.
`, {label: 'Approval Wire', isolation: 'worktree', phase: 'Approval Wire'})

// ── Phase 3: Frontend ──
phase('Frontend')
const frontendResult = await agent(`
You are implementing P1-1 (frontend confirmation dialog) for the AIOS execution gate.

Your working directory is the git worktree root. frontend-next/ is the frontend directory.

MAKE THESE CHANGES:

1. Create **frontend-next/src/components/actions/ActionConfirmModal.tsx**:

A reusable modal component with:
- Action types (id, title, agent_id, etc.)
- Risk level mapping: low/green/低风险, medium/orange/中风险, high/red/高风险, critical/purple/严重
- Props: action (ConfirmAction|null), open (boolean), mode ('approve'|'reject'|'execute'|null), loading (boolean), onClose (()=>void), onConfirm ((action, reason)=>void)
- Shows risk level banner with description
- Shows action details in Descriptions component: ID, title, type, execution mode, agent, confidence, trace_id, description, proposed_by
- Shows payload in pre/json
- Shows reason input (optional for approve/execute, required for reject)
- High-risk execute requires typing the action title to confirm
- Shows rollback notice for execute mode
- Different footers for each mode: view (close only), approve (cancel+confirm), reject (cancel+reject), execute (cancel+execute)

Import from antd: Alert, Button, Descriptions, Input, Modal, Space, Tag, Typography
Icons: CheckCircleOutlined, CloseCircleOutlined, ExclamationCircleOutlined, PlayCircleOutlined, SafetyCertificateOutlined

2. Update **frontend-next/src/app/(main)/actions/page.tsx**:

- Add import of ActionConfirmModal
- Remove Modal from antd imports
- Add actionMode state: useState<'approve'|'reject'|'execute'|null>(null)
- Change openModal to accept mode parameter
- Change handleDecision to handleConfirm with mode routing
- Change mutations to accept {id, reason} objects instead of just id
- Add executeMutation that passes 'execute' mode to openModal
- Replace inline Modal with ActionConfirmModal component

COMMIT with message: "P1-1: Frontend action confirmation dialog"

When done, return the branch name you committed to.
`, {label: 'Frontend', isolation: 'worktree', phase: 'Frontend'})

// ── Phase 4: Verify and integrate ──
phase('Integrate')
log(`AI Gate branch: ${aiResult}`)
log(`Approval Wire branch: ${approvalResult}`)
log(`Frontend branch: ${frontendResult}`)

await agent(`
All 3 subagents completed their work in parallel worktrees. Now you need to:

1. Cherry-pick or merge all 3 branches onto a single branch
2. Verify everything compiles (go build ./... and npm run build)
3. Run tests (go test ./...)
4. Create a draft PR

The 3 branches are:
- AI Gate: ${aiResult}
- Approval Wire: ${approvalResult}
- Frontend: ${frontendResult}

Start by creating a new branch from origin/main, then merge all 3 branches. Fix any conflicts.
`, {label: 'Verify & PR', phase: 'Integrate'})
