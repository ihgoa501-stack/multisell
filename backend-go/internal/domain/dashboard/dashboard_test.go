package dashboard

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

// ── lightweight stub models matching the table names the dashboard service queries ──

type testOrder struct {
	ID           int64   `gorm:"column:id;primaryKey;autoIncrement"`
	Status       string  `gorm:"column:status"`
	PayAmount    float64 `gorm:"column:pay_amount"`
	ProfitAmount float64 `gorm:"column:profit_amount"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (testOrder) TableName() string { return "sales_order" }

type testSku struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement"`
	ProductID    int64 `gorm:"column:product_id"`
	Code         string `gorm:"column:code"`
	SpecDesc     string `gorm:"column:spec_desc"`
	Stock        int   `gorm:"column:stock"`
	WarningStock int   `gorm:"column:warning_stock"`
}

func (testSku) TableName() string { return "sku" }

type testListing struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Status    string `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (testListing) TableName() string { return "product_listing" }

type testAfterSales struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Status    string `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (testAfterSales) TableName() string { return "after_sales_order" }

type testException struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Severity     string `gorm:"column:severity"`
	SourceModule string `gorm:"column:source_module"`
	Status       string `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (testException) TableName() string { return "exception_item" }

type testLedgerEntry struct {
	ID        int64   `gorm:"column:id;primaryKey;autoIncrement"`
	EntryType string  `gorm:"column:entry_type"`
	Amount    float64 `gorm:"column:amount"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (testLedgerEntry) TableName() string { return "finance_ledger_entry" }

type testInventory struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement"`
	SkuID     int64  `gorm:"column:sku_id"`
	Warehouse string `gorm:"column:warehouse"`
}

func (testInventory) TableName() string { return "inventory" }

// ── helpers ──

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t,
		&testOrder{},
		&testSku{},
		&testListing{},
		&testAfterSales{},
		&testException{},
		&testLedgerEntry{},
		&testInventory{},
	)
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func seedOverviewData(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()

	// orders
	db.Create(&testOrder{Status: "pending", PayAmount: 100, ProfitAmount: 10, CreatedAt: now.AddDate(0, 0, -1)})
	db.Create(&testOrder{Status: "completed", PayAmount: 200, ProfitAmount: 30, CreatedAt: now})
	db.Create(&testOrder{Status: "completed", PayAmount: 150, ProfitAmount: 20, CreatedAt: now})

	// skus
	db.Create(&testSku{ProductID: 1, Code: "SKU-001", Stock: 50, WarningStock: 10})
	db.Create(&testSku{ProductID: 1, Code: "SKU-002", Stock: 5, WarningStock: 10})  // low stock
	db.Create(&testSku{ProductID: 2, Code: "SKU-003", Stock: 0, WarningStock: 0})  // out of stock
	db.Create(&testSku{ProductID: 2, Code: "SKU-004", Stock: 3, WarningStock: 5})  // low stock

	// listings
	db.Create(&testListing{Status: "online", CreatedAt: now})
	db.Create(&testListing{Status: "draft", CreatedAt: now})

	// aftersales
	db.Create(&testAfterSales{Status: "pending", CreatedAt: now})
	db.Create(&testAfterSales{Status: "approved", CreatedAt: now})

	// exceptions
	db.Create(&testException{Severity: "high", SourceModule: "order", Status: "open", CreatedAt: now})
	db.Create(&testException{Severity: "medium", SourceModule: "inventory", Status: "resolved", CreatedAt: now})

	// ledger
	db.Create(&testLedgerEntry{EntryType: "revenue", Amount: 500, CreatedAt: now})
	db.Create(&testLedgerEntry{EntryType: "product_cost", Amount: 200, CreatedAt: now})
	db.Create(&testLedgerEntry{EntryType: "shipping_cost", Amount: 50, CreatedAt: now})

	// inventory
	db.Create(&testInventory{SkuID: 1, Warehouse: "Main"})
	db.Create(&testInventory{SkuID: 2, Warehouse: "Main"})
}

// ── tests ──

func TestOverview_ReturnsCounts(t *testing.T) {
	svc := newService(t)
	seedOverviewData(t, svc.db)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.OrderTotal != 3 {
		t.Fatalf("OrderTotal=%d, want 3", o.OrderTotal)
	}
	if o.SkuTotal != 4 {
		t.Fatalf("SkuTotal=%d, want 4", o.SkuTotal)
	}
}

func TestOverview_OrderByStatus(t *testing.T) {
	svc := newService(t)
	seedOverviewData(t, svc.db)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.OrderByStatus["pending"] != 1 {
		t.Fatalf("pending=%d, want 1", o.OrderByStatus["pending"])
	}
	if o.OrderByStatus["completed"] != 2 {
		t.Fatalf("completed=%d, want 2", o.OrderByStatus["completed"])
	}
}

