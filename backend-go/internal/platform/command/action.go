package command

import (
	"errors"
	"fmt"

	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
)

// ---------------------------------------------------------------------------
// ActionStatus — the lifecycle state of an AgentAction.
// ---------------------------------------------------------------------------

// ActionStatus represents the current state of an action in its lifecycle.
type ActionStatus int

const (
	StatusSuggested      ActionStatus = 0 // action has been proposed
	StatusPendingApproval ActionStatus = 1 // waiting for human approval
	StatusApproved       ActionStatus = 2 // approval granted, pending execution
	StatusRejected       ActionStatus = 3 // approval denied
	StatusExecuting      ActionStatus = 4 // currently being executed
	StatusCompleted      ActionStatus = 5 // execution finished successfully
	StatusFailed         ActionStatus = 6 // execution finished with error
	StatusBlocked        ActionStatus = 7 // blocked by policy / guardrail
)

func (s ActionStatus) String() string {
	switch s {
	case StatusSuggested:
		return "suggested"
	case StatusPendingApproval:
		return "pending_approval"
	case StatusApproved:
		return "approved"
	case StatusRejected:
		return "rejected"
	case StatusExecuting:
		return "executing"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusBlocked:
		return "blocked"
	default:
		return fmt.Sprintf("status(%d)", s)
	}
}

// ---------------------------------------------------------------------------
// Risk levels for Agent actions.
// ---------------------------------------------------------------------------

// RiskLevel categorises an action's potential business impact.
type RiskLevel int

const (
	RiskNone   RiskLevel = 0
	RiskLow    RiskLevel = 1
	RiskMedium RiskLevel = 2
	RiskHigh   RiskLevel = 3
)

func (r RiskLevel) String() string {
	switch r {
	case RiskNone:
		return "none"
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	default:
		return fmt.Sprintf("unknown(%d)", r)
	}
}

// ParseRiskLevel converts a string to a RiskLevel. Returns RiskLow for
// unrecognised values.
func ParseRiskLevel(s string) RiskLevel {
	switch s {
	case "none":
		return RiskNone
	case "low":
		return RiskLow
	case "medium":
		return RiskMedium
	case "high", "critical":
		return RiskHigh
	default:
		return RiskLow
	}
}

// ---------------------------------------------------------------------------
// Action mode — governs whether dry-run, sandbox, or real execution.
// ---------------------------------------------------------------------------

// ActionMode controls execution behaviour.
type ActionMode int

const (
	ModeDryRun     ActionMode = 0 // validate only, never mutate
	ModeSandbox    ActionMode = 1 // execute against test/stub data
	ModeProduction ActionMode = 2 // real execution with guardrails
)

func (m ActionMode) String() string {
	switch m {
	case ModeDryRun:
		return "dry_run"
	case ModeSandbox:
		return "sandbox"
	case ModeProduction:
		return "production"
	default:
		return fmt.Sprintf("mode(%d)", m)
	}
}

// ---------------------------------------------------------------------------
// AgentAction — the canonical typed action contract.
// Every action an Agent proposes or executes must be represented by this
// struct, which carries the metadata needed for policy evaluation, approval,
// audit, idempotency, and rollback.
// ---------------------------------------------------------------------------

// AgentAction is the canonical typed action envelope for all Agent actions.
type AgentAction struct {
	ActionType      string                 `json:"action_type"`                // e.g. "price_update", "stock_alert"
	Version         string                 `json:"version"`                    // semantic version of the action schema
	AgentID         string                 `json:"agent_id"`                   // who proposed the action
	Actor           string                 `json:"actor"`                      // system or user identity executing
	TenantID        string                 `json:"tenant_id,omitempty"`        // tenant scope
	TargetType      string                 `json:"target_type"`                // "sku", "product", "order", "listing"
	TargetID        string                 `json:"target_id"`
	RiskLevel       RiskLevel              `json:"risk_level"`                 // low, medium, high
	ApprovalRequired bool                  `json:"approval_required"`
	ApprovalID      *int64                 `json:"approval_id,omitempty"`      // set when approved
	Mode            ActionMode             `json:"mode"`                       // dry_run, sandbox, production
	Status          ActionStatus           `json:"status,omitempty"`           // current lifecycle state
	IdempotencyKey  string                 `json:"idempotency_key,omitempty"`  // prevents duplicate execution
	CorrelationID   string                 `json:"correlation_id,omitempty"`   // ties to agent workflow trace
	AuditID         string                 `json:"audit_id,omitempty"`         // audit record reference
	Input           map[string]interface{} `json:"input"`
	RollbackNote    string                 `json:"rollback_note,omitempty"`     // human guidance for reversing
}

// Validate checks that the AgentAction has all required fields for execution.
// It distinguishes between structural validation (identity, mode, risk) and
// policy validation (approval for high-risk). Returns an error describing
// the first violation found, or nil if the action is well-formed.
func (a AgentAction) Validate() error {
	if a.ActionType == "" {
		return fmt.Errorf("%w: action_type is required", ErrActionValidation)
	}
	if a.AgentID == "" {
		return fmt.Errorf("%w: agent_id is required", ErrActionValidation)
	}
	if a.Actor == "" {
		return fmt.Errorf("%w: actor is required", ErrActionValidation)
	}
	if a.Mode != ModeDryRun && a.Mode != ModeSandbox && a.Mode != ModeProduction {
		return fmt.Errorf("%w: invalid or unset mode (%d)", ErrActionValidation, a.Mode)
	}
	if a.RiskLevel < RiskLow || a.RiskLevel > RiskHigh {
		return fmt.Errorf("%w: invalid or unset risk_level (%d)", ErrActionValidation, a.RiskLevel)
	}

	// High-risk actions require approval_required=true by default.
	if a.RiskLevel >= RiskHigh && !a.ApprovalRequired {
		return fmt.Errorf("%w: high risk action %q must set approval_required=true", ErrActionValidation, a.ActionType)
	}

	// Production mode for high-risk actions requires an approval ID.
	if a.Mode == ModeProduction && a.RiskLevel >= RiskHigh && a.ApprovalID == nil {
		return ErrApprovalRequired
	}

	// Dry-run must never have an approval ID or be in executing/completed status.
	if a.Mode == ModeDryRun && a.ApprovalID != nil {
		return fmt.Errorf("%w: dry_run actions must not carry an approval_id", ErrActionValidation)
	}

	return nil
}

// HighRiskActions returns the canonical list of action types that are
// considered high risk by default. Derived from the action catalog so the
// two sources never drift.
func HighRiskActions() []string {
	entries := actioncatalog.DefaultEntries()
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.RiskLevel >= actioncatalog.RiskHigh {
			out = append(out, e.ActionType)
		}
	}
	return out
}

// AuditContext returns a structured map suitable for operation log audit.
func (a AgentAction) AuditContext() map[string]interface{} {
	return map[string]interface{}{
		"action_type":       a.ActionType,
		"action_version":    a.Version,
		"agent_id":          a.AgentID,
		"actor":             a.Actor,
		"target_type":       a.TargetType,
		"target_id":         a.TargetID,
		"risk_level":        a.RiskLevel.String(),
		"mode":              a.Mode.String(),
		"status":            a.Status.String(),
		"approval_required": a.ApprovalRequired,
		"correlation_id":    a.CorrelationID,
	}
}

// ErrActionValidation is returned when an AgentAction fails structural validation.
var ErrActionValidation = errors.New("command: action validation failed")
