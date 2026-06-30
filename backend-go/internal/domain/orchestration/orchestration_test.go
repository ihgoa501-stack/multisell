package orchestration

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

// ---------------------------------------------------------------------------
// Pure function tests
// ---------------------------------------------------------------------------

func TestParseStepsJSON(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{`["a","b","c"]`, []string{"a", "b", "c"}},
		{`[]`, []string{}},
		{``, nil},
		{`invalid`, nil},
	}
	for _, tc := range tests {
		got, _ := parseStepsJSON(tc.input)
		if tc.want == nil {
			if got != nil {
				t.Errorf("parseStepsJSON(%q) = %v, want nil", tc.input, got)
			}
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("parseStepsJSON(%q) len=%d, want %d", tc.input, len(got), len(tc.want))
			continue
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("parseStepsJSON(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

func TestExtractProductID(t *testing.T) {
	evt := eventbus.Event{Payload: map[string]interface{}{"product_id": float64(42)}}
	if id := extractProductID(evt); id != 42 {
		t.Errorf("expected 42, got %d", id)
	}

	evt = eventbus.Event{Payload: map[string]interface{}{}}
	if id := extractProductID(evt); id != 0 {
		t.Errorf("expected 0 for missing product_id, got %d", id)
	}
}

func TestStepAgentMapping_CoversDefaultPipeline(t *testing.T) {
	for _, step := range DefaultPipeline {
		if _, ok := stepAgentMapping[step]; !ok {
			t.Errorf("DefaultPipeline step %q has no agent mapping", step)
		}
	}
}

func TestDefaultPipeline_NotEmpty(t *testing.T) {
	if len(DefaultPipeline) == 0 {
		t.Fatal("DefaultPipeline is empty")
	}
	for _, s := range []string{StepStatusPending, StepStatusRunning, StepStatusCompleted, StepStatusFailed, StepStatusSkipped} {
		if s == "" {
			t.Error("empty status constant")
		}
	}
}

// ---------------------------------------------------------------------------
// Pipeline service tests
// ---------------------------------------------------------------------------

func newTestOrchestrator(t *testing.T) *PipelineOrchestrator {
	t.Helper()
	db := dbtest.NewDB(t, &LifecycleStep{}, &OrchestrationConfig{})
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	return NewPipelineOrchestrator(db, bus, nil, logger)
}

func TestStartPipeline_CreatesSteps(t *testing.T) {
	o := newTestOrchestrator(t)

	err := o.StartPipeline(context.Background(), 1)
	if err != nil {
		t.Fatalf("StartPipeline: %v", err)
	}

	var steps []LifecycleStep
	o.db.Where("product_id = ?", 1).Order("id ASC").Find(&steps)

	if len(steps) != len(DefaultPipeline) {
		t.Fatalf("expected %d steps, got %d", len(DefaultPipeline), len(steps))
	}
	if steps[0].Status != StepStatusRunning {
		t.Errorf("first step expected running, got %s", steps[0].Status)
	}
	if steps[0].StartedAt == nil {
		t.Error("first step should have started_at set")
	}
	for i := 1; i < len(steps); i++ {
		if steps[i].Status != StepStatusPending {
			t.Errorf("step %d expected pending, got %s", i, steps[i].Status)
		}
	}
	for i, s := range DefaultPipeline {
		if steps[i].Step != s {
			t.Errorf("step %d: expected %q, got %q", i, s, steps[i].Step)
		}
	}
}

func TestStartPipeline_Duplicate(t *testing.T) {
	o := newTestOrchestrator(t)
	o.StartPipeline(context.Background(), 1)
	err := o.StartPipeline(context.Background(), 1)
	if err != nil {
		t.Errorf("second StartPipeline should succeed, got: %v", err)
	}
}

func TestGetPipelineStatus(t *testing.T) {
	o := newTestOrchestrator(t)
	o.StartPipeline(context.Background(), 1)

	steps, err := o.GetPipelineStatus(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetPipelineStatus: %v", err)
	}
	if len(steps) != len(DefaultPipeline) {
		t.Errorf("expected %d steps, got %d", len(DefaultPipeline), len(steps))
	}
}

func TestGetPipelineStatus_NotFound(t *testing.T) {
	o := newTestOrchestrator(t)
	steps, err := o.GetPipelineStatus(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetPipelineStatus: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
}

func TestAdvancePipeline_Success(t *testing.T) {
	o := newTestOrchestrator(t)
	o.StartPipeline(context.Background(), 1)

	err := o.AdvancePipeline(context.Background(), 1, "sourcing", true)
	if err != nil {
		t.Fatalf("AdvancePipeline (sourcing): %v", err)
	}

	var steps []LifecycleStep
	o.db.Where("product_id = ?", 1).Order("id ASC").Find(&steps)

	if steps[0].Status != StepStatusCompleted {
		t.Errorf("step 0 expected completed, got %s", steps[0].Status)
	}
	if steps[0].CompletedAt == nil {
		t.Error("step 0 should have completed_at set")
	}
	if steps[1].Status != StepStatusRunning {
		t.Errorf("step 1 expected running, got %s", steps[1].Status)
	}
}

func TestAdvancePipeline_StepNotFound(t *testing.T) {
	o := newTestOrchestrator(t)
	o.StartPipeline(context.Background(), 1)

	err := o.AdvancePipeline(context.Background(), 1, "nonexistent_step", true)
	if err == nil {
		t.Fatal("expected error for unknown step")
	}
}

func TestAdvancePipeline_AllStepsComplete(t *testing.T) {
	o := newTestOrchestrator(t)
	o.StartPipeline(context.Background(), 1)

	for _, step := range DefaultPipeline {
		err := o.AdvancePipeline(context.Background(), 1, step, true)
		if err != nil {
			t.Fatalf("AdvancePipeline(%s): %v", step, err)
		}
	}

	// Last step completes the pipeline (no error for completing past the end)
	err := o.AdvancePipeline(context.Background(), 1, "delisting", true)
	if err != nil {
		t.Fatalf("AdvancePipeline(delisting): %v", err)
	}

	var steps []LifecycleStep
	o.db.Where("product_id = ?", 1).Order("id ASC").Find(&steps)
	for i, s := range steps {
		if s.Status != StepStatusCompleted {
			t.Errorf("step %d (%s) expected completed, got %s", i, s.Step, s.Status)
		}
	}
}

func TestAdvancePipeline_NoStepsForProduct(t *testing.T) {
	o := newTestOrchestrator(t)

	err := o.AdvancePipeline(context.Background(), 999, "sourcing", true)
	if err == nil {
		t.Fatal("expected error when no steps exist")
	}
}

func TestAdvancePipeline_FailureStop(t *testing.T) {
	o := newTestOrchestrator(t)
	o.StartPipeline(context.Background(), 1)

	// Failure with default FailureAction=stop: no triggerAgent call, no error
	err := o.AdvancePipeline(context.Background(), 1, "sourcing", false)
	if err != nil {
		t.Fatalf("AdvancePipeline(fail): %v", err)
	}

	var step LifecycleStep
	o.db.Where("product_id = ? AND step = ?", 1, "sourcing").First(&step)
	if step.Status != StepStatusFailed {
		t.Errorf("expected failed, got %s", step.Status)
	}

	// Next step should NOT start
	var next LifecycleStep
	o.db.Where("product_id = ? AND step = ?", 1, "enrichment").First(&next)
	if next.Status == StepStatusRunning {
		t.Errorf("enrichment should not be running on failure")
	}
}

func TestConfig_CreateAndList(t *testing.T) {
	o := newTestOrchestrator(t)

	cfg := &OrchestrationConfig{
		Name:          "test-cfg",
		Steps:         `["sourcing", "pricing", "listing"]`,
		FailureAction: FailureActionSkip,
	}
	err := o.CreateConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("CreateConfig: %v", err)
	}
	if cfg.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	configs, err := o.ListConfigs(context.Background())
	if err != nil {
		t.Fatalf("ListConfigs: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Name != "test-cfg" {
		t.Errorf("expected name test-cfg, got %s", configs[0].Name)
	}
}

func TestConfig_ParsedStepsOverridePipeline(t *testing.T) {
	o := newTestOrchestrator(t)

	o.CreateConfig(context.Background(), &OrchestrationConfig{
		Name:  "compact",
		Steps: `["sourcing", "pricing"]`,
	})

	err := o.StartPipeline(context.Background(), 2)
	if err != nil {
		t.Fatalf("StartPipeline with config: %v", err)
	}

	var steps []LifecycleStep
	o.db.Where("product_id = ?", 2).Order("id ASC").Find(&steps)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps from config, got %d", len(steps))
	}
	if steps[0].Step != "sourcing" || steps[1].Step != "pricing" {
		t.Errorf("unexpected steps: wanted sourcing, pricing")
	}
}

func TestRetryStep(t *testing.T) {
	o := newTestOrchestrator(t)
	o.StartPipeline(context.Background(), 1)

	// agentOrch is nil → triggerAgent logs and skips → no error
	err := o.RetryStep(context.Background(), 1, "sourcing")
	if err != nil {
		t.Fatalf("RetryStep: %v", err)
	}
}

func TestDurationMs_SetOnCompletion(t *testing.T) {
	o := newTestOrchestrator(t)
	o.StartPipeline(context.Background(), 1)

	o.AdvancePipeline(context.Background(), 1, "sourcing", true)

	var step LifecycleStep
	o.db.Where("product_id = ? AND step = ?", 1, "sourcing").First(&step)
	if step.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
	if step.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", step.DurationMs)
	}
}

// ---------------------------------------------------------------------------
// Subscriber tests
// ---------------------------------------------------------------------------

func TestExtractProductID_FromEvent(t *testing.T) {
	evt := eventbus.Event{
		Payload: map[string]interface{}{"product_id": float64(99)},
	}
	if id := extractProductID(evt); id != 99 {
		t.Errorf("expected 99, got %d", id)
	}

	evt = eventbus.Event{
		Payload: map[string]interface{}{"order_id": float64(5)},
	}
	if id := extractProductID(evt); id != 0 {
		t.Errorf("expected 0 for missing product_id, got %d", id)
	}
}
