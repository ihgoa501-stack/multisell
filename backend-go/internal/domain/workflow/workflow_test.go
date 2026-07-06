package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

func TestCreateDefAndList(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{
		Name:        "test survey",
		Description: "market research pipeline",
		Steps:       `[{"name":"research","type":"agent","agent_id":"A12","timeout_seconds":60}]`,
	}
	if err := e.CreateDef(context.Background(), def); err != nil {
		t.Fatalf("CreateDef: %v", err)
	}
	if def.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	defs, err := e.ListDefs(context.Background())
	if err != nil {
		t.Fatalf("ListDefs: %v", err)
	}
	if len(defs) != 1 {
		t.Errorf("expected 1 def, got %d", len(defs))
	}
}

func TestStartRunAndComplete(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	// Agent-type step with no AI orchestrator — returns immediately with input as output.
	def := &WorkflowDef{
		Name:  "two-step test",
		Steps: `[{"name":"step_a","type":"agent","timeout_seconds":1},{"name":"step_b","type":"agent","timeout_seconds":1}]`,
	}
	if err := e.CreateDef(context.Background(), def); err != nil {
		t.Fatalf("CreateDef: %v", err)
	}

	run, err := e.StartRun(context.Background(), def.ID, nil)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if run.Status != RunStatusRunning {
		t.Errorf("expected running, got %s", run.Status)
	}

	// Both goroutines auto-execute instantly (no real AI), then run completes.
	<-time.After(100 * time.Millisecond)

	var completedRun WorkflowRun
	db.First(&completedRun, run.ID)
	if completedRun.Status != RunStatusCompleted {
		t.Errorf("expected completed, got %s", completedRun.Status)
	}
}

func TestStartRunAndFail(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	// Event step blocks until timeout — simulate failure via AdvanceStep.
	def := &WorkflowDef{
		Name:  "fail-test",
		Steps: `[{"name":"failing","type":"event","wait_for_event":"manual","timeout_seconds":1}]`,
	}
	e.CreateDef(context.Background(), def)

	run, _ := e.StartRun(context.Background(), def.ID, nil)

	if err := e.AdvanceStep(context.Background(), run.ID, "failing", nil, &stepError{"oops"}); err == nil {
		t.Fatal("expected error on failed step")
	}

	var failed WorkflowRun
	db.First(&failed, run.ID)
	if failed.Status != RunStatusFailed {
		t.Errorf("expected failed, got %s", failed.Status)
	}
}

func TestPauseAndResume(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	// Event step blocks — gives us time to pause before it completes.
	def := &WorkflowDef{
		Name:  "pause-test",
		Steps: `[{"name":"step_1","type":"event","wait_for_event":"never","timeout_seconds":5}]`,
	}
	e.CreateDef(context.Background(), def)

	run, _ := e.StartRun(context.Background(), def.ID, nil)

	if err := e.PauseRun(context.Background(), run.ID); err != nil {
		t.Fatalf("PauseRun: %v", err)
	}
	var paused WorkflowRun
	db.First(&paused, run.ID)
	if paused.Status != RunStatusPaused {
		t.Errorf("expected paused, got %s", paused.Status)
	}

	if err := e.ResumeRun(context.Background(), run.ID); err != nil {
		t.Fatalf("ResumeRun: %v", err)
	}
	var resumed WorkflowRun
	db.First(&resumed, run.ID)
	if resumed.Status != RunStatusRunning {
		t.Errorf("expected running, got %s", resumed.Status)
	}
}

