package eventbus

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func TestMutationGuard_WithEventBus(t *testing.T) {
	db := dbtest.NewDB(t)
	createEventProcessedTable(t, db)
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	bus := New(logger, WithDB(db))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); bus.Stop() })
	bus.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var handlerCalled atomic.Int32
	bus.Subscribe("guard.test",
		guard.Guard(MutationInfo{SystemAction: "system.test.mutation", Domain: "test"},
			func(ctx context.Context, evt Event) error { handlerCalled.Add(1); return nil }))
	_, err := bus.Publish(context.Background(), "guard.test", "test", map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	poll(t, 2*time.Second, "handler called once", func() bool { return handlerCalled.Load() >= 1 })
	if audit.count() < 2 {
		t.Fatalf("expected >=2 audit entries, got %d", audit.count())
	}
	if audit.inputs[0].Result != "pending" {
		t.Errorf("first entry: expected 'pending', got %q", audit.inputs[0].Result)
	}
	if audit.inputs[0].TriggerType != "eventbus" {
		t.Errorf("first entry: expected trigger_type 'eventbus', got %q", audit.inputs[0].TriggerType)
	}
	last := audit.inputs[audit.count()-1]
	if last.Result != "executed" {
		t.Errorf("last entry: expected 'executed', got %q", last.Result)
	}
}

func TestMutationGuard_WithIdempotency(t *testing.T) {
	db := dbtest.NewDB(t)
	createEventProcessedTable(t, db)
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	bus := New(logger, WithDB(db))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); bus.Stop() })
	bus.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var callCount atomic.Int32
	bus.Subscribe("idemp.guard.*",
		guard.Guard(MutationInfo{SystemAction: "system.test.idempotent", Domain: "test"},
			func(ctx context.Context, evt Event) error { callCount.Add(1); return nil }))
	key := "guard-idemp-test-key"
	_, err := bus.Publish(WithIdempotencyKey(context.Background(), key), "idemp.guard.first", "test", nil)
	if err != nil {
		t.Fatalf("first Publish error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	_, err = bus.Publish(WithIdempotencyKey(context.Background(), key), "idemp.guard.second", "test", nil)
	if err != nil {
		t.Fatalf("second Publish error: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if callCount.Load() != 1 {
		t.Errorf("expected 1 handler call, got %d", callCount.Load())
	}
	if audit.count() < 2 {
		t.Fatalf("expected >=2 audit entries, got %d", audit.count())
	}
	var pc int64
	db.Model(&struct{}{}).Table("event_processed").
		Where("idempotency_key = ? AND state = 'succeeded'", key).Count(&pc)
	if pc != 1 {
		t.Errorf("expected 1 succeeded row, got %d", pc)
	}
}

func TestMutationGuard_FailureInEventBus(t *testing.T) {
	db := dbtest.NewDB(t)
	createEventProcessedTable(t, db)
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	bus := New(logger, WithDB(db))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); bus.Stop() })
	bus.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	bus.Subscribe("guard.fail",
		guard.Guard(MutationInfo{SystemAction: "system.test.fail", Domain: "test"},
			func(ctx context.Context, evt Event) error { return errors.New("handler error") }))
	_, err := bus.Publish(context.Background(), "guard.fail", "test", nil)
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if audit.count() < 2 {
		t.Fatalf("expected >=2 audit entries, got %d", audit.count())
	}
	if audit.last().Result != "failed" {
		t.Errorf("expected 'failed', got %q", audit.last().Result)
	}
}

func TestMutationGuard_NoDB(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	bus := New(logger)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); bus.Stop() })
	bus.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var called bool
	bus.Subscribe("guard.nodb",
		guard.Guard(MutationInfo{SystemAction: "system.test.nodb", Domain: "test"},
			func(ctx context.Context, evt Event) error { called = true; return nil }))
	_, err := bus.Publish(context.Background(), "guard.nodb", "test", nil)
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if !called {
		t.Fatal("handler was not called")
	}
	if audit.count() < 2 {
		t.Fatalf("expected >=2 audit entries, got %d", audit.count())
	}
}

func TestMutationGuard_PreservesDLQSemantics(t *testing.T) {
	db := dbtest.NewDB(t)
	createEventProcessedTable(t, db)
	createDLQTable(t, db)
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	bus := New(logger, WithDB(db), WithDLQ(db), WithMaxRetries(1))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); bus.Stop() })
	bus.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var callCount atomic.Int32
	bus.Subscribe("guard.dlq",
		guard.Guard(MutationInfo{SystemAction: "system.test.dlq", Domain: "test"},
			func(ctx context.Context, evt Event) error { callCount.Add(1); return errors.New("always fail") }))
	_, err := bus.Publish(WithIdempotencyKey(context.Background(), "guard-dlq-key"), "guard.dlq", "test", nil)
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	poll(t, 2*time.Second, "handler called at least once", func() bool { return callCount.Load() >= 1 })
	time.Sleep(200 * time.Millisecond)

	var failedRow struct{ EventID, State string }
	db.Raw(`SELECT event_id, state FROM event_processed WHERE idempotency_key = ?`, "guard-dlq-key").Scan(&failedRow)
	if failedRow.EventID == "" {
		t.Fatal("expected event_processed row, got empty")
	}
	if failedRow.State != "failed" {
		t.Errorf("expected state 'failed', got %q", failedRow.State)
	}
	var dlqCount int64
	db.Model(&struct{}{}).Table("event_dlq").Where("idempotency_key = ?", "guard-dlq-key").Count(&dlqCount)
	if dlqCount == 0 {
		t.Error("expected DLQ entry")
	}
	if audit.count() < 2 {
		t.Fatalf("expected >=2 audit entries, got %d", audit.count())
	}
}
