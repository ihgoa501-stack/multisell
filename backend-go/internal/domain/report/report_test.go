package report

import (
	"math"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ---------------------------------------------------------------------------
// Inline models matching the table schemas used by report queries.
// Each corresponds to a table the Service queries by raw Table("...") calls.
// ---------------------------------------------------------------------------

type testSalesOrder struct {
	ID           int64     `gorm:"primaryKey"`
	CreatedAt    time.Time `gorm:"index"`
	PlatformID   *int64
	PayAmount    float64
	ProfitAmount float64
	Status       string
}

func (testSalesOrder) TableName() string { return "sales_order" }

type testSku struct {
	ID           int64   `gorm:"primaryKey"`
	ProductID    int64   `gorm:"index"`
	Code         string
	SpecDesc     string
	Stock        int
	WarningStock int
	CostPrice    float64
}

func (testSku) TableName() string { return "sku" }

type testInventory struct {
	ID        int64  `gorm:"primaryKey"`
	SkuID     int64  `gorm:"index"`
	Warehouse string
	Quantity  int
}

func (testInventory) TableName() string { return "inventory" }

type testSettlement struct {
	ID         int64     `gorm:"primaryKey"`
	CreatedAt  time.Time `gorm:"index"`
	PlatformID *int64
	TotalNet   float64
}

func (testSettlement) TableName() string { return "settlement" }

type testSettlementItem struct {
	ID                   int64  `gorm:"primaryKey"`
	SettlementID         int64  `gorm:"index"`
	ReconciliationStatus string
}

func (testSettlementItem) TableName() string { return "settlement_item" }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertFloatEqual(t *testing.T, want, got, delta float64) {
	t.Helper()
	if math.Abs(want-got) > delta {
		t.Fatalf("expected %v, got %v (delta %v)", want, got, delta)
	}
}

// ---------------------------------------------------------------------------
// 1. TestNewService — service creation
// ---------------------------------------------------------------------------

func TestNewService(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if svc.db != db {
		t.Fatal("db not stored")
	}
	if svc.logger != logger {
		t.Fatal("logger not stored")
	}
}

// ---------------------------------------------------------------------------
// 2. TestParseRange — date parsing
// ---------------------------------------------------------------------------

func TestParseRange(t *testing.T) {
	from, to := parseRange("2026-01-15", "2026-01-20")
	if from.Year() != 2026 || from.Month() != time.January || from.Day() != 15 {
		t.Fatalf("from = %v, want 2026-01-15", from)
	}
	// to should be +1 day: Jan 21
	if to.Year() != 2026 || to.Month() != time.January || to.Day() != 21 {
		t.Fatalf("to = %v, want 2026-01-21", to)
	}
}

// ---------------------------------------------------------------------------
// 3. TestParseRange_Defaults — default values for empty strings
// ---------------------------------------------------------------------------

func TestParseRange_Defaults(t *testing.T) {
	// Both empty: from = zero, to = now
	from, to := parseRange("", "")
	if !from.IsZero() {
		t.Fatal("expected zero from when empty string given")
	}
	if time.Since(to) > 2*time.Second || to.IsZero() {
		t.Fatal("expected to ≈ time.Now() when empty string given")
	}

	// Only from: from = parsed, to = now
	from, to = parseRange("2026-01-01", "")
	if from.Year() != 2026 || from.Month() != time.January || from.Day() != 1 {
		t.Fatalf("from = %v, want 2026-01-01", from)
	}
	if time.Since(to) > 2*time.Second || to.IsZero() {
		t.Fatal("expected to ≈ time.Now() when only from given")
	}

	// Invalid date: from = zero
	from, to = parseRange("not-a-date", "2026-06-30")
	if !from.IsZero() {
		t.Fatal("expected zero from for invalid date string")
	}
	if to.Year() != 2026 || to.Month() != time.July || to.Day() != 1 { // July 1 = June 30 + 1
		t.Fatalf("to = %v, want 2026-07-01", to)
	}
}

// ---------------------------------------------------------------------------
// 4. TestSalesReport — sales report with inline data
// ---------------------------------------------------------------------------

func TestSalesReport(t *testing.T) {
	db := dbtest.NewDB(t, &testSalesOrder{})
	svc := NewService(db, dbtest.NewLogger(t))

	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	pid1 := int64(1)
	pid2 := int64(2)

	orders := []testSalesOrder{
		{CreatedAt: now, PlatformID: &pid1, PayAmount: 100.50, ProfitAmount: 20.10, Status: "delivered"},
		{CreatedAt: now, PlatformID: &pid1, PayAmount: 200.00, ProfitAmount: 40.00, Status: "delivered"},
		{CreatedAt: now, PlatformID: &pid2, PayAmount: 50.00, ProfitAmount: 5.00, Status: "cancelled"},
	}
	for i, o := range orders {
		if err := db.Create(&o).Error; err != nil {
			t.Fatalf("insert order %d: %v", i, err)
		}
	}

	from, to := parseRange("2026-06-01", "2026-06-30")
	r, err := svc.Sales(from, to, nil)
	if err != nil {
		t.Fatalf("Sales failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil report")
	}

	if r.TotalOrders != 3 {
		t.Fatalf("TotalOrders = %d, want 3", r.TotalOrders)
	}
	assertFloatEqual(t, 350.50, r.TotalRevenue, 0.01)
	assertFloatEqual(t, 65.10, r.TotalProfit, 0.01)

	if len(r.ByPlatform) != 2 {
		t.Fatalf("ByPlatform len = %d, want 2", len(r.ByPlatform))
	}
	if r.ByStatus["delivered"] != 2 {
		t.Fatalf("ByStatus[delivered] = %d, want 2", r.ByStatus["delivered"])
	}
	if r.ByStatus["cancelled"] != 1 {
		t.Fatalf("ByStatus[cancelled] = %d, want 1", r.ByStatus["cancelled"])
	}
}

// ---------------------------------------------------------------------------
// 5. TestInventoryReport — inventory report with inline data
// ---------------------------------------------------------------------------

func TestInventoryReport(t *testing.T) {
	db := dbtest.NewDB(t, &testSku{}, &testInventory{})
	svc := NewService(db, dbtest.NewLogger(t))

	skus := []testSku{
		{ProductID: 1, Code: "SKU001", SpecDesc: "Red", Stock: 100, WarningStock: 10, CostPrice: 5.00},
		{ProductID: 1, Code: "SKU002", SpecDesc: "Blue", Stock: 50, WarningStock: 20, CostPrice: 8.00},
		{ProductID: 2, Code: "SKU003", SpecDesc: "Large", Stock: 3, WarningStock: 10, CostPrice: 15.00},
	}
	for i, s := range skus {
		if err := db.Create(&s).Error; err != nil {
			t.Fatalf("insert sku %d: %v", i, err)
		}
	}

	invs := []testInventory{
		{SkuID: 1, Warehouse: "WH-A", Quantity: 60},
		{SkuID: 1, Warehouse: "WH-B", Quantity: 40},
		{SkuID: 2, Warehouse: "WH-A", Quantity: 50},
		{SkuID: 3, Warehouse: "WH-A", Quantity: 3},
	}
	for i, inv := range invs {
		if err := db.Create(&inv).Error; err != nil {
			t.Fatalf("insert inventory %d: %v", i, err)
		}
	}

	r, err := svc.Inventory()
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil report")
	}

	if r.SkuTotal != 3 {
		t.Fatalf("SkuTotal = %d, want 3", r.SkuTotal)
	}

	// TotalStockValue = 100*5 + 50*8 + 3*15 = 500 + 400 + 45 = 945
	assertFloatEqual(t, 945.00, r.TotalStockValue, 0.01)

	if len(r.ByWarehouse) != 2 {
		t.Fatalf("ByWarehouse len = %d, want 2", len(r.ByWarehouse))
	}

	// LowStockTop20 should only include SKU003 (stock=3 <= warning_stock=10)
	if len(r.LowStockTop20) != 1 {
		t.Fatalf("LowStockTop20 len = %d, want 1", len(r.LowStockTop20))
	}
	if r.LowStockTop20[0].Stock != 3 {
		t.Fatalf("LowStockTop20[0].Stock = %d, want 3", r.LowStockTop20[0].Stock)
	}
	if r.LowStockTop20[0].Code != "SKU003" {
		t.Fatalf("LowStockTop20[0].Code = %s, want SKU003", r.LowStockTop20[0].Code)
	}
}

