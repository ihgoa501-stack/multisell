package logistics

import (
	"testing"
)

// ── First+Additional Pricing ──────────────────────────────────────────────

func TestFirstAdditional_UnderFirstKg(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "first_additional",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			FirstKg: 1, FirstPrice: 60, AdditionalKg: 0.5, AdditionalPrice: 30,
			FuelSurchargePct: 5, SurchargeFixed: 2, MinimumCharge: 60,
			Currency: "CNY",
		},
	})

	resp, err := engine.CalculateRate(Cargo{ActualWeightKg: 0.5}, "RU", "normal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	r := resp.Results[0]
	// base = firstPrice = 60, surcharge = 2, fuel = 60*5% = 3, total = 65
	if r.BaseShippingFee != 60 {
		t.Errorf("base fee: expected 60, got %.2f", r.BaseShippingFee)
	}
	if r.SurchargeFee != 2 {
		t.Errorf("surcharge: expected 2, got %.2f", r.SurchargeFee)
	}
	if r.FuelSurchargeFee != 3 {
		t.Errorf("fuel: expected 3, got %.2f", r.FuelSurchargeFee)
	}
	if r.TotalShippingFee != 65 {
		t.Errorf("total: expected 65, got %.2f", r.TotalShippingFee)
	}
	if r.Currency != "CNY" {
		t.Errorf("currency: expected CNY, got %s", r.Currency)
	}
}

func TestFirstAdditional_OverFirstKg(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "first_additional",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			FirstKg: 1, FirstPrice: 60, AdditionalKg: 0.5, AdditionalPrice: 30,
			FuelSurchargePct: 5, SurchargeFixed: 2, MinimumCharge: 0,
			Currency: "CNY",
		},
	})

	// 1.5kg: ceil((1.5-1)/0.5) = 1 additional unit → 60 + 1*30 = 90
	resp, err := engine.CalculateRate(Cargo{ActualWeightKg: 1.5}, "RU", "normal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	r := resp.Results[0]
	if r.BaseShippingFee != 90 {
		t.Errorf("base fee: expected 90, got %.2f", r.BaseShippingFee)
	}
	if r.ChargeableWeightKg != 1.5 {
		t.Errorf("chargeable weight: expected 1.5, got %.4f", r.ChargeableWeightKg)
	}
}

func TestFirstAdditional_ExactAdditionalBracket(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "first_additional",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			FirstKg: 1, FirstPrice: 60, AdditionalKg: 0.5, AdditionalPrice: 30,
		},
	})

	// 2.0kg: ceil((2.0-1)/0.5) = 2 additional units → 60 + 2*30 = 120
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 2.0}, "RU", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 120 {
		t.Errorf("base fee: expected 120, got %.2f", r.BaseShippingFee)
	}
}

// ── Tiered Pricing ────────────────────────────────────────────────────────

func TestTieredPricing_MatchingTier(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "tiered",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			Tiers: []Tier{
				{Min: 2, Max: 3, Price: 140},
				{Min: 3, Max: 5, Price: 210},
				{Min: 5, Max: 0, Price: 340},
			},
			FuelSurchargePct: 5,
		},
	})

	// 4kg falls in [3,5) tier → 210, fuel = 210*5% = 10.5
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 4.0}, "RU", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 210 {
		t.Errorf("base fee: expected 210, got %.2f", r.BaseShippingFee)
	}
	if r.FuelSurchargeFee != 10.5 {
		t.Errorf("fuel: expected 10.5, got %.2f", r.FuelSurchargeFee)
	}
}

func TestTieredPricing_UnboundedLastTier(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "tiered",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			Tiers: []Tier{
				{Min: 2, Max: 3, Price: 140},
				{Min: 3, Max: 5, Price: 210},
				{Min: 5, Max: 0, Price: 340},
			},
		},
	})

	// 10kg → last tier (max=0 = unbounded) → 340
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 10}, "RU", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 340 {
		t.Errorf("base fee: expected 340, got %.2f", r.BaseShippingFee)
	}
}

// ── Fixed Fee ─────────────────────────────────────────────────────────────

func TestFixedFee(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "FlatRate", ProviderName: "Test", RuleType: "fixed",
			Priority: 1, DestinationCountry: "RU", CargoType: "battery",
			FixedFee: 150, Currency: "CNY",
		},
	})

	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 3.0}, "RU", "battery")
	r := resp.Results[0]
	if r.BaseShippingFee != 150 {
		t.Errorf("base fee: expected 150, got %.2f", r.BaseShippingFee)
	}
	if r.TotalShippingFee != 150 {
		t.Errorf("total: expected 150, got %.2f", r.TotalShippingFee)
	}
}

// ── Per-Kg Pricing ────────────────────────────────────────────────────────

