package productanalysis

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &ProductAnalysis{}, &AnalysisFeedback{})
}

func newService(t *testing.T, db *gorm.DB) Service {
	t.Helper()
	return NewService(db, dbtest.NewLogger(t))
}

func TestAnalyze_Basic(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	// Insert a sourcing product into the same DB (test uses SQLite)
	db.Exec("CREATE TABLE sourcing_1688_product (id INTEGER PRIMARY KEY, price_1688 REAL, status TEXT)")
	db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (1, 10.0, 'pending')")

	result, err := svc.Analyze(&AnalyzeInput{
		SourcingProductID: 1,
		TargetSalePrice:   25.0,
	}, "user1")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if result.Analysis.ID == 0 {
		t.Fatal("expected non-zero analysis ID")
	}
	if result.Analysis.Status != "completed" {
		t.Fatalf("status = %q, want completed", result.Analysis.Status)
	}
	// margin = (25-10)/25 = 60% → score = 100 (capped)
	if result.ProfitScore == nil || *result.ProfitScore != 100 {
		t.Fatalf("profit score = %v, want 100", valueOrZero(result.ProfitScore))
	}
	if result.DemandScoreStatus != "no_data" {
		t.Fatalf("demand status = %q, want no_data", result.DemandScoreStatus)
	}
	if result.CompetitionStatus != "no_data" {
		t.Fatalf("competition status = %q, want no_data", result.CompetitionStatus)
	}
	if result.Warning == "" {
		t.Fatal("expected disclaimer warning")
	}
}

func TestAnalyze_ZeroPrice(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	db.Exec("CREATE TABLE sourcing_1688_product (id INTEGER PRIMARY KEY, price_1688 REAL, status TEXT)")
	db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (2, 0, 'pending')")

	// cost is 0 → margin = 100% → score = 100
	result, err := svc.Analyze(&AnalyzeInput{
		SourcingProductID: 2,
		TargetSalePrice:   10.0,
	}, "user2")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if result.ProfitScore == nil || *result.ProfitScore != 100 {
		t.Fatalf("profit score = %v, want 100 for zero cost", valueOrZero(result.ProfitScore))
	}
}

func TestAnalyze_ProductNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	_, err := svc.Analyze(&AnalyzeInput{
		SourcingProductID: 999,
		TargetSalePrice:   10.0,
	}, "user3")
	if err == nil {
		t.Fatal("expected error for non-existent sourcing product")
	}
}

func TestAnalyze_NegativeMargin(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	db.Exec("CREATE TABLE sourcing_1688_product (id INTEGER PRIMARY KEY, price_1688 REAL, status TEXT)")
	db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (3, 50.0, 'pending')")

	// sale price (10) < cost (50) → negative margin → score 0
	result, err := svc.Analyze(&AnalyzeInput{
		SourcingProductID: 3,
		TargetSalePrice:   10.0,
	}, "user4")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if result.ProfitScore == nil || *result.ProfitScore != 0 {
		t.Fatalf("profit score = %v, want 0 for negative margin", valueOrZero(result.ProfitScore))
	}
}

func TestGetAnalysis_ScopedToUser(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	db.Exec("CREATE TABLE sourcing_1688_product (id INTEGER PRIMARY KEY, price_1688 REAL, status TEXT)")
	db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (4, 10.0, 'pending')")

	result, err := svc.Analyze(&AnalyzeInput{
		SourcingProductID: 4,
		TargetSalePrice:   20.0,
	}, "alice")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// alice can read her own analysis
	_, err = svc.GetAnalysis(result.Analysis.ID, "alice")
	if err != nil {
		t.Fatalf("GetAnalysis own: %v", err)
	}

	// bob cannot
	_, err = svc.GetAnalysis(result.Analysis.ID, "bob")
	if err == nil {
		t.Fatal("expected error: bob should not see alice's analysis")
	}
}

func TestGetAnalysis_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	_, err := svc.GetAnalysis(999, "user")
	if err == nil {
		t.Fatal("expected error for non-existent analysis")
	}
}

