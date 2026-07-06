package profit

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
)

func TestService_Calculate_CreateErrorIsReturned(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProfitSummary{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t), nil, 7.2)

	platformID := int64(1)
	prod := candidate.CandidateProduct{
		Title:            "Test Product for Profit Calc",
		PurchasePrice:    50,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		HSCode:           "1234.56",
		TargetSalePrice:  30,
		TargetPlatformID: &platformID,
		DestinationCountry: "US",
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	// Drop target table so Create fails
	db.Exec("DROP TABLE profit_summary")

	_, err := svc.Calculate(prod.ID, "tester")
	if err == nil {
		t.Fatal("expected error from Create (table dropped)")
	}
}

func TestService_ListSummaries(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProfitSummary{})
	svc := NewService(db, dbtest.NewLogger(t), nil, 7.2)

	db.Create(&ProfitSummary{ProductID: 1, Status: "profitable", EstimatedProfit: 25.0, ProfitMargin: 20.0})
	db.Create(&ProfitSummary{ProductID: 2, Status: "unprofitable", EstimatedProfit: -5.0, ProfitMargin: -5.0})

	items, total, err := svc.ListSummaries(1, 10, "")
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
}

func TestService_ListSummaries_Filtered(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProfitSummary{})
	svc := NewService(db, dbtest.NewLogger(t), nil, 7.2)

	db.Create(&ProfitSummary{ProductID: 1, Status: "profitable"})
	db.Create(&ProfitSummary{ProductID: 2, Status: "unprofitable"})

	items, total, err := svc.ListSummaries(1, 10, "profitable")
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if items[0].Status != "profitable" {
		t.Fatalf("status = %s", items[0].Status)
	}
}

func TestClassifyProfit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		margin float64
		want   string
	}{
		{20, "profitable"},
		{15, "profitable"},
		{10, "marginal"},
		{0, "marginal"},
		{-5, "unprofitable"},
	}
	for _, tt := range tests {
		got := classifyProfit(tt.margin)
		if got != tt.want {
			t.Fatalf("classifyProfit(%f) = %s, want %s", tt.margin, got, tt.want)
		}
	}
}
