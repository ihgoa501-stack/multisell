package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/observability"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

// Health-check constants.
const (
	// MaxMissedHeartbeats is the number of consecutive missed heartbeats
	// tolerated before the agent is marked as Crashed.
	MaxMissedHeartbeats = 3

	// HealthCheckInterval is the expected interval between agent heartbeats.
	// Any heartbeat older than this counts as one missed beat.
	HealthCheckInterval = 10 * time.Second

	// RecoveryInterval is the minimum wait between crash-recovery attempts.
	RecoveryInterval = 5 * time.Second

	// MaxRecoveryAttempts is the maximum number of auto-recovery attempts
	// before a crashed agent is left permanently Crashed.
	MaxRecoveryAttempts = 3
)

// Sentinel errors returned by Runtime methods.
var (
	ErrAgentNotFound          = errors.New("agent not found")
	ErrAgentAlreadyRegistered = errors.New("agent already registered")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrAgentAlreadyRunning    = errors.New("agent is already in a running state")
	ErrAgentNotRunning        = errors.New("agent is not in a running state")
	ErrLimitExceeded          = errors.New("resource limit exceeded")
)

// Runtime manages all agent instances — lifecycle, resource quotas, health
// monitoring, and automatic crash recovery.
type Runtime struct {
	instances map[string]*AgentInstance
	events    *eventbus.Bus
	logger    *zap.Logger
	mu        sync.RWMutex
}

// New creates a new Runtime with the given event bus and logger.
func New(logger *zap.Logger, bus *eventbus.Bus) *Runtime {
	return &Runtime{
		instances: make(map[string]*AgentInstance),
		events:    bus,
		logger:    logger,
	}
}

// ---------------------------------------------------------------------------
// Agent lifecycle
// ---------------------------------------------------------------------------

// RegisterAgent registers an agent manifest and creates a runtime instance
// in Init state. It does NOT start the agent — it only makes it known to the
// system. Returns ErrAgentAlreadyRegistered if the agent ID already exists.
func (r *Runtime) RegisterAgent(manifest AgentManifest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.instances[manifest.ID]; exists {
		return fmt.Errorf("%w: agent %q", ErrAgentAlreadyRegistered, manifest.ID)
	}

	inst := &AgentInstance{
		Manifest: &manifest,
		State:    StateInit,
	}

	r.instances[manifest.ID] = inst

	r.logger.Info("agent registered",
		zap.String("agent_id", manifest.ID),
		zap.String("name", manifest.Name),
		zap.String("squad", manifest.Squad),
		zap.String("version", manifest.Version))

	return nil
}

// StartAgent transitions an agent to Ready state.
	// - Init -> Ready: first start
	// - Suspended -> Ready: resume
	// - Ready/Idle -> Ready: no-op (already started)
	// - Stopped or other states: returns ErrInvalidStateTransition.
