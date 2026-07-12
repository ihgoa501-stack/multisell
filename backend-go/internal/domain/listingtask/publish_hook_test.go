package listingtask

import (
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"go.uber.org/zap"
)

func TestPublishHookRejectsProductionWithoutBoundApproval(t *testing.T) {
	db := dbtest.NewDB(t, &ListingTask{}, &approval.ApprovalRequest{})
	task := ListingTask{ProductID: 1, PlatformID: 1, Status: "approved"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	hook := NewPublishHook(db, nil, zap.NewNop())
	err := hook(task.ID, ExecutionModeProduction)
	if err == nil || !strings.Contains(err.Error(), "requires a bound approval") {
		t.Fatalf("error = %v", err)
	}
}

func TestPublishHookRejectsExpiredApprovalBeforeAdapter(t *testing.T) {
	db := dbtest.NewDB(t, &ListingTask{}, &approval.ApprovalRequest{})
	task := ListingTask{ProductID: 1, PlatformID: 1, Status: "approved"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Minute)
	req := approval.ApprovalRequest{
		ProductID: 1, RequestType: "listing_task", Requester: "owner", Status: approval.StatusApproved,
		EntityType: "listing_task", EntityID: task.ID, ExpiresAt: &past,
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&task).Update("approval_id", req.ID).Error; err != nil {
		t.Fatal(err)
	}
	task.ApprovalID = &req.ID
	hook := NewPublishHook(db, nil, zap.NewNop())
	err := hook(task.ID, ExecutionModeApprovalRequired)
	if err == nil || !strings.Contains(err.Error(), "expired or not bound") {
		t.Fatalf("error = %v", err)
	}
}
