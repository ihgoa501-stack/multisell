package logistics

import (
	"testing"
)

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

// routerTestData extends testTables (defined in logistics_test.go) with a few
// extra entries useful for router-specific test scenarios.
var routerTestTables = func() []RateTableEntry {
	tables := make([]RateTableEntry, len(testTables))
	copy(tables, testTables)

	// Add a slow-cheap channel for RU (per_kg, low cost, slow delivery).
	tables = append(tables, RateTableEntry{
		ID: 10, ChannelName: "SEA_ECONOMY", ProviderName: "CNExpress", RuleType: "per_kg",
		Priority: 10, MinWeightKg: 0, MaxWeightKg: 50,
		DestinationCountry: "RU", CargoType: "normal",
		PerKgPrice: 20, FuelSurchargePct: 0,
		Currency: "CNY",
		EstimatedDeliveryMin: 30, EstimatedDeliveryMax: 45,
	})

	// Add a fast-expensive channel for RU (per_kg, high cost, fast delivery).
	tables = append(tables, RateTableEntry{
		ID: 11, ChannelName: "AIR_EXP", ProviderName: "DHL", RuleType: "per_kg",
		Priority: 0, MinWeightKg: 0, MaxWeightKg: 5,
		DestinationCountry: "RU", CargoType: "normal",
		PerKgPrice: 120, FuelSurchargePct: 0,
		Currency: "CNY",
		EstimatedDeliveryMin: 3, EstimatedDeliveryMax: 5,
	})

	return tables
}()

// defaultRouter returns a Router backed by routerTestTables with default weights.
func defaultRouter() *Router {
	svc := NewService(routerTestTables)
	return NewRouter(svc)
}

// ---------------------------------------------------------------------------
// Basic recommendation
// ---------------------------------------------------------------------------

func TestGetRecommendations_ReturnsSortedResults(t *testing.T) {
	r := defaultRouter()
	req := RouteRequest{
		WeightKg:    5.0,
		Destination: "RU",
		CargoType:   "normal",
	}

	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected at least 1 recommendation for RU/normal/5kg")
	}

	// Results must be sorted by composite score descending.
	for i := 1; i < len(recs); i++ {
		if recs[i-1].CompositeScore < recs[i].CompositeScore {
			t.Errorf("recommendations not sorted descending at index %d-%.2f < %d-%.2f",
				i-1, recs[i-1].CompositeScore, i, recs[i].CompositeScore)
		}
	}

	// Verify all required fields are populated.
	for _, rec := range recs {
		if rec.ChannelName == "" {
			t.Error("expected non-empty ChannelName")
		}
		if rec.ProviderName == "" {
			t.Error("expected non-empty ProviderName")
		}
		if rec.EstimatedCost <= 0 {
			t.Errorf("channel %s: expected positive EstimatedCost, got %.2f", rec.ChannelName, rec.EstimatedCost)
		}
		if rec.RecommendationReason == "" {
			t.Errorf("channel %s: expected non-empty RecommendationReason", rec.ChannelName)
		}
	}
}

func TestGetRecommendations_TopResultHasHighestScore(t *testing.T) {
	r := defaultRouter()
	req := RouteRequest{
		WeightKg:    5.0,
		Destination: "RU",
	}

	recs := r.GetRecommendations(req)
	if len(recs) < 2 {
		t.Fatalf("expected at least 2 recommendations for RU, got %d", len(recs))
	}

	top := recs[0]
	for _, rec := range recs[1:] {
		if top.CompositeScore < rec.CompositeScore {
			t.Errorf("top result (%.2f) should have highest composite score, found %.2f for %s",
				top.CompositeScore, rec.CompositeScore, rec.ChannelName)
		}
	}
}

// ---------------------------------------------------------------------------
// Weight boundaries
// ---------------------------------------------------------------------------

