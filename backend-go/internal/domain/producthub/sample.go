package producthub

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SampleRequest maps to the "sample_request" table — V1 has iteration info inline.
type SampleRequest struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id,omitempty"`
	ProductMasterID int64           `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	SupplierOfferID *int64          `gorm:"column:supplier_offer_id" json:"supplier_offer_id,omitempty"`
	SupplierID      int64           `gorm:"column:supplier_id;not null" json:"supplier_id"`
	Quantity        int             `gorm:"column:quantity" json:"quantity,omitempty"`
	Requirements    string          `gorm:"column:requirements;type:text" json:"requirements,omitempty"`
	RequestedAt     time.Time       `gorm:"column:requested_at;autoCreateTime" json:"requested_at"`
	DueAt           *time.Time      `gorm:"column:due_at" json:"due_at,omitempty"`
	Status          string          `gorm:"column:status;size:32;default:pending" json:"status,omitempty"`
	IterationNo     int             `gorm:"column:iteration_no;default:0" json:"iteration_no"`
	ReceivedAt      *time.Time      `gorm:"column:received_at" json:"received_at,omitempty"`
	Evaluation      string          `gorm:"column:evaluation;type:text" json:"evaluation,omitempty"`
	QualityScore    float64         `gorm:"column:quality_score" json:"quality_score,omitempty"`
	Decision        string          `gorm:"column:decision;size:32" json:"decision,omitempty"`
	ImageURLs       json.RawMessage `gorm:"column:image_urls;type:jsonb" json:"image_urls,omitempty"`
	CreatedBy       int64           `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SampleRequest) TableName() string { return "sample_request" }

// SampleService handles sample requests.
type SampleService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewSampleService(db *gorm.DB, logger *zap.Logger) *SampleService {
	return &SampleService{db: db, logger: logger}
}

func (s *SampleService) ListByMaster(ctx context.Context, masterID int64) ([]SampleRequest, error) {
	var items []SampleRequest
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SampleService) GetLatestByMaster(ctx context.Context, masterID int64) (*SampleRequest, error) {
	var sr SampleRequest
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("id DESC").First(&sr).Error; err != nil {
		return nil, err
	}
	return &sr, nil
}

func (s *SampleService) Create(ctx context.Context, sr *SampleRequest) error {
	return s.db.WithContext(ctx).Create(sr).Error
}

func (s *SampleService) RecordEvaluation(ctx context.Context, id int64, eval string, score float64, decision string, imageURLs json.RawMessage) error {
	return s.db.WithContext(ctx).Model(&SampleRequest{}).Where("id = ?", id).Updates(map[string]interface{}{
		"received_at":   time.Now(),
		"evaluation":    eval,
		"quality_score": score,
		"decision":      decision,
		"image_urls":    imageURLs,
		"status":        "evaluated",
	}).Error
}
