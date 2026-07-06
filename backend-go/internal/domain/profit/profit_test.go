package profit

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/shipping"
)

func TestService_Calculate_CreateErrorIsReturned(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProfitSummary{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t), nil, 7.2)

	platformID := int64(1)
	prod := candidate.CandidateProduct{
		Title:            "Test Product for Profit Calc",
		PurchasePrice:    50,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		HSCode:           "1234.56",
		TargetSalePrice:  30,
		TargetPlatformID: &platformID,
		DestinationCountry: "US",
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	// Drop target table so Create fails
	db.Exec("DROP TABLE profit_summary")

	_, err := svc.Calculate(prod.ID, "tester")
	if err == nil {
		t.Fatal("expected error from Create (table dropped)")
	}
}

func TestService_ListSummaries(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProfitSummary{})
	svc := NewService(db, dbtest.NewLogger(t), nil, 7.2)

	db.Create(&ProfitSummary{ProductID: 1, Status: "profitable", EstimatedProfit: 25.0, ProfitMargin: 20.0})
	db.Create(&ProfitSummary{ProductID: 2, Status: "unprofitable", EstimatedProfit: -5.0, ProfitMargin: -5.0})

	items, total, err := svc.ListSummaries(1, 10, "", "", "")
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
}

func TestService_ListSummaries_Filtered(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProfitSummary{})
	svc := NewService(db, dbtest.NewLogger(t), nil, 7.2)

	db.Create(&ProfitSummary{ProductID: 1, Status: "profitable"})
	db.Create(&ProfitSummary{ProductID: 2, Status: "unprofitable"})

	items, total, err := svc.ListSummaries(1, 10, "profitable", "", "")
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if items[0].Status != "profitable" {
		t.Fatalf("status = %s", items[0].Status)
	}
}

func TestClassifyProfit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		margin float64
		want   string
	}{
		{20, "profitable"},
		{15, "profitable"},
		{10, "marginal"},
		{0, "marginal"},
		{-5, "unprofitable"},
	}
	for _, tt := range tests {
		got := classifyProfit(tt.margin)
		if got != tt.want {
			t.Fatalf("classifyProfit(%f) = %s, want %s", tt.margin, got, tt.want)
		}
	}
}

