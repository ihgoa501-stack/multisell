package producthub

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DataFreshness tracks the verification status of a product dimension.
type DataFreshness struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID      int64     `gorm:"column:product_id;not null;index" json:"product_id"`
	Dimension      string    `gorm:"column:dimension;not null" json:"dimension"`
	LastVerifiedAt time.Time `gorm:"column:last_verified_at" json:"last_verified_at"`
	NextCheckAt    time.Time `gorm:"column:next_check_at" json:"next_check_at"`
	FreshnessDays  int       `gorm:"column:freshness_days;default:30" json:"freshness_days"`
	Status         string    `gorm:"column:status;default:fresh" json:"status"`
	DriftDetected  bool      `gorm:"column:drift_detected;default:false" json:"drift_detected"`
	LastValue      string    `gorm:"column:last_value;type:text" json:"last_value,omitempty"`
	CurrentValue   string    `gorm:"column:current_value;type:text" json:"current_value,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (DataFreshness) TableName() string { return "data_freshness" }

// VerifyRequest is the input for verifying a dimension.
type VerifyRequest struct {
	Dimension    string `json:"dimension" binding:"required,oneof=pricing content inventory compliance"`
	CurrentValue string `json:"current_value" binding:"required"`
}

// FreshnessSummary enriches a DataFreshness with computed display fields.
type FreshnessSummary struct {
	DataFreshness
	FreshnessLabel string `json:"freshness_label"`
	DaysSinceCheck int    `json:"days_since_check"`
}

// StaleProductsResponse wraps the stale product list.
type StaleProductsResponse struct {
	Total     int                `json:"total"`
	Freshness []FreshnessSummary `json:"freshness"`
}

// FreshnessService defines operations for tracking product data freshness.
type FreshnessService interface {
	// RecordVerification marks a dimension as verified for a product.
	RecordVerification(ctx context.Context, productID int64, dimension string, currentValue string) error

	// CheckFreshness returns products with stale or expired data.
	CheckFreshness(ctx context.Context) ([]DataFreshness, error)

	// GetProductFreshness returns freshness status for all dimensions of a product.
	GetProductFreshness(ctx context.Context, productID int64) ([]FreshnessSummary, error)

	// DetectDrift checks if a dimension's current value differs from the last verified value.
	DetectDrift(ctx context.Context, productID int64, dimension string, currentValue string) (bool, error)
}

// freshnessSvc implements FreshnessService.
type freshnessSvc struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewFreshnessService creates a new FreshnessService.
func NewFreshnessService(db *gorm.DB, logger *zap.Logger) FreshnessService {
	return &freshnessSvc{
		db:     db,
		logger: logger,
	}
}

// RecordVerification marks a dimension as verified, comparing the new value to detect drift.
func (s *freshnessSvc) RecordVerification(ctx context.Context, productID int64, dimension string, currentValue string) error {
	now := time.Now()

	var existing DataFreshness
	err := s.db.WithContext(ctx).
		Where("product_id = ? AND dimension = ?", productID, dimension).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create a new freshness record.
		freshnessDays := 30
		freshness := DataFreshness{
			ProductID:      productID,
			Dimension:      dimension,
			LastVerifiedAt: now,
			NextCheckAt:    now.AddDate(0, 0, freshnessDays),
			FreshnessDays:  freshnessDays,
			Status:         "fresh",
			DriftDetected:  false,
			CurrentValue:   currentValue,
			LastValue:      currentValue,
		}
		return s.db.WithContext(ctx).Create(&freshness).Error
	}
	if err != nil {
		return fmt.Errorf("query freshness record: %w", err)
	}

	// Detect drift
	driftDetected := existing.LastValue != "" && existing.LastValue != currentValue

	updates := map[string]interface{}{
		"last_verified_at": now,
		"next_check_at":    now.AddDate(0, 0, existing.FreshnessDays),
		"current_value":    currentValue,
		"drift_detected":   driftDetected,
		"status":           "fresh",
	}
	// Only update last_value if it hasn't been set, to preserve the original reference value.
	if existing.LastValue == "" {
		updates["last_value"] = currentValue
	}

	if driftDetected {
		s.logger.Warn("data drift detected",
			zap.Int64("product_id", productID),
			zap.String("dimension", dimension),
			zap.String("last_value", existing.LastValue),
			zap.String("current_value", currentValue),
		)
	}

	return s.db.WithContext(ctx).Model(&existing).Updates(updates).Error
}

// CheckFreshness returns all records that need attention (stale or expired).
func (s *freshnessSvc) CheckFreshness(ctx context.Context) ([]DataFreshness, error) {
	now := time.Now()

	// Mark records where next_check_at is overdue beyond 2x freshness_days as expired.
	expireCutoff := now.AddDate(0, 0, -60) // 2 months default safety net
	if err := s.db.WithContext(ctx).Model(&DataFreshness{}).
		Where("next_check_at < ? AND status != ?", expireCutoff, "expired").
		Update("status", "expired").Error; err != nil {
		return nil, fmt.Errorf("expire stale records: %w", err)
	}

	// Mark overdue but not yet expired as stale.
	if err := s.db.WithContext(ctx).Model(&DataFreshness{}).
		Where("next_check_at < ? AND status = ?", now, "fresh").
		Update("status", "stale").Error; err != nil {
		return nil, fmt.Errorf("mark stale records: %w", err)
	}

	// Return all stale, expired, or drift-detected records.
	var results []DataFreshness
	if err := s.db.WithContext(ctx).
		Where("status IN ('stale', 'expired') OR drift_detected = ?", true).
		Order("next_check_at ASC").
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("query stale products: %w", err)
	}
	return results, nil
}

// GetProductFreshness returns all freshness records for a product with computed labels.
func (s *freshnessSvc) GetProductFreshness(ctx context.Context, productID int64) ([]FreshnessSummary, error) {
	var records []DataFreshness
	if err := s.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Order("dimension ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("query product freshness: %w", err)
	}

	now := time.Now()
	summaries := make([]FreshnessSummary, 0, len(records))
	for _, r := range records {
		label := computeFreshnessLabel(r, now)
		daysSince := int(now.Sub(r.LastVerifiedAt).Hours() / 24)
		summaries = append(summaries, FreshnessSummary{
			DataFreshness:  r,
			FreshnessLabel: label,
			DaysSinceCheck: daysSince,
		})
	}
	return summaries, nil
}

// DetectDrift checks if a dimension's current value differs from the last verified value.
func (s *freshnessSvc) DetectDrift(ctx context.Context, productID int64, dimension string, currentValue string) (bool, error) {
	var existing DataFreshness
	err := s.db.WithContext(ctx).
		Where("product_id = ? AND dimension = ?", productID, dimension).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query freshness record: %w", err)
	}
	if existing.LastValue == "" {
		return false, nil
	}

	drifted := existing.LastValue != currentValue
	if drifted {
		s.db.WithContext(ctx).Model(&existing).Update("drift_detected", true)
	}
	return drifted, nil
}

// computeFreshnessLabel derives a human-readable label from a freshness record.
func computeFreshnessLabel(f DataFreshness, now time.Time) string {
	if f.DriftDetected {
		return "drift"
	}
	switch f.Status {
	case "fresh":
		return "fresh"
	case "stale":
		return "stale"
	case "expired":
		return "expired"
	default:
		return "unknown"
	}
}

