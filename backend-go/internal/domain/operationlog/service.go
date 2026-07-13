package operationlog

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ListFilter holds operation-log list query parameters.
type ListFilter struct {
	Module   string
	Action   string
	Operator string
	From     time.Time
	To       time.Time
}

// VerifyIntegrity validates hashes and the linked-list shape independently of
// row ID allocation order. PostgreSQL triggers own hash generation so every
// insert path, including writes inside domain transactions, is covered.
func (s *Service) VerifyIntegrity(ctx context.Context) error {
	var status struct {
		Total               int64
		SelfHashBad         int64
		Roots               int64
		MissingPredecessors int64
		ForkPoints          int64
		Tips                int64
		Reachable           int64
	}
	err := s.db.WithContext(ctx).Raw(`SELECT * FROM audit_operation_chain_status()`).Scan(&status).Error
	if err != nil {
		auditIntegrity.Set(0)
		return err
	}
	validShape := status.Total == 0 || (status.Roots == 1 && status.Tips == 1)
	if status.SelfHashBad != 0 || status.MissingPredecessors != 0 || status.ForkPoints != 0 ||
		status.Reachable != status.Total || !validShape {
		auditIntegrity.Set(0)
		return fmt.Errorf(
			"operation log integrity broken: total=%d self_hash_bad=%d roots=%d missing_predecessors=%d forks=%d tips=%d reachable=%d",
			status.Total, status.SelfHashBad, status.Roots, status.MissingPredecessors,
			status.ForkPoints, status.Tips, status.Reachable,
		)
	}
	auditIntegrity.Set(1)
	return nil
}

// StructuredLogInput carries structured audit fields for LogStructured.
type StructuredLogInput struct {
	Module            string
	Action            string
	ResourceID        string
	Operator          string
	Content           string
	Result            string
	TriggerType       string
	AgentSuggestionID *int64
	ApprovalID        *int64
	EntityType        string
	EntityID          int64
}

// Service provides operationlog business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new operationlog service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns a paginated list of operation logs with filters.
func (s *Service) List(f ListFilter, page, size int) ([]OperationLog, int64, error) {
	q := s.db.Model(&OperationLog{})
	if f.Module != "" {
		q = q.Where("module = ?", f.Module)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.Operator != "" {
		q = q.Where("operator = ?", f.Operator)
	}
	if !f.From.IsZero() {
		q = q.Where("created_at >= ?", f.From)
	}
	if !f.To.IsZero() {
		q = q.Where("created_at <= ?", f.To)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var items []OperationLog
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single operation log.
func (s *Service) GetByID(id int64) (*OperationLog, error) {
	var l OperationLog
	if err := s.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Create records a new operation log entry (called by other services after mutations).
func (s *Service) Create(l *OperationLog) error {
	return s.CreateWithContext(context.Background(), l)
}

// CreateWithContext records an operation log using the caller's lifetime and
// cancellation policy. Infrastructure callers should prefer this method so an
// audit write cannot outlive its request or shutdown deadline.
func (s *Service) CreateWithContext(ctx context.Context, l *OperationLog) error {
	err := s.db.WithContext(ctx).Create(l).Error
	if err != nil {
		auditWritesTotal.WithLabelValues("failure").Inc()
		return err
	}
	auditWritesTotal.WithLabelValues("success").Inc()
	return nil
}

// Log is a convenience helper for recording mutations (backward-compatible).
func (s *Service) Log(module, action, resourceID, operator, content string) error {
	return s.Create(&OperationLog{
		Module:     module,
		Action:     action,
		ResourceID: resourceID,
		Operator:   operator,
		Content:    RedactSensitive(content),
	})
}

// LogStructured records a structured audit log entry with trigger metadata.
func (s *Service) LogStructured(input *StructuredLogInput) error {
	return s.Create(&OperationLog{
		Module:            input.Module,
		Action:            input.Action,
		ResourceID:        input.ResourceID,
		Operator:          input.Operator,
		Content:           RedactSensitive(input.Content),
		Result:            input.Result,
		TriggerType:       input.TriggerType,
		AgentSuggestionID: input.AgentSuggestionID,
		ApprovalID:        input.ApprovalID,
		EntityType:        input.EntityType,
		EntityID:          input.EntityID,
	})
}
