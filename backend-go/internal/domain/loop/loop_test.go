package loop

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/profit"
)

func TestService_GetRecommendations(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingRecommendation{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

	db.Create(&ListingRecommendation{ProductID: 1, Decision: "list", Confidence: 0.85, Reason: "good product"})
	db.Create(&ListingRecommendation{ProductID: 2, Decision: "skip", Confidence: 0.3, Reason: "bad product"})

	items, total, err := svc.GetRecommendations(1, 10, "")
	if err != nil {
		t.Fatalf("GetRecommendations: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}

	for _, item := range items {
		if item.FeedbackStatus != "pending" {
			t.Errorf("expected FeedbackStatus 'pending', got '%s' for item %d", item.FeedbackStatus, item.ID)
		}
	}
}

func TestService_GetRecommendations_Filtered(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingRecommendation{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

	db.Create(&ListingRecommendation{ProductID: 1, Decision: "list"})
	db.Create(&ListingRecommendation{ProductID: 2, Decision: "skip"})

	items, total, err := svc.GetRecommendations(1, 10, "list")
	if err != nil {
		t.Fatalf("GetRecommendations: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if items[0].Decision != "list" {
		t.Fatalf("decision = %s", items[0].Decision)
	}
}

func TestService_RecordExecutionResult_Success(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingRecommendation{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

	listingTaskID := int64(100)
	db.Create(&ListingRecommendation{
		ProductID:           1,
		Decision:            "list",
		CreatedListingTaskID: &listingTaskID,
		FeedbackStatus:      "adopted",
	})

	err := svc.RecordExecutionResult(1, 100, true, "")
	if err != nil {
		t.Fatalf("RecordExecutionResult: %v", err)
	}

	var rec ListingRecommendation
	db.First(&rec, 1)
	if rec.FeedbackStatus != "executed" {
		t.Errorf("expected 'executed', got '%s'", rec.FeedbackStatus)
	}
}

func TestService_RecordExecutionResult_Failure(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingRecommendation{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

	listingTaskID := int64(101)
	db.Create(&ListingRecommendation{
		ProductID:           2,
		Decision:            "list",
		CreatedListingTaskID: &listingTaskID,
		FeedbackStatus:      "adopted",
	})

	err := svc.RecordExecutionResult(2, 101, false, "platform publish failed")
	if err != nil {
		t.Fatalf("RecordExecutionResult: %v", err)
	}

	var rec ListingRecommendation
	db.First(&rec, 1) // id=1 because only 1 record inserted
	if rec.FeedbackStatus != "execution_failed" {
		t.Errorf("expected 'execution_failed', got '%s'", rec.FeedbackStatus)
	}
	if rec.FeedbackNote != "platform publish failed" {
		t.Errorf("expected note 'platform publish failed', got '%s'", rec.FeedbackNote)
	}
}

func TestService_EvaluateCreatesBlockedListingTaskAndApproval(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t,
		&candidate.CandidateProduct{},
		&completeness.CompletenessCheck{},
		&profit.ProfitSummary{},
		&listingtask.ListingTask{},
		&ListingRecommendation{},
		&approval.ApprovalRequest{},
		&operationlog.OperationLog{},
	)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, false)

	categoryID := int64(1)
	brandID := int64(1)
	platformID := int64(1)

	product := candidate.CandidateProduct{
		Title:              "Test Candidate",
		Description:        "Complete candidate product for testing purposes",
		MainImage:          "https://example.test/image.jpg",
		Images:             []byte(`["https://example.test/img1.jpg"]`),
		CategoryID:         &categoryID,
		BrandID:            &brandID,
		SpecJSON:           []byte(`{"color": "red", "size": "M"}`),
		PurchasePrice:      10,
		PurchaseCurrency:   "CNY",
		PackageWeightKg:    0.4,
		PackageLengthCm:    10,
		PackageWidthCm:     8,
		PackageHeightCm:    6,
		HSCode:             "1234.56",
		OriginCountry:      "CN",
		TargetSalePrice:    30,
		TargetCurrency:     "USD",
		TargetPlatformID:   &platformID,
		DestinationCountry: "RU",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	result, err := svc.Evaluate(product.ID, "A8")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.Decision != "list" {
		t.Fatalf("decision = %s, want list; reason=%s", result.Decision, result.Reason)
	}
	if result.ListingTaskID == nil {
		t.Fatal("expected listing task id")
	}

	var task listingtask.ListingTask
	if err := db.First(&task, *result.ListingTaskID).Error; err != nil {
		t.Fatalf("listing task: %v", err)
	}
	if task.Status != "blocked" {
		t.Fatalf("task status = %s, want blocked", task.Status)
	}

	var req approval.ApprovalRequest
	if err := db.Where("target_type = ? AND target_id = ? AND request_type = ?", "listing_task", task.ID, "publish").First(&req).Error; err != nil {
		t.Fatalf("approval request: %v", err)
	}
	if req.Status != "pending" || req.RiskLevel != "high" {
		t.Fatalf("unexpected approval: %+v", req)
	}
}

func TestEvaluate_MissingCriticalData_Blocked(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t,
		&candidate.CandidateProduct{},
		&completeness.CompletenessCheck{},
		&profit.ProfitSummary{},
		&listingtask.ListingTask{},
		&ListingRecommendation{},
		&approval.ApprovalRequest{},
		&operationlog.OperationLog{},
	)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, false)

	// Candidate with no weight, no dimensions, no price — critically incomplete
	categoryID := int64(1)
	platformID := int64(1)
	product := candidate.CandidateProduct{
		Title:             "Incomplete Product",
		Description:       "Missing critical data",
		MainImage:         "",
		CategoryID:        &categoryID,
		PurchasePrice:     0,
		PackageWeightKg:   0,
		PackageLengthCm:   0,
		PackageWidthCm:    0,
		PackageHeightCm:   0,
		TargetSalePrice:   0,
		TargetPlatformID:  &platformID,
		DestinationCountry: "RU",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	result, err := svc.Evaluate(product.ID, "A8")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.Decision != "skip" {
		t.Fatalf("decision = %s, want skip; reason=%s", result.Decision, result.Reason)
	}
	if result.Reason == "" {
		t.Fatal("expected a business-readable reason for the skip")
	}
	// No listing task should be created for skipped candidates
	if result.ListingTaskID != nil {
		t.Fatalf("expected no listing task for skip, got id=%d", *result.ListingTaskID)
	}
}

func TestEvaluate_LowProfit_NotRecommended(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t,
		&candidate.CandidateProduct{},
		&completeness.CompletenessCheck{},
		&profit.ProfitSummary{},
		&listingtask.ListingTask{},
		&ListingRecommendation{},
		&approval.ApprovalRequest{},
		&operationlog.OperationLog{},
	)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, false)

	categoryID := int64(1)
	brandID := int64(1)
	platformID := int64(1)

	// Candidate with purchase price far exceeding sale price → negative profit → should be "skip"
	product := candidate.CandidateProduct{
		Title:              "Negative Margin Product",
		Description:        "Purchase price exceeds sale price — always unprofitable",
		MainImage:          "https://example.test/image.jpg",
		Images:             []byte(`["https://example.test/img1.jpg"]`),
		CategoryID:         &categoryID,
		BrandID:            &brandID,
		SpecJSON:           []byte(`{"color": "red"}`),
		PurchasePrice:      500, // 500 CNY ≈ $69 — way above $30 sale price
		PurchaseCurrency:   "CNY",
		PackageWeightKg:    0.4,
		PackageLengthCm:    10,
		PackageWidthCm:     8,
		PackageHeightCm:    6,
		HSCode:             "1234.56",
		OriginCountry:      "CN",
		TargetSalePrice:    30, // $30 sale < $69+ cost → negative profit
		TargetCurrency:     "USD",
		TargetPlatformID:   &platformID,
		DestinationCountry: "RU",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	result, err := svc.Evaluate(product.ID, "A8")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.Decision != "skip" {
		t.Fatalf("decision = %s, want 'skip' for negative profit; reason=%s", result.Decision, result.Reason)
	}
	if result.Reason == "" {
		t.Fatal("expected a business-readable reason for the skip")
	}
	// No listing task should be created for skipped candidates
	if result.ListingTaskID != nil {
		t.Fatalf("expected no listing task for skip, got id=%d", *result.ListingTaskID)
	}
}

func TestEvaluate_AuditLogExists(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t,
		&candidate.CandidateProduct{},
		&completeness.CompletenessCheck{},
		&profit.ProfitSummary{},
		&listingtask.ListingTask{},
		&ListingRecommendation{},
		&approval.ApprovalRequest{},
		&operationlog.OperationLog{},
	)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, false)

	categoryID := int64(1)
	brandID := int64(1)
	platformID := int64(1)

	product := candidate.CandidateProduct{
		Title:              "Audit Log Test",
		Description:        "Complete enough to trigger list decision and audit log",
		MainImage:          "https://example.test/image.jpg",
		Images:             []byte(`["https://example.test/img1.jpg"]`),
		CategoryID:         &categoryID,
		BrandID:            &brandID,
		SpecJSON:           []byte(`{"color": "blue", "size": "L"}`),
		PurchasePrice:      10,
		PurchaseCurrency:   "CNY",
		PackageWeightKg:    0.4,
		PackageLengthCm:    10,
		PackageWidthCm:     8,
		PackageHeightCm:    6,
		HSCode:             "1234.56",
		OriginCountry:      "CN",
		TargetSalePrice:    30,
		TargetCurrency:     "USD",
		TargetPlatformID:   &platformID,
		DestinationCountry: "RU",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	result, err := svc.Evaluate(product.ID, "A8")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	// Check operation log was written
	var logs []operationlog.OperationLog
	if err := db.Where("module = ? AND action = ?", "loop", "evaluate_list").Find(&logs).Error; err != nil {
		t.Fatalf("query operation_log: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected at least one operation_log entry for the approval-gated transition")
	}
	found := false
	for _, l := range logs {
		if l.EntityType == "listing_task" && l.EntityID == *result.ListingTaskID {
			found = true
			if l.TriggerType != "agent" {
				t.Errorf("trigger_type = %s, want 'agent'", l.TriggerType)
			}
			if l.Result != "pending_approval" {
				t.Errorf("result = %s, want 'pending_approval'", l.Result)
			}
			break
		}
	}
	if !found {
		t.Fatal("no operation_log entry found matching the listing_task entity")
	}
}
