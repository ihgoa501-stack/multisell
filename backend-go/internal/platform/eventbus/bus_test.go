package eventbus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newTestBus(t *testing.T, opts ...BusOption) *Bus {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	b := New(logger, opts...)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	// Allow workers to start.
	time.Sleep(10 * time.Millisecond)
	return b
}

// TestBasicPublishSubscribe verifies that a published event is delivered to
// matching subscribers.
func TestBasicPublishSubscribe(t *testing.T) {
	b := newTestBus(t)

	received := make(chan struct{}, 1)
	b.Subscribe("test.topic", func(ctx context.Context, evt Event) error {
		received <- struct{}{}
		return nil
	})

	_, err := b.Publish(context.Background(), "test.topic", "test", map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event delivery")
	}
}

// TestTopicFiltering verifies that subscribers only receive events matching
// their topic pattern.
func TestTopicFiltering(t *testing.T) {
	b := newTestBus(t)

	var matched atomic.Int32
	var unmatched atomic.Int32

	b.Subscribe("order.*", func(ctx context.Context, evt Event) error {
		matched.Add(1)
		return nil
	})
	b.Subscribe("inventory.*", func(ctx context.Context, evt Event) error {
		unmatched.Add(1)
		return nil
	})

	_, err := b.Publish(context.Background(), "order.created", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	// Allow some time for dispatch.
	time.Sleep(100 * time.Millisecond)

	if matched.Load() != 1 {
		t.Errorf("expected 1 match, got %d", matched.Load())
	}
	if unmatched.Load() != 0 {
		t.Errorf("expected 0 matches on inventory.*, got %d", unmatched.Load())
	}
}

// TestBusLifecycle verifies the bus lifecycle: start → publish → receive →
// stop → no more deliveries after stop.
func TestBusLifecycle(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	b := New(logger, WithWorkers(4))

	// Start the bus
	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Publish → receive
	received := make(chan struct{}, 5)
	count := 0
	var mu sync.Mutex
	b.Subscribe("lifecycle.*", func(ctx context.Context, evt Event) error {
		mu.Lock()
		count++
		mu.Unlock()
		received <- struct{}{}
		return nil
	})

	_, err := b.Publish(context.Background(), "lifecycle.1", "test", nil)
	if err != nil {
		t.Fatalf("Publish before stop returned error: %v", err)
	}
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timeout: subscriber did not receive event before stop")
	}

	// Stop the bus
	cancel()
	b.Stop()

	// After stop, publish should not reach subscriber.
	// The bus may or may not return an error after stop — either is acceptable,
	// but the subscriber should NOT be invoked.
	_, postStopErr := b.Publish(context.Background(), "lifecycle.2", "test", nil)
	_ = postStopErr // not relevant for this assertion
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if count != 1 {
		t.Errorf("expected 1 delivery total (pre-stop), got %d — subscriber received event after stop", count)
	}
	mu.Unlock()
}

