package importbatch

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportBatch{}, &ImportBatchRow{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create batch
	b := &ImportBatch{
		SourceType: "product",
		FileName:   "products_202606.csv",
		CreatedBy:  "admin",
	}
	err := svc.CreateBatch(b)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if b.ID == 0 {
		t.Fatal("ID should be set")
	}

	// GetBatch
	got, err := svc.GetBatch(b.ID)
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}
	if got.FileName != "products_202606.csv" {
		t.Fatalf("FileName = %s", got.FileName)
	}
	if got.Status != "pending" {
		t.Fatalf("Status = %s", got.Status)
	}

	// ListBatches
	items, total, err := svc.ListBatches("", "", 1, 10)
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// Update batch
	b.Status = "completed"
	err = svc.UpdateBatch(b)
	if err != nil {
		t.Fatalf("UpdateBatch: %v", err)
	}
	got, _ = svc.GetBatch(b.ID)
	if got.Status != "completed" {
		t.Fatalf("Status = %s", got.Status)
	}

	// Create row
	r := &ImportBatchRow{
		BatchID:  b.ID,
		RowIndex: 1,
		Status:   "success",
	}
	err = svc.CreateRow(r)
	if err != nil {
		t.Fatalf("CreateRow: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("Row ID should be set")
	}

	// ListRows
	rows, total, err := svc.ListRows(b.ID, "", 1, 10)
	if err != nil {
		t.Fatalf("ListRows: %v", err)
	}
	if total != 1 {
		t.Fatalf("row total = %d", total)
	}
	_ = rows

	// Delete batch (cascades)
	err = svc.DeleteBatch(b.ID)
	if err != nil {
		t.Fatalf("DeleteBatch: %v", err)
	}
	_, err = svc.GetBatch(b.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_ListBatches_FilterBySourceType(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportBatch{}, &ImportBatchRow{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create two batches with different source types
	_ = svc.CreateBatch(&ImportBatch{SourceType: "excel", FileName: "a.xlsx"})
	_ = svc.CreateBatch(&ImportBatch{SourceType: "csv", FileName: "b.csv"})

	items, total, err := svc.ListBatches("excel", "", 1, 10)
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 batch, got %d", total)
	}
	if items[0].SourceType != "excel" {
		t.Fatalf("SourceType = %s", items[0].SourceType)
	}
}
