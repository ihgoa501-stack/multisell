package eventbus

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Schema registry tests
// ---------------------------------------------------------------------------

func TestSchemaRegistryRegisterAndLookup(t *testing.T) {
	r := NewSchemaRegistry()
	r.Register("stock.alert", StockAlertPayload{})

	schema, ok := r.Schema("stock.alert")
	if !ok {
		t.Fatal("expected schema for 'stock.alert'")
	}
	if _, ok := schema.(StockAlertPayload); !ok {
		t.Errorf("expected StockAlertPayload, got %T", schema)
	}

	// Unknown topic.
	if _, ok := r.Schema("unknown.topic"); ok {
		t.Error("expected no schema for unknown topic")
	}

	// Pattern matching: order.* should match order.created
	r.Register("order.*", ProfitWatchPayload{})
	schema, ok = r.Schema("order.created")
	if !ok {
		t.Fatal("expected schema for 'order.created' matching pattern 'order.*'")
	}
	if _, ok := schema.(ProfitWatchPayload); !ok {
		t.Errorf("expected ProfitWatchPayload for 'order.created', got %T", schema)
	}

	// Pattern not matching: inventory.* should not match order.shipped
	if _, ok := r.Schema("order.shipped"); !ok {
		t.Fatal("expected schema for 'order.shipped' matching pattern 'order.*'")
	}
	if _, ok := r.Schema("inventory.updated"); ok {
		t.Error("expected no schema for 'inventory.updated' since only order.* is registered")
	}

	// Broad pattern * matches everything
	r.Register("*", ComplianceCheckPayload{})
	schema, ok = r.Schema("anything.at.all")
	if !ok {
		t.Fatal("expected schema for 'anything.at.all' matching pattern '*'")
	}
	if _, ok := schema.(ComplianceCheckPayload); !ok {
		t.Errorf("expected ComplianceCheckPayload for '*', got %T", schema)
	}
}

// ---------------------------------------------------------------------------
// StockAlertPayload validation
// ---------------------------------------------------------------------------

