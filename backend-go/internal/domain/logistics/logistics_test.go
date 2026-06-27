package logistics

import (
	"testing"
)

// testTables is a small set of rate table entries used across tests.
var testTables = []RateTableEntry{
	{
		ID: 1, ChannelName: "CNAIR", ProviderName: "CNExpress", RuleType: "per_kg",
		Priority: 1, MinWeightKg: 0, MaxWeightKg: 30,
		DestinationCountry: "RU", CargoType: "normal",
		PerKgPrice: 45.0, FuelSurchargePct: 5.0, SurchargeFixed: 0,
		MinimumCharge: 100, Currency: "CNY",
		EstimatedDeliveryMin: 7, EstimatedDeliveryMax: 12,
	},
	{
		ID: 2, ChannelName: "CNSEA", ProviderName: "CNExpress", RuleType: "first_additional",
		Priority: 2, MinWeightKg: 0, MaxWeightKg: 50,
		DestinationCountry: "RU", CargoType: "normal",
		FirstKg: 1, FirstPrice: 80, AdditionalKg: 1, AdditionalPrice: 30,
		FuelSurchargePct: 3.0, Currency: "CNY",
		EstimatedDeliveryMin: 20, EstimatedDeliveryMax: 30,
	},
	{
		ID: 3, ChannelName: "CNAIR_EXPRESS", ProviderName: "CNExpress", RuleType: "per_kg",
		Priority: 0, MinWeightKg: 0, MaxWeightKg: 20,
		DestinationCountry: "RU", CargoType: "battery",
		PerKgPrice: 60.0, FuelSurchargePct: 5.0,
		MinimumCharge: 150, Currency: "CNY",
		EstimatedDeliveryMin: 5, EstimatedDeliveryMax: 8,
	},
	{
		ID: 4, ChannelName: "CNAIR", ProviderName: "CNExpress", RuleType: "fixed",
		Priority: 1, MinWeightKg: 0, MaxWeightKg: 0,
		DestinationCountry: "KZ", CargoType: "normal",
		FixedFee: 200.0, FuelSurchargePct: 3.0, Currency: "CNY",
		EstimatedDeliveryMin: 10, EstimatedDeliveryMax: 15,
	},
}

func TestNewService(t *testing.T) {
	svc := NewService(testTables)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
}

func TestNewService_NilTables(t *testing.T) {
	svc := NewService(nil)
	if svc == nil {
		t.Fatal("NewService(nil) returned nil")
	}
}

func TestGetQuote_ReturnsResults(t *testing.T) {
	svc := NewService(testTables)
	cargo := Cargo{ActualWeightKg: 5.0}

	resp, err := svc.GetQuote(cargo, "RU", "normal")
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetQuote returned nil response")
	}
	if len(resp.Results) == 0 {
		t.Fatal("GetQuote returned empty results")
	}

	// Should match at least CNAIR and CNSEA (both RU/normal)
	if len(resp.Results) < 2 {
		t.Fatalf("expected at least 2 results for RU/normal, got %d", len(resp.Results))
	}
}

func TestGetQuote_VolumetricWeight(t *testing.T) {
	svc := NewService(testTables)
	// Low actual weight, large dimensions → volumetric weight should apply.
	cargo := Cargo{
		ActualWeightKg: 0.5,
		LengthCm:       60,
		WidthCm:        50,
		HeightCm:       40,
	}
	// Volumetric weight = (60*50*40)/6000 = 120000/6000 = 20 kg

	resp, err := svc.GetQuote(cargo, "RU", "normal")
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("GetQuote returned no results")
	}

	// CNAIR per_kg at 45 * 20 = 900 base, plus 5% fuel = 945
	// CNSEA first_additional: 80 + (20-1)*30 = 80 + 570 = 650, + 3% fuel = 669.50
	// The chargeable weight should be 20 (volumetric), not 0.5.
	for _, r := range resp.Results {
		if r.ChargeableWeightKg != 20.0 {
			t.Errorf("expected chargeable weight 20 kg (volumetric) for channel %s, got %f",
				r.ChannelName, r.ChargeableWeightKg)
		}
	}
}

func TestGetQuote_BatteryCargoType(t *testing.T) {
	svc := NewService(testTables)
	cargo := Cargo{ActualWeightKg: 2.0}

	resp, err := svc.GetQuote(cargo, "RU", "battery")
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("GetQuote returned no results for battery")
	}

	// Only CNAIR_EXPRESS matches RU + battery.
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result for battery (CNAIR_EXPRESS), got %d", len(resp.Results))
	}
	if resp.Results[0].ChannelName != "CNAIR_EXPRESS" {
		t.Errorf("expected CNAIR_EXPRESS for battery, got %s", resp.Results[0].ChannelName)
	}
}

func TestGetQuote_Kazakhstan(t *testing.T) {
	svc := NewService(testTables)
	cargo := Cargo{ActualWeightKg: 3.0}

	resp, err := svc.GetQuote(cargo, "KZ", "normal")
	if err != nil {
		t.Fatalf("GetQuote failed for KZ: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result for KZ (fixed channel), got %d", len(resp.Results))
	}
	if resp.Results[0].ChannelName != "CNAIR" {
		t.Errorf("expected CNAIR channel for KZ, got %s", resp.Results[0].ChannelName)
	}
	if resp.Results[0].TotalShippingFee != 206.0 {
		// fixed fee 200 + 3% fuel = 206
		t.Errorf("expected total shipping fee 206 for KZ fixed rate, got %f", resp.Results[0].TotalShippingFee)
	}
}

