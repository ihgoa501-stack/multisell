package price

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides price business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new price service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ListPrices returns a paginated list of prices with optional filters.
func (s *Service) ListPrices(ctx context.Context, page, size int, skuID int64, priceType string) ([]Price, int64, error) {
	var items []Price
	var total int64

	q := s.db.WithContext(ctx).Model(&Price{})
	if skuID > 0 {
		q = q.Where("sku_id = ?", skuID)
	}
	if priceType != "" {
		q = q.Where("price_type = ?", priceType)
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

// ListPricesBySKU returns all prices for a given SKU.
func (s *Service) ListPricesBySKU(ctx context.Context, skuID int64) ([]Price, error) {
	var items []Price
	if err := s.db.WithContext(ctx).Where("sku_id = ? AND status = 1", skuID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetCurrentPrice returns the current sale price for a SKU.
func (s *Service) GetCurrentPrice(ctx context.Context, skuID int64) (*Price, error) {
	var p Price
	now := time.Now()
	err := s.db.WithContext(ctx).
		Where("sku_id = ? AND price_type = ? AND status = 1", skuID, "sale_price").
		Where("start_time IS NULL OR start_time <= ?", now).
		Where("end_time IS NULL OR end_time >= ?", now).
		Order("id DESC").
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPriceByID retrieves a single price by ID.
func (s *Service) GetPriceByID(ctx context.Context, id int64) (*Price, error) {
	var p Price
	if err := s.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// SetPrice creates or updates a price and logs the change.
func (s *Service) SetPrice(ctx context.Context, p *Price, operator string) error {
	p.PriceType = strings.TrimSpace(p.PriceType)
	if p.PriceType == "" {
		return gorm.ErrInvalidData
	}

	// Fetch the old price for the log
	var oldPrice *decimal.Decimal
	if p.ID > 0 {
		var existing Price
		if err := s.db.WithContext(ctx).First(&existing, p.ID).Error; err == nil {
			oldPrice = &existing.Price
		}
	} else if p.SkuID > 0 {
		// Check if there's an existing active price for this SKU/type
		var existing Price
		if err := s.db.WithContext(ctx).
			Where("sku_id = ? AND price_type = ? AND status = 1", p.SkuID, p.PriceType).
			Order("id DESC").First(&existing).Error; err == nil {
			oldPrice = &existing.Price
			p.ID = existing.ID
		}
	}

	if err := s.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}

	// Log the change
	log := PriceChangeLog{
		SkuID:      p.SkuID,
		OldPrice:   oldPrice,
		NewPrice:   &p.Price,
		PriceType:  p.PriceType,
		ChangeType: "manual",
		Operator:   operator,
	}
	if err := s.db.WithContext(ctx).Create(&log).Error; err != nil {
		s.logger.Warn("failed to create price change log", zap.Error(err))
	}

	return nil
}

// UpdatePrice saves changes to an existing price.
func (s *Service) UpdatePrice(ctx context.Context, p *Price) error {
	return s.db.WithContext(ctx).Save(p).Error
}

// DeletePrice removes a price by ID (hard delete).
func (s *Service) DeletePrice(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&Price{}, id).Error
}

// ListChangeLogs returns price change logs for a SKU.
func (s *Service) ListChangeLogs(ctx context.Context, skuID int64, page, size int) ([]PriceChangeLog, int64, error) {
	var items []PriceChangeLog
	var total int64

	q := s.db.WithContext(ctx).Model(&PriceChangeLog{})
	if skuID > 0 {
		q = q.Where("sku_id = ?", skuID)
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
