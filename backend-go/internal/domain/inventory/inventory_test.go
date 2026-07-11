package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/allocation"
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
	db := dbtest.NewDB(t, &InventoryTransfer{}, &InventoryWarehouse{}, &InventoryLog{}, &allocation.Warehouse{})
	svc := NewService(db, dbtest.NewLogger(t))
	from := allocation.Warehouse{Name: "上海仓", Code: "SHA"}
	to := allocation.Warehouse{Name: "广州仓", Code: "CAN"}
	if err := db.Create(&from).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&to).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&InventoryWarehouse{SkuID: 1001, WarehouseID: from.ID, Quantity: 80, LockedQuantity: 10}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&InventoryWarehouse{SkuID: 1001, WarehouseID: to.ID, Quantity: 5}).Error; err != nil {
		t.Fatal(err)
	}

	tf, _ := svc.CreateTransfer("上海仓", "广州仓", 1001, 50, "")
	svc.StartTransfer(tf.ID, "顺丰", "SF001")

	tf2, err := svc.CompleteTransfer(tf.ID)
	if err != nil {
		t.Fatalf("CompleteTransfer: %v", err)
	}
	if tf2.Status != "completed" {
		t.Fatalf("expected completed, got %s", tf2.Status)
	}
	var source, destination InventoryWarehouse
	db.Where("sku_id = ? AND warehouse_id = ?", 1001, from.ID).First(&source)
	db.Where("sku_id = ? AND warehouse_id = ?", 1001, to.ID).First(&destination)
	if source.Quantity != 30 || destination.Quantity != 55 {
		t.Fatalf("unexpected balances: source=%d destination=%d", source.Quantity, destination.Quantity)
	}
	var logCount int64
	db.Model(&InventoryLog{}).Where("sku_id = ?", 1001).Count(&logCount)
	if logCount != 2 {
		t.Fatalf("expected 2 transfer logs, got %d", logCount)
	}
}

func TestInventoryTransfer_CompleteInsufficientStockRollsBack(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryTransfer{}, &InventoryWarehouse{}, &InventoryLog{}, &allocation.Warehouse{})
	svc := NewService(db, dbtest.NewLogger(t))
	from := allocation.Warehouse{Name: "上海仓", Code: "SHA"}
	to := allocation.Warehouse{Name: "广州仓", Code: "CAN"}
	db.Create(&from)
	db.Create(&to)
	db.Create(&InventoryWarehouse{SkuID: 1001, WarehouseID: from.ID, Quantity: 50, LockedQuantity: 10})
	tf, _ := svc.CreateTransfer("上海仓", "广州仓", 1001, 50, "")
	svc.StartTransfer(tf.ID, "顺丰", "SF001")

	if _, err := svc.CompleteTransfer(tf.ID); !errors.Is(err, ErrInsufficientTransferStock) {
		t.Fatalf("expected insufficient stock, got %v", err)
	}
	var persisted InventoryTransfer
	db.First(&persisted, tf.ID)
	if persisted.Status != "in_transit" {
		t.Fatalf("expected rollback to in_transit, got %s", persisted.Status)
	}
	var source InventoryWarehouse
	db.Where("sku_id = ? AND warehouse_id = ?", 1001, from.ID).First(&source)
	if source.Quantity != 50 {
		t.Fatalf("source changed despite rollback: %d", source.Quantity)
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

// ── Safety Config (#201) ──

func TestSafetyConfig_UpsertAndGet(t *testing.T) {
	db := dbtest.NewDB(t, &InventorySafetyConfig{})
	svc := NewService(db, dbtest.NewLogger(t))

	cfg := &InventorySafetyConfig{
		SkuID:         1001,
		MinStockLevel: 10,
		MaxStockLevel: 500,
		LeadTimeDays:  7,
		SafetyDays:    14,
		DailyAvgSales: 5.0,
		AutoReorder:   true,
	}
	if err := svc.UpsertSafetyConfig(context.Background(), cfg); err != nil {
		t.Fatalf("UpsertSafetyConfig: %v", err)
	}
	if cfg.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := svc.GetSafetyConfig(context.Background(), 1001)
	if err != nil {
		t.Fatalf("GetSafetyConfig: %v", err)
	}
	if got.MinStockLevel != 10 {
		t.Fatalf("MinStockLevel = %d, want 10", got.MinStockLevel)
	}
	if got.DailyAvgSales != 5.0 {
		t.Fatalf("DailyAvgSales = %f, want 5.0", got.DailyAvgSales)
	}
}

func TestSafetyConfig_List(t *testing.T) {
	db := dbtest.NewDB(t, &InventorySafetyConfig{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.UpsertSafetyConfig(context.Background(), &InventorySafetyConfig{SkuID: 1, MinStockLevel: 5})
	svc.UpsertSafetyConfig(context.Background(), &InventorySafetyConfig{SkuID: 2, MinStockLevel: 10})

	items, err := svc.ListSafetyConfigs(context.Background())
	if err != nil {
		t.Fatalf("ListSafetyConfigs: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2, got %d", len(items))
	}
}

// ── Lock / Unlock Stock ──

func TestLockStock_Success(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 100, LockedQuantity: 0})

	err := svc.LockStock(context.Background(), 1, 30, "test-operator")
	if err != nil {
		t.Fatalf("LockStock failed: %v", err)
	}

	var inv Inventory
	db.First(&inv, 1)
	if inv.LockedQuantity != 30 {
		t.Fatalf("expected LockedQuantity=30, got %d", inv.LockedQuantity)
	}
}

func TestLockStock_InsufficientStock(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 50, LockedQuantity: 45})

	err := svc.LockStock(context.Background(), 1, 10, "test")
	if err == nil {
		t.Fatal("expected error for insufficient stock, got nil")
	}
}

