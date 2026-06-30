// Package supplychain implements the supply chain orchestration layer that
// bridges sourcing recommendations (A8) with logistics quoting (A10) and
// downstream fulfillment events.
//
// The escalation subsystem adds a 4-level error handling mechanism to the
// orchestrator state machine:
//
//   - Level 0: automatic retry with exponential backoff (1s -> 3s -> 9s).
//     After exhausting max attempts the event is promoted to Level 1.
//   - Level 1: skip the unreliable step and auto-switch to an alternative
//     carrier channel.
//   - Level 2: flag the flow for manual review and notify the seller via
//     a WebSocket dashboard notification.
//   - Level 3: global alert — reports the issue to Sentry and broadcasts
//     a critical dashboard alert to all WebSocket clients.
package supplychain

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
)

// EscalationLevel represents the severity of an issue within the supply chain
// orchestrator state machine. Higher numeric values indicate more severe
// conditions.
type EscalationLevel int

const (
	// EscalationLevel0: automatic retry with exponential backoff (1s -> 3s -> 9s).
	// When the max attempt count is exhausted the event is promoted to Level 1.
	EscalationLevel0 EscalationLevel = iota
	// EscalationLevel1: skip the unreliable step and auto-switch to an
	// alternative carrier channel.
	EscalationLevel1
	// EscalationLevel2: flag the flow for manual review and notify the seller
	// (or relevant user) via WebSocket dashboard notification.
	EscalationLevel2
	// EscalationLevel3: global alert — reports the issue to Sentry and
	// broadcasts a critical dashboard alert to all WebSocket clients.
	EscalationLevel3
)

// String returns the human-readable name for the escalation level.
func (l EscalationLevel) String() string {
	switch l {
	case EscalationLevel0:
		return "auto_retry"
	case EscalationLevel1:
		return "skip_and_switch"
	case EscalationLevel2:
		return "manual_review"
	case EscalationLevel3:
		return "global_alert"
	default:
		return fmt.Sprintf("unknown(%d)", l)
	}
}

// EscalationConfig holds tuning parameters for the escalation manager.
type EscalationConfig struct {
	// RetryMaxAttempts is the maximum number of retry attempts for Level 0.
	RetryMaxAttempts int
	// RetryBaseDelay is the initial delay for exponential backoff.
	RetryBaseDelay time.Duration
	// BackoffMultiplier is multiplied by the current delay each attempt.
	BackoffMultiplier float64
}

// DefaultEscalationConfig returns an EscalationConfig with standard values.
func DefaultEscalationConfig() EscalationConfig {
	return EscalationConfig{
		RetryMaxAttempts:  3,
		RetryBaseDelay:    time.Second,
		BackoffMultiplier: 3.0,
	}
}