func TestStepRetry(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{
		Name:  "retry-test",
		Steps: `[{"name":"flakey","type":"event","wait_for_event":"manual","retry_count":2,"timeout_seconds":1}]`,
	}
	e.CreateDef(context.Background(), def)

	run, _ := e.StartRun(context.Background(), def.ID, nil)

	for i := 1; i <= 2; i++ {
		err := e.AdvanceStep(context.Background(), run.ID, "flakey", nil, &stepError{"fail"})
		if err != nil {
			t.Fatalf("unexpected error on attempt %d: %v", i, err)
		}
		<-time.After(50 * time.Millisecond)
	}

	err3 := e.AdvanceStep(context.Background(), run.ID, "flakey", nil, &stepError{"fail_3"})
	if err3 == nil {
		t.Fatal("expected error after exhausting retries")
	}

	var sr WorkflowStepRun
	db.Where("workflow_run_id = ?", run.ID).First(&sr)
	if sr.Attempt != 3 {
		t.Errorf("expected 3 attempts, got %d", sr.Attempt)
	}
}

func TestWorkflowDefCRUD(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{Name: "crud-test", Steps: `[{"name":"s1","type":"agent"}]`}
	e.CreateDef(context.Background(), def)

	got, err := e.GetDef(context.Background(), def.ID)
	if err != nil {
		t.Fatalf("GetDef: %v", err)
	}
	if got.Name != "crud-test" {
		t.Errorf("expected 'crud-test', got '%s'", got.Name)
	}

	got.Name = "updated-test"
	e.UpdateDef(context.Background(), got)
	got2, _ := e.GetDef(context.Background(), def.ID)
	if got2.Name != "updated-test" {
		t.Errorf("expected 'updated-test', got '%s'", got2.Name)
	}

	e.DeleteDef(context.Background(), def.ID)
	defs, _ := e.ListDefs(context.Background())
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
}

func TestForkJoin(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	// Use event steps so we control execution via AdvanceStep.
	def := &WorkflowDef{
		Name: "fork-test",
		Steps: mustPrettyJSON([]StepDef{
			{Name: "prepare", Type: StepTypeEvent, WaitForEvent: "advance", TimeoutSeconds: 5},
			{Name: "parallel_work", Type: StepTypeFork, TimeoutSeconds: 5,
				Forks: []StepDef{
					{Name: "worker_1", Type: StepTypeEvent, WaitForEvent: "advance", TimeoutSeconds: 5},
					{Name: "worker_2", Type: StepTypeEvent, WaitForEvent: "advance", TimeoutSeconds: 5},
				}},
			{Name: "summarize", Type: StepTypeJoin, TimeoutSeconds: 10, JoinSteps: []string{"worker_1", "worker_2"}},
		}),
	}
	e.CreateDef(context.Background(), def)
	run, err := e.StartRun(context.Background(), def.ID, nil)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}

	// Step through manually.
	e.AdvanceStep(context.Background(), run.ID, "prepare", nil, nil)
	<-time.After(50 * time.Millisecond)
	e.AdvanceStep(context.Background(), run.ID, "worker_1", nil, nil)
	e.AdvanceStep(context.Background(), run.ID, "worker_2", nil, nil)
	<-time.After(50 * time.Millisecond)
	e.AdvanceStep(context.Background(), run.ID, "summarize", nil, nil)
	<-time.After(50 * time.Millisecond)

	var final RunResult
	db.Raw("SELECT status FROM workflow_run WHERE id = ?", run.ID).Scan(&final)
	if final.Status != RunStatusCompleted {
		t.Errorf("expected completed, got %s", final.Status)
	}
}

type RunResult struct {
	Status string
}

func mustPrettyJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// ── Condition expression tests ───────────────────────────────────────────────

func TestEvalCondition_stepStatusEq(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{
		Name:  "condition-test",
		Steps: `[{"name":"step_a","type":"agent","timeout_seconds":1},{"name":"step_b","type":"agent","timeout_seconds":1}]`,
	}
	e.CreateDef(context.Background(), def)
	run, _ := e.StartRun(context.Background(), def.ID, nil)
	<-time.After(50 * time.Millisecond)

	// step_a should be completed.
	ok, err := e.evalCondition(context.Background(), run.ID, `$steps.step_a.status == "completed"`)
	if err != nil {
		t.Fatalf("evalCondition: %v", err)
	}
	if !ok {
		t.Error("expected step_a.status == 'completed' to be true")
	}
}

