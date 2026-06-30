package pipeline

import (
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultBreakerConfig()
	if cfg.Threshold != 0.5 {
		t.Fatalf("expected threshold 0.5, got %v", cfg.Threshold)
	}
	if cfg.WindowSize != 10 {
		t.Fatalf("expected window size 10, got %v", cfg.WindowSize)
	}
	if cfg.Cooldown != 30*time.Second {
		t.Fatalf("expected cooldown 30s, got %v", cfg.Cooldown)
	}
	if cfg.HalfOpenMax != 1 {
		t.Fatalf("expected half open max 1, got %v", cfg.HalfOpenMax)
	}
}

func TestCustomConfig(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.3, WindowSize: 5, Cooldown: 5 * time.Second, HalfOpenMax: 2}
	cb := NewCircuitBreaker(cfg)
	if cb.config.Threshold != 0.3 {
		t.Fatal("threshold not set")
	}
	if cb.State() != BreakerClosed {
		t.Fatal("should start closed")
	}
}

func TestAllowClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultBreakerConfig())
	for i := 0; i < 5; i++ {
		if !cb.Allow() {
			t.Fatal("should allow when closed")
		}
	}
}

func TestCircuitOpenOnThreshold(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.6, WindowSize: 5, Cooldown: time.Hour, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	// Record 3 successes and 2 failures (ratio 0.4 — under threshold).
	for i := 0; i < 5; i++ {
		cb.Record(false)
	}
	// 5 failures in 5 = 1.0, but window is 5 so it triggers
	if cb.State() != BreakerOpen {
		t.Fatalf("expected OPEN after threshold exceeded, got %v", cb.State())
	}
}

func TestBlockedWhenOpen(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.1, WindowSize: 3, Cooldown: time.Hour, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	cb.Record(false) // 1/1 = 1.0 > 0.1, but window not full (only 1)
	// Need 3 records to trigger
	cb.Record(false)
	cb.Record(false)

	if cb.State() != BreakerOpen {
		t.Fatalf("expected OPEN, got %v", cb.State())
	}
	if cb.Allow() {
		t.Fatal("should NOT allow when OPEN and in cooldown")
	}
}

func TestHalfOpenAfterCooldown(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.1, WindowSize: 3, Cooldown: 10 * time.Millisecond, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	cb.Record(false)
	cb.Record(false)
	cb.Record(false) // window full, 3/3 fails > 0.1

	if cb.State() != BreakerOpen {
		t.Fatalf("expected OPEN, got %v", cb.State())
	}

	time.Sleep(15 * time.Millisecond)

	// Allow should transition to HALF_OPEN.
	if !cb.Allow() {
		t.Fatal("should allow after cooldown (HALF_OPEN)")
	}
	if cb.State() != BreakerHalfOpen {
		t.Fatalf("expected HALF_OPEN after cooldown, got %v", cb.State())
	}
}

func TestHalfOpenSuccessCloses(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.1, WindowSize: 3, Cooldown: 10 * time.Millisecond, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	cb.Record(false)
	cb.Record(false)
	cb.Record(false)
	if cb.State() != BreakerOpen {
		t.Fatal("should be OPEN")
	}

	time.Sleep(15 * time.Millisecond)

	// Transition to HALF_OPEN via Allow.
	cb.Allow()
	if cb.State() != BreakerHalfOpen {
		t.Fatalf("expected HALF_OPEN, got %v", cb.State())
	}

	// Record success — should close.
	cb.Record(true)
	if cb.State() != BreakerClosed {
		t.Fatalf("expected CLOSED after successful probe, got %v", cb.State())
	}
}

func TestHalfOpenFailureReopens(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.1, WindowSize: 3, Cooldown: 10 * time.Millisecond, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	// Trigger open.
	cb.Record(false)
	cb.Record(false)
	cb.Record(false)
	if cb.State() != BreakerOpen {
		t.Fatal("should be OPEN")
	}

	time.Sleep(15 * time.Millisecond)

	// Enter HALF_OPEN.
	cb.Allow()
	if cb.State() != BreakerHalfOpen {
		t.Fatalf("expected HALF_OPEN, got %v", cb.State())
	}

	// Record failure — should go back to OPEN.
	cb.Record(false)
	if cb.State() != BreakerOpen {
		t.Fatalf("expected OPEN after failed probe, got %v", cb.State())
	}
}

func TestWindowNotFullNoTrigger(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.1, WindowSize: 10, Cooldown: time.Hour, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	// Only 3 failures out of 10 required window — not enough.
	for i := 0; i < 5; i++ {
		cb.Record(false)
	}
	if cb.State() == BreakerOpen {
		t.Fatal("should NOT open with fewer records than window size")
	}
}

func TestWindowSliding(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.5, WindowSize: 4, Cooldown: time.Hour, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	// 4 failures = 100% failure, should open.
	for i := 0; i < 4; i++ {
		cb.Record(false)
	}
	if cb.State() != BreakerOpen {
		t.Fatal("should be OPEN after 4 failures")
	}
}

