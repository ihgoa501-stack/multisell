package loop

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
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