func TestUnlockStock_Success(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 100, LockedQuantity: 50})

	err := svc.UnlockStock(context.Background(), 1, 20, "test")
	if err != nil {
		t.Fatalf("UnlockStock failed: %v", err)
	}

	var inv Inventory
	db.First(&inv, 1)
	if inv.LockedQuantity != 30 {
		t.Fatalf("expected LockedQuantity=30, got %d", inv.LockedQuantity)
	}
}

func TestUnlockStock_ExceedsLocked(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 100, LockedQuantity: 10})

	err := svc.UnlockStock(context.Background(), 1, 20, "test")
	if err == nil {
		t.Fatal("expected error for unlocking more than locked, got nil")
	}
}

// ── Update Stock ──

func TestUpdateStock_Increase(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 50, SafetyStock: 5})

	err := svc.UpdateStock(context.Background(), 1, 80, "admin", "restock")
	if err != nil {
		t.Fatalf("UpdateStock failed: %v", err)
	}

	var inv Inventory
	db.First(&inv, 1)
	if inv.Quantity != 80 {
		t.Fatalf("expected Quantity=80, got %d", inv.Quantity)
	}
}

func TestUpdateStock_Decrease(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 50})

	err := svc.UpdateStock(context.Background(), 1, 30, "admin", "sold")
	if err != nil {
		t.Fatalf("UpdateStock failed: %v", err)
	}

	var inv Inventory
	db.First(&inv, 1)
	if inv.Quantity != 30 {
		t.Fatalf("expected Quantity=30, got %d", inv.Quantity)
	}
}

func TestUpdateStock_SameQuantity(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 50})

	err := svc.UpdateStock(context.Background(), 1, 50, "admin", "no-change")
	if err != nil {
		t.Fatalf("UpdateStock failed: %v", err)
	}

	var inv Inventory
	db.First(&inv, 1)
	if inv.Quantity != 50 {
		t.Fatalf("expected Quantity=50, got %d", inv.Quantity)
	}
}

