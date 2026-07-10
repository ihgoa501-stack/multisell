package aimapper

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMapEvent_Order(t *testing.T) {
	m := NewMapper()
	raw := jsonRaw(`{
		"posting_number": "ORD-001",
		"status": "delivered",
		"total_amount": "109.97",
		"shipping_fee": "15.50",
		"in_process_at": "2026-07-01T10:30:00Z",
		"items": [{"sku_code": "ABC", "quantity": 2, "unit_price": "29.99"}],
		"platform_code": "ozon"
	}`)

	r, err := m.MapEvent(context.Background(), "ozon", "order", raw)
	if err != nil {
		t.Fatalf("MapEvent order: %v", err)
	}

	if r.TargetTable != "sales_order" {
		t.Errorf("expected target sales_order, got %s", r.TargetTable)
	}
	if r.DomainModel["order_no"] != "ORD-001" {
		t.Errorf("expected order_no ORD-001, got %v", r.DomainModel["order_no"])
	}
	if r.DomainModel["status"] != "delivered" {
		t.Errorf("expected status delivered, got %v", r.DomainModel["status"])
	}
	if r.DomainModel["total_amount"] != "109.97" {
		t.Errorf("expected total_amount 109.97, got %v", r.DomainModel["total_amount"])
	}
	if r.DomainModel["shipping_fee"] != "15.50" {
		t.Errorf("expected shipping_fee 15.50, got %v", r.DomainModel["shipping_fee"])
	}
	if r.Confidence <= 0 {
		t.Errorf("expected confidence > 0, got %f", r.Confidence)
	}
}

func TestMapEvent_Settlement(t *testing.T) {
	m := NewMapper()
	raw := jsonRaw(`{
		"operation_id": "TXN-001",
		"operation_type": "sale",
		"posting_number": "ORD-001",
		"amount": "49.99",
		"currency_code": "RUB",
		"operation_date": "2026-07-01T10:30:00Z",
		"description": "sale of items",
		"platform_code": "ozon"
	}`)

	r, err := m.MapEvent(context.Background(), "ozon", "settlement", raw)
	if err != nil {
		t.Fatalf("MapEvent settlement: %v", err)
	}

	if r.TargetTable != "settlement_item" {
		t.Errorf("expected target settlement_item, got %s", r.TargetTable)
	}
	if r.DomainModel["transaction_id"] != "TXN-001" {
		t.Errorf("expected transaction_id TXN-001, got %v", r.DomainModel["transaction_id"])
	}
	if r.DomainModel["amount"] != "49.99" {
		t.Errorf("expected amount 49.99, got %v", r.DomainModel["amount"])
	}
}

func TestMapEvent_Return(t *testing.T) {
	m := NewMapper()
	raw := jsonRaw(`{
		"return_id": "RET-001",
		"posting_number": "ORD-001",
		"sku": "SKU-RET-001",
		"quantity": 1,
		"reason": "defective",
		"status": "in_progress",
		"created_at": "2026-07-02T10:00:00Z",
		"refund_amount": "29.99",
		"platform_code": "ozon"
	}`)

	r, err := m.MapEvent(context.Background(), "ozon", "return", raw)
	if err != nil {
		t.Fatalf("MapEvent return: %v", err)
	}

	if r.TargetTable != "after_sales_order" {
		t.Errorf("expected target after_sales_order, got %s", r.TargetTable)
	}
	if r.DomainModel["return_id"] != "RET-001" {
		t.Errorf("expected return_id RET-001, got %v", r.DomainModel["return_id"])
	}
	if r.DomainModel["refund_amount"] != "29.99" {
		t.Errorf("expected refund_amount 29.99, got %v", r.DomainModel["refund_amount"])
	}
}

func TestMapEvent_UnknownType(t *testing.T) {
	m := NewMapper()
	raw := jsonRaw(`{"event": "unknown"}`)

	r, err := m.MapEvent(context.Background(), "ozon", "unknown_type", raw)
	if err != nil {
		t.Fatalf("MapEvent unknown: %v", err)
	}
	if r.TargetTable != "unknown" {
		t.Errorf("expected target unknown for unknown type, got %s", r.TargetTable)
	}
	if r.Confidence != 0.1 {
		t.Errorf("expected confidence 0.1 for unknown, got %f", r.Confidence)
	}
}

func TestMapEvent_InvalidJSON(t *testing.T) {
	m := NewMapper()
	_, err := m.MapEvent(context.Background(), "ozon", "order", jsonRaw(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("expected unmarshal error, got: %v", err)
	}
}

// jsonRaw is a helper that wraps a JSON string into json.RawMessage.
func jsonRaw(s string) json.RawMessage {
	return json.RawMessage(s)
}
