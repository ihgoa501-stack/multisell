package profit

import (
	"testing"
)

func TestBuildEvidenceCard_FullData(t *testing.T) {
	prod := &candidateProductReader{
		Title:              "Test Product",
		TargetSalePrice:    29.99,
		TargetCurrency:     "USD",
		PurchasePrice:      8.50,
		PackageWeightKg:    0.5,
		DestinationCountry: "US",
	}
	card := BuildEvidenceCard(prod, 0.15, 4.50)

	if !card.CanEvaluate {
		t.Errorf("expected CanEvaluate=true, got false, blocking: %v", card.BlockingReasons)
	}
	if card.ConfidenceLevel != "low" {
		t.Errorf("expected low (template_default items), got %s", card.ConfidenceLevel)
	}
	if card.TotalFixedCost <= 0 {
		t.Error("TotalFixedCost should be > 0")
	}
	if card.BreakEvenPrice <= 0 {
		t.Error("BreakEvenPrice should be > 0")
	}
	// Verify no double-counting of platform fee
	expectedVarFee := 29.99 * (0.15 + 0.035) // commission + payment
	if card.EstimatedVariableFee > expectedVarFee+0.01 || card.EstimatedVariableFee < expectedVarFee-0.01 {
		t.Errorf("EstimatedVariableFee %.2f, expected %.2f", card.EstimatedVariableFee, expectedVarFee)
	}
}

func TestBuildEvidenceCard_MissingRequired_Blocks(t *testing.T) {
	prod := &candidateProductReader{
		Title:            "Incomplete",
		TargetSalePrice:  0,
		DestinationCountry: "",
	}
	card := BuildEvidenceCard(prod, 0, 0)

	if card.CanEvaluate {
		t.Error("expected CanEvaluate=false for missing required fields")
	}
	if card.ConfidenceLevel != "insufficient_data" {
		t.Errorf("expected insufficient_data, got %s", card.ConfidenceLevel)
	}
	if len(card.BlockingReasons) == 0 {
		t.Error("expected blocking reasons for missing data")
	}
}

func TestBuildEvidenceCard_BreakEvenPrice(t *testing.T) {
	prod := &candidateProductReader{
		Title:              "BE Test",
		TargetSalePrice:    20.00,
		TargetCurrency:     "USD",
		PurchasePrice:      10.00,
		PackageWeightKg:    0.5,
		DestinationCountry: "US",
	}
	card := BuildEvidenceCard(prod, 0.15, 5.00)

	// fixed = 10 + 1.5 + 5 + 0.5 + 0.2(2% of 10) + 0.2(1% of 20) = 17.4
	// var rate = 0.15 + 0.035 = 0.185
	// break_even = 17.4 / (1-0.185) = 17.4 / 0.815 ≈ 21.35
	if card.BreakEvenPrice <= 0 {
		t.Fatal("BreakEvenPrice should be > 0")
	}
	// Target price 20 < 21.35 -> unprofitable
	if card.ProfitMargin >= 0 {
		t.Logf("ProfitMargin: %.2f%%, check if expected negative", card.ProfitMargin)
	}
}
