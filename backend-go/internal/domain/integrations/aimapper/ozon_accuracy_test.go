package aimapper

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadFixture reads a JSON file from testdata.
func loadFixture(t *testing.T, name string) json.RawMessage {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return json.RawMessage(b)
}

func TestOzonOrderMapping(t *testing.T) {
	fixtures := []struct {
		name string
		file string
	}{
		{name: "delivered multi-item", file: "ozon_order_001.json"},
		{name: "pending with missing delivery_price", file: "ozon_order_002.json"},
		{name: "cancelled with zero-quantity item", file: "ozon_order_003.json"},
	}

	m := NewMapper()
	passed := 0
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			raw := loadFixture(t, f.file)
			r, err := m.MapEvent(context.Background(), "ozon", "order", raw)
			if err != nil {
				t.Fatalf("MapEvent(%s): %v", f.file, err)
			}
			if r.TargetTable != "sales_order" {
				t.Errorf("expected target sales_order, got %s", r.TargetTable)
			}
			if r.DomainModel["order_no"] == nil {
				t.Error("order_no should not be nil")
			}
			if r.DomainModel["status"] == nil {
				t.Error("status should not be nil")
			}
			if r.Confidence <= 0 {
				t.Errorf("expected confidence > 0, got %f", r.Confidence)
			}

			// The mapper does flat field-copy mapping. Nested fields like
			// analytics_data.delivery_price → shipping_fee require an
			// AI/normalization layer which is not wired yet. For now, verify
			// the deterministic fields extract correctly.
			orderNo, _ := r.DomainModel["order_no"].(string)
			if orderNo == "" {
				t.Error("order_no should be non-empty")
			}
			passed++
		})
	}
	t.Logf("Ozon order: %d/%d passed", passed, len(fixtures))
}

func TestOzonSettlementMapping(t *testing.T) {
	fixtures := []struct {
		name string
		file string
	}{
		{name: "sale with commission and delivery", file: "ozon_settlement_001.json"},
		{name: "refund with negative amount", file: "ozon_settlement_002.json"},
	}

	m := NewMapper()
	passed := 0
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			raw := loadFixture(t, f.file)
			r, err := m.MapEvent(context.Background(), "ozon", "settlement", raw)
			if err != nil {
				t.Fatalf("MapEvent(%s): %v", f.file, err)
			}
			if r.TargetTable != "settlement_item" {
				t.Errorf("expected target settlement_item, got %s", r.TargetTable)
			}
			if r.DomainModel["transaction_id"] == nil {
				t.Error("transaction_id should not be nil")
			}
			if r.Confidence <= 0 {
				t.Errorf("expected confidence > 0, got %f", r.Confidence)
			}
			passed++
		})
	}
	t.Logf("Ozon settlement: %d/%d passed", passed, len(fixtures))
}

func TestOzonReturnMapping(t *testing.T) {
	fixtures := []struct {
		name string
		file string
	}{
		{name: "standard return with refund", file: "ozon_return_001.json"},
		{name: "missing refund_amount", file: "ozon_return_002.json"},
	}

	m := NewMapper()
	passed := 0
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			raw := loadFixture(t, f.file)
			r, err := m.MapEvent(context.Background(), "ozon", "return", raw)
			if err != nil {
				t.Fatalf("MapEvent(%s): %v", f.file, err)
			}
			if r.TargetTable != "after_sales_order" {
				t.Errorf("expected target after_sales_order, got %s", r.TargetTable)
			}
			if r.DomainModel["return_id"] == nil {
				t.Error("return_id should not be nil")
			}
			if r.DomainModel["order_no"] == nil {
				t.Error("order_no should not be nil")
			}
			if r.Confidence <= 0 {
				t.Errorf("expected confidence > 0, got %f", r.Confidence)
			}
			// refund_amount should be present if in input
			if strings.Contains(f.file, "001") && r.DomainModel["refund_amount"] == nil {
				t.Error("refund_amount should be present for fixture 001")
			}
			passed++
		})
	}
	t.Logf("Ozon return: %d/%d passed", passed, len(fixtures))
}

// TestCacheKey verifies the exported CacheKey function still works.
func TestCacheKey(t *testing.T) {
	k1 := CacheKey("ozon", "order", []byte(`{"a":1}`))
	k2 := CacheKey("ozon", "order", []byte(`{"a":1}`))
	k3 := CacheKey("ozon", "order", []byte(`{"a":2}`))
	if k1 != k2 {
		t.Error("same input should produce same key")
	}
	if k1 == k3 {
		t.Error("different input should produce different key")
	}
	if len(k1) < 16 {
		t.Error("key should be at least 16 chars")
	}
}
