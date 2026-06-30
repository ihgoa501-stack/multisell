package pipeline

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

// =============================================================================
// CircuitBreaker unit tests
// =============================================================================

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	if cb.State() != StateClosed {
		t.Fatalf("expected CLOSED, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Fatal("expected Allow()=true in CLOSED state")
	}
}

func TestCircuitBreaker_ConfigDefaults(t *testing.T) {
	// Zero config should get defaults.
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	if cb.config.Threshold != 0.5 {
		t.Fatalf("expected threshold 0.5, got %f", cb.config.Threshold)
	}
	if cb.config.WindowSize != 10 {
		t.Fatalf("expected window size 10, got %d", cb.config.WindowSize)
	}
	if cb.config.Cooldown != 30*time.Second {
		t.Fatalf("expected cooldown 30s, got %s", cb.config.Cooldown)
	}
	if cb.config.HalfOpenMax != 1 {
		t.Fatalf("expected HalfOpenMax 1, got %d", cb.config.HalfOpenMax)
	}
}

func TestCircuitBreaker_CustomConfig(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Threshold:   0.3,
		WindowSize:  5,
		Cooldown:    10 * time.Second,
		HalfOpenMax: 2,
	}
	cb := NewCircuitBreaker(cfg)

	if cb.config.Threshold != 0.3 {
		t.Fatalf("expected threshold 0.3, got %f", cb.config.Threshold)
	}
	if cb.config.WindowSize != 5 {
		t.Fatalf("expected window size 5, got %d", cb.config.WindowSize)
	}
	if cb.config.Cooldown != 10*time.Second {
		t.Fatalf("expected cooldown 10s, got %s", cb.config.Cooldown)
	}
	if cb.config.HalfOpenMax != 2 {
		t.Fatalf("expected HalfOpenMax 2, got %d", cb.config.HalfOpenMax)
	}
}

func TestCircuitBreaker_OpenOnThreshold(t *testing.T) {
	// With a 40% threshold and window 5, 3 successes + 2 failures = 40%.
	// Since window is full (5), 40% >= 40% -> opens.
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  0.4,
		WindowSize: 5,
		Cooldown:   10 * time.Second,
	})

	for i := 0; i < 3; i++ {
		cb.Record(true)
	}
	cb.Record(false)
	cb.Record(false)

	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN after 2/5 failures (40%%), got %s", cb.State())
	}
}

func TestCircuitBreaker_StaysClosedBelowThreshold(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  0.5,
		WindowSize: 10,
		Cooldown:   30 * time.Second,
	})

	// 4 failures out of 10 = 40% < 50%.
	for i := 0; i < 6; i++ {
		cb.Record(true)
	}
	for i := 0; i < 4; i++ {
		cb.Record(false)
	}

	if cb.State() != StateClosed {
		t.Fatalf("expected CLOSED at 40%% failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_StaysClosedUntilWindowFull(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  0.3,
		WindowSize: 5,
		Cooldown:   1 * time.Hour,
	})

	// Even 2 failures with only 2 records (100% failure rate) won't trigger
	// because the window isn't full yet.
	cb.Record(false)
	cb.Record(false)

	if cb.State() != StateClosed {
		t.Fatalf("expected CLOSED (window not full), got %s", cb.State())
	}
}

func TestCircuitBreaker_BlocksWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  0.3,
		WindowSize: 5,
		Cooldown:   1 * time.Hour, // long cooldown to stay OPEN
	})

	// Fill window with 3 successes + 2 failures = 40% > 30%.
	for i := 0; i < 3; i++ {
		cb.Record(true)
	}
	cb.Record(false)
	cb.Record(false)

	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN state, got %s", cb.State())
	}

	if cb.Allow() {
		t.Fatal("expected Allow()=false while circuit is OPEN with long cooldown")
	}
}