func TestEvalCondition_stepStatusNe(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{
		Name:  "condition-ne-test",
		Steps: `[{"name":"step_a","type":"event","wait_for_event":"never","timeout_seconds":1}]`,
	}
	e.CreateDef(context.Background(), def)
	run, _ := e.StartRun(context.Background(), def.ID, nil)

	ok, err := e.evalCondition(context.Background(), run.ID, `$steps.step_a.status != "completed"`)
	if err != nil {
		t.Fatalf("evalCondition: %v", err)
	}
	if !ok {
		t.Error("expected step_a.status != 'completed' to be true when still running")
	}
}

func TestEvalCondition_invalidFormat(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	_, err := e.evalCondition(context.Background(), 0, "not valid")
	if err == nil {
		t.Fatal("expected error for invalid condition format")
	}
}

// ── Parallel fork tests ─────────────────────────────────────────────────────

func TestForkParallel(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	// Use event steps for reliable SQLite testing — we control timing via AdvanceStep.
	def := &WorkflowDef{
		Name: "parallel-fork-test",
		Steps: mustPrettyJSON([]StepDef{
			{Name: "start", Type: StepTypeEvent, WaitForEvent: "advance", TimeoutSeconds: 5},
			{Name: "fork_step", Type: StepTypeFork, TimeoutSeconds: 5,
				Forks: []StepDef{
					{Name: "worker_a", Type: StepTypeEvent, WaitForEvent: "advance", TimeoutSeconds: 5},
					{Name: "worker_b", Type: StepTypeEvent, WaitForEvent: "advance", TimeoutSeconds: 5},
					{Name: "worker_c", Type: StepTypeEvent, WaitForEvent: "advance", TimeoutSeconds: 5},
				}},
			{Name: "end", Type: StepTypeJoin, TimeoutSeconds: 10, JoinSteps: []string{"worker_a", "worker_b", "worker_c"}},
		}),
	}
	e.CreateDef(context.Background(), def)
	run, _ := e.StartRun(context.Background(), def.ID, nil)

	// Advance start step manually.
	e.AdvanceStep(context.Background(), run.ID, "start", nil, nil)
	<-time.After(50 * time.Millisecond)

	// Advance each worker.
	e.AdvanceStep(context.Background(), run.ID, "worker_a", nil, nil)
	e.AdvanceStep(context.Background(), run.ID, "worker_b", nil, nil)
	e.AdvanceStep(context.Background(), run.ID, "worker_c", nil, nil)
	<-time.After(50 * time.Millisecond)

	// Advance join.
	e.AdvanceStep(context.Background(), run.ID, "end", nil, nil)
	<-time.After(50 * time.Millisecond)

	var final WorkflowRun
	db.First(&final, run.ID)
	if final.Status != RunStatusCompleted {
		t.Errorf("expected completed, got %s", final.Status)
	}
}

// ── PublishEvent encapsulation tests ─────────────────────────────────────────

func TestPublishEvent_noBus(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	id, err := e.PublishEvent(context.Background(), "test.topic", "workflow", map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("PublishEvent with nil bus: %v", err)
	}
	if id != "" {
		t.Errorf("expected empty ID with nil bus, got %s", id)
	}
}

func TestPublishEvent_withBus(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{})
	bus := eventbus.New(dbtest.NewLogger(t))
	e := NewEngine(db, bus, nil, nil, dbtest.NewLogger(t))

	var received map[string]interface{}
	bus.Subscribe("test.topic", func(ctx context.Context, evt eventbus.Event) error {
		received = evt.Payload
		return nil
	})
	go bus.Start(context.Background())
	defer bus.Stop()

	id, err := e.PublishEvent(context.Background(), "test.topic", "workflow", map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("PublishEvent: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty event ID")
	}

	<-time.After(50 * time.Millisecond)
	if received == nil {
		t.Fatal("expected event to be received by subscriber")
	}
	if received["key"] != "val" {
		t.Errorf("expected key=val, got %v", received["key"])
	}
}

