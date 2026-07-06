package exceptions

import "time"

// Exception type constants for auto-detection.
const (
	TypeLossOrder        = "loss_order"
	TypeOutOfStock       = "out_of_stock"
	TypeLogisticsAbnormal = "logistics_abnormal"
	TypeFeeAbnormal      = "fee_abnormal"
)

// ExceptionItem maps to the `exception_item` table.
type ExceptionItem struct {
	ID                int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourceModule      string     `gorm:"column:source_module;size:50;not null" json:"source_module"`
	SourceType        string     `gorm:"column:source_type;size:50" json:"source_type"`
	SourceID          *int64     `gorm:"column:source_id" json:"source_id,omitempty"`
	Severity          string     `gorm:"column:severity;size:20;default:medium" json:"severity"`
	Status            string     `gorm:"column:status;size:20;default:open" json:"status"`
	Title             string     `gorm:"column:title;size:300;not null" json:"title"`
	Description       string     `gorm:"column:description;type:text" json:"description"`
	RecommendedAction string     `gorm:"column:recommended_action;size:500" json:"recommended_action"`
	AssignedTo        string     `gorm:"column:assigned_to;size:100" json:"assigned_to"`
	ResolvedAt        *time.Time `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy        string     `gorm:"column:resolved_by;size:100" json:"resolved_by"`
	Note              string     `gorm:"column:note;type:text" json:"note"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (ExceptionItem) TableName() string {
	return "exception_item"
}