func TestPerKgPricing(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "PerKg", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 80, SurchargeFixed: 5, Currency: "CNY",
		},
	})

	// 2kg: base=160, surcharge=5, total=165
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 2.0}, "RU", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 160 {
		t.Errorf("base fee: expected 160, got %.2f", r.BaseShippingFee)
	}
	if r.SurchargeFee != 5 {
		t.Errorf("surcharge: expected 5, got %.2f", r.SurchargeFee)
	}
	if r.TotalShippingFee != 165 {
		t.Errorf("total: expected 165, got %.2f", r.TotalShippingFee)
	}
}

// ── Fuel Surcharge ────────────────────────────────────────────────────────

func TestFuelSurcharge(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "FuelTest", ProviderName: "Test", RuleType: "first_additional",
			Priority: 1, DestinationCountry: "KZ", CargoType: "normal",
			FirstKg: 1, FirstPrice: 70, AdditionalKg: 1, AdditionalPrice: 50,
			FuelSurchargePct: 10,
		},
	})

	// 0.5kg within first kg → base=70, fuel=70*10%=7, total=77
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 0.5}, "KZ", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 70 {
		t.Errorf("base fee: expected 70, got %.2f", r.BaseShippingFee)
	}
	if r.FuelSurchargeFee != 7 {
		t.Errorf("fuel surcharge: expected 7, got %.2f", r.FuelSurchargeFee)
	}
	if r.TotalShippingFee != 77 {
		t.Errorf("total: expected 77, got %.2f", r.TotalShippingFee)
	}
}

// ── Volumetric Weight ─────────────────────────────────────────────────────

func TestVolumetricWeight_Dominates(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 50,
		},
	})

	// 30x40x50cm = 60000/6000 = 10kg volumetric
	// actual = 5kg → chargeable = max(5,10) = 10kg
	// fee = 10 * 50 = 500
	resp, _ := engine.CalculateRate(Cargo{
		ActualWeightKg: 5,
		LengthCm:       30, WidthCm: 40, HeightCm: 50,
	}, "RU", "normal")
	r := resp.Results[0]
	if r.ChargeableWeightKg != 10 {
		t.Errorf("chargeable weight: expected 10, got %.4f", r.ChargeableWeightKg)
	}
	if r.BaseShippingFee != 500 {
		t.Errorf("base fee: expected 500, got %.2f", r.BaseShippingFee)
	}
}

func TestNoDimensions_ActualWeightUsed(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 50,
		},
	})

	// No dimensions → volumetric = 0 → chargeable = actual = 5
	resp, _ := engine.CalculateRate(Cargo{
		ActualWeightKg: 5,
		// LengthCm=0, WidthCm=0, HeightCm=0
	}, "RU", "normal")
	r := resp.Results[0]
	if r.ChargeableWeightKg != 5 {
		t.Errorf("chargeable weight: expected 5, got %.4f", r.ChargeableWeightKg)
	}
}

// ── No Matching Rule ──────────────────────────────────────────────────────

func TestNoMatchingRule_EmptyResults(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 50,
		},
	})

	// Destination "US" not in table
	resp, err := engine.CalculateRate(Cargo{ActualWeightKg: 1}, "US", "normal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

// ── Minimum Charge ────────────────────────────────────────────────────────

func TestMinimumCharge_Enforced(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 65, MinimumCharge: 100,
			FuelSurchargePct: 0, SurchargeFixed: 0,
		},
	})

	// 1kg: base=65, fuel=0, total=65 → capped at minimum 100
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 1}, "RU", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 65 {
		t.Errorf("base fee: expected 65, got %.2f (should NOT be affected by min charge)", r.BaseShippingFee)
	}
	if r.TotalShippingFee != 100 {
		t.Errorf("total fee: expected 100 (minimum charge), got %.2f", r.TotalShippingFee)
	}
}

func TestMinimumCharge_NotAppliedAboveThreshold(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 65, MinimumCharge: 100,
			FuelSurchargePct: 0, SurchargeFixed: 0,
		},
	})

	// 2kg: base=130, total=130 > min=100 → no minimum applied
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 2}, "RU", "normal")
	r := resp.Results[0]
	if r.TotalShippingFee != 130 {
		t.Errorf("total fee: expected 130, got %.2f", r.TotalShippingFee)
	}
}

// ── Surcharge Calculations ────────────────────────────────────────────────

func TestSurchargeCalculations(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 50, FuelSurchargePct: 8, SurchargeFixed: 10,
		},
	})

	// base=50, surcharge=10, fuel=50*8%=4, total=64
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 1}, "RU", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 50 {
		t.Errorf("base fee: expected 50, got %.2f", r.BaseShippingFee)
	}
	if r.SurchargeFee != 10 {
		t.Errorf("surcharge fixed: expected 10, got %.2f", r.SurchargeFee)
	}
	if r.FuelSurchargeFee != 4 {
		t.Errorf("fuel surcharge: expected 4, got %.2f", r.FuelSurchargeFee)
	}
	if r.TotalShippingFee != 64 {
		t.Errorf("total: expected 64, got %.2f", r.TotalShippingFee)
	}
}

// ── Cargo Type Filtering ──────────────────────────────────────────────────

