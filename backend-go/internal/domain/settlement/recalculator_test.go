package settlement

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDBWithTables(t *testing.T, tables ...interface{}) *gorm.DB {
	t.Helper()
	testDBCounter++
	dsn := "file:recalc_test_" + itoa(testDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	tables = append([]interface{}{&Settlement{}}, tables...)
	if err := db.AutoMigrate(tables...); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	now := time.Now()
	for i := int64(1); i <= 10; i++ {
		if err := db.Create(&Settlement{ID: i, PlatformID: 1, SettlementNo: "TEST-" + itoa(i), Status: "reconciled", SourceType: "platform_import", ImportedAt: &now}).Error; err != nil {
			t.Fatalf("create trusted settlement: %v", err)
		}
	}
	return db
}

func reconcileAllTestItems(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	if err := db.Model(&SettlementItem{}).Where("1 = 1").Updates(map[string]interface{}{
		"reconciliation_status": "matched", "reconciled_at": &now, "reconciled_by": "test-auditor",
	}).Error; err != nil {
		t.Fatalf("reconcile test items: %v", err)
	}
}

func TestComputeProfit_Basic(t *testing.T) {
	t.Parallel()
	db := newTestDBWithTables(t, &SettlementItem{}, &order.Order{}, &profit.OrderProfitRecord{})

	// Create a sales order with product cost
	if err := db.Create(&order.Order{
		OrderNo:     "ORD-PROFIT-1",
		TotalAmount: 100,
		PayAmount:   100,
		ProductCost: 30,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Create settlement items: one sale with fee, one platform fee, one shipping fee
	now := time.Now()
	items := []SettlementItem{
		{SettlementID: 1, TransactionType: "order_sale", OrderNo: "ORD-PROFIT-1", Amount: 100, Fee: 15, Quantity: 1, OccurredAt: &now},
		{SettlementID: 1, TransactionType: "payment_fee", OrderNo: "ORD-PROFIT-1", Amount: 3, Quantity: 1, OccurredAt: &now},
		{SettlementID: 1, TransactionType: "shipping_fee", OrderNo: "ORD-PROFIT-1", Amount: 8, Quantity: 1, OccurredAt: &now},
		{SettlementID: 1, TransactionType: "tariff_fee", OrderNo: "ORD-PROFIT-1", Amount: 0, Quantity: 1, OccurredAt: &now},
	}
	for _, it := range items {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("create settlement item: %v", err)
		}
	}
	reconcileAllTestItems(t, db)

	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateProfit(context.Background(), "ORD-PROFIT-1"); err != nil {
		t.Fatalf("RecalculateProfit: %v", err)
	}

	// Verify sales_order updated
	var o order.Order
	if err := db.Where("order_no = ?", "ORD-PROFIT-1").First(&o).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}

	// Revenue remains gross. The sale fee is a cost exactly once.
	// Expected: revenue = 100, costs = 30 + 15 + 3 + 8 = 56, profit = 44.
	expectedProfit := 44.0
	expectedMargin := 44.0
	if math.Abs(o.ProfitAmount-expectedProfit) > 0.01 {
		t.Errorf("profit_amount = %.2f, want %.2f", o.ProfitAmount, expectedProfit)
	}
	if math.Abs(o.ProfitMargin-expectedMargin) > 0.01 {
		t.Errorf("profit_margin = %.2f, want %.2f", o.ProfitMargin, expectedMargin)
	}

	// Verify order_profit_record created
	var rec profit.OrderProfitRecord
	if err := db.Where("order_id = ?", o.ID).First(&rec).Error; err != nil {
		t.Fatalf("find order_profit_record: %v", err)
	}
	if math.Abs(rec.Profit-expectedProfit) > 0.01 {
		t.Errorf("record profit = %.2f, want %.2f", rec.Profit, expectedProfit)
	}
	if rec.ProfitStatus != "final" || rec.MissingCosts != "" {
		t.Errorf("truth status = %q missing=%q, want final with no missing costs", rec.ProfitStatus, rec.MissingCosts)
	}
}

func TestComputeProfit_Refund(t *testing.T) {
	t.Parallel()
	db := newTestDBWithTables(t, &SettlementItem{}, &order.Order{}, &profit.OrderProfitRecord{})

	if err := db.Create(&order.Order{
		OrderNo:     "ORD-REFUND-1",
		TotalAmount: 50,
		PayAmount:   50,
		ProductCost: 15,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	now := time.Now()
	items := []SettlementItem{
		{SettlementID: 2, TransactionType: "order_sale", OrderNo: "ORD-REFUND-1", Amount: 50, Fee: 7.5, Quantity: 1, OccurredAt: &now},
		{SettlementID: 2, TransactionType: "refund", OrderNo: "ORD-REFUND-1", Amount: 50, Quantity: 1, OccurredAt: &now},
		{SettlementID: 2, TransactionType: "payment_fee", OrderNo: "ORD-REFUND-1", Amount: 0, Quantity: 1, OccurredAt: &now},
		{SettlementID: 2, TransactionType: "shipping_fee", OrderNo: "ORD-REFUND-1", Amount: 0, Quantity: 1, OccurredAt: &now},
		{SettlementID: 2, TransactionType: "tariff_fee", OrderNo: "ORD-REFUND-1", Amount: 0, Quantity: 1, OccurredAt: &now},
	}
	for _, it := range items {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("create settlement item: %v", err)
		}
	}
	reconcileAllTestItems(t, db)

	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateProfit(context.Background(), "ORD-REFUND-1"); err != nil {
		t.Fatalf("RecalculateProfit: %v", err)
	}

	var o order.Order
	if err := db.Where("order_no = ?", "ORD-REFUND-1").First(&o).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}

	// revenue = 50, costs = 15 + 7.5 + 50 = 72.5, profit = -22.5
	expectedProfit := -22.5
	if math.Abs(o.ProfitAmount-expectedProfit) > 0.01 {
		t.Errorf("profit_amount = %.2f, want %.2f", o.ProfitAmount, expectedProfit)
	}
}

func TestComputeProfit_NoSettlementData(t *testing.T) {
	t.Parallel()
	db := newTestDBWithTables(t, &SettlementItem{}, &order.Order{}, &profit.OrderProfitRecord{})

	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateProfit(context.Background(), "ORD-NODATA"); err != nil {
		t.Fatalf("RecalculateProfit with no data should not error: %v", err)
	}
}

func TestComputeProfit_NoOrder(t *testing.T) {
	t.Parallel()
	db := newTestDBWithTables(t, &SettlementItem{}, &order.Order{}, &profit.OrderProfitRecord{})

	now := time.Now()
	if err := db.Create(&SettlementItem{
		SettlementID: 1, TransactionType: "order_sale", OrderNo: "ORD-NO-ORDER",
		Amount: 100, Fee: 15, Quantity: 1, OccurredAt: &now,
	}).Error; err != nil {
		t.Fatalf("create settlement item: %v", err)
	}

	// No sales_order record — should not error, just skip
	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateProfit(context.Background(), "ORD-NO-ORDER"); err != nil {
		t.Fatalf("RecalculateProfit with no order should not error: %v", err)
	}
}

func TestRecalculateAllProfit(t *testing.T) {
	t.Parallel()
	db := newTestDBWithTables(t, &SettlementItem{}, &order.Order{}, &profit.OrderProfitRecord{})

	if err := db.Create(&order.Order{
		OrderNo: "ORD-ALL-1", TotalAmount: 100, PayAmount: 100, ProductCost: 20,
	}).Error; err != nil {
		t.Fatalf("create order 1: %v", err)
	}
	if err := db.Create(&order.Order{
		OrderNo: "ORD-ALL-2", TotalAmount: 200, PayAmount: 200, ProductCost: 40,
	}).Error; err != nil {
		t.Fatalf("create order 2: %v", err)
	}

	now := time.Now()
	for _, it := range []SettlementItem{
		{SettlementID: 1, TransactionType: "order_sale", OrderNo: "ORD-ALL-1", Amount: 100, Fee: 15, Quantity: 1, OccurredAt: &now},
		{SettlementID: 2, TransactionType: "order_sale", OrderNo: "ORD-ALL-2", Amount: 200, Fee: 30, Quantity: 1, OccurredAt: &now},
	} {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("create item: %v", err)
		}
	}

	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateAllProfit(context.Background()); err != nil {
		t.Fatalf("RecalculateAllProfit: %v", err)
	}

	var count int64
	db.Model(&profit.OrderProfitRecord{}).Count(&count)
	if count != 2 {
		t.Errorf("order_profit_record count = %d, want 2", count)
	}
}