// ── Condition step in workflow ──────────────────────────────────────────────

func TestWorkflowWithConditionalStep(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	// step_a completes immediately, step_b has condition: only run if step_a completed.
	def := &WorkflowDef{
		Name: "conditional-test",
		Steps: mustPrettyJSON([]StepDef{
			{Name: "step_a", Type: StepTypeAgent, TimeoutSeconds: 1},
			{Name: "step_b", Type: StepTypeAgent, TimeoutSeconds: 1, Condition: `$steps.step_a.status == "completed"`},
		}),
	}
	e.CreateDef(context.Background(), def)
	run, _ := e.StartRun(context.Background(), def.ID, nil)
	<-time.After(100 * time.Millisecond)

	var final WorkflowRun
	db.First(&final, run.ID)
	if final.Status != RunStatusCompleted {
		t.Errorf("expected completed, got %s", final.Status)
	}
}

// ── Monitor stats test ──────────────────────────────────────────────────────

func TestGetMonitorStats(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	stats, err := e.GetMonitorStats(context.Background())
	if err != nil {
		t.Fatalf("GetMonitorStats: %v", err)
	}
	if stats.TotalRuns != 0 {
		t.Errorf("expected 0 total runs, got %d", stats.TotalRuns)
	}
	if stats.AverageDurationS != 0 {
		t.Errorf("expected 0 avg duration, got %f", stats.AverageDurationS)
	}

	// Create a run and check stats again.
	def := &WorkflowDef{
		Name:  "stats-test",
		Steps: `[{"name":"s1","type":"agent","timeout_seconds":1}]`,
	}
	e.CreateDef(context.Background(), def)
	e.StartRun(context.Background(), def.ID, nil)
	<-time.After(100 * time.Millisecond)

	stats, _ = e.GetMonitorStats(context.Background())
	if stats.TotalRuns != 1 {
		t.Errorf("expected 1 total run, got %d", stats.TotalRuns)
	}
}

// ── M5.2: Node CRUD tests ──────────────────────────────────────────────

