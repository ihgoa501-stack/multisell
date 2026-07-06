package pipeline

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

// Shared test helpers
var errTestSimulated = errors.New("simulated error")

// =============================================================================
// matchTopic tests
// =============================================================================

func TestMatchTopic_Exact(t *testing.T) {
	if !matchTopic("agent.decided.A5.stock_alert", "agent.decided.A5.stock_alert") {
		t.Fatal("expected exact match")
	}
}

func TestMatchTopic_Wildcard(t *testing.T) {
	if !matchTopic("agent.decided.A5.stock_alert", "agent.decided.*") {
		t.Fatal("expected wildcard suffix match")
	}
}

func TestMatchTopic_WildcardEdgeOnly(t *testing.T) {
	// Single "*" should match everything.
	if !matchTopic("any.topic", "*") {
		t.Fatal("expected single wildcard to match any topic")
	}
}

func TestMatchTopic_NoMatch(t *testing.T) {
	if matchTopic("agent.decided.A5.stock_alert", "other.topic") {
		t.Fatal("expected no match for unrelated topic")
	}
}

func TestMatchTopic_Empty(t *testing.T) {
	if matchTopic("", "agent.decided.*") {
		t.Fatal("empty event topic should not match")
	}
	if matchTopic("agent.decided.A5.stock_alert", "") {
		t.Fatal("empty edge topic should not match")
	}
	if !matchTopic("", "") {
		t.Fatal("two empty strings compare equal — exact match returns true")
	}
}

func TestMatchTopic_PartialPrefix(t *testing.T) {
	if !matchTopic("x.y.z", "x.y.*") {
		t.Fatal("expected suffix wildcard on multi-part topic")
	}
	if matchTopic("a.b.c", "x.y.*") {
		t.Fatal("expected no match for wrong prefix with wildcard")
	}
}

// =============================================================================
// evaluateCondition tests
// =============================================================================

func TestEvaluateCondition_EmptyField(t *testing.T) {
	if !evaluateCondition(map[string]interface{}{"a": 1}, Condition{}) {
		t.Fatal("empty condition (no field) should return true")
	}
}

func TestEvaluateCondition_Equals(t *testing.T) {
	if !evaluateCondition(
		map[string]interface{}{"status": "red"},
		Condition{Field: "status", Equals: "red"},
	) {
		t.Fatal("expected string equals match")
	}
}

func TestEvaluateCondition_EqualsNoMatch(t *testing.T) {
	if evaluateCondition(
		map[string]interface{}{"status": "green"},
		Condition{Field: "status", Equals: "red"},
	) {
		t.Fatal("expected no match for different string value")
	}
}

func TestEvaluateCondition_EqualsTypeMismatch(t *testing.T) {
	if evaluateCondition(
		map[string]interface{}{"status": 42},
		Condition{Field: "status", Equals: "42"},
	) {
		t.Fatal("expected no match for type mismatch (int vs string)")
	}
}

func TestEvaluateCondition_EqualsEmptyString(t *testing.T) {
	// Empty Equals means the check is skipped, so the condition falls through
	// to next checks. With only Field and empty Equals, no sub-condition matches.
	if evaluateCondition(
		map[string]interface{}{"status": "red"},
		Condition{Field: "status", Equals: ""},
	) {
		t.Fatal("empty Equals should not match (falls through to false)")
	}
}

func TestEvaluateCondition_GT_Int(t *testing.T) {
	if !evaluateCondition(
		map[string]interface{}{"anomaly_count": 10},
		Condition{Field: "anomaly_count", GT: 3},
	) {
		t.Fatal("expected 10 > 3")
	}
}

