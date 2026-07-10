package ai

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/costcontrol"
	"github.com/lingmirror/backend-go/internal/aios/guardrails"
)

// ----- mocks -----

type mockBus struct {
	mu     sync.Mutex
	events []busEvent
}

type busEvent struct {
	Topic   string
	Source  string
	Payload map[string]interface{}
}

func (m *mockBus) Publish(_ context.Context, topic, source string, payload map[string]interface{}) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, busEvent{Topic: topic, Source: source, Payload: payload})
	return "evt_" + topic, nil
}

func (m *mockBus) Events() []busEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]busEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

// passGuard is a no-op guardrail that always passes.
type passGuard struct{}

func (g *passGuard) Name() string { return "test_pass_guard" }

func (g *passGuard) Check(_ context.Context, _ *guardrails.GuardInput) (*guardrails.GuardResult, error) {
	return &guardrails.GuardResult{Pass: true, Blocked: false, Reason: "test pass", Risk: "low"}, nil
}

// blockGuard blocks any input whose RawInput contains "blockme".
type blockGuard struct{}

func (g *blockGuard) Name() string { return "test_block_guard" }

func (g *blockGuard) Check(_ context.Context, inp *guardrails.GuardInput) (*guardrails.GuardResult, error) {
	if strings.Contains(inp.RawInput, "blockme") {
		return &guardrails.GuardResult{Pass: false, Blocked: true, Reason: "test block", Risk: "high"}, nil
	}
	return &guardrails.GuardResult{Pass: true, Blocked: false, Reason: "test pass", Risk: "low"}, nil
}

var a5FullContext = map[string]interface{}{
	"sku_code":          "TEST-SKU-001",
	"sellable_stock":    float64(100),
	"sales_7d":          float64(35),
	"lead_time_days":    float64(30),
	"safety_stock_days": float64(16),
}

// ----- clampConfidence -----

