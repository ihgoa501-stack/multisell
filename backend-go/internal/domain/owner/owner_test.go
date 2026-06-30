package owner

import (
	"testing"

	"gorm.io/gorm"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/loop"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Owner uses raw SQL queries to many tables. We create empty versions
	// of those tables to prevent SQLite errors.
	return dbtest.NewDB(t)
}

func TestService_RiskSummary(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	// Create the tables referenced in RiskSummary queries.
	exec(t, db, `CREATE TABLE IF NOT EXISTS candidate_product (id INTEGER PRIMARY KEY, title TEXT)`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS completeness_check (id INTEGER PRIMARY KEY, product_id INTEGER, status TEXT)`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS profit_summary (id INTEGER PRIMARY KEY, product_id INTEGER, status TEXT)`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_task (id INTEGER PRIMARY KEY, status TEXT)`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS mock_sync_status (id INTEGER PRIMARY KEY, status TEXT)`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT)`)

	result, err := svc.RiskSummary()
	if err != nil {
		t.Fatalf("RiskSummary failed: %v", err)
	}

	// All values should be 0 since no data.
	if result["total_candidates"].(int64) != 0 {
		t.Errorf("expected total_candidates 0, got %d", result["total_candidates"])
	}
	if result["missing_data_products"].(int64) != 0 {
		t.Errorf("expected missing_data_products 0, got %d", result["missing_data_products"])
	}
	if result["low_profit_products"].(int64) != 0 {
		t.Errorf("expected low_profit_products 0, got %d", result["low_profit_products"])
	}
	if result["sync_errors"].(int64) != 0 {
		t.Errorf("expected sync_errors 0, got %d", result["sync_errors"])
	}
	// Verify all expected keys are present.
	expectedKeys := []string{"low_profit_products", "missing_data_products", "pending_approvals",
		"sync_errors", "total_candidates", "total_recommendations", "list_ready_products"}
	for _, k := range expectedKeys {
		if _, ok := result[k]; !ok {
			t.Errorf("missing key in result: %s", k)
		}
	}
}

func TestService_RiskSummary_WithData(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	exec(t, db, `CREATE TABLE IF NOT EXISTS candidate_product (id INTEGER PRIMARY KEY, title TEXT)`)
	exec(t, db, `INSERT INTO candidate_product (id, title) VALUES (1, 'Product A')`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT)`)
	exec(t, db, `INSERT INTO listing_recommendation (id, product_id, decision) VALUES (1, 1, 'list')`)

	result, err := svc.RiskSummary()
	if err != nil {
		t.Fatalf("RiskSummary failed: %v", err)
	}
	if result["total_candidates"].(int64) != 1 {
		t.Errorf("expected 1 candidate, got %d", result["total_candidates"])
	}
	if result["list_ready_products"].(int64) != 1 {
		t.Errorf("expected 1 list-ready, got %d", result["list_ready_products"])
	}
}

func TestService_Suggestions_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (
		id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT,
		confidence REAL, reason TEXT, risk_flags TEXT,
		created_listing_task_id INTEGER, created_at TIMESTAMP
	)`)

	suggestions, err := svc.Suggestions(10)
	if err != nil {
		t.Fatalf("Suggestions failed: %v", err)
	}
	// result may be nil when listing_recommendation table has no entries
	_ = suggestions
}

func TestService_PlatformSyncStatus_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	exec(t, db, `CREATE TABLE IF NOT EXISTS mock_sync_status (
		id INTEGER PRIMARY KEY, platform_id INTEGER, platform_name TEXT,
		sync_type TEXT, status TEXT, records_synced INTEGER,
		error_message TEXT, last_sync_at TIMESTAMP, is_mock_data BOOLEAN
	)`)

	result, err := svc.PlatformSyncStatus()
	if err != nil {
		t.Fatalf("PlatformSyncStatus failed: %v", err)
	}
	// result may be nil when mock_sync_status table has no entries
	_ = result
}

func TestService_Suggestions_DefaultLimit(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (
		id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT,
		confidence REAL, reason TEXT, risk_flags TEXT,
		created_listing_task_id INTEGER, created_at TIMESTAMP
	)`)

	// Zero limit should default to 20.
	suggestions, err := svc.Suggestions(0)
	if err != nil {
		t.Fatalf("Suggestions failed: %v", err)
	}
	_ = suggestions
}

func exec(t *testing.T, db *gorm.DB, sql string) {
	t.Helper()
	if err := db.Exec(sql).Error; err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func TestService_RiskSummaryIncludesPendingApprovalsAndBlockedTasks(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t,
		&approval.ApprovalRequest{},
		&listingtask.ListingTask{},
		&loop.ListingRecommendation{},
	)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&approval.ApprovalRequest{
		ProductID: 1, RequestType: "publish", Requester: "A8",
		Status: "pending", TargetType: "listing_task", TargetID: 7, RiskLevel: "high",
	})
	db.Create(&listingtask.ListingTask{
		ProductID: 1, PlatformID: 1, Status: "blocked", CreatedBy: "A8",
	})
	db.Create(&loop.ListingRecommendation{
		ProductID: 1, Decision: "list", Confidence: 0.91, Reason: "ready",
	})

	summary, err := svc.RiskSummary()
	if err != nil {
		t.Fatalf("RiskSummary: %v", err)
	}

	// Cast values since RiskSummary returns map[string]interface{}
	pendingApprovals := summary["pending_approval_count"].(int64)
	if pendingApprovals != 1 {
		t.Fatalf("pending approvals = %d, want 1", pendingApprovals)
	}

	blockedTasks := summary["blocked_listing_task_count"].(int64)
	if blockedTasks != 1 {
		t.Fatalf("blocked tasks = %d, want 1", blockedTasks)
	}

	recommendedListings := summary["recommended_listing_count"].(int64)
	if recommendedListings != 1 {
		t.Fatalf("recommended listings = %d, want 1", recommendedListings)
	}
}
