package operationlog

import (
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
	return s.db.Create(l).Error
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
