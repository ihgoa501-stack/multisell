package inventory

import (
	"time"

	"gorm.io/gorm"
)

// BinLocation represents a warehouse bin/location for physical inventory.
type BinLocation struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Warehouse    string `gorm:"column:warehouse;not null" json:"warehouse"`
	LocationCode string `gorm:"column:location_code;not null;uniqueIndex" json:"location_code"`
	SkuID        *int64 `gorm:"column:sku_id" json:"sku_id,omitempty"`
	Capacity     int    `gorm:"column:capacity;default:0" json:"capacity"`
	Used         int    `gorm:"column:used;default:0" json:"used"`
	Status       string `gorm:"column:status;default:available" json:"status"` // available, occupied, reserved, maintenance
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (BinLocation) TableName() string { return "bin_location" }

// AssignLocation assigns a SKU to a bin location.
func (s *Service) AssignLocation(skuID int64, locationID int64) (*BinLocation, error) {
	var loc BinLocation
	if err := s.db.First(&loc, locationID).Error; err != nil {
		return nil, err
	}
	if loc.Status != "available" {
		return nil, gorm.ErrRecordNotFound
	}
	loc.SkuID = &skuID
	loc.Status = "occupied"
	if err := s.db.Save(&loc).Error; err != nil {
		return nil, err
	}
	return &loc, nil
}

// ReleaseLocation clears a bin location assignment.
func (s *Service) ReleaseLocation(locationID int64) error {
	return s.db.Model(&BinLocation{}).Where("id = ?", locationID).
		Updates(map[string]interface{}{"sku_id": nil, "status": "available"}).Error
}

// ListLocations returns bin locations filtered by warehouse.
func (s *Service) ListLocations(warehouse string, page, size int) ([]BinLocation, int64, error) {
	var locs []BinLocation
	var total int64
	q := s.db.Model(&BinLocation{})
	if warehouse != "" {
		q = q.Where("warehouse = ?", warehouse)
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
	if err := q.Offset(offset).Limit(size).Order("location_code ASC").Find(&locs).Error; err != nil {
		return nil, 0, err
	}
	return locs, total, nil
}