func TestGetRecommendations_LightWeight(t *testing.T) {
	r := defaultRouter()
	// 0.5 kg — should match all RU/normal channels (CNAIR min=0, CNSEA min=0)
	req := RouteRequest{WeightKg: 0.5, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations for 0.5kg RU")
	}
	// CNAIR: per_kg at 45 * 0.5 = 22.5 → total 22.5 (no surcharge/fuel since all zero in testTables)
	// Wait, testTables CNAIR has FuelSurchargePct=5, SurchargeFixed=0, MinimumCharge=100
	// So 0.5*45 = 22.5 base, fuel=22.5*5%=1.125 → total=23.625 → min charge 100 → total=100
	// CNSEA: first_additional: 0.5 within first kg → first_price 80, fuel=80*3%=2.4 → total=82.4
	// SEA_ECONOMY: per_kg 20*0.5=10, total=10
	// AIR_EXP: per_kg 120*0.5=60, total=60 (MaxWeightKg=5, 0.5 within)
	for _, rec := range recs {
		if rec.EstimatedCost <= 0 {
			t.Errorf("channel %s: expected positive cost, got %.2f", rec.ChannelName, rec.EstimatedCost)
		}
	}
}

func TestGetRecommendations_HeavyWeight_ExcludesSome(t *testing.T) {
	r := defaultRouter()
	// 40 kg — only CNSEA (max=50) and SEA_ECONOMY (max=50) should match.
	// CNAIR max=30 excluded, AIR_EXP max=5 excluded.
	req := RouteRequest{WeightKg: 40, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations for 40kg RU (CNSEA + SEA_ECONOMY)")
	}
	for _, rec := range recs {
		if rec.ChannelName == "CNAIR" {
			t.Error("CNAIR (max 30kg) should be excluded for 40kg cargo")
		}
		if rec.ChannelName == "AIR_EXP" {
			t.Error("AIR_EXP (max 5kg) should be excluded for 40kg cargo")
		}
	}
}

func TestGetRecommendations_ExcessiveWeight_ReturnsNil(t *testing.T) {
	r := defaultRouter()
	// 100 kg — exceeds all max weights.
	req := RouteRequest{WeightKg: 100, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if recs != nil {
		t.Errorf("expected nil for 100kg (exceeds all rate limits), got %d results", len(recs))
	}
}

// ---------------------------------------------------------------------------
// Destination filtering
// ---------------------------------------------------------------------------

func TestGetRecommendations_DestinationKZ(t *testing.T) {
	r := defaultRouter()
	// KZ has CNAIR fixed (fee=200, fuel 3% → 206) only in testTables.
	req := RouteRequest{WeightKg: 3.0, Destination: "KZ"}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected at least 1 recommendation for KZ")
	}
	if recs[0].ChannelName != "CNAIR" {
		t.Errorf("expected CNAIR for KZ, got %s", recs[0].ChannelName)
	}
	if recs[0].EstimatedCost != 206.0 {
		t.Errorf("expected total 206 for KZ CNAIR fixed, got %.2f", recs[0].EstimatedCost)
	}
}

func TestGetRecommendations_NoChannelsForDestination(t *testing.T) {
	r := defaultRouter()
	// "US" has no rate tables in testTables.
	req := RouteRequest{WeightKg: 1.0, Destination: "US"}
	recs := r.GetRecommendations(req)
	if recs != nil {
		t.Errorf("expected nil for US (no rate tables), got %d results", len(recs))
	}
}

// ---------------------------------------------------------------------------
// Time constraint (MaxDays)
// ---------------------------------------------------------------------------

func TestGetRecommendations_MaxDaysFiltersSlowChannels(t *testing.T) {
	r := defaultRouter()
	req := RouteRequest{
		WeightKg:    5.0,
		Destination: "RU",
		MaxDays:     10, // only channels with min ETA <= 10 days
	}

	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected at least 1 recommendation within 10 days for RU/5kg")
	}

	// All returned channels must have estimated_min <= 10.
	// In testTables + extra:
	//   CNAIR: 7-12 → min=7 <= 10 ✓
	//   CNSEA: 20-30 → min=20 > 10 ✗
	//   SEA_ECONOMY: 30-45 → min=30 > 10 ✗
	//   AIR_EXP: 3-5 → min=3 <= 10 ✓
	for _, rec := range recs {
		if rec.EstimatedDaysMin > 10 {
			t.Errorf("channel %s (ETA %d-%d days) exceeds MaxDays=10",
				rec.ChannelName, rec.EstimatedDaysMin, rec.EstimatedDaysMax)
		}
	}
}

