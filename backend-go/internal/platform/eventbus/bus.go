// Package eventbus provides an in-process publish/subscribe event bus for
// asynchronous communication between agents, business modules, and infrastructure.
//
// Topics use glob-style patterns: "order.*" matches "order.created", "order.refund".
// The bus can optionally persist events to an event_outbox table for durability.
//
// QoS isolation: events are routed to per-priority worker pools. Each priority
// level (0=normal, 1=high, 2=critical) has dedicated workers and buffer capacity,
// preventing a backlog at one priority from starving others.
package eventbus

import (
	"container/heap"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Event represents a message on the bus.
type Event struct {
	ID               string                 `json:"id"`
	Topic            string                 `json:"topic"`
	Version          string                 `json:"version"`
	Source           string                 `json:"source"`
	Actor            string                 `json:"actor"`
	EntityID         string                 `json:"entity_id"`
	EntityType       string                 `json:"entity_type"`
	Payload          map[string]interface{} `json:"payload"`
	Priority         int                    `json:"priority"`   // 0=normal, 1=high, 2=critical
	CreatedAt        time.Time              `json:"created_at"`
	CorrelationID    string                 `json:"correlation_id"`
	IdempotencyKey   string                 `json:"idempotency_key,omitempty"`
	DeliveryAttempts int                    `json:"delivery_attempts"`
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

// DefaultEventVersion is the default event schema version used when one is
// not explicitly set via context.
const DefaultEventVersion = "1.0"

const actorContextKey contextKey = "eventbus_actor"
const entityIDContextKey contextKey = "eventbus_entity_id"
const entityTypeContextKey contextKey = "eventbus_entity_type"
const eventVersionContextKey contextKey = "eventbus_version"
const idempotencyKeyContextKey contextKey = "eventbus_idempotency_key"

// WithActor attaches an actor identity to the context for event bus operations.
// The actor is propagated to published events for tracing and audit compliance.
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorContextKey, actor)
}

// ActorFromContext extracts the actor identity from the context.
// Returns empty string if not set.
func ActorFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(actorContextKey).(string); ok {
		return v
	}
	return ""
}

// WithEntityID attaches an entity ID to the context for event bus operations.
// The entity ID is propagated to published events for source identification.
func WithEntityID(ctx context.Context, entityID string) context.Context {
	return context.WithValue(ctx, entityIDContextKey, entityID)
}

// EntityIDFromContext extracts the entity ID from the context.
// Returns empty string if not set.
func EntityIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(entityIDContextKey).(string); ok {
		return v
	}
	return ""
}

// WithEntityType attaches an entity type to the context for event bus operations.
// The entity type is propagated to published events for source identification.
func WithEntityType(ctx context.Context, entityType string) context.Context {
	return context.WithValue(ctx, entityTypeContextKey, entityType)
}

// EntityTypeFromContext extracts the entity type from the context.
// Returns empty string if not set.
func EntityTypeFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(entityTypeContextKey).(string); ok {
		return v
	}
	return ""
}

// WithEventVersion attaches an event schema version to the context.
// Overrides DefaultEventVersion for the published event.
func WithEventVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, eventVersionContextKey, version)
}

// EventVersionFromContext extracts the event schema version from the context.
// Returns DefaultEventVersion if not set.
func EventVersionFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(eventVersionContextKey).(string); ok {
		return v
	}
	return DefaultEventVersion
}

// WithIdempotencyKey attaches an idempotency key to the context for event bus operations.
// The key is propagated to published events and used for deduplication at the bus level:
// duplicate events with the same key are skipped during worker dispatch.
//
// Choose keys that are unique per business-logical operation:
//
//	purchase order received → fmt.Sprintf("purchase_order_received:%s", orderNo)
//	aftersale processed     → fmt.Sprintf("aftersale_processed:%d", aftersaleID)
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKeyContextKey, key)
}

// IdempotencyKeyFromContext extracts the idempotency key from the context.
// Returns empty string if not set.
func IdempotencyKeyFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(idempotencyKeyContextKey).(string); ok {
		return v
	}
	return ""
}

