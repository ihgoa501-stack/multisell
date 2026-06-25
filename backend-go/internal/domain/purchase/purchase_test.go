package purchase

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Supplier{}, &PurchaseOrder{}, &PurchaseOrderItem{}, &PurchaseSuggestion{})
}

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return dbtest.NewLogger(t)
}

func seedSupplier(t *testing.T, db *gorm.DB) *Supplier {
	t.Helper()
	sup := Supplier{
		Name:          "Test Supplier",
		ContactPerson: "Zhang San",
		Phone:         "13800138000",
		Email:         "test@supplier.com",
	}
	if err := db.Create(&sup).Error; err != nil {
		t.Fatalf("seed supplier: %v", err)
	}
	return &sup
}

// ---------- Tests ----------

func TestService_CreateOrder(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))
	sup := seedSupplier(t, db)

	o, err := svc.CreateOrder(&CreateOrderInput{
		SupplierID: sup.ID,
		Items: []OrderItemInput{
			{SkuID: 1, Quantity: 10, UnitPrice: 15.5},
			{SkuID: 2, Quantity: 5, UnitPrice: 30.0},
		},
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if o.ID == 0 {
		t.Fatal("ID should be set")
	}
	if o.Status != StatusDraft {
		t.Fatalf("status = %s, want draft", o.Status)
	}
	if o.OrderNo == "" {
		t.Fatal("OrderNo should be generated")
	}
	if len(o.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(o.Items))
	}
	// Total: 10*15.5 + 5*30 = 155 + 150 = 305
	if o.TotalAmount != 305.0 {
		t.Fatalf("total = %v, want 305", o.TotalAmount)
	}
}

func TestService_ApproveOrder(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))
	sup := seedSupplier(t, db)

	o, _ := svc.CreateOrder(&CreateOrderInput{
		SupplierID: sup.ID,
		Items:      []OrderItemInput{{SkuID: 1, Quantity: 10, UnitPrice: 10}},
	})
	approved, err := svc.ApproveOrder(o.ID)
	if err != nil {
		t.Fatalf("ApproveOrder: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Fatalf("status = %s, want approved", approved.Status)
	}
}

func TestService_ReceiveOrder_Full(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))
	sup := seedSupplier(t, db)

	o, _ := svc.CreateOrder(&CreateOrderInput{
		SupplierID: sup.ID,
		Items:      []OrderItemInput{{SkuID: 1, Quantity: 10, UnitPrice: 10}},
	})
	_, _ = svc.ApproveOrder(o.ID)

	reloaded, err := svc.ReceiveOrder(o.ID, &ReceiveOrderInput{
		Items: []ReceiveItemInput{{ItemID: o.Items[0].ID, ReceivedQty: 10}},
	})
	if err != nil {
		t.Fatalf("ReceiveOrder: %v", err)
	}
	if reloaded.Status != StatusCompleted {
		t.Fatalf("status = %s, want completed", reloaded.Status)
	}
	if reloaded.Items[0].ReceivedQty != 10 {
		t.Fatalf("received_qty = %d, want 10", reloaded.Items[0].ReceivedQty)
	}
}

func TestService_ReceiveOrder_Partial(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))
	sup := seedSupplier(t, db)

	o, _ := svc.CreateOrder(&CreateOrderInput{
		SupplierID: sup.ID,
		Items:      []OrderItemInput{{SkuID: 1, Quantity: 10, UnitPrice: 10}},
	})
	_, _ = svc.ApproveOrder(o.ID)

	// Partial receipt
	reloaded, err := svc.ReceiveOrder(o.ID, &ReceiveOrderInput{
		Items: []ReceiveItemInput{{ItemID: o.Items[0].ID, ReceivedQty: 4}},
	})
	if err != nil {
		t.Fatalf("ReceiveOrder (partial): %v", err)
	}
	if reloaded.Status != StatusPartiallyReceived {
		t.Fatalf("status = %s, want partially_received", reloaded.Status)
	}
	if reloaded.Items[0].ReceivedQty != 4 {
		t.Fatalf("received_qty = %d, want 4", reloaded.Items[0].ReceivedQty)
	}

	// Complete remaining
	reloaded, err = svc.ReceiveOrder(o.ID, &ReceiveOrderInput{
		Items: []ReceiveItemInput{{ItemID: o.Items[0].ID, ReceivedQty: 6}},
	})
	if err != nil {
		t.Fatalf("ReceiveOrder (complete): %v", err)
	}
	if reloaded.Status != StatusCompleted {
		t.Fatalf("status = %s, want completed", reloaded.Status)
	}
	if reloaded.Items[0].ReceivedQty != 10 {
		t.Fatalf("received_qty = %d, want 10", reloaded.Items[0].ReceivedQty)
	}
}

func TestService_CancelOrder(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))
	sup := seedSupplier(t, db)

	o, _ := svc.CreateOrder(&CreateOrderInput{
		SupplierID: sup.ID,
		Items:      []OrderItemInput{{SkuID: 1, Quantity: 10, UnitPrice: 10}},
	})
	cancelled, err := svc.CancelOrder(o.ID)
	if err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Fatalf("status = %s, want cancelled", cancelled.Status)
	}
}

