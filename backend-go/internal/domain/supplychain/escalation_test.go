package supplychain

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/realtime"
)

func TestEscalationLevel_String(t *testing.T) {
	tests := []struct {
		level EscalationLevel
		want  string
	}{
		{EscalationLevel0, "auto_retry"},
		{EscalationLevel1, "skip_and_switch"},
		{EscalationLevel2, "manual_review"},
		{EscalationLevel3, "global_alert"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("EscalationLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
		})
	}

	// Unknown level.
	if s := EscalationLevel(99).String(); s == "" {
		t.Error("unknown level should not return empty string")
	}
}

func TestDefaultEscalationConfig(t *testing.T) {
	cfg := DefaultEscalationConfig()
	if cfg.RetryMaxAttempts != 3 {
		t.Errorf("RetryMaxAttempts = %d, want 3", cfg.RetryMaxAttempts)
	}
	if cfg.RetryBaseDelay != time.Second {
		t.Errorf("RetryBaseDelay = %v, want 1s", cfg.RetryBaseDelay)
	}
	if cfg.BackoffMultiplier != 3.0 {
		t.Errorf("BackoffMultiplier = %f, want 3.0", cfg.BackoffMultiplier)
	}
}

func TestBackoffDelay(t *testing.T) {
	logger := dbtest.NewLogger(t)
	em := NewEscalationManager(logger, nil)

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, time.Second},
		{1, 3 * time.Second},
		{2, 9 * time.Second},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := em.backoffDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("backoffDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestNewEscalationManager_WithConfigOption(t *testing.T) {
	logger := dbtest.NewLogger(t)
	customCfg := EscalationConfig{
		RetryMaxAttempts:  5,
		RetryBaseDelay:    500 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	em := NewEscalationManager(logger, nil, WithEscalationConfig(customCfg))

	if em.config.RetryMaxAttempts != 5 {
		t.Errorf("RetryMaxAttempts = %d, want 5", em.config.RetryMaxAttempts)
	}
	if em.config.RetryBaseDelay != 500*time.Millisecond {
		t.Errorf("RetryBaseDelay = %v, want 500ms", em.config.RetryBaseDelay)
	}
	if em.config.BackoffMultiplier != 2.0 {
		t.Errorf("BackoffMultiplier = %f, want 2.0", em.config.BackoffMultiplier)
	}
}

func TestEscalationLevel0_MaxAttemptsExhausted(t *testing.T) {
	logger := dbtest.NewLogger(t)
	em := NewEscalationManager(logger, nil)

	evt := EscalationEvent{
		FlowID:  "flow-1",
		Level:   EscalationLevel0,
		Error:   "connection timeout",
		Attempt: 3, // >= max (3), so escalation to Level 1 should happen
	}

	// When attempts are exhausted, handleLevel0 escalates to Level 1,
	// which returns nil without requiring a WebSocket hub.
	ctx := context.Background()
	if err := em.Handle(ctx, evt); err != nil {
		t.Errorf("Handle(level 0 exhausted) should return nil (escalated to L1), got: %v", err)
	}
}

func TestEscalationLevel0_CancelledContext(t *testing.T) {
	logger := dbtest.NewLogger(t)
	em := NewEscalationManager(logger, nil)

	evt := EscalationEvent{
		FlowID:  "flow-2",
		Level:   EscalationLevel0,
		Error:   "temporary failure",
		Attempt: 0, // will start a timer
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	if err := em.Handle(ctx, evt); err != context.Canceled {
		t.Errorf("Handle with cancelled context should return context.Canceled, got: %v", err)
	}
}

func TestEscalationLevel1_ReturnsNil(t *testing.T) {
	logger := dbtest.NewLogger(t)
	em := NewEscalationManager(logger, nil)

	evt := EscalationEvent{
		FlowID:      "flow-3",
		Level:       EscalationLevel1,
		Error:       "carrier unavailable",
		ChannelName: "CNAIR",
	}

	ctx := context.Background()
	if err := em.Handle(ctx, evt); err != nil {
		t.Errorf("Handle(level 1) should return nil, got: %v", err)
	}
}

func TestEscalationLevel2_NoHub(t *testing.T) {
	logger := dbtest.NewLogger(t)
	em := NewEscalationManager(logger, nil)

	evt := EscalationEvent{
		FlowID:     "flow-4",
		Level:      EscalationLevel2,
		Error:      "profit margin below threshold",
		SourceType: "order",
		SourceID:   "ORD-001",
		Timestamp:  time.Now(),
	}

	ctx := context.Background()
	if err := em.Handle(ctx, evt); err != nil {
		t.Errorf("Handle(level 2, no hub) should return nil, got: %v", err)
	}
}

func TestEscalationLevel2_WithHub(t *testing.T) {
	logger := dbtest.NewLogger(t)
	hub := realtime.NewHub(logger)
	go hub.Run()

	em := NewEscalationManager(logger, hub)

	evt := EscalationEvent{
		FlowID:     "flow-5",
		Level:      EscalationLevel2,
		Error:      "profit margin at 3.2%",
		SourceType: "order",
		SourceID:   "ORD-002",
		Timestamp:  time.Now(),
	}

	ctx := context.Background()
	if err := em.Handle(ctx, evt); err != nil {
		t.Errorf("Handle(level 2, with hub) should return nil, got: %v", err)
	}
}

func TestEscalationLevel3_NoHub(t *testing.T) {
	logger := dbtest.NewLogger(t)
	em := NewEscalationManager(logger, nil)

	evt := EscalationEvent{
		FlowID:     "flow-6",
		Level:      EscalationLevel3,
		Error:      "inventory corruption detected",
		SourceType: "inventory",
		SourceID:   "SKU-999",
		Timestamp:  time.Now(),
	}

	ctx := context.Background()
	if err := em.Handle(ctx, evt); err != nil {
		t.Errorf("Handle(level 3, no hub) should return nil, got: %v", err)
	}
}

func TestEscalationLevel3_WithHub(t *testing.T) {
	logger := dbtest.NewLogger(t)
	hub := realtime.NewHub(logger)
	go hub.Run()

	em := NewEscalationManager(logger, hub)

	evt := EscalationEvent{
		FlowID:     "flow-7",
		Level:      EscalationLevel3,
		Error:      "system-wide carrier API failure",
		SourceType: "carrier",
		SourceID:   "SYS",
		Metadata: map[string]interface{}{
			"affected_flows": 12,
			"carrier":        "CNPOST",
		},
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	if err := em.Handle(ctx, evt); err != nil {
		t.Errorf("Handle(level 3, with hub) should return nil, got: %v", err)
	}
}

func TestEscalation_UnknownLevel(t *testing.T) {
	logger := dbtest.NewLogger(t)
	em := NewEscalationManager(logger, nil)

	evt := EscalationEvent{
		FlowID: "flow-8",
		Level:  EscalationLevel(99),
		Error:  "bogus",
	}

	ctx := context.Background()
	if err := em.Handle(ctx, evt); err == nil {
		t.Error("Handle(unknown level) should return an error")
	}
}

func TestEscalation_EscalateFromL0toL1(t *testing.T) {
	logger := dbtest.NewLogger(t)
	em := NewEscalationManager(logger, nil)

	// Attempt exactly at max (3) should escalate to Level 1 and return nil.
	evt := EscalationEvent{
		FlowID:  "flow-9",
		Level:   EscalationLevel0,
		Error:   "rate limit exceeded",
		Attempt: 3,
	}

	ctx := context.Background()
	if err := em.Handle(ctx, evt); err != nil {
		t.Errorf("escalation to L1 after max attempts should succeed, got: %v", err)
	}
}
