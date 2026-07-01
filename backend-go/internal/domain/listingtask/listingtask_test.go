package listingtask

import (
	"strings"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
)

func int64Ptr(v int64) *int64 { return &v }

func TestService_Task_CreateAndGetByID(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, err := svc.Create(&CreateTaskInput{
		ProductID:          1,
		PlatformID:         10,
		SourceType:         "decision",
		DestinationCountry: "US",
		CreatedBy:          "tester",
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
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	updated, err := svc.Update(task.ID, &UpdateTaskInput{
		Status:    dbtest.StringPtr("ready"),
		UpdatedBy: dbtest.StringPtr("admin"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "ready" {
		t.Fatalf("Status = %s", updated.Status)
	}
}

func TestService_Task_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})
	if err := svc.Delete(task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, _, err := svc.GetByID(task.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

// TestService_Task_Execute tests the full happy path through the execution gate.
func TestService_Task_Execute(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), nil, false, apprSvc, nil, nil)

	// Create task in blocked state
	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "blocked"})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 1, PlatformID: 10})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 2, PlatformID: 10})

	// Create an approved approval record for this task
	apprRec, _ := apprSvc.Create(&approval.CreateApprovalInput{
		ProductID:   1,
		RequestType: "publish",
		Requester:   "agent",
		EntityType:  "listing_task",
		EntityID:    task.ID,
	})
	apprSvc.Review(apprRec.ID, &approval.ReviewApprovalInput{
		Action:   "approve",
		Reviewer: "owner",
	})

	// Update task: status → approved, set approval_id
	task, _ = svc.Update(task.ID, &UpdateTaskInput{
		Status:     dbtest.StringPtr("approved"),
		ApprovalID: &apprRec.ID,
	})

	// Execute
	executed, err := svc.ExecuteTask(task.ID, "tester")
	if err != nil {
		t.Fatalf("ExecuteTask: %v", err)
	}
	if executed.Status != "completed" {
		t.Fatalf("Status = %s (expected completed)", executed.Status)
	}
}

// TestService_Task_Execute_BlockedCannotExecute tests that a blocked task cannot be executed.
func TestService_Task_Execute_BlockedCannotExecute(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "blocked"})
	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error executing blocked task")
	}
}

// TestService_Task_Execute_PendingApprovalCannotExecute tests pending_approval cannot execute.
func TestService_Task_Execute_PendingApprovalCannotExecute(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "pending_approval"})
	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error executing pending_approval task")
	}
}

// TestService_Task_Execute_NoApprovalID tests that a task without approval_id cannot execute.
func TestService_Task_Execute_NoApprovalID(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "approved"})
	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error executing task without approval_id")
	}
}

// TestService_Task_Execute_ApprovalNotApproved tests that rejected approval blocks execution.
func TestService_Task_Execute_ApprovalNotApproved(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), nil, false, apprSvc, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "approved"})

	// Create a rejected approval
	apprRec, _ := apprSvc.Create(&approval.CreateApprovalInput{
		ProductID:   1,
		RequestType: "publish",
		Requester:   "agent",
		EntityType:  "listing_task",
		EntityID:    task.ID,
	})
	apprSvc.Review(apprRec.ID, &approval.ReviewApprovalInput{
		Action:   "reject",
		Reviewer: "owner",
	})

	task, _ = svc.Update(task.ID, &UpdateTaskInput{
		ApprovalID: &apprRec.ID,
	})

	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error executing task with rejected approval")
	}
}

// TestService_Task_Execute_ApprovalSvcNotInjected tests handling when approvalSvc is nil.
func TestService_Task_Execute_ApprovalSvcNotInjected(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{
		ProductID:  1,
		PlatformID: 10,
		Status:     "approved",
		ApprovalID: int64Ptr(1),
	})
	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error when approval service not injected")
	}
}

// TestService_Task_Execute_AlreadyCompleted tests idempotency on completed tasks.
func TestService_Task_Execute_AlreadyCompleted(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "completed"})
	result, err := svc.ExecuteTask(task.ID, "tester")
	if err != nil {
		t.Fatalf("unexpected error on re-execute completed: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("Status = %s (expected completed)", result.Status)
	}
}

// TestService_Task_Execute_AlreadyExecuting tests idempotency on executing tasks.
func TestService_Task_Execute_AlreadyExecuting(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "executing"})
	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error on re-execute executing task")
	}
}

// TestService_Task_RetryFailed tests the retry flow.
func TestService_Task_RetryFailed(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "failed"})
	errMsg := "timeout"
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 1, PlatformID: 10, Status: "failed"})
	// The first created item has ID=1 because dbtest.NewDB starts with empty tables
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