func TestService_CalculateOrderProfit_Full(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderProfitRecord{}, &order.Order{}, &order.OrderItem{}, &shipping.SalesOrderShippingSnapshot{})
	log := dbtest.NewLogger(t)
	svc := NewService(db, log, nil, 7.2)

	// Create an order
	o := order.Order{
		OrderNo:     "ORD-TEST-001",
		TotalAmount: 200,
		PayAmount:   180,
		ProductCost: 60,
		ShippingFee: 15,
		PlatformFee: 18,
		PaymentFee:  3.6,
		OtherFee:    5,
		Status:      "delivered",
	}
	if err := db.Create(&o).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Create order items
	items := []order.OrderItem{
		{OrderID: o.ID, ProductID: 1, SkuID: 1, ProductName: "Item A", UnitPrice: 100, Quantity: 1, Subtotal: 100},
		{OrderID: o.ID, ProductID: 1, SkuID: 2, ProductName: "Item B", UnitPrice: 80, Quantity: 1, Subtotal: 80},
	}
	for _, it := range items {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("create order item: %v", err)
		}
	}

	// Create a shipping snapshot
	snap := shipping.SalesOrderShippingSnapshot{
		OrderID:          o.ID,
		SkuID:            1,
		Quantity:         1,
		TotalShippingFee: 12.5,
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		ProviderID:       1,
		ChannelID:        1,
		ActualWeightKg:   0.5,
		VolumetricWeightKg: 0.16,
		ChargeableWeightKg: 0.5,
		BaseShippingFee:    12.5,
	}
	if err := db.Create(&snap).Error; err != nil {
		t.Fatalf("create shipping snapshot: %v", err)
	}

	result, err := svc.CalculateOrderProfit(t.Context(), uint(o.ID))
	if err != nil {
		t.Fatalf("CalculateOrderProfit: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	// Revenue = PayAmount (180)
	if result.Revenue != 180 {
		t.Fatalf("Revenue = %f, want 180", result.Revenue)
	}
	// Cost = ProductCost (60)
	if result.Cost != 60 {
		t.Fatalf("Cost = %f, want 60", result.Cost)
	}
	// ShippingCost should come from snapshot (12.5), not order.ShippingFee (15)
	if result.ShippingCost != 12.5 {
		t.Fatalf("ShippingCost = %f, want 12.5", result.ShippingCost)
	}
	if result.PlatformFee != 18 {
		t.Fatalf("PlatformFee = %f, want 18", result.PlatformFee)
	}
	if result.PaymentFee != 3.6 {
		t.Fatalf("PaymentFee = %f, want 3.6", result.PaymentFee)
	}
	if result.TariffCost != 5 {
		t.Fatalf("TariffCost = %f, want 5", result.TariffCost)
	}

	// TotalCost = 60 + 12.5 + 18 + 3.6 + 5 = 99.1
	if result.TotalCost != 99.1 {
		t.Fatalf("TotalCost = %f, want 99.1", result.TotalCost)
	}
	// Profit = 180 - 99.1 = 80.9
	if result.Profit != 80.9 {
		t.Fatalf("Profit = %f, want 80.9", result.Profit)
	}
	// Margin = (80.9 / 180) * 100 = 44.94
	if result.Margin < 44.9 || result.Margin > 45.0 {
		t.Fatalf("Margin = %f, want ~44.94", result.Margin)
	}

	// Verify the record was persisted
	var record OrderProfitRecord
	if err := db.Where("order_id = ?", o.ID).First(&record).Error; err != nil {
		t.Fatalf("record not persisted: %v", err)
	}
	if record.OrderID != o.ID {
		t.Fatalf("record.OrderID = %d", record.OrderID)
	}
}

func TestService_CalculateOrderProfit_PartialData(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderProfitRecord{}, &order.Order{}, &order.OrderItem{})
	log := dbtest.NewLogger(t)
	svc := NewService(db, log, nil, 7.2)

	// Order with zero costs — should still compute gracefully
	o := order.Order{
		OrderNo:     "ORD-TEST-002",
		TotalAmount: 100,
		PayAmount:   0,
		Status:      "pending",
	}
	if err := db.Create(&o).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	result, err := svc.CalculateOrderProfit(t.Context(), uint(o.ID))
	if err != nil {
		t.Fatalf("CalculateOrderProfit: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	// Revenue falls back to TotalAmount when PayAmount is 0
	if result.Revenue != 100 {
		t.Fatalf("Revenue = %f, want 100 (fallback to TotalAmount)", result.Revenue)
	}
	// Profit = Revenue (all costs are 0)
	if result.Profit != 100 {
		t.Fatalf("Profit = %f, want 100", result.Profit)
	}
	if result.Margin != 100 {
		t.Fatalf("Margin = %f, want 100", result.Margin)
	}
}

func TestService_CalculateOrderProfit_MissingOrder(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	log := dbtest.NewLogger(t)
	svc := NewService(db, log, nil, 7.2)

	_, err := svc.CalculateOrderProfit(t.Context(), 99999)
	if err == nil {
		t.Fatal("expected error for missing order")
	}
}

func TestService_CalculateOrderProfit_Upsert(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderProfitRecord{}, &order.Order{}, &order.OrderItem{})
	log := dbtest.NewLogger(t)
	svc := NewService(db, log, nil, 7.2)

	// Create an order
	o := order.Order{
		OrderNo:     "ORD-TEST-003",
		TotalAmount: 100,
		PayAmount:   100,
		ProductCost: 40,
		Status:      "delivered",
	}
	if err := db.Create(&o).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	// First calculation
	_, err := svc.CalculateOrderProfit(t.Context(), uint(o.ID))
	if err != nil {
		t.Fatalf("first calculate: %v", err)
	}

	// Count records (should be 1)
	var count int64
	db.Model(&OrderProfitRecord{}).Where("order_id = ?", o.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 record after first calc, got %d", count)
	}

	// Second calculation (upsert, should still be 1)
	_, err = svc.CalculateOrderProfit(t.Context(), uint(o.ID))
	if err != nil {
		t.Fatalf("second calculate: %v", err)
	}

	db.Model(&OrderProfitRecord{}).Where("order_id = ?", o.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 record after upsert, got %d", count)
	}
}
