package approval

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func newTestDB(t *testing.T) *ApprovalRequest {
	t.Helper()
	return &ApprovalRequest{}
}

func TestService_Create(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	req, err := svc.Create(&CreateApprovalInput{
		ProductID:   100,
		RequestType: "publish",
		Requester:   "agent-A2",
		Reason:      "product is ready",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if req.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if req.Status != "pending" {
		t.Errorf("expected default status 'pending', got %s", req.Status)
	}
	if req.RequestType != "publish" {
		t.Errorf("expected request_type 'publish', got %s", req.RequestType)
	}
}

func TestService_Get_Found(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	created, _ := svc.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "price_change", Requester: "user1",
	})

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, got.ID)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	_, err := svc.Get(99999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestService_List_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	items, total, err := svc.List(1, 10, "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if items == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestService_List_Pagination(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	for i := 0; i < 5; i++ {
		svc.Create(&CreateApprovalInput{
			ProductID:   int64(i),
			RequestType: "publish",
			Requester:   "agent",
		})
	}

	items, total, err := svc.List(1, 2, "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
		_ = items
	}

	items, total, err = svc.List(2, 2, "", "")
	if err != nil {
		t.Fatalf("List page 2 failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
		_ = items
	}
}

func TestService_List_StatusFilter(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	svc.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "agent1"})
	svc.Create(&CreateApprovalInput{ProductID: 2, RequestType: "publish", Requester: "agent2"})

	items, total, err := svc.List(1, 10, "pending", "")
	if err != nil {
		t.Fatalf("List by status failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 pending, got %d", total)
		_ = items
	}
}

func TestService_Review_Approved(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	created, _ := svc.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "publish", Requester: "agent",
	})

	reviewed, err := svc.Review(created.ID, &ReviewApprovalInput{
		Action:     "approve",
		Reviewer:   "owner",
		ReviewNote: "looks good",
	})
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}
	if reviewed.Status != "approved" {
		t.Errorf("expected status 'approved', got %s", reviewed.Status)
	}
	if reviewed.Reviewer != "owner" {
		t.Errorf("expected reviewer 'owner', got %s", reviewed.Reviewer)
	}
}

func TestService_Review_Rejected(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	created, _ := svc.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "publish", Requester: "agent",
	})

	reviewed, err := svc.Review(created.ID, &ReviewApprovalInput{
		Action:     "reject",
		Reviewer:   "owner",
		ReviewNote: "not ready",
	})
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}
	if reviewed.Status != "rejected" {
		t.Errorf("expected status 'rejected', got %s", reviewed.Status)
	}
}

func TestService_Review_NotPending(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	created, _ := svc.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "publish", Requester: "agent",
	})
	svc.Review(created.ID, &ReviewApprovalInput{Action: "approve", Reviewer: "owner"})

	_, err := svc.Review(created.ID, &ReviewApprovalInput{Action: "reject", Reviewer: "owner"})
	if err == nil {
		t.Fatal("expected error reviewing already approved request")
	}
}

func TestService_MyPending(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	svc.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "agent1"})
	svc.Create(&CreateApprovalInput{ProductID: 2, RequestType: "publish", Requester: "agent2"})

	items, total, err := svc.MyPending(1, 10)
	if err != nil {
		t.Fatalf("MyPending failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 pending, got %d", total)
		_ = items
	}
}

func TestService_Stats(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	stats, err := svc.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.TotalCount != 0 {
		t.Errorf("expected 0 total, got %d", stats.TotalCount)
	}

	svc.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "a"})
	svc.Create(&CreateApprovalInput{ProductID: 2, RequestType: "publish", Requester: "a"})

	stats, err = svc.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.TotalCount != 2 {
		t.Errorf("expected 2 total, got %d", stats.TotalCount)
	}
	if stats.PendingCount != 2 {
		t.Errorf("expected 2 pending, got %d", stats.PendingCount)
	}
}

func TestService_HasPendingForEntity(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	has, err := svc.HasPendingForEntity("listing_task", 42)
	if err != nil {
		t.Fatalf("HasPendingForEntity: %v", err)
	}
	if has {
		t.Fatal("expected false for no pending approval")
	}

	svc.Create(&CreateApprovalInput{
		ProductID:   1,
		RequestType: "publish",
		Requester:   "agent",
		EntityType:  "listing_task",
		EntityID:    42,
	})

	has, err = svc.HasPendingForEntity("listing_task", 42)
	if err != nil {
		t.Fatalf("HasPendingForEntity: %v", err)
	}
	if !has {
		t.Fatal("expected true for existing pending approval")
	}

	has, err = svc.HasPendingForEntity("listing_task", 99)
	if err != nil {
		t.Fatalf("HasPendingForEntity: %v", err)
	}
	if has {
		t.Fatal("expected false for different entity ID")
	}
}

func TestService_EntityTypeEntityID(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	req, err := svc.Create(&CreateApprovalInput{
		ProductID:   1,
		RequestType: "publish",
		Requester:   "agent",
		EntityType:  "listing_task",
		EntityID:    100,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if req.EntityType != "listing_task" {
		t.Errorf("EntityType = %q", req.EntityType)
	}
	if req.EntityID != 100 {
		t.Errorf("EntityID = %d", req.EntityID)
	}

	got, err := svc.Get(req.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.EntityType != "listing_task" {
		t.Errorf("EntityType = %q", got.EntityType)
	}
	if got.EntityID != 100 {
		t.Errorf("EntityID = %d", got.EntityID)
	}
}

func TestService_FindApprovedByTarget(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	db.Create(&ApprovalRequest{
		ProductID:   101,
		RequestType: "publish",
		Requester:   "A8",
		Status:      "approved",
		TargetType:  "listing_task",
		TargetID:    55,
		Reason:      "资料完整且利润达标",
	})

	req, err := svc.FindApprovedByTarget("listing_task", 55, "publish")
	if err != nil {
		t.Fatalf("FindApprovedByTarget: %v", err)
	}
	if req.TargetType != "listing_task" || req.TargetID != 55 || req.Status != "approved" {
		t.Fatalf("unexpected approval: %+v", req)
	}
}

func TestService_FindApprovedByTargetRejectsPending(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil)

	db.Create(&ApprovalRequest{
		ProductID:   101,
		RequestType: "publish",
		Requester:   "A8",
		Status:      "pending",
		TargetType:  "listing_task",
		TargetID:    55,
	})

	_, err := svc.FindApprovedByTarget("listing_task", 55, "publish")
	if err == nil {
		t.Fatal("expected no approved approval for pending request")
	}
}
