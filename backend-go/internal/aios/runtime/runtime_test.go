package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.NewNop()
}

func testBus(t *testing.T) *eventbus.Bus {
	t.Helper()
	return eventbus.New(testLogger(t))
}

func testManifest(id string) AgentManifest {
	return AgentManifest{
		ID:          id,
		Name:        "Test Agent " + id,
		Squad:       "test",
		Version:     "1.0.0",
		Description: "Agent for testing",
		ResourceLimits: ResourceLimits{
			MaxTokensPerMinute: 1000,
			MaxTokensPerHour:   10000,
			MaxAPICallsPerMin:  50,
			MaxAPICallsPerHour: 500,
			MaxToolChainDepth:  3,
		},
		MemoryConfig: MemoryConfig{
			ShortTermTTL:     15 * time.Minute,
			LongTermEnabled:  true,
			LongTermTTL:      30 * 24 * time.Hour,
		},
	}
}

// newTestRuntime creates a Runtime with a nop logger and event bus.
func newTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	return New(testLogger(t), testBus(t))
}

// mustRegister is a test helper that registers an agent and fails the test on error.
func mustRegister(t *testing.T, rt *Runtime, m AgentManifest) {
	t.Helper()
	if err := rt.RegisterAgent(m); err != nil {
		t.Fatalf("RegisterAgent(%q): %v", m.ID, err)
	}
}