func TestCircuitBreaker_HalfOpenProbeSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  0.4,
		WindowSize: 5,
		Cooldown:   1 * time.Millisecond, // very short cooldown
	})

	// Trip the breaker: 2 failures out of 5 = 40%.
	for i := 0; i < 3; i++ {
		cb.Record(true)
	}
	cb.Record(false)
	cb.Record(false)

	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN before cooldown, got %s", cb.State())
	}

	// Wait for cooldown.
	time.Sleep(5 * time.Millisecond)

	// Allow() should transition to HALF_OPEN and permit a probe.
	allowed := cb.Allow()
	if !allowed {
		t.Fatal("expected Allow()=true after cooldown (HALF_OPEN)")
	}
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected HALF_OPEN after cooldown, got %s", cb.State())
	}

	// Record a success — should reset to CLOSED.
	cb.Record(true)
	if cb.State() != StateClosed {
		t.Fatalf("expected CLOSED after probe success, got %s", cb.State())
	}

	// Allow should work again.
	if !cb.Allow() {
		t.Fatal("expected Allow()=true after reset to CLOSED")
	}
}

func TestCircuitBreaker_HalfOpenProbeFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  0.4,
		WindowSize: 5,
		Cooldown:   1 * time.Millisecond,
	})

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		cb.Record(true)
	}
	cb.Record(false)
	cb.Record(false)

	// Wait for cooldown.
	time.Sleep(5 * time.Millisecond)

	// Allow transitions to HALF_OPEN.
	if !cb.Allow() {
		t.Fatal("expected Allow()=true in HALF_OPEN")
	}

	// Record a failure — should go back to OPEN.
	cb.Record(false)
	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN after probe failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_WindowSliding(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  0.5,
		WindowSize: 3,
		Cooldown:   30 * time.Second,
	})

	// Fill window: 3 successes, no failures.
	for i := 0; i < 3; i++ {
		cb.Record(true)
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected CLOSED, got %s", cb.State())
	}
	if r := cb.FailureRate(); r != 0 {
		t.Fatalf("expected failure rate 0, got %f", r)
	}

	// Add failures. Since window is 3, old successes slide out.
	// After 3 more records: first 3 slide out, we have [f, f, f].
	cb.Record(false) // window: [t, t, f]
	cb.Record(false) // window: [t, f, f]
	cb.Record(false) // window: [f, f, f] — 100% failure

	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN after sliding window fills with failures, got %s", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Threshold:  0.3,
		WindowSize: 5,
		Cooldown:   30 * time.Second,
	})

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		cb.Record(true)
	}
	cb.Record(false)
	cb.Record(false)

	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN before reset, got %s", cb.State())
	}

	cb.Reset()
	if cb.State() != StateClosed {
		t.Fatalf("expected CLOSED after reset, got %s", cb.State())
	}
	if cb.WindowSize() != 0 {
		t.Fatalf("expected empty window after reset, got %d", cb.WindowSize())
	}
}

func TestCircuitBreaker_ConcurrentSafety(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	var wg sync.WaitGroup

	// Fire off many concurrent calls to Allow() and Record().
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.Allow()
			cb.Record(true)
		}()
	}
	wg.Wait()

	// Should not panic, and should still be CLOSED with 0 failures.
	if cb.State() != StateClosed {
		t.Fatalf("expected CLOSED after all successes, got %s", cb.State())
	}
}

// =============================================================================
// CircuitBreakerRegistry tests
// =============================================================================

func TestRegistry_GetOrCreate(t *testing.T) {
	r := NewCircuitBreakerRegistry()

	cb1 := r.GetOrCreate("agent-A1")
	if cb1 == nil {
		t.Fatal("expected non-nil breaker")
	}
	if cb1.State() != StateClosed {
		t.Fatalf("expected CLOSED, got %s", cb1.State())
	}

	// Getting same name returns same breaker.
	cb2 := r.GetOrCreate("agent-A1")
	if cb1 != cb2 {
		t.Fatal("expected same breaker instance")
	}
}

func TestRegistry_AllowRecord(t *testing.T) {
	r := NewCircuitBreakerRegistry()

	// Allow creates a breaker if needed.
	if !r.Allow("agent-X") {
		t.Fatal("expected Allow()=true for new breaker")
	}

	// 8 successes + 2 failures = 20% < 50% threshold -> stays CLOSED.
	for i := 0; i < 8; i++ {
		r.Record("agent-X", true)
	}
	r.Record("agent-X", false)
	r.Record("agent-X", false)

	if r.Get("agent-X").State() != StateClosed {
		t.Fatalf("expected CLOSED (20%% failure rate < 50%% threshold), got %s", r.Get("agent-X").State())
	}
}

