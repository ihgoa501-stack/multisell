package pipeline

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// AgentHandler is a function that simulates or proxies an agent's decision logic.
// Used by RunFanOut and RunSelfCorrect to dispatch work to named agents.
type AgentHandler func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error)

// Engine executes decision pipelines.
//
// It supports four execution topologies:
//   - RunPipeline  — serial step execution
//   - RunFanOut    — parallel dispatch to multiple agents
//   - RunSelfCorrect — agent self-correction loop
//   - RunWithFallback — primary / backup fault tolerance
//
// For RunFanOut and RunSelfCorrect, callers must register handlers via
// RegisterAgentHandler before dispatching.
type Engine struct {
	logger        *zap.Logger
	agentHandlers map[string]AgentHandler
	mu            sync.RWMutex
}

// New creates a new Engine with the given logger.
func New(logger *zap.Logger) *Engine {
	return &Engine{
		logger:        logger,
		agentHandlers: make(map[string]AgentHandler),
	}
}

// RegisterAgentHandler registers a handler function for the given agent ID.
// The handler is used by RunFanOut and RunSelfCorrect to dispatch work.
// This method is thread-safe.
func (e *Engine) RegisterAgentHandler(agentID string, handler AgentHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.agentHandlers[agentID] = handler
}

func (e *Engine) getAgentHandler(agentID string) (AgentHandler, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	h, ok := e.agentHandlers[agentID]
	return h, ok
}

// RunPipeline executes a serial pipeline.
//
// Steps run in order; each step receives the previous step's output as prevOutput.
// The first step receives nil. If any step fails (or all retries are exhausted),
// the pipeline terminates early and returns a partial output slice and an error.
// Pipeline-level and per-step timeouts are applied when configured.
func (e *Engine) RunPipeline(ctx context.Context, p *Pipeline) ([]interface{}, error) {
	if p == nil {
		return nil, fmt.Errorf("pipeline: pipeline is nil")
	}

	// Apply pipeline-level timeout.
	var cancel context.CancelFunc
	if p.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, p.Timeout)
		defer cancel()
	}

	outputs := make([]interface{}, 0, len(p.Steps))
	var prevOutput interface{}

	for i, step := range p.Steps {
		e.logger.Info("running pipeline step",
			zap.String("pipeline", p.Name),
			zap.Int("step_index", i),
			zap.String("step_name", step.Name),
			zap.String("agent_id", step.AgentID),
		)

		// Apply per-step timeout as a derived context.
		stepCtx := ctx
		var stepCancel context.CancelFunc
		if step.Timeout > 0 {
			stepCtx, stepCancel = context.WithTimeout(ctx, step.Timeout)
		}

		// Execute with retry support.
		var output map[string]interface{}
		var stepErr error
		totalAttempts := 1 + step.RetryMax

		for attempt := 0; attempt < totalAttempts; attempt++ {
			// Check context cancellation before each attempt.
			select {
			case <-stepCtx.Done():
				stepErr = stepCtx.Err()
				goto stepDone
			default:
			}

			if attempt > 0 {
				e.logger.Info("retrying step",
					zap.String("step_name", step.Name),
					zap.Int("attempt", attempt+1),
					zap.Int("max_attempts", totalAttempts),
				)
			}

			output, stepErr = step.Input(stepCtx, prevOutput)
			if stepErr == nil {
				break
			}
		}

	stepDone:
		if stepCancel != nil {
			stepCancel()
		}

		if stepErr != nil {
			e.logger.Error("pipeline step failed",
				zap.String("pipeline", p.Name),
				zap.Int("step_index", i),
				zap.String("step_name", step.Name),
				zap.Error(stepErr),
			)
			return outputs[:i], fmt.Errorf("pipeline %q step %d (%s): %w", p.Name, i, step.Name, stepErr)
		}

		prevOutput = output
		outputs = append(outputs, output)
	}

	return outputs, nil
}

