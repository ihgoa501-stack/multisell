package inventory

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Inventory{}, &InventoryLog{}, &Warehouse{}, &InventoryWarehouse{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

// ── Inventory CRUD ──────────────────────────────────────────────────

func TestInventory_Create(t *testing.T) {
	db := newTestDB(t)
	inv := &Inventory{SkuID: 1, Quantity: 100, SafetyStock: 10}
	if err := db.Create(inv).Error; err != nil {
		t.Fatalf("create inventory failed: %v", err)
	}
	if inv.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestInventory_GetBySkuID(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	_ = db.Create(&Inventory{SkuID: 1, Quantity: 50})

	got, err := svc.GetBySkuID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetBySkuID failed: %v", err)
	}
	if got.Quantity != 50 {
		t.Fatalf("Quantity=%d, want 50", got.Quantity)
	}
}

func TestInventory_GetBySkuID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetBySkuID(context.Background(), 999); err == nil {
		t.Fatal("expected error for non-existent SKU")
	}
}

func TestInventory_GetByID(t *testing.T) {
	svc := newService(t)

	inv := &Inventory{SkuID: 1, Quantity: 30}
	_ = svc.db.Create(inv)

	got, err := svc.GetByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Quantity != 30 {
		t.Fatalf("Quantity=%d, want 30", got.Quantity)
	}
}

func TestInventory_List(t *testing.T) {
	svc := newService(t)

	_ = svc.db.Create(&Inventory{SkuID: 1, Quantity: 10})
	_ = svc.db.Create(&Inventory{SkuID: 2, Quantity: 20})

	items, total, err := svc.List(context.Background(), 1, 10, 0, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("items=%d, want 2", len(items))
	}
}

func TestInventory_List_FilterBySkuID(t *testing.T) {
	svc := newService(t)

	_ = svc.db.Create(&Inventory{SkuID: 1, Quantity: 10})
	_ = svc.db.Create(&Inventory{SkuID: 2, Quantity: 20})

	items, total, err := svc.List(context.Background(), 1, 10, 1, "")
	if err != nil {
		t.Fatalf("List by skuID failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 || items[0].SkuID != 1 {
		t.Fatalf("expected 1 item with SkuID=1, got %+v", items)
	}
}

func TestInventory_UpdateStock(t *testing.T) {
	svc := newService(t)

	inv := &Inventory{SkuID: 1, Quantity: 50}
	_ = svc.db.Create(inv)

	if err := svc.UpdateStock(context.Background(), inv.ID, 75, "admin", "restock"); err != nil {
		t.Fatalf("UpdateStock failed: %v", err)
	}

	updated, _ := svc.GetByID(context.Background(), inv.ID)
	if updated.Quantity != 75 {
		t.Fatalf("after UpdateStock Quantity=%d, want 75", updated.Quantity)
	}
}

func TestInventory_LockStock(t *testing.T) {
	svc := newService(t)

	inv := &Inventory{SkuID: 1, Quantity: 100, LockedQuantity: 0}
	_ = svc.db.Create(inv)

	if err := svc.LockStock(context.Background(), inv.ID, 30, "admin"); err != nil {
		t.Fatalf("LockStock failed: %v", err)
	}

	updated, _ := svc.GetByID(context.Background(), inv.ID)
	if updated.LockedQuantity != 30 {
		t.Fatalf("after LockStock LockedQuantity=%d, want 30", updated.LockedQuantity)
	}
}

func TestInventory_LockStock_Insufficient(t *testing.T) {
	svc := newService(t)

	inv := &Inventory{SkuID: 1, Quantity: 10, LockedQuantity: 0}
	_ = svc.db.Create(inv)

	if err := svc.LockStock(context.Background(), inv.ID, 100, "admin"); err == nil {
		t.Fatal("expected error for insufficient stock")
	}
}

func TestInventory_UnlockStock(t *testing.T) {
	svc := newService(t)

	inv := &Inventory{SkuID: 1, Quantity: 100, LockedQuantity: 50}
	_ = svc.db.Create(inv)

	if err := svc.UnlockStock(context.Background(), inv.ID, 20, "admin"); err != nil {
		t.Fatalf("UnlockStock failed: %v", err)
	}

	updated, _ := svc.GetByID(context.Background(), inv.ID)
	if updated.LockedQuantity != 30 {
		t.Fatalf("after UnlockStock LockedQuantity=%d, want 30", updated.LockedQuantity)
	}
}

func TestInventory_UnlockStock_Exceeds(t *testing.T) {
	svc := newService(t)

	inv := &Inventory{SkuID: 1, Quantity: 100, LockedQuantity: 50}
	_ = svc.db.Create(inv)

	if err := svc.UnlockStock(context.Background(), inv.ID, 100, "admin"); err == nil {
		t.Fatal("expected error when unlocking more than locked")
	}
}

// ── Inventory Logs ──────────────────────────────────────────────────

func TestInventory_ListLogs(t *testing.T) {
	svc := newService(t)

	// UpdateStock creates logs automatically
	inv := &Inventory{SkuID: 1, Quantity: 50}
	_ = svc.db.Create(inv)
	_ = svc.UpdateStock(context.Background(), inv.ID, 100, "admin", "restock")

	logs, total, err := svc.ListLogs(context.Background(), 0, 1, 20)
	if err != nil {
		t.Fatalf("ListLogs failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(logs) != 1 {
		t.Fatalf("logs=%d, want 1", len(logs))
	}
}

// ── Warehouse CRUD ──────────────────────────────────────────────────

func TestWarehouse_Create(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "Main"}
	if err := svc.CreateWarehouse(context.Background(), w); err != nil {
		t.Fatalf("CreateWarehouse failed: %v", err)
	}
	if w.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestWarehouse_Create_EmptyName(t *testing.T) {
	svc := newService(t)
	if err := svc.CreateWarehouse(context.Background(), &Warehouse{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestWarehouse_GetByID(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "Findable"}
	_ = svc.CreateWarehouse(context.Background(), w)

	got, err := svc.GetWarehouseByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("GetWarehouseByID failed: %v", err)
	}
	if got.Name != "Findable" {
		t.Fatalf("Name=%q, want Findable", got.Name)
	}
}

func TestWarehouse_Update(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "Old"}
	_ = svc.CreateWarehouse(context.Background(), w)

	w.Name = "Updated"
	if err := svc.UpdateWarehouse(context.Background(), w); err != nil {
		t.Fatalf("UpdateWarehouse failed: %v", err)
	}

	got, _ := svc.GetWarehouseByID(context.Background(), w.ID)
	if got.Name != "Updated" {
		t.Fatalf("Name=%q, want Updated", got.Name)
	}
}

func TestWarehouse_Delete(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "To Delete"}
	_ = svc.CreateWarehouse(context.Background(), w)

	if err := svc.DeleteWarehouse(context.Background(), w.ID); err != nil {
		t.Fatalf("DeleteWarehouse failed: %v", err)
	}
	if _, err := svc.GetWarehouseByID(context.Background(), w.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}
