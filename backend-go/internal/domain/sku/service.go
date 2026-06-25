package sku

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides SKU/Product business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new SKU service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ── Product ───────────────────────────────────────────────────────

// ListProducts returns a paginated list of products with optional search and filters.
func (s *Service) ListProducts(ctx context.Context, page, size int, search string, categoryID, brandID int64, status *int16) ([]Product, int64, error) {
	var items []Product
	var total int64

	q := s.db.WithContext(ctx).Model(&Product{})
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}
	if categoryID > 0 {
		q = q.Where("category_id = ?", categoryID)
	}
	if brandID > 0 {
		q = q.Where("brand_id = ?", brandID)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
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

// GetProductByID retrieves a single product by ID.
func (s *Service) GetProductByID(ctx context.Context, id int64) (*Product, error) {
	var p Product
	if err := s.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// CreateProduct inserts a new product.
func (s *Service) CreateProduct(ctx context.Context, p *Product) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(p).Error
}

// UpdateProduct saves changes to an existing product.
func (s *Service) UpdateProduct(ctx context.Context, p *Product) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(p).Error
}

// DeleteProduct removes a product by ID (hard delete).
func (s *Service) DeleteProduct(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&Product{}, id).Error
}

// ── SpecName ──────────────────────────────────────────────────────

// ListSpecNames returns spec names for a given product, including their values.
func (s *Service) ListSpecNames(ctx context.Context, productID int64) ([]SpecName, error) {
	var items []SpecName
	if err := s.db.WithContext(ctx).
		Preload("Values", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("product_id = ?", productID).
		Order("sort_order ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CreateSpecName inserts a new spec name.
func (s *Service) CreateSpecName(ctx context.Context, sn *SpecName) error {
	sn.Name = strings.TrimSpace(sn.Name)
	if sn.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(sn).Error
}

// UpdateSpecName saves changes to an existing spec name.
func (s *Service) UpdateSpecName(ctx context.Context, sn *SpecName) error {
	sn.Name = strings.TrimSpace(sn.Name)
	if sn.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(sn).Error
}

// DeleteSpecName removes a spec name by ID (hard delete).
func (s *Service) DeleteSpecName(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&SpecName{}, id).Error
}

// ── SpecValue ────────────────────────────────────────────────────

// CreateSpecValue inserts a new spec value.
func (s *Service) CreateSpecValue(ctx context.Context, sv *SpecValue) error {
	sv.Value = strings.TrimSpace(sv.Value)
	if sv.Value == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(sv).Error
}

// UpdateSpecValue saves changes to an existing spec value.
func (s *Service) UpdateSpecValue(ctx context.Context, sv *SpecValue) error {
	sv.Value = strings.TrimSpace(sv.Value)
	if sv.Value == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(sv).Error
}

// DeleteSpecValue removes a spec value by ID (hard delete).
func (s *Service) DeleteSpecValue(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&SpecValue{}, id).Error
}

// ── Sku ───────────────────────────────────────────────────────────

// ListSkus returns a paginated list of SKUs with optional search and product filter.
func (s *Service) ListSkus(ctx context.Context, page, size int, search string, productID int64) ([]Sku, int64, error) {
	var items []Sku
	var total int64

	q := s.db.WithContext(ctx).Model(&Sku{})
	if search != "" {
		q = q.Where("code ILIKE ? OR barcode ILIKE ? OR spec_desc ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if productID > 0 {
		q = q.Where("product_id = ?", productID)
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

// ListSkusByProduct returns all SKUs for a given product.
func (s *Service) ListSkusByProduct(ctx context.Context, productID int64) ([]Sku, error) {
	var items []Sku
	if err := s.db.WithContext(ctx).Where("product_id = ?", productID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetSkuByID retrieves a single SKU by ID.
func (s *Service) GetSkuByID(ctx context.Context, id int64) (*Sku, error) {
	var sk Sku
	if err := s.db.WithContext(ctx).First(&sk, id).Error; err != nil {
		return nil, err
	}
	return &sk, nil
}

// CreateSku inserts a new SKU.
func (s *Service) CreateSku(ctx context.Context, sk *Sku) error {
	return s.db.WithContext(ctx).Create(sk).Error
}

// UpdateSku saves changes to an existing SKU.
func (s *Service) UpdateSku(ctx context.Context, sk *Sku) error {
	return s.db.WithContext(ctx).Save(sk).Error
}

// DeleteSku removes a SKU by ID (hard delete).
func (s *Service) DeleteSku(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&Sku{}, id).Error
}