func TestRegistry_ResetAll(t *testing.T) {
	r := NewCircuitBreakerRegistry()

	// Create breakers and trip one.
	cbA := r.GetOrCreate("agent-A")
	cbB := r.GetOrCreate("agent-B")

	// Trip A by filling the window with all failures.
	for i := 0; i < 10; i++ {
		cbA.Record(false)
	}
	if cbA.State() != StateOpen {
		t.Fatalf("expected OPEN for agent-A, got %s", cbA.State())
	}

	r.ResetAll()
	if cbA.State() != StateClosed {
		t.Fatalf("expected CLOSED after ResetAll, got %s", cbA.State())
	}
	if cbB.State() != StateClosed {
		t.Fatalf("expected CLOSED after ResetAll, got %s", cbB.State())
	}
}

func TestRegistry_BreakerNames(t *testing.T) {
	r := NewCircuitBreakerRegistry()
	r.GetOrCreate("a1")
	r.GetOrCreate("b2")
	r.GetOrCreate("c3")

	names := r.BreakerNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d: %v", len(names), names)
	}
}

// =============================================================================
// Engine integration tests
// =============================================================================

// newTestLogger creates a no-op logger for tests.
func newTestLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level.SetLevel(zap.FatalLevel + 1) // suppress all output
	l, _ := cfg.Build()
	return l
}

func TestEngine_DispatchBreakerOpenDegrades(t *testing.T) {
	// Engine with an edge that always matches, but the agent's breaker is OPEN.
	runCount := 0
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			runCount++
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "agent.decided.*",
				TargetAgent: "A1",
				TargetDP:    "test_dp",
				Priority:    1,
			},
		},
		newTestLogger(),
	)

	// Trip the breaker for A1 by forcing OPEN state.
	breaker := eng.Breakers().GetOrCreate("A1")
	breaker.mu.Lock()
	breaker.state = StateOpen
	breaker.lastOpened = time.Now()
	breaker.mu.Unlock()

	err := eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "agent.decided.A5.stock_alert",
		Payload: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("expected nil error (degraded), got: %v", err)
	}
	if runCount != 0 {
		t.Fatalf("expected 0 runs (breaker open), got %d", runCount)
	}
}

func TestEngine_DispatchBreakerRecordsResult(t *testing.T) {
	var callCount int
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			callCount++
			if callCount == 1 {
				return nil // success
			}
			return errors.New("simulated failure")
		},
		[]PipelineEdge{
			{
				SourceTopic: "agent.decided.*",
				TargetAgent: "A2",
				TargetDP:    "test_dp",
				Priority:    0,
			},
		},
		newTestLogger(),
	)

	// Record 1 success.
	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "agent.decided.A5.stock_alert",
		Payload: map[string]interface{}{},
	})

	breaker := eng.Breakers().Get("A2")
	if breaker == nil {
		t.Fatal("expected breaker for A2")
	}
	if r := breaker.FailureRate(); r != 0.0 {
		t.Fatalf("expected 0%% failure rate, got %f", r)
	}

	// Record 1 failure.
	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "agent.decided.A5.stock_alert",
		Payload: map[string]interface{}{},
	})

	if r := breaker.FailureRate(); r != 0.5 {
		t.Fatalf("expected 50%% failure rate, got %f", r)
	}
}

func TestEngine_DispatchWithConfigurableBreaker(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return errors.New("always fail")
		},
		[]PipelineEdge{
			{
				SourceTopic: "agent.decided.*",
				TargetAgent: "A3",
				TargetDP:    "test_dp",
			},
		},
		newTestLogger(),
	)

	// Configure a breaker with 60% threshold and window 5.
	// 3 failures out of 5 = 60% -> trips on the 5th call.
	eng.ConfigureBreaker("A3", CircuitBreakerConfig{
		Threshold:  0.6,
		WindowSize: 5,
		Cooldown:   1 * time.Hour,
	})

	for i := 0; i < 5; i++ {
		_ = eng.Dispatch(context.Background(), eventbus.Event{
			Topic:   "agent.decided.A5.stock_alert",
			Payload: map[string]interface{}{},
		})
	}

	cb := eng.Breakers().Get("A3")
	if cb == nil {
		t.Fatal("expected breaker for A3")
	}
	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN after 5/5 failures (100%%), got %s", cb.State())
	}
	if cb.WindowSize() != 5 {
		t.Fatalf("expected window size 5, got %d", cb.WindowSize())
	}

	// Next dispatch should be degraded (breaker open).
	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "agent.decided.A5.stock_alert",
		Payload: map[string]interface{}{},
	})

	// Window should still be 5 (degraded calls aren't recorded).
	if cb.WindowSize() != 5 {
		t.Fatalf("expected window size 5 (degraded calls not recorded), got %d", cb.WindowSize())
	}
}