// mustStart is a test helper that starts an agent and fails the test on error.
func mustStart(t *testing.T, rt *Runtime, id string) {
	t.Helper()
	if err := rt.StartAgent(id); err != nil {
		t.Fatalf("StartAgent(%q): %v", id, err)
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestLifecycleTransition verifies the full lifecycle:
// Register -> Start -> Ready -> [manually set Active] -> [manually set Idle] -> Stop -> Stopped
func TestLifecycleTransition(t *testing.T) {
	rt := newTestRuntime(t)

	// Register
	m := testManifest("lifecycle-test")
	mustRegister(t, rt, m)

	// Verify Init state
	inst, ok := rt.GetInstance("lifecycle-test")
	if !ok {
		t.Fatal("GetInstance returned false after RegisterAgent")
	}
	if inst.State != StateInit {
		t.Fatalf("expected StateInit, got %s", inst.State)
	}

	// Start -> Ready
	mustStart(t, rt, "lifecycle-test")
	inst, _ = rt.GetInstance("lifecycle-test")
	if inst.State != StateReady {
		t.Fatalf("expected StateReady after StartAgent, got %s", inst.State)
	}
	if inst.StartedAt.IsZero() {
		t.Fatal("StartedAt should be set after StartAgent")
	}
	if inst.Heartbeat.IsZero() {
		t.Fatal("Heartbeat should be set after StartAgent")
	}

	// Manually transition to Active (simulating trigger firing)
	instRaw, _ := rt.getInstance("lifecycle-test")
	instRaw.mu.Lock()
	instRaw.State = StateActive
	instRaw.LastActive = time.Now()
	instRaw.mu.Unlock()
	inst, _ = rt.GetInstance("lifecycle-test")
	if inst.State != StateActive {
		t.Fatalf("expected StateActive, got %s", inst.State)
	}

	// Manually transition to Idle (simulating decision completion)
	instRaw.mu.Lock()
	instRaw.State = StateIdle
	instRaw.mu.Unlock()
	inst, _ = rt.GetInstance("lifecycle-test")
	if inst.State != StateIdle {
		t.Fatalf("expected StateIdle, got %s", inst.State)
	}

	// Stop -> Stopped
	if err := rt.StopAgent("lifecycle-test"); err != nil {
		t.Fatalf("StopAgent: %v", err)
	}
	inst, _ = rt.GetInstance("lifecycle-test")
	if inst.State != StateStopped {
		t.Fatalf("expected StateStopped after StopAgent, got %s", inst.State)
	}
}

// TestDegradedOnResourceLimit verifies that exceeding resource limits
// transitions the agent to Degraded state.
func TestDegradedOnResourceLimit(t *testing.T) {
	rt := newTestRuntime(t)

	m := testManifest("resource-test")
	m.ResourceLimits = ResourceLimits{
		MaxTokensPerMinute: 0,   // zero = disabled
		MaxTokensPerHour:   100, // low threshold
		MaxAPICallsPerMin:  0,
		MaxAPICallsPerHour: 0,
	}

	mustRegister(t, rt, m)
	mustStart(t, rt, "resource-test")

	// Record usage below limit — should stay Ready.
	err := rt.RecordUsage("resource-test", 50, 0)
	if err != nil {
		t.Fatalf("RecordUsage (under limit): %v", err)
	}
	inst, _ := rt.GetInstance("resource-test")
	if inst.State != StateReady {
		t.Fatalf("expected StateReady (under limit), got %s", inst.State)
	}
	if inst.TokensUsed != 50 {
		t.Fatalf("expected TokensUsed=50, got %d", inst.TokensUsed)
	}

	// Record usage that exceeds the limit — should transition to Degraded.
	err = rt.RecordUsage("resource-test", 60, 0)
	if err != nil {
		t.Fatalf("RecordUsage (exceed limit): %v", err)
	}
	inst, _ = rt.GetInstance("resource-test")
	if inst.State != StateDegraded {
		t.Fatalf("expected StateDegraded (limit exceeded), got %s", inst.State)
	}
	if inst.DegradedSince == nil {
		t.Fatal("DegradedSince should be set after degradation")
	}
}

// TestHeartbeatTimeoutAndRecovery verifies that a stale heartbeat causes
// a transition to Crashed, and then auto-recovery restores the agent to Ready.
func TestHeartbeatTimeoutAndRecovery(t *testing.T) {
	rt := newTestRuntime(t)

	mut := testManifest("heartbeat-test")
	mustRegister(t, rt, mut)
	mustStart(t, rt, "heartbeat-test")

	// Heartbeat once to set a fresh timestamp.
	if err := rt.Heartbeat("heartbeat-test"); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	// Simulate a stale heartbeat by setting Heartbeat far in the past.
	instRaw, _ := rt.getInstance("heartbeat-test")
	instRaw.mu.Lock()
	instRaw.Heartbeat = time.Now().Add(-2 * HealthCheckInterval)
	instRaw.MissedHeartbeats = 0
	instRaw.mu.Unlock()

	// First HealthCheck: missed heartbeat -> MissedHeartbeats = 1
	rt.HealthCheck(context.Background())
	inst, _ := rt.GetInstance("heartbeat-test")
	if inst.State != StateReady {
		t.Fatalf("expected StateReady (missed=1), got %s (missed=%d)",
			inst.State, inst.MissedHeartbeats)
	}
	if inst.MissedHeartbeats != 1 {
		t.Fatalf("expected MissedHeartbeats=1, got %d", inst.MissedHeartbeats)
	}

	// Second HealthCheck: missed again -> MissedHeartbeats = 2
	instRaw.mu.Lock()
	instRaw.Heartbeat = time.Now().Add(-2 * HealthCheckInterval)
	instRaw.mu.Unlock()
	rt.HealthCheck(context.Background())
	inst, _ = rt.GetInstance("heartbeat-test")
	if inst.State != StateReady {
		t.Fatalf("expected StateReady (missed=2), got %s", inst.State)
	}
	if inst.MissedHeartbeats != 2 {
		t.Fatalf("expected MissedHeartbeats=2, got %d", inst.MissedHeartbeats)
	}

	// Third HealthCheck: missed again -> MissedHeartbeats = 3 = MaxMissedHeartbeats -> Crashed
	instRaw.mu.Lock()
	instRaw.Heartbeat = time.Now().Add(-2 * HealthCheckInterval)
	instRaw.mu.Unlock()
	rt.HealthCheck(context.Background())
	inst, _ = rt.GetInstance("heartbeat-test")
	if inst.State != StateCrashed {
		t.Fatalf("expected StateCrashed (missed>=%d), got %s (missed=%d)",
			MaxMissedHeartbeats, inst.State, inst.MissedHeartbeats)
	}
	if inst.FailureCount != 1 {
		t.Fatalf("expected FailureCount=1, got %d", inst.FailureCount)
	}

	// Recovery: set LastRecoveryAttempt far in the past so the interval is satisfied.
	instRaw.mu.Lock()
	instRaw.LastRecoveryAttempt = time.Now().Add(-2 * RecoveryInterval)
	instRaw.mu.Unlock()
	rt.HealthCheck(context.Background())
	inst, _ = rt.GetInstance("heartbeat-test")
	if inst.State != StateReady {
		t.Fatalf("expected StateReady after auto-recovery, got %s (attempts=%d)",
			inst.State, inst.RecoveryAttempts)
	}
	if inst.RecoveryAttempts != 1 {
		t.Fatalf("expected RecoveryAttempts=1, got %d", inst.RecoveryAttempts)
	}
}

// TestSuspendResume verifies SuspendAgent -> Suspended state -> StartAgent -> Ready.
func TestSuspendResume(t *testing.T) {
	rt := newTestRuntime(t)

	m := testManifest("suspend-test")
	mustRegister(t, rt, m)
	mustStart(t, rt, "suspend-test")

	// Suspend
	if err := rt.SuspendAgent("suspend-test"); err != nil {
		t.Fatalf("SuspendAgent: %v", err)
	}
	inst, _ := rt.GetInstance("suspend-test")
	if inst.State != StateSuspended {
		t.Fatalf("expected StateSuspended, got %s", inst.State)
	}

	// Suspend again should fail (already suspended).
	err := rt.SuspendAgent("suspend-test")
	if err == nil {
		t.Fatal("expected error on second SuspendAgent, got nil")
	}

	// Resume via StartAgent
	mustStart(t, rt, "suspend-test")
	inst, _ = rt.GetInstance("suspend-test")
	if inst.State != StateReady {
		t.Fatalf("expected StateReady after resume, got %s", inst.State)
	}
}

// TestSuspendFromIdle verifies that an Idle agent can also be suspended.
func TestSuspendFromIdle(t *testing.T) {
	rt := newTestRuntime(t)

	m := testManifest("suspend-idle-test")
	mustRegister(t, rt, m)
	mustStart(t, rt, "suspend-idle-test")

	// Set to Idle manually.
	instRaw, _ := rt.getInstance("suspend-idle-test")
	instRaw.mu.Lock()
	instRaw.State = StateIdle
	instRaw.mu.Unlock()

	// Suspend from Idle.
	if err := rt.SuspendAgent("suspend-idle-test"); err != nil {
		t.Fatalf("SuspendAgent from Idle: %v", err)
	}
	inst, _ := rt.GetInstance("suspend-idle-test")
	if inst.State != StateSuspended {
		t.Fatalf("expected StateSuspended, got %s", inst.State)
	}
}

// TestConcurrentSafety runs multiple goroutines performing operations on the
// same Runtime concurrently. It should not panic or produce data races.
func TestConcurrentSafety(t *testing.T) {
	rt := newTestRuntime(t)

	const numAgents = 5
	const numOps = 20

	// Register all agents sequentially.
	for i := 0; i < numAgents; i++ {
		id := "conc-agent-" + string(rune('0'+i))
		m := testManifest(id)
		// Give each agent its own limits.
		m.ResourceLimits.MaxTokensPerHour = 1000
		m.ResourceLimits.MaxAPICallsPerHour = 200
		mustRegister(t, rt, m)
	}

	var wg sync.WaitGroup

	// Start all agents concurrently.
	wg.Add(numAgents)
	for i := 0; i < numAgents; i++ {
		id := "conc-agent-" + string(rune('0'+i))
		go func(aID string) {
			defer wg.Done()
			if err := rt.StartAgent(aID); err != nil {
				t.Errorf("concurrent StartAgent(%q): %v", aID, err)
			}
		}(id)
	}
	wg.Wait()

	// Run a mix of operations concurrently.
	wg.Add(numOps)
	for i := 0; i < numOps; i++ {
		idx := i % numAgents
		id := "conc-agent-" + string(rune('0'+idx))
		op := i % 4
		go func(aID string, operation int) {
			defer wg.Done()
			switch operation {
			case 0:
				_ = rt.Heartbeat(aID)
			case 1:
				_, _ = rt.CheckLimits(aID)
			case 2:
				_, _ = rt.GetInstance(aID)
			case 3:
				_ = rt.RecordUsage(aID, 10, 1)
			}
		}(id, op)
	}
	wg.Wait()

	// HealthCheck concurrently with more operations.
	var innerWg sync.WaitGroup
	innerWg.Add(2)
	go func() {
		defer innerWg.Done()
		rt.HealthCheck(context.Background())
	}()
	go func() {
		defer innerWg.Done()
		for i := 0; i < 10; i++ {
			_ = rt.Heartbeat("conc-agent-"+string(rune('0'+i%numAgents)))
			_, _ = rt.GetInstance("conc-agent-" + string(rune('0'+i%numAgents)))
		}
	}()
	innerWg.Wait()

	// Stop all agents concurrently.
	wg.Add(numAgents)
	for i := 0; i < numAgents; i++ {
		id := "conc-agent-" + string(rune('0'+i))
		go func(aID string) {
			defer wg.Done()
			_ = rt.StopAgent(aID)
		}(id)
	}
	wg.Wait()

	// Final check: all should be Stopped.
	for i := 0; i < numAgents; i++ {
		id := "conc-agent-" + string(rune('0'+i))
		inst, ok := rt.GetInstance(id)
		if !ok {
			t.Errorf("agent %q should exist after StopAgent", id)
			continue
		}
		if inst.State != StateStopped {
			t.Errorf("agent %q expected StateStopped, got %s", id, inst.State)
		}
	}
}

// TestListInstances checks that ListInstances correctly filters by state.
func TestListInstances(t *testing.T) {
	rt := newTestRuntime(t)

	// Register 3 agents.
	for i := 0; i < 3; i++ {
		id := "list-" + string(rune('0'+i))
		mustRegister(t, rt, testManifest(id))
	}

	// All in Init.
	instances := rt.ListInstances()
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances, got %d", len(instances))
	}

	initInstances := rt.ListInstances(StateInit)
	if len(initInstances) != 3 {
		t.Fatalf("expected 3 StateInit instances, got %d", len(initInstances))
	}

	// Start two agents.
	mustStart(t, rt, "list-0")
	mustStart(t, rt, "list-1")

	readyInstances := rt.ListInstances(StateReady)
	if len(readyInstances) != 2 {
		t.Fatalf("expected 2 StateReady instances, got %d", len(readyInstances))
	}

	initAfterStart := rt.ListInstances(StateInit)
	if len(initAfterStart) != 1 {
		t.Fatalf("expected 1 StateInit instance after start, got %d", len(initAfterStart))
	}

	// Filter by multiple states.
	multi := rt.ListInstances(StateInit, StateReady)
	if len(multi) != 3 {
		t.Fatalf("expected 3 instances filtered by Init+Ready, got %d", len(multi))
	}
}

// TestDoubleRegister checks that registering the same agent ID twice returns an error.
func TestDoubleRegister(t *testing.T) {
	rt := newTestRuntime(t)

	m := testManifest("double-reg")
	mustRegister(t, rt, m)

	err := rt.RegisterAgent(m)
	if err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
}

// TestStartInvalidStates verifies that StartAgent rejects invalid starting states.
func TestStartInvalidStates(t *testing.T) {
	rt := newTestRuntime(t)

	m := testManifest("start-invalid")
	mustRegister(t, rt, m)

	// Start normally.
	mustStart(t, rt, "start-invalid")

	// Transition directly to Stopped and try to start again.
	instRaw, _ := rt.getInstance("start-invalid")
	instRaw.mu.Lock()
	instRaw.State = StateStopped
	instRaw.mu.Unlock()

	err := rt.StartAgent("start-invalid")
	if err == nil {
		t.Fatal("expected error starting a Stopped agent, got nil")
	}
}

// TestHeartbeatDegraded validates that heartbeats work for Degraded agents.
func TestHeartbeatDegraded(t *testing.T) {
	rt := newTestRuntime(t)

	m := testManifest("hb-degraded")
	m.ResourceLimits.MaxTokensPerHour = 10
	mustRegister(t, rt, m)
	mustStart(t, rt, "hb-degraded")

	// Exceed limit -> Degraded.
	_ = rt.RecordUsage("hb-degraded", 20, 0)
	inst, _ := rt.GetInstance("hb-degraded")
	if inst.State != StateDegraded {
		t.Fatalf("expected StateDegraded, got %s", inst.State)
	}

	// Heartbeat should still succeed in Degraded state.
	if err := rt.Heartbeat("hb-degraded"); err != nil {
		t.Fatalf("Heartbeat in Degraded state: %v", err)
	}
}

// TestStopIdempotent verifies that calling StopAgent twice is safe.
func TestStopIdempotent(t *testing.T) {
	rt := newTestRuntime(t)

	m := testManifest("stop-idempotent")
	mustRegister(t, rt, m)
	mustStart(t, rt, "stop-idempotent")

	if err := rt.StopAgent("stop-idempotent"); err != nil {
		t.Fatalf("first StopAgent: %v", err)
	}

	if err := rt.StopAgent("stop-idempotent"); err != nil {
		t.Fatalf("second StopAgent (should be no-op): %v", err)
	}
}

// TestCrashedMaxRecoveryExhausted verifies that after exhausting recovery
// attempts, a crashed agent stays crashed.
func TestCrashedMaxRecoveryExhausted(t *testing.T) {
	rt := newTestRuntime(t)

	m := testManifest("exhaust-recovery")
	mustRegister(t, rt, m)
	mustStart(t, rt, "exhaust-recovery")
	_ = rt.Heartbeat("exhaust-recovery")

	instRaw, _ := rt.getInstance("exhaust-recovery")
	instRaw.mu.Lock()
	instRaw.Heartbeat = time.Now().Add(-2 * HealthCheckInterval)
	instRaw.MissedHeartbeats = 0
	instRaw.mu.Unlock()

	// Push to Crashed via 3 missed heartbeats.
	for i := 0; i < MaxMissedHeartbeats; i++ {
		instRaw.mu.Lock()
		instRaw.Heartbeat = time.Now().Add(-2 * HealthCheckInterval)
		instRaw.mu.Unlock()
		rt.HealthCheck(context.Background())
	}

	inst, _ := rt.GetInstance("exhaust-recovery")
	if inst.State != StateCrashed {
		t.Fatalf("expected StateCrashed, got %s", inst.State)
	}

	// Set recovery to full attempts and try to recover.
	instRaw.mu.Lock()
	instRaw.RecoveryAttempts = MaxRecoveryAttempts
	instRaw.LastRecoveryAttempt = time.Now().Add(-2 * RecoveryInterval)
	instRaw.mu.Unlock()

	// HealthCheck should NOT recover (max attempts exhausted).
	rt.HealthCheck(context.Background())
	inst, _ = rt.GetInstance("exhaust-recovery")
	if inst.State != StateCrashed {
		t.Fatalf("expected StateCrashed (recovery exhausted), got %s", inst.State)
	}
}

// TestGetInstanceNotFound checks that GetInstance returns false for unknown IDs.
func TestGetInstanceNotFound(t *testing.T) {
	rt := newTestRuntime(t)

	_, ok := rt.GetInstance("nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent agent")
	}
}

// TestListInstancesEmpty checks that a fresh Runtime has no instances.
func TestListInstancesEmpty(t *testing.T) {
	rt := newTestRuntime(t)

	instances := rt.ListInstances()
	if len(instances) != 0 {
		t.Fatalf("expected empty list, got %d", len(instances))
	}
}
