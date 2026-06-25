// Package sdk provides the declarative agent definition format and registration
// bootstrap for AIOS — the LingMirror Agent Operation System.
//
// Use AgentDef (YAML or struct) together with RegisterAgent to declare an agent
// once and have it registered with the runtime, wired to the event bus, and
// started automatically.
package sdk

// AgentDef is the declarative definition of an agent.
type AgentDef struct {
	AgentID        string            `yaml:"agent_id"`
	Name           string            `yaml:"name"`
	Squad          string            `yaml:"squad"`
	Version        string            `yaml:"version"`
	Description    string            `yaml:"description"`
	DecisionPoints []DecisionPointDef `yaml:"decision_points"`
	Tools          []string           `yaml:"tools"`
	Triggers       []TriggerDef       `yaml:"triggers"`
	ModelHint      string             `yaml:"model_hint"`
	Autonomy       string             `yaml:"autonomy"`
	RiskFloor      string             `yaml:"risk_floor"`
	ResourceLimits ResourceLimitsDef  `yaml:"resource_limits"`
	Memory         MemoryConfigDef    `yaml:"memory"`
}

// DecisionPointDef names one decision this agent can make.
type DecisionPointDef struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// TriggerDef describes what activates an agent: a schedule tick or an event bus
// topic match.
type TriggerDef struct {
	Type          string `yaml:"type"`                     // "schedule" | "event"
	Interval      string `yaml:"interval,omitempty"`       // e.g. "5m", "1h"
	Topic         string `yaml:"topic,omitempty"`          // e.g. "order.*"
	DecisionPoint string `yaml:"decision_point"`           // entry-point decision
}

// ResourceLimitsDef defines per-agent resource constraints. Duration fields use
// Go time.Duration string format (e.g. "30s", "5m").
type ResourceLimitsDef struct {
	MaxTokensPerMinute  int    `yaml:"max_tokens_per_minute"`
	MaxTokensPerHour    int    `yaml:"max_tokens_per_hour"`
	MaxAPICallsPerMin   int    `yaml:"max_api_calls_per_min"`
	MaxAPICallsPerHour  int    `yaml:"max_api_calls_per_hour"`
	MaxToolChainDepth   int    `yaml:"max_tool_chain_depth"`
	MaxDecisionDuration string `yaml:"max_decision_duration"` // Go duration, e.g. "30s"
}

// MemoryConfigDef configures the memory subsystem for an agent. Duration fields
// use Go time.Duration string format (e.g. "5m", "24h").
type MemoryConfigDef struct {
	ShortTermTTL    string `yaml:"short_term_ttl"`      // Go duration, e.g. "5m"
	LongTermEnabled bool   `yaml:"long_term_enabled"`
	LongTermTTL     string `yaml:"long_term_ttl"`       // Go duration, e.g. "24h"
}
