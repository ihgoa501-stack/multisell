package approval

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides CRUD operations and auto-escalation for approval requests.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new approval service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
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
		ProductID:   input.ProductID,
		RequestType: input.RequestType,
		Requester:   input.Requester,
		Status:      "pending",
		OldValue:    input.OldValue,
		NewValue:    input.NewValue,
		Reason:      input.Reason,
		ExpiresAt:   input.ExpiresAt,
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
	if req.Status != "pending" {
		return nil, fmt.Errorf("approval %d is not pending, current status: %s", id, req.Status)
	}

	status := "approved"
	if input.Action == "reject" {
		status = "rejected"
	}

	updates := map[string]interface{}{
		"status":      status,
		"reviewer":    input.Reviewer,
		"review_note": input.ReviewNote,
		"updated_at":  time.Now(),
	}
	if err := s.db.Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}

	req.Status = status
	req.Reviewer = input.Reviewer
	req.ReviewNote = input.ReviewNote
	return &req, nil
}

// MyPending returns approval requests pending review.
func (s *Service) MyPending(page, size int) ([]ApprovalRequest, int64, error) {
	q := s.db.Model(&ApprovalRequest{}).Where("status = ?", "pending")

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
		case "pending":
			stats.PendingCount = c.Count
		case "approved":
			stats.ApprovedCount = c.Count
		case "rejected":
			stats.RejectedCount = c.Count
		}
	}

	// Average review time (for approved/rejected requests) — computed in Go so
	// it works on both PostgreSQL and SQLite (tests).
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
		Where("status = 'pending' AND created_at < ?", time.Now().Add(-24*time.Hour)).
		Count(&escalated).Error; err != nil {
		return nil, err
	}
	stats.EscalatedCount = escalated

	return stats, nil
}



// Cancel cancels a pending approval request.
func (s *Service) Cancel(id int64, reason, canceledBy string) (*ApprovalRequest, error) {
	var req ApprovalRequest
	if err := s.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	sm := NewApprovalStateMachine()
	if err := sm.MustTransition(context.Background(), req.Status, "canceled", &req); err != nil {
		return nil, err
	}
	if err := s.db.Model(&req).Updates(map[string]interface{}{
		"status":      "canceled",
		"review_note": reason,
		"reviewer":    canceledBy,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = "canceled"
	req.ReviewNote = reason
	req.Reviewer = canceledBy
	return &req, nil
}

// ExpirePending marks all expired pending approvals as expired.
func (s *Service) ExpirePending() (int64, error) {
	res := s.db.Model(&ApprovalRequest{}).
		Where("status = 'pending' AND expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Update("status", "expired")
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// Supersede marks an approval as superseded by a newer one.
func (s *Service) Supersede(id int64, newApprovalID int64) (*ApprovalRequest, error) {
	var req ApprovalRequest
	if err := s.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	sm := NewApprovalStateMachine()
	if err := sm.MustTransition(context.Background(), req.Status, "superseded", &req); err != nil {
		return nil, err
	}
	note := fmt.Sprintf("superseded by approval %d", newApprovalID)
	if err := s.db.Model(&req).Updates(map[string]interface{}{
		"status":      "superseded",
		"review_note": note,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = "superseded"
	req.ReviewNote = note
	return &req, nil
}
// AutoEscalate checks for requests pending > 24h and returns their IDs.
func (s *Service) AutoEscalate() ([]ApprovalRequest, error) {
	cutoff := time.Now().Add(-24 * time.Hour)
	var items []ApprovalRequest
	if err := s.db.Where("status = 'pending' AND created_at < ?", cutoff).
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