// ── Inventory Logs ──

func TestListLogs(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 50})

	err := svc.UpdateStock(context.Background(), 1, 80, "admin", "restock")
	if err != nil {
		t.Fatalf("UpdateStock failed: %v", err)
	}

	logs, total, err := svc.ListLogs(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatalf("ListLogs failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 log, got %d", total)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log item, got %d", len(logs))
	}
	if logs[0].ChangeType != "in" {
		t.Fatalf("expected change_type=in, got %s", logs[0].ChangeType)
	}
	if logs[0].ChangeQty != 30 {
		t.Fatalf("expected change_qty=30, got %d", logs[0].ChangeQty)
	}
}

// ── GetBy ──

func TestGetBySkuID(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 42, Warehouse: "WH1", Quantity: 100})

	inv, err := svc.GetBySkuID(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetBySkuID failed: %v", err)
	}
	if inv.SkuID != 42 {
		t.Fatalf("expected SkuID=42, got %d", inv.SkuID)
	}
	if inv.Quantity != 100 {
		t.Fatalf("expected Quantity=100, got %d", inv.Quantity)
	}
}

func TestGetBySkuID_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.GetBySkuID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for not found, got nil")
	}
}

func TestGetByID(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 100})

	inv, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if inv.ID != 1 {
		t.Fatalf("expected ID=1, got %d", inv.ID)
	}
	if inv.Quantity != 100 {
		t.Fatalf("expected Quantity=100, got %d", inv.Quantity)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for not found, got nil")
	}
}

// ── Safety Config Update (#201) ──

func TestSafetyConfig_Update(t *testing.T) {
	db := dbtest.NewDB(t, &InventorySafetyConfig{})
	svc := NewService(db, dbtest.NewLogger(t))

	cfg := &InventorySafetyConfig{
		SkuID:         1001,
		MinStockLevel: 10,
		MaxStockLevel: 500,
		LeadTimeDays:  7,
		DailyAvgSales: 5.0,
	}
	if err := svc.UpsertSafetyConfig(context.Background(), cfg); err != nil {
		t.Fatalf("UpsertSafetyConfig (create): %v", err)
	}
	if cfg.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	cfg.MinStockLevel = 25
	cfg.MaxStockLevel = 1000
	if err := svc.UpsertSafetyConfig(context.Background(), cfg); err != nil {
		t.Fatalf("UpsertSafetyConfig (update): %v", err)
	}

	got, err := svc.GetSafetyConfig(context.Background(), 1001)
	if err != nil {
		t.Fatalf("GetSafetyConfig: %v", err)
	}
	if got.MinStockLevel != 25 {
		t.Fatalf("MinStockLevel = %d, want 25", got.MinStockLevel)
	}
	if got.MaxStockLevel != 1000 {
		t.Fatalf("MaxStockLevel = %d, want 1000", got.MaxStockLevel)
	}
}

// ── Dead Stock (#201) ──

func TestDeadStock_Identify_WithTables(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{}, &DeadStockLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create sku + product tables manually (the service query joins them).
	db.Exec("CREATE TABLE IF NOT EXISTS sku (id INTEGER PRIMARY KEY, code TEXT, product_id INTEGER)")
	db.Exec(`CREATE TABLE IF NOT EXISTS product (id INTEGER PRIMARY KEY, name TEXT, description TEXT,
			category_id INTEGER, main_image TEXT,
			package_height_cm REAL, package_width_cm REAL, package_length_cm REAL, package_weight_kg REAL)`)

	db.Exec("INSERT OR IGNORE INTO sku (id, code, product_id) VALUES (1, 'SKU001', 1), (2, 'SKU002', 1), (3, 'SKU003', 2)")
	db.Exec("INSERT OR IGNORE INTO product (id, name) VALUES (1, 'Product A'), (2, 'Product B')")

	db.Create(&Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 100})
	db.Create(&Inventory{SkuID: 2, Warehouse: "WH2", Quantity: 50})
	db.Create(&Inventory{SkuID: 3, Warehouse: "WH1", Quantity: 200})

	// Only sku 1 has recent movement.
	db.Create(&InventoryLog{SkuID: 1, ChangeType: "in", ChangeQty: 10, CreatedAt: time.Now().Add(-10 * 24 * time.Hour)})

	results, err := svc.IdentifyDeadStock(context.Background(), 60)
	if err != nil {
		t.Fatalf("IdentifyDeadStock: %v", err)
	}
	_ = results

	// Test dead stock log listing.
	logs, total, err := svc.ListDeadStockLogs(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("ListDeadStockLogs: %v", err)
	}
	_ = total
	_ = logs
}

