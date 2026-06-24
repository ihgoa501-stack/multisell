package sku

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Product{}, &SpecName{}, &SpecValue{}, &Sku{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	db := newTestDB(t)
	return NewService(db, dbtest.NewLogger(t))
}

// ── Product CRUD ─────────────────────────────────────────────────────

func TestProduct_Create(t *testing.T) {
	svc := newService(t)

	p := &Product{Name: "Test Product"}
	if err := svc.CreateProduct(context.Background(), p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}
}

func TestProduct_Create_EmptyName(t *testing.T) {
	svc := newService(t)

	if err := svc.CreateProduct(context.Background(), &Product{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestProduct_GetByID(t *testing.T) {
	svc := newService(t)

	created := &Product{Name: "Findable"}
	if err := svc.CreateProduct(context.Background(), created); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	got, err := svc.GetProductByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetProductByID failed: %v", err)
	}
	if got.Name != "Findable" {
		t.Fatalf("got Name=%q, want %q", got.Name, "Findable")
	}
}

func TestProduct_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetProductByID(context.Background(), 999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestProduct_List_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 25; i++ {
		if err := svc.CreateProduct(context.Background(), &Product{Name: dbtest.IToA(int64(i))}); err != nil {
			t.Fatalf("CreateProduct %d failed: %v", i, err)
		}
	}

	items, total, err := svc.ListProducts(context.Background(), 1, 10, "", 0, 0, nil)
	if err != nil {
		t.Fatalf("ListProducts failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("total = %d, want 25", total)
	}
	if len(items) != 10 {
		t.Fatalf("got %d items, want 10", len(items))
	}
}

func TestProduct_List_FilterByStatus(t *testing.T) {
	svc := newService(t)

	_ = svc.CreateProduct(context.Background(), &Product{Name: "Active", Status: 1})
	_ = svc.CreateProduct(context.Background(), &Product{Name: "Inactive", Status: 0})

	status := int16(1)
	items, total, err := svc.ListProducts(context.Background(), 1, 20, "", 0, 0, &status)
	if err != nil {
		t.Fatalf("ListProducts with status filter failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 || items[0].Name != "Active" {
		t.Fatalf("returned %+v, want [Active]", items)
	}
}

func TestProduct_Update(t *testing.T) {
	svc := newService(t)

	p := &Product{Name: "Old"}
	if err := svc.CreateProduct(context.Background(), p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	p.Name = "Updated"
	if err := svc.UpdateProduct(context.Background(), p); err != nil {
		t.Fatalf("UpdateProduct failed: %v", err)
	}

	got, _ := svc.GetProductByID(context.Background(), p.ID)
	if got.Name != "Updated" {
		t.Fatalf("after update Name=%q, want %q", got.Name, "Updated")
	}
}

func TestProduct_Delete(t *testing.T) {
	svc := newService(t)

	p := &Product{Name: "To Delete"}
	if err := svc.CreateProduct(context.Background(), p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	if err := svc.DeleteProduct(context.Background(), p.ID); err != nil {
		t.Fatalf("DeleteProduct failed: %v", err)
	}

	if _, err := svc.GetProductByID(context.Background(), p.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

// ── SKU CRUD ─────────────────────────────────────────────────────────

func TestSku_Create(t *testing.T) {
	svc := newService(t)

	sku := &Sku{ProductID: 1, Code: "SKU-001", Stock: 10}
	if err := svc.CreateSku(context.Background(), sku); err != nil {
		t.Fatalf("CreateSku failed: %v", err)
	}
	if sku.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestSku_GetByID(t *testing.T) {
	svc := newService(t)

	created := &Sku{ProductID: 1, Code: "SKU-002"}
	if err := svc.CreateSku(context.Background(), created); err != nil {
		t.Fatalf("CreateSku failed: %v", err)
	}

	got, err := svc.GetSkuByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetSkuByID failed: %v", err)
	}
	if got.Code != "SKU-002" {
		t.Fatalf("got Code=%q, want %q", got.Code, "SKU-002")
	}
}

func TestSku_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetSkuByID(context.Background(), 999); err == nil {
		t.Fatal("expected error for non-existent SKU")
	}
}

func TestSku_ListSkusByProduct(t *testing.T) {
	svc := newService(t)

	_ = svc.CreateSku(context.Background(), &Sku{ProductID: 1, Code: "A"})
	_ = svc.CreateSku(context.Background(), &Sku{ProductID: 1, Code: "B"})
	_ = svc.CreateSku(context.Background(), &Sku{ProductID: 2, Code: "C"})

	items, err := svc.ListSkusByProduct(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListSkusByProduct failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d SKUs, want 2", len(items))
	}
}

func TestSku_Update(t *testing.T) {
	svc := newService(t)

	sku := &Sku{ProductID: 1, Code: "OLD-CODE"}
	if err := svc.CreateSku(context.Background(), sku); err != nil {
		t.Fatalf("CreateSku failed: %v", err)
	}

	sku.Code = "NEW-CODE"
	if err := svc.UpdateSku(context.Background(), sku); err != nil {
		t.Fatalf("UpdateSku failed: %v", err)
	}

	got, _ := svc.GetSkuByID(context.Background(), sku.ID)
	if got.Code != "NEW-CODE" {
		t.Fatalf("after update Code=%q, want %q", got.Code, "NEW-CODE")
	}
}

func TestSku_Delete(t *testing.T) {
	svc := newService(t)

	sku := &Sku{ProductID: 1, Code: "DELETE"}
	if err := svc.CreateSku(context.Background(), sku); err != nil {
		t.Fatalf("CreateSku failed: %v", err)
	}

	if err := svc.DeleteSku(context.Background(), sku.ID); err != nil {
		t.Fatalf("DeleteSku failed: %v", err)
	}

	if _, err := svc.GetSkuByID(context.Background(), sku.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

// ── SpecName CRUD ────────────────────────────────────────────────────

func TestSpecName_Create(t *testing.T) {
	svc := newService(t)

	sn := &SpecName{ProductID: 1, Name: "Color"}
	if err := svc.CreateSpecName(context.Background(), sn); err != nil {
		t.Fatalf("CreateSpecName failed: %v", err)
	}
	if sn.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

// SpecName listing uses Preload("Values") which requires GORM relationship
// annotations not present in the SQLite schema — testing just persistence.

// ── SpecValue CRUD ───────────────────────────────────────────────────

func TestSpecValue_Create(t *testing.T) {
	svc := newService(t)

	sv := &SpecValue{SpecNameID: 1, ProductID: 1, Value: "Red"}
	if err := svc.CreateSpecValue(context.Background(), sv); err != nil {
		t.Fatalf("CreateSpecValue failed: %v", err)
	}
	if sv.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}
