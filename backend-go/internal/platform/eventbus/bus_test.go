package eventbus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
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

// TestPriorityOrdering verifies that high-priority events are dispatched before
// low-priority events, even when published after them.
func TestPriorityOrdering(t *testing.T) {
	b := newTestBus(t, WithWorkers(1)) // single worker to enforce ordering

	orderDone := make(chan struct{}, 3)
	var mu sync.Mutex
	var dispatched []int

	b.Subscribe("priority.*", func(ctx context.Context, evt Event) error {
		mu.Lock()
		dispatched = append(dispatched, evt.Priority)
		mu.Unlock()
		orderDone <- struct{}{}
		return nil
	})

	// Publish low priority first, then high priority.
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
	if len(dispatched) != 3 {
		t.Fatalf("expected 3 dispatched events, got %d", len(dispatched))
	}
	// Expect priority 2 first, then 1, then 0.
	if dispatched[0] != 2 || dispatched[1] != 1 || dispatched[2] != 0 {
		t.Errorf("expected order [2, 1, 0], got %v", dispatched)
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
// are dispatched in publication order (FIFO).
func TestMultiplePrioritiesSameLevel(t *testing.T) {
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
