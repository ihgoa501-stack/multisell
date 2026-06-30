package statemachine

import (
	"context"
	"errors"
	"testing"
)

// testTransitions defines a simple order-like workflow for unit tests.
var testTransitions = map[string]map[string]bool{
	"pending":   {"confirmed": true, "cancelled": true},
	"confirmed": {"shipped": true, "cancelled": true},
	"shipped":   {"delivered": true},
	"delivered": {"completed": true},
	"completed": {},
	"cancelled": {},
}

func TestNew(t *testing.T) {
	sm := New(testTransitions)
	if sm == nil {
		t.Fatal("New returned nil")
	}
}

func TestCanTransition_Valid(t *testing.T) {
	sm := New(testTransitions)
	if !sm.CanTransition("pending", "confirmed") {
		t.Error("expected pending -> confirmed to be valid")
	}
	if !sm.CanTransition("confirmed", "cancelled") {
		t.Error("expected confirmed -> cancelled to be valid")
	}
}

func TestCanTransition_Invalid(t *testing.T) {
	sm := New(testTransitions)
	if sm.CanTransition("pending", "delivered") {
		t.Error("expected pending -> delivered to be invalid")
	}
	if sm.CanTransition("completed", "pending") {
		t.Error("expected completed -> pending to be invalid (terminal)")
	}
}

func TestMustTransition_Valid(t *testing.T) {
	sm := New(testTransitions)
	err := sm.MustTransition(context.Background(), "pending", "confirmed", nil)
	if err != nil {
		t.Fatalf("MustTransition failed: %v", err)
	}
}

func TestMustTransition_InvalidTransition(t *testing.T) {
	sm := New(testTransitions)
	err := sm.MustTransition(context.Background(), "pending", "delivered", nil)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestMustTransition_TerminalStatus(t *testing.T) {
	sm := New(testTransitions)
	err := sm.MustTransition(context.Background(), "completed", "pending", nil)
	if err == nil {
		t.Fatal("expected error for terminal status")
	}
}

func TestMustTransition_GuardPasses(t *testing.T) {
	sm := New(testTransitions)
	sm.AddGuard("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		return nil
	})
	err := sm.MustTransition(context.Background(), "pending", "confirmed", nil)
	if err != nil {
		t.Fatalf("MustTransition failed with passing guard: %v", err)
	}
}

func TestMustTransition_GuardBlocks(t *testing.T) {
	sm := New(testTransitions)
	sm.AddGuard("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		return errors.New("not enough inventory")
	})
	err := sm.MustTransition(context.Background(), "pending", "confirmed", nil)
	if err == nil {
		t.Fatal("expected error from blocking guard")
	}
}

func TestMustTransition_GuardForDifferentTransitionOnly(t *testing.T) {
	sm := New(testTransitions)
	sm.AddGuard("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		return errors.New("block pending -> confirmed")
	})
	// pending -> cancelled should still work (no guard registered for it)
	err := sm.MustTransition(context.Background(), "pending", "cancelled", nil)
	if err != nil {
		t.Fatalf("MustTransition for pending -> cancelled should pass: %v", err)
	}
}

func TestMustTransition_Hooks(t *testing.T) {
	sm := New(testTransitions)
	called := false
	sm.AddHook("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		called = true
		return nil
	})
	err := sm.MustTransition(context.Background(), "pending", "confirmed", nil)
	if err != nil {
		t.Fatalf("MustTransition failed: %v", err)
	}
	if !called {
		t.Error("hook was not called")
	}
}

func TestMustTransition_HookError(t *testing.T) {
	sm := New(testTransitions)
	sm.AddHook("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		return errors.New("hook failed")
	})
	err := sm.MustTransition(context.Background(), "pending", "confirmed", nil)
	if err == nil {
		t.Fatal("expected error from failing hook")
	}
}

func TestAllowedTargets(t *testing.T) {
	sm := New(testTransitions)
	targets := sm.AllowedTargets("pending")
	if len(targets) != 2 {
		t.Fatalf("pending should have 2 targets, got %d: %v", len(targets), targets)
	}
	hasConfirmed := false
	hasCancelled := false
	for _, tgt := range targets {
		if tgt == "confirmed" {
			hasConfirmed = true
		}
		if tgt == "cancelled" {
			hasCancelled = true
		}
	}
	if !hasConfirmed || !hasCancelled {
		t.Error("pending targets should include confirmed and cancelled")
	}
}

func TestAllowedTargets_Terminal(t *testing.T) {
	sm := New(testTransitions)
	targets := sm.AllowedTargets("completed")
	if targets != nil {
		t.Errorf("completed should have nil targets, got %v", targets)
	}
}

func TestIsTerminal(t *testing.T) {
	sm := New(testTransitions)
	if !sm.IsTerminal("completed") {
		t.Error("expected completed to be terminal")
	}
	if !sm.IsTerminal("cancelled") {
		t.Error("expected cancelled to be terminal")
	}
	if sm.IsTerminal("pending") {
		t.Error("expected pending to not be terminal")
	}
}

func TestGuardReceivesEntity(t *testing.T) {
	sm := New(testTransitions)
	type Order struct {
		ID     int64
		Amount float64
	}
	entity := &Order{ID: 1, Amount: 100}
	sm.AddGuard("pending", "confirmed", func(ctx context.Context, e interface{}) error {
		o, ok := e.(*Order)
		if !ok {
			return errors.New("wrong type")
		}
		if o.Amount <= 0 {
			return errors.New("amount must be positive")
		}
		return nil
	})
	err := sm.MustTransition(context.Background(), "pending", "confirmed", entity)
	if err != nil {
		t.Fatalf("MustTransition failed: %v", err)
	}
}

func TestMultipleGuards(t *testing.T) {
	sm := New(testTransitions)
	var order []int
	sm.AddGuard("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		order = append(order, 1)
		return nil
	})
	sm.AddGuard("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		order = append(order, 2)
		return nil
	})
	_ = sm.MustTransition(context.Background(), "pending", "confirmed", nil)
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("guards executed out of order: %v", order)
	}
}

func TestGuardStopsHookExecution(t *testing.T) {
	sm := New(testTransitions)
	hookCalled := false
	sm.AddGuard("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		return errors.New("block")
	})
	sm.AddHook("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
		hookCalled = true
		return nil
	})
	_ = sm.MustTransition(context.Background(), "pending", "confirmed", nil)
	if hookCalled {
		t.Error("hook should not be called when guard blocks")
	}
}