// EscalationEvent carries the full context of an escalation incident.
type EscalationEvent struct {
	FlowID      string                 `json:"flow_id"`
	Level       EscalationLevel        `json:"level"`
	Error       string                 `json:"error"`
	SourceType  string                 `json:"source_type"`
	SourceID    string                 `json:"source_id"`
	Attempt     int                    `json:"attempt,omitempty"`
	ChannelName string                 `json:"channel_name,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// EscalationManager implements the 4-level escalation mechanism for the
// supply chain orchestrator state machine.
type EscalationManager struct {
	config EscalationConfig
	logger *zap.Logger
	hub    *realtime.Hub
}

// NewEscalationManager creates a new EscalationManager with the default
// configuration, overridable via options.
func NewEscalationManager(logger *zap.Logger, hub *realtime.Hub, opts ...EscalationOption) *EscalationManager {
	m := &EscalationManager{
		config: DefaultEscalationConfig(),
		logger: logger,
		hub:    hub,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// EscalationOption configures an EscalationManager.
type EscalationOption func(*EscalationManager)

// WithEscalationConfig replaces the default escalation config.
func WithEscalationConfig(cfg EscalationConfig) EscalationOption {
	return func(m *EscalationManager) { m.config = cfg }
}

// Handle dispatches an escalation event to the appropriate level handler.
// It returns the level handler's error, or the context error if the operation
// was cancelled. For Level 0, a nil return signals the caller that the retry
// delay has elapsed and the operation should be re-attempted.
func (em *EscalationManager) Handle(ctx context.Context, evt EscalationEvent) error {
	em.logger.Warn("escalation triggered",
		zap.Int("level", int(evt.Level)),
		zap.String("level_name", evt.Level.String()),
		zap.String("flow_id", evt.FlowID),
		zap.String("error", evt.Error),
	)

	switch evt.Level {
	case EscalationLevel0:
		return em.handleLevel0(ctx, evt)
	case EscalationLevel1:
		return em.handleLevel1(ctx, evt)
	case EscalationLevel2:
		return em.handleLevel2(ctx, evt)
	case EscalationLevel3:
		return em.handleLevel3(ctx, evt)
	default:
		return fmt.Errorf("escalation: unknown level %d", evt.Level)
	}
}

// handleLevel0 implements automatic retry with exponential backoff.
//
// If the attempt count has reached RetryMaxAttempts, the event is escalated
// to Level 1 (skip and auto-switch). Otherwise the method sleeps for the
// computed backoff delay and returns nil, signalling the caller to retry.
func (em *EscalationManager) handleLevel0(ctx context.Context, evt EscalationEvent) error {
	if evt.Attempt >= em.config.RetryMaxAttempts {
		nextEvt := evt
		nextEvt.Level = EscalationLevel1
		nextEvt.Error = fmt.Sprintf("max retries (%d) exhausted: %s", em.config.RetryMaxAttempts, evt.Error)
		em.logger.Warn("level0: max retries exhausted, escalating to level 1",
			zap.String("flow_id", evt.FlowID),
			zap.Int("attempts", evt.Attempt),
		)
		return em.Handle(ctx, nextEvt)
	}

	delay := em.backoffDelay(evt.Attempt)
	em.logger.Info("level0: scheduling retry",
		zap.Int("attempt", evt.Attempt+1),
		zap.Int("max", em.config.RetryMaxAttempts),
		zap.String("flow_id", evt.FlowID),
		zap.Duration("delay", delay),
	)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// backoffDelay computes the backoff duration for the given attempt index.
// For attempt 0 with default config: 1s; attempt 1: 3s; attempt 2: 9s.
func (em *EscalationManager) backoffDelay(attempt int) time.Duration {
	d := float64(em.config.RetryBaseDelay) * math.Pow(em.config.BackoffMultiplier, float64(attempt))
	return time.Duration(d)
}

// handleLevel1 skips the failed step and signals the caller to auto-switch
// to an alternative carrier channel. Returns nil unconditionally; the caller
// is responsible for executing the channel switch.
func (em *EscalationManager) handleLevel1(_ context.Context, evt EscalationEvent) error {
	em.logger.Info("level1: skip and auto-switch channel",
		zap.String("flow_id", evt.FlowID),
		zap.String("failed_channel", evt.ChannelName),
	)
	return nil
}

// handleLevel2 flags the flow for manual review and broadcasts a WebSocket
// notification so sellers can act on profit-threshold-edge cases.
func (em *EscalationManager) handleLevel2(ctx context.Context, evt EscalationEvent) error {
	em.logger.Warn("level2: flagging for manual review",
		zap.String("flow_id", evt.FlowID),
		zap.String("error", evt.Error),
	)

	if em.hub != nil {
		msg, err := json.Marshal(map[string]interface{}{
			"type":    "escalation",
			"level":   evt.Level.String(),
			"flow_id": evt.FlowID,
			"error":   evt.Error,
			"source_type": evt.SourceType,
			"source_id":   evt.SourceID,
			"timestamp":   evt.Timestamp,
		})
		if err != nil {
			return fmt.Errorf("level2: marshal notification: %w", err)
		}
		em.hub.Broadcast(msg)
	}

	return nil
}

// handleLevel3 reports the incident to Sentry and broadcasts a critical
// dashboard alert to all connected WebSocket clients.
func (em *EscalationManager) handleLevel3(ctx context.Context, evt EscalationEvent) error {
	em.logger.Error("level3: global alert",
		zap.String("flow_id", evt.FlowID),
		zap.String("error", evt.Error),
	)

	// Report to Sentry.
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("escalation_level", evt.Level.String())
		scope.SetTag("flow_id", evt.FlowID)
		scope.SetExtra("source_type", evt.SourceType)
		scope.SetExtra("source_id", evt.SourceID)
		if len(evt.Metadata) > 0 {
			scope.SetContext("metadata", evt.Metadata)
		}
		sentry.CaptureException(fmt.Errorf("level3 escalation: %s", evt.Error))
	})

	// Broadcast dashboard alert via WebSocket.
	if em.hub != nil {
		msg, err := json.Marshal(map[string]interface{}{
			"type":      "global_alert",
			"level":     evt.Level.String(),
			"flow_id":   evt.FlowID,
			"error":     evt.Error,
			"severity":  "critical",
			"timestamp": evt.Timestamp,
		})
		if err != nil {
			return fmt.Errorf("level3: marshal alert: %w", err)
		}
		em.hub.Broadcast(msg)
	}

	return nil
}