func TestCreateAndListNode(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowNode{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{Name: "node-test", Steps: "[]"}
	if err := e.CreateDef(context.Background(), def); err != nil {
		t.Fatalf("CreateDef: %v", err)
	}

	node := &WorkflowNode{
		WorkflowID: uint(def.ID),
		Type:       "approval",
		Config:     json.RawMessage(`{"timeout_seconds":300}`),
		OrderIndex: 0,
	}
	if err := e.CreateNode(context.Background(), node); err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if node.ID == 0 {
		t.Fatal("expected non-zero node ID")
	}

	nodes, err := e.ListNodes(context.Background(), uint(def.ID))
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Type != "approval" {
		t.Errorf("expected type 'approval', got '%s'", nodes[0].Type)
	}
}

func TestListNodesForEmptyWorkflow(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowNode{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	nodes, err := e.ListNodes(context.Background(), 999)
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for non-existent workflow, got %d", len(nodes))
	}
}

// ── M5.2: Condition step execution tests ──────────────────────────────

func TestExecCondition(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{
		Name:  "cond-exec-test",
		Steps: `[{"name":"s1","type":"agent","timeout_seconds":1},{"name":"s2","type":"condition","condition":"$steps.s1.status == \"completed\"","timeout_seconds":1},{"name":"s3","type":"condition","condition":"$steps.s1.status == \"failed\"","timeout_seconds":1}]`,
	}
	e.CreateDef(context.Background(), def)
	run, _ := e.StartRun(context.Background(), def.ID, nil)
	<-time.After(200 * time.Millisecond)

	var sr2, sr3 WorkflowStepRun
	db.Where("workflow_run_id = ? AND step_name = ?", run.ID, "s2").First(&sr2)
	db.Where("workflow_run_id = ? AND step_name = ?", run.ID, "s3").First(&sr3)

	if sr2.Status != StepStatusCompleted {
		t.Errorf("expected s2 to complete (condition true), got %s", sr2.Status)
	}
	if sr3.Status != StepStatusSkipped {
		t.Errorf("expected s3 to be skipped (condition false), got %s", sr3.Status)
	}

	var final WorkflowRun
	db.First(&final, run.ID)
	if final.Status != RunStatusCompleted {
		t.Errorf("expected run completed, got %s", final.Status)
	}
}

// ── M5.2: Approval step execution tests ──────────────────────────────

func TestApprovalApprove(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{
		Name:  "approve-test",
		Steps: `[{"name":"ask","type":"approval","timeout_seconds":5},{"name":"done","type":"agent","timeout_seconds":1}]`,
	}
	e.CreateDef(context.Background(), def)
	run, _ := e.StartRun(context.Background(), def.ID, nil)

	// approval step should be pending — signal approval.
	<-time.After(50 * time.Millisecond)
	if err := e.ApproveStep(context.Background(), run.ID, "ask", "admin", "looks good"); err != nil {
		t.Fatalf("ApproveStep: %v", err)
	}
	<-time.After(50 * time.Millisecond)

	var final WorkflowRun
	db.First(&final, run.ID)
	if final.Status != RunStatusCompleted {
		t.Errorf("expected completed after approval, got %s", final.Status)
	}
}

func TestApprovalReject(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{
		Name:  "reject-test",
		Steps: `[{"name":"ask","type":"approval","timeout_seconds":5}]`,
	}
	e.CreateDef(context.Background(), def)
	run, _ := e.StartRun(context.Background(), def.ID, nil)

	<-time.After(50 * time.Millisecond)
	if err := e.RejectStep(context.Background(), run.ID, "ask", "admin", "not approved"); err != nil {
		t.Fatalf("RejectStep: %v", err)
	}
	<-time.After(50 * time.Millisecond)

	// Rejected step should cause workflow failure.
	var final WorkflowRun
	db.First(&final, run.ID)
	if final.Status != RunStatusFailed {
		t.Errorf("expected failed after rejection, got %s", final.Status)
	}
}

func TestApprovalInvalidStep(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	err := e.ApproveStep(context.Background(), 0, "nonexistent", "admin", "")
	if err == nil {
		t.Fatal("expected error for non-existent approval step")
	}
}

// ── M5.1: Paginated list tests ────────────────────────────────────────

func TestListDefsPaginated(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowNode{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	// Insert 5 defs.
	for i := 0; i < 5; i++ {
		e.CreateDef(context.Background(), &WorkflowDef{
			Name:  fmt.Sprintf("def-%d", i),
			Steps: "[]",
		})
	}

	// Page 1, size 3.
	defs, total, err := e.ListDefsPaginated(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("ListDefsPaginated: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(defs) != 3 {
		t.Errorf("expected 3 defs on page 1, got %d", len(defs))
	}

	// Page 2, size 3.
	defs2, total2, _ := e.ListDefsPaginated(context.Background(), 2, 3)
	if len(defs2) != 2 {
		t.Errorf("expected 2 defs on page 2, got %d", len(defs2))
	}
	if total2 != 5 {
		t.Errorf("expected total=5, got %d", total2)
	}
}

// ── M5.2: Action step execution test ──────────────────────────────────

func TestExecActionStep(t *testing.T) {
	db := dbtest.NewDB(t, &WorkflowDef{}, &WorkflowRun{}, &WorkflowStepRun{})
	e := NewEngine(db, nil, nil, nil, dbtest.NewLogger(t))

	def := &WorkflowDef{
		Name:  "action-test",
		Steps: `[{"name":"act","type":"action","timeout_seconds":1}]`,
	}
	e.CreateDef(context.Background(), def)
	run, _ := e.StartRun(context.Background(), def.ID, nil)
	<-time.After(100 * time.Millisecond)

	var final WorkflowRun
	db.First(&final, run.ID)
	if final.Status != RunStatusCompleted {
		t.Errorf("expected completed for action step, got %s", final.Status)
	}
}