// RunFanOut dispatches the same input to multiple agents in parallel.
//
// Each agent ID must have a handler registered via RegisterAgentHandler.
// All registered handlers are validated before any goroutine is spawned,
// preventing partial execution on configuration errors.
// Results are returned in the same order as the agents slice.
func (e *Engine) RunFanOut(ctx context.Context, agents []string, input map[string]interface{}) ([]map[string]interface{}, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("fan-out: no agents specified")
	}

	// Validate all handlers before spawning goroutines.
	handlers := make([]AgentHandler, len(agents))
	for i, agentID := range agents {
		h, ok := e.getAgentHandler(agentID)
		if !ok {
			return nil, fmt.Errorf("fan-out: no handler registered for agent %q", agentID)
		}
		handlers[i] = h
	}

	type agentResult struct {
		index  int
		output map[string]interface{}
		err    error
	}

	resultsCh := make(chan agentResult, len(agents))
	var wg sync.WaitGroup

	for i, agentID := range agents {
		wg.Add(1)
		go func(idx int, id string, fn AgentHandler) {
			defer wg.Done()
			e.logger.Info("fan-out: dispatching to agent",
				zap.String("agent_id", id),
				zap.Int("index", idx),
			)
			out, err := fn(ctx, id, input)
			resultsCh <- agentResult{index: idx, output: out, err: err}
		}(i, agentID, handlers[i])
	}

	wg.Wait()
	close(resultsCh)

	results := make([]map[string]interface{}, len(agents))
	for r := range resultsCh {
		if r.err != nil {
			e.logger.Error("fan-out: agent failed",
				zap.Int("index", r.index),
				zap.String("agent_id", agents[r.index]),
				zap.Error(r.err),
			)
			return nil, fmt.Errorf("fan-out: agent %q failed: %w", agents[r.index], r.err)
		}
		results[r.index] = r.output
	}

	return results, nil
}

// RunSelfCorrect executes an agent decision with a self-correction loop.
//
// The agent handler produces output containing a "_self_check" key.
//   - If the value is "ok", the loop exits and returns the output.
//   - Otherwise, the engine passes the output back as "_previous_output"
//     in the enriched input and the agent regenerates.
//
// maxIter caps the total iterations. If reached, the last output is returned
// regardless of self-check status.
func (e *Engine) RunSelfCorrect(ctx context.Context, agentID string, input map[string]interface{}, maxIter int) (map[string]interface{}, error) {
	handler, ok := e.getAgentHandler(agentID)
	if !ok {
		return nil, fmt.Errorf("self-correct: no handler registered for agent %q", agentID)
	}

	if maxIter <= 0 {
		maxIter = 1
	}

	currentInput := input
	var lastOutput map[string]interface{}

	for iter := 0; iter < maxIter; iter++ {
		e.logger.Info("self-correct: iteration",
			zap.String("agent_id", agentID),
			zap.Int("iteration", iter+1),
			zap.Int("max_iter", maxIter),
		)

		output, err := handler(ctx, agentID, currentInput)
		if err != nil {
			return nil, fmt.Errorf("self-correct: agent %q iteration %d: %w", agentID, iter+1, err)
		}

		lastOutput = output

		// Check the self-evaluation result.
		if selfCheck, ok := output["_self_check"]; ok && selfCheck == "ok" {
			e.logger.Info("self-correct: passed",
				zap.String("agent_id", agentID),
				zap.Int("iterations_used", iter+1),
			)
			return output, nil
		}

		e.logger.Info("self-correct: regenerating",
			zap.String("agent_id", agentID),
			zap.Int("iteration", iter+1),
			zap.Any("self_check", output["_self_check"]),
		)

		// Enrich input with previous output context for the next iteration.
		enriched := make(map[string]interface{}, len(currentInput)+1)
		for k, v := range currentInput {
			enriched[k] = v
		}
		enriched["_previous_output"] = output
		currentInput = enriched
	}

	e.logger.Warn("self-correct: max iterations reached",
		zap.String("agent_id", agentID),
		zap.Int("max_iter", maxIter),
	)
	return lastOutput, nil
}

// RunWithFallback executes the primary pipeline first. If it fails, the
// fallback pipeline runs instead. If the fallback also fails, a combined
// error wrapping both failures is returned.
//
// On success, the return value is the last step's output of the winning
// pipeline, stored as interface{} (the underlying type is map[string]interface{}).
func (e *Engine) RunWithFallback(ctx context.Context, primary, fallback *Pipeline) (interface{}, error) {
	outputs, err := e.RunPipeline(ctx, primary)
	if err == nil {
		e.logger.Info("fallback: primary pipeline succeeded",
			zap.String("pipeline", primary.Name),
		)
		if len(outputs) > 0 {
			return outputs[len(outputs)-1], nil
		}
		return nil, nil
	}

	e.logger.Warn("fallback: primary failed, attempting fallback",
		zap.String("primary", primary.Name),
		zap.String("fallback", fallback.Name),
		zap.Error(err),
	)

	fallbackOutputs, fbErr := e.RunPipeline(ctx, fallback)
	if fbErr == nil {
		e.logger.Info("fallback: fallback pipeline succeeded",
			zap.String("pipeline", fallback.Name),
		)
		if len(fallbackOutputs) > 0 {
			return fallbackOutputs[len(fallbackOutputs)-1], nil
		}
		return nil, nil
	}

	e.logger.Error("fallback: all pipelines failed",
		zap.String("primary", primary.Name),
		zap.String("fallback", fallback.Name),
		zap.Error(fbErr),
	)
	return nil, fmt.Errorf("all pipelines failed: primary: %w; fallback: %v", err, fbErr)
}
