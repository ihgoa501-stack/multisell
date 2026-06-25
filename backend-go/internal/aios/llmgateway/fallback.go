package llmgateway

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// FallbackChain
// ---------------------------------------------------------------------------

// FallbackChain defines the contract for model fallback execution.
// The primary model is attempted first; on failure or timeout, the chain
// progresses through fallback models in priority order.
type FallbackChain interface {
	// Execute attempts to fulfill the request through the fallback chain.
	// Returns the response, whether a fallback was used (i.e. primary failed),
	// and any error if all models in the chain failed.
	Execute(ctx context.Context, provider Provider, target ModelTarget, req *Request) (*Response, bool, error)
}

// ---------------------------------------------------------------------------
// DefaultFallbackChain
// ---------------------------------------------------------------------------

// DefaultFallbackChain implements the standard fallback strategy:
//
//	Primary -> First fallback -> Second fallback -> Friendly error
//
// The default chain for Opus primary: Opus (2 retries, 15s) -> Sonnet (2 retries, 10s) -> Haiku (2 retries, 5s)
// The default chain for Sonnet primary: Sonnet (2 retries, 10s) -> Haiku (2 retries, 5s)
// The default chain for Haiku primary: Haiku (2 retries, 5s)
type DefaultFallbackChain struct{}

// Execute runs the fallback chain. It attempts the primary model (from target)
// first; on failure it descends through more available models.
func (d DefaultFallbackChain) Execute(ctx context.Context, provider Provider, target ModelTarget, req *Request) (*Response, bool, error) {
	chain := d.buildChain(target)
	var lastErr error

	for i, step := range chain {
		// Create a per-step context. WithTimeout when a deadline is configured,
		// otherwise WithCancel so we can still clean up on exit.
		var stepCtx context.Context
		var stepCancel context.CancelFunc
		if step.Timeout > 0 {
			stepCtx, stepCancel = context.WithTimeout(ctx, step.Timeout)
		} else {
			stepCtx, stepCancel = context.WithCancel(ctx)
		}

		for attempt := 0; attempt <= step.MaxRetries; attempt++ {
			stepReq := copyRequest(req)
			resp, err := provider.Chat(stepCtx, stepReq)

			if err == nil {
				stepCancel()
				resp.ModelUsed = step.Model
				return resp, i > 0, nil
			}

			lastErr = err

			// Context deadline exceeded or cancelled: do not retry this step.
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				break
			}
		}

		stepCancel()
	}

	return nil, false, fmt.Errorf("llm gateway: all models exhausted: %w", lastErr)
}

// buildChain constructs the fallback chain based on the primary model target.
// Higher-tier models get proportionally longer timeouts and more retries.
func (d DefaultFallbackChain) buildChain(target ModelTarget) []ModelTarget {
	primary := target.Model

	switch {
	case containsModel(primary, "opus"):
		return []ModelTarget{
			{Model: primary, Priority: 0, MaxRetries: 2, Timeout: 15 * time.Second, CostWeight: 15.0},
			{Model: "claude-sonnet-4", Priority: 1, MaxRetries: 2, Timeout: 10 * time.Second, CostWeight: 3.0},
			{Model: "claude-haiku-4", Priority: 2, MaxRetries: 1, Timeout: 5 * time.Second, CostWeight: 1.0},
		}
	case containsModel(primary, "sonnet"):
		return []ModelTarget{
			{Model: primary, Priority: 0, MaxRetries: 2, Timeout: 10 * time.Second, CostWeight: 3.0},
			{Model: "claude-haiku-4", Priority: 1, MaxRetries: 1, Timeout: 5 * time.Second, CostWeight: 1.0},
		}
	default:
		return []ModelTarget{
			{Model: primary, Priority: 0, MaxRetries: 2, Timeout: 5 * time.Second, CostWeight: 1.0},
		}
	}
}

// containsModel reports whether the model name contains the given fragment.
func containsModel(model, fragment string) bool {
	n := len(model)
	m := len(fragment)
	if m > n {
		return false
	}
	for i := 0; i <= n-m; i++ {
		if model[i:i+m] == fragment {
			return true
		}
	}
	return false
}

// copyRequest creates a shallow copy of a Request with a fresh Messages slice
// to avoid concurrent mutation during retries.
func copyRequest(original *Request) *Request {
	msgs := make([]Message, len(original.Messages))
	copy(msgs, original.Messages)

	tools := make([]ToolDef, len(original.Tools))
	copy(tools, original.Tools)

	return &Request{
		AgentID:     original.AgentID,
		UserID:      original.UserID,
		System:      original.System,
		Messages:    msgs,
		Tools:       tools,
		MinModel:    original.MinModel,
		MaxLatency:  original.MaxLatency,
		CacheKey:    original.CacheKey,
		BypassCache: original.BypassCache,
		Sensitive:   original.Sensitive,
	}
}