func TestGetRecommendations_MaxDaysExcludesAll_ReturnsNil(t *testing.T) {
	r := defaultRouter()
	// No RU channel delivers within 2 days.
	req := RouteRequest{WeightKg: 1.0, Destination: "RU", MaxDays: 2}
	recs := r.GetRecommendations(req)
	if recs != nil {
		t.Errorf("expected nil when MaxDays=2 excludes all channels, got %d results", len(recs))
	}
}

// ---------------------------------------------------------------------------
// Cost constraint (MaxCost)
// ---------------------------------------------------------------------------

func TestGetRecommendations_MaxCostFiltersExpensiveChannels(t *testing.T) {
	r := defaultRouter()
	req := RouteRequest{
		WeightKg:    5.0,
		Destination: "RU",
		MaxCost:     100.0, // only channels with total <= 100
	}

	recs := r.GetRecommendations(req)
	for _, rec := range recs {
		if rec.EstimatedCost > 100.0 {
			t.Errorf("channel %s (cost %.2f) exceeds MaxCost=100",
				rec.ChannelName, rec.EstimatedCost)
		}
	}
}

func TestGetRecommendations_MaxCostExcludesAll_ReturnsNil(t *testing.T) {
	r := defaultRouter()
	// All RU channels at 1kg cost more than 1 CNY.
	req := RouteRequest{WeightKg: 1.0, Destination: "RU", MaxCost: 1.0}
	recs := r.GetRecommendations(req)
	if recs != nil {
		t.Errorf("expected nil when MaxCost=1 excludes all, got %d results", len(recs))
	}
}

// ---------------------------------------------------------------------------
// Both constraints combined
// ---------------------------------------------------------------------------

func TestGetRecommendations_MaxDaysAndMaxCost(t *testing.T) {
	r := defaultRouter()
	// At 5kg RU:
	//   CNAIR:   cost=236.25,  days=7-12
	//   CNSEA:   cost=206,     days=20-30
	//   SEA_ECO: cost=100,     days=30-45
	//   AIR_EXP: cost=600,     days=3-5
	// With MaxDays=15 (ETA min <= 15) and MaxCost=250:
	//   CNAIR:   7 <= 15 ✓, 236.25 <= 250 ✓ → included
	//   CNSEA:   20 > 15 ✗ → excluded
	//   SEA_ECO: 30 > 15 ✗ → excluded
	//   AIR_EXP: 3 <= 15 ✓, 600 > 250 ✗ → excluded
	// Result: only CNAIR
	req := RouteRequest{WeightKg: 5.0, Destination: "RU", MaxDays: 15, MaxCost: 250}
	recs := r.GetRecommendations(req)
	if len(recs) != 1 {
		t.Fatalf("expected exactly 1 recommendation (CNAIR) with MaxDays=15 MaxCost=250, got %d", len(recs))
	}
	if recs[0].ChannelName != "CNAIR" {
		t.Errorf("expected CNAIR, got %s", recs[0].ChannelName)
	}
}

// ---------------------------------------------------------------------------
// Custom weights
// ---------------------------------------------------------------------------

func TestGetRecommendations_CustomWeights_CostFocused(t *testing.T) {
	svc := NewService(routerTestTables)

	// Cost-only weights: only cost score matters.
	r := NewRouter(svc, WithWeights(RouterWeights{CostWeight: 1.0, SpeedWeight: 0, ReliabilityWeight: 0}))

	req := RouteRequest{WeightKg: 5.0, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}

	// Cheapest channel should be SEA_ECONOMY (5*20=100).
	if recs[0].ChannelName != "SEA_ECONOMY" {
		t.Errorf("with cost-only weights, cheapest (SEA_ECONOMY) should be first, got %s (cost=%.2f)",
			recs[0].ChannelName, recs[0].EstimatedCost)
	}
}

