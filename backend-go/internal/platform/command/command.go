// Package command provides a dispatcher for executing business commands from
// agent actions. It maps action types to handler functions that call real
// domain services, bridging the gap between agent decisions and business logic.
//
// Usage:
//   d := command.NewDispatcher(logger)
//   d.Register("replenish", command.ReplenishHandler(db))
//   result, err := d.Dispatch(ctx, "replenish", payload)
package command

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Handler executes a business command and returns a result.
type Handler func(ctx context.Context, input map[string]interface{}) (*Result, error)

// Result captures the outcome of a command execution.
type Result struct {
	Success       bool                   `json:"success"`
	AfterSnapshot map[string]interface{} `json:"after_snapshot,omitempty"`
	BusinessID    string                 `json:"business_id,omitempty"`
	ErrorMessage  string                 `json:"error,omitempty"`
}

// Dispatcher routes action types to registered handler functions.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	logger   *zap.Logger
}

// NewDispatcher creates a new command dispatcher.
func NewDispatcher(logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]Handler),
		logger:   logger,
	}
}

// Register binds an action type to a handler. If a handler already exists
// for the given action type, it is overwritten.
func (d *Dispatcher) Register(actionType string, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[actionType] = handler
	d.logger.Info("command handler registered",
		zap.String("action_type", actionType))
}

// Unregister removes a handler for an action type.
func (d *Dispatcher) Unregister(actionType string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.handlers, actionType)
	d.logger.Info("command handler unregistered",
		zap.String("action_type", actionType))
}

// Dispatch executes the handler registered for the given action type.
// Returns ErrHandlerNotFound if no handler is registered.
func (d *Dispatcher) Dispatch(ctx context.Context, actionType string, payload map[string]interface{}) (*Result, error) {
	d.mu.RLock()
	handler, ok := d.handlers[actionType]
	d.mu.RUnlock()

	if !ok {
		return nil, &HandlerNotFoundError{ActionType: actionType}
	}

	d.logger.Debug("dispatching command",
		zap.String("action_type", actionType))

	result, err := handler(ctx, payload)
	if err != nil {
		d.logger.Warn("command handler failed",
			zap.String("action_type", actionType),
			zap.Error(err))
		return &Result{Success: false, ErrorMessage: err.Error()}, nil
	}

	return result, nil
}

// DispatchSafe validates and dispatches an AgentAction through the dispatcher,
// checking mode, risk level, and approval requirements before executing.
//
// Mode rules:
//   - dry_run: the action is validated (handler must exist) but never executed.
//   - sandbox: the action executes regardless of risk.
//   - production: high-risk or approval-required actions need a valid approval_id.
func (d *Dispatcher) DispatchSafe(ctx context.Context, action AgentAction, policy PolicyChecker) (*Result, error) {
	// Mode: dry_run — validate existence only.
	if action.Mode == ModeDryRun {
		d.mu.RLock()
		_, ok := d.handlers[action.ActionType]
		d.mu.RUnlock()
		if !ok {
			return nil, &HandlerNotFoundError{ActionType: action.ActionType}
		}
		return &Result{Success: true, BusinessID: "dry_run"}, nil
	}

	// Production mode: high-risk or approval-required actions need a valid approval.
	if action.Mode == ModeProduction && (action.RiskLevel >= RiskHigh || action.ApprovalRequired) {
		if action.ApprovalID == nil {
			return nil, ErrApprovalRequired
		}
		if policy != nil && !policy.IsApproved(*action.ApprovalID) {
			return nil, ErrApprovalRequired
		}
	}

	// Sandbox and approved production actions execute normally.
	return d.Dispatch(ctx, action.ActionType, action.Input)
}

// PolicyChecker checks whether an approval ID is still valid at execution time.
// The actual policy engine (actionpolicy.Service) lives in the domain layer;
// this interface lets platform code stay dependency-free.
type PolicyChecker interface {
	IsApproved(approvalID int64) bool
}

// ErrApprovalRequired is returned when a high-risk action is attempted without
// a valid approval.
var ErrApprovalRequired = fmt.Errorf("action requires approval before execution")

// RegisteredTypes returns a list of all registered action types.
func (d *Dispatcher) RegisteredTypes() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	types := make([]string, 0, len(d.handlers))
	for t := range d.handlers {
		types = append(types, t)
	}
	return types
}

// HandlerCount returns the number of registered handlers.
func (d *Dispatcher) HandlerCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.handlers)
}

// HandlerNotFoundError is returned when no handler is registered for an action type.
type HandlerNotFoundError struct {
	ActionType string
}

func (e *HandlerNotFoundError) Error() string {
	return fmt.Sprintf("no handler registered for action type: %s", e.ActionType)
}

// IsHandlerNotFound checks if the error is a HandlerNotFoundError.
func IsHandlerNotFound(err error) bool {
	_, ok := err.(*HandlerNotFoundError)
	return ok
}