func TestRecalculateProfit_MultiItemSale(t *testing.T) {
	t.Parallel()
	db := newTestDBWithTables(t, &SettlementItem{}, &order.Order{}, &profit.OrderProfitRecord{})

	if err := db.Create(&order.Order{
		OrderNo: "ORD-MULTI", TotalAmount: 200, PayAmount: 200, ProductCost: 50,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Two settlement items for the same order (e.g., two items in one order)
	now := time.Now()
	items := []SettlementItem{
		{SettlementID: 3, TransactionType: "order_sale", OrderNo: "ORD-MULTI", Amount: 120, Fee: 18, Quantity: 1, OccurredAt: &now},
		{SettlementID: 3, TransactionType: "order_sale", OrderNo: "ORD-MULTI", Amount: 80, Fee: 12, Quantity: 1, OccurredAt: &now},
		{SettlementID: 3, TransactionType: "platform_fee", OrderNo: "ORD-MULTI", Amount: 5, Quantity: 1, OccurredAt: &now},
		{SettlementID: 3, TransactionType: "payment_fee", OrderNo: "ORD-MULTI", Amount: 2, Quantity: 1, OccurredAt: &now},
		{SettlementID: 3, TransactionType: "shipping_fee", OrderNo: "ORD-MULTI", Amount: 10, Quantity: 1, OccurredAt: &now},
		{SettlementID: 3, TransactionType: "tariff_fee", OrderNo: "ORD-MULTI", Amount: 0, Quantity: 1, OccurredAt: &now},
	}
	for _, it := range items {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("create item: %v", err)
		}
	}
	reconcileAllTestItems(t, db)

	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateProfit(context.Background(), "ORD-MULTI"); err != nil {
		t.Fatalf("RecalculateProfit: %v", err)
	}

	var o order.Order
	if err := db.Where("order_no = ?", "ORD-MULTI").First(&o).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}

	// revenue = 200; costs = 50 + (18 + 12) + 5 + 2 + 10 = 97
	// profit = 103
	expectedProfit := 103.0
	if math.Abs(o.ProfitAmount-expectedProfit) > 0.01 {
		t.Errorf("profit_amount = %.2f, want %.2f", o.ProfitAmount, expectedProfit)
	}
}

