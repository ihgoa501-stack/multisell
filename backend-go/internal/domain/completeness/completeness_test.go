package completeness

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"github.com/lingmirror/backend-go/internal/domain/profit"
)

func TestService_Check_CreateErrorIsReturned(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert a candidate so First() succeeds
	platformID := int64(1)
	prod := candidate.CandidateProduct{
		Title:            "Test",
		Description:      "Enough chars for completeness to pass title/desc checks",
		MainImage:        "https://example.test/img.jpg",
		Images:           []byte(`["img1.jpg"]`),
		CategoryID:       nil,
		BrandID:          nil,
		SpecJSON:         []byte(`{}`),
		PurchasePrice:    10,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		HSCode:           "1234.56",
		TargetSalePrice:  30,
		TargetPlatformID: &platformID,
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	// Drop the target table so Create fails
	db.Exec("DROP TABLE completeness_check")

	_, err := svc.Check(prod.ID, "tester")
	if err == nil {
		t.Fatal("expected error from Create (table dropped)")
	}
}

func TestService_Check_CompleteProduct(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	// The check reads candidate_product which won't exist in this test DB
	svc := NewService(db, dbtest.NewLogger(t))

	// Without a candidate product, it should error
	_, err := svc.Check(999, "tester")
	if err == nil {
		t.Fatal("expected error for non-existent product")
	}
}

func TestService_ListChecks(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert some checks directly
	db.Create(&CompletenessCheck{ProductID: 1, Score: 90, Status: "complete", TriggeredBy: "system"})
	db.Create(&CompletenessCheck{ProductID: 2, Score: 45, Status: "incomplete", TriggeredBy: "system"})

	items, total, err := svc.ListChecks(1, 10, "")
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
}

func TestService_ListChecks_Filtered(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&CompletenessCheck{ProductID: 1, Score: 90, Status: "complete"})
	db.Create(&CompletenessCheck{ProductID: 2, Score: 45, Status: "incomplete"})

	items, total, err := svc.ListChecks(1, 10, "complete")
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if items[0].Status != "complete" {
		t.Fatalf("status = %s", items[0].Status)
	}
}

func TestService_CheckEnhanced_NoProfitSvc(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{}, &profit.ProfitSummary{})
	svc := NewService(db, dbtest.NewLogger(t))
	// profitSvc is nil — estimates will be nil

	platformID := int64(1)
	prod := candidate.CandidateProduct{
		Title:            "Enhanced Test Product",
		Description:      "This product has sufficient description for completeness.",
		MainImage:        "https://example.test/img.jpg",
		Images:           []byte(`["img1.jpg","img2.jpg"]`),
		CategoryID:       &platformID,
		BrandID:          &platformID,
		SpecJSON:         []byte(`{"color":"red"}`),
		PurchasePrice:    50,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  1.5,
		PackageLengthCm:  20,
		PackageWidthCm:   15,
		PackageHeightCm:  10,
		HSCode:           "8471.30",
		TargetSalePrice:  100,
		TargetPlatformID: &platformID,
		OriginCountry:    "CN",
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	r, err := svc.CheckEnhanced(prod.ID, "tester")
	if err != nil {
		t.Fatalf("CheckEnhanced: %v", err)
	}

	if r.CandidateID != prod.ID {
		t.Fatalf("CandidateID = %d, want %d", r.CandidateID, prod.ID)
	}
	if r.BaseInfoScore <= 0 {
		t.Fatalf("BaseInfoScore = %f, want >0", r.BaseInfoScore)
	}
	if r.CostScore != 1.0 {
		t.Fatalf("CostScore = %f, want 1.0", r.CostScore)
	}
	if r.LogisticsScore != 1.0 {
		t.Fatalf("LogisticsScore = %f, want 1.0", r.LogisticsScore)
	}
	if r.PlatformFeeScore != 1.0 {
		t.Fatalf("PlatformFeeScore = %f, want 1.0", r.PlatformFeeScore)
	}
	if r.ProfitScore <= 0 {
		t.Fatalf("ProfitScore = %f, want >0", r.ProfitScore)
	}
	if r.OverallScore <= 0 {
		t.Fatalf("OverallScore = %f, want >0", r.OverallScore)
	}
	// No profit service → estimates nil
	if r.EstimatedProfit != nil {
		t.Fatal("Expected nil EstimatedProfit without profit service")
	}
	if r.EstimatedMargin != nil {
		t.Fatal("Expected nil EstimatedMargin without profit service")
	}
	if r.EstimatedLogistics != nil {
		t.Fatal("Expected nil EstimatedLogistics without profit service")
	}
	if r.EstimatedPlatformFee != nil {
		t.Fatal("Expected nil EstimatedPlatformFee without profit service")
	}
}