func TestCargoType_Filtering(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "NormalOnly", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 50, Currency: "CNY",
		},
		{
			ChannelName: "BatteryOnly", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "battery",
			PerKgPrice: 80, Currency: "CNY",
		},
	})

	// battery cargo → only battery entry matches
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 1}, "RU", "battery")
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result for battery cargo, got %d", len(resp.Results))
	}
	if resp.Results[0].ChannelName != "BatteryOnly" {
		t.Errorf("expected BatteryOnly channel, got %s", resp.Results[0].ChannelName)
	}
}

// ── Weight Range Exclusion ────────────────────────────────────────────────

func TestWeightRange_Exclusion(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "LightOnly", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			MinWeightKg: 0, MaxWeightKg: 2, PerKgPrice: 50,
		},
		{
			ChannelName: "HeavyOnly", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			MinWeightKg: 2, MaxWeightKg: 10, PerKgPrice: 40,
		},
	})

	// 5kg → LightOnly excluded (max=2), only HeavyOnly matches
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 5}, "RU", "normal")
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result for 5kg cargo, got %d", len(resp.Results))
	}
	if resp.Results[0].ChannelName != "HeavyOnly" {
		t.Errorf("expected HeavyOnly channel, got %s", resp.Results[0].ChannelName)
	}

	// 1.5kg → LightOnly matches, HeavyOnly excluded (min=2)
	resp2, _ := engine.CalculateRate(Cargo{ActualWeightKg: 1.5}, "RU", "normal")
	if len(resp2.Results) != 1 {
		t.Fatalf("expected 1 result for 1.5kg cargo, got %d", len(resp2.Results))
	}
	if resp2.Results[0].ChannelName != "LightOnly" {
		t.Errorf("expected LightOnly channel, got %s", resp2.Results[0].ChannelName)
	}
}

// ── Multiple Results (priority ordering) ──────────────────────────────────

func TestPriorityOrdering_MultipleMatches(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "LowPriority", ProviderName: "Test", RuleType: "per_kg",
			Priority: 10, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 30, MinWeightKg: 0, MaxWeightKg: 10,
		},
		{
			ChannelName: "HighPriority", ProviderName: "Test", RuleType: "first_additional",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			FirstKg: 5, FirstPrice: 100, AdditionalKg: 1, AdditionalPrice: 25,
			MinWeightKg: 0, MaxWeightKg: 10,
		},
	})

	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 3}, "RU", "normal")
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	// Higher priority (lower number) should come first
	if resp.Results[0].ChannelName != "HighPriority" {
		t.Errorf("expected first result to be HighPriority, got %s", resp.Results[0].ChannelName)
	}
}

// ── Sample YAML Integration ──────────────────────────────────────────────

func TestSampleYAML_LoadAndQuote(t *testing.T) {
	entries, err := LoadRateTableFromYAML([]byte(SampleRateTableYAML))
	if err != nil {
		t.Fatalf("LoadRateTableFromYAML: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("loaded 0 entries")
	}

	engine := NewRateEngine(entries)

	// Yanwen to RU, 0.5kg, normal → entry 1 (first_additional)
	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 0.5}, "RU", "normal")
	if len(resp.Results) == 0 {
		t.Fatal("expected at least 1 result for RU/normal/0.5kg")
	}
	// First result should be highest priority (entry 1, Yanwen RU first_additional)
	r := resp.Results[0]
	if r.ProviderName != "Yanwen" {
		t.Errorf("expected Yanwen, got %s", r.ProviderName)
	}
	if r.BaseShippingFee != 60 {
		t.Errorf("base fee: expected 60, got %.2f", r.BaseShippingFee)
	}

	// KZ destination → Yanwen per_kg and Cainiao first_additional
	resp2, _ := engine.CalculateRate(Cargo{ActualWeightKg: 0.5}, "KZ", "normal")
	if len(resp2.Results) == 0 {
		t.Fatal("expected results for KZ")
	}
}

// ── Empty Cargo (zero values) ─────────────────────────────────────────────

func TestEmptyCargo_ReturnsAvailableRates(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "fixed",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			FixedFee: 100,
		},
	})

	// All zeros → chargeable=0, weight range 0 → matches
	resp, _ := engine.CalculateRate(Cargo{}, "RU", "normal")
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result for empty cargo, got %d", len(resp.Results))
	}
}

// ── Entirely empty cargo AND unknown cargo type defaults ──────────────────

func TestDefaultCargoType(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Normal", ProviderName: "Test", RuleType: "fixed",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			FixedFee: 50,
		},
		{
			ChannelName: "BatteryOnly", ProviderName: "Test", RuleType: "fixed",
			Priority: 1, DestinationCountry: "RU", CargoType: "battery",
			FixedFee: 100,
		},
	})

	// Empty cargoType → defaults to "normal"
	resp, _ := engine.CalculateRate(Cargo{}, "RU", "")
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result with empty cargoType, got %d", len(resp.Results))
	}
	if resp.Results[0].ChannelName != "Normal" {
		t.Errorf("expected Normal channel, got %s", resp.Results[0].ChannelName)
	}
}
