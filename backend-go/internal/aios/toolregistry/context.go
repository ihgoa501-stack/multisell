package toolregistry

import "context"

// Package-level context key strings used across packages (toolregistry + guardrails)
// to avoid import cycles. Both packages refer to the same string constants so
// context values set by one are readable by the other.
const (
	ctxAgentID       = "toolregistry-agent-id"
	ctxExecutionMode = "toolregistry-execution-mode"
	ctxUserID        = "toolregistry-user-id"
	ctxApprovalID    = "toolregistry-approval-id"
)

// WithAgentID attaches the calling agent's ID to the context so hooks and
// guardrails can identify which agent is invoking a tool.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, ctxAgentID, agentID)
}

// GetAgentID extracts the agent ID from context. Returns empty string if not set.
func GetAgentID(ctx context.Context) string {
	v, _ := ctx.Value(ctxAgentID).(string)
	return v
}

// WithExecutionMode attaches the execution mode ("production", "sandbox", "dry_run")
// to the context for hook enforcement.
func WithExecutionMode(ctx context.Context, mode string) context.Context {
	return context.WithValue(ctx, ctxExecutionMode, mode)
}

// GetExecutionMode extracts the execution mode from context.
// Returns "sandbox" if not set (safe default).
func GetExecutionMode(ctx context.Context) string {
	v, _ := ctx.Value(ctxExecutionMode).(string)
	if v == "" {
		return "sandbox"
	}
	return v
}

// WithUserID attaches the user ID to the context.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, ctxUserID, userID)
}

// GetUserID extracts the user ID from context. Returns 0 if not set.
func GetUserID(ctx context.Context) int64 {
	v, _ := ctx.Value(ctxUserID).(int64)
	return v
}

// WithApprovalID attaches an approval ID to the context.
func WithApprovalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, ctxApprovalID, id)
}

// GetApprovalID retrieves the approval ID from context. Returns 0 if not set.
func GetApprovalID(ctx context.Context) int64 {
	v, _ := ctx.Value(ctxApprovalID).(int64)
	return v
}
