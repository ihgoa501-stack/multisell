package approval

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"gorm.io/gorm"
)

// createUnifiedActionTable creates the unified_action table via raw SQL so
// the approval test package doesn't need to import the ai package's model.
func createUnifiedActionTable(tx *gorm.DB) {
	createUA := `CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_table TEXT NOT NULL DEFAULT '',
		source_id TEXT NOT NULL DEFAULT '',
		source_type TEXT NOT NULL DEFAULT '',
		trace_id TEXT DEFAULT '',
		agent_id TEXT DEFAULT '',
		squad_id TEXT DEFAULT '',
		action_type TEXT NOT NULL DEFAULT '',
		business_object_type TEXT DEFAULT '',
		business_object_id TEXT DEFAULT '',
		title TEXT NOT NULL DEFAULT '',
		description TEXT DEFAULT '',
		payload TEXT DEFAULT '{}',
		before_snapshot TEXT DEFAULT '{}',
		after_snapshot TEXT DEFAULT '{}',
		risk_level TEXT DEFAULT 'medium',
		requires_approval INTEGER DEFAULT 0,
		status TEXT DEFAULT 'suggested',
		confidence REAL,
		proposed_by TEXT DEFAULT '',
		approved_by TEXT DEFAULT '',
		approved_at TIMESTAMP,
		rejected_by TEXT DEFAULT '',
		rejected_at TIMESTAMP,
		rejection_reason TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if err := tx.Exec(createUA).Error; err != nil {
		panic("create unified_action table: " + err.Error())
	}
}

// insertUnifiedActionRow inserts a test unified_action row with default values.
func insertUnifiedActionRow(tx *gorm.DB, id int64, title string) {
	insertUA := `INSERT INTO unified_action (
		id, source_table, source_id, source_type, agent_id, action_type,
		title, risk_level, requires_approval, status, proposed_by
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if err := tx.Exec(insertUA,
		id, "ai_trace", "trc_test_appr_"+title, "agent_run", "A5", "stock_alert",
		title, "medium", 1, "suggested", "agent:A5",
	).Error; err != nil {
		panic("insert unified_action: " + err.Error())
	}
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

// Test 1: Direct action-approval linkage — verify approving an
// approval_request with entity_type="unified_action" syncs the linked
// unified_action.status to "approved". Uses raw SQL to avoid circular import.
func TestService_Review_SyncsUnifiedAction_Approved(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ApprovalRequest{})
	createUnifiedActionTable(db)
	insertUnifiedActionRow(db, 1001, "Test Action")

	svc := NewService(db, dbtest.NewLogger(t), nil)

	req, err := svc.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "unified_action", Requester: "agent:A5",
		EntityType: "unified_action", EntityID: 1001,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	_, err = svc.Review(req.ID, &ReviewApprovalInput{
		Action: "approve", Reviewer: "owner", ReviewNote: "approved by owner",
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	var status string
	db.Table("unified_action").Select("status").Where("id = ?", 1001).Scan(&status)
	if status != "approved" {
		t.Errorf("unified_action status = %q, want %q", status, "approved")
	}
}

// Test 2: Rejection sync — verify rejecting an approval_request with
// entity_type="unified_action" syncs the linked unified_action.status to "rejected".
func TestService_Review_SyncsUnifiedAction_Rejected(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ApprovalRequest{})
	createUnifiedActionTable(db)
	insertUnifiedActionRow(db, 1002, "Test Action Reject")

	svc := NewService(db, dbtest.NewLogger(t), nil)

	req, err := svc.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "unified_action", Requester: "agent:A5",
		EntityType: "unified_action", EntityID: 1002,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	_, err = svc.Review(req.ID, &ReviewApprovalInput{
		Action: "reject", Reviewer: "owner", ReviewNote: "not ready",
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	var status string
	db.Table("unified_action").Select("status").Where("id = ?", 1002).Scan(&status)
	if status != "rejected" {
		t.Errorf("unified_action status = %q, want %q", status, "rejected")
	}
}


func TestService_Review_PublishesApprovalEvent(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	createUnifiedActionTable(db)
	insertUnifiedActionRow(db, 2001, "Event Test Action")

	bus := eventbus.New(dbtest.NewLogger(t))
	busCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(busCtx)

	got := make(chan eventbus.Event, 10)
	bus.Subscribe("approval.approved.listing_task", func(ctx context.Context, evt eventbus.Event) error {
		got <- evt
		return nil
	})

	svc := NewService(db, dbtest.NewLogger(t), nil).WithBus(bus)

	req, err := svc.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "listing_task", Requester: "agent:A5",
		EntityType: "unified_action", EntityID: 2001,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = svc.Review(req.ID, &ReviewApprovalInput{
		Action: "approve", Reviewer: "owner",
		ReviewNote: "approved by owner", ReviewerUserID: int64Ptr(42),
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	// Verify event was published with correct payload.
	select {
	case evt := <-got:
		if evt.Topic != "approval.approved.listing_task" {
			t.Errorf("topic = %q", evt.Topic)
		}
		pl := evt.Payload
		if pl == nil {
			t.Fatal("nil payload")
		}
		// In-process bus preserves Go types: int64 for scalars, *int64 for pointer fields.
		if v, ok := pl["product_id"].(int64); !ok || v != 1 {
			t.Errorf("product_id type/value = %T(%v)", pl["product_id"], pl["product_id"])
		}
		if v, ok := pl["approval_id"].(int64); !ok || v != req.ID {
			t.Errorf("approval_id type/value = %T(%v)", pl["approval_id"], pl["approval_id"])
		}
		// reviewer_user_id can be int64 or *int64 depending on the value.
		switch v := pl["reviewer_user_id"].(type) {
		case int64:
			if v != 42 {
				t.Errorf("reviewer_user_id = %d, want 42", v)
			}
		case *int64:
			if v == nil || *v != 42 {
				t.Errorf("reviewer_user_id = %v, want 42", v)
			}
		default:
			t.Errorf("reviewer_user_id unexpected type: %T(%v)", pl["reviewer_user_id"], pl["reviewer_user_id"])
		}
	case <-make(chan struct{}):
		t.Fatal("timed out waiting for approval event")
	}
}

func int64Ptr(n int64) *int64 { return &n }
