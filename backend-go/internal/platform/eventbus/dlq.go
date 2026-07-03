package eventbus

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DLEvent represents an event that has been moved to the dead-letter queue.
type DLEvent struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	OriginalEventID  string     `json:"original_event_id" gorm:"column:original_event_id;type:varchar(36);not null"`
	IdempotencyKey   string     `json:"idempotency_key" gorm:"column:idempotency_key;type:varchar(255);default:''"`
	Topic            string     `json:"topic" gorm:"column:topic;type:varchar(100);not null"`
	Source           string     `json:"source" gorm:"column:source;type:varchar(50);not null"`
	Payload          string     `json:"payload" gorm:"column:payload;type:jsonb;not null;default:'{}'"`
	Priority         int        `json:"priority" gorm:"column:priority;not null;default:0"`
	CorrelationID    string     `json:"correlation_id" gorm:"column:correlation_id;type:varchar(36);default:''"`
	ErrorMessage     string     `json:"error_message" gorm:"column:error_message;type:text;default:''"`
	DeliveryAttempts int        `json:"delivery_attempts" gorm:"column:delivery_attempts;not null;default:0"`
	LastAttemptAt    time.Time  `json:"last_attempt_at" gorm:"column:last_attempt_at;type:timestamptz"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at;type:timestamptz;not null;default:now()"`
	ReplayedAt       *time.Time `json:"replayed_at" gorm:"column:replayed_at;type:timestamptz"`
	ReplayedBy       string     `json:"replayed_by" gorm:"column:replayed_by;type:varchar(100);default:''"`
}

// TableName overrides the default table name for DLEvent.
func (DLEvent) TableName() string {
	return "event_dlq"
}

// DLQManager handles dead-letter events.
type DLQManager struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewDLQManager creates a new DLQManager.
func NewDLQManager(db *gorm.DB, logger *zap.Logger) *DLQManager {
	return &DLQManager{db: db, logger: logger}
}

// MoveToDLQ persists a failed event to the event_dlq table.
func (m *DLQManager) MoveToDLQ(evt Event, lastError string, attempts int) {
	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		m.logger.Warn("DLQ: failed to marshal payload", zap.Error(err))
		payloadBytes = []byte("{}")
	}

	dlEvent := DLEvent{
		OriginalEventID:  evt.ID,
		IdempotencyKey:   evt.IdempotencyKey,
		Topic:            evt.Topic,
		Source:           evt.Source,
		Payload:          string(payloadBytes),
		Priority:         evt.Priority,
		CorrelationID:    evt.CorrelationID,
		ErrorMessage:     lastError,
		DeliveryAttempts: attempts,
		LastAttemptAt:    time.Now(),
		CreatedAt:        time.Now(),
	}

	//nolint:errcheck
	m.db.Create(&dlEvent)
}

// ReplayEvents re-publishes DLQ events back to the bus using the given publish function.
// Returns the number of events successfully replayed.
func (m *DLQManager) ReplayEvents(ids []uint, publishFn func(Event) error) (int, error) {
	var events []DLEvent
	if err := m.db.Where("id IN ? AND replayed_at IS NULL", ids).Find(&events).Error; err != nil {
		return 0, err
	}

	replayed := 0
	for _, dl := range events {
		payload := make(map[string]interface{})
		if dl.Payload != "" && dl.Payload != "{}" {
			if err := json.Unmarshal([]byte(dl.Payload), &payload); err != nil {
				m.logger.Warn("DLQ: failed to unmarshal payload for replay",
					zap.Uint("dlq_id", dl.ID),
					zap.Error(err))
				continue
			}
		}

		evt := Event{
			ID:             dl.OriginalEventID,
			IdempotencyKey: dl.IdempotencyKey,
			Topic:          dl.Topic,
			Source:         dl.Source,
			Payload:        payload,
			Priority:       dl.Priority,
			CorrelationID:  dl.CorrelationID,
		}

		if err := publishFn(evt); err != nil {
			m.logger.Warn("DLQ: replay publish failed",
				zap.Uint("dlq_id", dl.ID),
				zap.Error(err))
			continue
		}

		now := time.Now()
		m.db.Model(&DLEvent{}).Where("id = ?", dl.ID).Updates(map[string]interface{}{
			"replayed_at": now,
			"replayed_by": "system",
		})
		replayed++
	}

	return replayed, nil
}

// ReplayEventsByIDs re-publishes DLQ events back to the bus identified by their IDs.
// Uses the Bus's Publish function to re-queue events.
func (m *DLQManager) ReplayEventsByIDs(bus *Bus, ids []uint) (int, error) {
	return m.ReplayEvents(ids, func(evt Event) error {
		ctx := context.Background()
		if evt.IdempotencyKey != "" {
			ctx = WithIdempotencyKey(ctx, evt.IdempotencyKey)
		}
		_, err := bus.PublishWithPriority(ctx, evt.Topic, evt.Source, evt.Payload, evt.Priority)
		return err
	})
}

// ListDLQ returns paginated DLQ events.
func (m *DLQManager) ListDLQ(page, size int) ([]DLEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	var total int64
	m.db.Model(&DLEvent{}).Count(&total)

	var events []DLEvent
	offset := (page - 1) * size
	if err := m.db.Order("id DESC").Offset(offset).Limit(size).Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}
