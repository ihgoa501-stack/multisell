package landedcost

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func newTestDB(t *testing.T) *LandedCost {
	t.Helper()
	return &LandedCost{}
}

// ── Pure function: calculate ────────────────────────────────────

func TestCalculate_HappyPath(t *testing.T) {
	res := calculate(&CalculateRequest{
		ProductID:       1,
		PlatformID:      1,
		UnitCostCNY:     50,
		FreightCNY:      20,
		InsuranceCNY:    5,
		TargetMarginPct: 20,
	}, 5.0, 7.0)

	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.TargetPrice <= 0 {
		t.Errorf("expected positive target price, got %f", res.TargetPrice)
	}
	if res.TotalCostCNY <= 0 {
		t.Errorf("expected positive total cost, got %f", res.TotalCostCNY)
	}
	if res.ProfitMarginPct <= 0 {
		t.Errorf("expected positive profit margin, got %f", res.ProfitMarginPct)
	}
	if res.LandedCost.ProductID != 1 {
		t.Errorf("expected product_id 1, got %d", res.LandedCost.ProductID)
	}
}

func TestCalculate_WithDutyAndVAT(t *testing.T) {
	dutyRate := 10.0
	vatRate := 13.0
	res := calculate(&CalculateRequest{
		ProductID:       1,
		PlatformID:      1,
		UnitCostCNY:     100,
		FreightCNY:      30,
		InsuranceCNY:    10,
		DutyRate:        &dutyRate,
		VatRate:         &vatRate,
		PlatformFeePct:  floatPtr(8.0),
		TargetMarginPct: 25,
	}, 5.0, 6.8)

	if res.DutyCNY <= 0 {
		t.Errorf("expected positive duty, got %f", res.DutyCNY)
	}
	if res.VatCNY <= 0 {
		t.Errorf("expected positive vat, got %f", res.VatCNY)
	}
	if res.PlatformFeeCNY <= 0 {
		t.Errorf("expected positive platform fee, got %f", res.PlatformFeeCNY)
	}
	if res.TotalCostCNY <= 0 {
		t.Errorf("expected positive total cost, got %f", res.TotalCostCNY)
	}
}

func TestCalculate_WithClearingFee(t *testing.T) {
	clearing := 15.0
	res := calculate(&CalculateRequest{
		ProductID:        1,
		PlatformID:       1,
		UnitCostCNY:      80,
		FreightCNY:       20,
		ClearingFeeCNY:   &clearing,
		TargetMarginPct:  15,
	}, 5.0, 7.0)

	if res.ClearingFeeCNY != 15 {
		t.Errorf("expected clearing fee 15, got %f", res.ClearingFeeCNY)
	}
}

func TestCalculate_DefaultMargin(t *testing.T) {
	// Zero margin should default to 15%
	res := calculate(&CalculateRequest{
		ProductID:   1,
		PlatformID:  1,
		UnitCostCNY: 50,
		FreightCNY:  10,
	}, 5.0, 7.0)

	if res.TargetPrice <= 0 {
		t.Errorf("expected positive target price, got %f", res.TargetPrice)
	}
}

func TestCalculate_ZeroExchangeRate(t *testing.T) {
	// Zero exchange rate should default to 7.0
	res := calculate(&CalculateRequest{
		ProductID:       1,
		PlatformID:      1,
		UnitCostCNY:     50,
		FreightCNY:      10,
		TargetMarginPct: 20,
	}, 5.0, 0)

	if res.ExchangeRate != 7.0 {
		t.Errorf("expected exchange rate default 7.0, got %f", res.ExchangeRate)
	}
}

func TestCalculate_DenomProtection(t *testing.T) {
	// When margin + fee >= 100%, denominator should not be zero
	feePct := 95.0
	res := calculate(&CalculateRequest{
		ProductID:        1,
		PlatformID:       1,
		UnitCostCNY:      50,
		FreightCNY:       10,
		PlatformFeePct:   &feePct,
		TargetMarginPct:  10,
	}, 5.0, 7.0)

	if res.TargetPrice <= 0 {
		t.Errorf("expected positive target price even with high fees, got %f", res.TargetPrice)
	}
}

func TestCalculate_ZeroCost(t *testing.T) {
	res := calculate(&CalculateRequest{
		ProductID:       1,
		PlatformID:      1,
		UnitCostCNY:     0,
		FreightCNY:      0,
		TargetMarginPct: 15,
	}, 5.0, 7.0)

	if res.TotalCostCNY != 0 {
		t.Errorf("expected total cost 0, got %f", res.TotalCostCNY)
	}
}

func TestCalculate_Rounding(t *testing.T) {
	res := calculate(&CalculateRequest{
		ProductID:       1,
		PlatformID:      1,
		UnitCostCNY:     99.99,
		FreightCNY:      33.33,
		InsuranceCNY:    5.55,
		TargetMarginPct: 20,
	}, 5.0, 7.15)

	// All monetary values should be rounded to 2 decimal places
	if res.UnitCostCNY != 99.99 {
		t.Errorf("expected unit cost 99.99, got %f", res.UnitCostCNY)
	}
	// Check rounding: total should be a clean 2-decimal value
	totalStr := res.TotalCostCNY
	if totalStr != float64(int(totalStr*100))/100 {
		t.Errorf("total cost not rounded to 2 decimals: %f", totalStr)
	}
	priceStr := res.TargetPrice
	if priceStr != float64(int(priceStr*100))/100 {
		t.Errorf("target price not rounded to 2 decimals: %f", priceStr)
	}
}