// TestService_StateMachine tests the state machine transition definitions.
func TestService_StateMachine(t *testing.T) {
	sm := NewListingTaskStateMachine()

	tests := []struct {
		from  string
		to    string
		valid bool
		desc  string
	}{
		{"blocked", "pending_approval", true, "blocked → pending_approval"},
		{"pending_approval", "approved", true, "pending_approval → approved"},
		{"pending_approval", "rejected", true, "pending_approval → rejected"},
		{"approved", "executing", true, "approved → executing"},
		{"executing", "completed", true, "executing → completed"},
		{"executing", "failed", true, "executing → failed"},
		{"failed", "pending_approval", true, "failed → pending_approval"},
		{"blocked", "executing", false, "blocked → executing (forbidden)"},
		{"pending_approval", "executing", false, "pending_approval → executing (forbidden)"},
		{"blocked", "completed", false, "blocked → completed (forbidden)"},
		{"completed", "executing", false, "completed → executing (forbidden)"},
		{"cancelled", "executing", false, "cancelled → executing (forbidden)"},
		{"rejected", "executing", false, "rejected → executing (forbidden)"},
		{"blocked", "blocked", false, "blocked → blocked (no self-loop)"},
	}

	for _, tt := range tests {
		got := sm.CanTransition(tt.from, tt.to)
		if got != tt.valid {
			t.Errorf("StateMachine: %s: got %v, want %v", tt.desc, got, tt.valid)
		}
	}
}

// TestService_Task_Execute_FailedCannotExecute tests that a failed task cannot execute directly.
func TestService_Task_Execute_FailedCannotExecute(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "failed"})
	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error executing failed task")
	}
}

// TestService_Task_Execute_RejectedCannotExecute tests that a rejected task cannot execute.
func TestService_Task_Execute_RejectedCannotExecute(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "rejected"})
	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error executing rejected task")
	}
}

// TestService_ApprovalID_IsStored tests that the ApprovalID field is persisted.
func TestService_ApprovalID_IsStored(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	apprID := int64Ptr(999)
	task, _ := svc.Create(&CreateTaskInput{
		ProductID:  1,
		PlatformID: 10,
		ApprovalID: apprID,
	})
	if task.ApprovalID == nil {
		t.Fatal("ApprovalID should be set")
	}
	if *task.ApprovalID != 999 {
		t.Fatalf("ApprovalID = %d (expected 999)", *task.ApprovalID)
	}

	got, _, _ := svc.GetByID(task.ID)
	if got.ApprovalID == nil {
		t.Fatal("ApprovalID should be persisted")
	}
	if *got.ApprovalID != 999 {
		t.Fatalf("ApprovalID = %d (expected 999)", *got.ApprovalID)
	}
}

// TestService_Task_Execute_ApprovalWrongEntityType tests that mismatched entity_type blocks execution.
func TestService_Task_Execute_ApprovalWrongEntityType(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), nil, false, apprSvc, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "approved"})

	// Create approval with WRONG entity_type
	apprRec, _ := apprSvc.Create(&approval.CreateApprovalInput{
		ProductID:   1,
		RequestType: "publish",
		Requester:   "agent",
		EntityType:  "some_other_type",
		EntityID:    task.ID,
	})
	apprSvc.Review(apprRec.ID, &approval.ReviewApprovalInput{
		Action:   "approve",
		Reviewer: "owner",
	})
	task, _ = svc.Update(task.ID, &UpdateTaskInput{
		ApprovalID: &apprRec.ID,
	})

	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error for wrong entity_type")
	}
}

// TestService_Task_Execute_ApprovalWrongEntityID tests that mismatched entity_id blocks execution.
func TestService_Task_Execute_ApprovalWrongEntityID(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), nil, false, apprSvc, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "approved"})

	// Create approval with WRONG entity_id
	apprRec, _ := apprSvc.Create(&approval.CreateApprovalInput{
		ProductID:   1,
		RequestType: "publish",
		Requester:   "agent",
		EntityType:  "listing_task",
		EntityID:    99999, // wrong entity_id
	})
	apprSvc.Review(apprRec.ID, &approval.ReviewApprovalInput{
		Action:   "approve",
		Reviewer: "owner",
	})
	task, _ = svc.Update(task.ID, &UpdateTaskInput{
		ApprovalID: &apprRec.ID,
	})

	_, err := svc.ExecuteTask(task.ID, "tester")
	if err == nil {
		t.Fatal("expected error for wrong entity_id")
	}
}

// ── ListingTaskItem ──

func TestService_Item_CreateGetUpdateDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

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

func TestService_ExecuteTaskBlocksWithoutApproval(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false, nil, nil, nil)

	task := ListingTask{ProductID: 10, PlatformID: 1, Status: "blocked", CreatedBy: "A8"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, err := svc.ExecuteTask(task.ID, "A8")
	if err == nil {
		t.Fatal("expected error for unapproved task")
	}
	if !strings.Contains(err.Error(), "cannot be executed from status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_ExecuteTaskAllowsApprovedBlockedTask(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), nil, false, apprSvc, nil, nil)

	task := ListingTask{ProductID: 10, PlatformID: 1, Status: "approved", CreatedBy: "A8"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	item := ListingTaskItem{TaskID: task.ID, ProductID: 10, PlatformID: 1, Status: "pending"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	// Directly create an approved approval request.
	req := approval.ApprovalRequest{
		ProductID:   10,
		RequestType: "publish",
		Requester:   "A8",
		Status:      "approved",
		EntityType:  "listing_task",
		EntityID:    task.ID,
		RiskLevel:   "high",
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create approval: %v", err)
	}
	db.Model(&task).Update("approval_id", req.ID)

	updated, err := svc.ExecuteTask(task.ID, "A8")
	if err != nil {
		t.Fatalf("ExecuteTask: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("status = %s, want completed", updated.Status)
	}
}
