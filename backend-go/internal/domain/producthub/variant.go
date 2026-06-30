package producthub

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProductVariant maps to the "product_variant" table.
type ProductVariant struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductMasterID int64     `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	SKUProductID    int64     `gorm:"column:sku_product_id;index" json:"sku_product_id"`
	SKUCode         string    `gorm:"column:sku_code;size:64" json:"sku_code"`
	VariantLabel    string    `gorm:"column:variant_label;size:128" json:"variant_label"`
	Barcode         string    `gorm:"column:barcode;size:64" json:"barcode"`
	Weight          float64   `gorm:"column:weight" json:"weight"`
	Dimensions      string    `gorm:"column:dimensions;size:64" json:"dimensions"`
	CountryOfOrigin string    `gorm:"column:country_of_origin;size:8" json:"country_of_origin"`
	HSCode          string    `gorm:"column:hs_code;size:32" json:"hs_code"`
	Status          string    `gorm:"column:status;size:32;default:active" json:"status"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ProductVariant) TableName() string { return "product_variant" }

// VariantService handles product variant business logic.
type VariantService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewVariantService(db *gorm.DB, logger *zap.Logger) *VariantService {
	return &VariantService{db: db, logger: logger}
}

func (s *VariantService) ListByMaster(ctx context.Context, masterID int64) ([]ProductVariant, error) {
	var items []ProductVariant
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *VariantService) GetByID(ctx context.Context, id int64) (*ProductVariant, error) {
	var v ProductVariant
	if err := s.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *VariantService) Create(ctx context.Context, v *ProductVariant) error {
	return s.db.WithContext(ctx).Create(v).Error
}

func (s *VariantService) Update(ctx context.Context, v *ProductVariant) error {
	updates := map[string]interface{}{}
	if v.SKUProductID != 0 {
		updates["sku_product_id"] = v.SKUProductID
	}
	if v.SKUCode != "" {
		updates["sku_code"] = v.SKUCode
	}
	if v.VariantLabel != "" {
		updates["variant_label"] = v.VariantLabel
	}
	if v.Barcode != "" {
		updates["barcode"] = v.Barcode
	}
	if v.Weight > 0 {
		updates["weight"] = v.Weight
	}
	if v.Dimensions != "" {
		updates["dimensions"] = v.Dimensions
	}
	if v.CountryOfOrigin != "" {
		updates["country_of_origin"] = v.CountryOfOrigin
	}
	if v.HSCode != "" {
		updates["hs_code"] = v.HSCode
	}
	if v.Status != "" {
		updates["status"] = v.Status
	}
	return s.db.WithContext(ctx).Model(&ProductVariant{}).Where("id = ?", v.ID).Updates(updates).Error
}

func (s *VariantService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&ProductVariant{}, id).Error
}
