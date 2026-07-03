package toolbridge

import (
	"errors"
	"fmt"
)

// ---------------------------------------------------------------------------
// Tool categories — every tool is read-only, suggestion, or mutation.
// ---------------------------------------------------------------------------

// ToolCategory describes the side-effect class of a tool.
type ToolCategory int

const (
	ToolCategoryRead       ToolCategory = 0 // read-only: search, fetch, inspect
	ToolCategorySuggestion ToolCategory = 1 // analyse and recommend, no side effects
	ToolCategoryMutation   ToolCategory = 2 // create, update, delete, publish, sync
)

func (c ToolCategory) String() string {
	switch c {
	case ToolCategoryRead:
		return "read"
	case ToolCategorySuggestion:
		return "suggestion"
	case ToolCategoryMutation:
		return "mutation"
	default:
		return fmt.Sprintf("category(%d)", c)
	}
}

// RiskLevel returns the recommended risk level for this tool category.
func (c ToolCategory) RiskLevel() int {
	switch c {
	case ToolCategoryRead:
		return 1 // low
	case ToolCategorySuggestion:
		return 1 // low
	case ToolCategoryMutation:
		return 3 // high
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// ToolCall — the envelope for calling any tool through the bridge.
// ---------------------------------------------------------------------------

// ExecutionMode mirrors command.ActionMode without importing it.
type ExecutionMode int

const (
	ModeDryRun     ExecutionMode = 0
	ModeSandbox    ExecutionMode = 1
	ModeProduction ExecutionMode = 2
)

func (m ExecutionMode) String() string {
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

// ToolCall is the typed envelope for invoking a tool through ToolBridge.
type ToolCall struct {
	ToolName         string                 `json:"tool_name"`
	Version          string                 `json:"version"`
	Category         ToolCategory           `json:"category"`
	Mode             ExecutionMode          `json:"mode"`
	Input            map[string]interface{} `json:"input"`
	ApprovalID       *int64                 `json:"approval_id,omitempty"`
	IdempotencyKey   string                 `json:"idempotency_key,omitempty"`
	CorrelationID    string                 `json:"correlation_id,omitempty"`
}

// ToolResult is the typed result of a tool call.
type ToolResult struct {
	Success      bool                   `json:"success"`
	Data         map[string]interface{} `json:"data,omitempty"`
	ErrorMessage string                 `json:"error,omitempty"`
	Mode         ExecutionMode          `json:"mode"`
}

// ---------------------------------------------------------------------------
// Approval and mode validation.
// ---------------------------------------------------------------------------

// Validate checks whether a tool call can proceed based on category and mode.
// Returns nil if the call is allowed, or an error describing why it is blocked.
func (tc ToolCall) Validate() error {
	// Dry-run is always allowed.
	if tc.Mode == ModeDryRun {
		return nil
	}

	// Production mutation calls require approval.
	if tc.Mode == ModeProduction && tc.Category == ToolCategoryMutation {
		if tc.ApprovalID == nil {
			return ErrMutationRequiresApproval
		}
	}

	// Sandbox mutation calls are allowed (they operate on test data).
	return nil
}

// ErrMutationRequiresApproval is returned when a production mutation tool call
// is attempted without a valid approval ID.
// NOTE: toolregistry also defines ErrMutationRequiresApproval; these are
// separate packages. Use the one from your import context.
var ErrMutationRequiresApproval = errors.New("toolbridge: mutation tools require approval in production mode")
