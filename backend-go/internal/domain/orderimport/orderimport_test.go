package orderimport

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, err := svc.Create(&CreateInput{
		FileName:  "orders_20260601.csv",
		SourceType: "manual",
		CreatedBy: "test_user",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if o.ID == 0 {
		t.Fatal("ID should be set")
	}
	if o.FileName != "orders_20260601.csv" {
		t.Fatalf("FileName = %s", o.FileName)
	}
	if o.Status != "pending" {
		t.Fatalf("Status = %s", o.Status)
	}

	got, err := svc.Get(o.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FileName != "orders_20260601.csv" {
		t.Fatalf("FileName = %s", got.FileName)
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "test.csv"})
	totalRows := 100
	updated, err := svc.Update(o.ID, &UpdateInput{
		TotalRows: &totalRows,
		Status:    dbtest.StringPtr("processing"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.TotalRows != 100 {
		t.Fatalf("TotalRows = %d", updated.TotalRows)
	}
	if updated.Status != "processing" {
		t.Fatalf("Status = %s", updated.Status)
	}
}

func TestService_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{FileName: "a.csv"})
	svc.Create(&CreateInput{FileName: "b.csv", Status: "completed"})
	svc.Create(&CreateInput{FileName: "c.csv", Status: "completed"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	items, total, err = svc.List(&p, &ListFilter{Status: "completed"})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 completed, got %d", total)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "del.csv"})
	if err := svc.Delete(o.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.Get(o.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_ProcessAndComplete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "proc.csv"})

	proc, err := svc.Process(o.ID)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if proc.Status != "processing" {
		t.Fatalf("Status = %s (expected processing)", proc.Status)
	}

	comp, err := svc.Complete(o.ID, &CompleteInput{
		SuccessCount: 80,
		ErrorCount:   20,
		Status:       "completed",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if comp.SuccessCount != 80 {
		t.Fatalf("SuccessCount = %d", comp.SuccessCount)
	}
	if comp.ErrorCount != 20 {
		t.Fatalf("ErrorCount = %d", comp.ErrorCount)
	}
	if comp.Status != "completed" {
		t.Fatalf("Status = %s", comp.Status)
	}
}

func TestService_Summary(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{FileName: "a.csv", Status: "completed", TotalRows: dbtest.IntPtr(50)})
	svc.Create(&CreateInput{FileName: "b.csv", Status: "processing", TotalRows: dbtest.IntPtr(30)})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Total != 2 {
		t.Fatalf("Total = %d", summary.Total)
	}
	if summary.ByStatus["completed"] != 1 {
		t.Fatalf("completed count = %d", summary.ByStatus["completed"])
	}
}

func TestCreate_Validation(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.Create(&CreateInput{})
	if err == nil {
		t.Fatal("expected error for empty FileName")
	}
}

func TestUpdate_StatusTransition(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "test.csv"})

	updated, err := svc.Update(o.ID, &UpdateInput{
		Status: dbtest.StringPtr("processing"),
	})
	if err != nil {
		t.Fatalf("Update to processing: %v", err)
	}
	if updated.Status != "processing" {
		t.Fatalf("Status = %s", updated.Status)
	}

	updated, err = svc.Update(o.ID, &UpdateInput{
		Status: dbtest.StringPtr("completed"),
	})
	if err != nil {
		t.Fatalf("Update to completed: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("Status = %s", updated.Status)
	}
}

func TestUpdate_InvalidStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "test.csv"})
	bogus := "bogus_status"
	updated, err := svc.Update(o.ID, &UpdateInput{
		Status: &bogus,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "bogus_status" {
		t.Fatalf("Status = %s", updated.Status)
	}
}

func TestProcess_StatusUpdate(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "proc.csv"})
	if o.Status != "pending" {
		t.Fatalf("initial Status = %s", o.Status)
	}

	proc, err := svc.Process(o.ID)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if proc.Status != "processing" {
		t.Fatalf("Status = %s (expected processing)", proc.Status)
	}
}

func TestComplete_WithItems(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "items.csv", TotalRows: dbtest.IntPtr(100)})
	svc.Process(o.ID)

	comp, err := svc.Complete(o.ID, &CompleteInput{
		SuccessCount: 80,
		ErrorCount:   20,
		ErrorDetail:  json.RawMessage(`[{"row":5,"reason":"invalid sku"}]`),
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if comp.SuccessCount != 80 {
		t.Fatalf("SuccessCount = %d", comp.SuccessCount)
	}
	if comp.ErrorCount != 20 {
		t.Fatalf("ErrorCount = %d", comp.ErrorCount)
	}
	if comp.Status != "completed" {
		t.Fatalf("Status = %s", comp.Status)
	}
}

func TestComplete_TransitionError(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "failed.csv"})

	// ponytail: Complete from "pending" skipping Process — currently allowed.
	comp, err := svc.Complete(o.ID, &CompleteInput{
		SuccessCount: 0,
		ErrorCount:   1,
		Status:       "failed",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if comp.Status != "failed" {
		t.Fatalf("Status = %s (expected failed)", comp.Status)
	}

	// Re-complete from "failed" — also allowed, no transition guard.
	comp2, err := svc.Complete(o.ID, &CompleteInput{
		SuccessCount: 5,
		ErrorCount:   0,
		Status:       "completed",
	})
	if err != nil {
		t.Fatalf("Complete from failed: %v", err)
	}
	if comp2.Status != "completed" {
		t.Fatalf("Status = %s (expected completed)", comp2.Status)
	}
}

func TestList_FilterByPlatform(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	p1 := int64(1)
	p2 := int64(2)

	svc.Create(&CreateInput{FileName: "a.csv", PlatformID: &p1})
	svc.Create(&CreateInput{FileName: "b.csv", PlatformID: &p1})
	svc.Create(&CreateInput{FileName: "c.csv", PlatformID: &p2})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, &ListFilter{PlatformID: &p1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d (expected 2)", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d (expected 2)", len(items))
	}
}