func TestGetRecommendations_CustomWeights_SpeedFocused(t *testing.T) {
	svc := NewService(routerTestTables)

	// Speed-only weights: only delivery time matters.
	r := NewRouter(svc, WithWeights(RouterWeights{CostWeight: 0, SpeedWeight: 1.0, ReliabilityWeight: 0}))

	req := RouteRequest{WeightKg: 1.0, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}

	// Fastest channel should be AIR_EXP (ETA 3-5 days).
	if recs[0].ChannelName != "AIR_EXP" {
		t.Errorf("with speed-only weights, fastest (AIR_EXP) should be first, got %s (ETA %d-%d days)",
			recs[0].ChannelName, recs[0].EstimatedDaysMin, recs[0].EstimatedDaysMax)
	}
}

func TestGetRecommendations_ZeroWeightsFallsBackToDefaults(t *testing.T) {
	svc := NewService(routerTestTables)
	// Zero weights should fall back to defaults (i.e., not panic/crash).
	r := NewRouter(svc, WithWeights(RouterWeights{}))

	req := RouteRequest{WeightKg: 5.0, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations with fallback default weights")
	}
	// Verify composite scores are computed (not zero).
	for _, rec := range recs {
		if rec.CompositeScore == 0 {
			t.Errorf("channel %s: expected non-zero composite score with default weights", rec.ChannelName)
		}
	}
}

func TestGetRecommendations_NegativeWeightFallsBackToDefaults(t *testing.T) {
	svc := NewService(routerTestTables)
	// Negative weights should fall back to defaults.
	r := NewRouter(svc, WithWeights(RouterWeights{CostWeight: -1, SpeedWeight: 0, ReliabilityWeight: 0}))

	req := RouteRequest{WeightKg: 5.0, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations with fallback default weights")
	}
	// Should still have non-zero composite scores.
	for _, rec := range recs {
		if rec.CompositeScore == 0 {
			t.Errorf("expected non-zero composite score with default fallback")
		}
	}
}

// ---------------------------------------------------------------------------
// Carrier performance integration (reliability score)
// ---------------------------------------------------------------------------

func TestGetRecommendations_ReliabilityFromCarrierPerformance(t *testing.T) {
	svc := NewService(routerTestTables)
	// Record carrier performance with low loss rate → high reliability.
	svc.recordCarrierPerformance("CNAIR", "CNExpress", 236.25, 9, false)
	svc.recordCarrierPerformance("CNAIR", "CNExpress", 200.00, 10, false)

	r := NewRouter(svc)
	req := RouteRequest{WeightKg: 5.0, Destination: "RU"}

	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}

	// Find CNAIR recommendation and verify its reliability score.
	for _, rec := range recs {
		if rec.ChannelName == "CNAIR" {
			// LossRate = 0% (0 lost out of 2) → reliability = 1.0 - 0 = 1.0
			if rec.ReliabilityScore != 1.0 {
				t.Errorf("expected reliability score 1.0 for CNAIR (no losses), got %.2f", rec.ReliabilityScore)
			}
			break
		}
	}
}

func TestGetRecommendations_ReliabilityDegradedByLosses(t *testing.T) {
	svc := NewService(routerTestTables)
	// Record high loss rate for CNAIR → low reliability.
	svc.recordCarrierPerformance("CNAIR", "CNExpress", 236.25, 9, false)
	svc.recordCarrierPerformance("CNAIR", "CNExpress", 200.00, 10, true) // lost package

	r := NewRouter(svc)
	req := RouteRequest{WeightKg: 5.0, Destination: "RU"}

	recs := r.GetRecommendations(req)
	for _, rec := range recs {
		if rec.ChannelName == "CNAIR" {
			// LossRate = 50% → reliability = 1.0 - 0.5 = 0.5
			if rec.ReliabilityScore != 0.5 {
				t.Errorf("expected reliability score 0.5 for CNAIR (50%% loss), got %.2f", rec.ReliabilityScore)
			}
			break
		}
	}
}

func TestGetRecommendations_NeutralReliabilityWhenNoData(t *testing.T) {
	svc := NewService(routerTestTables)
	// No carrier performance recorded → all channels get 0.5 reliability.

	r := NewRouter(svc)
	req := RouteRequest{WeightKg: 1.0, Destination: "RU"}

	recs := r.GetRecommendations(req)
	for _, rec := range recs {
		if rec.ReliabilityScore != 0.5 {
			t.Errorf("channel %s: expected neutral reliability 0.5, got %.2f", rec.ChannelName, rec.ReliabilityScore)
		}
	}
}

