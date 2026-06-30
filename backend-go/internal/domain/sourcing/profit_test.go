package sourcing

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/domain/logistics"
)

func TestCalculateProfit_HighMarkupViable(t *testing.T) {
	s := NewProfitService(nil)
	r := s.CalculateProfit(&ProfitInput{SourcePriceCNY: 50, WeightKg: 1, Destination: "US", MarkupPct: 400})
	if !r.IsViable {
		t.Errorf("expected viable for 400%% markup, got margin=%.2f%%", r.MarginPct)
	}
	if r.ProfitCNY <= 0 {
		t.Errorf("expected positive profit, got %.2f", r.ProfitCNY)
	}
}

func TestCalculateProfit_LowMarkupNotViable(t *testing.T) {
	s := NewProfitService(nil)
	r := s.CalculateProfit(&ProfitInput{SourcePriceCNY: 200, WeightKg: 5, Destination: "RU", MarkupPct: 120})
	if r.IsViable {
		t.Errorf("expected not viable, got margin=%.2f%%", r.MarginPct)
	}
}

func TestCalculateProfit_ZeroMarkupDefaults(t *testing.T) {
	s := NewProfitService(nil)
	r := s.CalculateProfit(&ProfitInput{SourcePriceCNY: 50, WeightKg: 1, Destination: "US"})
	if r.MarkupPct != 250 {
		t.Errorf("expected default 250%%, got %.0f%%", r.MarkupPct)
	}
}

func TestCalculateProfit_ZeroWeight(t *testing.T) {
	s := NewProfitService(nil)
	r := s.CalculateProfit(&ProfitInput{SourcePriceCNY: 50, WeightKg: 0, Destination: "US", MarkupPct: 300})
	if r.ShippingCostCNY != 0 {
		t.Errorf("expected zero shipping, got %.2f", r.ShippingCostCNY)
	}
}

func TestCalculateProfit_UnknownDestinationDefaults(t *testing.T) {
	s := NewProfitService(nil)
	r := s.CalculateProfit(&ProfitInput{SourcePriceCNY: 50, WeightKg: 1, Destination: "XX", MarkupPct: 300})
	if r.ShippingCostCNY <= 0 {
		t.Errorf("expected default shipping cost, got %.2f", r.ShippingCostCNY)
	}
}

func TestCalculateProfit_Precision(t *testing.T) {
	s := NewProfitService(nil)
	r := s.CalculateProfit(&ProfitInput{SourcePriceCNY: 15.333, WeightKg: 0.567, Destination: "EU", MarkupPct: 250})
	if r.SourcePriceCNY != 15.33 {
		t.Errorf("expected 15.33, got %.2f", r.SourcePriceCNY)
	}
}

func TestCalculateProfit_WithRateEngine(t *testing.T) {
	// Create a RateEngine with entries for JP at 30.0 CNY/kg.
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

	s := NewProfitService(engine)
	// JP hardcoded map value is 35.0; engine should override to 30.0.
	r := s.CalculateProfit(&ProfitInput{SourcePriceCNY: 100, WeightKg: 2, Destination: "JP", MarkupPct: 250})

	// Expected shipping: 30.0 * 2 = 60.0 (from engine, not 35.0*2=70.0 from map)
	if r.ShippingCostCNY != 60.0 {
		t.Errorf("expected engine-based shipping cost 60.0, got %.2f", r.ShippingCostCNY)
	}
	// Verify it's not using the hardcoded map value.
	if r.ShippingCostCNY == 70.0 {
		t.Errorf("engine was bypassed — shipping cost matches hardcoded map value 70.0")
	}
}

func TestCalculateProfit_EngineFallback(t *testing.T) {
	// Nil engine should fall back to the hardcoded shipping estimate map.
	s := NewProfitService(nil)
	r := s.CalculateProfit(&ProfitInput{SourcePriceCNY: 100, WeightKg: 2, Destination: "JP", MarkupPct: 250})

	// JP hardcoded map value is 35.0; shipping = 35.0 * 2 = 70.0.
	if r.ShippingCostCNY != 70.0 {
		t.Errorf("expected fallback shipping cost 70.0, got %.2f", r.ShippingCostCNY)
	}
}
