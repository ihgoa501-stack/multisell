package approval

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ExecutionProcessing = "processing"
	ExecutionSucceeded  = "succeeded"
	ExecutionFailed     = "failed"
)

var (
	ErrExecutionUnauthorized = errors.New("approval execution is not authorized")
	ErrExecutionInProgress   = errors.New("approval execution is already processing")
	ErrApprovalConsumed      = errors.New("approval was already consumed")
)

// ApprovalExecution binds one approval to one logical production side effect.
// ApprovalID is the primary key so a different idempotency key can never reuse
// the same Owner decision.
type ApprovalExecution struct {
	ApprovalID     int64      `gorm:"column:approval_id;primaryKey" json:"approval_id"`
	IdempotencyKey string     `gorm:"column:idempotency_key;size:255;not null;uniqueIndex" json:"idempotency_key"`
	ActionType     string     `gorm:"column:action_type;size:100;not null" json:"action_type"`
	TargetType     string     `gorm:"column:target_type;size:100;not null" json:"target_type"`
	TargetID       string     `gorm:"column:target_id;size:255;not null" json:"target_id"`
	State          string     `gorm:"column:state;size:20;not null;index" json:"state"`
	ErrorMessage   string     `gorm:"column:error_message;type:text;not null;default:''" json:"error_message,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
}

func (ApprovalExecution) TableName() string { return "approval_execution" }

func approvalBindingMatches(req *ApprovalRequest, actionType, targetType, targetID string) bool {
	if req.Status != StatusApproved || (req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now())) {
		return false
	}
	if !RequestTypeCoversAction(req.RequestType, actionType) {
		return false
	}
	if req.TargetType != "" && req.TargetType != targetType {
		return false
	}
	if req.TargetID == 0 && req.EntityID == 0 && req.ProductID == 0 {
		return true
	}
	if strings.Contains(targetID, "=") {
		values := map[string]int64{}
		for _, part := range strings.Split(targetID, ";") {
			pair := strings.SplitN(part, "=", 2)
			if len(pair) != 2 {
				return false
			}
			value, err := strconv.ParseInt(pair[1], 10, 64)
			if err != nil || value <= 0 {
				return false
			}
			values[pair[0]] = value
		}
		return (req.ProductID == 0 || values["product"] == req.ProductID) &&
			(req.TargetID == 0 || values["target"] == req.TargetID) &&
			(req.EntityID == 0 || values["entity"] == req.EntityID)
	}
	id, err := strconv.ParseInt(targetID, 10, 64)
	if err != nil || id <= 0 {
		return false
	}
	return (req.TargetID == 0 || req.TargetID == id) &&
		(req.EntityID == 0 || req.EntityID == id) &&
		(req.ProductID == 0 || req.ProductID == id)
}

func executionBindingMatches(ex *ApprovalExecution, actionType, targetType, targetID, key string) bool {
	return ex.IdempotencyKey == key && ex.ActionType == actionType && ex.TargetType == targetType && ex.TargetID == targetID
}

// AuthorizeExecution validates both the Owner approval and any prior execution
// binding. A succeeded binding is authorized only so the caller can replay its
// durable result; ConsumeExecution will never execute it again.
func (s *Service) AuthorizeExecution(ctx context.Context, approvalID int64, actionType, targetType, targetID, key string) error {
	if approvalID <= 0 || actionType == "" || key == "" {
		return ErrExecutionUnauthorized
	}
	var req ApprovalRequest
	if err := s.db.WithContext(ctx).First(&req, approvalID).Error; err != nil || !approvalBindingMatches(&req, actionType, targetType, targetID) {
		return ErrExecutionUnauthorized
	}
	var ex ApprovalExecution
	err := s.db.WithContext(ctx).First(&ex, "approval_id = ?", approvalID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !executionBindingMatches(&ex, actionType, targetType, targetID, key) {
		return ErrApprovalConsumed
	}
	return nil
}

// ConsumeExecution atomically claims an approval for one logical action.
func (s *Service) ConsumeExecution(ctx context.Context, approvalID int64, actionType, targetType, targetID, key string) error {
	if err := s.AuthorizeExecution(ctx, approvalID, actionType, targetType, targetID, key); err != nil {
		return err
	}
	ex := ApprovalExecution{ApprovalID: approvalID, IdempotencyKey: key, ActionType: actionType, TargetType: targetType, TargetID: targetID, State: ExecutionProcessing}
	created := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&ex)
	if created.Error != nil {
		return created.Error
	}
	if created.RowsAffected == 1 {
		return nil
	}
	var existing ApprovalExecution
	if err := s.db.WithContext(ctx).First(&existing, "approval_id = ?", approvalID).Error; err != nil {
		return err
	}
	if !executionBindingMatches(&existing, actionType, targetType, targetID, key) {
		return ErrApprovalConsumed
	}
	switch existing.State {
	case ExecutionFailed:
		updated := s.db.WithContext(ctx).Model(&ApprovalExecution{}).
			Where("approval_id = ? AND idempotency_key = ? AND state = ?", approvalID, key, ExecutionFailed).
			Updates(map[string]interface{}{"state": ExecutionProcessing, "error_message": "", "completed_at": nil})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 1 {
			return nil
		}
		return ErrExecutionInProgress
	case ExecutionProcessing:
		return ErrExecutionInProgress
	default:
		return ErrApprovalConsumed
	}
}

func (s *Service) CompleteExecution(ctx context.Context, approvalID int64, key string) error {
	now := time.Now()
	updated := s.db.WithContext(ctx).Model(&ApprovalExecution{}).
		Where("approval_id = ? AND idempotency_key = ? AND state = ?", approvalID, key, ExecutionProcessing).
		Updates(map[string]interface{}{"state": ExecutionSucceeded, "completed_at": &now, "error_message": ""})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return fmt.Errorf("complete approval execution: %w", ErrApprovalConsumed)
	}
	return nil
}

func (s *Service) FailExecution(ctx context.Context, approvalID int64, key string, cause error) error {
	message := "execution failed"
	if cause != nil {
		message = cause.Error()
	}
	now := time.Now()
	updated := s.db.WithContext(ctx).Model(&ApprovalExecution{}).
		Where("approval_id = ? AND idempotency_key = ? AND state = ?", approvalID, key, ExecutionProcessing).
		Updates(map[string]interface{}{"state": ExecutionFailed, "completed_at": &now, "error_message": message})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return fmt.Errorf("fail approval execution: %w", ErrApprovalConsumed)
	}
	return nil
}
