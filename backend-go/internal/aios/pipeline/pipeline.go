// Package pipeline provides the Decision Pipeline for the AIOS kernel.
//
// It supports four decision execution topologies:
//   - Serial (RunPipeline): steps execute in order, each receiving the previous output.
//   - Fan-out (RunFanOut): the same input is dispatched to multiple agents in parallel.
//   - Self-correct (RunSelfCorrect): an agent produces output, self-checks it, and
//     regenerates if issues are found, up to a maximum iteration count.
//   - Fallback (RunWithFallback): a primary pipeline is attempted first; on failure
//     the backup pipeline runs instead.
package pipeline

import (
	"context"
	"time"
)

// PipelineStep is a single step in a decision pipeline.
//
// The Input function is the execution core: it receives the previous step's output
// (nil for the first step) and returns this step's result. When dispatched through
// the full AIOS stack, AgentID / DecisionPoint / ToolName would route to the Agent
// Runtime or Tool Registry; in the standalone engine they serve as metadata.
type PipelineStep struct {
	// Name is a human-readable label for this step.
	Name string

	// AgentID identifies the Agent responsible for this step.
	AgentID string

	// DecisionPoint is the agent's entry-point name for this step.
	DecisionPoint string

	// ToolName identifies the Tool to invoke for this step.
	ToolName string

	// Input is the execution function. It receives the previous step's output
	// (nil for the first step) and returns this step's result.
	Input func(ctx context.Context, prevOutput interface{}) (map[string]interface{}, error)

	// Timeout is the per-step execution deadline. Zero means no per-step timeout.
	Timeout time.Duration

	// RetryMax is the number of additional retries on failure.
	// Zero means no retries (execute exactly once).
	RetryMax int
}

// Pipeline is a named sequence of steps that produce a combined decision.
// Steps run in order; each step receives the previous step's output.
type Pipeline struct {
	// Name is a human-readable identifier for this pipeline.
	Name string

	// Steps is the ordered list of decision steps.
	Steps []PipelineStep

	// Timeout is the overall pipeline execution deadline. Zero means no timeout.
	Timeout time.Duration
}
