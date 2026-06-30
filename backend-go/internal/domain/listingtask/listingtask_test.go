package listingtask

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
		"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"time"
)

func int64Ptr(v int64) *int64 { return &v }

func TestService_Task_CreateAndGetByID(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, err := svc.Create(&CreateTaskInput{
		ProductID:         1,
		PlatformID:        10,
		SourceType:        "decision",
		DestinationCountry: "US",
		CreatedBy:         "tester",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.ID == 0 {
		t.Fatal("ID should be set")
	}
	if task.ProductID != 1 {
		t.Fatalf("ProductID = %d", task.ProductID)
	}
	if task.Status != "blocked" {
		t.Fatalf("Status = %s", task.Status)
	}

	got, items, err := svc.GetByID(task.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != task.ID {
		t.Fatal("ID mismatch")
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestService_Task_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	updated, err := svc.Update(task.ID, &UpdateTaskInput{
		Status:    dbtest.StringPtr("cancelled"),
		UpdatedBy: dbtest.StringPtr("admin"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "cancelled" {
		t.Fatalf("Status = %s", updated.Status)
	}
}

func TestService_Task_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "blocked"})
	svc.Create(&CreateTaskInput{ProductID: 2, PlatformID: 10, Status: "ready"})
	svc.Create(&CreateTaskInput{ProductID: 3, PlatformID: 20, Status: "completed"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, nil, nil, "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	items, total, err = svc.List(&p, int64Ptr(10), nil, "", "")
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 for platform 10, got %d", total)
	}
}

func TestService_Task_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	if err := svc.Delete(task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, _, err := svc.GetByID(task.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Task_Execute(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "pending"})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 1, PlatformID: 10})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 2, PlatformID: 10})

	// Create an approved approval so the gate passes
	now := time.Now()
	future := now.Add(24 * time.Hour)
	db.Create(&approval.ApprovalRequest{
		ProductID: 1, RequestType: "publish", Requester: "test",
		Status: "approved", ExpiresAt: &future,
	})

	executed, err := svc.ExecuteTask(task.ID)
	if err != nil {
		t.Fatalf("ExecuteTask: %v", err)
	}
	if executed.Status != "completed" {
		t.Fatalf("Status = %s (expected completed)", executed.Status)
	}
}

func TestService_Task_RetryFailed(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "failed"})
	errMsg := "timeout"
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 1, PlatformID: 10, Status: "failed"})
	svc.UpdateItem(1, &UpdateTaskItemInput{ErrorMessage: &errMsg})

	retried, err := svc.RetryFailed(task.ID)
	if err != nil {
		t.Fatalf("RetryFailed: %v", err)
	}
	if retried.Status != "pending" {
		t.Fatalf("Status = %s (expected pending)", retried.Status)
	}

	item, _ := svc.GetItem(1)
	if item.Status != "pending" {
		t.Fatalf("Item Status = %s (expected pending)", item.Status)
	}
}

// ── ListingTaskItem ──

func TestService_Item_CreateGetUpdateDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})

	item, err := svc.CreateItem(&CreateTaskItemInput{
		TaskID:     task.ID,
		ProductID:  1,
		PlatformID: 10,
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if item.ID == 0 {
		t.Fatal("Item ID should be set")
	}
	if item.Status != "pending" {
		t.Fatalf("Status = %s", item.Status)
	}

	got, err := svc.GetItem(item.ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if got.TaskID != task.ID {
		t.Fatal("TaskID mismatch")
	}

	status := "completed"
	updated, err := svc.UpdateItem(item.ID, &UpdateTaskItemInput{Status: &status})
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("Status = %s", updated.Status)
	}

	if err := svc.DeleteItem(item.ID); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	_, err = svc.GetItem(item.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Item_ListItems(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 1, PlatformID: 10})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 2, PlatformID: 10})

	p := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListItems(&p, task.ID)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	_ = items
}


// ── ListingTask State Machine ──

