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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	// Create the tables referenced in RiskSummary queries.
	exec(t, db, `CREATE TABLE IF NOT EXISTS candidate_product (id INTEGER PRIMARY KEY, title TEXT)`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS completeness_check (id INTEGER PRIMARY KEY, product_id INTEGER, status TEXT)`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS profit_summary (id INTEGER PRIMARY KEY, product_id INTEGER, status TEXT)`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_task (id INTEGER PRIMARY KEY, status TEXT, updated_at TIMESTAMP, created_at TIMESTAMP)`)
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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (
		id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT,
		confidence REAL, reason TEXT, risk_flags TEXT,
		feedback_status TEXT, feedback_note TEXT,
		completeness_score REAL, profit_margin REAL, estimated_profit REAL,
		created_listing_task_id INTEGER, created_at TIMESTAMP
	)`)

	suggestions, err := svc.Suggestions(10)
	if err != nil {
		t.Fatalf("Suggestions failed: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(suggestions))
	}
}

func TestService_Suggestions_DefaultLimit(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t))
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (
		id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT,
		confidence REAL, reason TEXT, risk_flags TEXT,
		feedback_status TEXT, feedback_note TEXT,
		completeness_score REAL, profit_margin REAL, estimated_profit REAL,
		created_listing_task_id INTEGER, created_at TIMESTAMP
	)`)

	// Zero limit should default to 20.
	suggestions, err := svc.Suggestions(0)
	if err != nil {
		t.Fatalf("Suggestions failed: %v", err)
	}
	_ = suggestions
}

func TestService_Suggestions_WithFeedbackStatus(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t))
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	exec(t, db, `CREATE TABLE IF NOT EXISTS candidate_product (
		id INTEGER PRIMARY KEY, title TEXT
	)`)
	exec(t, db, `INSERT INTO candidate_product (id, title) VALUES (1, 'Test Product')`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (
		id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT,
		confidence REAL, reason TEXT, risk_flags TEXT,
		feedback_status TEXT, feedback_note TEXT,
		completeness_score REAL, profit_margin REAL, estimated_profit REAL,
		created_listing_task_id INTEGER, created_at TIMESTAMP
	)`)
	exec(t, db, `INSERT INTO listing_recommendation (id, product_id, decision, confidence, feedback_status, completeness_score, profit_margin, estimated_profit)
		VALUES (1, 1, 'list', 0.85, 'adopted', 90, 20.5, 150.0)`)

	suggestions, err := svc.Suggestions(10)
	if err != nil {
		t.Fatalf("Suggestions failed: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	s := suggestions[0]
	if s.FeedbackStatus != "adopted" {
		t.Errorf("expected feedback_status 'adopted', got '%s'", s.FeedbackStatus)
	}
	if s.ProductTitle != "Test Product" {
		t.Errorf("expected title 'Test Product', got '%s'", s.ProductTitle)
	}
	if s.CompletenessScore != 90 {
		t.Errorf("expected completeness_score 90, got %f", s.CompletenessScore)
	}
	if s.ProfitMargin != 20.5 {
		t.Errorf("expected profit_margin 20.5, got %f", s.ProfitMargin)
	}
	if s.EstimatedProfit != 150.0 {
		t.Errorf("expected estimated_profit 150.0, got %f", s.EstimatedProfit)
	}
}

func TestService_RecordFeedback_InvalidAction(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t))
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (
		id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT,
		confidence REAL, feedback_status TEXT, feedback_note TEXT
	)`)
	exec(t, db, `INSERT INTO listing_recommendation (id, product_id, decision) VALUES (1, 1, 'list')`)

	err := svc.RecordFeedback(1, &FeedbackInput{Action: "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid action, got nil")
	}
}

func TestService_RecordFeedback_Reject(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t))
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (
		id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT,
		confidence REAL, feedback_status TEXT, feedback_note TEXT,
		created_listing_task_id INTEGER
	)`)
	exec(t, db, `INSERT INTO listing_recommendation (id, product_id, decision) VALUES (1, 1, 'list')`)

	err := svc.RecordFeedback(1, &FeedbackInput{Action: "reject", Note: "not ready yet"})
	if err != nil {
		t.Fatalf("RecordFeedback reject failed: %v", err)
	}

	// Verify feedback status updated
	type RecCheck struct {
		FeedbackStatus string
		FeedbackNote   string
	}
	var rc RecCheck
	db.Table("listing_recommendation").Select("feedback_status, feedback_note").First(&rc)
	if rc.FeedbackStatus != "rejected" {
		t.Errorf("expected 'rejected', got '%s'", rc.FeedbackStatus)
	}
	if rc.FeedbackNote != "not ready yet" {
		t.Errorf("expected note 'not ready yet', got '%s'", rc.FeedbackNote)
	}
}

