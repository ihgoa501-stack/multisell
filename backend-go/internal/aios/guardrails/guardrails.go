// Package guardrails provides a layered guardrail system for the AIOS platform.
//
// Guardrails are defensive checks that protect the AIOS kernel from malicious
// or erroneous inputs, unauthorized tool calls, invalid outputs, and
// unrecoverable actions. The system is organized into five layers:
//
//   - L1 (Input Guard): Prompt injection detection on raw agent input.
//   - L2 (Call Guard): Permission and rate-limit checks on tool invocations.
//   - L3 (Output Guard): Schema and business-rule validation on LLM output.
//   - L4 (Execution Guard): Financial thresholds, dual-approval, risk checks.
//   - L5 (Rollback Guard): Compensating-action registration for reversibility.
//
// This implementation covers L1 and L2. L3-L5 are stubbed for Phase 2.
package guardrails

import (
	"context"

	"go.uber.org/zap"
)

// Guardrail defines a single check in the guardrail chain.
//
// Each guardrail implementation focuses on one layer of defence and returns
// a GuardResult that the Chain uses to decide whether to continue, warn,
// block, or request a retry with corrected input.
type Guardrail interface {
	// Name returns a human-readable identifier for this guardrail (e.g.
	// "prompt_injection_guard", "permission_guard").
	Name() string

	// Check runs the guardrail against the given input. A nil error means
	// the check completed successfully (even if the result indicates a
	// violation). An error means the check itself failed and the caller
	// should treat the action as unsafe-by-default.
	Check(ctx context.Context, input *GuardInput) (*GuardResult, error)
}

// GuardInput is the unified input for all guardrail checks.
//
// Fields are grouped by guardrail layer. L1 uses RawInput, L2 uses ToolName
// /ToolInput/ToolCallCount, and L3-L5 fields are schema stubs for Phase 2.
type GuardInput struct {
	Level    int    `json:"level"`
	AgentID  string `json:"agent_id"`
	UserID   int64  `json:"user_id"`
	TenantID int64  `json:"tenant_id"`

	// L1: Input guard — the raw text the agent received.
	RawInput string `json:"raw_input,omitempty"`

	// L2: Call guard — the tool the agent wants to invoke.
	ToolName      string                 `json:"tool_name,omitempty"`
	ToolInput     map[string]interface{} `json:"tool_input,omitempty"`
	ToolCallCount map[string]int         `json:"tool_call_count,omitempty"` // per-tool call counts

	// L3-L5 fields (schema stub, filled in Phase 2).
	RawOutput    string      `json:"raw_output,omitempty"`
	OutputSchema interface{} `json:"output_schema,omitempty"`
}

// GuardResult from a single guardrail check.
//
// The Chain uses these fields as follows:
//   - Pass = true: the input/call is acceptable.
//   - Blocked = true: the action MUST be rejected (non-retryable).
//   - Retry = true: the action MAY be retried with corrected input.
//   - Reason: a human-readable explanation.
//   - Risk: severity classification ("low", "medium", "high", "critical").
type GuardResult struct {
	Pass    bool   `json:"pass"`
	Blocked bool   `json:"blocked"`
	Retry   bool   `json:"retry"`
	Reason  string `json:"reason"`
	Risk    string `json:"risk"`
}

// Chain runs all registered guardrails in order. If any guardrail returns
// Blocked=true the chain stops immediately and returns that result.
//
// If no guardrail blocks but one or more return a warning (Pass=false,
// Blocked=false), the chain returns the first warning result so the caller
// can inspect the risk. If all guardrails pass, a default pass result is
// returned. An empty chain always returns a pass result.
type Chain struct {
	guardrails []Guardrail
	logger     *zap.Logger
}

// NewChain creates an empty guardrail chain backed by a no-op logger.
func NewChain() *Chain {
	return &Chain{
		logger: zap.NewNop(),
	}
}