func TestEvaluateCondition_GT_Float64(t *testing.T) {
	// Float64 comparisons: math.Floor(float64) > GT
	if !evaluateCondition(
		map[string]interface{}{"confidence": 7.5},
		Condition{Field: "confidence", GT: 5},
	) {
		t.Fatal("expected float64 7.5 > 5")
	}

	// Float64 not greater than threshold.
	if evaluateCondition(
		map[string]interface{}{"confidence": 2.0},
		Condition{Field: "confidence", GT: 5},
	) {
		t.Fatal("expected float64 2.0 is not > 5")
	}

	// Float64 exactly at threshold: strictly greater, so equal fails.
	if evaluateCondition(
		map[string]interface{}{"confidence": 5.0},
		Condition{Field: "confidence", GT: 5},
	) {
		t.Fatal("expected float64 5.0 is not > 5 (strict GT)")
	}
}

func TestEvaluateCondition_GT_NoMatch(t *testing.T) {
	if evaluateCondition(
		map[string]interface{}{"anomaly_count": 2},
		Condition{Field: "anomaly_count", GT: 3},
	) {
		t.Fatal("expected 2 is not > 3")
	}
}

func TestEvaluateCondition_GT_Equal(t *testing.T) {
	// GT is strictly greater-than, so equal should not match.
	if evaluateCondition(
		map[string]interface{}{"anomaly_count": 3},
		Condition{Field: "anomaly_count", GT: 3},
	) {
		t.Fatal("expected 3 is not > 3 (strict GT)")
	}
}

func TestEvaluateCondition_GT_Zero(t *testing.T) {
	// GT == 0 means the check is skipped; falls through to false.
	if evaluateCondition(
		map[string]interface{}{"score": 100},
		Condition{Field: "score", GT: 0},
	) {
		t.Fatal("GT=0 should not match (zero is the skip sentinel)")
	}
}

func TestEvaluateCondition_GT_NegativeValue(t *testing.T) {
	if evaluateCondition(
		map[string]interface{}{"anomaly_count": -1},
		Condition{Field: "anomaly_count", GT: 0},
	) {
		t.Fatal("expected -1 is not > 0")
	}
}

func TestEvaluateCondition_BoolEquals_True(t *testing.T) {
	val := true
	if !evaluateCondition(
		map[string]interface{}{"is_loss": true},
		Condition{Field: "is_loss", BoolEquals: &val},
	) {
		t.Fatal("expected bool equals true to match")
	}
}

func TestEvaluateCondition_BoolEquals_False(t *testing.T) {
	val := false
	if !evaluateCondition(
		map[string]interface{}{"is_loss": false},
		Condition{Field: "is_loss", BoolEquals: &val},
	) {
		t.Fatal("expected bool equals false to match")
	}
}

func TestEvaluateCondition_BoolEqualsNoMatch(t *testing.T) {
	val := true
	if evaluateCondition(
		map[string]interface{}{"is_loss": false},
		Condition{Field: "is_loss", BoolEquals: &val},
	) {
		t.Fatal("expected no match for different bool value")
	}
}

func TestEvaluateCondition_BoolEquals_NonBoolValue(t *testing.T) {
	val := true
	if evaluateCondition(
		map[string]interface{}{"is_loss": "true"},
		Condition{Field: "is_loss", BoolEquals: &val},
	) {
		t.Fatal("expected no match for string value with BoolEquals check")
	}
}

func TestEvaluateCondition_Exists(t *testing.T) {
	if !evaluateCondition(
		map[string]interface{}{"a": 1, "b": 2},
		Condition{Field: "a", Exists: "b"},
	) {
		t.Fatal("expected exists check to match when 'b' key is present")
	}
}

func TestEvaluateCondition_ExistsNoMatch(t *testing.T) {
	if evaluateCondition(
		map[string]interface{}{"a": 1},
		Condition{Field: "a", Exists: "b"},
	) {
		t.Fatal("expected no match when 'b' key is missing")
	}
}

func TestEvaluateCondition_MissingField(t *testing.T) {
	if evaluateCondition(
		map[string]interface{}{},
		Condition{Field: "missing", Equals: "val"},
	) {
		t.Fatal("expected no match for missing field")
	}
}

