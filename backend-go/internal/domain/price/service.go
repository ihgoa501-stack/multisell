package price

import (
	"context"
	"encoding/json"
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

// ---------------------------------------------------------------------------
// Competitor Price CRUD
// ---------------------------------------------------------------------------

// CreateCompetitorPrice records a new competitor price observation.
func (s *Service) CreateCompetitorPrice(ctx context.Context, cp *CompetitorPrice) error {
	if cp.CapturedAt.IsZero() {
		cp.CapturedAt = time.Now()
	}
	if cp.Currency == "" {
		cp.Currency = "USD"
	}
	return s.db.WithContext(ctx).Create(cp).Error
}

// ListCompetitorPrices returns paginated competitor prices, optionally filtered by SKU.
func (s *Service) ListCompetitorPrices(ctx context.Context, page, size int, skuID int64) ([]CompetitorPrice, int64, error) {
	var items []CompetitorPrice
	var total int64

	q := s.db.WithContext(ctx).Model(&CompetitorPrice{})
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
	if err := q.Order("captured_at DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetCompetitorPriceByID returns a single competitor price record.
func (s *Service) GetCompetitorPriceByID(ctx context.Context, id int64) (*CompetitorPrice, error) {
	var cp CompetitorPrice
	if err := s.db.WithContext(ctx).First(&cp, id).Error; err != nil {
		return nil, err
	}
	return &cp, nil
}

// DeleteCompetitorPrice removes a competitor price record (hard delete).
func (s *Service) DeleteCompetitorPrice(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&CompetitorPrice{}, id).Error
}

// GetLatestCompetitorPrices returns the most recent price per competitor for a SKU.
func (s *Service) GetLatestCompetitorPrices(ctx context.Context, skuID int64) ([]CompetitorPrice, error) {
	var all []CompetitorPrice
	if err := s.db.WithContext(ctx).
		Where("sku_id = ?", skuID).
		Order("captured_at DESC").
		Find(&all).Error; err != nil {
		return nil, err
	}
	// Deduplicate: keep latest entry per competitor.
	seen := make(map[string]bool)
	var latest []CompetitorPrice
	for _, p := range all {
		if !seen[p.CompetitorName] {
			seen[p.CompetitorName] = true
			latest = append(latest, p)
		}
	}
	return latest, nil
}

// ---------------------------------------------------------------------------
// Pricing Strategy CRUD
// ---------------------------------------------------------------------------

// SavePricingStrategy creates or updates a pricing strategy.
func (s *Service) SavePricingStrategy(ctx context.Context, ps *PricingStrategy) error {
	return s.db.WithContext(ctx).Save(ps).Error
}

// GetPricingStrategyByID returns a pricing strategy by ID.
func (s *Service) GetPricingStrategyByID(ctx context.Context, id int64) (*PricingStrategy, error) {
	var ps PricingStrategy
	if err := s.db.WithContext(ctx).First(&ps, id).Error; err != nil {
		return nil, err
	}
	return &ps, nil
}

// ListPricingStrategies returns paginated pricing strategies, optionally filtered by SKU.
func (s *Service) ListPricingStrategies(ctx context.Context, page, size int, skuID int64) ([]PricingStrategy, int64, error) {
	var items []PricingStrategy
	var total int64

	q := s.db.WithContext(ctx).Model(&PricingStrategy{})
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

// DeletePricingStrategy removes a pricing strategy (hard delete).
func (s *Service) DeletePricingStrategy(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&PricingStrategy{}, id).Error
}

// GetEffectiveStrategy returns the matching strategy for a SKU, falling back to the
// global default strategy (sku_id IS NULL) if no SKU-specific strategy is active.
// Returns nil when no strategy is configured at all.
func (s *Service) GetEffectiveStrategy(ctx context.Context, skuID int64) (*PricingStrategy, error) {
	// First try SKU-specific active strategy.
	var ps PricingStrategy
	err := s.db.WithContext(ctx).
		Where("sku_id = ? AND active = ?", skuID, true).
		Order("id DESC").
		First(&ps).Error
	if err == nil {
		return &ps, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Fall back to global default.
	err = s.db.WithContext(ctx).
		Where("sku_id IS NULL AND active = ?", true).
		Order("id DESC").
		First(&ps).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ps, nil
}

// ---------------------------------------------------------------------------
// Pricing Recommendations
// ---------------------------------------------------------------------------

// GenerateRecommendation uses the pricing engine to produce a recommendation for a SKU,
// then persists it to the database.
func (s *Service) GenerateRecommendation(ctx context.Context, input GenerateRecommendationInput) (*PricingRecommendation, error) {
	engine := NewPricingEngine()

	// 1. Get current price (best-effort; may be zero).
	currentPrice := decimal.Zero
	cp, err := s.GetCurrentPrice(ctx, input.SkuID)
	if err == nil {
		currentPrice = cp.Price
	}

	// 2. Get latest competitor prices.
	competitorPrices, err := s.GetLatestCompetitorPrices(ctx, input.SkuID)
	if err != nil {
		return nil, err
	}

	// 3. Determine strategy type and parameters.
	strategyType := input.StrategyType
	params := StrategyParameters{}

	if strategyType == "" {
		strategy, err := s.GetEffectiveStrategy(ctx, input.SkuID)
		if err != nil {
			return nil, err
		}
		if strategy != nil {
			strategyType = strategy.StrategyType
			if strategy.Parameters != "" && strategy.Parameters != "{}" {
				_ = json.Unmarshal([]byte(strategy.Parameters), &params)
			}
		}
	}

	if strategyType == "" {
		strategyType = StrategyBuyBoxFirst // sensible fallback
	}

	// 4. Run engine.
	engineInput := EngineInput{
		SkuID:            input.SkuID,
		CurrentPrice:     currentPrice,
		CompetitorPrices: competitorPrices,
		StrategyType:     strategyType,
		Parameters:       params,
		Cost:             input.Cost,
		PlatformFeeRate:  input.PlatformFeeRate,
	}
	recommendation := engine.Generate(engineInput)

	// 5. Persist.
	if err := s.db.WithContext(ctx).Create(recommendation).Error; err != nil {
		return nil, err
	}
	return recommendation, nil
}

// ListRecommendations returns paginated pricing recommendations, optionally filtered by SKU.
func (s *Service) ListRecommendations(ctx context.Context, page, size int, skuID int64) ([]PricingRecommendation, int64, error) {
	var items []PricingRecommendation
	var total int64

	q := s.db.WithContext(ctx).Model(&PricingRecommendation{})
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

// ApplyRecommendation marks a recommendation as applied.
func (s *Service) ApplyRecommendation(ctx context.Context, id int64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&PricingRecommendation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"applied":    true,
			"applied_at": &now,
		}).Error
}

// MarshalStrategyParams serializes StrategyParameters to a JSON string.
func MarshalStrategyParams(params StrategyParameters) (string, error) {
	b, err := json.Marshal(params)
	return string(b), err
}

// UnmarshalStrategyParams deserializes a JSON string into StrategyParameters.
func UnmarshalStrategyParams(s string) (StrategyParameters, error) {
	var params StrategyParameters
	if s == "" || s == "{}" {
		return params, nil
	}
	err := json.Unmarshal([]byte(s), &params)
	return params, err
}