func TestService_CheckEnhanced_MissingFields(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	prod := candidate.CandidateProduct{
		Title:       "Minimal",
		Description: "Enough chars for desc check to pass",
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	r, err := svc.CheckEnhanced(prod.ID, "tester")
	if err != nil {
		t.Fatalf("CheckEnhanced: %v", err)
	}

	if r.CostScore != 0 {
		t.Fatalf("CostScore = %f, want 0", r.CostScore)
	}
	if r.LogisticsScore != 0 {
		t.Fatalf("LogisticsScore = %f, want 0", r.LogisticsScore)
	}
	if r.PlatformFeeScore != 0 {
		t.Fatalf("PlatformFeeScore = %f, want 0", r.PlatformFeeScore)
	}
	if r.ProfitScore != 0 {
		t.Fatalf("ProfitScore = %f, want 0", r.ProfitScore)
	}

	foundMissing := map[string]bool{}
	for _, m := range r.MissingFields {
		foundMissing[m] = true
	}
	expected := []string{"采购成本", "目标售价", "包装重量", "包装尺寸", "目标平台"}
	for _, e := range expected {
		if !foundMissing[e] {
			t.Fatalf("missing field %q not found in report", e)
		}
	}
}

func TestService_CheckEnhanced_WithProfitService(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{}, &profit.ProfitSummary{})
	rateSvc := exchangerate.NewService(db, dbtest.NewLogger(t))
	profitSvc := profit.NewService(db, dbtest.NewLogger(t), rateSvc, 7.2)

	svc := NewService(db, dbtest.NewLogger(t))
	svc.profitSvc = profitSvc

	platformID := int64(1)
	prod := candidate.CandidateProduct{
		Title:            "Profit Check Product",
		Description:      "Sufficiently long description for the completeness check to pass on desc.",
		MainImage:        "https://example.test/img.jpg",
		Images:           []byte(`["img1.jpg"]`),
		CategoryID:       &platformID,
		BrandID:          &platformID,
		SpecJSON:         []byte(`{}`),
		PurchasePrice:    50,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  1.0,
		PackageLengthCm:  20,
		PackageWidthCm:   15,
		PackageHeightCm:  10,
		HSCode:           "8471.30",
		TargetSalePrice:  100,
		TargetPlatformID: &platformID,
		OriginCountry:    "CN",
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	r, err := svc.CheckEnhanced(prod.ID, "tester")
	if err != nil {
		t.Fatalf("CheckEnhanced: %v", err)
	}

	if r.EstimatedProfit == nil {
		t.Fatal("Expected EstimatedProfit, got nil")
	}
	if r.EstimatedMargin == nil {
		t.Fatal("Expected EstimatedMargin, got nil")
	}
	if r.EstimatedLogistics == nil {
		t.Fatal("Expected EstimatedLogistics, got nil")
	}
	if r.EstimatedPlatformFee == nil {
		t.Fatal("Expected EstimatedPlatformFee, got nil")
	}
	if *r.EstimatedProfit <= 0 {
		t.Fatalf("EstimatedProfit = %f, want > 0", *r.EstimatedProfit)
	}
	if *r.EstimatedLogistics <= 0 {
		t.Fatalf("EstimatedLogistics = %f, want > 0", *r.EstimatedLogistics)
	}
	if r.CostScore != 1.0 {
		t.Fatalf("CostScore = %f, want 1.0", r.CostScore)
	}
	if r.LogisticsScore != 1.0 {
		t.Fatalf("LogisticsScore = %f, want 1.0", r.LogisticsScore)
	}
	if r.PlatformFeeScore != 1.0 {
		t.Fatalf("PlatformFeeScore = %f, want 1.0", r.PlatformFeeScore)
	}
	if r.ProfitScore != 1.0 {
		t.Fatalf("ProfitScore = %f, want 1.0", r.ProfitScore)
	}
}
