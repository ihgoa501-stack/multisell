package supplyevent

import (
	"encoding/json"
	"testing"
	"time"
)

func TestToPayload_OrderReceived(t *testing.T) {
	evt := OrderReceived{
		OrderNo:    "PO-2026-001",
		SupplierID: 42,
		Items: []ReceivedItem{
			{SkuID: 1001, Qty: 50},
			{SkuID: 1002, Qty: 25},
		},
		ReceivedAt: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC),
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload == nil {
		t.Fatal("expected non-nil payload")
	}
	if payload["order_no"] != "PO-2026-001" {
		t.Errorf("expected order_no 'PO-2026-001', got %v", payload["order_no"])
	}
	if payload["supplier_id"] != float64(42) {
		t.Errorf("expected supplier_id 42, got %v", payload["supplier_id"])
	}
}

func TestToPayload_StockAdjusted(t *testing.T) {
	evt := StockAdjusted{
		SkuID:     2001,
		Warehouse: "Shenzhen-Hub",
		Delta:     -10,
		Reason:    "write-off",
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload["sku_id"] != float64(2001) {
		t.Errorf("expected sku_id 2001, got %v", payload["sku_id"])
	}
	if payload["delta"] != float64(-10) {
		t.Errorf("expected delta -10, got %v", payload["delta"])
	}
	if payload["reason"] != "write-off" {
		t.Errorf("expected reason 'write-off', got %v", payload["reason"])
	}
}

func TestToPayload_AfterSaleProcessed(t *testing.T) {
	evt := AfterSaleProcessed{
		AftersaleID: 1,
		OrderID:     100,
		SkuID:       3001,
		Quantity:    2,
		Type:        "return",
		ProcessedAt: time.Now(),
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload["type"] != "return" {
		t.Errorf("expected type 'return', got %v", payload["type"])
	}
	if payload["quantity"] != float64(2) {
		t.Errorf("expected quantity 2, got %v", payload["quantity"])
	}
}

func TestToPayload_AftersaleReturned(t *testing.T) {
	evt := AftersaleReturned{
		AftersaleID: 5,
		OrderID:     200,
		SkuID:       4001,
		Quantity:    1,
		Reason:      "defective",
		ReturnedAt:  time.Now(),
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload["reason"] != "defective" {
		t.Errorf("expected reason 'defective', got %v", payload["reason"])
	}
}

func TestToPayload_StockCritical(t *testing.T) {
	evt := StockCritical{
		SkuID:        5001,
		CurrentStock: 5,
		SafetyStock:  20,
		SellableDays: 1.5,
		ReportedAt:   time.Now(),
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload["current_stock"] != float64(5) {
		t.Errorf("expected current_stock 5, got %v", payload["current_stock"])
	}
	if payload["safety_stock"] != float64(20) {
		t.Errorf("expected safety_stock 20, got %v", payload["safety_stock"])
	}
}

func TestToPayload_QuoteRequested(t *testing.T) {
	evt := QuoteRequested{
		ProductID:   10,
		SourceURL:   "https://1688.com/item/123",
		Destination: "Moscow",
		WeightKg:    0.5,
		Timestamp:   time.Now(),
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload["product_id"] != float64(10) {
		t.Errorf("expected product_id 10, got %v", payload["product_id"])
	}
}

func TestToPayload_QuoteReady(t *testing.T) {
	evt := QuoteReady{
		ProductID:        10,
		ChannelName:      "EMS",
		TotalShippingFee: 25.50,
		Currency:         "USD",
		Timestamp:        time.Now(),
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload["channel_name"] != "EMS" {
		t.Errorf("expected channel_name 'EMS', got %v", payload["channel_name"])
	}
	if payload["total_shipping_fee"] != float64(25.50) {
		t.Errorf("expected shipping fee 25.50, got %v", payload["total_shipping_fee"])
	}
}

func TestToPayload_OrderRequested(t *testing.T) {
	evt := OrderRequested{
		ProductID:   10,
		ChannelName: "China Post",
		TotalAmount: 150.00,
		Timestamp:   time.Now(),
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload["total_amount"] != float64(150.00) {
		t.Errorf("expected total_amount 150.00, got %v", payload["total_amount"])
	}
}

func TestToPayload_FlywheelEvent(t *testing.T) {
	evt := FlywheelEvent{
		FlowID:       "flow-001",
		ChannelName:  "DHL Express",
		ProviderName: "DHL",
		CategoryName: "electronics",
		ActualCost:   35.00,
		Currency:     "USD",
		DeliveryDays: 5,
		IsLost:       false,
		SourceType:   "order",
		SourceID:     "SO-12345",
		Timestamp:    time.Now(),
	}

	payload, err := ToPayload(&evt)
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if payload["flow_id"] != "flow-001" {
		t.Errorf("expected flow_id 'flow-001', got %v", payload["flow_id"])
	}
	if payload["is_lost"] != false {
		t.Errorf("expected is_lost false, got %v", payload["is_lost"])
	}
}

func TestToPayload_NilInput(t *testing.T) {
	payload, err := ToPayload(nil)
	if err != nil {
		t.Fatalf("ToPayload(nil) should not error: %v", err)
	}
	// nil input serializes as JSON null, which unmarshal to nil map
	_ = payload
}

func TestEventStructs_JSONRoundTrip(t *testing.T) {
	original := OrderReceived{
		OrderNo:    "PO-001",
		SupplierID: 100,
		Items:      []ReceivedItem{{SkuID: 1, Qty: 10}},
		ReceivedAt: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded OrderReceived
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.OrderNo != original.OrderNo {
		t.Errorf("round-trip OrderNo: got %s, want %s", decoded.OrderNo, original.OrderNo)
	}
	if len(decoded.Items) != 1 || decoded.Items[0].SkuID != 1 {
		t.Errorf("round-trip items failed")
	}
}

func TestToPayload_ThenJSON(t *testing.T) {
	// Round-trip: struct → ToPayload → JSON → map
	evt := StockAdjusted{SkuID: 999, Warehouse: "WH-1", Delta: 5, Reason: "return"}
	payload, _ := ToPayload(&evt)

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload failed: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal payload failed: %v", err)
	}

	if m["sku_id"] != float64(999) {
		t.Errorf("expected sku_id 999, got %v", m["sku_id"])
	}
	if m["warehouse"] != "WH-1" {
		t.Errorf("expected warehouse 'WH-1', got %v", m["warehouse"])
	}
	if m["reason"] != "return" {
		t.Errorf("expected reason 'return', got %v", m["reason"])
	}
}
