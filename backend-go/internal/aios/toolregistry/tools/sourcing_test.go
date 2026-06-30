package tools

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/domain/logistics"
	"github.com/lingmirror/backend-go/internal/domain/sourcing"
)

// invokeSourcingRecommend finds the "sourcing.recommend" tool from
// SourcingTools() and invokes its handler directly with the given input.
func invokeSourcingRecommend(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	for _, tool := range SourcingTools() {
		if tool.Name == "sourcing.recommend" {
			return tool.Handler(ctx, input)
		}
	}
	return nil, nil
}

// ── SetSourcingEngine + Handler ─────────────────────────────────────────

func TestSourcingRecommend_NoEngine_UsesStaticFallback(t *testing.T) {
	// Ensure a clean engine state. Save & restore so other tests are unaffected.
	prev := sourcingEngine
	defer func() { sourcingEngine = prev }()
	sourcingEngine = nil

	resp, err := invokeSourcingRecommend(context.Background(), map[string]interface{}{
		"source_url":   "https://detail.1688.com/offer/example.html",
		"price_1688":   100,
		"weight_kg":    2,
		"destination":  "JP",
		"markup_pct":   250,
	})
	if err != nil {
		t.Fatalf("sourcing.recommend handler error: %v", err)
	}

	m, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map response, got %T", resp)
	}
	pb, ok := m["profit_breakdown"].(*sourcing.ProfitBreakdown)
	if !ok {
		t.Fatalf("expected *ProfitBreakdown, got %T", m["profit_breakdown"])
	}
	// Without an engine, JP falls back to the hardcoded map: 35 CNY/kg * 2kg = 70.
	if pb.ShippingCostCNY != 70 {
		t.Errorf("expected fallback shipping cost 70 (35/kg * 2kg), got %.2f", pb.ShippingCostCNY)
	}
}

func TestSourcingRecommend_WithEngine_OverridesShipping(t *testing.T) {
	prev := sourcingEngine
	defer func() { sourcingEngine = prev }()

	// Build an engine with JP per_kg = 30 (cheaper than the 35 static fallback).
	engine := logistics.NewRateEngine([]logistics.RateTableEntry{
		{
			Priority:           1,
			DestinationCountry: "JP",
			CargoType:          "normal",
			MinWeightKg:        0,
			RuleType:           "per_kg",
			PerKgPrice:         30.0,
		},
	})
	SetSourcingEngine(engine)

	resp, err := invokeSourcingRecommend(context.Background(), map[string]interface{}{
		"source_url":   "https://detail.1688.com/offer/example.html",
		"price_1688":   100,
		"weight_kg":    2,
		"destination":  "JP",
		"markup_pct":   250,
	})
	if err != nil {
		t.Fatalf("sourcing.recommend handler error: %v", err)
	}

	m, _ := resp.(map[string]interface{})
	pb, ok := m["profit_breakdown"].(*sourcing.ProfitBreakdown)
	if !ok {
		t.Fatalf("expected *ProfitBreakdown, got %T", m["profit_breakdown"])
	}
	// Engine should override: 30/kg * 2kg = 60, not 70.
	if pb.ShippingCostCNY != 60 {
		t.Errorf("expected engine-based shipping cost 60 (30/kg * 2kg), got %.2f", pb.ShippingCostCNY)
	}
	if pb.ShippingCostCNY == 70 {
		t.Errorf("engine was bypassed — shipping cost matches static fallback 70")
	}
}

func TestSourcingRecommend_EngineNoMatch_FallsBackToMap(t *testing.T) {
	prev := sourcingEngine
	defer func() { sourcingEngine = prev }()

	// Engine only has rates for "RU"; requesting "JP" should fall back to the
	// hardcoded map (35/kg * 2kg = 70).
	engine := logistics.NewRateEngine([]logistics.RateTableEntry{
		{
			Priority:           1,
			DestinationCountry: "RU",
			CargoType:          "normal",
			MinWeightKg:        0,
			RuleType:           "per_kg",
			PerKgPrice:         65.0,
		},
	})
	SetSourcingEngine(engine)

	resp, err := invokeSourcingRecommend(context.Background(), map[string]interface{}{
		"source_url":   "https://detail.1688.com/offer/example.html",
		"price_1688":   100,
		"weight_kg":    2,
		"destination":  "JP",
		"markup_pct":   250,
	})
	if err != nil {
		t.Fatalf("sourcing.recommend handler error: %v", err)
	}

	m, _ := resp.(map[string]interface{})
	pb, ok := m["profit_breakdown"].(*sourcing.ProfitBreakdown)
	if !ok {
		t.Fatalf("expected *ProfitBreakdown, got %T", m["profit_breakdown"])
	}
	if pb.ShippingCostCNY != 70 {
		t.Errorf("expected fallback shipping 70 when engine has no JP match, got %.2f", pb.ShippingCostCNY)
	}
}

func TestSetSourcingEngine_NilIsSafe(t *testing.T) {
	// SetSourcingEngine(nil) must not panic and must clear the engine.
	prev := sourcingEngine
	defer func() { sourcingEngine = prev }()

	SetSourcingEngine(nil)
	if sourcingEngine != nil {
		t.Errorf("expected sourcingEngine to be nil after SetSourcingEngine(nil)")
	}
}