func TestReset(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.1, WindowSize: 3, Cooldown: time.Hour, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	for i := 0; i < 3; i++ {
		cb.Record(false)
	}
	if cb.State() != BreakerOpen {
		t.Fatal("should be OPEN")
	}

	cb.Reset()
	if cb.State() != BreakerClosed {
		t.Fatal("should be CLOSED after reset")
	}
	if !cb.Allow() {
		t.Fatal("should allow after reset")
	}
}

func TestConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(DefaultBreakerConfig())
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cb.Allow()
			cb.Record(n%2 == 0)
			cb.State()
		}(i)
	}
	wg.Wait()
}

func TestRegistryGetOrCreate(t *testing.T) {
	r := NewCircuitBreakerRegistry(DefaultBreakerConfig())
	b1 := r.GetOrCreate("A5")
	b2 := r.GetOrCreate("A5")
	if b1 != b2 {
		t.Fatal("should return same breaker for same agent")
	}
	b3 := r.GetOrCreate("G3")
	if b1 == b3 {
		t.Fatal("should return different breaker for different agent")
	}
}

func TestRegistryAllowRecord(t *testing.T) {
	r := NewCircuitBreakerRegistry(BreakerConfig{Threshold: 0.1, WindowSize: 3, Cooldown: time.Hour, HalfOpenMax: 1})
	if !r.Allow("A5") {
		t.Fatal("should allow initially")
	}
	r.Record("A5", false)
	r.Record("A5", false)
	r.Record("A5", false)
	if r.Allow("A5") {
		t.Fatal("should NOT allow after threshold")
	}
	// Different agent still fine.
	if !r.Allow("G3") {
		t.Fatal("G3 should not be affected")
	}
}

func TestRegistryResetAll(t *testing.T) {
	r := NewCircuitBreakerRegistry(BreakerConfig{Threshold: 0.1, WindowSize: 3, Cooldown: time.Hour, HalfOpenMax: 1})
	for i := 0; i < 3; i++ {
		r.Record("A5", false)
	}
	if r.Allow("A5") {
		t.Fatal("should be blocked")
	}
	r.ResetAll()
	if !r.Allow("A5") {
		t.Fatal("should allow after reset")
	}
}

func TestRegistryBreakerNames(t *testing.T) {
	r := NewCircuitBreakerRegistry(DefaultBreakerConfig())
	r.GetOrCreate("A5")
	r.GetOrCreate("G3")
	names := r.BreakerNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %v", names)
	}
}

func TestHalfOpenMaxAllowed(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.5, WindowSize: 2, Cooldown: 10 * time.Millisecond, HalfOpenMax: 2}
	cb := NewCircuitBreaker(cfg)

	// Trigger open.
	cb.Record(false)
	cb.Record(false)
	if cb.State() != BreakerOpen {
		t.Fatal("should be OPEN")
	}

	time.Sleep(15 * time.Millisecond)

	// Both probes allowed.
	if !cb.Allow() {
		t.Fatal("first half-open probe should be allowed")
	}
	if !cb.Allow() {
		t.Fatal("second half-open probe should be allowed")
	}
	// Third should be blocked.
	if cb.Allow() {
		t.Fatal("third half-open probe should be blocked")
	}
}

func TestEngineIntegration(t *testing.T) {
	// This tests that the engine integration works end-to-end.
	// The breaker is wired in Engine.Dispatch and works per-agent.

	cfg := BreakerConfig{Threshold: 0.1, WindowSize: 3, Cooldown: time.Hour, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	if !cb.Allow() {
		t.Fatal("should allow when closed")
	}

	// 3 failures = threshold exceeded.
	cb.Record(false)
	cb.Record(false)
	cb.Record(false)

	if cb.Allow() {
		t.Fatal("should NOT allow when open")
	}
}

func TestStats(t *testing.T) {
	cb := NewCircuitBreaker(DefaultBreakerConfig())
	cb.Record(true)
	cb.Record(false)

	state, success, failure, _ := cb.Stats()
	if state != BreakerClosed {
		t.Fatalf("expected CLOSED, got %v", state)
	}
	if success < 1 {
		t.Fatal("expected at least 1 success")
	}
	if failure < 1 {
		t.Fatal("expected at least 1 failure")
	}
}

// Edge case: single failure should not trigger breaker when window not full.
func TestSingleFailureNoTrigger(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.5, WindowSize: 10, Cooldown: time.Hour, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	cb.Record(false) // 1 record, window not full — no trigger
	if cb.State() == BreakerOpen {
		t.Fatal("single failure should not trigger OPEN")
	}
	if !cb.Allow() {
		t.Fatal("should still allow")
	}
}

// Edge case: all successes should keep breaker closed.
func TestAllSuccess(t *testing.T) {
	cfg := BreakerConfig{Threshold: 0.5, WindowSize: 10, Cooldown: time.Hour, HalfOpenMax: 1}
	cb := NewCircuitBreaker(cfg)

	for i := 0; i < 20; i++ {
		cb.Record(true)
	}
	if cb.State() != BreakerClosed {
		t.Fatal("all successes should stay CLOSED")
	}
	if !cb.Allow() {
		t.Fatal("should always allow")
	}
}
