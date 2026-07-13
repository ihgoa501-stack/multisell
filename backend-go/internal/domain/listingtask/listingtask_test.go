package listingtask

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/platform"
)

var errPublishHookFailed = errors.New("publish hook failed")

func int64Ptr(v int64) *int64 { return &v }

func TestService_Task_CreateAndGetByID(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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

func TestService_ApplyOwnerApprovalCommitsStateAndAuditTogether(t *testing.T) {
	db := dbtest.NewDB(t, &ListingTask{}, &operationlog.OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t), nil, operationlog.NewService(db, dbtest.NewLogger(t)), nil)
	task, err := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "pending_approval"})
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.ApplyOwnerApproval(task.ID, 91, "owner:7"); err != nil {
		t.Fatal(err)
	}
	got, _, err := svc.GetByID(task.ID)
	if err != nil || got.Status != "approved" || got.ApprovalID == nil || *got.ApprovalID != 91 {
		t.Fatalf("approved task=%+v err=%v", got, err)
	}
	var count int64
	if err = db.Model(&operationlog.OperationLog{}).Where("action=? AND entity_id=? AND result=?", "listing_task.approval_approved", task.ID, "approved").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("durable approval audit count=%d err=%v", count, err)
	}
}

func TestService_ApplyOwnerApprovalRollsBackWithoutAuditStore(t *testing.T) {
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)
	task, err := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "pending_approval"})
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.ApplyOwnerApproval(task.ID, 91, "owner:7"); err == nil {
		t.Fatal("approval transition must fail when its audit cannot be stored")
	}
	got, _, err := svc.GetByID(task.ID)
	if err != nil || got.Status != "pending_approval" {
		t.Fatalf("state changed without audit: task=%+v err=%v", got, err)
	}
}

func TestService_Task_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

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
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

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

