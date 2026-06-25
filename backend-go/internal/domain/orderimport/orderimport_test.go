package orderimport

import (
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
