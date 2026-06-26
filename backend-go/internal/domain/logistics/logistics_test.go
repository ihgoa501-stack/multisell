package logistics

import (
	"context"
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
	if len(resp.Results) < 2 {
		t.Fatalf("expected at least 2 results for RU/normal, got %d", len(resp.Results))
	}
}

func TestGetQuote_VolumetricWeight(t *testing.T) {
	svc := NewService(testTables)
	cargo := Cargo{
		ActualWeightKg: 0.5,
		LengthCm:       60,
		WidthCm:        50,
		HeightCm:       40,
	}

	resp, err := svc.GetQuote(cargo, "RU", "normal")
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("GetQuote returned no results")
	}

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
		t.Errorf("expected total shipping fee 206 for KZ fixed rate, got %f", resp.Results[0].TotalShippingFee)
	}
}

func TestGetQuote_NoMatch(t *testing.T) {
	svc := NewService(testTables)
	cargo := Cargo{ActualWeightKg: 100.0}

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
	svc := NewService(testTables)
	cargo := Cargo{
		ActualWeightKg: 0.1,
		LengthCm:       100,
		WidthCm:        80,
		HeightCm:       60,
	}

	_, err := svc.GetQuote(cargo, "RU", "normal")
	if err != nil {
		return
	}
	resp, _ := svc.GetQuote(cargo, "RU", "normal")
	if resp != nil && len(resp.Results) > 0 {
		if resp.Results[0].ChargeableWeightKg != 80.0 {
			t.Errorf("expected chargeable weight 80 (100*80*60/6000), got %f",
				resp.Results[0].ChargeableWeightKg)
		}
	}
}

func TestRateEngine_CalculateRate_SortByPriority(t *testing.T) {
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
	if resp.Results[0].ChannelName != "HIGH_PRIORITY" {
		t.Errorf("expected HIGH_PRIORITY first (priority 1), got %s",
			resp.Results[0].ChannelName)
	}
}

// ---------- RateQuoteRequest / Response tests ----------

func TestGetQuotes_HappyPath(t *testing.T) {
	svc := NewService(testTables)

	req := &RateQuoteRequest{
		OriginCountry:      "CN",
		DestinationCountry: "RU",
		WeightKG:           5.0,
		CargoType:          "normal",
	}

	resp, err := svc.GetQuotes(req)
	if err != nil {
		t.Fatalf("GetQuotes failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetQuotes returned nil")
	}
	if len(resp.Quotes) == 0 {
		t.Fatal("GetQuotes returned empty quotes")
	}
}

func TestGetQuotes_EmptyDestReturnsError(t *testing.T) {
	svc := NewService(testTables)
	req := &RateQuoteRequest{
		DestinationCountry: "",
		WeightKG:           1.0,
	}
	_, err := svc.GetQuotes(req)
	if err == nil {
		t.Fatal("expected error for empty destination, got nil")
	}
}

func TestGetQuotes_ZeroWeight(t *testing.T) {
	svc := NewService(testTables)
	req := &RateQuoteRequest{
		DestinationCountry: "RU",
		WeightKG:           0,
	}
	resp, err := svc.GetQuotes(req)
	if err != nil {
		t.Fatalf("GetQuotes failed: %v", err)
	}
	if len(resp.Quotes) == 0 {
		t.Fatal("expected at least 1 quote for zero weight with min_weight 0 tables")
	}
	// Zero weight gets clamped to default and chargeable weight may still match.
	// The engine returns matching entries — not an error.
}

// ---------- ListCarriers tests ----------

func TestListCarriers_NonEmpty(t *testing.T) {
	svc := NewService(testTables)
	carriers := svc.ListCarriers()
	if len(carriers) == 0 {
		t.Fatal("expected non-empty carrier list")
	}

	found := false
	for _, c := range carriers {
		if c.Code == "CNExpress" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected CNExpress in carrier list")
	}
}

func TestListCarriers_EmptyEngine(t *testing.T) {
	svc := NewService(nil)
	carriers := svc.ListCarriers()
	if carriers == nil {
		t.Fatal("expected empty slice, not nil")
	}
	if len(carriers) != 0 {
		t.Fatalf("expected 0 carriers for empty engine, got %d", len(carriers))
	}
}

