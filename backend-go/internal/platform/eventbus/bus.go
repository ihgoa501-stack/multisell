// Package eventbus provides an in-process publish/subscribe event bus for
// asynchronous communication between agents, business modules, and infrastructure.
//
// Topics use glob-style patterns: "order.*" matches "order.created", "order.refund".
// The bus can optionally persist events to an event_outbox table for durability.
package eventbus

import (
	"container/heap"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Event represents a message on the bus.
type Event struct {
	ID        string                 `json:"id"`
	Topic     string                 `json:"topic"`
	Source    string                 `json:"source"`
	Payload   map[string]interface{} `json:"payload"`
	Priority  int                    `json:"priority"`   // 0=normal, 1=high, 2=critical
	CreatedAt time.Time              `json:"created_at"`
}

// Handler processes a single event. Return an error to signal delivery failure.
type Handler func(ctx context.Context, event Event) error

// subscription binds a handler to a topic pattern.
type subscription struct {
	id      string
	topic   string
	handler Handler
}

// ErrQueueFull is returned when the event queue is at capacity and a
// backpressure timeout elapses before the event can be enqueued.
var ErrQueueFull = errors.New("eventbus: queue is full")

// priorityQueueItem is an item in the priority queue.
type priorityQueueItem struct {
	event Event
	index int // index in the heap
}

// priorityQueue implements heap.Interface. Higher-priority events are
// dequeued first; events at the same priority are ordered FIFO.
type priorityQueue []*priorityQueueItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Higher numeric priority first; ties broken by creation time.
	if pq[i].event.Priority != pq[j].event.Priority {
		return pq[i].event.Priority > pq[j].event.Priority
	}
	return pq[i].event.CreatedAt.Before(pq[j].event.CreatedAt)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*priorityQueueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// Bus is the central event bus.
type Bus struct {
	mu          sync.RWMutex
	subs        []*subscription
	queueMu     sync.Mutex
	queue       priorityQueue
	queueCond   *sync.Cond
	db          *gorm.DB
	logger      *zap.Logger
	bufferSize  int
	workerCount int
	done        chan struct{}
}

// BusOption configures the event bus.
type BusOption func(*Bus)

// WithDB enables outbox persistence via the given DB connection.
func WithDB(db *gorm.DB) BusOption {
	return func(b *Bus) { b.db = db }
}

// WithBufferSize sets the channel buffer size (default 256).
func WithBufferSize(n int) BusOption {
	return func(b *Bus) { b.bufferSize = n }
}

// WithWorkers sets the number of concurrent event dispatchers (default 4).
func WithWorkers(n int) BusOption {
	return func(b *Bus) { b.workerCount = n }
}

// New creates a new event bus.
func New(logger *zap.Logger, opts ...BusOption) *Bus {
	b := &Bus{
		logger:      logger,
		bufferSize:  256,
		workerCount: 4,
		queue:       make(priorityQueue, 0),
		done:        make(chan struct{}),
	}
	b.queueCond = sync.NewCond(&b.queueMu)
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Start launches the worker goroutines that dispatch events to handlers.
func (b *Bus) Start(ctx context.Context) {
	for i := 0; i < b.workerCount; i++ {
		go b.workerLoop(ctx, i)
	}
	b.logger.Info("event bus started",
		zap.Int("workers", b.workerCount),
		zap.Int("buffer", b.bufferSize))
}

// Stop shuts down the event bus gracefully.
func (b *Bus) Stop() {
	close(b.done)
	b.queueCond.Broadcast() // wake all workers so they see done
	b.logger.Info("event bus stopped")
}

// Publish delivers an event to all matching subscribers.
// Returns the event ID and any handler error.
func (b *Bus) Publish(ctx context.Context, topic string, source string, payload map[string]interface{}) (string, error) {
	return b.PublishWithPriority(ctx, topic, source, payload, 0)
}

// PublishWithPriority delivers an event with a given priority level.
func (b *Bus) PublishWithPriority(ctx context.Context, topic, source string, payload map[string]interface{}, priority int) (string, error) {
	evt := Event{
		ID:        uuid.New().String(),
		Topic:     topic,
		Source:    source,
		Payload:   payload,
		Priority:  priority,
		CreatedAt: time.Now(),
	}

	// Persist to outbox if DB is configured.
	if b.db != nil {
		if err := b.persistOutbox(ctx, &evt); err != nil {
			b.logger.Warn("failed to persist event to outbox",
				zap.String("topic", topic),
				zap.Error(err))
		}
	}

	item := &priorityQueueItem{event: evt}

	// Enqueue to the priority queue with backpressure.
	// If the queue is at capacity the event is dropped immediately and
	// ErrQueueFull is returned to the caller.
	b.queueMu.Lock()
	if b.bufferSize > 0 && b.queue.Len() >= b.bufferSize {
		b.queueMu.Unlock()
		return "", ErrQueueFull
	}
	heap.Push(&b.queue, item)
	b.queueCond.Signal()
	b.queueMu.Unlock()

	return evt.ID, nil
}

// Subscribe registers a handler for a topic pattern.
// Returns a subscription ID that can be used to unsubscribe.
func (b *Bus) Subscribe(topic string, handler Handler) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := &subscription{
		id:      uuid.New().String(),
		topic:   topic,
		handler: handler,
	}
	b.subs = append(b.subs, sub)

	b.logger.Debug("subscription added",
		zap.String("subscription_id", sub.id),
		zap.String("topic", topic))
	return sub.id
}

// Unsubscribe removes a subscription by ID.
func (b *Bus) Unsubscribe(subscriptionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.subs {
		if sub.id == subscriptionID {
			b.subs = append(b.subs[:i], b.subs[i+1:]...)
			b.logger.Debug("subscription removed",
				zap.String("subscription_id", subscriptionID))
			return
		}
	}
}

// Close gracefully shuts down the bus.
func (b *Bus) Close() error {
	b.Stop()
	return nil
}

// workerLoop pops events from the priority queue and dispatches them to
// matching subscribers in FIFO order per priority level.
func (b *Bus) workerLoop(ctx context.Context, id int) {
	b.logger.Debug("event bus worker started", zap.Int("worker_id", id))
	for {
		// Check for shutdown before blocking.
		select {
		case <-ctx.Done():
			b.logger.Debug("event bus worker stopped (context cancelled)", zap.Int("worker_id", id))
			return
		case <-b.done:
			b.logger.Debug("event bus worker stopped", zap.Int("worker_id", id))
			return
		default:
		}

		// Wait for an item on the priority queue.
		b.queueMu.Lock()
		for b.queue.Len() == 0 {
			b.queueCond.Wait()
			// Re-check shutdown signals after Cond.Wait returns (woken by
			// Broadcast on Stop or Signal from a new enqueue).
			select {
			case <-b.done:
				b.queueMu.Unlock()
				b.logger.Debug("event bus worker stopped", zap.Int("worker_id", id))
				return
			case <-ctx.Done():
				b.queueMu.Unlock()
				b.logger.Debug("event bus worker stopped (context cancelled)", zap.Int("worker_id", id))
				return
			default:
			}
		}
		item := heap.Pop(&b.queue).(*priorityQueueItem)
		b.queueMu.Unlock()

		// Dispatch to all matching subscribers sequentially within this worker.
		evt := item.event
		b.mu.RLock()
		for _, sub := range b.subs {
			if matchTopic(sub.topic, evt.Topic) {
				if err := sub.handler(ctx, evt); err != nil {
					b.logger.Warn("handler error",
						zap.String("event_id", evt.ID),
						zap.String("topic", evt.Topic),
						zap.String("subscription_id", sub.id),
						zap.Error(err))
				}
			}
		}
		b.mu.RUnlock()
	}
}

// matchTopic checks whether a subscription pattern matches an event topic.
// - "order.*" matches "order.created", "order.refund"
// - "order.**" matches "order.created.something"
// - "*" matches everything
func matchTopic(pattern, topic string) bool {
	if pattern == "*" || pattern == topic {
		return true
	}

	parts := strings.Split(pattern, ".")
	subj := strings.Split(topic, ".")

	for i, p := range parts {
		if p == "**" {
			return true
		}
		if p == "*" {
			continue
		}
		if i >= len(subj) || subj[i] != p {
			return false
		}
	}

	return len(parts) == len(subj)
}

// persistOutbox writes the event to the event_outbox table for durability.
func (b *Bus) persistOutbox(ctx context.Context, evt *Event) error {
	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		return err
	}
	return b.db.WithContext(ctx).Exec(
		`INSERT INTO event_outbox (topic, source, payload, priority, status, created_at)
		 VALUES (?, ?, ?, ?, 'delivered', ?)`,
		evt.Topic, evt.Source, payloadBytes, evt.Priority, evt.CreatedAt,
	).Error
}