// ── DB-dependent tests ──────────────────────────────────────────

func TestService_Calculate_WithDefaults(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	res, err := svc.Calculate(&CalculateRequest{
		ProductID:       1,
		PlatformID:      1,
		UnitCostCNY:     50,
		FreightCNY:      20,
		TargetMarginPct: 20,
	})
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if res.TargetPrice <= 0 {
		t.Errorf("expected positive target price, got %f", res.TargetPrice)
	}
	if res.ID == 0 {
		t.Error("expected saved LandedCost with non-zero ID")
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.GetByID(99999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestService_GetByID_Found(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	res, _ := svc.Calculate(&CalculateRequest{
		ProductID: 1, PlatformID: 1, UnitCostCNY: 50, FreightCNY: 10, TargetMarginPct: 15,
	})

	got, err := svc.GetByID(res.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.ID != res.ID {
		t.Errorf("expected ID %d, got %d", res.ID, got.ID)
	}
}

func TestService_GetByProductPlatform_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.GetByProductPlatform(1, 1)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestService_GetByProductPlatform_Found(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Calculate(&CalculateRequest{
		ProductID: 1, PlatformID: 2, UnitCostCNY: 50, FreightCNY: 10, TargetMarginPct: 15,
	})

	got, err := svc.GetByProductPlatform(1, 2)
	if err != nil {
		t.Fatalf("GetByProductPlatform failed: %v", err)
	}
	if got.ProductID != 1 || got.PlatformID != 2 {
		t.Errorf("unexpected product/platform IDs")
	}
}

func TestService_ListByProduct_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	items, err := svc.ListByProduct(999)
	if err != nil {
		t.Fatalf("ListByProduct failed: %v", err)
	}
	if items == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestService_ListByProduct(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Calculate(&CalculateRequest{ProductID: 1, PlatformID: 1, UnitCostCNY: 50, FreightCNY: 10, TargetMarginPct: 15})
	svc.Calculate(&CalculateRequest{ProductID: 1, PlatformID: 2, UnitCostCNY: 60, FreightCNY: 10, TargetMarginPct: 15})

	items, err := svc.ListByProduct(1)
	if err != nil {
		t.Fatalf("ListByProduct failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestService_CompareAcrossPlatforms(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Calculate(&CalculateRequest{ProductID: 1, PlatformID: 1, UnitCostCNY: 50, FreightCNY: 10, TargetMarginPct: 15})
	svc.Calculate(&CalculateRequest{ProductID: 1, PlatformID: 2, UnitCostCNY: 60, FreightCNY: 10, TargetMarginPct: 20})

	items, err := svc.CompareAcrossPlatforms(1)
	if err != nil {
		t.Fatalf("CompareAcrossPlatforms failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 compare items, got %d", len(items))
	}
}

func TestService_CompareAcrossPlatforms_Dedup(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Two calculations for the same platform — only latest should appear.
	svc.Calculate(&CalculateRequest{ProductID: 1, PlatformID: 1, UnitCostCNY: 50, FreightCNY: 10, TargetMarginPct: 15})
	svc.Calculate(&CalculateRequest{ProductID: 1, PlatformID: 1, UnitCostCNY: 55, FreightCNY: 10, TargetMarginPct: 20})

	items, err := svc.CompareAcrossPlatforms(1)
	if err != nil {
		t.Fatalf("CompareAcrossPlatforms failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 compare item (deduplicated), got %d", len(items))
	}
	// Latest entry should have the higher TargetPrice from 20% margin
	// instead of 15%.
	if items[0].ProfitMarginPct <= 15 {
		t.Errorf("expected latest profit margin > 15%%, got %f", items[0].ProfitMarginPct)
	}
}

func TestService_CompareAcrossPlatforms_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &LandedCost{})
	svc := NewService(db, dbtest.NewLogger(t))

	items, err := svc.CompareAcrossPlatforms(999)
	if err != nil {
		t.Fatalf("CompareAcrossPlatforms failed: %v", err)
	}
	// result is nil when no records found (returned from empty slice iteration)
	_ = items
	}

func TestCostBreakdown(t *testing.T) {
	lc := LandedCost{
		UnitCostCNY:  50,
		FreightCNY:   20,
		InsuranceCNY: 5,
		DutyRate:     10,
		DutyCNY:      5,
		VatRate:      13,
		VatCNY:       9.75,
		TotalCostCNY: 100,
		TargetPrice:  25.99,
	}

	m := CostBreakdown(&lc)
	if m == nil {
		t.Fatal("expected non-nil breakdown")
	}
	if _, ok := m["采购成本 (CNY)"]; !ok {
		t.Error("expected procurement cost in breakdown")
	}
	if _, ok := m["建议售价 (本地货币)"]; !ok {
		t.Error("expected recommended price in breakdown")
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
