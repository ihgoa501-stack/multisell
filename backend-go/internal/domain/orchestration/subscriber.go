package orchestration

import (
	"context"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

// RegisterSubscribers registers all orchestration event handlers.
func RegisterSubscribers(bus *eventbus.Bus, orch *PipelineOrchestrator) {
	// sourcing.complete — advance to enrichment.
	bus.Subscribe("agent.decided.A8.sourcing_scan", func(ctx context.Context, evt eventbus.Event) error {
		productID := extractProductID(evt)
		if productID == 0 {
			return nil
		}
		return orch.AdvancePipeline(ctx, productID, "sourcing", true)
	})

	// enrichment.complete — advance to compliance.
	bus.Subscribe("product.content.ai_generated", func(ctx context.Context, evt eventbus.Event) error {
		productID := extractProductID(evt)
		if productID == 0 {
			return nil
		}
		return orch.AdvancePipeline(ctx, productID, "enrichment", true)
	})

	// compliance.complete — advance to pricing.
	bus.Subscribe("compliance.product_status_changed", func(ctx context.Context, evt eventbus.Event) error {
		productID := extractProductID(evt)
		if productID == 0 {
			return nil
		}
		return orch.AdvancePipeline(ctx, productID, "compliance", true)
	})

	// pricing.complete — advance to listing.
	bus.Subscribe("price.updated", func(ctx context.Context, evt eventbus.Event) error {
		productID := extractProductID(evt)
		if productID == 0 {
			return nil
		}
		return orch.AdvancePipeline(ctx, productID, "pricing", true)
	})

	// listing.complete — advance to monitoring.
	bus.Subscribe("listing.live", func(ctx context.Context, evt eventbus.Event) error {
		productID := extractProductID(evt)
		if productID == 0 {
			return nil
		}
		return orch.AdvancePipeline(ctx, productID, "listing", true)
	})
}

// extractProductID extracts product_id from an event payload.
func extractProductID(evt eventbus.Event) int64 {
	if pid, ok := evt.Payload["product_id"].(float64); ok {
		return int64(pid)
	}
	return 0
}
