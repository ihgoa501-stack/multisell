package producthub

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CostVersion maps to the "cost_version" table.
type CostVersion struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id,omitempty"`
	ProductMasterID  int64      `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	Version          string     `gorm:"column:version;size:16" json:"version,omitempty"`
	BaseCost         float64    `gorm:"column:base_cost" json:"base_cost,omitempty"`
	MaterialCost     float64    `gorm:"column:material_cost" json:"material_cost,omitempty"`
	PackagingCost    float64    `gorm:"column:packaging_cost" json:"packaging_cost,omitempty"`
	FreightCost      float64    `gorm:"column:freight_cost" json:"freight_cost,omitempty"`
	DutyCost         float64    `gorm:"column:duty_cost" json:"duty_cost,omitempty"`
	PlatformFeeRate  float64    `gorm:"column:platform_fee_rate" json:"platform_fee_rate,omitempty"`
	AdCostEstimate   float64    `gorm:"column:ad_cost_estimate" json:"ad_cost_estimate,omitempty"`
	FXRate           float64    `gorm:"column:fx_rate;default:1" json:"fx_rate,omitempty"`
	FXRateDate       *time.Time `gorm:"column:fx_rate_date" json:"fx_rate_date,omitempty"`
	LandedCost       float64    `gorm:"column:landed_cost" json:"landed_cost"`
	RecommendedPrice float64    `gorm:"column:recommended_price" json:"recommended_price,omitempty"`
	GrossMargin      float64    `gorm:"column:gross_margin" json:"gross_margin"`
	EffectiveFrom    time.Time  `gorm:"column:effective_from;autoCreateTime" json:"effective_from"`
	Status           string     `gorm:"column:status;size:16;default:draft" json:"status,omitempty"`
	Notes            string     `gorm:"column:notes;type:text" json:"notes,omitempty"`
	CreatedBy        int64      `gorm:"column:created_by" json:"created_by"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CostVersion) TableName() string { return "cost_version" }

// CostLandedCost calculates landed cost from components.
func (c *CostVersion) CostLandedCost() float64 {
	return c.BaseCost + c.MaterialCost + c.PackagingCost + c.FreightCost + c.DutyCost
}

// CostGrossMargin calculates gross margin given a price.
func (c *CostVersion) CostGrossMargin(price float64) float64 {
	if price <= 0 {
		return 0
	}
	landed := c.CostLandedCost()
	return (price - landed - price*c.PlatformFeeRate/100 - c.AdCostEstimate) / price * 100
}

// CostVersionService handles cost snapshots.
type CostVersionService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewCostVersionService(db *gorm.DB, logger *zap.Logger) *CostVersionService {
	return &CostVersionService{db: db, logger: logger}
}

func (s *CostVersionService) GetLatestByMaster(ctx context.Context, masterID int64) (*CostVersion, error) {
	var cv CostVersion
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("id DESC").First(&cv).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

func (s *CostVersionService) ListByMaster(ctx context.Context, masterID int64) ([]CostVersion, error) {
	var items []CostVersion
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *CostVersionService) Create(ctx context.Context, cv *CostVersion) error {
	cv.LandedCost = cv.CostLandedCost()
	if cv.RecommendedPrice > 0 {
		cv.GrossMargin = cv.CostGrossMargin(cv.RecommendedPrice)
	}
	return s.db.WithContext(ctx).Create(cv).Error
}

func (s *CostVersionService) Confirm(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Model(&CostVersion{}).Where("id = ?", id).Update("status", "confirmed").Error
}
