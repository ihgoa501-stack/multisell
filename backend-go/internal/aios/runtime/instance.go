package runtime

import (
	"fmt"
	"sync"
	"time"
)

// AgentInstanceState represents the runtime lifecycle state of an agent instance.
type AgentInstanceState int

// Agent instance lifecycle states.
// See the lifecycle diagram in the AIOS architecture doc (Section 4.4).
const (
	// StateInit means the agent is registered but not yet started.
	StateInit AgentInstanceState = iota

	// StateReady means the agent is started and waiting for a trigger to fire.
	StateReady

	// StateActive means the agent is currently executing a decision cycle.
	StateActive

	// StateIdle means the agent has completed a decision cycle and is idle.
	StateIdle

	// StateSuspended means the agent has been manually paused.
	StateSuspended

	// StateDegraded means the agent is running in degraded mode due to
	// resource limit violations or circuit-breaker activation.
	StateDegraded

	// StateCrashed means the agent has crashed due to heartbeat timeout.
	StateCrashed

	// StateStopped is the terminal state — the agent has been stopped.
	StateStopped
)

// String returns the human-readable name of the state.
func (s AgentInstanceState) String() string {
	switch s {
	case StateInit:
		return "init"
	case StateReady:
		return "ready"
	case StateActive:
		return "active"
	case StateIdle:
		return "idle"
	case StateSuspended:
		return "suspended"
	case StateDegraded:
		return "degraded"
	case StateCrashed:
		return "crashed"
	case StateStopped:
		return "stopped"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}


// AgentInstance is a running agent with full lifecycle tracking.
// Each instance wraps an AgentManifest and tracks runtime state, resource
// consumption, health metrics, and failure history.
type AgentInstance struct {
	// Manifest is the agent's identity and capability definition.
	// Shared across lookups — the manifest is immutable after registration.
	Manifest *AgentManifest

	// State is the agent's current lifecycle state.
	State AgentInstanceState `json:"state"`

	// StartedAt is when the agent was last started.
	StartedAt time.Time `json:"started_at"`

	// LastActive is when the agent last completed a decision cycle.
	LastActive time.Time `json:"last_active"`

	// Resource tracking — cumulative counters.
	TokensUsed int64 `json:"tokens_used"`
	APICalls   int64 `json:"api_calls"`

	// Health tracking.
	Heartbeat     time.Time `json:"heartbeat"`
	FailureCount  int       `json:"failure_count"`
	MissedHeartbeats int   `json:"missed_heartbeats"`

	// DegradedSince tracks when the instance entered Degraded state.
	DegradedSince *time.Time `json:"degraded_since,omitempty"`

	// Crash recovery tracking.
	RecoveryAttempts   int       `json:"recovery_attempts"`
	LastRecoveryAttempt time.Time `json:"last_recovery_attempt"`

	mu sync.RWMutex
}
