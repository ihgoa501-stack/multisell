# Action Gate & Owner Cockpit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the AI Action Execution Expiration validation, the Sandbox execution shell runner in Go, and update the Next.js Owner Cockpit to handle sandbox runs.

**Architecture:**
1. In the Go backend, update `ExecuteAction` to fail if an action is older than 2 hours (`ErrActionExpired`).
2. Add support for `"sandbox"` execution mode in Go by triggering `scripts/run_sandbox.sh` in a subprocess, parsing exit status, and updating action state.
3. Update the Next.js Owner page to show sandbox loading indicators and log report links.

**Tech Stack:** Go 1.21 / Gin / GORM v2 / React / Next.js 14 / AntD

## Global Constraints
- Individual Go files must remain under 300 lines of code if possible.
- Use explicit types, proper error wrapping, and GORM database transactions.
- Frontend files must be TypeScript compliant with zero linter errors.

---

### Task 1: Action Execution Expiration Gate (Go Backend)

**Files:**
- Modify: `backend-go/internal/ai/service.go` (implement expiration check)
- Modify: `backend-go/internal/ai/ai_test.go` (unit tests for expiration check)

- [ ] **Step 1: Write the failing unit test**
  Add `TestService_ExecuteAction_Expired` in `backend-go/internal/ai/ai_test.go`:
  ```go
  func TestService_ExecuteAction_Expired(t *testing.T) {
      db := newTestDB(t)
      svc := NewService(db, testLogger())
      noApproval := false
      a, _ := svc.CreateAction(&CreateActionInput{
          SourceTable: "ai_trace", SourceID: "trc_exp_1", SourceType: "agent_run",
          AgentID: "A2", ActionType: "listing_optimize", Title: "expired action",
          RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
      })
      // Force Creation Date to 3 hours ago
      db.Model(&UnifiedAction{}).Where("id = ?", a.ID).Update("created_at", time.Now().Add(-3 * time.Hour))

      _, err := svc.ExecuteAction(a.ID, nil, "alice", "")
      if err == nil || !errors.Is(err, ErrActionExpired) {
          t.Fatalf("expected ErrActionExpired, got: %v", err)
      }
  }
  ```

- [ ] **Step 2: Run test to verify it fails**
  Run: `cd backend-go && go test -v -run TestService_ExecuteAction_Expired ./internal/ai`
  Expected: FAIL with compilation error (ErrActionExpired undefined) or test failure.

- [ ] **Step 3: Define error and write expiration validation**
  In `backend-go/internal/ai/service.go`, define `ErrActionExpired = errors.New("ai: action execution expired")`.
  In `ExecuteAction` (around Gate 3):
  ```go
  if time.Since(a.CreatedAt) > 2*time.Hour {
      return nil, ErrActionExpired
  }
  ```

- [ ] **Step 4: Run test to verify it passes**
  Run: `cd backend-go && go test -v -run TestService_ExecuteAction_Expired ./internal/ai`
  Expected: PASS

- [ ] **Step 5: Commit**
  Run: `git add backend-go/internal/ai/service.go backend-go/internal/ai/ai_test.go` && `git commit -m "feat(safety): add action execution expiration gate"`

---

### Task 2: Go Backend Sandbox Executor (Go Subprocess)

**Files:**
- Modify: `backend-go/internal/ai/service.go` (implement sandbox case)
- Modify: `backend-go/internal/ai/ai_test.go` (unit tests for sandbox run)

- [ ] **Step 1: Write unit check for sandbox mode**
  Add `TestService_ExecuteAction_Sandbox` in `backend-go/internal/ai/ai_test.go`:
  ```go
  func TestService_ExecuteAction_Sandbox(t *testing.T) {
      db := newTestDB(t)
      svc := NewService(db, testLogger())
      noApproval := false
      a, _ := svc.CreateAction(&CreateActionInput{
          SourceTable: "ai_trace", SourceID: "trc_sb_1", SourceType: "agent_run",
          AgentID: "A2", ActionType: "listing_optimize", Title: "sandbox run",
          RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
          ExecutionMode: "sandbox",
      })

      result, err := svc.ExecuteAction(a.ID, nil, "alice", "")
      if err != nil {
          t.Fatalf("expected sandbox trigger attempt, got: %v", err)
      }
      if result.Status != "executed" && result.Status != "failed" {
          t.Errorf("expected executed/failed status from sandbox run, got: %s", result.Status)
      }
  }
  ```

- [ ] **Step 2: Run test to verify it fails**
  Run: `cd backend-go && go test -v -run TestService_ExecuteAction_Sandbox ./internal/ai`
  Expected: FAIL with "sandbox execution requires a sandbox executor; no sandbox configured".

- [ ] **Step 3: Implement sandbox subprocess runner in Go**
  In `backend-go/internal/ai/service.go`, replace `case "sandbox":` logic with:
  ```go
  case "sandbox":
      // Execute scripts/run_sandbox.sh <action_id> in background/foreground subprocess
      cmd := exec.Command("bash", "../scripts/run_sandbox.sh", fmt.Sprintf("%d", a.ID))
      cmd.Dir = ".."
      output, err := cmd.CombinedOutput()
      if err != nil {
          s.logger.Error("sandbox run failed", zap.Error(err), zap.String("output", string(output)))
          a.Status = "failed"
          s.db.Save(&a)
          return &a, fmt.Errorf("sandbox run failed: %w", err)
      }
      a.Status = "executed"
      s.db.Save(&a)
      return &a, nil
  ```
  *(Note: Include required package imports like `"os/exec"`)*

- [ ] **Step 4: Run test to verify it passes**
  Run: `cd backend-go && go test -v -run TestService_ExecuteAction_Sandbox ./internal/ai`
  Expected: PASS (mocking the script output or letting it run cleanly).

- [ ] **Step 5: Commit**
  Run: `git add backend-go/internal/ai/service.go backend-go/internal/ai/ai_test.go` && `git commit -m "feat(sandbox): implement sandbox executor in Go backend"`

---

### Task 3: Next.js Cockpit Sandbox execution UI (Frontend)

**Files:**
- Modify: `frontend-next/src/app/(main)/owner/page.tsx`
- Modify: `frontend-next/src/components/ui/HighRiskConfirmDialog.tsx`

- [ ] **Step 1: Update HighRiskConfirmDialog parameters**
  Verify `HighRiskConfirmDialog` can render the sandbox mode title and custom details:
  Ensure `environmentMode` parameter is dynamically mapped to "sandbox" | "production" and shows warning background colors correctly.

- [ ] **Step 2: Implement Execution states on Owner page**
  In `frontend-next/src/app/(main)/owner/page.tsx`, update `approveFlow` and `executeAction` API calls:
  When an action with `execution_mode === 1` (sandbox) is executing, show a spin state: `"正在调度沙盒环境并执行..."` and once done, offer a download button linking to `/tmp/reports/pr-{action_id}/playwright-report` (or display test outcome logs).

- [ ] **Step 3: Run linter verification**
  Run: `cd frontend-next && npm run lint`
  Expected: PASS

- [ ] **Step 4: Commit**
  Run: `git add frontend-next/src/app/(main)/owner/page.tsx frontend-next/src/components/ui/HighRiskConfirmDialog.tsx` && `git commit -m "feat(ui): update owner cockpit for sandbox execution support"`
