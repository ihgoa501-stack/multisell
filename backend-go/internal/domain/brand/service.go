package brand

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides brand business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new brand service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns a paginated list of brands with optional search.
func (s *Service) List(ctx context.Context, page, size int, search string) ([]Brand, int64, error) {
	var items []Brand
	var total int64

	q := s.db.WithContext(ctx).Model(&Brand{})
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
	if err := q.Order("sort_order ASC, id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListAll returns all enabled brands (for dropdown selectors).
func (s *Service) ListAll(ctx context.Context) ([]Brand, error) {
	var items []Brand
	if err := s.db.WithContext(ctx).Where("status = 1").Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetByID retrieves a single brand by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Brand, error) {
	var b Brand
	if err := s.db.WithContext(ctx).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// Create inserts a new brand.
func (s *Service) Create(ctx context.Context, b *Brand) error {
	b.Name = strings.TrimSpace(b.Name)
	if b.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(b).Error
}

// Update saves changes to an existing brand.
func (s *Service) Update(ctx context.Context, b *Brand) error {
	b.Name = strings.TrimSpace(b.Name)
	if b.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(b).Error
}

// Delete removes a brand by ID (hard delete — no soft-delete column).
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&Brand{}, id).Error
}
