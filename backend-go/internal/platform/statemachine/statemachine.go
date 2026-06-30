// Package statemachine provides a generic, zero-dependency finite state machine
// for status transition validation.
package statemachine

import (
	"context"
	"fmt"
)

// StateMachine implements a generic finite state machine for status transitions.
type StateMachine struct {
	transitions map[string]map[string]bool
	guards      map[string][]GuardFn
	hooks       map[string][]HookFn
}

// GuardFn is a pre-transition validation function.
type GuardFn func(ctx context.Context, entity interface{}) error

// HookFn is a post-transition callback.
type HookFn func(ctx context.Context, entity interface{}) error

// New creates a StateMachine from a transition map.
func New(transitions map[string]map[string]bool) *StateMachine {
	return &StateMachine{
		transitions: transitions,
		guards:      make(map[string][]GuardFn),
		hooks:       make(map[string][]HookFn),
	}
}

// CanTransition checks if the transition from current to target is defined.
func (sm *StateMachine) CanTransition(current, target string) bool {
	allowed, ok := sm.transitions[current]
	if !ok || len(allowed) == 0 {
		return false
	}
	return allowed[target]
}

// MustTransition validates a state transition and runs guards + hooks.
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
func (sm *StateMachine) AddGuard(from, to string, fn GuardFn) {
	key := transitionKey(from, to)
	sm.guards[key] = append(sm.guards[key], fn)
}

// AddHook registers a hook function for a specific transition.
func (sm *StateMachine) AddHook(from, to string, fn HookFn) {
	key := transitionKey(from, to)
	sm.hooks[key] = append(sm.hooks[key], fn)
}

// AllowedTargets returns all allowed target statuses from a given current status.
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
func (sm *StateMachine) IsTerminal(status string) bool {
	allowed, ok := sm.transitions[status]
	return !ok || len(allowed) == 0
}

func transitionKey(from, to string) string {
	return from + "->" + to
}
