package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	idempotencyProcessing = "processing"
	idempotencySucceeded  = "succeeded"
	idempotencyFailed     = "failed"
)

var ErrIdempotencyInProgress = errors.New("command: idempotent action is already processing")

// ActionExecution is the durable claim and result for one logical command.
// The idempotency key is globally unique because callers must never reuse a
// key for a different logical side effect.
type ActionExecution struct {
	IdempotencyKey string     `gorm:"column:idempotency_key;primaryKey;size:255"`
	ActionType     string     `gorm:"column:action_type;size:100;not null"`
	AgentID        string     `gorm:"column:agent_id;size:100;not null"`
	State          string     `gorm:"column:state;size:20;not null;index"`
	ResultJSON     string     `gorm:"column:result_json;type:text;not null;default:''"`
	ErrorMessage   string     `gorm:"column:error_message;type:text;not null;default:''"`
	LeaseExpiresAt time.Time  `gorm:"column:lease_expires_at;not null"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
}

func (ActionExecution) TableName() string { return "command_execution" }

type IdempotencyClaim struct {
	Execute bool
	Result  *Result
}

type IdempotencyStore interface {
	Claim(ctx context.Context, action AgentAction) (*IdempotencyClaim, error)
	Complete(ctx context.Context, action AgentAction, result *Result) error
	Fail(ctx context.Context, action AgentAction, cause error) error
}

type GormIdempotencyStore struct {
	db    *gorm.DB
	lease time.Duration
	now   func() time.Time
}

func NewGormIdempotencyStore(db *gorm.DB, lease time.Duration) *GormIdempotencyStore {
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	return &GormIdempotencyStore{db: db, lease: lease, now: time.Now}
}

func (s *GormIdempotencyStore) Claim(ctx context.Context, action AgentAction) (*IdempotencyClaim, error) {
	now := s.now()
	row := ActionExecution{IdempotencyKey: action.IdempotencyKey, ActionType: action.ActionType, AgentID: action.AgentID, State: idempotencyProcessing, LeaseExpiresAt: now.Add(s.lease)}
	created := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if created.Error != nil {
		return nil, created.Error
	}
	if created.RowsAffected == 1 {
		return &IdempotencyClaim{Execute: true}, nil
	}

	var existing ActionExecution
	if err := s.db.WithContext(ctx).Where("idempotency_key = ?", action.IdempotencyKey).First(&existing).Error; err != nil {
		return nil, err
	}
	if existing.ActionType != action.ActionType || existing.AgentID != action.AgentID {
		return nil, fmt.Errorf("command: idempotency key %q was already used by another action", action.IdempotencyKey)
	}
	if existing.State == idempotencySucceeded {
		var result Result
		if err := json.Unmarshal([]byte(existing.ResultJSON), &result); err != nil {
			return nil, fmt.Errorf("command: decode idempotent result: %w", err)
		}
		return &IdempotencyClaim{Result: &result}, nil
	}
	if existing.State == idempotencyProcessing && existing.LeaseExpiresAt.After(now) {
		return nil, ErrIdempotencyInProgress
	}

	updated := s.db.WithContext(ctx).Model(&ActionExecution{}).
		Where("idempotency_key = ? AND (state = ? OR (state = ? AND lease_expires_at <= ?))", action.IdempotencyKey, idempotencyFailed, idempotencyProcessing, now).
		Updates(map[string]any{"state": idempotencyProcessing, "lease_expires_at": now.Add(s.lease), "result_json": "", "error_message": "", "completed_at": nil})
	if updated.Error != nil {
		return nil, updated.Error
	}
	if updated.RowsAffected != 1 {
		return nil, ErrIdempotencyInProgress
	}
	return &IdempotencyClaim{Execute: true}, nil
}

func (s *GormIdempotencyStore) Complete(ctx context.Context, action AgentAction, result *Result) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	now := s.now()
	updated := s.db.WithContext(ctx).Model(&ActionExecution{}).
		Where("idempotency_key = ? AND action_type = ? AND agent_id = ? AND state = ?", action.IdempotencyKey, action.ActionType, action.AgentID, idempotencyProcessing).
		Updates(map[string]any{"state": idempotencySucceeded, "result_json": string(raw), "completed_at": &now, "lease_expires_at": now})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return errors.New("command: idempotency completion lost its processing claim")
	}
	return nil
}

func (s *GormIdempotencyStore) Fail(ctx context.Context, action AgentAction, cause error) error {
	message := "command failed"
	if cause != nil {
		message = cause.Error()
	}
	now := s.now()
	return s.db.WithContext(ctx).Model(&ActionExecution{}).
		Where("idempotency_key = ? AND action_type = ? AND agent_id = ? AND state = ?", action.IdempotencyKey, action.ActionType, action.AgentID, idempotencyProcessing).
		Updates(map[string]any{"state": idempotencyFailed, "error_message": message, "completed_at": &now, "lease_expires_at": now}).Error
}