// TestService_ExecuteTask_PublishHookNil confirms nil publishHook has zero effect.
func TestService_ExecuteTask_PublishHookNil(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

	task := ListingTask{ProductID: 10, PlatformID: 1, Status: "approved", CreatedBy: "tester"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := db.Create(&ListingTaskItem{TaskID: task.ID, ProductID: 10, PlatformID: 1, Status: "pending"}).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	req := approval.ApprovalRequest{
		ProductID: 10, RequestType: "publish", Requester: "tester",
		Status: "approved", EntityType: "listing_task", EntityID: task.ID, RiskLevel: "high",
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create approval: %v", err)
	}
	db.Model(&task).Update("approval_id", req.ID)

	updated, err := svc.ExecuteTask(task.ID, "tester")
	if err != nil {
		t.Fatalf("ExecuteTask: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("status = %s, want completed", updated.Status)
	}
	if updated.LastError != "" {
		t.Fatalf("last_error = %q, want empty", updated.LastError)
	}
}

// TestService_ExecuteTask_MockDoesNotCallPublishHook verifies the default
// dry-run mode is an explicit mock and never reaches the legacy write hook.
func TestService_ExecuteTask_MockDoesNotCallPublishHook(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	var hookCalled bool
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)
	svc.publishHook = func(taskID int64, mode ExecutionMode) error {
		hookCalled = true
		return nil
	}

	task := ListingTask{ProductID: 10, PlatformID: 1, Status: "approved", CreatedBy: "tester"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := db.Create(&ListingTaskItem{TaskID: task.ID, ProductID: 10, PlatformID: 1, Status: "pending"}).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	req := approval.ApprovalRequest{
		ProductID: 10, RequestType: "publish", Requester: "tester",
		Status: "approved", EntityType: "listing_task", EntityID: task.ID, RiskLevel: "high",
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create approval: %v", err)
	}
	db.Model(&task).Update("approval_id", req.ID)

	updated, err := svc.ExecuteTask(task.ID, "tester")
	if err != nil {
		t.Fatalf("ExecuteTask: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("status = %s, want completed", updated.Status)
	}
	if hookCalled {
		t.Fatal("mock execution must not call publishHook")
	}
}

// TestService_ExecuteTask_MockIgnoresLegacyPublishHook verifies a configured
// legacy hook cannot turn a simulation into an external write.
func TestService_ExecuteTask_MockIgnoresLegacyPublishHook(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)
	svc.publishHook = func(taskID int64, mode ExecutionMode) error {
		return errPublishHookFailed
	}

	task := ListingTask{ProductID: 10, PlatformID: 1, Status: "approved", CreatedBy: "tester"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	item := ListingTaskItem{TaskID: task.ID, ProductID: 10, PlatformID: 1, Status: "pending"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	req := approval.ApprovalRequest{
		ProductID: 10, RequestType: "publish", Requester: "tester",
		Status: "approved", EntityType: "listing_task", EntityID: task.ID, RiskLevel: "high",
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create approval: %v", err)
	}
	db.Model(&task).Update("approval_id", req.ID)

	updated, err := svc.ExecuteTask(task.ID, "tester")
	if err != nil {
		t.Fatalf("ExecuteTask: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("status = %s, want completed mock", updated.Status)
	}
	if updated.LastError != "" {
		t.Fatalf("last_error = %q, want empty", updated.LastError)
	}
	// Mock items are explicitly labelled and have no platform error.
	var items []ListingTaskItem
	db.Where("task_id = ?", task.ID).Find(&items)
	for _, it := range items {
		if it.ErrorMessage != "" || !strings.Contains(string(it.Result), `"external_write":false`) {
			t.Fatalf("item %d is not a clean mock result: error=%q result=%s", it.ID, it.ErrorMessage, it.Result)
		}
	}
}

// ── CreateFromSuggestion ──

// TestService_CreateFromSuggestion tests the happy path: candidate found, approval created, task created.
func TestService_CreateFromSuggestion(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})

	// Create candidate_product table and insert a test candidate.
	db.Exec("CREATE TABLE candidate_product (id INTEGER PRIMARY KEY, title TEXT, purchase_price REAL DEFAULT 0, target_sale_price REAL DEFAULT 0, target_platform_id INTEGER, destination_country TEXT DEFAULT 'US')")
	db.Exec("INSERT INTO candidate_product (id, title, purchase_price, target_sale_price, target_platform_id, destination_country) VALUES (1, 'Test Product', 10.0, 25.0, 1, 'US')")

	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

	task, err := svc.CreateFromSuggestion(1, "owner")
	if err != nil {
		t.Fatalf("CreateFromSuggestion: %v", err)
	}
	if task.Status != "pending_approval" {
		t.Fatalf("Status = %s (expected pending_approval)", task.Status)
	}
	if task.ApprovalID == nil {
		t.Fatal("ApprovalID should be set")
	}
	if task.SourceType != "suggestion" {
		t.Fatalf("SourceType = %s (expected suggestion)", task.SourceType)
	}

	// Verify approval was created and linked.
	var appr approval.ApprovalRequest
	if err := db.First(&appr, *task.ApprovalID).Error; err != nil {
		t.Fatalf("approval not found: %v", err)
	}
	if appr.Status != "pending" {
		t.Fatalf("approval status = %s (expected pending)", appr.Status)
	}
	if appr.EntityType != "listing_task" {
		t.Fatalf("EntityType = %s (expected listing_task)", appr.EntityType)
	}
	if appr.EntityID != task.ID {
		t.Fatalf("EntityID = %d (expected %d)", appr.EntityID, task.ID)
	}
}

// TestService_CreateFromSuggestion_NotFound tests error when candidate doesn't exist.
func TestService_CreateFromSuggestion_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

	// Create table but insert no rows.
	db.Exec("CREATE TABLE candidate_product (id INTEGER PRIMARY KEY, title TEXT, purchase_price REAL, target_sale_price REAL, target_platform_id INTEGER, destination_country TEXT)")

	_, err := svc.CreateFromSuggestion(999, "owner")
	if err == nil {
		t.Fatal("expected error for non-existent candidate")
	}
}

// TestService_CreateFromSuggestion_NoApprovalService tests error when approvalSvc is nil.
func TestService_CreateFromSuggestion_NoApprovalService(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &approval.ApprovalRequest{})
	db.Exec("CREATE TABLE candidate_product (id INTEGER PRIMARY KEY, title TEXT)")
	db.Exec("INSERT INTO candidate_product (id, title) VALUES (1, 'Test')")

	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

	_, err := svc.CreateFromSuggestion(1, "owner")
	if err == nil {
		t.Fatal("expected error when approval service not configured")
	}
}

// ── ReviewTask ──

func TestService_ReviewTask_Published(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &platform.Platform{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

	// Create platform record
	db.Create(&platform.Platform{ID: 1, Name: "Ozon", Code: "ozon"})

	task, _ := svc.Create(&CreateTaskInput{
		ProductID:          10,
		PlatformID:         1,
		Status:             "completed",
		TargetSalePrice:    dbtest.FloatPtr(100.0),
		TargetProfitMargin: dbtest.FloatPtr(0.25),
		DestinationCountry: "US",
	})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 10, PlatformID: 1, Status: "completed"})

	review, err := svc.ReviewTask(task.ID)
	if err != nil {
		t.Fatalf("ReviewTask: %v", err)
	}
	if !review.Published {
		t.Fatal("expected published = true")
	}
	if review.Status != "completed" {
		t.Fatalf("status = %s", review.Status)
	}
	if review.Platform != "ozon" {
		t.Fatalf("platform = %s", review.Platform)
	}
	if review.ProfitExpected == nil {
		t.Fatal("expected profit_expected to be set")
	}
	if *review.ProfitExpected != 25.0 {
		t.Fatalf("profit_expected = %f (want 25.0)", *review.ProfitExpected)
	}
	if review.MarginExpected == nil || *review.MarginExpected != 0.25 {
		t.Fatalf("margin_expected = %v", review.MarginExpected)
	}
	// No orders exist — actual fields should be nil
	if review.ProfitActual != nil {
		t.Fatal("expected profit_actual to be nil (no orders)")
	}
	if review.MarginActual != nil {
		t.Fatal("expected margin_actual to be nil (no orders)")
	}
}

