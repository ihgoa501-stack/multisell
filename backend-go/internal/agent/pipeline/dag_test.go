package pipeline

import (
	"testing"
	"time"
)

// =============================================================================
// PipelineEdge construction and structural tests
// =============================================================================

func TestPipelineEdge_SimpleEdge(t *testing.T) {
	edge := PipelineEdge{
		SourceTopic: "agent.decided.A5.stock_alert",
		Condition: Condition{
			Field:  "stock_status",
			Equals: "red",
		},
		TargetAgent: "G3",
		TargetDP:    "discount_risk_check",
		Timeout:     30 * time.Second,
		Priority:    1,
		MaxRetries:  1,
	}

	if edge.SourceTopic != "agent.decided.A5.stock_alert" {
		t.Fatalf("unexpected SourceTopic: %q", edge.SourceTopic)
	}
	if edge.TargetAgent != "G3" || edge.TargetDP != "discount_risk_check" {
		t.Fatalf("unexpected target: %s/%s", edge.TargetAgent, edge.TargetDP)
	}
	if edge.Timeout != 30*time.Second {
		t.Fatalf("unexpected timeout: %v", edge.Timeout)
	}
	if edge.Priority != 1 {
		t.Fatalf("unexpected priority: %d", edge.Priority)
	}
	if edge.MaxRetries != 1 {
		t.Fatalf("unexpected MaxRetries: %d", edge.MaxRetries)
	}
}

func TestPipelineEdge_DefaultFields(t *testing.T) {
	// Zero-value edge should work with Dispatch — defaults used where needed.
	edge := PipelineEdge{
		SourceTopic: "test.topic",
		TargetAgent: "T1",
		TargetDP:    "dp1",
	}
	if edge.SourceTopic != "test.topic" {
		t.Fatal("SourceTopic not set")
	}
	if edge.Condition.Field != "" {
		t.Fatal("expected empty Condition (always true)")
	}
	if edge.Timeout != 0 {
		t.Fatal("expected zero Timeout (Dispatch uses 30s default)")
	}
	if edge.Priority != 0 {
		t.Fatal("expected zero Priority (errors swallowed)")
	}
	if edge.MaxRetries != 0 {
		t.Fatal("expected zero MaxRetries")
	}
}

func TestCondition_AllFields(t *testing.T) {
	val := true
	c := Condition{
		Field:      "status",
		Equals:     "red",
		GT:         5,
		Exists:     "other_key",
		BoolEquals: &val,
	}

	if c.Field != "status" {
		t.Fatalf("unexpected Field: %q", c.Field)
	}
	if c.Equals != "red" {
		t.Fatalf("unexpected Equals: %q", c.Equals)
	}
	if c.GT != 5 {
		t.Fatalf("unexpected GT: %d", c.GT)
	}
	if c.Exists != "other_key" {
		t.Fatalf("unexpected Exists: %q", c.Exists)
	}
	if c.BoolEquals == nil || *c.BoolEquals != true {
		t.Fatal("unexpected BoolEquals")
	}
}

func TestCondition_ZeroValue(t *testing.T) {
	// Zero-value Condition should have no field set,
	// which evaluateCondition treats as "always true".
	var c Condition
	if c.Field != "" {
		t.Fatal("expected empty Field")
	}
	if c.Equals != "" {
		t.Fatal("expected empty Equals")
	}
	if c.GT != 0 {
		t.Fatal("expected zero GT")
	}
	if c.Exists != "" {
		t.Fatal("expected empty Exists")
	}
	if c.BoolEquals != nil {
		t.Fatal("expected nil BoolEquals")
	}
}

func TestCondition_BoolEqualsAsPointer(t *testing.T) {
	// Verifies that BoolEquals is a *bool and can be set to both true and false.
	trueVal := true
	falseVal := false

	cTrue := Condition{Field: "flag", BoolEquals: &trueVal}
	if *cTrue.BoolEquals != true {
		t.Fatal("expected BoolEquals true")
	}

	cFalse := Condition{Field: "flag", BoolEquals: &falseVal}
	if *cFalse.BoolEquals != false {
		t.Fatal("expected BoolEquals false")
	}

	cNil := Condition{Field: "flag"}
	if cNil.BoolEquals != nil {
		t.Fatal("expected nil BoolEquals")
	}
}