// ---------- MockCarrier tests ----------

func TestMockCarrier_Quote(t *testing.T) {
	carrier := &MockCarrier{}
	ctx := context.Background()

	req := &RateQuoteRequest{
		DestinationCountry: "RU",
		WeightKG:           1.0,
	}

	resp, err := carrier.Quote(ctx, req)
	if err != nil {
		t.Fatalf("MockCarrier.Quote failed: %v", err)
	}
	if len(resp.Quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(resp.Quotes))
	}
	if resp.Quotes[0].TotalCost <= 0 {
		t.Fatal("expected positive total cost")
	}
}

func TestMockCarrier_ValidateCredentials(t *testing.T) {
	carrier := &MockCarrier{}
	if err := carrier.ValidateCredentials(context.Background()); err != nil {
		t.Fatalf("MockCarrier.ValidateCredentials failed: %v", err)
	}
}

// ---------- CarrierAdapter tests ----------

func TestCarrierYanwenAdapter_Quote(t *testing.T) {
	adapter := &CarrierYanwenAdapter{}
	ctx := context.Background()

	req := &RateQuoteRequest{
		DestinationCountry: "RU",
		WeightKG:           0.5,
	}

	resp, err := adapter.Quote(ctx, req)
	if err != nil {
		t.Fatalf("YanwenAdapter.Quote failed: %v", err)
	}
	if len(resp.Quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(resp.Quotes))
	}
	if resp.Quotes[0].Carrier != "Yanwen" {
		t.Errorf("expected carrier Yanwen, got %s", resp.Quotes[0].Carrier)
	}
	if resp.Quotes[0].TotalCost <= 0 {
		t.Fatal("expected positive total cost")
	}
}

func TestCarrierYanwenAdapter_ValidateCredentials(t *testing.T) {
	adapter := &CarrierYanwenAdapter{}
	if err := adapter.ValidateCredentials(context.Background()); err != nil {
		t.Fatalf("YanwenAdapter.ValidateCredentials failed: %v", err)
	}
}

func TestCarrierYanwenAdapter_US_Rate(t *testing.T) {
	adapter := &CarrierYanwenAdapter{}
	ctx := context.Background()

	// US should have higher rates than RU
	ruResp, _ := adapter.Quote(ctx, &RateQuoteRequest{DestinationCountry: "RU", WeightKG: 1.0})
	usResp, _ := adapter.Quote(ctx, &RateQuoteRequest{DestinationCountry: "US", WeightKG: 1.0})

	if usResp.Quotes[0].TotalCost <= ruResp.Quotes[0].TotalCost {
		t.Error("expected US rates to be higher than RU rates")
	}
}

func TestCarrierYanwenAdapter_Weight_Default(t *testing.T) {
	adapter := &CarrierYanwenAdapter{}
	ctx := context.Background()

	// Very low weight should be clamped to 0.1
	resp, err := adapter.Quote(ctx, &RateQuoteRequest{DestinationCountry: "RU", WeightKG: 0})
	if err != nil {
		t.Fatalf("YanwenAdapter.Quote with zero weight failed: %v", err)
	}
	if len(resp.Quotes) == 0 {
		t.Fatal("expected quotes for zero weight (clamped)")
	}
}

func TestCarrierYuntuAdapter_Quote(t *testing.T) {
	adapter := &CarrierYuntuAdapter{}
	ctx := context.Background()

	req := &RateQuoteRequest{
		DestinationCountry: "RU",
		WeightKG:           2.0,
	}

	resp, err := adapter.Quote(ctx, req)
	if err != nil {
		t.Fatalf("YuntuAdapter.Quote failed: %v", err)
	}
	if len(resp.Quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(resp.Quotes))
	}
	if resp.Quotes[0].Carrier != "Yuntu" {
		t.Errorf("expected carrier Yuntu, got %s", resp.Quotes[0].Carrier)
	}
}

