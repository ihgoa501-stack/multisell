package sourcing

import "testing"

func TestCalculateProfit_HighMarkupViable(t *testing.T) {
	r := CalculateProfit(&ProfitInput{SourcePriceCNY: 50, WeightKg: 1, Destination: "US", MarkupPct: 400})
	if !r.IsViable {
		t.Errorf("expected viable for 400%% markup, got margin=%.2f%%", r.MarginPct)
	}
	if r.ProfitCNY <= 0 {
		t.Errorf("expected positive profit, got %.2f", r.ProfitCNY)
	}
}

func TestCalculateProfit_LowMarkupNotViable(t *testing.T) {
	r := CalculateProfit(&ProfitInput{SourcePriceCNY: 200, WeightKg: 5, Destination: "RU", MarkupPct: 120})
	if r.IsViable {
		t.Errorf("expected not viable, got margin=%.2f%%", r.MarginPct)
	}
}

func TestCalculateProfit_ZeroMarkupDefaults(t *testing.T) {
	r := CalculateProfit(&ProfitInput{SourcePriceCNY: 50, WeightKg: 1, Destination: "US"})
	if r.MarkupPct != 250 {
		t.Errorf("expected default 250%%, got %.0f%%", r.MarkupPct)
	}
}

func TestCalculateProfit_ZeroWeight(t *testing.T) {
	r := CalculateProfit(&ProfitInput{SourcePriceCNY: 50, WeightKg: 0, Destination: "US", MarkupPct: 300})
	if r.ShippingCostCNY != 0 {
		t.Errorf("expected zero shipping, got %.2f", r.ShippingCostCNY)
	}
}

func TestCalculateProfit_UnknownDestinationDefaults(t *testing.T) {
	r := CalculateProfit(&ProfitInput{SourcePriceCNY: 50, WeightKg: 1, Destination: "XX", MarkupPct: 300})
	if r.ShippingCostCNY <= 0 {
		t.Errorf("expected default shipping cost, got %.2f", r.ShippingCostCNY)
	}
}

func TestCalculateProfit_Precision(t *testing.T) {
	r := CalculateProfit(&ProfitInput{SourcePriceCNY: 15.333, WeightKg: 0.567, Destination: "EU", MarkupPct: 250})
	if r.SourcePriceCNY != 15.33 {
		t.Errorf("expected 15.33, got %.2f", r.SourcePriceCNY)
	}
}
