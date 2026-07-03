package {{ModuleName}}

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides {{module_name}} business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new {{module_name}} service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns a paginated list of {{module_name}}s with optional search.
func (s *Service) List(ctx context.Context, page, size int, search string) ([]{{ModuleName}}, int64, error) {
	var items []{{ModuleName}}
	var total int64

	q := s.db.WithContext(ctx).Model(&{{ModuleName}}{})
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID retrieves a single {{module_name}} by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*{{ModuleName}}, error) {
	var m {{ModuleName}}
	if err := s.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// Create inserts a new {{module_name}}.
func (s *Service) Create(ctx context.Context, m *{{ModuleName}}) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(m).Error
}

// Update saves changes to an existing {{module_name}}.
func (s *Service) Update(ctx context.Context, m *{{ModuleName}}) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(m).Error
}

// Delete removes a {{module_name}} by ID.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&{{ModuleName}}{}, id).Error
}
