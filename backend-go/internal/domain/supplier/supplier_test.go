package supplier

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Supplier{}, &ProductSupplier{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	db := newTestDB(t)
	return NewService(db, dbtest.NewLogger(t))
}

// ── Supplier CRUD ───────────────────────────────────────────────────

func TestSupplier_Create(t *testing.T) {
	svc := newService(t)

	sup := &Supplier{Name: "Test Supplier", ContactPerson: "Zhang", ContactPhone: "13800138000"}
	if err := svc.Create(context.Background(), sup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sup.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}
}

func TestSupplier_Create_EmptyName(t *testing.T) {
	svc := newService(t)

	sup := &Supplier{Name: ""}
	if err := svc.Create(context.Background(), sup); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestSupplier_Create_WhitespaceName(t *testing.T) {
	svc := newService(t)

	sup := &Supplier{Name: "   "}
	if err := svc.Create(context.Background(), sup); err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestSupplier_GetByID(t *testing.T) {
	svc := newService(t)

	created := &Supplier{Name: "Findable Supplier"}
	if err := svc.Create(context.Background(), created); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != "Findable Supplier" {
		t.Fatalf("GetByID Name = %q, want %q", got.Name, "Findable Supplier")
	}
}

func TestSupplier_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetByID(context.Background(), 999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSupplier_Update(t *testing.T) {
	svc := newService(t)

	sup := &Supplier{Name: "Old Name"}
	if err := svc.Create(context.Background(), sup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sup.Name = "Updated Name"
	if err := svc.Update(context.Background(), sup); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := svc.GetByID(context.Background(), sup.ID)
	if got.Name != "Updated Name" {
		t.Fatalf("after Update Name = %q, want %q", got.Name, "Updated Name")
	}
}

func TestSupplier_Update_EmptyName(t *testing.T) {
	svc := newService(t)

	sup := &Supplier{Name: "Valid"}
	_ = svc.Create(context.Background(), sup)

	sup.Name = ""
	if err := svc.Update(context.Background(), sup); err == nil {
		t.Fatal("expected error for empty name on update")
	}
}

func TestSupplier_Delete(t *testing.T) {
	svc := newService(t)

	sup := &Supplier{Name: "To Delete"}
	if err := svc.Create(context.Background(), sup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(context.Background(), sup.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := svc.GetByID(context.Background(), sup.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

// ── List / ListAll ──────────────────────────────────────────────────

func TestSupplier_List_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 25; i++ {
		if err := svc.Create(context.Background(), &Supplier{Name: dbtest.IToA(int64(i))}); err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
	}

	items, total, err := svc.List(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("List total = %d, want 25", total)
	}
	if len(items) != 10 {
		t.Fatalf("List returned %d items, want 10", len(items))
	}
}

// Note: ILIKE search tests are skipped because SQLite does not support ILIKE.
// These would pass on PostgreSQL which is the production database.
// The List method's ILIKE search is tested manually against PostgreSQL.

func TestSupplier_ListAll_OnlyEnabled(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	_ = svc.Create(context.Background(), &Supplier{Name: "Active1", Status: 1})
	_ = svc.Create(context.Background(), &Supplier{Name: "Active2", Status: 1})
	_ = svc.Create(context.Background(), &Supplier{Name: "Inactive"})
	// SQLite + GORM zero-value handling: explicit UPDATE to set inactive status
	db.Model(&Supplier{}).Where("name = ?", "Inactive").Update("status", 0)

	items, err := svc.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ListAll returned %d items, want 2", len(items))
	}
	for _, item := range items {
		if item.Status != 1 {
			t.Fatalf("expected only enabled suppliers, got status=%d", item.Status)
		}
	}
}

// ── ProductSupplier CRUD ────────────────────────────────────────────

func TestSupplier_CreateProductSupplier(t *testing.T) {
	svc := newService(t)

	price := decimal.NewFromInt(50)
	ps := &ProductSupplier{ProductID: 1, SupplierID: 1, SupplyPrice: &price, MinOrderQty: 100}
	if err := svc.CreateProductSupplier(context.Background(), ps); err != nil {
		t.Fatalf("CreateProductSupplier failed: %v", err)
	}
	if ps.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}
}

func TestSupplier_UpdateProductSupplier(t *testing.T) {
	svc := newService(t)

	price := decimal.NewFromInt(50)
	ps := &ProductSupplier{ProductID: 1, SupplierID: 1, SupplyPrice: &price, MinOrderQty: 100}
	if err := svc.CreateProductSupplier(context.Background(), ps); err != nil {
		t.Fatalf("CreateProductSupplier failed: %v", err)
	}

	newPrice := decimal.NewFromInt(45)
	ps.SupplyPrice = &newPrice
	ps.MinOrderQty = 200
	if err := svc.UpdateProductSupplier(context.Background(), ps); err != nil {
		t.Fatalf("UpdateProductSupplier failed: %v", err)
	}

	items, _ := svc.ListProductSuppliers(context.Background(), 1)
	if len(items) != 1 {
		t.Fatalf("expected 1 product supplier, got %d", len(items))
	}
	if !items[0].SupplyPrice.Equal(decimal.NewFromInt(45)) {
		t.Fatalf("SupplyPrice = %s, want 45", items[0].SupplyPrice)
	}
	if items[0].MinOrderQty != 200 {
		t.Fatalf("MinOrderQty = %d, want 200", items[0].MinOrderQty)
	}
}

func TestSupplier_DeleteProductSupplier(t *testing.T) {
	svc := newService(t)

	price := decimal.NewFromInt(50)
	ps := &ProductSupplier{ProductID: 1, SupplierID: 1, SupplyPrice: &price, MinOrderQty: 100}
	if err := svc.CreateProductSupplier(context.Background(), ps); err != nil {
		t.Fatalf("CreateProductSupplier failed: %v", err)
	}

	if err := svc.DeleteProductSupplier(context.Background(), ps.ID); err != nil {
		t.Fatalf("DeleteProductSupplier failed: %v", err)
	}

	items, _ := svc.ListProductSuppliers(context.Background(), 1)
	if len(items) != 0 {
		t.Fatalf("expected 0 product suppliers after delete, got %d", len(items))
	}
}

func TestSupplier_ListProductSuppliers_FilterByProduct(t *testing.T) {
	svc := newService(t)

	p1 := decimal.NewFromInt(10)
	p2 := decimal.NewFromInt(20)
	_ = svc.CreateProductSupplier(context.Background(), &ProductSupplier{ProductID: 10, SupplierID: 1, SupplyPrice: &p1, MinOrderQty: 50})
	_ = svc.CreateProductSupplier(context.Background(), &ProductSupplier{ProductID: 10, SupplierID: 2, SupplyPrice: &p2, MinOrderQty: 100})
	_ = svc.CreateProductSupplier(context.Background(), &ProductSupplier{ProductID: 20, SupplierID: 1, SupplyPrice: &p1, MinOrderQty: 50})

	items, err := svc.ListProductSuppliers(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListProductSuppliers failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestSupplier_ListProductSuppliers_All(t *testing.T) {
	svc := newService(t)

	p1 := decimal.NewFromInt(10)
	_ = svc.CreateProductSupplier(context.Background(), &ProductSupplier{ProductID: 10, SupplierID: 1, SupplyPrice: &p1, MinOrderQty: 50})
	_ = svc.CreateProductSupplier(context.Background(), &ProductSupplier{ProductID: 20, SupplierID: 2, SupplyPrice: &p1, MinOrderQty: 100})

	items, err := svc.ListProductSuppliers(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListProductSuppliers failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}
