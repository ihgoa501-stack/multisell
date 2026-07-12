package toolbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	toolExecutionProcessing = "processing"
	toolExecutionSucceeded  = "succeeded"
	toolExecutionFailed     = "failed"
)

var ErrToolExecutionInProgress = errors.New("toolbridge: idempotent execution is already processing")
var ErrIdempotencyUnavailable = errors.New("toolbridge: production mutation requires durable idempotency")

// IdempotentToolDriver forwards the same key to the external provider. This is
// required for production mutations because a process can crash after the
// provider commits but before the local result is stored.
type IdempotentToolDriver interface {
	ExecuteIdempotent(ctx context.Context, input map[string]interface{}, idempotencyKey string) (*ToolResult, error)
}

type ToolExecution struct {
	IdempotencyKey string     `gorm:"column:idempotency_key;primaryKey;size:255"`
	CallHash       string     `gorm:"column:call_hash;size:64;not null"`
	ToolName       string     `gorm:"column:tool_name;size:100;not null"`
	TargetType     string     `gorm:"column:target_type;size:100;not null"`
	TargetID       string     `gorm:"column:target_id;size:255;not null"`
	State          string     `gorm:"column:state;size:20;not null;index"`
	ResultJSON     string     `gorm:"column:result_json;type:text;not null;default:''"`
	ErrorMessage   string     `gorm:"column:error_message;type:text;not null;default:''"`
	LeaseExpiresAt time.Time  `gorm:"column:lease_expires_at;not null"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
}

func (ToolExecution) TableName() string { return "tool_execution" }

type ToolExecutionClaim struct {
	Execute bool
	Result  *ToolResult
}

type ToolIdempotencyStore interface {
	Claim(ctx context.Context, call ToolCall) (*ToolExecutionClaim, error)
	Complete(ctx context.Context, call ToolCall, result *ToolResult) error
	Fail(ctx context.Context, call ToolCall, cause error) error
}

type GormToolIdempotencyStore struct {
	db    *gorm.DB
	lease time.Duration
	now   func() time.Time
}

func NewGormToolIdempotencyStore(db *gorm.DB, lease time.Duration) *GormToolIdempotencyStore {
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	return &GormToolIdempotencyStore{db: db, lease: lease, now: time.Now}
}

func hashToolCall(call ToolCall) (string, error) {
	// CorrelationID is observability metadata, not logical action identity.
	call.CorrelationID = ""
	raw, err := json.Marshal(call)
	if err != nil {
		return "", fmt.Errorf("toolbridge: encode call identity: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *GormToolIdempotencyStore) Claim(ctx context.Context, call ToolCall) (*ToolExecutionClaim, error) {
	hash, err := hashToolCall(call)
	if err != nil {
		return nil, err
	}
	now := s.now()
	row := ToolExecution{IdempotencyKey: call.IdempotencyKey, CallHash: hash, ToolName: call.ToolName, TargetType: call.TargetType, TargetID: call.TargetID, State: toolExecutionProcessing, LeaseExpiresAt: now.Add(s.lease)}
	created := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if created.Error != nil {
		return nil, created.Error
	}
	if created.RowsAffected == 1 {
		return &ToolExecutionClaim{Execute: true}, nil
	}
	var existing ToolExecution
	if err := s.db.WithContext(ctx).Where("idempotency_key = ?", call.IdempotencyKey).First(&existing).Error; err != nil {
		return nil, err
	}
	if existing.CallHash != hash {
		return nil, fmt.Errorf("toolbridge: idempotency key %q was already used by another call", call.IdempotencyKey)
	}
	if existing.State == toolExecutionSucceeded {
		var result ToolResult
		if err := json.Unmarshal([]byte(existing.ResultJSON), &result); err != nil {
			return nil, fmt.Errorf("toolbridge: decode idempotent result: %w", err)
		}
		return &ToolExecutionClaim{Result: &result}, nil
	}
	if existing.State == toolExecutionProcessing && existing.LeaseExpiresAt.After(now) {
		return nil, ErrToolExecutionInProgress
	}
	updated := s.db.WithContext(ctx).Model(&ToolExecution{}).
		Where("idempotency_key = ? AND call_hash = ? AND (state = ? OR (state = ? AND lease_expires_at <= ?))", call.IdempotencyKey, hash, toolExecutionFailed, toolExecutionProcessing, now).
		Updates(map[string]any{"state": toolExecutionProcessing, "lease_expires_at": now.Add(s.lease), "result_json": "", "error_message": "", "completed_at": nil})
	if updated.Error != nil {
		return nil, updated.Error
	}
	if updated.RowsAffected != 1 {
		return nil, ErrToolExecutionInProgress
	}
	return &ToolExecutionClaim{Execute: true}, nil
}

func (s *GormToolIdempotencyStore) Complete(ctx context.Context, call ToolCall, result *ToolResult) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	hash, err := hashToolCall(call)
	if err != nil {
		return err
	}
	now := s.now()
	updated := s.db.WithContext(ctx).Model(&ToolExecution{}).
		Where("idempotency_key = ? AND call_hash = ? AND state = ?", call.IdempotencyKey, hash, toolExecutionProcessing).
		Updates(map[string]any{"state": toolExecutionSucceeded, "result_json": string(raw), "completed_at": &now, "lease_expires_at": now})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return errors.New("toolbridge: idempotency completion lost its processing claim")
	}
	return nil
}

func (s *GormToolIdempotencyStore) Fail(ctx context.Context, call ToolCall, cause error) error {
	hash, err := hashToolCall(call)
	if err != nil {
		return err
	}
	message := "tool execution failed"
	if cause != nil {
		message = cause.Error()
	}
	now := s.now()
	return s.db.WithContext(ctx).Model(&ToolExecution{}).
		Where("idempotency_key = ? AND call_hash = ? AND state = ?", call.IdempotencyKey, hash, toolExecutionProcessing).
		Updates(map[string]any{"state": toolExecutionFailed, "error_message": message, "completed_at": &now, "lease_expires_at": now}).Error
}
