package loop

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
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

	// New fields should default to "pending" and empty
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

func TestService_EvaluateCreatesBlockedListingTaskAndApproval(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t,
		&candidate.CandidateProduct{},
		&completeness.CompletenessCheck{},
		&profit.ProfitSummary{},
		&listingtask.ListingTask{},
		&ListingRecommendation{},
		&approval.ApprovalRequest{},
	)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, false)

	categoryID := int64(1)
	brandID := int64(1)
	platformID := int64(1)

	product := candidate.CandidateProduct{
		Title:             "Test Candidate",
		Description:       "Complete candidate product for testing purposes",
		MainImage:         "https://example.test/image.jpg",
		Images:            []byte(`["https://example.test/img1.jpg"]`),
		CategoryID:        &categoryID,
		BrandID:           &brandID,
		SpecJSON:          []byte(`{"color": "red", "size": "M"}`),
		PurchasePrice:     10,
		PurchaseCurrency:  "CNY",
		PackageWeightKg:   0.4,
		PackageLengthCm:   10,
		PackageWidthCm:    8,
		PackageHeightCm:   6,
		HSCode:            "1234.56",
		OriginCountry:     "CN",
		TargetSalePrice:   30,
		TargetCurrency:    "USD",
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
func TestService_RecordExecutionResult_Success(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingRecommendation{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

	listingTaskID := int64(100)
	db.Create(&ListingRecommendation{
		ProductID:          1,
		Decision:           "list",
		CreatedListingTaskID: &listingTaskID,
		FeedbackStatus:     "adopted",
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
		ProductID:          2,
		Decision:           "list",
		CreatedListingTaskID: &listingTaskID,
		FeedbackStatus:     "adopted",
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