func TestOverview_Revenue(t *testing.T) {
	svc := newService(t)
	seedOverviewData(t, svc.db)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.OrderRevenue != 450 {
		t.Fatalf("OrderRevenue=%v, want 450", o.OrderRevenue)
	}
	if o.OrderProfit != 60 {
		t.Fatalf("OrderProfit=%v, want 60", o.OrderProfit)
	}
}

func TestOverview_LowStockCount(t *testing.T) {
	svc := newService(t)
	seedOverviewData(t, svc.db)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.LowStockCount != 2 {
		t.Fatalf("LowStockCount=%d, want 2", o.LowStockCount)
	}
}

func TestOverview_OutOfStockCount(t *testing.T) {
	svc := newService(t)
	seedOverviewData(t, svc.db)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.OutOfStockCount != 1 {
		t.Fatalf("OutOfStockCount=%d, want 1", o.OutOfStockCount)
	}
}

func TestOverview_ListingActiveCount(t *testing.T) {
	svc := newService(t)
	seedOverviewData(t, svc.db)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.ListingActiveCount != 1 {
		t.Fatalf("ListingActiveCount=%d, want 1", o.ListingActiveCount)
	}
}

func TestOverview_AftersalesPendingCount(t *testing.T) {
	svc := newService(t)
	seedOverviewData(t, svc.db)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.AftersalesPendingCount != 1 {
		t.Fatalf("AftersalesPendingCount=%d, want 1", o.AftersalesPendingCount)
	}
}

func TestOverview_ExceptionOpenCount(t *testing.T) {
	svc := newService(t)
	seedOverviewData(t, svc.db)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.ExceptionOpenCount != 1 {
		t.Fatalf("ExceptionOpenCount=%d, want 1", o.ExceptionOpenCount)
	}
}

func TestOverview_EmptyDB(t *testing.T) {
	svc := newService(t)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if o.OrderTotal != 0 || o.SkuTotal != 0 || o.LowStockCount != 0 {
		t.Fatalf("expected all zeros for empty DB, got orders=%d skus=%d low=%d",
			o.OrderTotal, o.SkuTotal, o.LowStockCount)
	}
}

func TestOrdersTrend_WithData(t *testing.T) {
	// DATE(created_at) returns a string in SQLite but the service scans into time.Time.
	// This test validates the PostgreSQL code path; skip on SQLite.
	t.Skip("requires PostgreSQL (DATE() scan into time.Time)")
}

func TestOrdersTrend_EmptyDB(t *testing.T) {
	svc := newService(t)

	points, err := svc.OrdersTrend(7)
	if err != nil {
		t.Fatalf("OrdersTrend failed: %v", err)
	}
	if len(points) != 0 {
		t.Fatalf("expected 0 points, got %d", len(points))
	}
}

func TestInventoryHealth_WithData(t *testing.T) {
	svc := newService(t)

	svc.db.Create(&testSku{ProductID: 1, Code: "SKU-L1", SpecDesc: "Red/L", Stock: 2, WarningStock: 10})
	svc.db.Create(&testSku{ProductID: 1, Code: "SKU-L2", SpecDesc: "Blue/M", Stock: 8, WarningStock: 10})
	svc.db.Create(&testSku{ProductID: 2, Code: "SKU-OK", SpecDesc: "Green/S", Stock: 50, WarningStock: 10})
	svc.db.Create(&testInventory{SkuID: 1, Warehouse: "WH-A"})

	items, err := svc.InventoryHealth(20)
	if err != nil {
		t.Fatalf("InventoryHealth failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
	// should be ordered by stock ASC
	if items[0].Stock > items[1].Stock {
		t.Fatalf("expected ascending stock order: %d > %d", items[0].Stock, items[1].Stock)
	}
}

func TestInventoryHealth_EmptyDB(t *testing.T) {
	svc := newService(t)

	items, err := svc.InventoryHealth(20)
	if err != nil {
		t.Fatalf("InventoryHealth failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestExceptionDistribution_WithData(t *testing.T) {
	svc := newService(t)
	now := time.Now()

	svc.db.Create(&testException{Severity: "high", SourceModule: "order", Status: "open", CreatedAt: now})
	svc.db.Create(&testException{Severity: "high", SourceModule: "order", Status: "open", CreatedAt: now})
	svc.db.Create(&testException{Severity: "medium", SourceModule: "inventory", Status: "open", CreatedAt: now})

	dist, err := svc.ExceptionDistribution()
	if err != nil {
		t.Fatalf("ExceptionDistribution failed: %v", err)
	}
	if len(dist) != 2 {
		t.Fatalf("len(dist)=%d, want 2", len(dist))
	}
	// first entry should be high/order with count 2
	if dist[0].Severity != "high" || dist[0].Cnt != 2 {
		t.Fatalf("first entry: severity=%q cnt=%d, want high/2", dist[0].Severity, dist[0].Cnt)
	}
}

func TestExceptionDistribution_EmptyDB(t *testing.T) {
	svc := newService(t)

	dist, err := svc.ExceptionDistribution()
	if err != nil {
		t.Fatalf("ExceptionDistribution failed: %v", err)
	}
	if len(dist) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(dist))
	}
}
