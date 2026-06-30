package profit

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_ListSummaries(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProfitSummary{})
	svc := NewService(db, dbtest.NewLogger(t))

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
	svc := NewService(db, dbtest.NewLogger(t))

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
