// Package statemachine provides a generic, zero-dependency finite state machine
// for status transition validation. It uses the same map-based transition pattern
// as the existing aftersales and purchase modules, but extracted into a reusable
// framework with support for guards and hooks.
package statemachine

import (
	"context"
	"fmt"
)

// StateMachine implements a generic finite state machine for status transitions.
// It validates transitions, runs pre-transition guards, and executes post-transition
// hooks. It does NOT persist the entity — the caller is responsible for that.
type StateMachine struct {
	transitions map[string]map[string]bool // current -> allowed targets
	guards      map[string][]GuardFn       // "from->to" -> pre-transition guards
	hooks       map[string][]HookFn        // "from->to" -> post-transition hooks
}

// GuardFn is a pre-transition validation function.
// Return error to block the transition.
type GuardFn func(ctx context.Context, entity interface{}) error

// HookFn is a post-transition callback.
// Return error to signal failure (caller decides rollback behavior).
type HookFn func(ctx context.Context, entity interface{}) error

// New creates a StateMachine from a transition map.
//
// Format: map[currentStatus]map[targetStatus]bool
//
//	A status with no outgoing transitions is considered terminal.
//
// Example:
//
//	sm := New(map[string]map[string]bool{
//	    "pending":  {"confirmed": true, "cancelled": true},
//	    "confirmed": {"shipped": true},
//	    "shipped":   {"delivered": true},
//	    "delivered": {},
//	})
func New(transitions map[string]map[string]bool) *StateMachine {
	return &StateMachine{
		transitions: transitions,
		guards:      make(map[string][]GuardFn),
		hooks:       make(map[string][]HookFn),
	}
}

// CanTransition checks if the transition from current to target is defined.
// Does NOT evaluate guards.
func (sm *StateMachine) CanTransition(current, target string) bool {
	allowed, ok := sm.transitions[current]
	if !ok || len(allowed) == 0 {
		return false
	}
	return allowed[target]
}

// MustTransition validates a state transition and runs guards + hooks.
//
// Returns a descriptive error if:
//   - current is a terminal status (no transitions defined from it)
//   - transition from current to target is not allowed by the map
//   - any guard function returns an error
//
// Guards run first in registration order. If all guards pass, hooks run
// in registration order. If a guard fails, hooks are skipped.
// This method does NOT modify the entity or persist anything.
func (sm *StateMachine) MustTransition(ctx context.Context, current, target string, entity interface{}) error {
	allowed, ok := sm.transitions[current]
	if !ok || len(allowed) == 0 {
		return fmt.Errorf("cannot transition from terminal status %s", current)
	}
	if !allowed[target] {
		return fmt.Errorf("cannot transition from %s to %s", current, target)
	}

	key := transitionKey(current, target)
	for _, guard := range sm.guards[key] {
		if err := guard(ctx, entity); err != nil {
			return fmt.Errorf("guard blocked transition %s -> %s: %w", current, target, err)
		}
	}
	for _, hook := range sm.hooks[key] {
		if err := hook(ctx, entity); err != nil {
			return fmt.Errorf("hook failed on transition %s -> %s: %w", current, target, err)
		}
	}
	return nil
}

// AddGuard registers a guard function for a specific transition.
// Multiple guards can be registered for the same transition; they run in order.
func (sm *StateMachine) AddGuard(from, to string, fn GuardFn) {
	key := transitionKey(from, to)
	sm.guards[key] = append(sm.guards[key], fn)
}

// AddHook registers a hook function for a specific transition.
// Multiple hooks can be registered for the same transition; they run in order
// after all guards pass.
func (sm *StateMachine) AddHook(from, to string, fn HookFn) {
	key := transitionKey(from, to)
	sm.hooks[key] = append(sm.hooks[key], fn)
}

// AllowedTargets returns all allowed target statuses from a given current status.
// Returns nil if the status has no outgoing transitions (is terminal).
func (sm *StateMachine) AllowedTargets(current string) []string {
	allowed, ok := sm.transitions[current]
	if !ok || len(allowed) == 0 {
		return nil
	}
	targets := make([]string, 0, len(allowed))
	for t := range allowed {
		targets = append(targets, t)
	}
	return targets
}

// IsTerminal returns true if the status has no outgoing transitions defined.
// This includes both statuses not in the map and statuses with an empty map.
func (sm *StateMachine) IsTerminal(status string) bool {
	allowed, ok := sm.transitions[status]
	return !ok || len(allowed) == 0
}

func transitionKey(from, to string) string {
	return from + "->" + to
}