// subscription binds a handler to a topic pattern.

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

// workerPool isolates events at a single priority level with dedicated
// workers and buffer capacity.
type workerPool struct {
	backend    Backend
	bufferSize int
	nWorkers   int
}

// DefaultMaxRetries is the default number of delivery attempts before an
// event is moved to the dead-letter queue.
const DefaultMaxRetries = 3

// Bus is the central event bus with per-priority QoS worker pools.
type Bus struct {
	mu             sync.RWMutex
	subs           []*subscription
	db             *gorm.DB
	logger         *zap.Logger
	done           chan struct{}
	schemaRegistry *SchemaRegistry
	dlq            *DLQManager
	maxRetries     int

	// Per-priority worker pools.
	pools  map[int]*workerPool
	poolWg sync.WaitGroup
}

// BusOption configures the event bus.
type BusOption func(*Bus)

// WithDB enables outbox persistence via the given DB connection.
func WithDB(db *gorm.DB) BusOption {
	return func(b *Bus) { b.db = db }
}

// WithBufferSize sets the channel buffer size for priority 0 (default 256).
func WithBufferSize(n int) BusOption {
	return func(b *Bus) {
		if pool, ok := b.pools[0]; ok {
			pool.bufferSize = n
		}
	}
}

// WithWorkers sets the number of workers for priority 0 (default 4).
// For per-priority worker configuration use WithWorkersPerPriority.
func WithWorkers(n int) BusOption {
	return func(b *Bus) {
		if pool, ok := b.pools[0]; ok {
			pool.nWorkers = n
		}
	}
}

// WithBackend sets a custom Backend implementation for priority 0.
func WithBackend(backend Backend) BusOption {
	return func(b *Bus) {
		if pool, ok := b.pools[0]; ok {
			pool.backend = backend
		}
	}
}

// WithDLQ enables the dead-letter queue using the given DB connection.
func WithDLQ(db *gorm.DB) BusOption {
	return func(b *Bus) {
		b.dlq = NewDLQManager(db, b.logger)
	}
}

// WithMaxRetries sets the maximum delivery attempts before DLQ (default 3).
func WithMaxRetries(n int) BusOption {
	return func(b *Bus) { b.maxRetries = n }
}

// WithWorkersPerPriority configures per-priority worker counts.
// The map key is priority, value is the number of workers.
// Example: map[int]int{0: 4, 1: 2, 2: 2}
func WithWorkersPerPriority(workers map[int]int) BusOption {
	return func(b *Bus) {
		for pri, n := range workers {
			if pool, ok := b.pools[pri]; ok {
				pool.nWorkers = n
			}
		}
	}
}