// ---------------------------------------------------------------------------
// Recommendation reasons
// ---------------------------------------------------------------------------

func TestGetRecommendations_TopResultReason(t *testing.T) {
	r := defaultRouter()
	req := RouteRequest{WeightKg: 1.0, Destination: "RU"}

	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}

	// Top result should start with "Recommended choice".
	if len(recs[0].RecommendationReason) == 0 {
		t.Fatal("expected non-empty recommendation reason for top result")
	}
	if !contains(recs[0].RecommendationReason, "Recommended choice") {
		t.Errorf("top result should contain 'Recommended choice', got %q", recs[0].RecommendationReason)
	}

	// Non-top results should contain "Alternative".
	if len(recs) > 1 {
		for i := 1; i < len(recs); i++ {
			if !contains(recs[i].RecommendationReason, "Alternative") {
				t.Errorf("result %d should contain 'Alternative', got %q", i, recs[i].RecommendationReason)
			}
		}
	}
}

func TestGetRecommendations_CheapestChannelMarked(t *testing.T) {
	r := defaultRouter()
	req := RouteRequest{WeightKg: 1.0, Destination: "RU"}

	recs := r.GetRecommendations(req)

	// Find the cheapest channel.
	minCost := recs[0].EstimatedCost
	for _, rec := range recs {
		if rec.EstimatedCost < minCost {
			minCost = rec.EstimatedCost
		}
	}

	// At least one channel should have "Lowest price" in its reason.
	found := false
	for _, rec := range recs {
		if contains(rec.RecommendationReason, "Lowest price") {
			found = true
			if rec.EstimatedCost != minCost {
				t.Errorf("channel %s marked as lowest price but cost=%.2f > min=%.2f",
					rec.ChannelName, rec.EstimatedCost, minCost)
			}
			break
		}
	}
	if !found {
		t.Error("expected at least one channel marked 'Lowest price'")
	}
}

func TestGetRecommendations_NonCheapestHasCostDiff(t *testing.T) {
	r := defaultRouter()
	req := RouteRequest{WeightKg: 1.0, Destination: "RU"}

	recs := r.GetRecommendations(req)
	if len(recs) < 2 {
		t.Fatalf("need at least 2 recommendations for cost diff test, got %d", len(recs))
	}

	// Cheapest channel should NOT have a price diff in its reason.
	// (We find the cheapest by looking for "Lowest price" or computing.)
	minCost := recs[0].EstimatedCost
	for _, rec := range recs {
		if rec.EstimatedCost < minCost {
			minCost = rec.EstimatedCost
		}
	}

	for _, rec := range recs {
		if rec.EstimatedCost == minCost {
			continue
		}
		if !contains(rec.RecommendationReason, "more than cheapest") {
			t.Errorf("non-cheapest channel %s should show price diff in reason, got %q",
				rec.ChannelName, rec.RecommendationReason)
		}
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestGetRecommendations_NilServiceEngine(t *testing.T) {
	// Service with nil rate tables (engine created empty).
	svc := NewService(nil)
	r := NewRouter(svc)

	req := RouteRequest{WeightKg: 1.0, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if recs != nil {
		t.Errorf("expected nil for empty rate engine, got %d results", len(recs))
	}
}

func TestGetRecommendations_ZeroWeight(t *testing.T) {
	// Zero weight should still work — matches all entries with minWeightKg=0.
	r := defaultRouter()
	req := RouteRequest{WeightKg: 0, Destination: "RU"}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations for 0kg RU")
	}
}

func TestGetRecommendations_EmptyCargoTypeDefaultsToNormal(t *testing.T) {
	r := defaultRouter()
	req := RouteRequest{WeightKg: 3.0, Destination: "KZ", CargoType: ""}
	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations for KZ with empty cargo type (defaults to normal)")
	}
	// KZ normal should match CNAIR fixed.
	if recs[0].ChannelName != "CNAIR" {
		t.Errorf("expected CNAIR for KZ normal, got %s", recs[0].ChannelName)
	}
}

// ---------------------------------------------------------------------------
// Volumetric weight integration
// ---------------------------------------------------------------------------

func TestGetRecommendations_VolumetricWeightApplied(t *testing.T) {
	r := defaultRouter()
	// Light package, large dimensions: 60x50x40cm = 20kg volumetric.
	req := RouteRequest{
		WeightKg:    0.5,
		LengthCm:    60,
		WidthCm:     50,
		HeightCm:    40,
		Destination: "RU",
	}

	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations for volumetric RU")
	}

	// CNAIR per_kg at 45 * 20 = 900 base + 5% fuel = 945
	// CNSEA first_additional: 80 + (20-1)*30 = 650 + 3% fuel = 669.50
	// AIR_EXP: max=5, 20 > 5 → excluded by rate engine
	// SEA_ECONOMY: per_kg 20*20=400, no fuel → 400
	// So SEA_ECONOMY should be cheapest at 400.
	foundSea := false
	for _, rec := range recs {
		if rec.ChannelName == "SEA_ECONOMY" {
			foundSea = true
			if rec.EstimatedCost != 400 {
				t.Errorf("SEA_ECONOMY expected cost 400 for 20kg volumetric, got %.2f", rec.EstimatedCost)
			}
			break
		}
	}
	if !foundSea {
		t.Error("SEA_ECONOMY should be recommended for RU/20kg volumetric")
	}
}