// ---------------------------------------------------------------------------
// 6. TestSettlementReport — settlement report with inline data
// ---------------------------------------------------------------------------

func TestSettlementReport(t *testing.T) {
	db := dbtest.NewDB(t, &testSettlement{}, &testSettlementItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	pid := int64(1)

	s1 := testSettlement{CreatedAt: now, PlatformID: &pid, TotalNet: 500.00}
	if err := db.Create(&s1).Error; err != nil {
		t.Fatalf("insert settlement 1: %v", err)
	}
	s2 := testSettlement{CreatedAt: now, PlatformID: &pid, TotalNet: 300.00}
	if err := db.Create(&s2).Error; err != nil {
		t.Fatalf("insert settlement 2: %v", err)
	}

	items := []testSettlementItem{
		{SettlementID: s1.ID, ReconciliationStatus: "matched"},
		{SettlementID: s1.ID, ReconciliationStatus: "matched"},
		{SettlementID: s2.ID, ReconciliationStatus: "unmatched"},
	}
	for i, item := range items {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("insert settlement_item %d: %v", i, err)
		}
	}

	from, to := parseRange("2026-06-01", "2026-06-30")
	r, err := svc.Settlement(from, to, nil)
	if err != nil {
		t.Fatalf("Settlement failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil report")
	}

	if r.TotalSettlements != 2 {
		t.Fatalf("TotalSettlements = %d, want 2", r.TotalSettlements)
	}
	assertFloatEqual(t, 800.00, r.TotalNet, 0.01)
	if r.ReconciliationDist["matched"] != 2 {
		t.Fatalf("ReconciliationDist[matched] = %d, want 2", r.ReconciliationDist["matched"])
	}
	if r.ReconciliationDist["unmatched"] != 1 {
		t.Fatalf("ReconciliationDist[unmatched] = %d, want 1", r.ReconciliationDist["unmatched"])
	}
}
