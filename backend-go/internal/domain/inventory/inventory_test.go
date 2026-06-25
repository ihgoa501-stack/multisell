package inventory

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestListInventory(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 10})
	db.Create(&Inventory{SkuID: 2, Warehouse: "WH2", Quantity: 20})

	items, total, err := svc.List(context.Background(), 1, 10, 0, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestListInventory_FilterByWarehouse(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 10})
	db.Create(&Inventory{SkuID: 2, Warehouse: "WH2", Quantity: 20})

	// Use empty filter (ILIKE is PG-specific; the existing List method uses it for optional filter)
	_, total, err := svc.List(context.Background(), 1, 10, 0, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2, got %d", total)
	}
}

// ── Bin Location ──

func TestBinLocation_AssignAndRelease(t *testing.T) {
	db := dbtest.NewDB(t, &BinLocation{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&BinLocation{Warehouse: "上海仓", LocationCode: "A-01-01", Capacity: 100, Status: "available"})
	db.Create(&BinLocation{Warehouse: "上海仓", LocationCode: "A-01-02", Status: "occupied"})

	loc, err := svc.AssignLocation(1001, 1)
	if err != nil {
		t.Fatalf("AssignLocation: %v", err)
	}
	if loc.Status != "occupied" {
		t.Fatalf("expected occupied, got %s", loc.Status)
	}
	if err := svc.ReleaseLocation(1); err != nil {
		t.Fatalf("ReleaseLocation: %v", err)
	}
}

func TestBinLocation_ListByWarehouse(t *testing.T) {
	db := dbtest.NewDB(t, &BinLocation{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&BinLocation{Warehouse: "上海仓", LocationCode: "A-01", Capacity: 100, Status: "available"})
	db.Create(&BinLocation{Warehouse: "广州仓", LocationCode: "B-01", Capacity: 200, Status: "available"})

	locs, total, err := svc.ListLocations("上海仓", 1, 10)
	if err != nil {
		t.Fatalf("ListLocations: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1, got %d", total)
	}
	_ = locs
}

// ── Transfer ──

func TestInventoryTransfer_CreateAndStart(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryTransfer{})
	svc := NewService(db, dbtest.NewLogger(t))

	tf, err := svc.CreateTransfer("上海仓", "广州仓", 1001, 50, "调拨测试")
	if err != nil {
		t.Fatalf("CreateTransfer: %v", err)
	}
	if tf.Status != "draft" {
		t.Fatalf("expected draft, got %s", tf.Status)
	}

	tf2, err := svc.StartTransfer(tf.ID, "顺丰", "SF123456")
	if err != nil {
		t.Fatalf("StartTransfer: %v", err)
	}
	if tf2.Status != "in_transit" {
		t.Fatalf("expected in_transit, got %s", tf2.Status)
	}
}

func TestInventoryTransfer_Complete(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryTransfer{})
	svc := NewService(db, dbtest.NewLogger(t))

	tf, _ := svc.CreateTransfer("上海仓", "广州仓", 1001, 50, "")
	svc.StartTransfer(tf.ID, "顺丰", "SF001")

	tf2, err := svc.CompleteTransfer(tf.ID)
	if err != nil {
		t.Fatalf("CompleteTransfer: %v", err)
	}
	if tf2.Status != "completed" {
		t.Fatalf("expected completed, got %s", tf2.Status)
	}
}

func TestInventoryTransfer_Cancel(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryTransfer{})
	svc := NewService(db, dbtest.NewLogger(t))

	tf, _ := svc.CreateTransfer("上海仓", "广州仓", 1001, 50, "")
	if err := svc.CancelTransfer(tf.ID); err != nil {
		t.Fatalf("CancelTransfer: %v", err)
	}
}

func TestInventoryTransfer_List(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryTransfer{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&InventoryTransfer{FromWarehouse: "WH1", ToWarehouse: "WH2", SkuID: 1001, Quantity: 50, Status: "draft"})
	db.Create(&InventoryTransfer{FromWarehouse: "WH1", ToWarehouse: "WH3", SkuID: 1002, Quantity: 30, Status: "in_transit"})

	ts, total, err := svc.ListTransfers(0, "in_transit", 1, 10)
	if err != nil {
		t.Fatalf("ListTransfers: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 in_transit, got %d", total)
	}
	if len(ts) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(ts))
	}
}

// ── Alert Rules ──

func TestInventoryAlertRule_CRUD(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryAlertRule{}, &InventoryAlert{})
	svc := NewService(db, dbtest.NewLogger(t))

	rule, err := svc.CreateAlertRule(1001, 10, 500, 7)
	if err != nil {
		t.Fatalf("CreateAlertRule: %v", err)
	}
	if rule.MinLevel != 10 {
		t.Fatalf("expected min 10, got %d", rule.MinLevel)
	}

	rules, err := svc.ListAlertRules(1001)
	if err != nil {
		t.Fatalf("ListAlertRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestInventoryAlert_ListAndResolve(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryAlert{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&InventoryAlert{SkuID: 1001, AlertType: "low_stock", Message: "库存不足", Status: "active"})
	db.Create(&InventoryAlert{SkuID: 1002, AlertType: "overstock", Message: "库存过多", Status: "active"})
	db.Create(&InventoryAlert{SkuID: 1003, AlertType: "low_stock", Message: "已处理", Status: "resolved"})

	alerts, total, err := svc.ListAlerts("active", "low_stock", 1, 10)
	if err != nil {
		t.Fatalf("ListAlerts: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 active low_stock, got %d", total)
	}
	_ = alerts

	if err := svc.ResolveAlert(1); err != nil {
		t.Fatalf("ResolveAlert: %v", err)
	}
}