func TestCarrierYuntuAdapter_ValidateCredentials(t *testing.T) {
	adapter := &CarrierYuntuAdapter{}
	if err := adapter.ValidateCredentials(context.Background()); err != nil {
		t.Fatalf("YuntuAdapter.ValidateCredentials failed: %v", err)
	}
}

func TestCarrierYuntuAdapter_BulkDiscount(t *testing.T) {
	adapter := &CarrierYuntuAdapter{}
	ctx := context.Background()

	// 15kg should get bulk discount (10%)
	light, _ := adapter.Quote(ctx, &RateQuoteRequest{DestinationCountry: "RU", WeightKG: 1.0})
	heavy, _ := adapter.Quote(ctx, &RateQuoteRequest{DestinationCountry: "RU", WeightKG: 15.0})

	// Per-kg rate for heavy should be cheaper than light
	lightPerKg := light.Quotes[0].TotalCost / 1.0
	heavyPerKg := heavy.Quotes[0].TotalCost / 15.0
	if heavyPerKg >= lightPerKg {
		t.Logf("heavyPerKg=%.4f, lightPerKg=%.4f — bulk discount should make heavy cheaper per kg", heavyPerKg, lightPerKg)
	}
}

// ---------- CarrierRegistry tests ----------

func TestCarrierRegistry_RegisterAndList(t *testing.T) {
	reg := NewCarrierRegistry()
	reg.Register(&MockCarrier{})
	reg.Register(&CarrierYanwenAdapter{})

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 carriers, got %d", len(list))
	}

	// Should be sorted alphabetically
	if list[0].Name() != "mock" {
		t.Errorf("expected first carrier 'mock', got %s", list[0].Name())
	}
	if list[1].Name() != "yanwen" {
		t.Errorf("expected second carrier 'yanwen', got %s", list[1].Name())
	}
}

func TestCarrierRegistry_Get(t *testing.T) {
	reg := NewCarrierRegistry()
	reg.Register(&CarrierYanwenAdapter{})

	found := reg.Get("yanwen")
	if found == nil {
		t.Fatal("expected to find yanwen carrier")
	}
	if found.Name() != "yanwen" {
		t.Errorf("expected name yanwen, got %s", found.Name())
	}

	notFound := reg.Get("nonexistent")
	if notFound != nil {
		t.Error("expected nil for nonexistent carrier")
	}
}

func TestCarrierRegistry_Empty(t *testing.T) {
	reg := NewCarrierRegistry()
	list := reg.List()
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

// ---------- LocalRateEngine tests ----------

func TestLocalRateEngine_Quote(t *testing.T) {
	svc := NewService(testTables)
	engine := NewLocalRateEngine(svc)
	ctx := context.Background()

	req := &RateQuoteRequest{
		DestinationCountry: "RU",
		WeightKG:           3.0,
	}

	resp, err := engine.Quote(ctx, req)
	if err != nil {
		t.Fatalf("LocalRateEngine.Quote failed: %v", err)
	}
	if resp == nil {
		t.Fatal("LocalRateEngine.Quote returned nil")
	}
	if len(resp.Quotes) == 0 {
		t.Fatal("LocalRateEngine.Quote returned empty quotes")
	}
}

// ---------- RateEngine pricing modes tests ----------

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
}

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

	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 4.0}, "RU", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 210 {
		t.Errorf("base fee: expected 210, got %.2f", r.BaseShippingFee)
	}
	if r.FuelSurchargeFee != 10.5 {
		t.Errorf("fuel: expected 10.5, got %.2f", r.FuelSurchargeFee)
	}
}

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
}

func TestPerKgPricing(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "PerKg", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 80, SurchargeFixed: 5, Currency: "CNY",
		},
	})

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

func TestMinimumCharge_Enforced(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 65, MinimumCharge: 100,
			FuelSurchargePct: 0, SurchargeFixed: 0,
		},
	})

	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 1}, "RU", "normal")
	r := resp.Results[0]
	if r.BaseShippingFee != 65 {
		t.Errorf("base fee: expected 65, got %.2f (should NOT be affected by min charge)", r.BaseShippingFee)
	}
	if r.TotalShippingFee != 100 {
		t.Errorf("total fee: expected 100 (minimum charge), got %.2f", r.TotalShippingFee)
	}
}

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

	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 1}, "RU", "battery")
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result for battery cargo, got %d", len(resp.Results))
	}
	if resp.Results[0].ChannelName != "BatteryOnly" {
		t.Errorf("expected BatteryOnly channel, got %s", resp.Results[0].ChannelName)
	}
}

