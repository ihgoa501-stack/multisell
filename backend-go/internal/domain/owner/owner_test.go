package owner

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/loop"
)

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
