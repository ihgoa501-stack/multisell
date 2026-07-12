package approval

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func approvedExecutionFixture(t *testing.T) (*Service, *ApprovalRequest) {
	t.Helper()
	db := dbtest.NewDB(t, &ApprovalRequest{}, &ApprovalExecution{})
	svc := NewService(db, dbtest.NewLogger(t), nil)
	future := time.Now().Add(time.Hour)
	req, err := svc.Create(&CreateApprovalInput{ProductID: 42, RequestType: "publish", Requester: "owner", TargetType: "listing", TargetID: 42, ExpiresAt: &future})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(req.ID, &ReviewApprovalInput{Action: "approve", Reviewer: "owner"}); err != nil {
		t.Fatal(err)
	}
	return svc, req
}

func TestApprovalExecutionSuccessAllowsReplayButNotReexecution(t *testing.T) {
	svc, req := approvedExecutionFixture(t)
	ctx := context.Background()
	key := "publish:listing:42"
	if err := svc.ConsumeExecution(ctx, req.ID, "auto_publish", "listing", "42", key); err != nil {
		t.Fatal(err)
	}
	if err := svc.CompleteExecution(ctx, req.ID, key); err != nil {
		t.Fatal(err)
	}
	if err := svc.AuthorizeExecution(ctx, req.ID, "auto_publish", "listing", "42", key); err != nil {
		t.Fatalf("same-key result replay was not authorized: %v", err)
	}
	if err := svc.ConsumeExecution(ctx, req.ID, "auto_publish", "listing", "42", key); !errors.Is(err, ErrApprovalConsumed) {
		t.Fatalf("succeeded execution was consumed again: %v", err)
	}
	if err := svc.AuthorizeExecution(ctx, req.ID, "auto_publish", "listing", "42", "different-key"); !errors.Is(err, ErrApprovalConsumed) {
		t.Fatalf("different key reused approval: %v", err)
	}
}

func TestApprovalExecutionFailureRetriesOnlySameBinding(t *testing.T) {
	svc, req := approvedExecutionFixture(t)
	ctx := context.Background()
	key := "publish:retry:42"
	if err := svc.ConsumeExecution(ctx, req.ID, "auto_publish", "listing", "42", key); err != nil {
		t.Fatal(err)
	}
	if err := svc.FailExecution(ctx, req.ID, key, errors.New("provider timeout")); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConsumeExecution(ctx, req.ID, "auto_publish", "listing", "42", key); err != nil {
		t.Fatalf("same binding could not retry: %v", err)
	}
	if err := svc.FailExecution(ctx, req.ID, key, errors.New("still down")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ action, targetType, targetID, key string }{
		{"auto_publish", "listing", "42", "other-key"},
		{"price_update", "listing", "42", key},
		{"auto_publish", "product", "42", key},
		{"auto_publish", "listing", "43", key},
	} {
		if err := svc.AuthorizeExecution(ctx, req.ID, tc.action, tc.targetType, tc.targetID, tc.key); err == nil {
			t.Fatalf("unrelated binding authorized: %+v", tc)
		}
	}
}

func TestApprovalExecutionConcurrentDifferentKeysOnlyOneWins(t *testing.T) {
	svc, req := approvedExecutionFixture(t)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, key := range []string{"concurrent-a", "concurrent-b"} {
		key := key
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- svc.ConsumeExecution(context.Background(), req.ID, "auto_publish", "listing", "42", key)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent consumers=%d, want 1", successes)
	}
}

func TestApprovalExecutionSupportsCompositeHTTPTarget(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{}, &ApprovalExecution{})
	svc := NewService(db, dbtest.NewLogger(t), nil)
	req := &ApprovalRequest{ProductID: 42, RequestType: "publish", Requester: "owner", Status: StatusApproved, TargetType: "listing", TargetID: 3}
	if err := db.Create(req).Error; err != nil {
		t.Fatal(err)
	}
	target := "product=42;target=3"
	if err := svc.ConsumeExecution(context.Background(), req.ID, "auto_publish", "listing", target, "composite-publish-42-3"); err != nil {
		t.Fatalf("composite target rejected: %v", err)
	}
	if err := svc.AuthorizeExecution(context.Background(), req.ID, "auto_publish", "listing", "product=42;target=4", "composite-publish-42-3"); err == nil {
		t.Fatal("mismatched composite target was authorized")
	}
}