func TestListAnalyses_Pagination(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	db.Exec("CREATE TABLE sourcing_1688_product (id INTEGER PRIMARY KEY, price_1688 REAL, status TEXT)")
	for i := 1; i <= 5; i++ {
		db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (?, 10.0, 'pending')", int64(100+i))
	}

	// no analyses yet → 0 results
	items, total, err := svc.ListAnalyses(&ListFilter{UserID: "u1", Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("ListAnalyses failed: %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}

	// create 3 analyses
	for i := 1; i <= 3; i++ {
		_, err := svc.Analyze(&AnalyzeInput{
			SourcingProductID: int64(100 + i),
			TargetSalePrice:   20.0,
		}, "u1")
		if err != nil {
			t.Fatalf("Analyze %d: %v", i, err)
		}
	}

	items, total, err = svc.ListAnalyses(&ListFilter{UserID: "u1", Page: 1, Size: 2})
	if err != nil {
		t.Fatalf("ListAnalyses failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestListAnalyses_UserScoped(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	db.Exec("CREATE TABLE sourcing_1688_product (id INTEGER PRIMARY KEY, price_1688 REAL, status TEXT)")
	db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (50, 10.0, 'pending')")
	db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (51, 10.0, 'pending')")

	svc.Analyze(&AnalyzeInput{SourcingProductID: 50, TargetSalePrice: 20}, "u1")
	svc.Analyze(&AnalyzeInput{SourcingProductID: 51, TargetSalePrice: 20}, "u2")

	items, total, _ := svc.ListAnalyses(&ListFilter{UserID: "u1", Page: 1, Size: 20})
	if total != 1 || len(items) != 1 {
		t.Fatalf("u1: total=%d items=%d, want 1,1", total, len(items))
	}
}

func TestRecordFeedback(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	db.Exec("CREATE TABLE sourcing_1688_product (id INTEGER PRIMARY KEY, price_1688 REAL, status TEXT)")
	db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (60, 10.0, 'pending')")

	result, _ := svc.Analyze(&AnalyzeInput{SourcingProductID: 60, TargetSalePrice: 20}, "u1")

	err := svc.RecordFeedback(result.Analysis.ID, &FeedbackInput{
		Decision: "imported",
	}, "u1")
	if err != nil {
		t.Fatalf("RecordFeedback: %v", err)
	}

	// Another user cannot add feedback
	err = svc.RecordFeedback(result.Analysis.ID, &FeedbackInput{
		Decision: "abandoned",
	}, "u2")
	if err == nil {
		t.Fatal("expected error for cross-user feedback")
	}
}

func TestRecordFeedback_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	err := svc.RecordFeedback(999, &FeedbackInput{Decision: "abandoned"}, "u1")
	if err == nil {
		t.Fatal("expected error for non-existent analysis")
	}
}

// --- profit.go unit tests ---

func TestCalculateProfitMargin_Basic(t *testing.T) {
	m, s := CalculateProfitMargin(100, 40)
	if m == nil || *m != 60.0 {
		t.Fatalf("margin = %v, want 60", valueOrZero(m))
	}
	if s == nil || *s != 100 {
		t.Fatalf("score = %v, want 100", valueOrZero(s))
	}
}

func TestCalculateProfitMargin_ZeroCost(t *testing.T) {
	m, _ := CalculateProfitMargin(50, 0)
	if m == nil || *m != 100.0 {
		t.Fatalf("margin = %v, want 100", valueOrZero(m))
	}
}

func TestCalculateProfitMargin_NegativeCost(t *testing.T) {
	m, _ := CalculateProfitMargin(50, -10)
	// cost clamped to 0
	if m == nil || *m != 100.0 {
		t.Fatalf("margin = %v, want 100", valueOrZero(m))
	}
}

func TestCalculateProfitMargin_CostExceedsPrice(t *testing.T) {
	m, s := CalculateProfitMargin(30, 50)
	if m == nil || *m != 0 {
		t.Fatalf("margin = %v, want 0", valueOrZero(m))
	}
	if s == nil || *s != 0 {
		t.Fatalf("score = %v, want 0", valueOrZero(s))
	}
}

func TestCalculateProfitMargin_ZeroPrice(t *testing.T) {
	m, s := CalculateProfitMargin(0, 10)
	if m != nil || s != nil {
		t.Fatal("expected nil for zero price")
	}
}

func TestCalculateProfitMargin_ZeroBoth(t *testing.T) {
	m, s := CalculateProfitMargin(0, 0)
	if m != nil || s != nil {
		t.Fatal("expected nil for zero price")
	}
}

func TestAnalyze_CacheHit(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	db.Exec("CREATE TABLE sourcing_1688_product (id INTEGER PRIMARY KEY, price_1688 REAL, status TEXT)")
	db.Exec("INSERT INTO sourcing_1688_product (id, price_1688, status) VALUES (100, 10.0, 'pending')")

	r1, err := svc.Analyze(&AnalyzeInput{SourcingProductID: 100, TargetSalePrice: 25}, "cache_user")
	if err != nil {
		t.Fatalf("first analyze: %v", err)
	}

	r2, err := svc.Analyze(&AnalyzeInput{SourcingProductID: 100, TargetSalePrice: 25}, "cache_user")
	if err != nil {
		t.Fatalf("cached analyze: %v", err)
	}
	if r1.Analysis.ID != r2.Analysis.ID {
		t.Fatalf("cache miss: r1.id=%d r2.id=%d", r1.Analysis.ID, r2.Analysis.ID)
	}

	r3, err := svc.Analyze(&AnalyzeInput{SourcingProductID: 100, TargetSalePrice: 25}, "other_user")
	if err != nil {
		t.Fatalf("other user analyze: %v", err)
	}
	if r3.Analysis.ID == r1.Analysis.ID {
		t.Fatal("cache leak: other user got cached result")
	}
}