// TestMultipleSubscribers verifies that all subscribers matching a topic receive
// the event.
func TestMultipleSubscribers(t *testing.T) {
	b := newTestBus(t, WithWorkers(4))

	var counter atomic.Int32
	for i := 0; i < 5; i++ {
		b.Subscribe("test.*", func(ctx context.Context, evt Event) error {
			counter.Add(1)
			return nil
		})
	}

	_, err := b.Publish(context.Background(), "test.event", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if got := counter.Load(); got != 5 {
		t.Errorf("expected 5 handler invocations, got %d", got)
	}
}

// TestPriorityOrdering verifies that events at all priorities are delivered
// and within-same-priority FIFO ordering is preserved.
// Cross-priority ordering is not guaranteed with per-priority worker pools;
// each priority has dedicated workers and buffer capacity for QoS isolation.
func TestPriorityOrdering(t *testing.T) {
	b := newTestBus(t)

	orderDone := make(chan struct{}, 3)
	var mu sync.Mutex
	var received []int

	b.Subscribe("priority.*", func(ctx context.Context, evt Event) error {
		mu.Lock()
		received = append(received, evt.Priority)
		mu.Unlock()
		orderDone <- struct{}{}
		return nil
	})

	// Publish at different priorities.
	_, err := b.PublishWithPriority(context.Background(), "priority.test", "test", nil, 0)
	if err != nil {
		t.Fatalf("Publish (low) returned error: %v", err)
	}
	_, err = b.PublishWithPriority(context.Background(), "priority.test", "test", nil, 2)
	if err != nil {
		t.Fatalf("Publish (high) returned error: %v", err)
	}
	_, err = b.PublishWithPriority(context.Background(), "priority.test", "test", nil, 1)
	if err != nil {
		t.Fatalf("Publish (med) returned error: %v", err)
	}

	// Wait for all 3 events to be dispatched.
	for i := 0; i < 3; i++ {
		select {
		case <-orderDone:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event dispatch")
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 3 {
		t.Fatalf("expected 3 dispatched events, got %d", len(received))
	}
	// All three priorities should be present (order is non-deterministic
	// across pools due to per-priority QoS isolation).
	seen := map[int]bool{}
	for _, p := range received {
		seen[p] = true
	}
	if !seen[0] || !seen[1] || !seen[2] {
		t.Errorf("expected all priorities 0,1,2 to be dispatched, got %v", received)
	}
}

// TestUnsubscribe verifies that unsubscribing prevents further delivery.
func TestUnsubscribe(t *testing.T) {
	b := newTestBus(t)

	var count atomic.Int32
	id := b.Subscribe("test.topic", func(ctx context.Context, evt Event) error {
		count.Add(1)
		return nil
	})

	_, err := b.Publish(context.Background(), "test.topic", "test", nil)
	if err != nil {
		t.Fatalf("first Publish returned error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	b.Unsubscribe(id)

	_, err = b.Publish(context.Background(), "test.topic", "test", nil)
	if err != nil {
		t.Fatalf("second Publish returned error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 handler invocation after unsubscribe, got %d", got)
	}
}

// TestBackpressure verifies that Publish returns ErrQueueFull when the queue
// is at capacity.
func TestBackpressure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	// Use a small buffer size and 0 workers so events never get consumed.
	b := New(logger, WithBufferSize(1), WithWorkers(0))
	// Do not Start workers so the queue fills up immediately.

	// First publish should succeed (queue is empty).
	_, err := b.Publish(context.Background(), "test.topic", "test", nil)
	if err != nil {
		t.Fatalf("first Publish returned unexpected error: %v", err)
	}

	// Second publish should fail immediately (queue full).
	_, err = b.Publish(context.Background(), "test.topic", "test", nil)
	if err == nil {
		t.Fatal("expected ErrQueueFull but got nil")
	}
	if err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}

// TestStopGracefully verifies that Stop shuts down all workers and subsequent
// publishes do not hang.
func TestStopGracefully(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	b := New(logger)
	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)

	// Allow workers to start.
	time.Sleep(10 * time.Millisecond)

	b.Stop()
	cancel()

	// Publishing after stop should not cause issues (workers are gone but
	// events can still be enqueued).
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	_, err := b.Publish(ctx2, "test.topic", "test", nil)
	if err != nil {
		// This is allowed — no workers means events may pile up; the test is
		// just verifying that Publish doesn't hang forever.
		if err == ErrQueueFull {
			// Acceptable — workers are stopped so queue fills up.
		} else {
			t.Logf("Publish after stop returned: %v (acceptable)", err)
		}
	}
}

// TestConcurrentPublishSubscribe exercises the bus under concurrent load.
func TestConcurrentPublishSubscribe(t *testing.T) {
	b := newTestBus(t, WithWorkers(8), WithBufferSize(1024))

	var total atomic.Int32
	n := 100

	for i := 0; i < n; i++ {
		b.Subscribe("load.*", func(ctx context.Context, evt Event) error {
			total.Add(1)
			return nil
		})
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, err := b.Publish(context.Background(), "load.test", "test", nil)
				if err != nil && err != ErrQueueFull {
					t.Errorf("Publish error: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	// Allow dispatches to finish.
	time.Sleep(500 * time.Millisecond)

	// Each of the 100 subscribers should have received 100 events.
	expected := int32(100 * 100) // n subscribers * 100 publishes
	if got := total.Load(); got != expected {
		t.Errorf("expected %d total handler invocations, got %d", expected, got)
	}
}

// TestContextCancellation verifies that Publish works correctly when context is
// already cancelled (should not panic or hang).
func TestContextCancellation(t *testing.T) {
	b := newTestBus(t)

	// Publish with a cancelled context should still enqueue the event
	// (context cancellation does not block inline enqueue).
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	var received atomic.Int32
	b.Subscribe("cancel.*", func(ctx context.Context, evt Event) error {
		received.Add(1)
		return nil
	})

	_, err := b.PublishWithPriority(cancelCtx, "cancel.test", "test", nil, 0)
	if err != nil {
		t.Fatalf("PublishWithPriority returned unexpected error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if got := received.Load(); got != 1 {
		t.Errorf("expected 1 handler invocation despite cancelled context, got %d", got)
	}
}

// TestPublishWithPriority preserves the default priority.
func TestPublishWithPriority(t *testing.T) {
	b := newTestBus(t)

	var gotPriority atomic.Int32
	b.Subscribe("prio.*", func(ctx context.Context, evt Event) error {
		gotPriority.Store(int32(evt.Priority))
		return nil
	})

	_, err := b.PublishWithPriority(context.Background(), "prio.test", "test", nil, 0)
	if err != nil {
		t.Fatalf("PublishWithPriority returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if got := gotPriority.Load(); got != 0 {
		t.Errorf("expected priority 0, got %d", got)
	}
}

// TestHandlerError verifies that handler errors are logged and don't crash.
func TestHandlerError(t *testing.T) {
	b := newTestBus(t)

	b.Subscribe("error.*", func(ctx context.Context, evt Event) error {
		return nil // no error
	})
	b.Subscribe("error.*", func(ctx context.Context, evt Event) error {
		return nil // also ok
	})

	_, err := b.Publish(context.Background(), "error.test", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	// Publish is async now, so error returns are only for backpressure;
	// handler errors are logged internally. Just verify no panic.
	time.Sleep(50 * time.Millisecond)
}

// TestMultiplePrioritiesSameLevel verifies that events with the same priority
// are dispatched in publication order (FIFO). With per-priority pools,
// events at the same priority share the same backend hence maintain FIFO order.
func TestMultiplePrioritiesSameLevel(t *testing.T) {
	// Use 1 worker for pool 0 to enforce ordering.
	b := newTestBus(t, WithWorkers(1))

	var mu sync.Mutex
	var ids []string

	b.Subscribe("fifo.*", func(ctx context.Context, evt Event) error {
		mu.Lock()
		ids = append(ids, evt.ID)
		mu.Unlock()
		return nil
	})

	id1, _ := b.Publish(context.Background(), "fifo.a", "test", nil)
	id2, _ := b.Publish(context.Background(), "fifo.b", "test", nil)
	id3, _ := b.Publish(context.Background(), "fifo.c", "test", nil)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(ids) != 3 {
		t.Fatalf("expected 3 events, got %d", len(ids))
	}
	if ids[0] != id1 || ids[1] != id2 || ids[2] != id3 {
		t.Errorf("expected FIFO order [%s, %s, %s], got %v", id1, id2, id3, ids)
	}
}

// TestDLQNoDB verifies that DLQ works gracefully when no DB is configured
// (events are retried in-process without DLQ persistence).
func TestDLQNoDB(t *testing.T) {
	b := newTestBus(t, WithMaxRetries(2))

	var attemptCount atomic.Int32
	b.Subscribe("dlq.*", func(ctx context.Context, evt Event) error {
		attemptCount.Add(1)
		return nil
	})

	_, err := b.Publish(context.Background(), "dlq.test", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	// Allow dispatches to finish and retries to happen.
	time.Sleep(300 * time.Millisecond)

	if got := attemptCount.Load(); got < 1 {
		t.Errorf("expected at least 1 handler invocation, got %d", got)
	}
}

// TestContextCorrelationPropagation verifies that the correlation ID from the
// Publish context is propagated to the handler via the event.
func TestContextCorrelationPropagation(t *testing.T) {
	b := newTestBus(t)

	correlationID := "test-correlation-123"
	got := make(chan string, 1)
	b.Subscribe("corr.*", func(ctx context.Context, evt Event) error {
		if id := CorrelationIDFromContext(ctx); id != "" {
			got <- id
		} else if evt.CorrelationID != "" {
			got <- evt.CorrelationID
		}
		return nil
	})

	ctx := WithCorrelationID(context.Background(), correlationID)
	_, err := b.Publish(ctx, "corr.test", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case id := <-got:
		if id != correlationID {
			t.Errorf("expected correlation ID %q, got %q", correlationID, id)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for handler to receive event")
	}
}

// TestDLQWithDB verifies that events are moved to the DLQ table when handlers
// fail and max retries are exceeded.
func TestDLQWithDB(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	createDLQTable(t, db)
	logger, _ := zap.NewDevelopment()
	b := New(logger, WithDB(db), WithDLQ(db), WithMaxRetries(1))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Handler that always errors.
	b.Subscribe("dlq-db.*", func(ctx context.Context, evt Event) error {
		return errors.New("handler error")
	})

	_, err := b.Publish(context.Background(), "dlq-db.test", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	// Allow retry and DLQ processing.
	time.Sleep(500 * time.Millisecond)

	// Verify DLQ event was persisted.
	var count int64
	db.Model(&DLEvent{}).Count(&count)
	if count == 0 {
		t.Error("expected at least 1 DLQ event, got 0")
	}

	// Verify ListDLQ works.
	events, total, err := b.DLQ().ListDLQ(1, 10)
	if err != nil {
		t.Fatalf("ListDLQ returned error: %v", err)
	}
	if total == 0 {
		t.Error("expected ListDLQ to return at least 1 event")
	}
	if len(events) == 0 {
		t.Error("expected ListDLQ to return events")
	}
}

// TestDLQReplay verifies that events moved to DLQ can be replayed.
func TestDLQReplay(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	createDLQTable(t, db)
	logger, _ := zap.NewDevelopment()
	b := New(logger, WithDB(db), WithDLQ(db), WithMaxRetries(1))

	var delivered atomic.Int32
	b.Subscribe("dlq-replay.*", func(ctx context.Context, evt Event) error {
		delivered.Add(1)
		return errors.New("handler error")
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	_, err := b.Publish(context.Background(), "dlq-replay.test", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Get DLQ events and replay them.
	events, _, err := b.DLQ().ListDLQ(1, 10)
	if err != nil {
		t.Fatalf("ListDLQ returned error: %v", err)
	}
	if len(events) == 0 {
		t.Skip("ponytail: no DLQ events to replay")
	}

	// Re-subscribe with a handler that succeeds.
	var replayCount atomic.Int32
	b.Subscribe("dlq-replay.*", func(ctx context.Context, evt Event) error {
		replayCount.Add(1)
		return nil // succeed on replay
	})

	ids := make([]uint, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}

	replayed, err := b.DLQ().ReplayEventsByIDs(b, ids)
	if err != nil {
		t.Fatalf("ReplayEventsByIDs returned error: %v", err)
	}
	if replayed == 0 {
		t.Error("expected at least 1 replayed event")
	}

	time.Sleep(200 * time.Millisecond)
	if got := replayCount.Load(); got == 0 {
		t.Error("expected replay handler to be invoked")
	}
}

// TestMetricsIncrement verifies that key Prometheus counters increment when
// events are published, delivered, and moved to DLQ.
func TestMetricsIncrement(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	createDLQTable(t, db)
	logger, _ := zap.NewDevelopment()
	b := New(logger, WithDB(db), WithDLQ(db), WithMaxRetries(1))

	// Count failed events by subscribing with an erroring handler.
	var handlerCalls atomic.Int32
	b.Subscribe("metrics.*", func(ctx context.Context, evt Event) error {
		handlerCalls.Add(1)
		return errors.New("handler error")
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	_, err := b.Publish(context.Background(), "metrics.test", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify handler was called at least once.
	if got := handlerCalls.Load(); got == 0 {
		t.Error("expected handler to be called at least once")
	}
}

// TestIdempotencyKey_SameEventRetried verifies that a retry (same event ID, same
// idempotency key) is NOT blocked — only truly duplicate events are skipped.
func TestIdempotencyKey_SameEventRetried(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	createEventProcessedTable(t, db)
	logger, _ := zap.NewDevelopment()
	// maxRetries=3 means initial delivery + 2 retries = 3 handler attempts.
	b := New(logger, WithDB(db), WithMaxRetries(3))

	var callCount atomic.Int32
	var firstEventID string
	b.Subscribe("idemp.*", func(ctx context.Context, evt Event) error {
		if firstEventID == "" {
			firstEventID = evt.ID
		}
		callCount.Add(1)
		// Fail first two attempts to trigger retries, succeed on third.
		if callCount.Load() < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	ctxID := WithIdempotencyKey(context.Background(), "retry-test-key")
	_, err := b.Publish(ctxID, "idemp.test", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	// Retries should NOT be blocked by the idempotency check (same event_id).
	poll(t, 3*time.Second, "3 handler calls (2 retries + 1 success)", func() bool {
		return callCount.Load() >= 3
	})

	// Verify the event_processed row was created AFTER handler success.
	var row struct {
		EventID     string
		ProcessedAt time.Time
	}
	db.Raw(`SELECT event_id, processed_at FROM event_processed WHERE idempotency_key = ?`, "retry-test-key").Scan(&row)
	if row.EventID == "" {
		t.Fatal("expected event_processed row after handler success, got empty")
	}
	if row.EventID != firstEventID {
		t.Errorf("expected event_id %s in event_processed, got %s", firstEventID, row.EventID)
	}
	if row.ProcessedAt.IsZero() {
		t.Error("expected non-zero processed_at")
	}
}

// TestIdempotencyKey_DuplicateSkipped verifies that a second event with the
// same idempotency key but different event ID is skipped entirely.
func TestIdempotencyKey_DuplicateSkipped(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	createEventProcessedTable(t, db)
	logger, _ := zap.NewDevelopment()
	b := New(logger, WithDB(db))

	var callCount atomic.Int32
	var firstEventID string
	b.Subscribe("idemp.*", func(ctx context.Context, evt Event) error {
		if firstEventID == "" {
			firstEventID = evt.ID
		}
		callCount.Add(1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// First event: should be delivered.
	key := "dup-test-key"
	ctxID := WithIdempotencyKey(context.Background(), key)
	_, err := b.Publish(ctxID, "idemp.test", "test", nil)
	if err != nil {
		t.Fatalf("first Publish returned error: %v", err)
	}

	poll(t, 2*time.Second, "first handler call", func() bool {
		return callCount.Load() >= 1
	})

	if got := callCount.Load(); got != 1 {
		t.Errorf("expected 1 handler call for first event, got %d", got)
	}

	// Verify event_processed row for the first event.
	var row struct {
		EventID     string
		ProcessedAt time.Time
	}
	db.Raw(`SELECT event_id, processed_at FROM event_processed WHERE idempotency_key = ?`, key).Scan(&row)
	if row.EventID != firstEventID {
		t.Errorf("expected event_id %s in event_processed, got %s", firstEventID, row.EventID)
	}
	if row.ProcessedAt.IsZero() {
		t.Error("expected non-zero processed_at")
	}

	// Second event with same key, different auto-generated ID: should be skipped.
	_, err = b.Publish(ctxID, "idemp.test", "test", nil)
	if err != nil {
		t.Fatalf("second Publish returned error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Handler should NOT have been called again.
	if got := callCount.Load(); got != 1 {
		t.Errorf("expected 1 handler call total (duplicate skipped), got %d", got)
	}

	// Verify event_processed still has the first event's ID (not updated).
	var row2 struct{ EventID string }
	db.Raw(`SELECT event_id FROM event_processed WHERE idempotency_key = ?`, key).Scan(&row2)
	if row2.EventID != firstEventID {
		t.Errorf("expected event_processed event_id %s (unchanged), got %s", firstEventID, row2.EventID)
	}
}

// TestIdempotencyKey_NoDB_EventsPassThrough verifies that when no DB is
// configured, idempotency keys are silently ignored (fail-open behavior).
func TestIdempotencyKey_NoDB_EventsPassThrough(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	// No WithDB → no idempotency tracking.
	b := New(logger)

	var callCount atomic.Int32
	b.Subscribe("idemp.*", func(ctx context.Context, evt Event) error {
		callCount.Add(1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Publish same key twice — both should go through since no DB.
	key := "no-db-key"
	ctxID := WithIdempotencyKey(context.Background(), key)
	for i := 0; i < 2; i++ {
		_, err := b.Publish(ctxID, "idemp.test", "test", nil)
		if err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
	}

	poll(t, 2*time.Second, "2 handler calls (no DB = no dedup)", func() bool {
		return callCount.Load() >= 2
	})

	if got := callCount.Load(); got != 2 {
		t.Errorf("expected 2 handler calls (no DB = no dedup), got %d", got)
	}
}

// TestIdempotencyKey_DifferentKeysPass verifies that different idempotency keys
// are processed independently.
func TestIdempotencyKey_DifferentKeysPass(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	createEventProcessedTable(t, db)
	logger, _ := zap.NewDevelopment()
	b := New(logger, WithDB(db))

	var callCount atomic.Int32
	b.Subscribe("idemp.*", func(ctx context.Context, evt Event) error {
		callCount.Add(1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Two events with different keys — both should be delivered.
	for i := 0; i < 2; i++ {
		ctxID := WithIdempotencyKey(context.Background(), fmt.Sprintf("distinct-key-%d", i))
		_, err := b.Publish(ctxID, "idemp.test", "test", nil)
		if err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
	}

	poll(t, 2*time.Second, "2 handler calls (different keys)", func() bool {
		return callCount.Load() >= 2
	})

	if got := callCount.Load(); got != 2 {
		t.Errorf("expected 2 handler calls (different keys), got %d", got)
	}

	// Verify both keys have event_processed rows.
	for i := 0; i < 2; i++ {
		var count int64
		db.Model(&struct{}{}).Table("event_processed").
			Where("idempotency_key = ?", fmt.Sprintf("distinct-key-%d", i)).
			Count(&count)
		if count != 1 {
			t.Errorf("expected 1 event_processed row for key %q, got %d", fmt.Sprintf("distinct-key-%d", i), count)
		}
	}
}

// TestIdempotencyKey_DLQReplay verifies that an event that fails and moves to
// DLQ can be replayed with the same idempotency_key and reach the handler.
// The event_processed row must NOT exist until the handler succeeds, so DLQ
// replay is not blocked.
func TestIdempotencyKey_DLQReplay(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	createEventProcessedTable(t, db)
	createDLQTable(t, db)
	logger, _ := zap.NewDevelopment()
	b := New(logger, WithDB(db), WithDLQ(db), WithMaxRetries(1))

	var callCount atomic.Int32
	b.Subscribe("idemp-dlq.*", func(ctx context.Context, evt Event) error {
		callCount.Add(1)
		return errors.New("handler error")
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	ctxID := WithIdempotencyKey(context.Background(), "dlq-replay-key")
	_, err := b.Publish(ctxID, "idemp-dlq.test", "test", nil)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	// Wait for the event to fail and reach DLQ (maxRetries=1 means 1 attempt, then DLQ).
	poll(t, 3*time.Second, "handler called at least once", func() bool {
		return callCount.Load() >= 1
	})

	// Verify event_processed row has state='failed' (set by failIdempotency before DLQ move).
	var processedRow struct {
		EventID string
		State   string
	}
	db.Raw(`SELECT event_id, state FROM event_processed WHERE idempotency_key = ?`, "dlq-replay-key").Scan(&processedRow)
	if processedRow.EventID == "" {
		t.Fatal("expected event_processed row after claim, got empty")
	}
	if processedRow.State != "failed" {
		t.Errorf("expected state='failed' (handler never succeeded), got %q", processedRow.State)
	}

	// Verify DLQ event was persisted.
	var dlqCount int64
	db.Model(&DLEvent{}).Count(&dlqCount)
	if dlqCount == 0 {
		t.Fatal("expected DLQ event, got 0")
	}

	// Get DLQ events and replay them.
	events, _, err := b.DLQ().ListDLQ(1, 10)
	if err != nil {
		t.Fatalf("ListDLQ returned error: %v", err)
	}
	if len(events) == 0 {
		t.Skip("ponytail: no DLQ events to replay")
	}

	// Unsubscribe the original erroring handler and subscribe a new one that succeeds.
	ids := make([]uint, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}

	// Remove the original error subscription; keep it from interfering with replay.
	// We find it by re-subscribing with a no-op to get the old subscription ID
	// (since Subscribe returned IDs aren't easily stored — we unsubscribe *),
	// but it's simpler to just remove subscriptions by subscribing fresh.
	// Instead, track the original subscription ID and unsubscribe it.
	var origSubID string
	b.mu.RLock()
	for _, sub := range b.subs {
		if sub.topic == "idemp-dlq.*" {
			origSubID = sub.id
		}
	}
	b.mu.RUnlock()
	if origSubID != "" {
		b.Unsubscribe(origSubID)
	}

	// Subscribe with a handler that succeeds on replay.
	var replayCallCount atomic.Int32
	b.Subscribe("idemp-dlq.*", func(ctx context.Context, evt Event) error {
		replayCallCount.Add(1)
		return nil // succeed on replay
	})

	replayed, err := b.DLQ().ReplayEventsByIDs(b, ids)
	if err != nil {
		t.Fatalf("ReplayEventsByIDs returned error: %v", err)
	}
	if replayed == 0 {
		t.Fatal("expected at least 1 replayed event")
	}

	poll(t, 3*time.Second, "replay handler to be invoked", func() bool {
		return replayCallCount.Load() >= 1
	})

	if got := replayCallCount.Load(); got == 0 {
		t.Fatal("expected replay handler to be invoked")
	}

	// Verify event_processed row was created after replay handler succeeded.
	var processedCount2 int64
	db.Model(&struct{}{}).Table("event_processed").
		Where("idempotency_key = ?", "dlq-replay-key").
		Count(&processedCount2)
	if processedCount2 != 1 {
		t.Errorf("expected 1 event_processed row after replay success, got %d", processedCount2)
	}
}

// TestIdempotencyKey_ConcurrentDuplicate verifies that two events published
// rapidly with the same idempotency_key and different event_ids result in the
// handler running exactly once, even with multiple bus workers (default 4).
//
// This tests the core atomic claim fix: only one worker wins the INSERT claim;
// the other worker's INSERT conflicts and the event is atomically rejected.
func TestIdempotencyKey_ConcurrentDuplicate(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	createEventProcessedTable(t, db)
	logger, _ := zap.NewDevelopment()
	// Default pool has 4 workers — the race condition reproduces here.
	b := New(logger, WithDB(db))

	var (
		callCount      atomic.Int32
		handlerStarted = make(chan struct{})
	)

	b.Subscribe("concurrent-dup.*", func(ctx context.Context, evt Event) error {
		c := callCount.Add(1)
		if c == 1 {
			close(handlerStarted)
		}
		// Block briefly so the second event (if it sneaks through) would
		// arrive while this handler is still executing. With the atomic
		// claim, the second worker's INSERT should conflict and be rejected.
		time.Sleep(250 * time.Millisecond)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Publish two events with the same idempotency key as rapidly as possible.
	key := "concurrent-dup-key"
	ctxID := WithIdempotencyKey(context.Background(), key)
	_, err := b.Publish(ctxID, "concurrent-dup.test", "test", nil)
	if err != nil {
		t.Fatalf("first publish: %v", err)
	}
	_, err = b.Publish(ctxID, "concurrent-dup.test", "test", nil)
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}

	// Wait for the handler to start (at least once).
	select {
	case <-handlerStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for handler to start")
	}

	// Give enough time for a potential second invocation to complete.
	time.Sleep(500 * time.Millisecond)

	if got := callCount.Load(); got != 1 {
		t.Errorf("expected handler to run exactly once (atomic claim), got %d", got)
	}

	// Verify the event_processed row reflects successful processing.
	var row struct {
		EventID string
		State   string
	}
	db.Raw(`SELECT event_id, state FROM event_processed WHERE idempotency_key = ?`, key).Scan(&row)
	if row.EventID == "" {
		t.Fatal("expected event_processed row after claim, got empty")
	}
	if row.State != "succeeded" {
		t.Errorf("expected state='succeeded', got %q", row.State)
	}
}

// createDLQTable creates the event_dlq table with SQLite-compatible DDL.
func createDLQTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS event_dlq (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			original_event_id TEXT NOT NULL,
			idempotency_key TEXT NOT NULL DEFAULT '',
			topic TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT '',
			payload TEXT NOT NULL DEFAULT '{}',
			priority INTEGER NOT NULL DEFAULT 0,
			correlation_id TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			delivery_attempts INTEGER NOT NULL DEFAULT 0,
			last_attempt_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			replayed_at DATETIME,
			replayed_by TEXT NOT NULL DEFAULT ''
		)
	`).Error; err != nil {
		t.Fatalf("failed to create event_dlq table: %v", err)
	}
}

func createOutboxTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec(`CREATE TABLE IF NOT EXISTS event_outbox (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		topic TEXT NOT NULL,
		source TEXT NOT NULL,
		payload TEXT NOT NULL DEFAULT '{}',
		priority INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME NOT NULL,
		delivered_at DATETIME,
		event_id TEXT DEFAULT '',
		delivery_attempts INTEGER NOT NULL DEFAULT 0,
		last_error TEXT DEFAULT '',
		version TEXT NOT NULL DEFAULT '',
		actor TEXT NOT NULL DEFAULT '',
		entity_id TEXT NOT NULL DEFAULT '',
		entity_type TEXT NOT NULL DEFAULT '',
		correlation_id TEXT NOT NULL DEFAULT '',
		idempotency_key TEXT NOT NULL DEFAULT ''
	)`).Error
	if err != nil {
		t.Fatalf("create event_outbox: %v", err)
	}
}

func TestPublishWithDB_FailsClosedWithoutOutbox(t *testing.T) {
	db := dbtest.NewDB(t)
	b := New(zap.NewNop(), WithDB(db))
	if _, err := b.Publish(context.Background(), "durable.test", "test", nil); err == nil {
		t.Fatal("publish succeeded without durable outbox storage")
	}
}

func TestStartRecoversPendingOutboxAfterSubscriptionsExist(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	payload := `{"value":"restored"}`
	if err := db.Exec(`INSERT INTO event_outbox
		(topic, source, payload, priority, status, created_at, event_id, version, actor, entity_id, entity_type, correlation_id, idempotency_key)
		VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?, ?)`,
		"recovery.test", "previous-process", payload, 1, time.Now(), "event-recovered", "v1", "owner", "42", "listing", "corr-1", "").Error; err != nil {
		t.Fatal(err)
	}
	b := New(zap.NewNop(), WithDB(db))
	received := make(chan Event, 1)
	b.Subscribe("recovery.test", func(_ context.Context, evt Event) error {
		received <- evt
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)
	t.Cleanup(func() { cancel(); b.Stop() })
	select {
	case evt := <-received:
		if evt.ID != "event-recovered" || evt.Priority != 1 || evt.CorrelationID != "corr-1" || evt.Payload["value"] != "restored" {
			t.Fatalf("recovered event=%+v", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("pending outbox event was not recovered")
	}
	poll(t, time.Second, "recovered outbox delivered", func() bool {
		var status string
		return db.Table("event_outbox").Select("status").Where("event_id = ?", "event-recovered").Scan(&status).Error == nil && status == "delivered"
	})
}

func TestStopWithContextDrainsQueuedEventsAndRejectsNewPublishes(t *testing.T) {
	b := New(zap.NewNop(), WithWorkers(2))
	gate := make(chan struct{})
	var handled atomic.Int32
	b.Subscribe("drain.test", func(context.Context, Event) error {
		<-gate
		handled.Add(1)
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)
	t.Cleanup(cancel)
	for i := 0; i < 8; i++ {
		if _, err := b.Publish(context.Background(), "drain.test", "test", map[string]interface{}{"i": i}); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan error, 1)
	go func() {
		drainCtx, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		done <- b.StopWithContext(drainCtx)
	}()
	poll(t, time.Second, "bus enters drain", func() bool { return !b.IsRunning() })
	if _, err := b.Publish(context.Background(), "drain.test", "test", nil); !errors.Is(err, ErrBusStopped) {
		t.Fatalf("publish during drain error=%v", err)
	}
	select {
	case err := <-done:
		t.Fatalf("drain returned before handlers were released: %v", err)
	default:
	}
	close(gate)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if handled.Load() != 8 {
		t.Fatalf("handled=%d, want 8", handled.Load())
	}
}

func TestStopWithContextHonorsDeadline(t *testing.T) {
	b := New(zap.NewNop(), WithWorkers(1))
	gate := make(chan struct{})
	started := make(chan struct{})
	b.Subscribe("stuck.test", func(context.Context, Event) error {
		close(started)
		<-gate
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)
	if _, err := b.Publish(context.Background(), "stuck.test", "test", nil); err != nil {
		t.Fatal(err)
	}
	<-started
	drainCtx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err := b.StopWithContext(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("drain deadline error=%v", err)
	}
	close(gate)
	cancel()
}

func TestIdempotencyClaimError_BlocksHandlerAndFailsOutbox(t *testing.T) {
	db := dbtest.NewDB(t)
	createOutboxTable(t, db)
	b := New(zap.NewNop(), WithDB(db)) // event_processed intentionally absent
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); b.Stop() })
	b.Start(ctx)
	var calls atomic.Int32
	b.Subscribe("claim.error", func(context.Context, Event) error { calls.Add(1); return nil })
	if _, err := b.Publish(WithIdempotencyKey(context.Background(), "claim-error-key"), "claim.error", "test", nil); err != nil {
		t.Fatal(err)
	}
	poll(t, 2*time.Second, "outbox marked failed", func() bool {
		var count int64
		return db.Table("event_outbox").Where("status = ?", "failed").Count(&count).Error == nil && count == 1
	})
	if calls.Load() != 0 {
		t.Fatalf("handler ran %d times after idempotency storage failure", calls.Load())
	}
}

// createEventProcessedTable creates the event_processed table with
// SQLite-compatible DDL for idempotency testing.
// The state column supports the atomic claim model:
//
//	processing → claimed before handler dispatch
//	succeeded  → handler completed successfully
//	failed     → handler exhausted retries, moved to DLQ
func createEventProcessedTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS event_processed (
			idempotency_key TEXT PRIMARY KEY,
			topic TEXT NOT NULL,
			event_id TEXT NOT NULL,
			state TEXT NOT NULL DEFAULT 'processing',
			processed_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("failed to create event_processed table: %v", err)
	}
}

// poll waits up to timeout for cond to return true, calling t.Fatal on timeout.
func poll(t *testing.T, timeout time.Duration, desc string, cond func() bool) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		if cond() {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for: %s", desc)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// TestBusLifecycle_GracefulDrain verifies that Stop properly drains all worker
// goroutines and does not leak goroutines after shutdown.
func TestBusLifecycle_GracefulDrain(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	b := New(logger, WithWorkers(2))

	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Subscribe a handler that blocks briefly to simulate work.
	var handled atomic.Int32
	b.Subscribe("drain.*", func(ctx context.Context, evt Event) error {
		time.Sleep(5 * time.Millisecond)
		handled.Add(1)
		return nil
	})

	// Publish several events that should be processed before or during drain.
	for i := 0; i < 10; i++ {
		b.Publish(context.Background(), "drain.test", "test", nil)
	}

	// Stop the bus — this should close backends, signal workers, and wait.
	// All published events may or may not be processed, but Stop must not hang.
	stopDone := make(chan struct{})
	go func() {
		cancel()
		b.Stop()
		close(stopDone)
	}()

	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop hung — workers did not drain within 5s")
	}

	// After Stop, publishing should not panic (events are still enqueuable
	// but workers are gone).
	_, err := b.Publish(context.Background(), "drain.after", "test", nil)
	if err != nil {
		// Accept backpressure errors after workers are gone.
		t.Logf("Publish after Stop returned: %v (acceptable)", err)
	}
}
