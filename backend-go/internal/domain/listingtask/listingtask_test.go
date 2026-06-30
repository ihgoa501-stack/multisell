package listingtask

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func int64Ptr(v int64) *int64 { return &v }

func TestService_Task_CreateAndGetByID(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

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
	db := dbtest.NewDB(t, &ListingTask{}, &ListingTaskItem{})
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

	task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10, Status: "pending"})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 1, PlatformID: 10})
	svc.CreateItem(&CreateTaskItemInput{TaskID: task.ID, ProductID: 2, PlatformID: 10})

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
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

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
	svc := NewService(db, dbtest.NewLogger(t), nil, false)

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