func TestService_RecordFeedback_AdoptWithListingTask(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t))
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_recommendation (
		id INTEGER PRIMARY KEY, product_id INTEGER, decision TEXT,
		confidence REAL, feedback_status TEXT, feedback_note TEXT,
		created_listing_task_id INTEGER
	)`)
	listingTaskID := int64(99)
	exec(t, db, `INSERT INTO listing_recommendation (id, product_id, decision, created_listing_task_id)
		VALUES (1, 1, 'list', ?)`, listingTaskID)
	exec(t, db, `CREATE TABLE IF NOT EXISTS listing_task (id INTEGER PRIMARY KEY, status TEXT, updated_at TIMESTAMP, created_at TIMESTAMP)`)
	exec(t, db, `INSERT INTO listing_task (id, status) VALUES (99, 'blocked')`)
	exec(t, db, `CREATE TABLE IF NOT EXISTS approval_request (
		id INTEGER PRIMARY KEY, product_id INTEGER, request_type TEXT,
		requester TEXT, reviewer TEXT, status TEXT, old_value TEXT,
		new_value TEXT, reason TEXT, review_note TEXT, entity_type TEXT,
		entity_id INTEGER, expires_at TIMESTAMP, updated_at TIMESTAMP, created_at TIMESTAMP
	)`)

	err := svc.RecordFeedback(1, &FeedbackInput{Action: "adopt", Note: "looks good"})
	if err != nil {
		t.Fatalf("RecordFeedback adopt failed: %v", err)
	}

	// Verify recommendation feedback_status
	type RecCheck struct {
		FeedbackStatus string
		FeedbackNote   string
	}
	var rc RecCheck
	db.Table("listing_recommendation").Select("feedback_status, feedback_note").First(&rc)
	if rc.FeedbackStatus != "adopted" {
		t.Errorf("expected 'adopted', got '%s'", rc.FeedbackStatus)
	}
	if rc.FeedbackNote != "looks good" {
		t.Errorf("expected note 'looks good', got '%s'", rc.FeedbackNote)
	}

	// Verify listing task status updated to pending_approval
	var taskStatus string
	db.Table("listing_task").Select("status").Where("id = ?", 99).Scan(&taskStatus)
	if taskStatus != "pending_approval" {
		t.Errorf("expected listing_task.status 'pending_approval', got '%s'", taskStatus)
	}

	// Verify approval request was created
	var approvalCount int64
	db.Table("approval_request").Count(&approvalCount)
	if approvalCount != 1 {
		t.Errorf("expected 1 approval request, got %d", approvalCount)
	}

	var entityType string
	db.Table("approval_request").Select("entity_type").Scan(&entityType)
	if entityType != "listing_task" {
		t.Errorf("expected entity_type 'listing_task', got '%s'", entityType)
	}
}

func TestService_RecordFeedback_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t))
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	err := svc.RecordFeedback(999, &FeedbackInput{Action: "adopt"})
	if err == nil {
		t.Fatal("expected error for non-existent recommendation, got nil")
	}
}

func TestService_PlatformSyncStatus_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

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

func exec(t *testing.T, db *gorm.DB, sql string, args ...interface{}) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
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
