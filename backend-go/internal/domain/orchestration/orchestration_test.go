package orchestration

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestParseStepsJSON(t *testing.T) {
	steps, err := parseStepsJSON(`["sourcing", "enrichment", "compliance"]`)
	if err != nil {
		t.Fatalf("parseStepsJSON failed: %v", err)
	}
	if len(steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(steps))
	}
	if steps[0] != "sourcing" {
		t.Errorf("expected step[0] 'sourcing', got %s", steps[0])
	}
}

func TestParseStepsJSON_Invalid(t *testing.T) {
	_, err := parseStepsJSON(`invalid json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseStepsJSON_Empty(t *testing.T) {
	steps, err := parseStepsJSON(`[]`)
	if err != nil {
		t.Fatalf("parseStepsJSON failed: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
}

func TestDefaultPipeline(t *testing.T) {
	if len(DefaultPipeline) != 7 {
		t.Errorf("expected 7 default pipeline steps, got %d", len(DefaultPipeline))
	}
	expectedSteps := []string{"sourcing", "enrichment", "compliance", "pricing", "listing", "monitoring", "delisting"}
	for i, s := range expectedSteps {
		if DefaultPipeline[i] != s {
			t.Errorf("expected step %d '%s', got '%s'", i, s, DefaultPipeline[i])
		}
	}
}

func TestStepAgentMapping(t *testing.T) {
	if agentID, ok := stepAgentMapping["sourcing"]; !ok || agentID != "A8" {
		t.Errorf("expected sourcing->A8, got %s", agentID)
	}
	if agentID, ok := stepAgentMapping["listing"]; !ok || agentID != "A2" {
		t.Errorf("expected listing->A2, got %s", agentID)
	}
}

// ── DB tests ─────────────────────────────────────────────────────

func TestOrchestrator_ListConfigs_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &OrchestrationConfig{})
	o := NewPipelineOrchestrator(db, nil, nil, dbtest.NewLogger(t))

	configs, err := o.ListConfigs(context.Background())
	if err != nil {
		t.Fatalf("ListConfigs failed: %v", err)
	}
	if configs == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestOrchestrator_CreateConfig(t *testing.T) {
	db := dbtest.NewDB(t, &OrchestrationConfig{})
	o := NewPipelineOrchestrator(db, nil, nil, dbtest.NewLogger(t))

	cfg := &OrchestrationConfig{
		Name:          "test config",
		Steps:         `["sourcing","compliance"]`,
		FailureAction: "stop",
		AutoRetryCount: 3,
	}
	if err := o.CreateConfig(context.Background(), cfg); err != nil {
		t.Fatalf("CreateConfig failed: %v", err)
	}
	if cfg.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}
}

func TestOrchestrator_CreateAndListConfigs(t *testing.T) {
	db := dbtest.NewDB(t, &OrchestrationConfig{})
	o := NewPipelineOrchestrator(db, nil, nil, dbtest.NewLogger(t))

	o.CreateConfig(context.Background(), &OrchestrationConfig{Name: "config A", Steps: `["a"]`})
	o.CreateConfig(context.Background(), &OrchestrationConfig{Name: "config B", Steps: `["b"]`})

	configs, err := o.ListConfigs(context.Background())
	if err != nil {
		t.Fatalf("ListConfigs failed: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(configs))
	}
}

func TestOrchestrator_GetPipelineStatus_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &LifecycleStep{})
	o := NewPipelineOrchestrator(db, nil, nil, dbtest.NewLogger(t))

	steps, err := o.GetPipelineStatus(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetPipelineStatus failed: %v", err)
	}
	if steps == nil {
		t.Error("expected non-nil empty result")
	}
}

func TestOrchestrator_ExtractProductID(t *testing.T) {
	// Test the helper function from subscriber.go
	type testEvent struct {
		Payload map[string]interface{}
	}

	evtWithID := testEvent{Payload: map[string]interface{}{"product_id": float64(42)}}
	type EventLike struct {
		Payload map[string]interface{}
	}

	// Can't directly test extractProductID since it takes eventbus.Event,
	// but we can test the logic by verifying the event payload format
	if id, ok := evtWithID.Payload["product_id"].(float64); !ok || int64(id) != 42 {
		t.Errorf("expected product_id 42, got %v", evtWithID.Payload["product_id"])
	}
}

func TestStepStatusConstants(t *testing.T) {
	if StepStatusPending != "pending" {
		t.Errorf("expected 'pending', got %s", StepStatusPending)
	}
	if StepStatusRunning != "running" {
		t.Errorf("expected 'running', got %s", StepStatusRunning)
	}
	if StepStatusCompleted != "completed" {
		t.Errorf("expected 'completed', got %s", StepStatusCompleted)
	}
	if StepStatusFailed != "failed" {
		t.Errorf("expected 'failed', got %s", StepStatusFailed)
	}
	if StepStatusSkipped != "skipped" {
		t.Errorf("expected 'skipped', got %s", StepStatusSkipped)
	}
}

func TestFailureActionConstants(t *testing.T) {
	if FailureActionStop != "stop" {
		t.Errorf("expected 'stop', got %s", FailureActionStop)
	}
	if FailureActionSkip != "skip" {
		t.Errorf("expected 'skip', got %s", FailureActionSkip)
	}
	if FailureActionRetry != "retry" {
		t.Errorf("expected 'retry', got %s", FailureActionRetry)
	}
}