// ---------------------------------------------------------------------------
// RouterWeights validation
// ---------------------------------------------------------------------------

func TestRouterWeights_Validate(t *testing.T) {
	tests := []struct {
		name    string
		w       RouterWeights
		wantErr bool
	}{
		{"default weights", DefaultRouterWeights, false},
		{"cost only", RouterWeights{CostWeight: 1}, false},
		{"speed only", RouterWeights{SpeedWeight: 1}, false},
		{"reliability only", RouterWeights{ReliabilityWeight: 1}, false},
		{"all zero", RouterWeights{}, true},
		{"negative cost", RouterWeights{CostWeight: -0.5, SpeedWeight: 0.5}, true},
		{"negative speed", RouterWeights{CostWeight: 0.5, SpeedWeight: -0.3, ReliabilityWeight: 0.2}, true},
		{"all positive", RouterWeights{CostWeight: 0.4, SpeedWeight: 0.35, ReliabilityWeight: 0.25}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.w.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Single-result scoring
// ---------------------------------------------------------------------------

func TestGetRecommendations_SingleCandidateGetsMaxScore(t *testing.T) {
	svc := NewService(routerTestTables)

	// KZ only has one channel (CNAIR fixed) in testTables → single candidate.
	r := NewRouter(svc)
	req := RouteRequest{WeightKg: 3.0, Destination: "KZ"}

	recs := r.GetRecommendations(req)
	if len(recs) != 1 {
		t.Fatalf("expected exactly 1 recommendation for KZ, got %d", len(recs))
	}
	if recs[0].CompositeScore != 1.0 {
		t.Errorf("single candidate should have CompositeScore=1.0, got %.2f", recs[0].CompositeScore)
	}
}

// ---------------------------------------------------------------------------
// Battery cargo type (matches cargo-type-specific channels)
// ---------------------------------------------------------------------------

func TestGetRecommendations_BatteryCargoType(t *testing.T) {
	r := defaultRouter()
	// Only CNAIR_EXPRESS (battery) matches RU + battery.
	req := RouteRequest{WeightKg: 2.0, Destination: "RU", CargoType: "battery"}

	recs := r.GetRecommendations(req)
	if len(recs) == 0 {
		t.Fatal("expected recommendations for RU/battery")
	}
	if recs[0].ChannelName != "CNAIR_EXPRESS" {
		t.Errorf("expected CNAIR_EXPRESS for battery cargo, got %s", recs[0].ChannelName)
	}
	// CNAIR_EXPRESS per_kg at 60 * 2 = 120, fuel=120*5%=6, min=150 → total=150
	if recs[0].EstimatedCost != 150.0 {
		t.Errorf("CNAIR_EXPRESS 2kg battery expected total 150, got %.2f", recs[0].EstimatedCost)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

// containsStr is a simple substring check to avoid importing strings in tests.
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
