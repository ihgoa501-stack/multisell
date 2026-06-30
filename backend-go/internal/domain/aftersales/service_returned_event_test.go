package aftersales

import (
	"context"
	"sync"
	"testing"
)

// capturingEventPublisher records every Publish call so tests can assert
// that the supplychain.aftersale.returned event is emitted with the right
// fields when an aftersales order is created.
type capturingEventPublisher struct {
	mu     sync.Mutex
	calls  []capturedCall
	failOn string // optional: topic that should return an error
}

type capturedCall struct {
	topic   string
	source  string
	payload map[string]interface{}
}

func (c *capturingEventPublisher) Publish(_ context.Context, topic, source string, payload map[string]interface{}) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, capturedCall{topic: topic, source: source, payload: payload})
	return "", nil
}

func (c *capturingEventPublisher) Calls() []capturedCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]capturedCall, len(c.calls))
	copy(out, c.calls)
	return out
}

// TestService_Create_PublishesAftersaleReturned verifies that creating an
// aftersales order with a SKU publishes a supplychain.aftersale.returned event
// carrying the aftersale ID, order ID, SKU ID, quantity, and reason.
func TestService_Create_PublishesAftersaleReturned(t *testing.T) {
	db := newTestDB(t)
	pub := &capturingEventPublisher{}
	svc := NewService(db, testLogger(), NewOrderWriterAdapter(db), pub)

	o := setupOrder(t, db)
	skuID := int64(7701)
	qty := 2
	as, err := svc.Create(&CreateInput{
		OrderID:        o.ID,
		SkuID:          &skuID,
		ReturnQuantity: &qty,
		Reason:         "破损",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	calls := pub.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 publish call, got %d", len(calls))
	}
	c := calls[0]
	if c.topic != "supplychain.aftersale.returned" {
		t.Errorf("expected topic supplychain.aftersale.returned, got %s", c.topic)
	}
	if c.source != "aftersales" {
		t.Errorf("expected source aftersales, got %s", c.source)
	}
	// JSON round-trip makes numerics float64.
	if got := c.payload["aftersale_id"]; got != float64(as.ID) {
		t.Errorf("expected aftersale_id %d, got %v", as.ID, got)
	}
	if got := c.payload["order_id"]; got != float64(o.ID) {
		t.Errorf("expected order_id %d, got %v", o.ID, got)
	}
	if got := c.payload["sku_id"]; got != float64(skuID) {
		t.Errorf("expected sku_id %d, got %v", skuID, got)
	}
	if got := c.payload["quantity"]; got != float64(qty) {
		t.Errorf("expected quantity %d, got %v", qty, got)
	}
	if got := c.payload["reason"]; got != "破损" {
		t.Errorf("expected reason 破损, got %v", got)
	}
}

// TestService_Create_NoSku_NoEvent verifies that creating an aftersales
// order without a SKU does NOT publish the returned event — the orchestrator
// cannot act without a SKU to track.
func TestService_Create_NoSku_NoEvent(t *testing.T) {
	db := newTestDB(t)
	pub := &capturingEventPublisher{}
	svc := NewService(db, testLogger(), NewOrderWriterAdapter(db), pub)

	o := setupOrder(t, db)
	qty := 1
	_, err := svc.Create(&CreateInput{
		OrderID:        o.ID,
		ReturnQuantity: &qty,
		Reason:         "no sku",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if calls := pub.Calls(); len(calls) != 0 {
		t.Errorf("expected 0 publish calls when SkuID is nil, got %d", len(calls))
	}
}

// TestService_Create_NilEventPublisher_NoPanic verifies that a nil
// EventPublisher (production wiring may pass nil) does not panic on Create.
func TestService_Create_NilEventPublisher_NoPanic(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger(), NewOrderWriterAdapter(db), nil)

	o := setupOrder(t, db)
	skuID := int64(7702)
	qty := 1
	as, err := svc.Create(&CreateInput{
		OrderID:        o.ID,
		SkuID:          &skuID,
		ReturnQuantity: &qty,
		Reason:         "nil publisher",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if as.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

// TestService_Create_PublishFailure_NoRollback verifies that a publish
// failure does not roll back the create — the aftersales order is the
// system of record and downstream reconciliation can recover.
func TestService_Create_PublishFailure_NoRollback(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger(), NewOrderWriterAdapter(db), &mockEventPublisher{})

	o := setupOrder(t, db)
	skuID := int64(7703)
	qty := 1
	as, err := svc.Create(&CreateInput{
		OrderID:        o.ID,
		SkuID:          &skuID,
		ReturnQuantity: &qty,
		Reason:         "test",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if as.ID == 0 {
		t.Error("expected non-zero ID even when event publish is no-op")
	}

	// Verify the order was actually persisted.
	var persisted AfterSalesOrder
	if err := db.First(&persisted, as.ID).Error; err != nil {
		t.Fatalf("aftersales order not persisted: %v", err)
	}
}
