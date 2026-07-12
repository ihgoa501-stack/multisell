package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type RetryRecord struct {
	ID            string    `gorm:"column:id;primaryKey;size:36"`
	TaskID        string    `gorm:"column:task_id;size:100;not null;index"`
	AgentID       string    `gorm:"column:agent_id;size:100;not null"`
	DecisionPoint string    `gorm:"column:decision_point;size:100;not null"`
	FailedAt      time.Time `gorm:"column:failed_at;not null"`
	LastError     string    `gorm:"column:last_error;type:text;not null"`
	Attempts      int       `gorm:"column:attempts;not null"`
	PayloadJSON   string    `gorm:"column:payload_json;type:text;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (RetryRecord) TableName() string { return "scheduler_retry" }

type RetryStore interface {
	Save(context.Context, RetryEntry) error
	List(context.Context) ([]RetryEntry, error)
	Update(context.Context, RetryEntry) error
	Delete(context.Context, string) error
}

type GormRetryStore struct{ db *gorm.DB }

func NewGormRetryStore(db *gorm.DB) *GormRetryStore { return &GormRetryStore{db: db} }

func retryRecord(entry RetryEntry) (RetryRecord, error) {
	raw, err := json.Marshal(entry.Payload)
	if err != nil {
		return RetryRecord{}, err
	}
	return RetryRecord{ID: entry.ID, TaskID: entry.TaskID, AgentID: entry.AgentID, DecisionPoint: entry.DecisionPoint, FailedAt: entry.FailedAt, LastError: entry.LastError, Attempts: entry.Attempts, PayloadJSON: string(raw)}, nil
}

func (s *GormRetryStore) Save(ctx context.Context, entry RetryEntry) error {
	record, err := retryRecord(entry)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(&record).Error
}

func (s *GormRetryStore) List(ctx context.Context) ([]RetryEntry, error) {
	var records []RetryRecord
	if err := s.db.WithContext(ctx).Order("failed_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	entries := make([]RetryEntry, 0, len(records))
	for _, record := range records {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(record.PayloadJSON), &payload); err != nil {
			return nil, err
		}
		entries = append(entries, RetryEntry{ID: record.ID, TaskID: record.TaskID, AgentID: record.AgentID, DecisionPoint: record.DecisionPoint, FailedAt: record.FailedAt, LastError: record.LastError, Attempts: record.Attempts, Payload: payload})
	}
	return entries, nil
}

func (s *GormRetryStore) Update(ctx context.Context, entry RetryEntry) error {
	record, err := retryRecord(entry)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&RetryRecord{}).Where("id = ?", entry.ID).Updates(map[string]interface{}{"attempts": record.Attempts, "last_error": record.LastError, "payload_json": record.PayloadJSON}).Error
}

func (s *GormRetryStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&RetryRecord{}, "id = ?", id).Error
}
