// Package toolregistry provides a centralized registry for agent-callable tools
// with hook-based middleware, circuit breaker support, and thread-safe access.
//
// This is the core of the AIOS Tool Registry infrastructure, enabling agents
// and LLMs to discover, inspect, and invoke tools with full schema metadata,
// permission checks, rate limiting, and observability.
package toolregistry

import (
	"context"
	"fmt"
	"time"
)

// RiskLevel classifies the operational risk of a tool execution.
type RiskLevel string

const (
	// RiskLow indicates read-only or non-destructive operations.
	RiskLow RiskLevel = "low"
	// RiskMedium indicates data creation or modification operations.
	RiskMedium RiskLevel = "medium"
	// RiskHigh indicates destructive or impactful operations.
	RiskHigh RiskLevel = "high"
	// RiskCritical indicates financial, legal, or system-critical operations.
	RiskCritical RiskLevel = "critical"
)

// Schema represents a simplified JSON Schema definition for tool parameters
// and return values. This subset is sufficient for LLM function calling schema
// generation and agent discovery.
type Schema struct {
	Type        string             `json:"type"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
	Format      string             `json:"format,omitempty"`
}

// CircuitConfig defines the circuit breaker configuration for a tool.
// When the failure threshold is reached, the circuit opens and rejects all
// requests until the recovery timeout expires, at which point it transitions
// to half-open state to test recovery.
type CircuitConfig struct {
	// FailureThreshold is the number of consecutive failures before tripping the circuit.
	FailureThreshold int `json:"failure_threshold"`
	// RecoveryTimeout is the duration to wait before attempting recovery (half-open state).
	RecoveryTimeout time.Duration `json:"recovery_timeout"`
	// HalfOpenMax is the maximum number of requests allowed in half-open state.
	HalfOpenMax int `json:"half_open_max"`
}

// Tool describes a single callable tool that agents can invoke.
// Each tool has a unique Name+Version combination, a JSON Schema for parameters
// and return values, permission requirements, risk classification, and an
// execution handler. Tools are registered in a ToolRegistry and invoked via
// the registry's Call method, which applies the hook chain.
type Tool struct {
	// Name is the unique identifier for the tool, e.g. "purchase_order.create".
	Name string `json:"name"`
	// Version follows semantic versioning, e.g. "1.0.0".
	Version string `json:"version"`
	// Description is a human-readable description visible to LLMs.
	Description string `json:"description"`
	// Squad indicates which squad owns or may call this tool.
	Squad string `json:"squad"`

	// Parameters defines the JSON Schema for input parameters.
	Parameters *Schema `json:"parameters,omitempty"`
	// Returns defines the JSON Schema for the return value.
	Returns *Schema `json:"returns,omitempty"`

	// RequiredPermissions lists the permissions a caller must possess.
	RequiredPermissions []string `json:"required_permissions,omitempty"`
	// RiskLevel classifies this tool's risk (low/medium/high/critical).
	RiskLevel RiskLevel `json:"risk_level"`

	// Handler is the actual execution function for this tool.
	Handler func(ctx context.Context, input map[string]interface{}) (interface{}, error)

	// CostTokens is an estimate of LLM token cost for using this tool.
	CostTokens int `json:"cost_tokens,omitempty"`
	// MaxDuration is the maximum allowed execution time. If zero, no timeout is applied.
	MaxDuration time.Duration `json:"max_duration,omitempty"`
	// CircuitBreaker configures per-tool circuit breaker settings. Nil means no circuit breaker.
	CircuitBreaker *CircuitConfig `json:"circuit_breaker,omitempty"`
	// SensitiveData indicates whether the tool handles sensitive data (PII, financial, etc.).
	SensitiveData bool `json:"sensitive_data"`
}

// Key returns a unique registry key combining name and version in the
// format "name@version". This is used as the internal map key.
func (t *Tool) Key() string {
	return fmt.Sprintf("%s@%s", t.Name, t.Version)
}

// Call executes this tool's handler with the given hooks chain applied.
// Hooks' Before methods run first in registration order; if any Before hook
// returns an error, execution is aborted and After hooks run in reverse order.
// After hooks always execute, even on handler failure, to support audit logging
// and circuit breaker state updates.
func (t *Tool) Call(ctx context.Context, input map[string]interface{}, hooks []ToolHook) (interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if input == nil {
		input = make(map[string]interface{})
	}

	// Run Before hooks
	var err error
	for _, hook := range hooks {
		ctx, err = hook.Before(ctx, t, input)
		if err != nil {
			t.runAfterHooks(ctx, input, nil, err, hooks)
			return nil, err
		}
	}

	// Execute handler with timeout if configured
	var output interface{}
	if t.MaxDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.MaxDuration)
		defer cancel()
	}
	output, err = t.Handler(ctx, input)

	// Run After hooks in reverse order
	t.runAfterHooks(ctx, input, output, err, hooks)

	if err != nil {
		return nil, err
	}
	return output, nil
}

// runAfterHooks runs After hooks in reverse order. Errors from After hooks
// are not propagated since they must not mask the handler's result — each
// hook is responsible for its own error handling and logging.
func (t *Tool) runAfterHooks(ctx context.Context, input map[string]interface{}, output interface{}, err error, hooks []ToolHook) {
	for i := len(hooks) - 1; i >= 0; i-- {
		_ = hooks[i].After(ctx, t, input, output, err)
	}
}