// NewChainWithLogger creates a chain with the given logger.
func NewChainWithLogger(logger *zap.Logger) *Chain {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Chain{
		logger: logger,
	}
}

// Add appends a guardrail to the end of the chain.
//
// Guardrails execute in registration order. Add early guards (L1) first
// so that cheap checks run before expensive ones.
func (c *Chain) Add(g Guardrail) {
	c.guardrails = append(c.guardrails, g)
}

// Check runs every guardrail in the chain against the input.
//
// Behaviour:
//   - Guardrails execute in registration order.
//   - If any guardrail returns Blocked=true, the chain stops immediately
//     and returns that result. Subsequent guardrails are NOT run.
//   - If a guardrail returns a warning (Pass=false, Blocked=false), the
//     chain records it as the first warning and continues. If a later
//     guardrail blocks, the block takes precedence.
//   - If all guardrails complete without any Blocked=true, the first
//     warning result (if any) is returned. Otherwise a default pass result.
//   - If any guardrail errors, the chain is aborted and the error returned.
//   - An empty chain passes immediately.
func (c *Chain) Check(ctx context.Context, input *GuardInput) (*GuardResult, error) {
	if len(c.guardrails) == 0 {
		c.logger.Debug("empty guardrail chain — passing by default")
		return &GuardResult{
			Pass:    true,
			Blocked: false,
			Retry:   false,
			Reason:  "no guardrails configured",
			Risk:    "low",
		}, nil
	}

	var firstWarn *GuardResult

	for _, g := range c.guardrails {
		result, err := g.Check(ctx, input)
		if err != nil {
			c.logger.Error("guardrail check failed with error",
				zap.String("guardrail", g.Name()),
				zap.Error(err),
			)
			return nil, err
		}

		c.logger.Info("guardrail result",
			zap.String("guardrail", g.Name()),
			zap.Bool("pass", result.Pass),
			zap.Bool("blocked", result.Blocked),
			zap.String("risk", result.Risk),
			zap.String("reason", result.Reason),
		)

		if result.Blocked {
			return result, nil
		}

		// Record the first warning (non-passing, non-blocking) result.
		if !result.Pass && firstWarn == nil {
			firstWarn = result
		}
	}

	// If any guardrail warned, propagate the first warning.
	if firstWarn != nil {
		return firstWarn, nil
	}

	return &GuardResult{
		Pass:    true,
		Blocked: false,
		Retry:   false,
		Reason:  "all guardrails passed",
		Risk:    "low",
	}, nil
}

// ToolCallCheck returns a function that runs the guardrail chain against a
// tool invocation. It builds a GuardInput using shared string keys so values
// set by toolregistry.WithAgentID / WithUserID are readable across packages.
func (c *Chain) ToolCallCheck() func(ctx context.Context, toolName string, toolInput map[string]interface{}) (*GuardResult, error) {
	return func(ctx context.Context, toolName string, toolInput map[string]interface{}) (*GuardResult, error) {
		agentID, _ := ctx.Value(ctxAgentID).(string)
		userID, _ := ctx.Value(ctxUserID).(int64)
		inp := &GuardInput{
			AgentID:   agentID,
			ToolName:  toolName,
			ToolInput: toolInput,
			UserID:    userID,
		}
		return c.Check(ctx, inp)
	}
}

// RollbackGuard returns the RollbackGuard instance from the chain, or nil
// if none is registered. This allows wiring code to access the rollback
// guard for recording compensating actions directly.
func (c *Chain) RollbackGuard() *RollbackGuard {
	for _, g := range c.guardrails {
		if rg, ok := g.(*RollbackGuard); ok {
			return rg
		}
	}
	return nil
}

// Context key strings matching toolregistry's keys. String keys avoid import
// cycles — both packages read/write the same context values.
const (
	ctxAgentID = "toolregistry-agent-id"
	ctxUserID  = "toolregistry-user-id"
)