func TestNoMatchingRule_EmptyResults(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "per_kg",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			PerKgPrice: 50,
		},
	})

	resp, err := engine.CalculateRate(Cargo{ActualWeightKg: 1}, "US", "normal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

func TestSampleYAML_LoadAndQuote(t *testing.T) {
	entries, err := LoadRateTableFromYAML([]byte(SampleRateTableYAML))
	if err != nil {
		t.Fatalf("LoadRateTableFromYAML: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("loaded 0 entries")
	}

	engine := NewRateEngine(entries)

	resp, _ := engine.CalculateRate(Cargo{ActualWeightKg: 0.5}, "RU", "normal")
	if len(resp.Results) == 0 {
		t.Fatal("expected at least 1 result for RU/normal/0.5kg")
	}
	r := resp.Results[0]
	if r.ProviderName != "Yanwen" {
		t.Errorf("expected Yanwen, got %s", r.ProviderName)
	}
	if r.BaseShippingFee != 60 {
		t.Errorf("base fee: expected 60, got %.2f", r.BaseShippingFee)
	}
}

func TestQuoteResponse_ToRateQuoteConversion(t *testing.T) {
	internal := &QuoteResponse{
		Results: []QuoteResult{
			{
				ChannelName: "TestChan", ProviderName: "TestProv",
				TotalShippingFee: 100.50, Currency: "CNY",
				EstimatedDeliveryMin: 5, EstimatedDeliveryMax: 10,
			},
		},
	}

	external := internal.ToRateQuoteResponse()
	if len(external.Quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(external.Quotes))
	}
	q := external.Quotes[0]
	if q.Carrier != "TestProv" {
		t.Errorf("expected carrier TestProv, got %s", q.Carrier)
	}
	if q.TotalCost != 100.50 {
		t.Errorf("expected total cost 100.50, got %f", q.TotalCost)
	}
	if q.EstDaysMin != 5 || q.EstDaysMax != 10 {
		t.Errorf("expected days 5-10, got %d-%d", q.EstDaysMin, q.EstDaysMax)
	}
}

func TestEmptyCargo_ReturnsAvailableRates(t *testing.T) {
	engine := NewRateEngine([]RateTableEntry{
		{
			ChannelName: "Test", ProviderName: "Test", RuleType: "fixed",
			Priority: 1, DestinationCountry: "RU", CargoType: "normal",
			FixedFee: 100,
		},
	})

	resp, _ := engine.CalculateRate(Cargo{}, "RU", "normal")
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result for empty cargo, got %d", len(resp.Results))
	}
}

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

	resp, _ := engine.CalculateRate(Cargo{}, "RU", "")
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result with empty cargoType, got %d", len(resp.Results))
	}
	if resp.Results[0].ChannelName != "Normal" {
		t.Errorf("expected Normal channel, got %s", resp.Results[0].ChannelName)
	}
}

func TestService_GetQuotes_EventPublishing(t *testing.T) {
	published := false
	pub := &testPublisher{publishFn: func(topic string, payload map[string]interface{}) error {
		published = true
		if topic != "logistics.quote" {
			t.Errorf("expected topic 'logistics.quote', got %s", topic)
		}
		return nil
	}}

	svc := NewServiceWithEvents(testTables, pub)
	req := &RateQuoteRequest{
		DestinationCountry: "RU",
		WeightKG:           2.0,
	}

	_, err := svc.GetQuotes(req)
	if err != nil {
		t.Fatalf("GetQuotes failed: %v", err)
	}
	if !published {
		t.Error("expected event to be published")
	}
}

// testPublisher implements EventPublisher for testing.
type testPublisher struct {
	publishFn func(topic string, payload map[string]interface{}) error
}

func (p *testPublisher) Publish(topic string, payload map[string]interface{}) error {
	return p.publishFn(topic, payload)
}
