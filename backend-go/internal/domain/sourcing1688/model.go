package sourcing1688

import (
	"encoding/json"
	"time"
)

// Sourcing1688Product maps to "sourcing_1688_product".
type Sourcing1688Product struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID    *int64          `gorm:"column:product_id" json:"product_id,omitempty"`
	SourceURL    string          `gorm:"column:source_url" json:"source_url"`
	SupplierName string          `gorm:"column:supplier_name" json:"supplier_name"`
	SupplierID1688 string       `gorm:"column:supplier_id_1688" json:"supplier_id_1688"`
	Price1688    float64         `gorm:"column:price_1688;default:0" json:"price_1688"`
	MinOrderQty  int             `gorm:"column:min_order_qty;default:1" json:"min_order_qty"`
	ImageURL     string          `gorm:"column:image_url" json:"image_url"`
	SpecSummary  string          `gorm:"column:spec_summary" json:"spec_summary"`
	Status       string          `gorm:"column:status;default:pending" json:"status"`
	SourceData   json.RawMessage `gorm:"column:source_data;type:jsonb" json:"source_data,omitempty"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Sourcing1688Product) TableName() string { return "sourcing_1688_product" }

// CreateInput is the payload for POST /sourcing1688.
type CreateInput struct {
	ProductID      *int64          `json:"product_id"`
	SourceURL      string          `json:"source_url" binding:"required"`
	SupplierName   string          `json:"supplier_name"`
	SupplierID1688 string          `json:"supplier_id_1688"`
	Price1688      *float64        `json:"price_1688"`
	MinOrderQty    *int            `json:"min_order_qty"`
	ImageURL       string          `json:"image_url"`
	SpecSummary    string          `json:"spec_summary"`
	Status         string          `json:"status"`
	SourceData     json.RawMessage `json:"source_data"`
}

// UpdateInput allows partial updates.
type UpdateInput struct {
	ProductID      *int64           `json:"product_id"`
	SourceURL      *string          `json:"source_url"`
	SupplierName   *string          `json:"supplier_name"`
	SupplierID1688 *string          `json:"supplier_id_1688"`
	Price1688      *float64         `json:"price_1688"`
	MinOrderQty    *int             `json:"min_order_qty"`
	ImageURL       *string          `json:"image_url"`
	SpecSummary    *string          `json:"spec_summary"`
	Status         *string          `json:"status"`
	SourceData     *json.RawMessage `json:"source_data"`
}

// ListFilter captures query parameters.
type ListFilter struct {
	Search    string
	Status    string
	ProductID *int64
}

// ImportInput is the body for POST /sourcing1688/:id/import.
type ImportInput struct {
	ImportedBy string `json:"imported_by"`
}

// RejectInput is the body for POST /sourcing1688/:id/reject.
type RejectInput struct {
	RejectedBy string `json:"rejected_by"`
	Reason     string `json:"reason"`
}

// Summary is the aggregation payload.
type Summary struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
}
