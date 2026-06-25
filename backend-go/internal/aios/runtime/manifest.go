// Package runtime implements the AIOS Agent Runtime — lifecycle management,
// resource quotas, health monitoring, and state transitions for all agent instances.
//
// Each agent is declared via an AgentManifest and managed through a Runtime
// that tracks its state, resource consumption, heartbeat, and auto-recovery.
package runtime

import "time"

// TriggerDef describes what activates an agent.
type TriggerDef struct {
	// Type is the trigger kind: "schedule" for time-based, "event" for event-driven.
	Type string `json:"type"`

	// Interval is the schedule interval (e.g., "5m", "1h"). Only meaningful for type "schedule".
	Interval string `json:"interval,omitempty"`

	// DecisionPoint is the agent's entry-point decision this trigger maps to.
	DecisionPoint string `json:"decision_point"`
}

// ResourceLimits defines per-agent resource constraints.
// These are hard ceilings — when exceeded the agent transitions to Degraded state.
type ResourceLimits struct {
	// MaxTokensPerMinute caps LLM token consumption per rolling minute.
	MaxTokensPerMinute int `json:"max_tokens_per_minute"`

	// MaxTokensPerHour caps LLM token consumption per rolling hour.
	MaxTokensPerHour int `json:"max_tokens_per_hour"`

	// MaxAPICallsPerMin caps external API calls per rolling minute.
	MaxAPICallsPerMin int `json:"max_api_calls_per_min"`

	// MaxAPICallsPerHour caps external API calls per rolling hour.
	MaxAPICallsPerHour int `json:"max_api_calls_per_hour"`

	// MaxToolChainDepth is the maximum nesting depth of Tool->Agent->Tool calls.
	MaxToolChainDepth int `json:"max_tool_chain_depth"`

	// MaxDecisionDuration is the hard deadline for a single decision cycle.
	MaxDecisionDuration time.Duration `json:"max_decision_duration"`
}

// MemoryConfig defines the memory system settings for an agent.
type MemoryConfig struct {
	// ShortTermTTL is how long short-term (working) memory entries live.
	ShortTermTTL time.Duration `json:"short_term_ttl"`

	// LongTermEnabled enables long-term memory persistence for this agent.
	LongTermEnabled bool `json:"long_term_enabled"`

	// LongTermTTL is how long long-term memory entries live before eviction.
	LongTermTTL time.Duration `json:"long_term_ttl"`
}

// AgentManifest defines an agent's identity, capabilities, and constraints.
// Think of this as a container image manifest — it describes what an agent IS,
// not its current runtime state.
type AgentManifest struct {
	// ID is the unique agent identifier (e.g., "A5", "G3").
	ID string `json:"agent_id"`

	// Name is a human-readable label.
	Name string `json:"name"`

	// Squad is the squad/team this agent belongs to (e.g., "fulfillment", "governance").
	Squad string `json:"squad"`

	// Version is the semantic version of this agent definition.
	Version string `json:"version"`

	// Description explains this agent's purpose, visible to other agents and the UI.
	Description string `json:"description"`

	// AllowedTools lists the tools this agent may call. Empty means all tools allowed by squad.
	AllowedTools []string `json:"allowed_tools,omitempty"`

	// DeniedTools lists tools this agent is explicitly forbidden from calling.
	DeniedTools []string `json:"denied_tools,omitempty"`

	// Triggers define what activates this agent (schedules, events).
	Triggers []TriggerDef `json:"triggers"`

	// ResourceLimits caps this agent's resource consumption.
	ResourceLimits ResourceLimits `json:"resource_limits"`

	// MemoryConfig configures this agent's memory subsystem.
	MemoryConfig MemoryConfig `json:"memory"`
}
