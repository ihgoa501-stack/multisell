package logistics

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ── Evaluate: matched ──────────────────────────────────────────────────────

func TestEvaluate_Match(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationConfig{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	// Create a config: CN→RU, min 50kg → 60 CNY/kg
	db.Create(&ConsolidationConfig{
		SourceCountry:    "CN",
		Destination:      "RU",
		MinTotalWeightKg: 50,
		NegotiatedRate:   60,
		EffectiveFrom:    time.Now().Add(-24 * time.Hour),
	})

	items := []ConsolidationItem{
		{SellerID: 1, SkuID: 100, WeightKg: 30},
		{SellerID: 2, SkuID: 200, WeightKg: 30},
	}

	result, err := svc.Evaluate(items, "RU")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if result.TotalWeightKg != 60 {
		t.Errorf("total weight: expected 60, got %.2f", result.TotalWeightKg)
	}
	if result.NegotiatedRate != 60 {
		t.Errorf("negotiated rate: expected 60, got %.2f", result.NegotiatedRate)
	}
	if result.MatchedConfig == nil {
		t.Fatal("matched config should not be nil")
	}
	if result.MatchedConfig.Destination != "RU" {
		t.Errorf("matched config destination: expected RU, got %s", result.MatchedConfig.Destination)
	}

	// OriginalRate should be DefaultStandardRate = 100
	if result.OriginalRate != DefaultStandardRate {
		t.Errorf("original rate: expected %.2f, got %.2f", DefaultStandardRate, result.OriginalRate)
	}

	// Verify negotiated rate < original rate
	if result.NegotiatedRate >= result.OriginalRate {
		t.Errorf("negotiated rate (%.2f) should be lower than original rate (%.2f)",
			result.NegotiatedRate, result.OriginalRate)
	}

	// Savings: (100-60) * 60 = 2400
	expectedSavings := (DefaultStandardRate - result.NegotiatedRate) * result.TotalWeightKg
	if result.Savings != expectedSavings {
		t.Errorf("savings: expected %.2f, got %.2f", expectedSavings, result.Savings)
	}

	// SavingsPct: 2400 / (100*60) * 100 = 40%
	expectedSavingsPct := (result.Savings / (result.OriginalRate * result.TotalWeightKg)) * 100
	if result.SavingsPct != expectedSavingsPct {
		t.Errorf("savings pct: expected %.2f, got %.2f", expectedSavingsPct, result.SavingsPct)
	}
}

// ── Evaluate: not matched ──────────────────────────────────────────────────

func TestEvaluate_NoMatch(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationConfig{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	// Config requires 200kg minimum
	db.Create(&ConsolidationConfig{
		SourceCountry:    "CN",
		Destination:      "RU",
		MinTotalWeightKg: 200,
		NegotiatedRate:   55,
		EffectiveFrom:    time.Now().Add(-24 * time.Hour),
	})

	items := []ConsolidationItem{
		{SellerID: 1, SkuID: 100, WeightKg: 30},
	}

	_, err := svc.Evaluate(items, "RU")
	if err == nil {
		t.Fatal("expected error for weight below min_total_weight_kg, got nil")
	}
}

// ── Evaluate: wrong destination ────────────────────────────────────────────

func TestEvaluate_WrongDestination(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationConfig{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	db.Create(&ConsolidationConfig{
		SourceCountry:    "CN",
		Destination:      "RU",
		MinTotalWeightKg: 10,
		NegotiatedRate:   60,
		EffectiveFrom:    time.Now().Add(-24 * time.Hour),
	})

	items := []ConsolidationItem{
		{WeightKg: 50},
	}

	// Query for "KZ" — no config matches
	_, err := svc.Evaluate(items, "KZ")
	if err == nil {
		t.Fatal("expected error for wrong destination, got nil")
	}
}

// ── Evaluate: multiple rules, priority by lowest rate ──────────────────────

func TestEvaluate_MultipleRules_Priority(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationConfig{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	// Rule A: higher min weight, lower rate (better deal if qualified)
	db.Create(&ConsolidationConfig{
		SourceCountry:    "CN",
		Destination:      "RU",
		MinTotalWeightKg: 100,
		NegotiatedRate:   55,
		EffectiveFrom:    time.Now().Add(-24 * time.Hour),
	})
	// Rule B: lower min weight, higher rate
	db.Create(&ConsolidationConfig{
		SourceCountry:    "CN",
		Destination:      "RU",
		MinTotalWeightKg: 10,
		NegotiatedRate:   65,
		EffectiveFrom:    time.Now().Add(-24 * time.Hour),
	})

	// 150kg ≥ both thresholds → both match → pick lowest rate (55)
	items := []ConsolidationItem{
		{WeightKg: 150},
	}

	result, err := svc.Evaluate(items, "RU")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result.NegotiatedRate != 55 {
		t.Errorf("expected lowest negotiated rate 55, got %.2f", result.NegotiatedRate)
	}
	if result.TotalWeightKg != 150 {
		t.Errorf("total weight: expected 150, got %.2f", result.TotalWeightKg)
	}
}

// ── Evaluate: effective date range ─────────────────────────────────────────

func TestEvaluate_EffectiveDateRange(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationConfig{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	now := time.Now()

	// Config with EffectiveTo in the past — should NOT match
	past := now.Add(-48 * time.Hour)
	db.Create(&ConsolidationConfig{
		SourceCountry:    "CN",
		Destination:      "RU",
		MinTotalWeightKg: 10,
		NegotiatedRate:   70,
		EffectiveFrom:    now.Add(-72 * time.Hour),
		EffectiveTo:      &past,
	})

	items := []ConsolidationItem{
		{WeightKg: 30},
	}

	_, err := svc.Evaluate(items, "RU")
	if err == nil {
		t.Fatal("expected error for expired config, got nil")
	}
}
