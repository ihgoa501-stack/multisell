package approval

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides CRUD operations and auto-escalation for approval requests.
type Service struct {
	db       *gorm.DB
	logger   *zap.Logger
	oplogSvc *operationlog.Service // optional, may be nil
	bus      *eventbus.Bus         // optional, may be nil — used to publish approval lifecycle events
}

// NewService creates a new approval service.
// oplogSvc may be nil (audit logging disabled).
func NewService(db *gorm.DB, logger *zap.Logger, oplogSvc *operationlog.Service) *Service {
	return &Service{db: db, logger: logger, oplogSvc: oplogSvc}
}

// WithBus attaches an event bus for publishing approval lifecycle events.
func (s *Service) WithBus(bus *eventbus.Bus) *Service {
	s.bus = bus
	return s
}

// RequireApproval creates a pending approval request for a high-risk mutation
// and returns the created request. The caller should NOT proceed with the
// mutation — a human must review the request first.
func (s *Service) RequireApproval(input *CreateApprovalInput) (*ApprovalRequest, error) {
	if input.Requester == "" {
		input.Requester = "system"
	}
	req, err := s.Create(input)
	if err != nil {
		return nil, fmt.Errorf("approval required: %w", err)
	}
	return req, nil
}

// List returns paginated approval requests with optional filters.
func (s *Service) List(page, size int, status, requestType string) ([]ApprovalRequest, int64, error) {
	q := s.db.Model(&ApprovalRequest{})

	if status != "" {
		q = q.Where("status = ?", status)
	}
	if requestType != "" {
		q = q.Where("request_type = ?", requestType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []ApprovalRequest
	offset := (page - 1) * size
	if err := q.Order("created_at DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []ApprovalRequest{}
	}
	return items, total, nil
}

// Get returns a single approval request by ID.
func (s *Service) Get(id int64) (*ApprovalRequest, error) {
	var req ApprovalRequest
	if err := s.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

// Create inserts a new approval request.
func (s *Service) Create(input *CreateApprovalInput) (*ApprovalRequest, error) {
	req := &ApprovalRequest{
		ProductID:       input.ProductID,
		RequestType:     input.RequestType,
		Requester:       input.Requester,
		RequesterUserID: input.RequesterUserID,
		Status:          StatusPending,
		OldValue:        input.OldValue,
		NewValue:        input.NewValue,
		Reason:          input.Reason,
		TargetType:      input.TargetType,
		TargetID:        input.TargetID,
		RiskLevel:       input.RiskLevel,
		ExpiresAt:       input.ExpiresAt,
		EntityType:      input.EntityType,
		EntityID:        input.EntityID,
	}
	if err := s.db.Create(req).Error; err != nil {
		return nil, fmt.Errorf("creating approval request: %w", err)
	}
	return req, nil
}

// Review approves or rejects a pending approval request.
func (s *Service) Review(id int64, input *ReviewApprovalInput) (*ApprovalRequest, error) {
	var req ApprovalRequest
	if err := s.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	if req.Status != StatusPending {
		return nil, fmt.Errorf("approval %d is not pending, current status: %s", id, req.Status)
	}

	status := StatusApproved
	if input.Action == "reject" {
		status = StatusRejected
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":           status,
		"reviewer":         input.Reviewer,
		"reviewer_user_id": input.ReviewerUserID,
		"review_note":      input.ReviewNote,
		"updated_at":       now,
	}

	// Build unified_action sync updates if applicable.
	uaUpdates := map[string]interface{}{}
	if req.EntityType == "unified_action" && req.EntityID > 0 {
		uaUpdates["status"] = status
		uaUpdates["updated_at"] = now
		if input.Action == "approve" {
			uaUpdates["approved_by"] = input.Reviewer
			uaUpdates["approved_at"] = now
		} else {
			uaUpdates["rejected_by"] = input.Reviewer
			uaUpdates["rejected_at"] = now
			uaUpdates["rejection_reason"] = input.ReviewNote
		}
	}

	// Wrap approval update + UA sync in a single transaction.
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&req).Updates(updates).Error; err != nil {
			return err
		}
		if len(uaUpdates) > 0 {
			if err := tx.Table("unified_action").Where("id = ?", req.EntityID).Updates(uaUpdates).Error; err != nil {
				return fmt.Errorf("sync unified_action: %w", err)
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	req.Status = status
	req.Reviewer = input.Reviewer
	req.ReviewNote = input.ReviewNote

	// Audit: record structured operation log for the review action
	if s.oplogSvc != nil {
		result := status
		_ = s.oplogSvc.LogStructured(&operationlog.StructuredLogInput{
			Module:      "approval",
			Action:      "approval.review",
			ResourceID:  fmt.Sprintf("%d", id),
			Operator:    input.Reviewer,
			Content:     fmt.Sprintf("approval_id=%d action=%s reviewer=%s", id, input.Action, input.Reviewer),
			Result:      result,
			TriggerType: "owner_approval",
			EntityType:  req.EntityType,
			EntityID:    req.EntityID,
		})
		// Log review note separately as structured field to avoid log injection.
		if input.ReviewNote != "" {
			_ = s.oplogSvc.LogStructured(&operationlog.StructuredLogInput{
				Module:      "approval",
				Action:      "approval.review.note",
				ResourceID:  fmt.Sprintf("%d", id),
				Operator:    input.Reviewer,
				Content:     input.ReviewNote, // ponytail: review_note is plain text recorded as-is for audit completeness
				Result:      "recorded",
				TriggerType: "owner_approval",
				EntityType:  req.EntityType,
				EntityID:    req.EntityID,
			})
		}
	}

	// Publish approval lifecycle event for closed-loop workflows
	// (e.g. listing task creation on approval.approved.listing_task).
	// Only publish on success — never on rollback.
	req.ReviewerUserID = input.ReviewerUserID
	s.publishApprovalEvent(&req, status)

	return &req, nil
}

// MyPending returns approval requests pending review.
func (s *Service) MyPending(page, size int) ([]ApprovalRequest, int64, error) {
	q := s.db.Model(&ApprovalRequest{}).Where("status = ?", StatusPending)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []ApprovalRequest
	offset := (page - 1) * size
	if err := q.Order("created_at ASC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []ApprovalRequest{}
	}
	return items, total, nil
}

// Stats returns aggregate approval statistics.
func (s *Service) Stats() (*ApprovalStats, error) {
	stats := &ApprovalStats{}

	// Count by status
	type statusCount struct {
		Status string
		Count  int64
	}
	var counts []statusCount
	if err := s.db.Model(&ApprovalRequest{}).
		Select("status, count(*) as count").
		Group("status").Find(&counts).Error; err != nil {
		return nil, err
	}
	for _, c := range counts {
		stats.TotalCount += c.Count
		switch c.Status {
		case StatusPending:
			stats.PendingCount = c.Count
		case StatusApproved:
			stats.ApprovedCount = c.Count
		case StatusRejected:
			stats.RejectedCount = c.Count
		}
	}

	// Average review time (for approved/rejected requests)
	var reviewed []ApprovalRequest
	s.db.Where("status IN ('approved', 'rejected')").Find(&reviewed)
	if len(reviewed) > 0 {
		var totalHours float64
		for _, r := range reviewed {
			if r.UpdatedAt.After(r.CreatedAt) {
				totalHours += r.UpdatedAt.Sub(r.CreatedAt).Hours()
			}
		}
		stats.AvgReviewHours = math.Round(totalHours/float64(len(reviewed))*100) / 100
	}

	// Count escalated (pending > 24h)
	var escalated int64
	if err := s.db.Model(&ApprovalRequest{}).
		Where("status = ? AND created_at < ?", StatusPending, time.Now().Add(-24*time.Hour)).
		Count(&escalated).Error; err != nil {
		return nil, err
	}
	stats.EscalatedCount = escalated

	return stats, nil
}

// HasPendingForEntity checks if there is a pending approval for the given entity.
// Used for duplicate-prevention when creating approvals linked to a listing task.
func (s *Service) HasPendingForEntity(entityType string, entityID int64) (bool, error) {
	var count int64
	err := s.db.Model(&ApprovalRequest{}).
		Where("entity_type = ? AND entity_id = ? AND status = ?", entityType, entityID, StatusPending).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindApprovedByTarget returns the most recent approved approval for a given target and request type.
func (s *Service) FindApprovedByTarget(targetType string, targetID int64, requestType string) (*ApprovalRequest, error) {
	var req ApprovalRequest
	err := s.db.
		Where("target_type = ? AND target_id = ? AND request_type = ? AND status = ?", targetType, targetID, requestType, StatusApproved).
		Order("updated_at DESC, id DESC").
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// AutoEscalate checks for requests pending > 24h and returns their IDs.
func (s *Service) AutoEscalate() ([]ApprovalRequest, error) {
	cutoff := time.Now().Add(-24 * time.Hour)
	var items []ApprovalRequest
	if err := s.db.Where("status = ? AND created_at < ?", StatusPending, cutoff).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	if items == nil {
		items = []ApprovalRequest{}
	}

	for _, item := range items {
		s.logger.Warn("approval request auto-escalated",
			zap.Int64("approval_id", item.ID),
			zap.Int64("product_id", item.ProductID),
			zap.String("request_type", item.RequestType),
			zap.Duration("pending_duration", time.Since(item.CreatedAt)),
		)
	}
	return items, nil
}

// publishApprovalEvent publishes an approval lifecycle event when there is an
// event bus attached. Used by router.go subscribers to trigger cross-domain
// workflows (e.g. listing task creation on approval).
func (s *Service) publishApprovalEvent(req *ApprovalRequest, status string) {
	if s.bus == nil {
		return
	}
	topic := fmt.Sprintf("approval.%s.%s", status, req.RequestType)
	ctx := context.Background()
	if _, err := s.bus.Publish(ctx, topic, "approval", map[string]interface{}{
		"approval_id":    req.ID,
		"status":          status,
		"request_type":    req.RequestType,
		"entity_type":     req.EntityType,
		"entity_id":       req.EntityID,
		"product_id":      req.ProductID,
		"reviewer":        req.Reviewer,
		"reviewer_user_id": req.ReviewerUserID,
		"target_type":     req.TargetType,
		"target_id":       req.TargetID,
	}); err != nil {
		s.logger.Warn("failed to publish approval event",
			zap.String("topic", topic),
			zap.Int64("approval_id", req.ID),
			zap.Error(err))
	}
}
