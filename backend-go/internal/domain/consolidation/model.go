package consolidation

import (
	"time"
)

// ---------------------------------------------------------------------------
// Status types
// ---------------------------------------------------------------------------

// ConsolidationGroupStatus represents the lifecycle status of a consolidation group.
type ConsolidationGroupStatus string

const (
	GroupStatusOpen        ConsolidationGroupStatus = "open"
	GroupStatusNegotiating ConsolidationGroupStatus = "negotiating"
	GroupStatusClosed      ConsolidationGroupStatus = "closed"
	GroupStatusShipped     ConsolidationGroupStatus = "shipped"
)

// ConsolidationItemStatus represents the status of a single item within a group.
type ConsolidationItemStatus string

const (
	ItemStatusPending   ConsolidationItemStatus = "pending"
	ItemStatusConfirmed ConsolidationItemStatus = "confirmed"
	ItemStatusRemoved   ConsolidationItemStatus = "removed"
)

// ---------------------------------------------------------------------------
// ConsolidationGroup — 集单组
// ---------------------------------------------------------------------------

// ConsolidationGroup groups multiple shipment items together for bulk rate negotiation.
type ConsolidationGroup struct {
	ID             int64                      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Status         ConsolidationGroupStatus   `gorm:"column:status;not null;default:open" json:"status"`
	TotalWeightKg  float64                    `gorm:"column:total_weight_kg;type:decimal(12,2);default:0" json:"total_weight_kg"`
	TotalVolumeM3  float64                    `gorm:"column:total_volume_m3;type:decimal(12,2);default:0" json:"total_volume_m3"`
	Destination    string                     `gorm:"column:destination;not null" json:"destination"`
	CarrierID      *int64                     `gorm:"column:carrier_id" json:"carrier_id,omitempty"`
	NegotiatedRate *float64                   `gorm:"column:negotiated_rate;type:decimal(12,2)" json:"negotiated_rate,omitempty"`
	DiscountRate   float64                    `gorm:"column:discount_rate;type:decimal(5,2);default:0" json:"discount_rate"`
	CreatedAt      time.Time                  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time                  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (ConsolidationGroup) TableName() string { return "consolidation_group" }

// ---------------------------------------------------------------------------
// ConsolidationItem — 集单中的单个商品
// ---------------------------------------------------------------------------

// ConsolidationItem represents a single item within a consolidation group.
type ConsolidationItem struct {
	ID          int64                    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupID     int64                    `gorm:"column:group_id;not null;index" json:"group_id"`
	SkuID       int64                    `gorm:"column:sku_id;not null" json:"sku_id"`
	WeightKg    float64                  `gorm:"column:weight_kg;type:decimal(12,2);not null" json:"weight_kg"`
	VolumeM3    float64                  `gorm:"column:volume_m3;type:decimal(12,2);default:0" json:"volume_m3"`
	Destination string                   `gorm:"column:destination;not null" json:"destination"`
	Status      ConsolidationItemStatus  `gorm:"column:status;not null;default:pending" json:"status"`
	CreatedAt   time.Time                `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time                `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (ConsolidationItem) TableName() string { return "consolidation_item" }

// ---------------------------------------------------------------------------
// Request / Response DTOs
// ---------------------------------------------------------------------------

// CreateGroupRequest is the payload for POST /consolidation/groups.
type CreateGroupRequest struct {
	Destination string  `json:"destination" binding:"required"`
	TimeWindowH int     `json:"time_window_h"`           // time window in hours (default 72 = 3 days)
	CarrierID   *int64  `json:"carrier_id,omitempty"`
}

// AddItemRequest is the payload for POST /consolidation/groups/:groupId/items.
type AddItemRequest struct {
	SkuID       int64   `json:"sku_id" binding:"required"`
	WeightKg    float64 `json:"weight_kg" binding:"required"`
	VolumeM3    float64 `json:"volume_m3"`
	Destination string  `json:"destination" binding:"required"`
}

// NegotiateResult is the response for POST /consolidation/groups/:groupId/negotiate.
type NegotiateResult struct {
	Group          ConsolidationGroup `json:"group"`
	DiscountRate   float64            `json:"discount_rate"`
	TotalWeightKg  float64            `json:"total_weight_kg"`
	DiscountLabel  string             `json:"discount_label"`
}
