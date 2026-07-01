package workflow

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
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