func TestService_StateMachine_ValidTransitions(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})

	// blocked -> pending
	updated, err := svc.Update(task.ID, &UpdateTaskInput{Status: dbtest.StringPtr("pending")})
	if err != nil {
		t.Fatalf("blocked->pending: %v", err)
	}
	if updated.Status != "pending" {
		t.Fatalf("expected pending, got %s", updated.Status)
	}

	// pending -> cancelled
	updated, err = svc.Update(task.ID, &UpdateTaskInput{Status: dbtest.StringPtr("cancelled")})
	if err != nil {
		t.Fatalf("pending->cancelled: %v", err)
	}
	if updated.Status != "cancelled" {
		t.Fatalf("expected cancelled, got %s", updated.Status)
	}
}

func TestService_StateMachine_InvalidTransitions(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})

	// blocked -> completed (invalid)
	_, err := svc.Update(task.ID, &UpdateTaskInput{Status: dbtest.StringPtr("completed")})
	if err == nil {
		t.Fatal("blocked->completed should be rejected")
	}

	// blocked -> executing (invalid)
	_, err = svc.Update(task.ID, &UpdateTaskInput{Status: dbtest.StringPtr("executing")})
	if err == nil {
		t.Fatal("blocked->executing should be rejected")
	}

	// pending -> completed (invalid)
	svc.Update(task.ID, &UpdateTaskInput{Status: dbtest.StringPtr("pending")})
	_, err = svc.Update(task.ID, &UpdateTaskInput{Status: dbtest.StringPtr("completed")})
	if err == nil {
		t.Fatal("pending->completed should be rejected")
	}
}

func TestService_StateMachine_TerminalStatus(t *testing.T) {
	t.Parallel()
	_ = dbtest.NewDB(t, &ListingTask{})

	sm := NewListingTaskStateMachine()

	if !sm.IsTerminal("completed") {
		t.Fatal("completed should be terminal")
	}
	if !sm.IsTerminal("cancelled") {
		t.Fatal("cancelled should be terminal")
	}
	if sm.IsTerminal("pending") {
		t.Fatal("pending should not be terminal")
	}
	if sm.IsTerminal("blocked") {
		t.Fatal("blocked should not be terminal")
	}
	if sm.IsTerminal("executing") {
		t.Fatal("executing should not be terminal")
	}
	if sm.IsTerminal("failed") {
		t.Fatal("failed should not be terminal")
	}
}

func TestService_StateMachine_ExecuteFromBlocked(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	// Create items so ExecuteTask has something to process
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 1, PlatformID: 10})

	// ExecuteTask from blocked status should fail (blocked cannot transition to executing)
	_, err := svc.ExecuteTask(task.ID)
	if err == nil {
		t.Fatal("ExecuteTask from blocked should fail")
	}
}

// ── Agent Feedback ──

func TestService_SubmitFeedback_Accepted(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	updated, err := svc.SubmitFeedback(task.ID, "accepted", "good suggestion", "user-1")
	if err != nil {
		t.Fatalf("SubmitFeedback: %v", err)
	}
	if updated.AgentFeedbackStatus == nil || *updated.AgentFeedbackStatus != "accepted" {
		t.Fatalf("expected accepted, got %v", updated.AgentFeedbackStatus)
	}
	if updated.AgentFeedbackNote != "good suggestion" {
		t.Fatalf("expected 'good suggestion', got %s", updated.AgentFeedbackNote)
	}
}

func TestService_SubmitFeedback_Rejected(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	updated, err := svc.SubmitFeedback(task.ID, "rejected", "price too high", "user-2")
	if err != nil {
		t.Fatalf("SubmitFeedback: %v", err)
	}
	if *updated.AgentFeedbackStatus != "rejected" {
		t.Fatalf("expected rejected, got %v", *updated.AgentFeedbackStatus)
	}
}

func TestService_SubmitFeedback_InvalidStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	_, err := svc.SubmitFeedback(task.ID, "invalid", "", "user-1")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestService_SubmitFeedback_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, false)

	_, err := svc.SubmitFeedback(99999, "accepted", "", "user-1")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}
