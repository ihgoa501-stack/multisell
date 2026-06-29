package producthub

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SupplierOffer maps to the "supplier_offer" table.
type SupplierOffer struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id,omitempty"`
	SupplierID      int64     `gorm:"column:supplier_id;index;not null" json:"supplier_id"`
	ProductMasterID int64     `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	OfferType       string    `gorm:"column:offer_type;size:32" json:"offer_type,omitempty"`
	UnitCost        float64   `gorm:"column:unit_cost" json:"unit_cost,omitempty"`
	Currency        string    `gorm:"column:currency;size:8;default:CNY" json:"currency,omitempty"`
	MOQ             int       `gorm:"column:moq" json:"moq,omitempty"`
	LeadTimeDays    int       `gorm:"column:lead_time_days" json:"lead_time_days,omitempty"`
	Incoterm        string    `gorm:"column:incoterm;size:32" json:"incoterm,omitempty"`
	IsPreferred     bool      `gorm:"column:is_preferred;default:false" json:"is_preferred"`
	ValidFrom       time.Time `gorm:"column:valid_from" json:"valid_from,omitempty"`
	ValidTo         time.Time `gorm:"column:valid_to" json:"valid_to,omitempty"`
	Notes           string    `gorm:"column:notes;type:text" json:"notes,omitempty"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SupplierOffer) TableName() string { return "supplier_offer" }

// SupplierOfferService handles supplier offers.
type SupplierOfferService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewSupplierOfferService(db *gorm.DB, logger *zap.Logger) *SupplierOfferService {
	return &SupplierOfferService{db: db, logger: logger}
}

func (s *SupplierOfferService) ListByMaster(ctx context.Context, masterID int64) ([]SupplierOffer, error) {
	var items []SupplierOffer
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("is_preferred DESC, unit_cost ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SupplierOfferService) GetByID(ctx context.Context, id int64) (*SupplierOffer, error) {
	var o SupplierOffer
	if err := s.db.WithContext(ctx).First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *SupplierOfferService) Create(ctx context.Context, o *SupplierOffer) error {
	return s.db.WithContext(ctx).Create(o).Error
}

func (s *SupplierOfferService) Update(ctx context.Context, o *SupplierOffer) error {
	updates := map[string]interface{}{}
	if o.UnitCost > 0 {
		updates["unit_cost"] = o.UnitCost
	}
	if o.Currency != "" {
		updates["currency"] = o.Currency
	}
	if o.MOQ > 0 {
		updates["moq"] = o.MOQ
	}
	if o.LeadTimeDays > 0 {
		updates["lead_time_days"] = o.LeadTimeDays
	}
	if o.Incoterm != "" {
		updates["incoterm"] = o.Incoterm
	}
	updates["is_preferred"] = o.IsPreferred
	return s.db.WithContext(ctx).Model(&SupplierOffer{}).Where("id = ?", o.ID).Updates(updates).Error
}

func (s *SupplierOfferService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&SupplierOffer{}, id).Error
}