func TestStockAlertPayload_Valid(t *testing.T) {
	s := StockAlertPayload{}
	err := s.Validate(map[string]interface{}{
		"sku_id":   "SKU-001",
		"quantity": 5,
		"threshold": 10,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStockAlertPayload_MissingField(t *testing.T) {
	s := StockAlertPayload{}
	err := s.Validate(map[string]interface{}{
		"sku_id": "SKU-001",
		// missing quantity and threshold
	})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestStockAlertPayload_WrongType(t *testing.T) {
	s := StockAlertPayload{}
	err := s.Validate(map[string]interface{}{
		"sku_id":   123, // should be string
		"quantity": "five",
		"threshold": 10,
	})
	if err == nil {
		t.Fatal("expected error for wrong field types")
	}
}

func TestStockAlertPayload_Float64AsInt(t *testing.T) {
	// JSON unmarshalling produces float64 for numeric values;
	// the validator should accept whole-number float64 for int fields.
	s := StockAlertPayload{}
	err := s.Validate(map[string]interface{}{
		"sku_id":   "SKU-001",
		"quantity": float64(5),
		"threshold": float64(10),
	})
	if err != nil {
		t.Fatalf("expected no error for float64 whole numbers, got %v", err)
	}
}

func TestStockAlertPayload_NonIntegerFloat(t *testing.T) {
	// Non-whole-number float64 should NOT be accepted for int fields.
	s := StockAlertPayload{}
	err := s.Validate(map[string]interface{}{
		"sku_id":   "SKU-001",
		"quantity": 5.7,
		"threshold": 10,
	})
	if err == nil {
		t.Fatal("expected error for non-integer float64 in int field")
	}
}

// ---------------------------------------------------------------------------
// ProfitWatchPayload validation
// ---------------------------------------------------------------------------

func TestProfitWatchPayload_Valid(t *testing.T) {
	s := ProfitWatchPayload{}
	err := s.Validate(map[string]interface{}{
		"order_id":      "ORD-001",
		"profit_margin": 12.5,
		"threshold":     5.0,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProfitWatchPayload_IntAsFloat(t *testing.T) {
	s := ProfitWatchPayload{}
	err := s.Validate(map[string]interface{}{
		"order_id":      "ORD-001",
		"profit_margin": 12, // Go callers may pass int
		"threshold":     5,
	})
	if err != nil {
		t.Fatalf("expected no error for int values in float fields, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ComplianceCheckPayload validation
// ---------------------------------------------------------------------------

func TestComplianceCheckPayload_Valid(t *testing.T) {
	s := ComplianceCheckPayload{}
	err := s.Validate(map[string]interface{}{
		"listing_id": "LST-001",
		"platform":   "shopee",
		"rule_id":    "R-42",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration: bus with schema registry
// ---------------------------------------------------------------------------

func newTestBusWithSchema(t *testing.T, sr *SchemaRegistry, opts ...BusOption) *Bus {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	allOpts := append([]BusOption{WithSchema(sr)}, opts...)
	b := New(logger, allOpts...)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)
	return b
}

func TestBusWithSchema_ValidPayloadAccepted(t *testing.T) {
	sr := NewSchemaRegistry()
	sr.Register("stock.alert", StockAlertPayload{})

	received := make(chan struct{}, 1)
	b := newTestBusWithSchema(t, sr)
	b.Subscribe("stock.alert", func(ctx context.Context, evt Event) error {
		received <- struct{}{}
		return nil
	})

	_, err := b.Publish(context.Background(), "stock.alert", "test", map[string]interface{}{
		"sku_id":   "SKU-001",
		"quantity": 10,
		"threshold": 5,
	})
	if err != nil {
		t.Fatalf("expected no error for valid payload, got %v", err)
	}

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event delivery")
	}
}

func TestBusWithSchema_InvalidPayloadRejected(t *testing.T) {
	sr := NewSchemaRegistry()
	sr.Register("stock.alert", StockAlertPayload{})

	b := newTestBusWithSchema(t, sr)
	b.Subscribe("stock.alert", func(ctx context.Context, evt Event) error {
		t.Error("handler should not be called for invalid payload")
		return nil
	})

	// quantity is a string instead of int
	_, err := b.Publish(context.Background(), "stock.alert", "test", map[string]interface{}{
		"sku_id":   "SKU-001",
		"quantity": "ten",
		"threshold": 5,
	})
	if err == nil {
		t.Fatal("expected ErrSchemaValidation, got nil")
	}
	if err == ErrQueueFull {
		t.Fatalf("expected schema validation error, got ErrQueueFull")
	}
	if !errors.Is(err, ErrSchemaValidation) {
		t.Fatalf("expected ErrSchemaValidation, got %v", err)
	}
}

func TestBusWithSchema_GlobMatching(t *testing.T) {
	sr := NewSchemaRegistry()
	sr.Register("profit.*", ProfitWatchPayload{})

	b := newTestBusWithSchema(t, sr)
	b.Subscribe("profit.update", func(ctx context.Context, evt Event) error {
		t.Error("handler should not be called for invalid payload")
		return nil
	})

	// profit.* should match profit.update, so validation runs
	_, err := b.Publish(context.Background(), "profit.update", "test", map[string]interface{}{
		"order_id":      "ORD-001",
		"profit_margin": "not-a-number", // invalid
		"threshold":     5.0,
	})
	if err == nil {
		t.Fatal("expected ErrSchemaValidation for invalid payload matched by glob pattern")
	}
}

func TestBusWithSchema_UnregisteredTopicPassesThrough(t *testing.T) {
	sr := NewSchemaRegistry()
	sr.Register("stock.alert", StockAlertPayload{})

	received := make(chan struct{}, 1)
	b := newTestBusWithSchema(t, sr)
	b.Subscribe("some.other.topic", func(ctx context.Context, evt Event) error {
		received <- struct{}{}
		return nil
	})

	// No schema registered for "some.other.topic", should pass through.
	_, err := b.Publish(context.Background(), "some.other.topic", "test", map[string]interface{}{
		"anything": "goes",
	})
	if err != nil {
		t.Fatalf("expected no error for unregistered topic, got %v", err)
	}

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event delivery")
	}
}

func TestBusWithSchema_NilRegistryNoPanic(t *testing.T) {
	// Passing nil to WithSchema should not cause issues.
	b := newTestBus(t, WithSchema(nil))

	received := make(chan struct{}, 1)
	b.Subscribe("test.topic", func(ctx context.Context, evt Event) error {
		received <- struct{}{}
		return nil
	})

	_, err := b.Publish(context.Background(), "test.topic", "test", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event delivery")
	}
}
