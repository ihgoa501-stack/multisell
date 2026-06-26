package sourcing

import (
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AgentEventPublisher wraps an eventbus.Bus to implement EventPublisher.
// This avoids a direct import of the toolbridge package in the sourcing service.
type AgentEventPublisher struct {
	bus *eventbus.Bus
}

// NewAgentEventPublisher creates an EventPublisher backed by the event bus.
func NewAgentEventPublisher(bus *eventbus.Bus) *AgentEventPublisher {
	return &AgentEventPublisher{bus: bus}
}

// Publish sends an event through the event bus.
func (p *AgentEventPublisher) Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error) {
	return p.bus.Publish(ctx, topic, source, payload)
}

// ==========================================================
// A8 Agent event handler functions
// ==========================================================

// HandleSourcingRecommend handles "sourcing.recommend" events.
// Called when a new sourcing recommendation has been generated.
// It processes the recommendation and triggers the listing_optimize pipeline
// for high-scoring products.
func HandleSourcingRecommend(db *gorm.DB, logger *zap.Logger) eventbus.Handler {
	return func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		logger.Info("A8 sourcing.recommend event received",
			zap.Any("payload", payload),
		)

		// Extract score and decide whether to proceed to listing_optimize.
		score, _ := payload["score"].(int)
		// Also check float64 (JSON unmarshal from scheduler/outbox).
		if scoreFloat, ok := payload["score"].(float64); ok {
			score = int(scoreFloat)
		}

		if score >= 7 {
			logger.Info("A8: high-score recommendation, listing_optimize pipeline ready",
				zap.Int("score", score),
				zap.String("source_url", toString(payload["source_url"])),
			)
		} else {
			logger.Debug("A8: recommendation below threshold, no pipeline trigger",
				zap.Int("score", score),
			)
		}

		return nil
	}
}

// HandleSourcingTick handles "scheduler.tick.A8" events.
// Placeholder for periodic batch scanning — will be implemented in Phase 2.
func HandleSourcingTick(db *gorm.DB, logger *zap.Logger) eventbus.Handler {
	return func(ctx context.Context, evt eventbus.Event) error {
		logger.Debug("A8 scheduler tick received — batch scanning not yet implemented")
		return nil
	}
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
