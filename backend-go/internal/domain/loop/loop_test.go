package loop

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_GetRecommendations(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingRecommendation{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil)

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
}

func TestService_GetRecommendations_Filtered(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingRecommendation{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil)

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


func TestService_RecommendationStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingRecommendation{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil)

	// Create a recommendation via DB directly
	rec := ListingRecommendation{
		ProductID: 1, Decision: "list", Confidence: 0.85,
		Reason: "good product", Status: RecStatusPending,
	}
	db.Create(&rec)

	// Accept it
	updated, err := svc.UpdateRecommendationStatus(rec.ID, RecStatusAccepted)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if updated.Status != RecStatusAccepted {
		t.Fatalf("expected accepted, got %s", updated.Status)
	}

	// Rejecting an accepted one should fail (terminal state)
	_, err = svc.UpdateRecommendationStatus(rec.ID, RecStatusRejected)
	if err == nil {
		t.Fatal("expected error rejecting accepted (terminal)")
	}

	// State machine unit checks
	sm := NewRecommendationStateMachine()
	if !sm.CanTransition(RecStatusPending, RecStatusAccepted) {
		t.Error("pending->accepted should be valid")
	}
	if sm.CanTransition(RecStatusAccepted, RecStatusRejected) {
		t.Error("accepted->rejected should be invalid")
	}
	if !sm.IsTerminal(RecStatusAccepted) {
		t.Error("accepted should be terminal")
	}
	if sm.IsTerminal(RecStatusPending) {
		t.Error("pending should not be terminal")
	}
}
