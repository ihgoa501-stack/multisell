package command

import "fmt"

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
	ActionType     string                 `json:"action_type"`     // e.g. "price_update", "stock_alert"
	Version        string                 `json:"version"`         // semantic version of the action schema
	AgentID        string                 `json:"agent_id"`        // who proposed the action
	Actor          string                 `json:"actor"`           // system or user identity executing
	TenantID       string                 `json:"tenant_id,omitempty"`
	TargetType     string                 `json:"target_type"`     // "sku", "product", "order", "listing"
	TargetID       string                 `json:"target_id"`
	RiskLevel      RiskLevel              `json:"risk_level"`
	ApprovalRequired bool                 `json:"approval_required"`
	ApprovalID     *int64                 `json:"approval_id,omitempty"` // set when approved
	Mode           ActionMode             `json:"mode"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	Input          map[string]interface{} `json:"input"`
	RollbackNote   string                 `json:"rollback_note,omitempty"` // human guidance for reversing
}

// AuditContext returns a structured map suitable for operation log audit.
func (a AgentAction) AuditContext() map[string]interface{} {
	return map[string]interface{}{
		"action_type": a.ActionType,
		"action_version": a.Version,
		"agent_id":       a.AgentID,
		"actor":          a.Actor,
		"target_type":    a.TargetType,
		"target_id":      a.TargetID,
		"risk_level":     a.RiskLevel.String(),
		"mode":           a.Mode.String(),
		"approval_required": a.ApprovalRequired,
	}
}