func TestService_ReviewTask_Failed(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{
		ProductID:          10,
		PlatformID:         1,
		Status:             "failed",
		TargetSalePrice:    dbtest.FloatPtr(50.0),
		TargetProfitMargin: dbtest.FloatPtr(0.20),
		DestinationCountry: "US",
	})
	errMsg := "platform rejected: missing images"
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 10, PlatformID: 1, Status: "failed"})
	svc.UpdateItem(1, &UpdateTaskItemInput{ErrorMessage: &errMsg})

	review, err := svc.ReviewTask(task.ID)
	if err != nil {
		t.Fatalf("ReviewTask: %v", err)
	}
	if review.Published {
		t.Fatal("expected published = false for failed task")
	}
	if len(review.PlatformErrors) != 1 {
		t.Fatalf("expected 1 platform error, got %d", len(review.PlatformErrors))
	}
	if review.PlatformErrors[0] != "platform rejected: missing images" {
		t.Fatalf("platform error = %q", review.PlatformErrors[0])
	}
	// Expected profit should still be computed from target fields
	if review.ProfitExpected == nil || *review.ProfitExpected != 10.0 {
		t.Fatalf("profit_expected = %v", review.ProfitExpected)
	}
}

func TestService_ReviewTask_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

	_, err := svc.ReviewTask(99999)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestService_ReviewTask_WithActualProfit(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, nil)

	task, _ := svc.Create(&CreateTaskInput{
		ProductID:          100,
		PlatformID:         2,
		Status:             "completed",
		TargetSalePrice:    dbtest.FloatPtr(200.0),
		TargetProfitMargin: dbtest.FloatPtr(0.30),
		DestinationCountry: "RU",
	})

	// Create order tables and insert a matching order with profit data.
	db.Exec("CREATE TABLE sales_order (id INTEGER PRIMARY KEY, platform_id INTEGER, profit_amount REAL DEFAULT 0, profit_margin REAL DEFAULT 0)")
	db.Exec("CREATE TABLE sales_order_item (id INTEGER PRIMARY KEY, order_id INTEGER, product_id INTEGER)")
	db.Exec("INSERT INTO sales_order (id, platform_id, profit_amount, profit_margin) VALUES (1, 2, 45.0, 0.225)")
	db.Exec("INSERT INTO sales_order_item (id, order_id, product_id) VALUES (1, 1, 100)")

	review, err := svc.ReviewTask(task.ID)
	if err != nil {
		t.Fatalf("ReviewTask: %v", err)
	}
	if !review.Published {
		t.Fatal("expected published = true")
	}
	if review.ProfitActual == nil {
		t.Fatal("expected profit_actual to be set")
	}
	if *review.ProfitActual != 45.0 {
		t.Fatalf("profit_actual = %f (want 45.0)", *review.ProfitActual)
	}
	if review.MarginActual == nil {
		t.Fatal("expected margin_actual to be set")
	}
	if *review.MarginActual != 0.225 {
		t.Fatalf("margin_actual = %f (want 0.225)", *review.MarginActual)
	}
}

