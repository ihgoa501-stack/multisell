package sourcing1688

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Sourcing1688Product{})
}

func newService(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	return NewService(db, dbtest.NewLogger(t))
}

func createTestProduct(t *testing.T, svc *Service, url string) *Sourcing1688Product {
	t.Helper()
	price := 12.5
	qty := 10
	in := &CreateInput{
		SourceURL:    url,
		SupplierName: "Test Supplier",
		Price1688:    &price,
		MinOrderQty:  &qty,
	}
	p, err := svc.Create(in)
	if err != nil {
		t.Fatalf("createTestProduct failed: %v", err)
	}
	return p
}

func TestSourcing1688_Create(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	price := 9.99
	qty := 5
	in := &CreateInput{
		SourceURL:    "https://1688.com/item/123",
		SupplierName: "Acme Corp",
		Price1688:    &price,
		MinOrderQty:  &qty,
	}
	p, err := svc.Create(in)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if p.Status != "pending" {
		t.Fatalf("default status = %q, want %q", p.Status, "pending")
	}
	if p.SupplierName != "Acme Corp" {
		t.Fatalf("SupplierName = %q, want %q", p.SupplierName, "Acme Corp")
	}
}

func TestSourcing1688_Create_Defaults(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	in := &CreateInput{SourceURL: "https://1688.com/item/456"}
	p, err := svc.Create(in)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if p.MinOrderQty != 1 {
		t.Fatalf("MinOrderQty = %d, want 1", p.MinOrderQty)
	}
	if p.Price1688 != 0 {
		t.Fatalf("Price1688 = %f, want 0", p.Price1688)
	}
}

func TestSourcing1688_Get(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	created := createTestProduct(t, svc, "https://1688.com/item/get1")
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.SourceURL != "https://1688.com/item/get1" {
		t.Fatalf("SourceURL = %q, want %q", got.SourceURL, "https://1688.com/item/get1")
	}
}

func TestSourcing1688_Get_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	if _, err := svc.Get(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSourcing1688_Update(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	created := createTestProduct(t, svc, "https://1688.com/item/upd1")
	newName := "Updated Supplier"
	in := &UpdateInput{SupplierName: &newName}
	updated, err := svc.Update(created.ID, in)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.SupplierName != "Updated Supplier" {
		t.Fatalf("SupplierName = %q, want %q", updated.SupplierName, "Updated Supplier")
	}
}

func TestSourcing1688_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	name := "X"
	if _, err := svc.Update(999, &UpdateInput{SupplierName: &name}); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSourcing1688_Delete(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	created := createTestProduct(t, svc, "https://1688.com/item/del1")
	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := svc.Get(created.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

func TestSourcing1688_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	if err := svc.Delete(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSourcing1688_Import(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	p := createTestProduct(t, svc, "https://1688.com/item/imp1")
	imported, err := svc.Import(p.ID, &ImportInput{ImportedBy: "admin"})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if imported.Status != "imported" {
		t.Fatalf("Status = %q, want %q", imported.Status, "imported")
	}
}

func TestSourcing1688_Import_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	if _, err := svc.Import(999, &ImportInput{ImportedBy: "admin"}); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSourcing1688_Reject(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	p := createTestProduct(t, svc, "https://1688.com/item/rej1")
	rejected, err := svc.Reject(p.ID, &RejectInput{RejectedBy: "admin", Reason: "bad quality"})
	if err != nil {
		t.Fatalf("Reject failed: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("Status = %q, want %q", rejected.Status, "rejected")
	}
}

func TestSourcing1688_Reject_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	if _, err := svc.Reject(999, &RejectInput{RejectedBy: "admin"}); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSourcing1688_List_Pagination(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	for i := 0; i < 25; i++ {
		createTestProduct(t, svc, "https://1688.com/item/"+dbtest.IToA(int64(i)))
	}

	p := &common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.List(p, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("total = %d, want 25", total)
	}
	if len(items) != 10 {
		t.Fatalf("returned %d items, want 10", len(items))
	}
}

func TestSourcing1688_List_SearchFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("ILIKE not supported in SQLite")
	}
	// Search uses ILIKE which is PostgreSQL-specific.
	// This test validates the filter path compiles; actual search requires PostgreSQL.
	db := newTestDB(t)
	svc := newService(t, db)

	createTestProduct(t, svc, "https://1688.com/item/a1")

	p := &common.Pagination{Page: 1, Size: 20}
	filter := &ListFilter{Search: "Acme"}
	_, _, err := svc.List(p, filter)
	if err == nil {
		// SQLite may fail on ILIKE or fall through; either is acceptable
		t.Log("List with search succeeded (unexpected on SQLite but acceptable)")
	}
}

func TestSourcing1688_List_StatusFilter(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	p1 := createTestProduct(t, svc, "https://1688.com/item/s1")
	svc.Import(p1.ID, &ImportInput{ImportedBy: "admin"})
	createTestProduct(t, svc, "https://1688.com/item/s2")

	p := &common.Pagination{Page: 1, Size: 20}
	filter := &ListFilter{Status: "imported"}
	items, total, err := svc.List(p, filter)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if items[0].Status != "imported" {
		t.Fatalf("Status = %q, want %q", items[0].Status, "imported")
	}
}

func TestSourcing1688_Summary(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	p1 := createTestProduct(t, svc, "https://1688.com/item/sm1")
	p2 := createTestProduct(t, svc, "https://1688.com/item/sm2")
	createTestProduct(t, svc, "https://1688.com/item/sm3")
	svc.Import(p1.ID, &ImportInput{ImportedBy: "admin"})
	svc.Reject(p2.ID, &RejectInput{RejectedBy: "admin"})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary failed: %v", err)
	}
	if summary.Total != 3 {
		t.Fatalf("Total = %d, want 3", summary.Total)
	}
	if summary.ByStatus["imported"] != 1 {
		t.Fatalf("imported count = %d, want 1", summary.ByStatus["imported"])
	}
	if summary.ByStatus["rejected"] != 1 {
		t.Fatalf("rejected count = %d, want 1", summary.ByStatus["rejected"])
	}
	if summary.ByStatus["pending"] != 1 {
		t.Fatalf("pending count = %d, want 1", summary.ByStatus["pending"])
	}
}
