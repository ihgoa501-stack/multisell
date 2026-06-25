package importbatch

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &ImportBatch{}, &ImportBatchRow{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func seedBatch(t *testing.T, svc *Service, batchType string) *ImportBatch {
	t.Helper()
	b := &ImportBatch{
		Type:     batchType,
		FileName: "test.xlsx",
		CreatedBy: "admin",
	}
	if err := svc.CreateBatch(b); err != nil {
		t.Fatalf("seedBatch failed: %v", err)
	}
	return b
}

func seedRow(t *testing.T, svc *Service, batchID int64, rowIdx int) *ImportBatchRow {
	t.Helper()
	raw, _ := json.Marshal(map[string]string{"name": "test"})
	r := &ImportBatchRow{
		BatchID:  batchID,
		RowIndex: rowIdx,
		Status:   "pending",
		RawData:  raw,
	}
	if err := svc.CreateRow(r); err != nil {
		t.Fatalf("seedRow failed: %v", err)
	}
	return r
}

func TestBatch_Create(t *testing.T) {
	svc := newService(t)

	b := &ImportBatch{
		Type:     "sku",
		FileName: "products.csv",
		CreatedBy: "user1",
	}
	if err := svc.CreateBatch(b); err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if b.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestBatch_Create_DefaultStatus(t *testing.T) {
	svc := newService(t)

	b := &ImportBatch{
		Type:     "order",
		FileName: "orders.xlsx",
		CreatedBy: "user1",
	}
	if err := svc.CreateBatch(b); err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if b.Status != "pending" {
		t.Fatalf("Status = %q, want pending", b.Status)
	}
}

func TestBatch_GetByID(t *testing.T) {
	svc := newService(t)

	created := seedBatch(t, svc, "sku")

	got, err := svc.GetBatch(created.ID)
	if err != nil {
		t.Fatalf("GetBatch failed: %v", err)
	}
	if got.Type != "sku" {
		t.Fatalf("Type = %q, want sku", got.Type)
	}
	if got.FileName != "test.xlsx" {
		t.Fatalf("FileName = %q, want test.xlsx", got.FileName)
	}
}

func TestBatch_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	_, err := svc.GetBatch(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestBatch_List_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 12; i++ {
		seedBatch(t, svc, "sku")
	}

	items, total, err := svc.ListBatches("", "", 1, 5)
	if err != nil {
		t.Fatalf("ListBatches failed: %v", err)
	}
	if total != 12 {
		t.Fatalf("total = %d, want 12", total)
	}
	if len(items) != 5 {
		t.Fatalf("returned %d items, want 5", len(items))
	}
}

func TestBatch_List_FilterByType(t *testing.T) {
	svc := newService(t)

	seedBatch(t, svc, "sku")
	seedBatch(t, svc, "sku")
	seedBatch(t, svc, "order")

	items, total, err := svc.ListBatches("sku", "", 1, 20)
	if err != nil {
		t.Fatalf("ListBatches failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("returned %d items, want 2", len(items))
	}
}

func TestBatch_Update(t *testing.T) {
	svc := newService(t)

	b := seedBatch(t, svc, "sku")
	b.Status = "completed"
	b.TotalRows = 100
	b.SuccessCount = 95
	b.ErrorCount = 5

	if err := svc.UpdateBatch(b); err != nil {
		t.Fatalf("UpdateBatch failed: %v", err)
	}

	got, err := svc.GetBatch(b.ID)
	if err != nil {
		t.Fatalf("GetBatch failed: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("Status = %q, want completed", got.Status)
	}
	if got.SuccessCount != 95 {
		t.Fatalf("SuccessCount = %d, want 95", got.SuccessCount)
	}
}

func TestBatch_Delete_CascadesRows(t *testing.T) {
	svc := newService(t)

	b := seedBatch(t, svc, "sku")
	seedRow(t, svc, b.ID, 1)
	seedRow(t, svc, b.ID, 2)
	seedRow(t, svc, b.ID, 3)

	rows, _, err := svc.ListRows(b.ID, "", 1, 50)
	if err != nil {
		t.Fatalf("ListRows failed: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("pre-delete row count = %d, want 3", len(rows))
	}

	if err := svc.DeleteBatch(b.ID); err != nil {
		t.Fatalf("DeleteBatch failed: %v", err)
	}

	_, err = svc.GetBatch(b.ID)
	if err == nil {
		t.Fatal("expected error for deleted batch")
	}

	rowsAfter, total, err := svc.ListRows(b.ID, "", 1, 50)
	if err != nil {
		t.Fatalf("ListRows after delete failed: %v", err)
	}
	if total != 0 {
		t.Fatalf("row count after delete = %d, want 0", total)
	}
	if len(rowsAfter) != 0 {
		t.Fatalf("returned %d rows after delete, want 0", len(rowsAfter))
	}
}

func TestBatch_Delete_NotFound(t *testing.T) {
	svc := newService(t)

	if err := svc.DeleteBatch(999); err != nil {
		t.Fatalf("DeleteBatch for non-existent ID should succeed: %v", err)
	}
}

func TestBatchRow_Create(t *testing.T) {
	svc := newService(t)

	b := seedBatch(t, svc, "sku")
	raw, _ := json.Marshal(map[string]string{"sku": "ABC-123"})
	r := &ImportBatchRow{
		BatchID:  b.ID,
		RowIndex: 1,
		Status:   "pending",
		RawData:  raw,
	}
	if err := svc.CreateRow(r); err != nil {
		t.Fatalf("CreateRow failed: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestBatchRow_List_Pagination(t *testing.T) {
	svc := newService(t)

	b := seedBatch(t, svc, "sku")
	for i := 0; i < 15; i++ {
		seedRow(t, svc, b.ID, i+1)
	}

	items, total, err := svc.ListRows(b.ID, "", 1, 5)
	if err != nil {
		t.Fatalf("ListRows failed: %v", err)
	}
	if total != 15 {
		t.Fatalf("total = %d, want 15", total)
	}
	if len(items) != 5 {
		t.Fatalf("returned %d items, want 5", len(items))
	}
}

func TestBatchRow_List_FilterByStatus(t *testing.T) {
	svc := newService(t)

	b := seedBatch(t, svc, "sku")

	raw, _ := json.Marshal(map[string]string{"x": "1"})
	svc.CreateRow(&ImportBatchRow{BatchID: b.ID, RowIndex: 1, Status: "completed", RawData: raw})
	svc.CreateRow(&ImportBatchRow{BatchID: b.ID, RowIndex: 2, Status: "completed", RawData: raw})
	svc.CreateRow(&ImportBatchRow{BatchID: b.ID, RowIndex: 3, Status: "failed", RawData: raw})

	items, total, err := svc.ListRows(b.ID, "failed", 1, 20)
	if err != nil {
		t.Fatalf("ListRows failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 {
		t.Fatalf("returned %d items, want 1", len(items))
	}
	if items[0].Status != "failed" {
		t.Fatalf("Status = %q, want failed", items[0].Status)
	}
}
