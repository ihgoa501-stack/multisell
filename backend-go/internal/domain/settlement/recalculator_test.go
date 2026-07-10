package settlement

import (
	"context"
	"math"
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
	if err := db.AutoMigrate(tables...); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
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
	}
	for _, it := range items {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("create settlement item: %v", err)
		}
	}

	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateProfit(context.Background(), "ORD-PROFIT-1"); err != nil {
		t.Fatalf("RecalculateProfit: %v", err)
	}

	// Verify sales_order updated
	var o order.Order
	if err := db.Where("order_no = ?", "ORD-PROFIT-1").First(&o).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}

	// Expected: revenue = 100 - 15 = 85, costs = 30 + 15 + 3 + 8 = 56, profit = 29
	// margin = (29 / 85) * 100 ≈ 34.12
	expectedProfit := 29.0
	expectedMargin := math.Round((29.0/85.0)*100*100) / 100
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
	}
	for _, it := range items {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("create settlement item: %v", err)
		}
	}

	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateProfit(context.Background(), "ORD-REFUND-1"); err != nil {
		t.Fatalf("RecalculateProfit: %v", err)
	}

	var o order.Order
	if err := db.Where("order_no = ?", "ORD-REFUND-1").First(&o).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}

	// revenue = 50 - 7.5 = 42.5, costs = 15 + 7.5 + 50 = 72.5, profit = 42.5 - 72.5 = -30
	expectedProfit := -30.0
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
	}
	for _, it := range items {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("create item: %v", err)
		}
	}

	recalc := NewRecalculator(db, testLogger())
	if err := recalc.RecalculateProfit(context.Background(), "ORD-MULTI"); err != nil {
		t.Fatalf("RecalculateProfit: %v", err)
	}

	var o order.Order
	if err := db.Where("order_no = ?", "ORD-MULTI").First(&o).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}

	// revenue = (120 + 80) - (18 + 12) = 170
	// costs = 50 + (18 + 12) + 5 + 2 + 10 = 97
	// profit = 170 - 97 = 73
	expectedProfit := 73.0
	if math.Abs(o.ProfitAmount-expectedProfit) > 0.01 {
		t.Errorf("profit_amount = %.2f, want %.2f", o.ProfitAmount, expectedProfit)
	}
}
