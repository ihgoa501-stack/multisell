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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Event represents a message on the bus.
type Event struct {
	ID            string                 `json:"id"`
	Topic         string                 `json:"topic"`
	Source        string                 `json:"source"`
	Payload       map[string]interface{} `json:"payload"`
	Priority      int                    `json:"priority"`   // 0=normal, 1=high, 2=critical
	CreatedAt     time.Time              `json:"created_at"`
	CorrelationID string                 `json:"correlation_id"`
}

// Handler processes a single event. Return an error to signal delivery failure.
type Handler func(ctx context.Context, event Event) error

// contextKey is used for context-scoped values to avoid collisions.
type contextKey string

const correlationContextKey contextKey = "eventbus_correlation_id"

// WithCorrelationID attaches a correlation ID to the context for event bus
// operations. The correlation ID is propagated to published events and can
// be used for distributed tracing across event handlers.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationContextKey, correlationID)
}

// CorrelationIDFromContext extracts the correlation ID from the context.
// Returns empty string if not set.
func CorrelationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(correlationContextKey).(string); ok {
		return v
	}
	return ""
}

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

// Backend is the pluggable storage backend for the event bus.
type Backend interface {
	Enqueue(event Event) error
	Dequeue() (Event, bool)
	Len() int
	Close()
}

// InProcessBackend is the default in-process priority queue backend. It
// implements the Backend interface using a heap-based priority queue with
// sync.Cond for blocking dequeue operations.
type InProcessBackend struct {
	mu     sync.Mutex
	cond   *sync.Cond
	queue  priorityQueue
	closed bool
}

// NewInProcessBackend creates a new InProcessBackend.
func NewInProcessBackend() *InProcessBackend {
	b := &InProcessBackend{
		queue: make(priorityQueue, 0),
	}
	b.cond = sync.NewCond(&b.mu)
	return b
}

// Enqueue adds an event to the priority queue and signals a waiting dequeue.
func (b *InProcessBackend) Enqueue(event Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	item := &priorityQueueItem{event: event}
	heap.Push(&b.queue, item)
	b.cond.Signal()
	return nil
}

// Dequeue removes and returns the highest-priority event, blocking until one
// is available. Returns false if the backend has been closed.
func (b *InProcessBackend) Dequeue() (Event, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for b.queue.Len() == 0 {
		if b.closed {
			return Event{}, false
		}
		b.cond.Wait()
	}
	item := heap.Pop(&b.queue).(*priorityQueueItem)
	return item.event, true
}

// Len returns the number of events currently in the queue.
func (b *InProcessBackend) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.queue.Len()
}

// Close marks the backend as closed and wakes all waiting goroutines.
func (b *InProcessBackend) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.cond.Broadcast()
}

// Bus is the central event bus.
type Bus struct {
	mu             sync.RWMutex
	subs           []*subscription
	backend        Backend
	db             *gorm.DB
	logger         *zap.Logger
	bufferSize     int
	workerCount    int
	done           chan struct{}
	schemaRegistry *SchemaRegistry
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

// WithBackend sets a custom Backend implementation.
func WithBackend(b Backend) BusOption {
	return func(bus *Bus) { bus.backend = b }
}

// New creates a new event bus.
func New(logger *zap.Logger, opts ...BusOption) *Bus {
	b := &Bus{
		logger:      logger,
		bufferSize:  256,
		workerCount: 4,
		backend:     NewInProcessBackend(),
		done:        make(chan struct{}),
	}
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
	b.backend.Close()
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
		ID:            uuid.New().String(),
		Topic:         topic,
		Source:        source,
		Payload:       payload,
		Priority:      priority,
		CreatedAt:     time.Now(),
		CorrelationID: CorrelationIDFromContext(ctx),
	}

	// Validate against schema registry if configured.
	if b.schemaRegistry != nil {
		if schema, ok := b.schemaRegistry.Schema(topic); ok {
			if err := schema.Validate(payload); err != nil {
				return "", err
			}
		}
	}

	// Persist to outbox if DB is configured.
	if b.db != nil {
		if err := b.persistOutbox(ctx, &evt); err != nil {
			b.logger.Warn("failed to persist event to outbox",
				zap.String("topic", topic),
				zap.Error(err))
		}
	}

	// Enqueue to the backend with backpressure.
	// If the queue is at capacity the event is dropped immediately and
	// ErrQueueFull is returned to the caller.
	if b.bufferSize > 0 && b.backend.Len() >= b.bufferSize {
		return "", ErrQueueFull
	}
		heap.Push(&b.queue, item)
		eventsQueueDepth.Set(float64(b.queue.Len()))
		b.queueCond.Signal()
		b.queueMu.Unlock()

	eventsPublished.WithLabelValues(topic, "published").Inc()

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

		// Dequeue blocks until an event is available or the backend is closed.
		evt, ok := b.backend.Dequeue()
		if !ok {
			b.logger.Debug("event bus worker stopped", zap.Int("worker_id", id))
			return
		}
		item := heap.Pop(&b.queue).(*priorityQueueItem)
		eventsQueueDepth.Set(float64(b.queue.Len()))
		b.queueMu.Unlock()

		// Dispatch to all matching subscribers with panic recovery per handler.
		evt := item.event
		var panicked bool
		var lastErr string
		b.mu.RLock()
		for _, sub := range b.subs {
			if matchTopic(sub.topic, evt.Topic) {
				func(s *subscription) {
					defer func() {
						if r := recover(); r != nil {
							panicked = true
							lastErr = fmt.Sprintf("%v", r)
							b.logger.Error("handler panic recovered",
								zap.String("event_id", evt.ID),
								zap.String("topic", evt.Topic),
								zap.String("subscription_id", s.id),
								zap.Any("panic", r))
							eventsHandlerErrors.WithLabelValues(evt.Topic).Inc()
						}
					}()
					if err := s.handler(ctx, evt); err != nil {
						lastErr = err.Error()
						b.logger.Warn("handler error",
							zap.String("event_id", evt.ID),
							zap.String("topic", evt.Topic),
							zap.String("subscription_id", s.id),
							zap.Error(err))
						eventsHandlerErrors.WithLabelValues(evt.Topic).Inc()
					}
				}(sub)
			}
		}
		b.mu.RUnlock()

		// Update outbox status after all handlers have run.
		if b.db != nil {
			if panicked || lastErr != "" {
				b.db.WithContext(ctx).Exec(
					`UPDATE event_outbox SET status='failed', last_error=?, delivery_attempts=delivery_attempts+1 WHERE event_id=? AND status='pending'`,
					lastErr, evt.ID,
				)
				eventsPublished.WithLabelValues(evt.Topic, "failed").Inc()
			} else {
				b.db.WithContext(ctx).Exec(
					`UPDATE event_outbox SET status='delivered', delivered_at=NOW(), delivery_attempts=delivery_attempts+1 WHERE event_id=? AND status='pending'`,
					evt.ID,
				)
				eventsPublished.WithLabelValues(evt.Topic, "delivered").Inc()
			}
		}
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

// persistOutbox writes the event to the event_outbox table with 'pending'
// status. The workerLoop updates the status to 'delivered' or 'failed' after
// all handlers have run.
func (b *Bus) persistOutbox(ctx context.Context, evt *Event) error {
	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		return err
	}
	return b.db.WithContext(ctx).Exec(
		`INSERT INTO event_outbox (topic, source, payload, priority, status, created_at, event_id)
		 VALUES (?, ?, ?, ?, 'pending', ?, ?)`,
		evt.Topic, evt.Source, payloadBytes, evt.Priority, evt.CreatedAt, evt.ID,
	).Error
}
