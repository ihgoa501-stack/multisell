package sentiment

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *ProductSentiment {
	t.Helper()
	return &ProductSentiment{}
}

func createTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	// Create order_item table for CalculateSentiment queries.
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS order_item (
		id INTEGER PRIMARY KEY,
		product_id INTEGER,
		rating INTEGER,
		return_qty INTEGER DEFAULT 0
	)`).Error; err != nil {
		t.Fatalf("create order_item: %v", err)
	}
	// Create product table for ListNegativeSentiment / GetSentiment queries.
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS product (
		id INTEGER PRIMARY KEY,
		name TEXT
	)`).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
}

func TestService_GetSentiment_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	svc := NewService(db, dbtest.NewLogger(t))

	resp, err := svc.GetSentiment(999)
	if err != nil {
		t.Fatalf("GetSentiment failed: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response for product with no sentiment data")
	}
}

func TestService_GetSentiment_Found(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	createTables(t, db)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductSentiment{
		ProductID:      1,
		AvgRating:      4.5,
		ReviewCount:    10,
		PositivePct:    80,
		NegativePct:    5,
		ReturnRate:     2.5,
		SentimentScore: 85,
	})

	resp, err := svc.GetSentiment(1)
	if err != nil {
		t.Fatalf("GetSentiment failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.SentimentScore != 85 {
		t.Errorf("expected score 85, got %f", resp.SentimentScore)
	}
	if resp.AvgRating != 4.5 {
		t.Errorf("expected avg_rating 4.5, got %f", resp.AvgRating)
	}
}

func TestService_CalculateSentiment_NoData(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	createTables(t, db)
	svc := NewService(db, dbtest.NewLogger(t))

	// When no order_item data exists, CalculateSentiment should still return
	// a sentiment record with 0 values (not an error).
	sentiment, err := svc.CalculateSentiment(1)
	if err != nil {
		t.Fatalf("CalculateSentiment failed: %v", err)
	}
	if sentiment == nil {
		t.Fatal("expected non-nil sentiment")
	}
	if sentiment.ProductID != 1 {
		t.Errorf("expected product_id 1, got %d", sentiment.ProductID)
	}
	// With no data: ratingScore=0, returnScore=100 (no returns), volumeBonus=0
	// Score = 0*0.5 + 100*0.3 + 0*0.2 = 30
	if sentiment.SentimentScore != 30 {
		t.Errorf("expected score 30 (default formula), got %f", sentiment.SentimentScore)
	}
}

func TestService_TopComplaints_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	svc := NewService(db, dbtest.NewLogger(t))

	resp, err := svc.TopComplaints(999)
	if err != nil {
		t.Fatalf("TopComplaints failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Complaints) != 0 {
		t.Errorf("expected 0 complaints, got %d", len(resp.Complaints))
	}
}

func TestService_TopComplaints_WithData(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductSentiment{
		ProductID:      1,
		TopComplaints:  `[{"category":"quality","frequency":5,"pct":0.5},{"category":"shipping","frequency":3,"pct":0.3}]`,
		SentimentScore: 60,
	})

	resp, err := svc.TopComplaints(1)
	if err != nil {
		t.Fatalf("TopComplaints failed: %v", err)
	}
	if len(resp.Complaints) != 2 {
		t.Errorf("expected 2 complaints, got %d", len(resp.Complaints))
	}
	if resp.Complaints[0].Category != "quality" {
		t.Errorf("expected first complaint 'quality', got %s", resp.Complaints[0].Category)
	}
}

func TestService_TopComplaints_EmptyJSON(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductSentiment{
		ProductID:     1,
		TopComplaints: `[]`,
	})

	resp, err := svc.TopComplaints(1)
	if err != nil {
		t.Fatalf("TopComplaints failed: %v", err)
	}
	if len(resp.Complaints) != 0 {
		t.Errorf("expected 0 complaints for empty JSON, got %d", len(resp.Complaints))
	}
}

func TestService_ListNegativeSentiment_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	createTables(t, db)
	svc := NewService(db, dbtest.NewLogger(t))

	items, err := svc.ListNegativeSentiment()
	if err != nil {
		t.Fatalf("ListNegativeSentiment failed: %v", err)
	}
	if items == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestService_ListNegativeSentiment_WithData(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	createTables(t, db)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductSentiment{ProductID: 1, SentimentScore: 30, ReviewCount: 5, AvgRating: 2.0, PositivePct: 20, NegativePct: 60, ReturnRate: 15})
	db.Create(&ProductSentiment{ProductID: 2, SentimentScore: 70, ReviewCount: 10, AvgRating: 4.0, PositivePct: 80, NegativePct: 5, ReturnRate: 2})

	items, err := svc.ListNegativeSentiment()
	if err != nil {
		t.Fatalf("ListNegativeSentiment failed: %v", err)
	}
	// Only product 1 has score < 50
	if len(items) != 1 {
		t.Errorf("expected 1 negative sentiment item, got %d", len(items))
	}
	if len(items) > 0 && items[0].ProductID != 1 {
		t.Errorf("expected product_id 1, got %d", items[0].ProductID)
	}
}

func TestService_CalculateSentiment_Idempotent(t *testing.T) {
	db := dbtest.NewDB(t, &ProductSentiment{})
	createTables(t, db)
	svc := NewService(db, dbtest.NewLogger(t))

	// Calculate twice for the same product should upsert (single record).
	s1, _ := svc.CalculateSentiment(1)
	s2, _ := svc.CalculateSentiment(1)

	if s1.ID != s2.ID {
		t.Errorf("expected same ID on upsert (idempotent), got %d vs %d", s1.ID, s2.ID)
	}
}

func TestService_GetSentiment_NoProductTable(t *testing.T) {
	// Even without the 'product' table (for product name lookup),
	// GetSentiment should return the sentiment data without error.
	db := dbtest.NewDB(t, &ProductSentiment{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductSentiment{ProductID: 1, SentimentScore: 90, AvgRating: 4.8, ReviewCount: 20})

	resp, err := svc.GetSentiment(1)
	if err != nil {
		t.Fatalf("GetSentiment failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ProductName != "" {
		t.Errorf("expected empty product name (no product table), got %q", resp.ProductName)
	}
}
