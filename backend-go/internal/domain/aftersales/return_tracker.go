package aftersales

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SkuReturnStat maps to the PostgreSQL "sku_return_stats" table.
type SkuReturnStat struct {
	SkuID        int64     `gorm:"column:sku_id;primaryKey" json:"sku_id"`
	TotalOrders  int64     `gorm:"column:total_orders;default:0" json:"total_orders"`
	TotalReturns int64     `gorm:"column:total_returns;default:0" json:"total_returns"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name for SkuReturnStat.
func (SkuReturnStat) TableName() string { return "sku_return_stats" }

// ReturnRateTracker is a DB-backed tracker for SKU-level return rates.
// It records every return event in the sku_return_stats table and computes
// the return rate as (total_returns / total_orders) * 100.
type ReturnRateTracker struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewReturnRateTracker creates a new DB-backed ReturnRateTracker.
func NewReturnRateTracker(db *gorm.DB, logger *zap.Logger) *ReturnRateTracker {
	return &ReturnRateTracker{db: db, logger: logger}
}

// TrackReturn increments the order count by 1 and the return count by qty
// for the given SKU. Creates a new stats row if none exists.
func (t *ReturnRateTracker) TrackReturn(skuID int64, qty int) error {
	var stat SkuReturnStat
	err := t.db.Where("sku_id = ?", skuID).First(&stat).Error
	if err == gorm.ErrRecordNotFound {
		// First return for this SKU — create new stats record.
		return t.db.Create(&SkuReturnStat{
			SkuID:        skuID,
			TotalOrders:  1,
			TotalReturns: int64(qty),
		}).Error
	}
	if err != nil {
		return err
	}
	// Subsequent return — increment existing counters.
	return t.db.Model(&SkuReturnStat{}).
		Where("sku_id = ?", skuID).
		Updates(map[string]interface{}{
			"total_orders":  stat.TotalOrders + 1,
			"total_returns": stat.TotalReturns + int64(qty),
		}).Error
}

// GetReturnRate returns the return rate for a given SKU as a percentage:
// (total_returns / total_orders) * 100.
// Returns 0 if the SKU has no recorded orders.
func (t *ReturnRateTracker) GetReturnRate(skuID int64) (float64, error) {
	var stat SkuReturnStat
	err := t.db.Where("sku_id = ?", skuID).First(&stat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	if stat.TotalOrders == 0 {
		return 0, nil
	}
	return float64(stat.TotalReturns) / float64(stat.TotalOrders) * 100, nil
}