func TestService_CreateOrder_SupplierNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))

	_, err := svc.CreateOrder(&CreateOrderInput{
		SupplierID: 999,
		Items:      []OrderItemInput{{SkuID: 1, Quantity: 10, UnitPrice: 10}},
	})
	if err == nil {
		t.Fatal("expected error for non-existent supplier")
	}
}

func TestService_ListOrders(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))
	sup := seedSupplier(t, db)

	_, _ = svc.CreateOrder(&CreateOrderInput{
		SupplierID: sup.ID,
		Items:      []OrderItemInput{{SkuID: 1, Quantity: 5, UnitPrice: 10}},
	})
	_, _ = svc.CreateOrder(&CreateOrderInput{
		SupplierID: sup.ID,
		Items:      []OrderItemInput{{SkuID: 2, Quantity: 3, UnitPrice: 20}},
	})

	p := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListOrders(&p, &PurchaseOrderListFilter{})
	if err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("total=%d len=%d, want 2", total, len(items))
	}
}

func TestService_InvalidTransitions(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))
	sup := seedSupplier(t, db)

	o, _ := svc.CreateOrder(&CreateOrderInput{
		SupplierID: sup.ID,
		Items:      []OrderItemInput{{SkuID: 1, Quantity: 10, UnitPrice: 10}},
	})

	// Cannot receive a draft order
	_, err := svc.ReceiveOrder(o.ID, &ReceiveOrderInput{
		Items: []ReceiveItemInput{{ItemID: o.Items[0].ID, ReceivedQty: 5}},
	})
	if err == nil {
		t.Fatal("expected error receiving draft order")
	}

	// Cancel then try to approve
	_, _ = svc.CancelOrder(o.ID)
	_, err = svc.ApproveOrder(o.ID)
	if err == nil {
		t.Fatal("expected error approving cancelled order")
	}
}

func TestService_GenerateSuggestions(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))

	// Seed inventory data into the shared "inventory" table.
	type InventoryRow struct {
		SkuID       int64 `gorm:"column:sku_id"`
		Quantity    int   `gorm:"column:quantity"`
		SafetyStock int   `gorm:"column:safety_stock"`
	}
	if err := db.Table("inventory").AutoMigrate(&InventoryRow{}); err != nil {
		t.Fatalf("migrate inventory: %v", err)
	}
	db.Table("inventory").Create(&InventoryRow{SkuID: 1, Quantity: 5, SafetyStock: 20})  // below safety stock
	db.Table("inventory").Create(&InventoryRow{SkuID: 2, Quantity: 30, SafetyStock: 10}) // OK
	db.Table("inventory").Create(&InventoryRow{SkuID: 3, Quantity: 0, SafetyStock: 15})  // out of stock

	suggestions, err := svc.GenerateSuggestions()
	if err != nil {
		t.Fatalf("GenerateSuggestions: %v", err)
	}
	// 2 suggestions: sku 1 and sku 3 have low stock
	if len(suggestions) != 2 {
		t.Fatalf("suggestions = %d, want 2", len(suggestions))
	}

	found := make(map[int64]bool)
	for _, s := range suggestions {
		found[s.SkuID] = true
		if s.SuggestedQty <= 0 {
			t.Fatalf("sku %d: suggested_qty = %d", s.SkuID, s.SuggestedQty)
		}
	}
	if !found[1] {
		t.Fatal("sku 1 should have a suggestion")
	}
	if !found[3] {
		t.Fatal("sku 3 should have a suggestion")
	}
	// Sku 2 has enough stock, should not have suggestion
	if found[2] {
		t.Fatal("sku 2 should not have a suggestion")
	}
}

func TestService_SupplierCRUD(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, newTestLogger(t))

	// Create
	sup, err := svc.CreateSupplier(&CreateSupplierInput{
		Name:          "New Supplier",
		ContactPerson: "Li Si",
		Phone:         "13900139000",
	})
	if err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}
	if sup.ID == 0 {
		t.Fatal("ID should be set")
	}

	// Get
	fetched, err := svc.GetSupplier(sup.ID)
	if err != nil {
		t.Fatalf("GetSupplier: %v", err)
	}
	if fetched.Name != "New Supplier" {
		t.Fatalf("name = %s", fetched.Name)
	}

	// Update
	newName := "Updated Supplier"
	updated, err := svc.UpdateSupplier(sup.ID, &UpdateSupplierInput{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateSupplier: %v", err)
	}
	if updated.Name != "Updated Supplier" {
		t.Fatalf("name = %s", updated.Name)
	}

	// List
	p := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListSuppliers(&p, "")
	if err != nil {
		t.Fatalf("ListSuppliers: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("total=%d len=%d, want 1", total, len(items))
	}

	// KPI
	kpi, err := svc.GetSupplierKPI(sup.ID)
	if err != nil {
		t.Fatalf("GetSupplierKPI: %v", err)
	}
	if kpi.SupplierID != sup.ID {
		t.Fatalf("kpi supplier_id = %d", kpi.SupplierID)
	}

	// Delete
	if err := svc.DeleteSupplier(sup.ID); err != nil {
		t.Fatalf("DeleteSupplier: %v", err)
	}
	_, err = svc.GetSupplier(sup.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