func TestRecalculateProfit_MissingCriticalCostStaysProvisional(t *testing.T) {
	t.Parallel()
	db := newTestDBWithTables(t, &SettlementItem{}, &order.Order{}, &profit.OrderProfitRecord{})
	o := order.Order{OrderNo: "ORD-PROVISIONAL", PayAmount: 100, ProductCost: 30, ProfitAmount: 999, ProfitMargin: 99}
	if err := db.Create(&o).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Create(&SettlementItem{SettlementID: 4, TransactionType: "order_sale", OrderNo: o.OrderNo, Amount: 100, Fee: 15, OccurredAt: &now}).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewRecalculator(db, testLogger()).RecalculateProfit(context.Background(), o.OrderNo); err != nil {
		t.Fatal(err)
	}
	var gotOrder order.Order
	if err := db.First(&gotOrder, o.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotOrder.ProfitAmount != 999 || gotOrder.ProfitMargin != 99 {
		t.Fatalf("provisional calculation overwrote final-facing order fields: %+v", gotOrder)
	}
	var rec profit.OrderProfitRecord
	if err := db.Where("order_id = ?", o.ID).First(&rec).Error; err != nil {
		t.Fatal(err)
	}
	if rec.ProfitStatus != "provisional" || rec.MissingCosts == "" {
		t.Fatalf("status=%q missing=%q, want provisional with missing costs", rec.ProfitStatus, rec.MissingCosts)
	}
}

func TestRecalculateProfit_ManualOrIncompleteSettlementNeverBecomesFinal(t *testing.T) {
	t.Parallel()
	db := newTestDBWithTables(t, &SettlementItem{}, &order.Order{}, &profit.OrderProfitRecord{})
	o := order.Order{OrderNo: "ORD-MANUAL", PayAmount: 100, ProductCost: 30, ProfitAmount: 777, ProfitMargin: 77}
	if err := db.Create(&o).Error; err != nil {
		t.Fatal(err)
	}
	// Even fully reconciled zero-value costs are legitimate only when their parent
	// settlement came from a trusted import and has completed reconciliation.
	if err := db.Model(&Settlement{}).Where("id = ?", 5).Updates(map[string]interface{}{
		"status": "pending", "source_type": "manual", "imported_at": nil,
	}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, it := range []SettlementItem{
		{SettlementID: 5, TransactionType: "order_sale", OrderNo: o.OrderNo, Amount: 100, Fee: 15, OccurredAt: &now},
		{SettlementID: 5, TransactionType: "payment_fee", OrderNo: o.OrderNo, Amount: 0, OccurredAt: &now},
		{SettlementID: 5, TransactionType: "shipping_fee", OrderNo: o.OrderNo, Amount: 0, OccurredAt: &now},
		{SettlementID: 5, TransactionType: "tariff_fee", OrderNo: o.OrderNo, Amount: 0, OccurredAt: &now},
	} {
		it.ReconciliationStatus = "matched"
		it.ReconciledAt = &now
		it.ReconciledBy = "auditor"
		if err := db.Create(&it).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := NewRecalculator(db, testLogger()).RecalculateProfit(context.Background(), o.OrderNo); err != nil {
		t.Fatal(err)
	}
	var got order.Order
	if err := db.First(&got, o.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.ProfitAmount != 777 || got.ProfitMargin != 77 {
		t.Fatalf("untrusted settlement overwrote final-facing order profit: %+v", got)
	}
	var rec profit.OrderProfitRecord
	if err := db.Where("order_id = ?", o.ID).First(&rec).Error; err != nil {
		t.Fatal(err)
	}
	if rec.ProfitStatus != "provisional" || !strings.Contains(rec.MissingCosts, "settlement_not_completed") || !strings.Contains(rec.MissingCosts, "settlement_source_unverified") {
		t.Fatalf("status=%q missing=%q", rec.ProfitStatus, rec.MissingCosts)
	}
}
