package command

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
)

func TestDispatchSafeConsumesApprovalOnceAndReplaysSameKey(t *testing.T) {
	db := dbtest.NewDB(t, &ActionExecution{}, &approval.ApprovalRequest{}, &approval.ApprovalExecution{})
	approvalSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	future := time.Now().Add(time.Hour)
	req, err := approvalSvc.Create(&approval.CreateApprovalInput{ProductID: 42, RequestType: "publish", Requester: "owner", TargetType: "listing", TargetID: 42, ExpiresAt: &future})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approvalSvc.Review(req.ID, &approval.ReviewApprovalInput{Action: "approve", Reviewer: "owner"}); err != nil {
		t.Fatal(err)
	}
	checker := approval.NewApprovalPolicyChecker(approvalSvc)
	d := NewDispatcher(dbtest.NewLogger(t), WithIdempotencyStore(NewGormIdempotencyStore(db, time.Minute)))
	var calls atomic.Int32
	d.Register("auto_publish", func(context.Context, map[string]interface{}) (*Result, error) {
		calls.Add(1)
		return &Result{Success: true, BusinessID: "listing-42"}, nil
	})
	action := AgentAction{ActionType: "auto_publish", AgentID: "publisher", Actor: "owner", Mode: ModeProduction, RiskLevel: RiskHigh, ApprovalRequired: true, ApprovalID: &req.ID, IdempotencyKey: "command-publish-42", TargetType: "listing", TargetID: "42"}
	first, err := d.DispatchSafe(context.Background(), action, checker)
	if err != nil {
		t.Fatal(err)
	}
	second, err := d.DispatchSafe(context.Background(), action, checker)
	if err != nil || second.BusinessID != first.BusinessID || calls.Load() != 1 {
		t.Fatalf("first=%+v second=%+v calls=%d err=%v", first, second, calls.Load(), err)
	}
	action.IdempotencyKey = "command-publish-42-other"
	if _, err := d.DispatchSafe(context.Background(), action, checker); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("different-key approval reuse error=%v", err)
	}
}
