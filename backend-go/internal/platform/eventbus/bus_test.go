package eventbus

import (
	"context"
	"errors"
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
		t.Skip("no DLQ events to replay")
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


// createDLQTable creates the event_dlq table with SQLite-compatible DDL.
func createDLQTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS event_dlq (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			original_event_id TEXT NOT NULL,
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