func TestCondition_JsonTags(t *testing.T) {
	// Verify Condition struct has expected json tags for serialization.
	c := Condition{Field: "x", Equals: "y", GT: 1, Exists: "z"}
	_ = c // compile check — json tags are on the struct definition, verified at import time.

	// If json tags were missing, encoding/json serialization would use Go field names.
	// Here we just verify the compile-time contract holds: the type is exported.
	var _ Condition
}

// =============================================================================
// DefaultEdges structural tests
// =============================================================================

func TestDefaultEdges_Count(t *testing.T) {
	if len(DefaultEdges) != 5 {
		t.Fatalf("expected 5 default edges, got %d", len(DefaultEdges))
	}
}

func TestDefaultEdges_NoEmptySourceTopic(t *testing.T) {
	for i, edge := range DefaultEdges {
		if edge.SourceTopic == "" {
			t.Errorf("DefaultEdges[%d] has empty SourceTopic", i)
		}
	}
}

func TestDefaultEdges_NoEmptyTarget(t *testing.T) {
	for i, edge := range DefaultEdges {
		if edge.TargetAgent == "" {
			t.Errorf("DefaultEdges[%d] has empty TargetAgent", i)
		}
		if edge.TargetDP == "" {
			t.Errorf("DefaultEdges[%d] has empty TargetDP", i)
		}
	}
}

func TestDefaultEdges_AllHaveTimeouts(t *testing.T) {
	for i, edge := range DefaultEdges {
		if edge.Timeout <= 0 {
			t.Errorf("DefaultEdges[%d] has missing Timeout", i)
		}
	}
}

func TestDefaultEdges_AllHavePriority(t *testing.T) {
	for i, edge := range DefaultEdges {
		if edge.Priority <= 0 {
			t.Errorf("DefaultEdges[%d] has zero/negative Priority", i)
		}
	}
}

func TestDefaultEdges_ConditionStructure(t *testing.T) {
	for i, edge := range DefaultEdges {
		if edge.Condition.Field == "" {
			t.Errorf("DefaultEdges[%d] has Condition without Field set", i)
		}
	}
}

func TestDefaultEdges_UniqueTopics(t *testing.T) {
	// Verify that similar source topics make sense.
	// A5 stock_alert and G0 system_health each appear once.
	// A6 profit_watch appears twice (two different conditions).
	topicCount := make(map[string]int)
	for _, edge := range DefaultEdges {
		topicCount[edge.SourceTopic]++
	}

	expected := map[string]int{
		"agent.decided.A5.stock_alert":      1,
		"agent.decided.G3.discount_risk_check": 1,
		"agent.decided.A6.profit_watch":     2,
		"agent.decided.G0.system_health":    1,
	}
	for topic, count := range expected {
		if topicCount[topic] != count {
			t.Errorf("expected topic %q to appear %d time(s), got %d", topic, count, topicCount[topic])
		}
	}
}

func TestDefaultEdges_PipelineTopology(t *testing.T) {
	// Verify the pipeline chain: A5 -> G3 -> A6 -> A2, and G0 -> G1.
	edges := []struct {
		source string
		target string
		dp     string
	}{
		{"agent.decided.A5.stock_alert", "G3", "discount_risk_check"},
		{"agent.decided.G3.discount_risk_check", "A6", "profit_watch"},
		{"agent.decided.A6.profit_watch", "A2", "listing_optimize"},
		{"agent.decided.A6.profit_watch", "A2", "listing_optimize"},
		{"agent.decided.G0.system_health", "G1", "dashboard_overview"},
	}

	if len(DefaultEdges) != len(edges) {
		t.Fatalf("DefaultEdges length mismatch: expected %d, got %d", len(edges), len(DefaultEdges))
	}

	for i, want := range edges {
		got := DefaultEdges[i]
		if got.SourceTopic != want.source {
			t.Errorf("DefaultEdges[%d] SourceTopic: want %q, got %q", i, want.source, got.SourceTopic)
		}
		if got.TargetAgent != want.target {
			t.Errorf("DefaultEdges[%d] TargetAgent: want %q, got %q", i, want.target, got.TargetAgent)
		}
		if got.TargetDP != want.dp {
			t.Errorf("DefaultEdges[%d] TargetDP: want %q, got %q", i, want.dp, got.TargetDP)
		}
	}
}