func TestClampConfidence(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{-0.5, 0},
		{0.0, 0},
		{0.5, 0.5},
		{0.99, 0.99},
		{1.0, 0.99},
		{1.5, 0.99},
	}
	for _, tc := range tests {
		got := clampConfidence(tc.input)
		if got != tc.want {
			t.Errorf("clampConfidence(%v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ----- RunWithContext timeout -----

func TestOrchestrator_RunWithContext_Timeout(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	// A cancelled context should immediately return ctx.Err().
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := orch.RunWithContext(ctx, &RunAgentRequest{
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		Context:       a5FullContext,
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected 'context canceled', got: %v", err)
	}
}

// ----- decision cache -----

func TestOrchestrator_DecisionCache(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	req := &RunAgentRequest{
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		Context:       a5FullContext,
	}

	// First call populates cache.
	r1, err := orch.Run(req)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Verify cache has the entry.
	key := orch.decisionCache.cacheKey(req.AgentID, req.DecisionPoint, req.Context)
	orch.decisionCache.mu.RLock()
	entry, exists := orch.decisionCache.entries[key]
	orch.decisionCache.mu.RUnlock()
	if !exists {
		t.Fatal("expected cache entry after first Run")
	}
	if entry.result.TraceID != r1.TraceID {
		t.Fatalf("cached traceID = %s, want %s", entry.result.TraceID, r1.TraceID)
	}

	// Second call with same params — no cache-hit short-circuit yet, but
	// cache remains populated and output is deterministic.
	r2, err := orch.Run(req)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if r2.TraceID == r1.TraceID {
		t.Fatal("expected different traceID for second call (cache is write-only)")
	}
	// Deterministic output should match between runs.
	if r1.Output["risk_reason"] != r2.Output["risk_reason"] {
		t.Fatalf("output mismatch: %q vs %q", r1.Output["risk_reason"], r2.Output["risk_reason"])
	}
}

// ----- guardrails integration -----

func TestOrchestrator_GuardrailsIntegration(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	chain := guardrails.NewChain()
	chain.Add(&passGuard{})
	orch.WithGuardrails(chain)

	result, err := orch.Run(&RunAgentRequest{
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		Context:       a5FullContext,
	})
	if err != nil {
		t.Fatalf("Run with guardrails: %v", err)
	}
	if result.TraceID == "" {
		t.Fatal("empty traceID")
	}
	if result.Output == nil {
		t.Fatal("nil output")
	}
	if result.Output["risk_reason"] == nil {
		t.Fatal("missing risk_reason in output")
	}
}

// ----- Chat guardrails -----

func TestOrchestrator_ChatGuardrails_BlocksSuspiciousInput(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	chain := guardrails.NewChain()
	chain.Add(&blockGuard{})
	orch.WithGuardrails(chain)

	uid := int64(1)
	_, err := orch.Chat("this should blockme", &uid)
	if err == nil {
		t.Fatal("expected error from blocked input")
	}
	if !errors.Is(err, ErrBlockedByGuardrails) {
		t.Fatalf("expected ErrBlockedByGuardrails, got: %v", err)
	}
}

func TestOrchestrator_ChatGuardrails_PassesCleanInput(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	chain := guardrails.NewChain()
	chain.Add(&passGuard{})
	orch.WithGuardrails(chain)

	uid := int64(1)
	result, err := orch.Chat("what are my sales", &uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TraceID == "" {
		t.Fatal("expected non-empty traceID")
	}
}

func TestOrchestrator_ChatGuardrails_NilChainSkipsCheck(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())
	// No guardrails set — chain is nil.
	uid := int64(1)
	result, err := orch.Chat("any message works", &uid)
	if err != nil {
		t.Fatalf("unexpected error with nil guardrails: %v", err)
	}
	if result.TraceID == "" {
		t.Fatal("expected non-empty traceID")
	}
}

func TestOrchestrator_ChatGuardrails_NilUserID(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	chain := guardrails.NewChain()
	chain.Add(&passGuard{})
	orch.WithGuardrails(chain)

	result, err := orch.Chat("what are my sales", nil)
	if err != nil {
		t.Fatalf("unexpected error with nil userID: %v", err)
	}
	if result.TraceID == "" {
		t.Fatal("expected non-empty traceID")
	}
}

// ----- budget control -----

func TestOrchestrator_BudgetControl(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	// High daily cap — allows everything.
	ctrl := costcontrol.NewController(db, testLogger(), 100.0, 5*time.Minute, 2.0)
	orch.WithBudget(ctrl)

	result, err := orch.Run(&RunAgentRequest{
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		Context:       a5FullContext,
	})
	if err != nil {
		t.Fatalf("Run with budget: %v", err)
	}
	if result.Output["risk_reason"] == nil {
		t.Fatal("missing risk_reason in output")
	}
}

// ----- event bus publishing -----

func TestOrchestrator_EventBusPublish(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	bus := &mockBus{}
	orch.WithBus(bus)

	_, err := orch.Run(&RunAgentRequest{
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		Context:       a5FullContext,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// publishDecisionEvent runs in a goroutine; give it time to execute.
	time.Sleep(50 * time.Millisecond)

	events := bus.Events()
	if len(events) == 0 {
		t.Fatal("expected at least one event on bus")
	}

	wantTopic := "agent.decided.A5.stock_alert"
	gotTopic := events[0].Topic
	if gotTopic != wantTopic {
		t.Fatalf("topic = %q, want %q", gotTopic, wantTopic)
	}
	if events[0].Source != "orchestrator" {
		t.Fatalf("source = %q, want 'orchestrator'", events[0].Source)
	}
	payload := events[0].Payload
	if payload["agent_id"] != "A5" {
		t.Fatalf("payload agent_id = %v, want 'A5'", payload["agent_id"])
	}
	if payload["decision_point"] != "stock_alert" {
		t.Fatalf("payload decision_point = %v, want 'stock_alert'", payload["decision_point"])
	}
	if _, ok := payload["trace_id"]; !ok {
		t.Fatal("payload missing trace_id")
	}
}

// ----- advisory agent (G1) does not create an action -----

func TestOrchestrator_RunAdvisory_NoAction(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	result, err := orch.Run(&RunAgentRequest{
		AgentID:       "G1",
		DecisionPoint: "dashboard_overview",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Action != nil {
		t.Fatalf("expected no action for advisory agent, got action id=%d status=%s", result.Action.ID, result.Action.Status)
	}
}