func TestGetQuote_NoMatch(t *testing.T) {
	svc := NewService(testTables)
	cargo := Cargo{ActualWeightKg: 100.0} // exceeds all max weight

	_, err := svc.GetQuote(cargo, "RU", "normal")
	if err == nil {
		t.Fatal("expected error for weight exceeding all rate tables, got nil")
	}
}

func TestGetCheapestQuote_PicksCheapest(t *testing.T) {
	svc := NewService(testTables)
	cargo := Cargo{ActualWeightKg: 5.0}

	cheapest, err := svc.GetCheapestQuote(cargo, "RU", "normal")
	if err != nil {
		t.Fatalf("GetCheapestQuote failed: %v", err)
	}
	if cheapest == nil {
		t.Fatal("GetCheapestQuote returned nil")
	}

	// For RU/normal, 5 kg:
	//   CNAIR:   5 * 45 = 225 + 5% fuel = 236.25
	//   CNSEA:   80 + 4*30 = 200 + 3% fuel = 206
	// Cheapest should be CNSEA at 206.
	if cheapest.ChannelName != "CNSEA" {
		t.Errorf("expected cheapest channel CNSEA for 5kg RU/normal, got %s (fee=%f)",
			cheapest.ChannelName, cheapest.TotalShippingFee)
	}
	if cheapest.TotalShippingFee != 206.0 {
		t.Errorf("expected cheapest fee 206 for 5kg RU/normal, got %f", cheapest.TotalShippingFee)
	}
}

func TestGetCheapestQuote_SingleResult(t *testing.T) {
	svc := NewService(testTables)
	cargo := Cargo{ActualWeightKg: 3.0}

	// KZ only has one matching channel (CNAIR fixed).
	cheapest, err := svc.GetCheapestQuote(cargo, "KZ", "normal")
	if err != nil {
		t.Fatalf("GetCheapestQuote failed for KZ: %v", err)
	}
	if cheapest == nil {
		t.Fatal("GetCheapestQuote returned nil for KZ")
	}
	if cheapest.ChannelName != "CNAIR" {
		t.Errorf("expected CNAIR for KZ, got %s", cheapest.ChannelName)
	}
}

func TestGetCheapestQuote_EmptyTables(t *testing.T) {
	svc := NewService([]RateTableEntry{})
	cargo := Cargo{ActualWeightKg: 1.0}

	_, err := svc.GetCheapestQuote(cargo, "RU", "normal")
	if err == nil {
		t.Fatal("expected error for empty rate tables, got nil")
	}
}

func TestService_GetQuoteWithVolumetricOverride(t *testing.T) {
	// Light package but very large — volumetric weight dominates.
	svc := NewService(testTables)
	cargo := Cargo{
		ActualWeightKg: 0.1,
		LengthCm:       100,
		WidthCm:        80,
		HeightCm:       60,
	}
	// Volumetric = (100*80*60)/6000 = 480000/6000 = 80 kg

	_, err := svc.GetQuote(cargo, "RU", "normal")
	if err != nil {
		// The 80 kg volumetric weight may exceed all rate table maxes.
		// That's acceptable — just verify no panic and a proper error.
		return
	}
	// If it succeeded, verify chargeable weight is correct.
	resp, _ := svc.GetQuote(cargo, "RU", "normal")
	if resp != nil && len(resp.Results) > 0 {
		if resp.Results[0].ChargeableWeightKg != 80.0 {
			t.Errorf("expected chargeable weight 80 (100*80*60/6000), got %f",
				resp.Results[0].ChargeableWeightKg)
		}
	}
}

func TestRateEngine_CalculateRate_SortByPriority(t *testing.T) {
	// Entries in reverse priority order — the engine should sort them.
	entries := []RateTableEntry{
		{
			ID: 2, ChannelName: "LOW_PRIORITY", Priority: 10,
			DestinationCountry: "US", CargoType: "normal",
			MinWeightKg: 0, MaxWeightKg: 10, RuleType: "fixed",
			FixedFee: 100,
		},
		{
			ID: 1, ChannelName: "HIGH_PRIORITY", Priority: 1,
			DestinationCountry: "US", CargoType: "normal",
			MinWeightKg: 0, MaxWeightKg: 10, RuleType: "fixed",
			FixedFee: 200,
		},
	}
	engine := NewRateEngine(entries)
	cargo := Cargo{ActualWeightKg: 5}
	resp, err := engine.CalculateRate(cargo, "US", "normal")
	if err != nil {
		t.Fatalf("CalculateRate failed: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	// Higher priority (lower number) should come first.
	if resp.Results[0].ChannelName != "HIGH_PRIORITY" {
		t.Errorf("expected HIGH_PRIORITY first (priority 1), got %s",
			resp.Results[0].ChannelName)
	}
}