func TestEvaluateCondition_Prioritization(t *testing.T) {
	// BoolEquals has highest priority, then Equals, then Exists, then GT.
	// When all are set: field "x" is "true", BoolEquals should match first.
	val := true
	if !evaluateCondition(
		map[string]interface{}{"x": true, "y": "exists"},
		Condition{Field: "x", Equals: "wrong", Exists: "y", GT: 100, BoolEquals: &val},
	) {
		t.Fatal("expected BoolEquals to take priority over other checks")
	}

	// BoolEquals nil, Equals set: Equals should match.
	if !evaluateCondition(
		map[string]interface{}{"x": "hello"},
		Condition{Field: "x", Equals: "hello", GT: 100},
	) {
		t.Fatal("expected Equals to match when BoolEquals is nil")
	}
}

// =============================================================================
// Engine creation and dispatch tests
// =============================================================================

func TestNewEngine(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return nil
		},
		[]PipelineEdge{},
		newTestLogger(),
	)
	if eng == nil {
		t.Fatal("expected non-nil engine")
	}
	if eng.Breakers() == nil {
		t.Fatal("expected non-nil breakers registry")
	}
}

func TestEngine_Dispatch_SimpleMatch(t *testing.T) {
	var mu sync.Mutex
	type call struct {
		agentID, dp string
		payload     map[string]interface{}
	}
	var calls []call

	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			mu.Lock()
			calls = append(calls, call{agentID, dp, ctxMap})
			mu.Unlock()
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "agent.decided.A5.stock_alert",
				Condition:   Condition{Field: "stock_status", Equals: "red"},
				TargetAgent: "G3",
				TargetDP:    "discount_risk_check",
				Priority:    1,
			},
		},
		newTestLogger(),
	)

	err := eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "agent.decided.A5.stock_alert",
		Payload: map[string]interface{}{"stock_status": "red"},
	})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	mu.Lock()
	if len(calls) != 1 {
		mu.Unlock()
		t.Fatalf("expected 1 dispatch call, got %d", len(calls))
	}
	if calls[0].agentID != "G3" || calls[0].dp != "discount_risk_check" {
		mu.Unlock()
		t.Fatalf("expected G3/discount_risk_check, got %s/%s", calls[0].agentID, calls[0].dp)
	}
	if calls[0].payload["stock_status"] != "red" {
		mu.Unlock()
		t.Fatalf("expected payload stock_status=red, got %v", calls[0].payload)
	}
	mu.Unlock()
}

func TestEngine_Dispatch_ConditionBlocks(t *testing.T) {
	var runCount int
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			runCount++
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "agent.decided.A5.stock_alert",
				Condition:   Condition{Field: "stock_status", Equals: "red"},
				TargetAgent: "G3",
				TargetDP:    "discount_risk_check",
			},
		},
		newTestLogger(),
	)

	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "agent.decided.A5.stock_alert",
		Payload: map[string]interface{}{"stock_status": "green"},
	})
	if runCount != 0 {
		t.Fatalf("expected 0 dispatches (condition blocks), got %d", runCount)
	}
}

func TestEngine_Dispatch_WildcardTopic(t *testing.T) {
	var runCount int
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			runCount++
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "agent.decided.*",
				TargetAgent: "A1",
				TargetDP:    "generic_handler",
			},
		},
		newTestLogger(),
	)

	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "agent.decided.A5.stock_alert",
		Payload: map[string]interface{}{},
	})
	if runCount != 1 {
		t.Fatalf("expected 1 dispatch with wildcard topic, got %d", runCount)
	}
}

func TestEngine_Dispatch_DefaultTimeout(t *testing.T) {
	// When Timeout is zero, Engine should use 30s default.
	// We can't easily test the wall clock, but we can verify dispatch still works.
	var runCount int
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			runCount++
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "test.timeout",
				Condition:   Condition{Field: "x", Equals: "y"},
				TargetAgent: "T1",
				TargetDP:    "timeout_test",
				// Timeout intentionally zero — should use default
			},
		},
		newTestLogger(),
	)

	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "test.timeout",
		Payload: map[string]interface{}{"x": "y"},
	})
	if runCount != 1 {
		t.Fatalf("expected 1 dispatch with default timeout, got %d", runCount)
	}
}

