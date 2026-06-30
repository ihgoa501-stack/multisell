package producthub

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MasterService handles product_master business logic.
type MasterService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewMasterService creates a MasterService.
func NewMasterService(db *gorm.DB, logger *zap.Logger) *MasterService {
	return &MasterService{db: db, logger: logger}
}

// List returns paginated product masters with optional lifecycle filter.
func (s *MasterService) List(ctx context.Context, page, size int, lifecycleStatus string) ([]ProductMaster, int64, error) {
	q := s.db.WithContext(ctx).Model(&ProductMaster{})
	if lifecycleStatus != "" {
		if !IsValidLifecycleStatus(lifecycleStatus) {
			return nil, 0, fmt.Errorf("invalid lifecycle status: %s", lifecycleStatus)
		}
		q = q.Where("lifecycle_status = ?", lifecycleStatus)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ProductMaster
	if err := q.Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single product master by ID.
func (s *MasterService) GetByID(ctx context.Context, id int64) (*ProductMaster, error) {
	var p ProductMaster
	if err := s.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Create inserts a new product master.
func (s *MasterService) Create(ctx context.Context, p *ProductMaster) error {
	if p.BusinessModel != "" && !IsValidBusinessModel(p.BusinessModel) {
		return fmt.Errorf("invalid business model: %s", p.BusinessModel)
	}
	if !IsValidLifecycleStatus(p.LifecycleStatus) {
		p.LifecycleStatus = LifecycleIdea
	}
	return s.db.WithContext(ctx).Create(p).Error
}

// Update updates an existing product master.
func (s *MasterService) Update(ctx context.Context, p *ProductMaster) error {
	updates := map[string]interface{}{}
	if p.Name != "" {
		updates["name"] = p.Name
	}
	updates["brand_id"] = p.BrandID
	updates["category_id"] = p.CategoryID
	if p.BusinessModel != "" {
		if !IsValidBusinessModel(p.BusinessModel) {
			return fmt.Errorf("invalid business model: %s", p.BusinessModel)
		}
		updates["business_model"] = p.BusinessModel
	}
	if p.LifecycleStatus != "" {
		if !IsValidLifecycleStatus(p.LifecycleStatus) {
			return fmt.Errorf("invalid lifecycle status: %s", p.LifecycleStatus)
		}
		updates["lifecycle_status"] = p.LifecycleStatus
	}
	updates["owner_id"] = p.OwnerID
	if p.Description != "" {
		updates["description"] = p.Description
	}
	if p.TargetMarket != "" {
		updates["target_market"] = p.TargetMarket
	}
	if p.TargetPrice > 0 {
		updates["target_price"] = p.TargetPrice
	}
	if p.TargetMargin > 0 {
		updates["target_margin"] = p.TargetMargin
	}
	return s.db.WithContext(ctx).Model(&ProductMaster{}).Where("id = ?", p.ID).Updates(updates).Error
}

// Delete removes a product master.
func (s *MasterService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&ProductMaster{}, id).Error
}

// validTransitions defines allowed lifecycle state transitions.
var validTransitions = map[string][]string{
	LifecycleIdea:        {LifecycleResearching},
	LifecycleResearching: {LifecycleSampling, LifecycleIdea},
	LifecycleSampling:    {LifecycleApproved},
	LifecycleApproved:    {LifecycleCosted},
	LifecycleCosted:      {LifecycleReadyToList},
	LifecycleReadyToList: {LifecycleListed},
	LifecycleListed:      {LifecycleActive},
	LifecycleActive:      {LifecycleSunset},
	LifecycleSunset:      {LifecycleArchived, LifecycleIdea},
	LifecycleArchived:    {},
}

// TransitionLifecycle advances the product lifecycle status.
func (s *MasterService) TransitionLifecycle(ctx context.Context, id int64, newStatus string) (*ProductMaster, error) {
	if !IsValidLifecycleStatus(newStatus) {
		return nil, fmt.Errorf("invalid lifecycle status: %s", newStatus)
	}
	p, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	allowed, ok := validTransitions[p.LifecycleStatus]
	if !ok {
		return nil, fmt.Errorf("unknown current status: %s", p.LifecycleStatus)
	}
	if !contains(allowed, newStatus) {
		return nil, fmt.Errorf("cannot transition from %s to %s", p.LifecycleStatus, newStatus)
	}
	if err := s.db.WithContext(ctx).Model(p).Update("lifecycle_status", newStatus).Error; err != nil {
		return nil, err
	}
	p.LifecycleStatus = newStatus
	return p, nil
}

// contains checks if a string is in a slice.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
