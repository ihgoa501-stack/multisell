package producthub

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProductConcept maps to the "product_concept" table.
type ProductConcept struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id,omitempty"`
	ProductMasterID int64           `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	Brief           string          `gorm:"column:brief;type:text" json:"brief,omitempty"`
	TargetCustomer  string          `gorm:"column:target_customer;type:text" json:"target_customer,omitempty"`
	PainPoint       string          `gorm:"column:pain_point;type:text" json:"pain_point,omitempty"`
	MarketResearch  string          `gorm:"column:market_research;type:text" json:"market_research,omitempty"`
	CompetitorInfo  string          `gorm:"column:competitor_info;type:text" json:"competitor_info,omitempty"`
	DesignSource    string          `gorm:"column:design_source;size:32" json:"design_source,omitempty"`
	AttachmentURLs  json.RawMessage `gorm:"column:attachment_urls;type:jsonb" json:"attachment_urls,omitempty"`
	Status          string          `gorm:"column:status;size:32;default:draft" json:"status,omitempty"`
	CreatedBy       int64           `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ProductConcept) TableName() string { return "product_concept" }

// ConceptService handles product concept business logic.
type ConceptService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewConceptService(db *gorm.DB, logger *zap.Logger) *ConceptService {
	return &ConceptService{db: db, logger: logger}
}

func (s *ConceptService) GetByMasterID(ctx context.Context, masterID int64) (*ProductConcept, error) {
	var c ProductConcept
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *ConceptService) Upsert(ctx context.Context, c *ProductConcept) error {
	existing, err := s.GetByMasterID(ctx, c.ProductMasterID)
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(c).Error
	}
	if err != nil {
		return err
	}
	c.ID = existing.ID
	return s.db.WithContext(ctx).Model(c).Updates(map[string]interface{}{
		"brief":           c.Brief,
		"target_customer": c.TargetCustomer,
		"pain_point":      c.PainPoint,
		"market_research": c.MarketResearch,
		"competitor_info": c.CompetitorInfo,
		"design_source":   c.DesignSource,
		"attachment_urls": c.AttachmentURLs,
		"status":          c.Status,
	}).Error
}

func (s *ConceptService) Delete(ctx context.Context, masterID int64) error {
	return s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Delete(&ProductConcept{}).Error
}
