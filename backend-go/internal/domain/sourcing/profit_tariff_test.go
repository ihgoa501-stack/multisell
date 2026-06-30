package sourcing

import (
	"errors"
	"testing"

	"github.com/lingmirror/backend-go/internal/domain/tariff"
)

// fakeTariffDecider is a test double for TariffDecider that returns a
// canned decision. Used to exercise ApplyTariff without a DB.
type fakeTariffDecider struct {
	result *tariff.DecisionResult
	err    error
	last   *tariff.DecisionRequest
}

func (f *fakeTariffDecider) Decide(req *tariff.DecisionRequest) (*tariff.DecisionResult, error) {
	f.last = req
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestApplyTariff_NilBase_NilSvc_ReturnsEmpty(t *testing.T) {
	// nil base + nil svc should not panic.
	out := ApplyTariff(nil, nil, &ApplyTariffInput{})
	if out == nil {
		t.Fatal("expected non-nil wrapper")
	}
	if out.ProfitBreakdown != nil {
		t.Errorf("expected nil base, got %v", out.ProfitBreakdown)
	}
}

func TestApplyTariff_NilTariffSvc_DegradesToDDU(t *testing.T) {
	base := &ProfitBreakdown{
		SourcePriceCNY: 100,
		TargetPriceCNY: 400,
		ProfitCNY:      200,
		MarginPct:      50,
		Destination:    "US",
	}
	out := ApplyTariff(base, nil, &ApplyTariffInput{ProfitInput: &ProfitInput{Destination: "US"}})
	if out.DDP {
		t.Errorf("expected DDP=false when tariffSvc is nil")
	}
	if out.TariffCostCNY != 0 {
		t.Errorf("expected zero tariff cost, got %v", out.TariffCostCNY)
	}
}

func TestApplyTariff_DecideError_DegradesToDDU(t *testing.T) {
	base := &ProfitBreakdown{
		SourcePriceCNY: 100,
		TargetPriceCNY: 400,
		ProfitCNY:      200,
		MarginPct:      50,
		Destination:    "US",
	}
	svc := &fakeTariffDecider{err: errors.New("db down")}
	out := ApplyTariff(base, svc, &ApplyTariffInput{ProfitInput: &ProfitInput{Destination: "US"}})
	if out.DDP {
		t.Errorf("expected DDP=false on Decide error")
	}
	if out.TariffCostCNY != 0 {
		t.Errorf("expected zero tariff cost on error, got %v", out.TariffCostCNY)
	}
	if out.IncotermReason == "" {
		t.Errorf("expected non-empty incoterm reason on error")
	}
}

func TestApplyTariff_DDP_ReducesProfit(t *testing.T) {
	// Source price = 100 CNY, rate = 7.2 → productValueUSD ≈ 13.89
	// Rule: 5% duty + 20% VAT (on value+duty) → small amount, ratio < 1% → DDP
	// dutyUSD = 13.89 * 5% = 0.6944
	// vatUSD = (13.89 + 0.6944) * 20% = 2.9169
	// totalDutyTaxUSD ≈ 3.61 → tariffCostCNY ≈ 3.61 * 7.2 = 26.0
	base := &ProfitBreakdown{
		SourcePriceCNY: 100,
		TargetPriceCNY: 400,
		ProfitCNY:      200,
		MarginPct:      50,
		Destination:    "JP",
	}
	svc := &fakeTariffDecider{result: &tariff.DecisionResult{
		Incoterm:        "DDP",
		TotalDutyTaxUSD: 3.6113,
		IncotermReason:  "Duty/tax is negligible (under 1% of value); seller can absorb",
		RulesMatched:    []tariff.RuleMatchItem{{RuleID: 7}},
	}}
	out := ApplyTariff(base, svc, &ApplyTariffInput{
		ProfitInput: &ProfitInput{Destination: "JP"},
		HSCode:      "847130",
		Quantity:    1,
	})

	if !out.DDP {
		t.Errorf("expected DDP=true, got false (reason: %s)", out.IncotermReason)
	}
	if out.TariffRuleID != 7 {
		t.Errorf("expected rule id 7, got %d", out.TariffRuleID)
	}
	// TariffCostCNY = 3.6113 * 7.2 ≈ 26.0
	if out.TariffCostCNY < 25.9 || out.TariffCostCNY > 26.1 {
		t.Errorf("expected TariffCostCNY ≈ 26.0, got %v", out.TariffCostCNY)
	}
	// AfterTariffProfitCNY = 200 - 26 = 174
	if out.AfterTariffProfitCNY != 174 {
		t.Errorf("expected AfterTariffProfitCNY 174, got %v", out.AfterTariffProfitCNY)
	}
	// AfterTariffMarginPct = 174 / 400 * 100 = 43.5
	if out.AfterTariffMarginPct != 43.5 {
		t.Errorf("expected AfterTariffMarginPct 43.5, got %v", out.AfterTariffMarginPct)
	}

	// Verify the tariff Decide was called with the converted USD value.
	if svc.last == nil {
		t.Fatal("expected Decide to be called")
	}
	if svc.last.DestinationCountry != "JP" {
		t.Errorf("expected destination JP, got %s", svc.last.DestinationCountry)
	}
	if svc.last.HSCode != "847130" {
		t.Errorf("expected HSCode 847130, got %s", svc.last.HSCode)
	}
	// 100 CNY / 7.2 ≈ 13.89 USD
	if svc.last.ProductValueUSD < 13.8 || svc.last.ProductValueUSD > 14.0 {
		t.Errorf("expected ProductValueUSD ≈ 13.89, got %v", svc.last.ProductValueUSD)
	}
}

func TestApplyTariff_DDU_KeepsProfit(t *testing.T) {
	// Under DDU, tariff cost is borne by the buyer — seller profit unchanged.
	base := &ProfitBreakdown{
		SourcePriceCNY: 100,
		TargetPriceCNY: 400,
		ProfitCNY:      200,
		MarginPct:      50,
		Destination:    "IN",
	}
	svc := &fakeTariffDecider{result: &tariff.DecisionResult{
		Incoterm:        "DDU",
		TotalDutyTaxUSD: 50.0,
		IncotermReason:  "Duty/tax exceeds 20% of product value; buyer should bear cost",
		RulesMatched:    []tariff.RuleMatchItem{{RuleID: 9}},
	}}
	out := ApplyTariff(base, svc, &ApplyTariffInput{
		ProfitInput: &ProfitInput{Destination: "IN"},
		Quantity:    2,
	})

	if out.DDP {
		t.Errorf("expected DDU (DDP=false)")
	}
	// Tariff cost still computed (for visibility): 50 * 7.2 = 360
	if out.TariffCostCNY != 360 {
		t.Errorf("expected TariffCostCNY 360, got %v", out.TariffCostCNY)
	}
	// But seller profit unchanged.
	if out.AfterTariffProfitCNY != 200 {
		t.Errorf("expected AfterTariffProfitCNY 200 (unchanged), got %v", out.AfterTariffProfitCNY)
	}
	if out.AfterTariffMarginPct != 50 {
		t.Errorf("expected AfterTariffMarginPct 50 (unchanged), got %v", out.AfterTariffMarginPct)
	}
}

func TestApplyTariff_NoRuleMatched_DDU(t *testing.T) {
	base := &ProfitBreakdown{
		SourcePriceCNY: 100,
		TargetPriceCNY: 400,
		ProfitCNY:      200,
		MarginPct:      50,
		Destination:    "RU",
	}
	svc := &fakeTariffDecider{result: &tariff.DecisionResult{
		Incoterm:        "DDU",
		TotalDutyTaxUSD: 0,
		IncotermReason:  "No applicable tariff rule found; defaulting to DDU",
		RulesMatched:    nil,
	}}
	out := ApplyTariff(base, svc, &ApplyTariffInput{ProfitInput: &ProfitInput{Destination: "RU"}})
	if out.DDP {
		t.Errorf("expected DDU when no rule matched")
	}
	if out.TariffRuleID != 0 {
		t.Errorf("expected rule id 0, got %d", out.TariffRuleID)
	}
	if out.TariffCostCNY != 0 {
		t.Errorf("expected zero tariff cost, got %v", out.TariffCostCNY)
	}
}

func TestApplyTariff_CustomExchangeRate(t *testing.T) {
	// With USDCNYRate = 10, source price 100 CNY → 10 USD
	// duty 5% = 0.5 USD → tariffCostCNY = 0.5 * 10 = 5
	base := &ProfitBreakdown{
		SourcePriceCNY: 100,
		TargetPriceCNY: 400,
		ProfitCNY:      200,
		MarginPct:      50,
		Destination:    "JP",
	}
	svc := &fakeTariffDecider{result: &tariff.DecisionResult{
		Incoterm:        "DDP",
		TotalDutyTaxUSD: 0.5,
		RulesMatched:    []tariff.RuleMatchItem{{RuleID: 1}},
	}}
	out := ApplyTariff(base, svc, &ApplyTariffInput{
		ProfitInput: &ProfitInput{Destination: "JP"},
		USDCNYRate:  10.0,
	})
	if out.TariffCostCNY != 5 {
		t.Errorf("expected TariffCostCNY 5 with rate 10, got %v", out.TariffCostCNY)
	}
	// Verify the tariff Decide was called with USD value computed at rate 10.
	if svc.last.ProductValueUSD != 10 {
		t.Errorf("expected ProductValueUSD 10 (100/10), got %v", svc.last.ProductValueUSD)
	}
}