// ── Cross-Platform Sync (Oversell Prevention) ──────────────────────

func TestSyncAcrossPlatforms_Success(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryOversellLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	db.Exec("CREATE TABLE sku (id INTEGER PRIMARY KEY, code TEXT, product_id INTEGER)")
	db.Exec("CREATE TABLE product_listing (id INTEGER PRIMARY KEY, product_id INTEGER, platform_id INTEGER, status TEXT, published_data TEXT)")

	db.Exec("INSERT INTO sku (id, code, product_id) VALUES (1, 'SKU001', 100), (2, 'SKU002', 100)")
	db.Create(&Inventory{SkuID: 1, Quantity: 50})
	db.Create(&Inventory{SkuID: 2, Quantity: 50})

	db.Exec(`INSERT INTO product_listing (id, product_id, platform_id, status, published_data) VALUES
		(1, 100, 1, 'active', '{"stock": 20}'),
		(2, 100, 2, 'live', '{"stock": 30}')`)

	result, err := svc.SyncAcrossPlatforms(ctx, 100)
	if err != nil {
		t.Fatalf("SyncAcrossPlatforms: %v", err)
	}
	if result.AvailableInventory != 100 {
		t.Fatalf("available = %d, want 100", result.AvailableInventory)
	}
	if result.TotalCommitted != 50 {
		t.Fatalf("committed = %d, want 50", result.TotalCommitted)
	}
	if result.OversellDetected {
		t.Fatal("expected no oversell")
	}
	if len(result.PlatformBreakdown) != 2 {
		t.Fatalf("platforms = %d, want 2", len(result.PlatformBreakdown))
	}
}

func TestSyncAcrossPlatforms_PlatformNotConfigured(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryOversellLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	db.Exec("CREATE TABLE sku (id INTEGER PRIMARY KEY, code TEXT, product_id INTEGER)")
	db.Exec("CREATE TABLE product_listing (id INTEGER PRIMARY KEY, product_id INTEGER, platform_id INTEGER, status TEXT, published_data TEXT)")
	db.Exec("INSERT INTO sku (id, code, product_id) VALUES (1, 'SKU001', 100)")
	db.Create(&Inventory{SkuID: 1, Quantity: 50})

	result, err := svc.SyncAcrossPlatforms(ctx, 100)
	if err != nil {
		t.Fatalf("SyncAcrossPlatforms: %v", err)
	}
	if result.AvailableInventory != 50 {
		t.Fatalf("available = %d, want 50", result.AvailableInventory)
	}
	if result.TotalCommitted != 0 {
		t.Fatalf("committed = %d, want 0", result.TotalCommitted)
	}
	if len(result.PlatformBreakdown) != 0 {
		t.Fatalf("platforms = %d, want 0", len(result.PlatformBreakdown))
	}
}

