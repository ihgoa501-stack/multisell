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

func TestGetBatch_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportBatch{}, &ImportBatchRow{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.GetBatch(999)
	if err == nil {
		t.Fatal("expected error for non-existent batch")
	}
}

func TestListBatches_FilterByStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportBatch{}, &ImportBatchRow{})
	svc := NewService(db, dbtest.NewLogger(t))

	_ = svc.CreateBatch(&ImportBatch{SourceType: "csv", FileName: "a.csv", Status: "pending"})
	_ = svc.CreateBatch(&ImportBatch{SourceType: "csv", FileName: "b.csv", Status: "completed"})
	_ = svc.CreateBatch(&ImportBatch{SourceType: "csv", FileName: "c.csv", Status: "failed"})

	items, total, err := svc.ListBatches("", "completed", 1, 10)
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 completed batch, got %d", total)
	}
	if items[0].Status != "completed" {
		t.Fatalf("Status = %s", items[0].Status)
	}
}

func TestListBatches_Pagination(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportBatch{}, &ImportBatchRow{})
	svc := NewService(db, dbtest.NewLogger(t))

	for i := 0; i < 5; i++ {
		_ = svc.CreateBatch(&ImportBatch{SourceType: "csv", FileName: "a.csv"})
	}

	// Page 1 with size 2
	items, total, err := svc.ListBatches("", "", 1, 2)
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(items))
	}

	// Page 3 should have 1 item
	items, total, err = svc.ListBatches("", "", 3, 2)
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item on page 3, got %d", len(items))
	}

	// Page 4 should be empty
	items, total, err = svc.ListBatches("", "", 4, 2)
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items on page 4, got %d", len(items))
	}
}

func TestListRows_ByStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportBatch{}, &ImportBatchRow{})
	svc := NewService(db, dbtest.NewLogger(t))

	_ = svc.CreateBatch(&ImportBatch{SourceType: "csv", FileName: "a.csv"})
	batch, _ := svc.GetBatch(1)

	_ = svc.CreateRow(&ImportBatchRow{BatchID: batch.ID, RowIndex: 1, Status: "pending"})
	_ = svc.CreateRow(&ImportBatchRow{BatchID: batch.ID, RowIndex: 2, Status: "success"})
	_ = svc.CreateRow(&ImportBatchRow{BatchID: batch.ID, RowIndex: 3, Status: "failed"})

	items, total, err := svc.ListRows(batch.ID, "success", 1, 10)
	if err != nil {
		t.Fatalf("ListRows: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 success row, got %d", total)
	}
	if items[0].Status != "success" {
		t.Fatalf("Status = %s", items[0].Status)
	}
}

func TestListRows_BatchNotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportBatch{}, &ImportBatchRow{})
	svc := NewService(db, dbtest.NewLogger(t))

	items, total, err := svc.ListRows(999, "", 1, 10)
	if err != nil {
		t.Fatalf("ListRows: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected 0 rows, got %d", total)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(items))
	}
}
