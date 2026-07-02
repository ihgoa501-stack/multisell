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

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
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
	mu          sync.RWMutex
	handlers    map[string]Handler
	logger      *zap.Logger
	approvalSvc *approval.Service // optional, nil means no dispatch-level approval check
	catalog     *actioncatalog.Catalog // optional, nil means no catalog enforcement
}

// DispatcherOption configures a Dispatcher.
type DispatcherOption func(*Dispatcher)

// WithApprovalService sets the approval service for high-risk action checking.
func WithApprovalService(svc *approval.Service) DispatcherOption {
	return func(d *Dispatcher) {
		d.approvalSvc = svc
	}
}

// WithCatalog sets the action catalog for production enforcement.
// When set, DispatchSafe validates actions against the catalog in
// production mode — unknown action types are rejected.
func WithCatalog(cat *actioncatalog.Catalog) DispatcherOption {
	return func(d *Dispatcher) {
		d.catalog = cat
	}
}

// NewDispatcher creates a new command dispatcher.
func NewDispatcher(logger *zap.Logger, opts ...DispatcherOption) *Dispatcher {
	d := &Dispatcher{
		handlers: make(map[string]Handler),
		logger:   logger,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
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

// highRiskActionTypes maps action types that require an approved approval
// before their handler executes.
var highRiskActionTypes = map[string]string{
	"price_adjust":    "price_change",
	"price_update":    "price_change",
	"stock_update":    "stock_update",
	"listing_publish": "listing_publish",
}

// Dispatch executes the handler registered for the given action type.
// Returns ErrHandlerNotFound if no handler is registered.
// For high-risk action types, an approved approval must exist before execution.
func (d *Dispatcher) Dispatch(ctx context.Context, actionType string, payload map[string]interface{}) (*Result, error) {
	d.mu.RLock()
	handler, ok := d.handlers[actionType]
	d.mu.RUnlock()

	if !ok {
		return nil, &HandlerNotFoundError{ActionType: actionType}
	}

	// High-risk action approval gate: verify an approved approval exists.
	if reqType, isHighRisk := highRiskActionTypes[actionType]; isHighRisk && d.approvalSvc != nil {
		if err := d.checkHighRiskApproval(ctx, actionType, reqType, payload); err != nil {
			return nil, err
		}
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
//   - dry_run: the action is validated (handler must exist, catalog is checked)
//     but never executed.
//   - sandbox: the action executes regardless of risk.
//   - production: catalog, risk, approval, and L4 block checks are enforced.
func (d *Dispatcher) DispatchSafe(ctx context.Context, action AgentAction, policy PolicyChecker) (*Result, error) {
	// Dry-run: validate only — check handler and catalog exist, never mutate.
	if action.Mode == ModeDryRun {
		d.mu.RLock()
		_, ok := d.handlers[action.ActionType]
		d.mu.RUnlock()
		if !ok {
			return nil, &HandlerNotFoundError{ActionType: action.ActionType}
		}
		if d.catalog != nil {
			if _, ok := d.catalog.Lookup(action.ActionType); !ok {
				return nil, fmt.Errorf("dry-run: action type %q is not registered in the action catalog", action.ActionType)
			}
		}
		return &Result{Success: true, BusinessID: "dry_run"}, nil
	}

	// Production mode: enforce catalog validation.
	if action.Mode == ModeProduction && d.catalog != nil {
		hasApproval := action.ApprovalID != nil
		if err := d.catalog.ValidateProduction(action.ActionType, int(action.RiskLevel), hasApproval); err != nil {
			return nil, err
		}
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

// checkHighRiskApproval verifies that an approved approval request exists for
// the given action type and target before the handler executes.
func (d *Dispatcher) checkHighRiskApproval(ctx context.Context, actionType, reqType string, payload map[string]interface{}) error {
	targetID := extractInt64(payload, "sku_id")
	if targetID == 0 {
		targetID = extractInt64(payload, "listing_id")
	}
	if targetID == 0 {
		d.logger.Warn("cannot check approval for high-risk action: no target ID in payload",
			zap.String("action_type", actionType))
		return nil
	}
	_, err := d.approvalSvc.FindApprovedByTarget("sku", targetID, reqType)
	if err != nil {
		return fmt.Errorf("%s requires approved approval: %w", actionType, err)
	}
	return nil
}

// extractInt64 extracts an int64 value from a map by key.
// Supports float64 (JSON unmarshaling), int64, and int types.
func extractInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		case int:
			return int64(n)
		}
	}
	return 0
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