func TestList_FilterByStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{FileName: "a.csv", Status: "pending"})
	svc.Create(&CreateInput{FileName: "b.csv", Status: "completed"})
	svc.Create(&CreateInput{FileName: "c.csv", Status: "completed"})
	svc.Create(&CreateInput{FileName: "d.csv", Status: "failed"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, &ListFilter{Status: "completed"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d (expected 2)", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d (expected 2)", len(items))
	}
}

func TestList_WithPagination(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	for i := 0; i < 5; i++ {
		svc.Create(&CreateInput{FileName: fmt.Sprintf("file_%d.csv", i)})
	}

	p1 := common.Pagination{Page: 1, Size: 2}
	items1, total, err := svc.List(&p1, nil)
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d (expected 5)", total)
	}
	if len(items1) != 2 {
		t.Fatalf("len(items1) = %d (expected 2)", len(items1))
	}

	p3 := common.Pagination{Page: 3, Size: 2}
	items3, _, err := svc.List(&p3, nil)
	if err != nil {
		t.Fatalf("List page 3: %v", err)
	}
	if len(items3) != 1 {
		t.Fatalf("len(items3) = %d (expected 1)", len(items3))
	}
}

func TestSummary_Counts(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{FileName: "a.csv", Status: "completed", TotalRows: dbtest.IntPtr(50)})
	svc.Create(&CreateInput{FileName: "b.csv", Status: "completed", TotalRows: dbtest.IntPtr(30)})
	svc.Create(&CreateInput{FileName: "c.csv", Status: "processing"})
	svc.Create(&CreateInput{FileName: "d.csv", Status: "failed"})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Total != 4 {
		t.Fatalf("Total = %d (expected 4)", summary.Total)
	}
	if summary.ByStatus["completed"] != 2 {
		t.Fatalf("completed count = %d", summary.ByStatus["completed"])
	}
	if summary.ByStatus["processing"] != 1 {
		t.Fatalf("processing count = %d", summary.ByStatus["processing"])
	}
	if summary.ByStatus["failed"] != 1 {
		t.Fatalf("failed count = %d", summary.ByStatus["failed"])
	}
	if summary.TotalRows != 80 {
		t.Fatalf("TotalRows = %d", summary.TotalRows)
	}
}
