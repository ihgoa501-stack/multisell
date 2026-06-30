// Package pipeline provides a declarative DAG-based engine for chaining agent
// decisions through event bus subscriptions. Instead of hard-coding handler
// callbacks in router.go, edges are declared as data (PipelineEdge) and
// evaluated at runtime by Engine.Dispatch.
package pipeline

import "time"

// Condition defines when a pipeline edge should trigger based on the event
// payload. Only one rule is evaluated per condition -- the first matching rule
// wins in order: BoolEquals > Equals > Exists > GT. An empty Condition (all
// zero) always evaluates to true.
type Condition struct {
	Field      string `json:"field"`
	Equals     string `json:"equals,omitempty"`      // string equality
	GT         int    `json:"gt,omitempty"`           // greater-than (int or float64)
	Exists     string `json:"exists,omitempty"`       // key exists in payload
	BoolEquals *bool  `json:"bool_equals,omitempty"` // exact boolean match
}

// PipelineEdge declares a single edge in the agent decision pipeline.
// When an event matching SourceTopic is published AND the Condition is met,
// the engine dispatches the target agent at TargetDP with the event payload.
type PipelineEdge struct {
	SourceTopic string        `json:"source_topic"`
	Condition   Condition     `json:"condition"`
	TargetAgent string        `json:"target_agent"`
	TargetDP    string        `json:"target_dp"`
	Timeout     time.Duration `json:"timeout"`
	Priority    int           `json:"priority"`    // higher = more important; errors from high-priority edges are returned
	MaxRetries  int           `json:"max_retries"`
}