func (r *Runtime) StartAgent(agentID string) error {
	inst, err := r.getInstance(agentID)
	if err != nil {
		return err
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	switch inst.State {
	case StateReady, StateIdle:
		// Already started or idle — ensure heartbeat is current and return.
		inst.Heartbeat = time.Now()
		return nil
	case StateInit, StateSuspended:
		oldState := inst.State
		inst.State = StateReady
		inst.StartedAt = time.Now()
		inst.Heartbeat = time.Now()
		inst.LastActive = time.Now()
		inst.MissedHeartbeats = 0
		r.logger.Info("agent started",
			zap.String("agent_id", agentID),
			zap.String("from_state", oldState.String()))
		r.publishEvent(context.Background(), "agent."+agentID+".started", map[string]interface{}{
			"agent_id":   agentID,
			"from_state": oldState.String(),
		})
		return nil
	default:
		return fmt.Errorf("%w: cannot start agent from state %s",
			ErrInvalidStateTransition, inst.State)
	}
}

// StopAgent transitions an agent to Stopped state (terminal).
// It can be called from any state and is idempotent (Stopped -> Stopped is a no-op).
func (r *Runtime) StopAgent(agentID string) error {
	inst, err := r.getInstance(agentID)
	if err != nil {
		return err
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.State == StateStopped {
		return nil
	}

	oldState := inst.State
	inst.State = StateStopped

	r.logger.Info("agent stopped",
		zap.String("agent_id", agentID),
		zap.String("from_state", oldState.String()))

	r.publishEvent(context.Background(), "agent."+agentID+".stopped", map[string]interface{}{
		"agent_id":   agentID,
		"from_state": oldState.String(),
	})

	return nil
}

// SuspendAgent transitions an agent from Ready or Idle to Suspended.
// A suspended agent can be resumed with StartAgent.
func (r *Runtime) SuspendAgent(agentID string) error {
	inst, err := r.getInstance(agentID)
	if err != nil {
		return err
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.State != StateReady && inst.State != StateIdle {
		return fmt.Errorf("%w: can only suspend a Ready or Idle agent, current state: %s",
			ErrInvalidStateTransition, inst.State)
	}

	oldState := inst.State
	inst.State = StateSuspended

	r.logger.Info("agent suspended",
		zap.String("agent_id", agentID),
		zap.String("from_state", oldState.String()))

	r.publishEvent(context.Background(), "agent."+agentID+".suspended", map[string]interface{}{
		"agent_id":   agentID,
		"from_state": oldState.String(),
	})

	return nil
}

// ---------------------------------------------------------------------------
// Query methods
// ---------------------------------------------------------------------------

// GetInstance returns a snapshot copy of the agent instance.
// Returns false if the agent is not found.
func (r *Runtime) GetInstance(agentID string) (*AgentInstance, bool) {
	r.mu.RLock()
	inst, exists := r.instances[agentID]
	r.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Return a deep copy of the value fields.
	inst.mu.RLock()
	defer inst.mu.RUnlock()

	copy := &AgentInstance{
		Manifest:           inst.Manifest,
		State:              inst.State,
		StartedAt:          inst.StartedAt,
		LastActive:         inst.LastActive,
		TokensUsed:         inst.TokensUsed,
		APICalls:           inst.APICalls,
		Heartbeat:          inst.Heartbeat,
		FailureCount:       inst.FailureCount,
		MissedHeartbeats:   inst.MissedHeartbeats,
		RecoveryAttempts:   inst.RecoveryAttempts,
		LastRecoveryAttempt: inst.LastRecoveryAttempt,
	}

	if inst.DegradedSince != nil {
		t := *inst.DegradedSince
		copy.DegradedSince = &t
	}

	return copy, true
}

// ListInstances returns all agent instances, optionally filtered by state.
// If no state arguments are provided, all instances are returned.
func (r *Runtime) ListInstances(states ...AgentInstanceState) []*AgentInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*AgentInstance

	for _, inst := range r.instances {
		if len(states) == 0 {
			// Return a safe copy.
			result = append(result, copyInstance(inst))
			continue
		}
		inst.mu.RLock()
		state := inst.State
		inst.mu.RUnlock()

		for _, s := range states {
			if state == s {
				result = append(result, copyInstance(inst))
				break
			}
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Resource tracking
// ---------------------------------------------------------------------------

// Heartbeat updates the agent's heartbeat timestamp and resets the
// MissedHeartbeats counter. Returns ErrAgentNotRunning if the agent is not in
// a running state (Ready, Active, or Idle).
func (r *Runtime) Heartbeat(agentID string) error {
	inst, err := r.getInstance(agentID)
	if err != nil {
		return err
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.State != StateReady && inst.State != StateActive && inst.State != StateIdle {
		if inst.State == StateDegraded {
			// Allow heartbeats in Degraded state so health monitoring still works.
			inst.Heartbeat = time.Now()
			inst.MissedHeartbeats = 0
			return nil
		}
		return fmt.Errorf("%w: current state is %s", ErrAgentNotRunning, inst.State)
	}

	inst.Heartbeat = time.Now()
	inst.MissedHeartbeats = 0
	return nil
}

// RecordUsage updates an agent's resource consumption counters.
// If the update causes any resource limit to be exceeded, the agent is
// transitioned to Degraded state.
func (r *Runtime) RecordUsage(agentID string, tokens int, apiCalls int) error {
	inst, err := r.getInstance(agentID)
	if err != nil {
		return err
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.State != StateReady && inst.State != StateActive &&
		inst.State != StateIdle && inst.State != StateDegraded {
		return fmt.Errorf("%w: current state is %s", ErrAgentNotRunning, inst.State)
	}

	inst.TokensUsed += int64(tokens)
	inst.APICalls += int64(apiCalls)

	ok, reason := checkLimitsLocked(inst)
	if !ok && inst.State != StateDegraded {
		oldState := inst.State
		now := time.Now()
		inst.State = StateDegraded
		inst.DegradedSince = &now

		r.logger.Warn("agent degraded due to resource limit",
			zap.String("agent_id", agentID),
			zap.String("reason", reason),
			zap.String("old_state", oldState.String()))

		r.publishEvent(context.Background(), "agent."+agentID+".degraded", map[string]interface{}{
			"agent_id":   agentID,
			"reason":     reason,
			"from_state": oldState.String(),
		})
	}

	return nil
}

// CheckLimits checks whether the agent has remaining resource capacity.
// Returns (true, "") if within limits, or (false, reason) if exceeded.
func (r *Runtime) CheckLimits(agentID string) (bool, string) {
	inst, err := r.getInstance(agentID)
	if err != nil {
		return false, err.Error()
	}

	inst.mu.RLock()
	defer inst.mu.RUnlock()

	return checkLimitsLocked(inst)
}

// checkLimitsLocked checks resource limits while the instance's read lock is held.
func checkLimitsLocked(inst *AgentInstance) (bool, string) {
	limits := inst.Manifest.ResourceLimits

	if limits.MaxTokensPerHour > 0 && inst.TokensUsed > int64(limits.MaxTokensPerHour) {
		return false, fmt.Sprintf("tokens used %d exceeds hourly limit %d",
			inst.TokensUsed, limits.MaxTokensPerHour)
	}
	if limits.MaxAPICallsPerHour > 0 && inst.APICalls > int64(limits.MaxAPICallsPerHour) {
		return false, fmt.Sprintf("API calls %d exceeds hourly limit %d",
			inst.APICalls, limits.MaxAPICallsPerHour)
	}
	if limits.MaxTokensPerMinute > 0 && inst.TokensUsed > int64(limits.MaxTokensPerMinute) {
		return false, fmt.Sprintf("tokens used %d exceeds minute limit %d",
			inst.TokensUsed, limits.MaxTokensPerMinute)
	}
	if limits.MaxAPICallsPerMin > 0 && inst.APICalls > int64(limits.MaxAPICallsPerMin) {
		return false, fmt.Sprintf("API calls %d exceeds minute limit %d",
			inst.APICalls, limits.MaxAPICallsPerMin)
	}

	return true, ""
}

// ---------------------------------------------------------------------------
// Health monitoring
// ---------------------------------------------------------------------------

// HealthCheck runs periodic health checks on all agent instances.
//
// For agents in Ready, Active, or Idle states, it checks heartbeat freshness.
// If a heartbeat is stale (older than HealthCheckInterval), it increments
// MissedHeartbeats. Once MissedHeartbeats exceeds MaxMissedHeartbeats, the
// agent transitions to Crashed.
//
// For agents in Crashed state, it attempts auto-recovery with up to
// MaxRecoveryAttempts tries, each spaced at least RecoveryInterval apart.
// If recovery succeeds, the agent transitions back to Ready.
func (r *Runtime) HealthCheck(ctx context.Context) {
	now := time.Now()

	r.mu.RLock()
	agentIDs := make([]string, 0, len(r.instances))
	for id := range r.instances {
		agentIDs = append(agentIDs, id)
	}
	r.mu.RUnlock()

	for _, id := range agentIDs {
		inst, err := r.getInstance(id)
		if err != nil {
			continue
		}

		inst.mu.Lock()

		switch inst.State {
		case StateReady, StateActive, StateIdle:
			r.checkHeartbeat(ctx, inst, now)

		case StateCrashed:
			r.tryRecovery(ctx, inst, now)

		case StateDegraded:
			// Update heartbeat if agent is still sending heartbeats while degraded.
			// This prevents a degraded agent from being subsequently marked as crashed.
			r.checkHeartbeat(ctx, inst, now)
		}

		// Set Prometheus metrics for this agent instance.
		agentID := inst.Manifest.ID
		observability.AgentMissedHeartbeats.WithLabelValues(agentID).Set(float64(inst.MissedHeartbeats))
		observability.AgentInstanceState.WithLabelValues(agentID).Set(float64(agentStateToCode(inst.State)))

		inst.mu.Unlock()
	}
}

// checkHeartbeat checks whether the agent's heartbeat is still fresh.
// If stale, it increments MissedHeartbeats and crashes the agent if the
// threshold is exceeded.
func (r *Runtime) checkHeartbeat(ctx context.Context, inst *AgentInstance, now time.Time) {
	heartbeatAge := now.Sub(inst.Heartbeat)

	if heartbeatAge <= HealthCheckInterval {
		// Heartbeat is fresh — reset missed counter.
		inst.MissedHeartbeats = 0
		return
	}

	// Heartbeat is stale — count one more missed beat.
	inst.MissedHeartbeats++

	if inst.MissedHeartbeats < MaxMissedHeartbeats {
		// Still within tolerance.
		return
	}

	// Missed heartbeat threshold exceeded — mark as crashed.
	r.logger.Warn("agent crashed due to heartbeat timeout",
		zap.String("agent_id", inst.Manifest.ID),
		zap.Int("missed_heartbeats", inst.MissedHeartbeats),
		zap.Duration("heartbeat_age", heartbeatAge),
		zap.String("state", inst.State.String()))

	inst.State = StateCrashed
	inst.FailureCount++

	r.publishEvent(ctx, "agent."+inst.Manifest.ID+".crashed", map[string]interface{}{
		"agent_id":          inst.Manifest.ID,
		"heartbeat_age_ms":  heartbeatAge.Milliseconds(),
		"missed_heartbeats": inst.MissedHeartbeats,
		"from_state":        "idle_or_ready",
	})
}

// tryRecovery attempts to auto-recover a crashed agent.
// Recovery is attempted at most MaxRecoveryAttempts times, with at least
// RecoveryInterval between attempts.
func (r *Runtime) tryRecovery(ctx context.Context, inst *AgentInstance, now time.Time) {
	if inst.RecoveryAttempts >= MaxRecoveryAttempts {
		return
	}

	if now.Sub(inst.LastRecoveryAttempt) < RecoveryInterval {
		return
	}

	// Recover the agent.
	inst.State = StateReady
	inst.RecoveryAttempts++
	inst.LastRecoveryAttempt = now
	inst.MissedHeartbeats = 0
	inst.Heartbeat = now
	inst.LastActive = now

	r.logger.Info("agent auto-recovered",
		zap.String("agent_id", inst.Manifest.ID),
		zap.Int("attempt", inst.RecoveryAttempts),
		zap.Int("max_attempts", MaxRecoveryAttempts))

	r.publishEvent(ctx, "agent."+inst.Manifest.ID+".recovered", map[string]interface{}{
		"agent_id":          inst.Manifest.ID,
		"recovery_attempts": inst.RecoveryAttempts,
	})
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// agentStateToCode maps agent lifecycle states to Prometheus metric codes:
// 0 = Running (Ready, Active, Idle), 1 = Degraded, 2 = Stopped (all others).
func agentStateToCode(s AgentInstanceState) int {
	switch s {
	case StateReady, StateActive, StateIdle:
		return 0 // Running
	case StateDegraded:
		return 1 // Degraded
	default:
		return 2 // Stopped
	}
}

// getInstance returns the agent instance by ID, acquiring the runtime read
// lock briefly to find the pointer, then returning it. The caller must
// acquire the instance's own mutex before reading or writing its fields.
func (r *Runtime) getInstance(agentID string) (*AgentInstance, error) {
	r.mu.RLock()
	inst, exists := r.instances[agentID]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrAgentNotFound, agentID)
	}
	return inst, nil
}

// publishEvent publishes a named event on the event bus. It is a non-blocking
// best-effort call — failures are logged but not returned to callers.
func (r *Runtime) publishEvent(ctx context.Context, topic string, payload map[string]interface{}) {
	if r.events == nil {
		return
	}
	_, err := r.events.Publish(ctx, topic, "runtime", payload)
	if err != nil {
		r.logger.Warn("failed to publish event",
			zap.String("topic", topic),
			zap.Error(err))
	}
}

// copyInstance creates a shallow copy of an AgentInstance for safe external
// consumption. The Manifest pointer is shared (immutable data). The caller
// must hold the instance's read lock.
func copyInstance(inst *AgentInstance) *AgentInstance {
	c := &AgentInstance{
		Manifest:            inst.Manifest,
		State:               inst.State,
		StartedAt:           inst.StartedAt,
		LastActive:          inst.LastActive,
		TokensUsed:          inst.TokensUsed,
		APICalls:            inst.APICalls,
		Heartbeat:           inst.Heartbeat,
		FailureCount:        inst.FailureCount,
		MissedHeartbeats:    inst.MissedHeartbeats,
		RecoveryAttempts:    inst.RecoveryAttempts,
		LastRecoveryAttempt: inst.LastRecoveryAttempt,
	}
	if inst.DegradedSince != nil {
		t := *inst.DegradedSince
		c.DegradedSince = &t
	}
	return c
}
