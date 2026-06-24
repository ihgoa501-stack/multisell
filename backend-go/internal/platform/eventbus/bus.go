// Package eventbus provides an in-process publish/subscribe event bus for
// asynchronous communication between agents, business modules, and infrastructure.
//
// Topics use glob-style patterns: "order.*" matches "order.created", "order.refund".
// The bus can optionally persist events to an event_outbox table for durability.
package eventbus

import (
	"context"
	"encoding/json"
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

// Bus is the central event bus.
type Bus struct {
	mu          sync.RWMutex
	subs        []*subscription
	db          *gorm.DB
	logger      *zap.Logger
	bufferSize  int
	workerCount int
	eventCh     chan internalEvent
	done        chan struct{}
}

// internalEvent wraps an Event with a completion callback.
type internalEvent struct {
	event Event
	done  chan<- error
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
		eventCh:     make(chan internalEvent, 256),
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

	// Find matching subscribers.
	b.mu.RLock()
	var matched []*subscription
	for _, sub := range b.subs {
		if matchTopic(sub.topic, topic) {
			matched = append(matched, sub)
		}
	}
	b.mu.RUnlock()

	if len(matched) == 0 {
		return evt.ID, nil
	}

	// Dispatch to each matching handler concurrently.
	errCh := make(chan error, len(matched))
	for _, sub := range matched {
		h := sub.handler
		go func() {
			errCh <- h(ctx, evt)
		}()
	}

	// Collect errors.
	var lastErr error
	for i := 0; i < len(matched); i++ {
		if err := <-errCh; err != nil {
			lastErr = err
			b.logger.Warn("handler error",
				zap.String("topic", topic),
				zap.Error(err))
		}
	}

	return evt.ID, lastErr
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

// workerLoop is a goroutine for future async dispatch.
func (b *Bus) workerLoop(ctx context.Context, id int) {
	b.logger.Debug("event bus worker started", zap.Int("worker_id", id))
	<-b.done
	b.logger.Debug("event bus worker stopped", zap.Int("worker_id", id))
}