func TestEngine_BreakersAccessor(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return nil
		},
		[]PipelineEdge{},
		newTestLogger(),
	)

	r := eng.Breakers()
	if r == nil {
		t.Fatal("expected non-nil breakers registry")
	}

	r.GetOrCreate("test-agent")
	if len(r.BreakerNames()) != 1 {
		t.Fatalf("expected 1 breaker, got %d", len(r.BreakerNames()))
	}
}

func TestEngine_MultipleEdgesIndependentBreakers(t *testing.T) {
	var callCount int
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			callCount++
			if agentID == "A5" {
				return errors.New("A5 always fails")
			}
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "agent.decided.A5.stock_alert",
				Condition: Condition{
					Field:  "stock_status",
					Equals: "red",
				},
				TargetAgent: "A5",
				TargetDP:    "stock_alert",
				Priority:    1,
			},
			{
				SourceTopic: "agent.decided.A6.profit_watch",
				Condition: Condition{
					Field:      "is_loss",
					BoolEquals: boolPtr(true),
				},
				TargetAgent: "A6",
				TargetDP:    "profit_watch",
				Priority:    1,
			},
			{
				SourceTopic: "agent.decided.A6.profit_watch",
				Condition: Condition{
					Field:      "below_threshold",
					BoolEquals: boolPtr(true),
				},
				TargetAgent: "A6",
				TargetDP:    "profit_watch",
				Priority:    1,
			},
		},
		newTestLogger(),
	)

	// Trip A5 with failures (window=10, threshold=50%).
	for i := 0; i < 10; i++ {
		_ = eng.Dispatch(context.Background(), eventbus.Event{
			Topic: "agent.decided.A5.stock_alert",
			Payload: map[string]interface{}{
				"stock_status": "red",
			},
		})
	}

	cbA5 := eng.Breakers().Get("A5")
	cbA6 := eng.Breakers().Get("A6")

	if cbA5 == nil || cbA5.State() != StateOpen {
		t.Fatalf("expected A5 breaker OPEN, got %v", cbA5)
	}
	if cbA6 != nil {
		t.Fatalf("expected A6 breaker to be nil (never called), got state=%s", cbA6.State())
	}
}

func TestEngine_DispatchNoBreakerForNonMatchingEdge(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "some.other.topic",
				TargetAgent: "A99",
				TargetDP:    "noop",
			},
		},
		newTestLogger(),
	)

	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic:   "unrelated.event",
		Payload: map[string]interface{}{},
	})

	// No breaker should have been created since no edge matched.
	if len(eng.Breakers().BreakerNames()) != 0 {
		t.Fatalf("expected no breakers created, got %v", eng.Breakers().BreakerNames())
	}
}

func TestEngine_UnrelatedEdge(t *testing.T) {
	eng := NewEngine(
		func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
			return nil
		},
		[]PipelineEdge{
			{
				SourceTopic: "agent.decided.A5.stock_alert",
				TargetAgent: "A5",
				TargetDP:    "alert",
			},
		},
		newTestLogger(),
	)

	// Dispatch an event that doesn't match any edge.
	_ = eng.Dispatch(context.Background(), eventbus.Event{
		Topic: "scheduler.tick.G0",
		Payload: map[string]interface{}{
			"anomaly_count": 5,
		},
	})

	// No runs, no breaker created.
	if len(eng.Breakers().BreakerNames()) != 0 {
		t.Fatalf("expected no breakers for unrelated event, got %v", eng.Breakers().BreakerNames())
	}
}

func TestCircuitBreaker_StateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "CLOSED"},
		{StateOpen, "OPEN"},
		{StateHalfOpen, "HALF_OPEN"},
		{State(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