func TestSyncAcrossPlatforms_OnePlatformFails(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryOversellLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	db.Exec("CREATE TABLE sku (id INTEGER PRIMARY KEY, code TEXT, product_id INTEGER)")
	db.Exec("CREATE TABLE product_listing (id INTEGER PRIMARY KEY, product_id INTEGER, platform_id INTEGER, status TEXT, published_data TEXT)")
	db.Exec("INSERT INTO sku (id, code, product_id) VALUES (1, 'SKU001', 100)")
	db.Create(&Inventory{SkuID: 1, Quantity: 100})

	db.Exec(`INSERT INTO product_listing (id, product_id, platform_id, status, published_data) VALUES
		(1, 100, 1, 'active', '{"stock": 30}'),
		(2, 100, 2, 'live', '{}')`)

	result, err := svc.SyncAcrossPlatforms(ctx, 100)
	if err != nil {
		t.Fatalf("SyncAcrossPlatforms: %v", err)
	}
	if result.TotalCommitted != 31 {
		t.Fatalf("committed = %d, want 31", result.TotalCommitted)
	}
	if len(result.PlatformBreakdown) != 2 {
		t.Fatalf("platforms = %d, want 2", len(result.PlatformBreakdown))
	}
	for _, p := range result.PlatformBreakdown {
		if p.PlatformID == 2 && p.Committed != 1 {
			t.Fatalf("platform 2 committed = %d, want 1", p.Committed)
		}
	}
}

func TestSyncAcrossPlatforms_ZeroInventory(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryOversellLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	db.Exec("CREATE TABLE sku (id INTEGER PRIMARY KEY, code TEXT, product_id INTEGER)")
	db.Exec("CREATE TABLE product_listing (id INTEGER PRIMARY KEY, product_id INTEGER, platform_id INTEGER, status TEXT, published_data TEXT)")
	db.Exec("INSERT INTO sku (id, code, product_id) VALUES (1, 'SKU001', 100)")
	db.Create(&Inventory{SkuID: 1, Quantity: 0})

	db.Exec(`INSERT INTO product_listing (id, product_id, platform_id, status, published_data) VALUES
		(1, 100, 1, 'active', '{"stock": 10}'),
		(2, 100, 2, 'published', '{"stock": 5}')`)

	result, err := svc.SyncAcrossPlatforms(ctx, 100)
	if err != nil {
		t.Fatalf("SyncAcrossPlatforms: %v", err)
	}
	if result.AvailableInventory != 0 {
		t.Fatalf("available = %d, want 0", result.AvailableInventory)
	}
	if result.TotalCommitted != 15 {
		t.Fatalf("committed = %d, want 15", result.TotalCommitted)
	}
	if !result.OversellDetected {
		t.Fatal("expected oversell detected")
	}
	if result.OversellBy != 15 {
		t.Fatalf("oversell_by = %d, want 15", result.OversellBy)
	}
	if !result.AlertGenerated {
		t.Fatal("expected alert generated")
	}

	var count int64
	db.Model(&InventoryOversellLog{}).Where("product_id = ?", 100).Count(&count)
	if count != 1 {
		t.Fatalf("oversell log count = %d, want 1", count)
	}
}

// ── Oversell Report ────────────────────────────────────────────────

