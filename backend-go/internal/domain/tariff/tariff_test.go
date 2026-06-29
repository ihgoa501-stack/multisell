package tariff

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func float64Ptr(v float64) *float64 { return &v }
func intPtr(v int) *int             { return &v }
func stringPtr(v string) *string    { return &v }

func TestService_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	r, err := svc.Create(&CreateRuleInput{
		CountryCode: "US",
		DutyRatePct: float64Ptr(5.0),
		Incoterm:    "DDP",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("ID should be set")
	}
	if r.DutyRatePct != 5.0 {
		t.Fatalf("DutyRatePct = %v", r.DutyRatePct)
	}
	if r.Incoterm != "DDP" {
		t.Fatalf("Incoterm = %s", r.Incoterm)
	}

	got, err := svc.Get(r.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.CountryCode != "US" {
		t.Fatalf("CountryCode = %s", got.CountryCode)
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	r, _ := svc.Create(&CreateRuleInput{
		CountryCode: "DE",
		DutyRatePct: float64Ptr(3.0),
		Incoterm:    "DDU",
	})

	updated, err := svc.Update(r.ID, &UpdateRuleInput{
		DutyRatePct: float64Ptr(6.0),
		Status:      stringPtr("inactive"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DutyRatePct != 6.0 {
		t.Fatalf("DutyRatePct = %v", updated.DutyRatePct)
	}
	if updated.Status != "inactive" {
		t.Fatalf("Status = %s", updated.Status)
	}
}

func TestService_List_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{CountryCode: "US", DutyRatePct: float64Ptr(3.0)})
	svc.Create(&CreateRuleInput{CountryCode: "DE", DutyRatePct: float64Ptr(5.0)})
	svc.Create(&CreateRuleInput{CountryCode: "US", DutyRatePct: float64Ptr(8.0), Status: "inactive"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// Filter by country_code
	items, total, err = svc.List(&p, &RuleListFilter{CountryCode: "DE"})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 DE, got %d", total)
	}

	// Filter by status
	items, total, err = svc.List(&p, &RuleListFilter{Status: "inactive"})
	if err != nil {
		t.Fatalf("List filtered by status: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 inactive, got %d", total)
	}

	// Delete
	if err := svc.Delete(items[0].ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.Get(items[0].ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Decide_DDP_NegligibleDuty(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Country with 0.5% duty — negligible → DDP
	svc.Create(&CreateRuleInput{
		CountryCode: "JP",
		DutyRatePct: float64Ptr(0.5),
		VatRatePct:  float64Ptr(0),
		Incoterm:    "DDU",
		Priority:    intPtr(1),
	})

	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "JP",
		ProductValueUSD:    100,
		Quantity:           1,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if res.Incoterm != "DDP" {
		t.Fatalf("expected DDP (negligible duty), got %s (reason: %s)", res.Incoterm, res.IncotermReason)
	}
	if res.TotalDutyTaxUSD != 0.5 {
		t.Fatalf("TotalDutyTaxUSD = %v (expected 0.5)", res.TotalDutyTaxUSD)
	}
	if res.DutyAmountUSD != 0.5 {
		t.Fatalf("DutyAmountUSD = %v (expected 0.5)", res.DutyAmountUSD)
	}
}

func TestService_Decide_DDU_HighDuty(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Country with 25% duty — exceeds 20% threshold → DDU
	svc.Create(&CreateRuleInput{
		CountryCode: "IN",
		DutyRatePct: float64Ptr(25.0),
		VatRatePct:  float64Ptr(0),
		Incoterm:    "DDP",
		Priority:    intPtr(1),
	})

	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "IN",
		ProductValueUSD:    200,
		Quantity:           1,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if res.Incoterm != "DDU" {
		t.Fatalf("expected DDU (high duty >20%%), got %s (reason: %s)", res.Incoterm, res.IncotermReason)
	}
	if res.TotalDutyTaxUSD != 50.0 {
		t.Fatalf("TotalDutyTaxUSD = %v (expected 50)", res.TotalDutyTaxUSD)
	}
}

func TestService_Decide_DDP_RulePreference(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Country with 10% duty — mid-range, respects DDP rule preference
	svc.Create(&CreateRuleInput{
		CountryCode: "UK",
		DutyRatePct: float64Ptr(10.0),
		VatRatePct:  float64Ptr(0),
		Incoterm:    "DDP",
		Priority:    intPtr(1),
	})

	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "UK",
		ProductValueUSD:    500,
		Quantity:           1,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	// 10% of 500 = 50 duty. Ratio = 0.10. Between 1% and 20% → respect rule preference (DDP)
	if res.Incoterm != "DDP" {
		t.Fatalf("expected DDP (rule preference), got %s (reason: %s)", res.Incoterm, res.IncotermReason)
	}
	if res.TotalDutyTaxUSD != 50.0 {
		t.Fatalf("TotalDutyTaxUSD = %v (expected 50)", res.TotalDutyTaxUSD)
	}
}

func TestService_Decide_DDU_NoRules(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// No rules for this country
	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "RU",
		ProductValueUSD:    100,
		Quantity:           1,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if res.Incoterm != "DDU" {
		t.Fatalf("expected DDU (no rules), got %s", res.Incoterm)
	}
	if res.TotalDutyTaxUSD != 0 {
		t.Fatalf("TotalDutyTaxUSD = %v (expected 0)", res.TotalDutyTaxUSD)
	}
}

func TestService_Decide_DeMinimisThreshold(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Country with 20% duty but de minimis threshold of 800 USD
	svc.Create(&CreateRuleInput{
		CountryCode:     "AU",
		DutyRatePct:     float64Ptr(20.0),
		MinThresholdUSD: float64Ptr(800.0),
		Incoterm:        "DDP",
		Priority:        intPtr(1),
	})

	// 500 < 800 threshold, so no duty applies
	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "AU",
		ProductValueUSD:    500,
		Quantity:           1,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if res.Incoterm != "DDU" {
		t.Fatalf("expected DDU (de minimis), got %s (reason: %s)", res.Incoterm, res.IncotermReason)
	}
	if res.TotalDutyTaxUSD != 0 {
		t.Fatalf("TotalDutyTaxUSD = %v (expected 0)", res.TotalDutyTaxUSD)
	}
}

func TestService_Decide_HS_Code_PrefersExactMatch(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// General rule for country (5%)
	svc.Create(&CreateRuleInput{
		CountryCode: "BR",
		DutyRatePct: float64Ptr(5.0),
		Incoterm:    "DDU",
		Priority:    intPtr(1),
	})
	// Specific HS code prefix rule (15%)
	svc.Create(&CreateRuleInput{
		CountryCode:  "BR",
		HSCodePrefix: "84",
		DutyRatePct:  float64Ptr(15.0),
		Incoterm:     "DDU",
		Priority:     intPtr(2),
	})
	// Exact HS code match (25%)
	svc.Create(&CreateRuleInput{
		CountryCode: "BR",
		HSCode:      "847130",
		DutyRatePct: float64Ptr(25.0),
		Incoterm:    "DDU",
		Priority:    intPtr(3),
	})

	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "BR",
		ProductValueUSD:    100,
		HSCode:             "847130",
		Quantity:           1,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	// Should match 25% exact HS code (847130) → 25 duty
	if res.DutyAmountUSD != 25.0 {
		t.Fatalf("DutyAmountUSD = %v (expected 25.0)", res.DutyAmountUSD)
	}
	if len(res.RulesMatched) != 1 {
		t.Fatalf("expected 1 matched rule, got %d", len(res.RulesMatched))
	}
	if res.RulesMatched[0].RuleID == 0 {
		t.Fatal("matched rule should have a non-zero ID")
	}
}

func TestService_Decide_HS_Code_PrefersPrefixMatch(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// General country rule (5%)
	svc.Create(&CreateRuleInput{
		CountryCode: "CN",
		DutyRatePct: float64Ptr(5.0),
		Incoterm:    "DDU",
		Priority:    intPtr(1),
	})
	// HS code prefix rule (12%)
	svc.Create(&CreateRuleInput{
		CountryCode:  "CN",
		HSCodePrefix: "61",
		DutyRatePct:  float64Ptr(12.0),
		Incoterm:     "DDU",
		Priority:     intPtr(2),
	})

	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "CN",
		ProductValueUSD:    100,
		HSCode:             "610910",
		Quantity:           1,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	// Should match prefix "61" → 12% duty = 12.0
	if res.DutyAmountUSD != 12.0 {
		t.Fatalf("DutyAmountUSD = %v (expected 12.0)", res.DutyAmountUSD)
	}
}

func TestService_Decide_WithVAT(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// EU country with duty 5% + VAT 20%
	svc.Create(&CreateRuleInput{
		CountryCode: "FR",
		DutyRatePct: float64Ptr(5.0),
		VatRatePct:  float64Ptr(20.0),
		Incoterm:    "DDU",
		Priority:    intPtr(1),
	})

	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "FR",
		ProductValueUSD:    100,
		Quantity:           1,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	// Duty = 100 * 5% = 5
	// VAT = (100 + 5) * 20% = 21
	// Total = 26
	if res.DutyAmountUSD != 5.0 {
		t.Fatalf("DutyAmountUSD = %v (expected 5.0)", res.DutyAmountUSD)
	}
	if res.VatAmountUSD != 21.0 {
		t.Fatalf("VatAmountUSD = %v (expected 21.0)", res.VatAmountUSD)
	}
	if res.TotalDutyTaxUSD != 26.0 {
		t.Fatalf("TotalDutyTaxUSD = %v (expected 26.0)", res.TotalDutyTaxUSD)
	}
}

func TestService_Decide_ZeroQuantityDefaultsToOne(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		CountryCode: "SG",
		DutyRatePct: float64Ptr(10.0),
		Incoterm:    "DDU",
		Priority:    intPtr(1),
	})

	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "SG",
		ProductValueUSD:    200,
		Quantity:           0, // zero should default to 1
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	// Duty = 200 * 10% = 20
	if res.DutyAmountUSD != 20.0 {
		t.Fatalf("DutyAmountUSD = %v (expected 20.0)", res.DutyAmountUSD)
	}
}

func TestService_Decide_MultipleQuantity(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &TariffRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		CountryCode: "MX",
		DutyRatePct: float64Ptr(8.0),
		Incoterm:    "DDU",
		Priority:    intPtr(1),
	})

	res, err := svc.Decide(&DecisionRequest{
		DestinationCountry: "MX",
		ProductValueUSD:    100,
		Quantity:           5,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	// Total value = 100 * 5 = 500
	// Duty = 500 * 8% = 40
	if res.TotalValueUSD != 500 {
		t.Fatalf("TotalValueUSD = %v (expected 500)", res.TotalValueUSD)
	}
	if res.DutyAmountUSD != 40.0 {
		t.Fatalf("DutyAmountUSD = %v (expected 40.0)", res.DutyAmountUSD)
	}
}
