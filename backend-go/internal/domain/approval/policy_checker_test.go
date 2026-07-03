package approval

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ---------------------------------------------------------------------------
// ApprovalPolicyChecker tests.
// ---------------------------------------------------------------------------

func TestApprovalPolicyChecker_Pending_ReturnsFalse(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)

	req, err := svc.Create(&CreateApprovalInput{
		ProductID:   1,
		RequestType: "price_change",
		Requester:   "A5",
		Reason:      "test",
		TargetType:  "sku",
		TargetID:    100,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	checker := NewApprovalPolicyChecker(svc)
	if checker.IsApproved(req.ID) {
		t.Error("pending approval should not pass IsApproved")
	}
}

func TestApprovalPolicyChecker_Approved_ReturnsTrue(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)

	req, err := svc.Create(&CreateApprovalInput{
		ProductID:   1,
		RequestType: "price_change",
		Requester:   "A5",
		Reason:      "test",
		TargetType:  "sku",
		TargetID:    100,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	// Approve it.
	_, err = svc.Review(req.ID, &ReviewApprovalInput{
		Action:     "approve",
		Reviewer:   "owner",
		ReviewNote: "OK",
	})
	if err != nil {
		t.Fatalf("Review approval: %v", err)
	}

	checker := NewApprovalPolicyChecker(svc)
	if !checker.IsApproved(req.ID) {
		t.Error("approved approval should pass IsApproved")
	}
}

func TestApprovalPolicyChecker_Rejected_ReturnsFalse(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)

	req, err := svc.Create(&CreateApprovalInput{
		ProductID:   1,
		RequestType: "price_change",
		Requester:   "A5",
		TargetType:  "sku",
		TargetID:    100,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	_, err = svc.Review(req.ID, &ReviewApprovalInput{
		Action:   "reject",
		Reviewer: "owner",
	})
	if err != nil {
		t.Fatalf("Review approval: %v", err)
	}

	checker := NewApprovalPolicyChecker(svc)
	if checker.IsApproved(req.ID) {
		t.Error("rejected approval should not pass IsApproved")
	}
}

func TestApprovalPolicyChecker_Nonexistent_ReturnsFalse(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)
	checker := NewApprovalPolicyChecker(svc)

	if checker.IsApproved(99999) {
		t.Error("nonexistent approval should not pass IsApproved")
	}
}

func TestApprovalPolicyChecker_Expired_ReturnsFalse(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)

	past := time.Now().Add(-1 * time.Hour)
	req, err := svc.Create(&CreateApprovalInput{
		ProductID:   1,
		RequestType: "price_change",
		Requester:   "A5",
		Reason:      "test",
		TargetType:  "sku",
		TargetID:    100,
		ExpiresAt:   &past,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	// Approve — but expiry is already in the past.
	_, err = svc.Review(req.ID, &ReviewApprovalInput{
		Action:   "approve",
		Reviewer: "owner",
	})
	if err != nil {
		t.Fatalf("Review approval: %v", err)
	}

	checker := NewApprovalPolicyChecker(svc)
	if checker.IsApproved(req.ID) {
		t.Error("expired approval should not pass IsApproved")
	}
}
