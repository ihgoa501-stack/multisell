package exceptions

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ListFilter holds exception list query parameters.
type ListFilter struct {
	SourceModule string
	SourceType  string
	Severity    string
	Status      string
	AssignedTo  string
}

// Service provides exceptions business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new exceptions service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns a paginated list of exceptions filtered by severity/status/type.
func (s *Service) List(f ListFilter, page, size int) ([]ExceptionItem, int64, error) {
	q := s.db.Model(&ExceptionItem{})
	if f.SourceModule != "" {
		q = q.Where("source_module = ?", f.SourceModule)
	}
	if f.SourceType != "" {
		q = q.Where("source_type = ?", f.SourceType)
	}
	if f.Severity != "" {
		q = q.Where("severity = ?", f.Severity)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.AssignedTo != "" {
		q = q.Where("assigned_to = ?", f.AssignedTo)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var items []ExceptionItem
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single exception item.
func (s *Service) GetByID(id int64) (*ExceptionItem, error) {
	var e ExceptionItem
	if err := s.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// Create inserts a new exception item.
func (s *Service) Create(e *ExceptionItem) error {
	return s.db.Create(e).Error
}

// Resolve marks an exception as resolved and records the resolver + note.
func (s *Service) Resolve(id int64, resolvedBy, note string) (*ExceptionItem, error) {
	var e ExceptionItem
	if err := s.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	e.Status = "resolved"
	e.ResolvedAt = &now
	e.ResolvedBy = resolvedBy
	if note != "" {
		e.Note = note
	}
	if err := s.db.Save(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// Assign assigns an exception to a user and sets status to assigned.
func (s *Service) Assign(id int64, assignedTo string) (*ExceptionItem, error) {
	var e ExceptionItem
	if err := s.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	e.AssignedTo = assignedTo
	e.Status = "assigned"
	if err := s.db.Save(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// Update performs a general update on an exception item.
func (s *Service) Update(e *ExceptionItem) error {
	return s.db.Save(e).Error
}

// Delete removes an exception item.
func (s *Service) Delete(id int64) error {
	return s.db.Delete(&ExceptionItem{}, id).Error
}