// New creates a new event bus with per-priority QoS worker pools.
//
// Default pool configuration:
//
//	priority 2 (critical): 2 workers, buffer 64
//	priority 1 (high):     2 workers, buffer 128
//	priority 0 (normal):   4 workers, buffer 256
func New(logger *zap.Logger, opts ...BusOption) *Bus {
	b := &Bus{
		logger:     logger,
		done:       make(chan struct{}),
		maxRetries: DefaultMaxRetries,
		pools: map[int]*workerPool{
			0: {backend: NewInProcessBackend(), bufferSize: 256, nWorkers: 4},
			1: {backend: NewInProcessBackend(), bufferSize: 128, nWorkers: 2},
			2: {backend: NewInProcessBackend(), bufferSize: 64, nWorkers: 2},
		},
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Start launches the worker goroutines that dispatch events to handlers.
func (b *Bus) Start(ctx context.Context) {
	for priority, pool := range b.pools {
		pool := pool
		for i := 0; i < pool.nWorkers; i++ {
			b.poolWg.Add(1)
			go b.workerLoop(ctx, priority, i)
		}
	}
	b.logger.Info("event bus started with QoS pools",
		zap.Int("pool[0].workers", b.pools[0].nWorkers),
		zap.Int("pool[0].buffer", b.pools[0].bufferSize),
		zap.Int("pool[1].workers", b.pools[1].nWorkers),
		zap.Int("pool[1].buffer", b.pools[1].bufferSize),
		zap.Int("pool[2].workers", b.pools[2].nWorkers),
		zap.Int("pool[2].buffer", b.pools[2].bufferSize))
}

// Stop shuts down the event bus gracefully.
func (b *Bus) Stop() {
	close(b.done)
	for _, pool := range b.pools {
		pool.backend.Close()
	}
	b.poolWg.Wait()
	b.logger.Info("event bus stopped")
}

// Publish delivers an event to all matching subscribers.
// Returns the event ID.
func (b *Bus) Publish(ctx context.Context, topic string, source string, payload map[string]interface{}) (string, error) {
	return b.PublishWithPriority(ctx, topic, source, payload, 0)
}

// PublishWithPriority delivers an event with a given priority level.
// The priority determines which worker pool handles the event:
// 0=normal, 1=high, 2=critical.
func (b *Bus) PublishWithPriority(ctx context.Context, topic, source string, payload map[string]interface{}, priority int) (string, error) {
	evt := Event{
		ID:             uuid.New().String(),
		Topic:          topic,
		Version:        EventVersionFromContext(ctx),
		Source:         source,
		Actor:          ActorFromContext(ctx),
		EntityID:       EntityIDFromContext(ctx),
		EntityType:     EntityTypeFromContext(ctx),
		Payload:        payload,
		Priority:       priority,
		CreatedAt:      time.Now(),
		CorrelationID:  CorrelationIDFromContext(ctx),
		IdempotencyKey: IdempotencyKeyFromContext(ctx),
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

	// Route to the appropriate pool based on priority.
	pool, ok := b.pools[priority]
	if !ok {
		pool = b.pools[0]
	}

	// Enqueue to the pool's backend with backpressure.
	if pool.bufferSize > 0 && pool.backend.Len() >= pool.bufferSize {
		return "", ErrQueueFull
	}

	if err := pool.backend.Enqueue(evt); err != nil {
		return "", err
	}

	eventsQueueDepthVec.WithLabelValues(strconv.Itoa(priority)).Set(float64(pool.backend.Len()))
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

// DLQ returns the dead-letter queue manager, or nil if not configured.
func (b *Bus) DLQ() *DLQManager {
	return b.dlq
}

// checkIdempotency returns true if the event should be skipped (duplicate).
// It is a read-only check: the event_processed row is written only after the
// handler succeeds (see markProcessed), so retries and DLQ replays for the same
// key pass through.
//
// - No row for this key yet → returns (false, nil), handler will run
// - Existing row has same event_id → returns (false, nil), retry passes through
// - Existing row has different event_id → returns (true, nil), duplicate skipped
func (b *Bus) checkIdempotency(ctx context.Context, evt Event) (bool, error) {
	var existingID string
	if err := b.db.WithContext(ctx).
		Raw(`SELECT COALESCE(event_id, '') FROM event_processed WHERE idempotency_key = ?`, evt.IdempotencyKey).
		Scan(&existingID).Error; err != nil {
		return false, err
	}
	if existingID == "" {
		return false, nil // first time seeing this key
	}
	return existingID != evt.ID, nil // skip if different event_id already processed successfully
}

// markProcessed records a successful handler completion for idempotency tracking.
// Called after all handlers succeed so that processed_at reflects actual completion
// time, not dispatch start. This allows failed events and DLQ replays with the
// same idempotency_key to reach handlers.
func (b *Bus) markProcessed(ctx context.Context, evt Event) {
	b.db.WithContext(ctx).Exec(
		`INSERT INTO event_processed (idempotency_key, topic, event_id, processed_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP) ON CONFLICT (idempotency_key) DO NOTHING`,
		evt.IdempotencyKey, evt.Topic, evt.ID,
	)
}

// workerLoop pops events from the given pool's backend and dispatches them to
// matching subscribers. Failed events are retried (up to maxRetries) and then
// moved to the dead-letter queue if configured.
func (b *Bus) workerLoop(ctx context.Context, poolID, workerID int) {
	pool, ok := b.pools[poolID]
	if !ok {
		return
	}

	b.logger.Debug("event bus worker started",
		zap.Int("pool_id", poolID),
		zap.Int("worker_id", workerID))
	defer b.poolWg.Done()

	for {
		// Check for shutdown before blocking.
		select {
		case <-ctx.Done():
			return
		case <-b.done:
			return
		default:
		}

		// Dequeue blocks until an event is available or the backend is closed.
		evt, ok := pool.backend.Dequeue()
		if !ok {
			return
		}

		// Propagate correlation ID to handler context.
	handlerCtx := WithCorrelationID(ctx, evt.CorrelationID)

		// Idempotency check: skip duplicate events with the same idempotency_key.
		// The row in event_processed is written only AFTER handler success (see
		// markProcessed), so retries (same event_id) and DLQ replays (same key)
		// always pass through to the handler.
		if evt.IdempotencyKey != "" && b.db != nil {
			skip, checkErr := b.checkIdempotency(handlerCtx, evt)
			if checkErr != nil {
				b.logger.Warn("idempotency check failed, letting event through",
					zap.String("event_id", evt.ID),
					zap.String("idempotency_key", evt.IdempotencyKey),
					zap.Error(checkErr))
				// fail open: let the event through on DB error
			} else if skip {
				b.logger.Info("skipping duplicate event",
					zap.String("event_id", evt.ID),
					zap.String("topic", evt.Topic),
					zap.String("idempotency_key", evt.IdempotencyKey))
				eventsSkipped.WithLabelValues(evt.Topic, "duplicate").Inc()
				continue
			}
		}

		// Dispatch to all matching subscribers with panic recovery per handler.
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
					if err := s.handler(handlerCtx, evt); err != nil {
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
					`UPDATE event_outbox SET status='failed', last_error=?, delivery_attempts=COALESCE(delivery_attempts,0)+1 WHERE event_id=? AND status='pending'`,
					lastErr, evt.ID,
				)
				eventsPublished.WithLabelValues(evt.Topic, "failed").Inc()
			} else {
				b.db.WithContext(ctx).Exec(
					`UPDATE event_outbox SET status='delivered', delivered_at=NOW(), delivery_attempts=COALESCE(delivery_attempts,0)+1 WHERE event_id=? AND status='pending'`,
					evt.ID,
				)
				eventsPublished.WithLabelValues(evt.Topic, "delivered").Inc()

				// Mark idempotency processed only after handler success.
				if evt.IdempotencyKey != "" {
					b.markProcessed(ctx, evt)
				}
			}
		}

		// Track delivery metrics.
		if !panicked && lastErr == "" {
			eventsDelivered.Inc()
		}

		// Retry or DLQ logic. If the event failed and DLQ is configured,
		// move to DLQ after max retries. Otherwise re-enqueue for retry.
		if panicked || lastErr != "" {
			evt.DeliveryAttempts++
			if evt.DeliveryAttempts >= b.maxRetries && b.dlq != nil {
				b.dlq.MoveToDLQ(evt, lastErr, evt.DeliveryAttempts)
				b.logger.Warn("event moved to DLQ after max retries",
					zap.String("event_id", evt.ID),
					zap.String("topic", evt.Topic),
					zap.Int("attempts", evt.DeliveryAttempts),
					zap.String("last_error", lastErr))
				eventsDLQTotal.WithLabelValues(evt.Topic).Inc()
			} else if evt.DeliveryAttempts < b.maxRetries {
				// Re-enqueue for retry.
				_ = pool.backend.Enqueue(evt)
				b.logger.Debug("re-enqueuing event for retry",
					zap.String("event_id", evt.ID),
					zap.String("topic", evt.Topic),
					zap.Int("attempt", evt.DeliveryAttempts))
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