func TestEngine_Dispatch_CustomTimeout(t *testing.T) {
	var runCount int
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			// Verify that the context has a deadline set.
			if _, ok := ctx.Deadline(); !ok {
				t.Error("expected context with deadline for custom timeout")
			}
			runCount++
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "test.timeout",
				Condition:   Condition{Field: "x", Equals: "y"},
				TargetAgent: "T2",
				TargetDP:    "timeout_custom",
				Timeout:     100 * time.Millisecond,
			},
		},
		newTestLogger(),
	)

	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "test.timeout",
		Payload: map[string]interface{}{"x": "y"},
	})
	if runCount != 1 {
		t.Fatalf("expected 1 dispatch with custom timeout, got %d", runCount)
	}
}

func TestEngine_Dispatch_ErrorOnHighPriority(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return errTestSimulated
		},
		[]PipelineEdge{
			{
				SourceTopic: "test.error",
				TargetAgent: "E1",
				TargetDP:    "error_dp",
				Priority:    1,
				Timeout:     10 * time.Millisecond,
			},
		},
		newTestLogger(),
	)

	err := eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "test.error",
		Payload: map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected error for high-priority edge failure")
	}
}

func TestEngine_Dispatch_SkipErrorOnLowPriority(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return errTestSimulated
		},
		[]PipelineEdge{
			{
				SourceTopic: "test.error",
				TargetAgent: "E2",
				TargetDP:    "error_dp",
				Priority:    0, // low priority — error should be swallowed
				Timeout:     10 * time.Millisecond,
			},
		},
		newTestLogger(),
	)

	err := eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "test.error",
		Payload: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("expected nil error for low-priority edge failure, got: %v", err)
	}
}

func TestEngine_Dispatch_ContextCancelled(t *testing.T) {
	var runCount int
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			runCount++
			return ctx.Err()
		},
		[]PipelineEdge{
			{
				SourceTopic: "test.cancel",
				TargetAgent: "C1",
				TargetDP:    "cancel_dp",
				Priority:    0,
				Timeout:     100 * time.Millisecond,
			},
		},
		newTestLogger(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_ = eng.Dispatch(ctx, eventbus.Event{
		Topic:   "test.cancel",
		Payload: map[string]interface{}{},
	})
	// The agent is still called (Dispatch creates a fresh timeout context),
	// but the underlying context is already cancelled so runAgent gets the error.
	if runCount != 1 {
		t.Fatalf("expected 1 dispatch attempt, got %d", runCount)
	}
}

func TestEngine_ConfigureBreaker(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return nil
		},
		[]PipelineEdge{},
		newTestLogger(),
	)

	eng.ConfigureBreaker("test-agent", CircuitBreakerConfig{
		Threshold:  0.2,
		WindowSize: 20,
		Cooldown:   60 * time.Second,
	})

	cb := eng.Breakers().Get("test-agent")
	if cb == nil {
		t.Fatal("expected breaker for test-agent after ConfigureBreaker")
	}
	if cb.config.Threshold != 0.2 {
		t.Fatalf("expected threshold 0.2, got %f", cb.config.Threshold)
	}
	if cb.config.WindowSize != 20 {
		t.Fatalf("expected window size 20, got %d", cb.config.WindowSize)
	}
}

func TestEngine_Breakers_LazyCreation(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return nil
		},
		[]PipelineEdge{
			{SourceTopic: "test.lazy", TargetAgent: "L1", TargetDP: "lazy"},
		},
		newTestLogger(),
	)

	// Breaker should not exist before first dispatch.
	if eng.Breakers().Get("L1") != nil {
		t.Fatal("expected breaker to be nil before first dispatch")
	}

	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "test.lazy",
		Payload: map[string]interface{}{},
	})

	// Breaker should be created lazily by Allow() during dispatch.
	if eng.Breakers().Get("L1") == nil {
		t.Fatal("expected breaker to exist after first dispatch")
	}
}
