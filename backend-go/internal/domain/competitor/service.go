package competitor

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides competitor business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new competitor service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ── Competitor Product ───────────────────────────────────────────────

// List returns a paginated list of competitor products.
func (s *Service) List(ctx context.Context, page, size int, platform, search string) ([]CompetitorProduct, int64, error) {
	var items []CompetitorProduct
	var total int64

	q := s.db.WithContext(ctx).Model(&CompetitorProduct{})
	if platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if search != "" {
		q = q.Where("name ILIKE ? OR sku_code ILIKE ?", "%"+search+"%", "%"+search+"%")
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

// GetByID retrieves a single competitor product.
func (s *Service) GetByID(ctx context.Context, id int64) (*CompetitorProduct, error) {
	var cp CompetitorProduct
	if err := s.db.WithContext(ctx).First(&cp, id).Error; err != nil {
		return nil, err
	}
	return &cp, nil
}

// Create inserts a new competitor product.
func (s *Service) Create(ctx context.Context, input *CreateCompetitorInput) (*CompetitorProduct, error) {
	cp := &CompetitorProduct{
		Name:       strings.TrimSpace(input.Name),
		Platform:   strings.ToLower(strings.TrimSpace(input.Platform)),
		ProductURL: input.ProductURL,
		SkuCode:    input.SkuCode,
		Category:   input.Category,
		Brand:      input.Brand,
		Status:     1,
	}
	if cp.Name == "" || cp.Platform == "" {
		return nil, fmt.Errorf("name and platform are required")
	}
	if err := s.db.WithContext(ctx).Create(cp).Error; err != nil {
		return nil, err
	}
	return cp, nil
}

// Update saves changes to an existing competitor product.
func (s *Service) Update(ctx context.Context, cp *CompetitorProduct) error {
	cp.Name = strings.TrimSpace(cp.Name)
	return s.db.WithContext(ctx).Save(cp).Error
}

// Delete removes a competitor product by ID.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&CompetitorProduct{}, id).Error
}

// ── Price Snapshots ──────────────────────────────────────────────────

// RecordPrice records a price snapshot for a competitor.
func (s *Service) RecordPrice(ctx context.Context, competitorID int64, input *RecordPriceInput) (*PriceSnapshot, error) {
	// Verify competitor exists.
	if _, err := s.GetByID(ctx, competitorID); err != nil {
		return nil, fmt.Errorf("competitor not found: %w", err)
	}

	snapshotDate := time.Now()
	if input.SnapshotDate != "" {
		if parsed, err := time.Parse(time.RFC3339, input.SnapshotDate); err == nil {
			snapshotDate = parsed
		}
	}
	currency := input.Currency
	if currency == "" {
		currency = "CNY"
	}

	snapshot := &PriceSnapshot{
		CompetitorID:  competitorID,
		Price:         input.Price,
		OriginalPrice: input.OriginalPrice,
		Currency:      currency,
		SalesLast30d:  input.SalesLast30d,
		Rating:        input.Rating,
		ReviewCount:   input.ReviewCount,
		IsInStock:     input.IsInStock,
		SnapshotDate:  snapshotDate,
	}
	if err := s.db.WithContext(ctx).Create(snapshot).Error; err != nil {
		return nil, err
	}
	return snapshot, nil
}

// ListPrices returns price snapshots for a competitor, ordered by date desc.
func (s *Service) ListPrices(ctx context.Context, competitorID int64, limit int) ([]PriceSnapshot, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var snapshots []PriceSnapshot
	if err := s.db.WithContext(ctx).
		Where("competitor_id = ?", competitorID).
		Order("snapshot_date DESC").
		Limit(limit).
		Find(&snapshots).Error; err != nil {
		return nil, err
	}
	return snapshots, nil
}

// GetPriceTrend returns price trend analysis for a competitor.
func (s *Service) GetPriceTrend(ctx context.Context, competitorID int64) (*PriceTrend, error) {
	// Get all snapshots for analysis (no limit for trend calc).
	var snapshots []PriceSnapshot
	if err := s.db.WithContext(ctx).
		Where("competitor_id = ?", competitorID).
		Order("snapshot_date ASC").
		Find(&snapshots).Error; err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no price snapshots for competitor %d", competitorID)
	}

	trend := &PriceTrend{
		CompetitorID: competitorID,
		Snapshots:    snapshots,
		CurrentPrice: snapshots[len(snapshots)-1].Price,
	}

	// Min/Max/Avg.
	var sum float64
	trend.MinPrice = math.MaxFloat64
	for _, s := range snapshots {
		if s.Price < trend.MinPrice {
			trend.MinPrice = s.Price
		}
		if s.Price > trend.MaxPrice {
			trend.MaxPrice = s.Price
		}
		sum += s.Price
	}
	trend.AvgPrice = round2(sum / float64(len(snapshots)))

	// 7-day change.
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	var sevenDayOld *PriceSnapshot
	for _, s := range snapshots {
		if !s.SnapshotDate.After(sevenDaysAgo) {
			sevenDayOld = &s
		}
	}
	if sevenDayOld != nil && sevenDayOld.Price > 0 {
		trend.PriceChange7d = round2((trend.CurrentPrice - sevenDayOld.Price) / sevenDayOld.Price * 100)
	}

	// 30-day change.
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var thirtyDayOld *PriceSnapshot
	for _, s := range snapshots {
		if !s.SnapshotDate.After(thirtyDaysAgo) {
			thirtyDayOld = &s
		}
	}
	if thirtyDayOld != nil && thirtyDayOld.Price > 0 {
		trend.PriceChange30d = round2((trend.CurrentPrice - thirtyDayOld.Price) / thirtyDayOld.Price * 100)
	}

	return trend, nil
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