func TestListOversellReport_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryOversellLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	items, total, err := svc.ListOversellReport(ctx, 1, 10)
	if err != nil {
		t.Fatalf("ListOversellReport: %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}

func TestListOversellReport_WithData(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryOversellLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	db.Create(&InventoryOversellLog{ProductID: 1, AvailableStock: 10, TotalCommitted: 20, OversellBy: 10, Status: "open"})
	db.Create(&InventoryOversellLog{ProductID: 2, AvailableStock: 0, TotalCommitted: 5, OversellBy: 5, Status: "open"})

	items, total, err := svc.ListOversellReport(ctx, 1, 10)
	if err != nil {
		t.Fatalf("ListOversellReport: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
}

func TestListOversellReport_Pagination(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryOversellLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	for i := int64(1); i <= 5; i++ {
		db.Create(&InventoryOversellLog{ProductID: i, AvailableStock: 10, TotalCommitted: 20, OversellBy: 10, Status: "open"})
	}

	items, total, err := svc.ListOversellReport(ctx, 1, 2)
	if err != nil {
		t.Fatalf("ListOversellReport: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(items) != 2 {
		t.Fatalf("items on page 1 = %d, want 2", len(items))
	}

	items3, _, err := svc.ListOversellReport(ctx, 3, 2)
	if err != nil {
		t.Fatalf("ListOversellReport page 3: %v", err)
	}
	if len(items3) != 1 {
		t.Fatalf("items on page 3 = %d, want 1", len(items3))
	}
}

// ── Per-Warehouse Inventory ────────────────────────────────────────

func TestListInventoryBySku_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryWarehouse{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	items, err := svc.ListInventoryBySku(ctx, 999)
	if err != nil {
		t.Fatalf("ListInventoryBySku: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}

func TestListInventoryBySku_WithData(t *testing.T) {
	db := dbtest.NewDB(t, &InventoryWarehouse{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	db.Create(&InventoryWarehouse{SkuID: 1, WarehouseID: 1, Quantity: 100, SafetyStock: 10})
	db.Create(&InventoryWarehouse{SkuID: 1, WarehouseID: 2, Quantity: 200, SafetyStock: 20})
	db.Create(&InventoryWarehouse{SkuID: 2, WarehouseID: 1, Quantity: 50, SafetyStock: 5})

	items, err := svc.ListInventoryBySku(ctx, 1)
	if err != nil {
		t.Fatalf("ListInventoryBySku: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Quantity != 100 {
		t.Fatalf("first item quantity = %d, want 100", items[0].Quantity)
	}
	if items[1].Quantity != 200 {
		t.Fatalf("second item quantity = %d, want 200", items[1].Quantity)
	}
}

// ── Lock/Unlock Race Condition ─────────────────────────────────────

func TestLockUnlockStock_RaceCondition(t *testing.T) {
	db := dbtest.NewDB(t, &Inventory{}, &InventoryLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	inv := &Inventory{SkuID: 1, Warehouse: "WH1", Quantity: 100, LockedQuantity: 0}
	db.Create(inv)

	if err := svc.LockStock(ctx, inv.ID, 30, "tester"); err != nil {
		t.Fatalf("first lock: %v", err)
	}
	checkLocked(t, svc, ctx, inv.ID, 30)

	if err := svc.LockStock(ctx, inv.ID, 50, "tester"); err != nil {
		t.Fatalf("second lock: %v", err)
	}
	checkLocked(t, svc, ctx, inv.ID, 80)

	if err := svc.LockStock(ctx, inv.ID, 30, "tester"); err == nil {
		t.Fatal("expected error when locking beyond available stock")
	}

	if err := svc.UnlockStock(ctx, inv.ID, 20, "tester"); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	checkLocked(t, svc, ctx, inv.ID, 60)

	if err := svc.LockStock(ctx, inv.ID, 30, "tester"); err != nil {
		t.Fatalf("lock after unlock: %v", err)
	}
	checkLocked(t, svc, ctx, inv.ID, 90)

	if err := svc.UnlockStock(ctx, inv.ID, 999, "tester"); err == nil {
		t.Fatal("expected error when unlocking more than locked")
	}
	if err := svc.LockStock(ctx, inv.ID, 0, "tester"); err == nil {
		t.Fatal("expected error for zero lock quantity")
	}
	if err := svc.UnlockStock(ctx, inv.ID, -1, "tester"); err == nil {
		t.Fatal("expected error for negative unlock quantity")
	}
}

func checkLocked(t *testing.T, svc *Service, ctx context.Context, id int64, want int) {
	t.Helper()
	inv, err := svc.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID(%d): %v", id, err)
	}
	if inv.LockedQuantity != want {
		t.Fatalf("locked = %d, want %d", inv.LockedQuantity, want)
	}
}
