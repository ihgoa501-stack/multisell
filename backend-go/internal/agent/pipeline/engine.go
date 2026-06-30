package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

// RunAgentFunc is the function signature for executing a target agent.
// Implementations typically call the AI orchestrator's RunWithContext.
type RunAgentFunc func(ctx context.Context, agentID, decisionPoint string, context map[string]interface{}) error

// Engine evaluates pipeline edges against incoming events and dispatches
// target agents when conditions are met.
type Engine struct {
	runAgent RunAgentFunc
	edges    []PipelineEdge
	logger   *zap.Logger

	// breakers is the per-agent circuit breaker registry.
	// Each target agent gets its own breaker, keyed by agent ID.
	breakers *CircuitBreakerRegistry
}

// NewEngine creates a pipeline engine with the given runner function and edges.
// Pass DefaultEdges to use the standard pipeline chain defined in registry.go.
func NewEngine(runAgent RunAgentFunc, edges []PipelineEdge, logger *zap.Logger) *Engine {
	return &Engine{
		runAgent: runAgent,
		edges:    edges,
		logger:   logger,
		breakers: NewCircuitBreakerRegistry(),
	}
}

// Breakers returns the circuit breaker registry, allowing callers to
// configure per-agent breakers or check state.
func (e *Engine) Breakers() *CircuitBreakerRegistry {
	return e.breakers
}

// ConfigureBreaker sets a custom circuit breaker config for an agent.
// If no breaker exists yet for this agent, it is created with the given config.
func (e *Engine) ConfigureBreaker(agentID string, config CircuitBreakerConfig) {
	e.breakers.GetOrCreateWithConfig(agentID, config)
}

// Dispatch evaluates all registered edges against an event and triggers
// matching target agents. Each edge whose SourceTopic matches the event topic
// AND whose Condition is satisfied triggers a RunAgentFunc call.
//
// Before dispatching, the circuit breaker for the target agent is checked.
// If the breaker is OPEN, the dispatch is skipped with a warning log (degraded
// response). After the call completes, the breaker records success or failure.
func (e *Engine) Dispatch(ctx context.Context, evt eventbus.Event) error {
	for _, edge := range e.edges {
		if !matchTopic(evt.Topic, edge.SourceTopic) {
			continue
		}
		if !evaluateCondition(evt.Payload, edge.Condition) {
			continue
		}

		// --- Circuit breaker check ---
		if !e.breakers.Allow(edge.TargetAgent) {
			e.logger.Warn("circuit breaker open, dispatch degraded",
				zap.String("source_topic", evt.Topic),
				zap.String("target_agent", edge.TargetAgent),
				zap.String("target_dp", edge.TargetDP),
			)
			// Degraded — skip this edge, but continue evaluating others.
			continue
		}

		e.logger.Info("pipeline edge triggered",
			zap.String("source_topic", evt.Topic),
			zap.String("target_agent", edge.TargetAgent),
			zap.String("target_dp", edge.TargetDP),
		)

		timeout := edge.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		runCtx, cancel := context.WithTimeout(ctx, timeout)
		err := e.runAgent(runCtx, edge.TargetAgent, edge.TargetDP, evt.Payload)
		cancel()

		// --- Record breaker result ---
		e.breakers.Record(edge.TargetAgent, err == nil)

		if err != nil {
			e.logger.Error("pipeline edge dispatch failed",
				zap.String("target_agent", edge.TargetAgent),
				zap.String("target_dp", edge.TargetDP),
				zap.Error(err),
			)
			if edge.Priority > 0 {
				return fmt.Errorf("pipeline: dispatch %s/%s: %w", edge.TargetAgent, edge.TargetDP, err)
			}
		}
	}
	return nil
}

// matchTopic returns true if eventTopic matches edgeTopic. Supports exact
// match and glob suffix match (e.g. "agent.decided.A5.*").
func matchTopic(eventTopic, edgeTopic string) bool {
	if eventTopic == edgeTopic {
		return true
	}
	if strings.HasSuffix(edgeTopic, "*") {
		prefix := strings.TrimSuffix(edgeTopic, "*")
		return strings.HasPrefix(eventTopic, prefix)
	}
	return false
}

// evaluateCondition checks whether the payload satisfies the condition.
// An empty condition (no field set) always returns true.
func evaluateCondition(payload map[string]interface{}, cond Condition) bool {
	if cond.Field == "" {
		return true
	}

	val, ok := payload[cond.Field]
	if !ok {
		return false
	}

	// BoolEquals check (highest priority)
	if cond.BoolEquals != nil {
		b, ok := val.(bool)
		return ok && b == *cond.BoolEquals
	}

	// String equality
	if cond.Equals != "" {
		s, ok := val.(string)
		return ok && s == cond.Equals
	}

	// Exists check -- verify a second field exists
	if cond.Exists != "" {
		_, exists := payload[cond.Exists]
		return exists
	}

	// GT (greater than) -- supports int or float64 (JSON numbers)
	if cond.GT > 0 {
		switch v := val.(type) {
		case int:
			return v > cond.GT
		case float64:
			return int(v) > cond.GT
		}
	}

	return false
}