// TestService_ExecuteTask_DryRunFromDecisionSnapshot verifies that when DecisionSnapshot
// contains mode="dry_run", ExecuteTask enters the dry-run path (no real platform publish).
func TestService_ExecuteTask_DryRunFromDecisionSnapshot(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)

	// Create a listing task with DecisionSnapshot containing mode=dry_run.
	ds := json.RawMessage(`{"mode":"dry_run","completeness_score":90}`)
	task := ListingTask{
		ProductID:        20,
		PlatformID:       1,
		Status:           "approved",
		CreatedBy:        "A8",
		DecisionSnapshot: ds,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	item := ListingTaskItem{TaskID: task.ID, ProductID: 20, PlatformID: 1, Status: "pending"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	// Create approved approval.
	req := approval.ApprovalRequest{
		ProductID:   20,
		RequestType: "listing_task",
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
	// Verify items were completed (dry-run path marks items completed).
	var items []ListingTaskItem
	db.Where("task_id = ?", task.ID).Find(&items)
	for _, it := range items {
		if it.Status != "completed" {
			t.Fatalf("item %d status = %s, want completed", it.ID, it.Status)
		}
		// Dry-run result should contain dry_run=true.
		var result map[string]interface{}
		if err := json.Unmarshal(it.Result, &result); err != nil {
			t.Fatalf("unmarshal item result: %v", err)
		}
		dryRun, ok := result["dry_run"].(bool)
		if !ok || !dryRun {
			t.Fatalf("item %d result does not have dry_run=true: %v", it.ID, result)
		}
	}
}

// TestService_ExecuteTask_DefaultModeIsMock verifies an omitted mode cannot
// silently become a production publish.
func TestService_ExecuteTask_DefaultModeIsMock(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	var publishCalled bool
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)
	svc.publishHook = func(taskID int64, mode ExecutionMode) error {
		publishCalled = true
		return nil
	}

	// Task without dry_run mode — DecisionSnapshot is nil.
	task := ListingTask{
		ProductID:  21,
		PlatformID: 1,
		Status:     "approved",
		CreatedBy:  "A8",
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	item := ListingTaskItem{TaskID: task.ID, ProductID: 21, PlatformID: 1, Status: "pending"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	req := approval.ApprovalRequest{
		ProductID:   21,
		RequestType: "listing_task",
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

	_, err := svc.ExecuteTask(task.ID, "A8")
	if err != nil {
		t.Fatalf("ExecuteTask: %v", err)
	}
	if publishCalled {
		t.Fatal("default mode must not call publishHook")
	}
}

func TestService_ExecuteTask_ProductionRequiresImageReleaseBeforePublish(t *testing.T) {
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{})
	apprSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	publishCalls := 0
	svc := NewService(db, dbtest.NewLogger(t), apprSvc, nil, nil)
	svc.publishHook = func(int64, ExecutionMode) error { publishCalls++; return nil }

	task := ListingTask{ProductID: 1, PlatformID: 1, Status: "approved", ExecutionMode: ExecutionModeProduction}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	req := approval.ApprovalRequest{RequestType: "listing_task", Requester: "owner", Status: "approved", EntityType: "listing_task", EntityID: task.ID, RiskLevel: "high"}
	if err := db.Create(&req).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&task).Update("approval_id", req.ID).Error; err != nil {
		t.Fatal(err)
	}

	_, err := svc.ExecuteTask(task.ID, "owner")
	if !errors.Is(err, ErrImageReleaseAttestationRequired) {
		t.Fatalf("error = %v", err)
	}
	if publishCalls != 0 {
		t.Fatalf("publish calls = %d", publishCalls)
	}
	var unchanged ListingTask
	if err := db.First(&unchanged, task.ID).Error; err != nil || unchanged.Status != "approved" {
		t.Fatalf("production gate mutated task: %+v err=%v", unchanged, err)
	}
}
