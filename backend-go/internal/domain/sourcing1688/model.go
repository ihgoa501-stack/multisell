package sourcing1688

import (
	"encoding/json"
	"time"
)

// Sourcing1688Product maps to "sourcing_1688_product".
// Column definitions match migration 000001_init_schema.up.sql.
type Sourcing1688Product struct {
	ID              int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourceURL       string           `gorm:"column:source_url;uniqueIndex;not null" json:"source_url"`
	Title           *string          `gorm:"column:title" json:"title,omitempty"`
	Price           *float64         `gorm:"column:price;type:numeric(10,2)" json:"price,omitempty"`
	MOQ             int              `gorm:"column:moq;default:1" json:"moq"`
	SupplierName    string           `gorm:"column:supplier_name" json:"supplier_name"`
	ShopURL         *string          `gorm:"column:shop_url" json:"shop_url,omitempty"`
	ShopLocation    *string          `gorm:"column:shop_location" json:"shop_location,omitempty"`
	Images          *json.RawMessage `gorm:"column:images;type:jsonb" json:"images,omitempty"`
	Attributes      *json.RawMessage `gorm:"column:attributes;type:jsonb" json:"attributes,omitempty"`
	SkuVariants     *json.RawMessage `gorm:"column:sku_variants;type:jsonb" json:"sku_variants,omitempty"`
	Description     *string          `gorm:"column:description" json:"description,omitempty"`
	PackageLengthCm *float64         `gorm:"column:package_length_cm;type:numeric(10,2)" json:"package_length_cm,omitempty"`
	PackageWidthCm  *float64         `gorm:"column:package_width_cm;type:numeric(10,2)" json:"package_width_cm,omitempty"`
	PackageHeightCm *float64         `gorm:"column:package_height_cm;type:numeric(10,2)" json:"package_height_cm,omitempty"`
	PackageWeightKg *float64         `gorm:"column:package_weight_kg;type:numeric(10,2)" json:"package_weight_kg,omitempty"`
	RawData         *json.RawMessage `gorm:"column:raw_data;type:jsonb" json:"raw_data,omitempty"`
	Status          string           `gorm:"column:status;default:collected" json:"status"`
	ProductID       *int64           `gorm:"column:product_id" json:"product_id,omitempty"`
	SupplierID      *int64           `gorm:"column:supplier_id" json:"supplier_id,omitempty"`
	CollectedBy     *string          `gorm:"column:collected_by" json:"collected_by,omitempty"`
	ImportedBy      *string          `gorm:"column:imported_by" json:"imported_by,omitempty"`
	ImportedAt      *time.Time       `gorm:"column:imported_at" json:"imported_at,omitempty"`
	CreatedAt       time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Sourcing1688Product) TableName() string { return "sourcing_1688_product" }

// CreateInput is the payload for POST /sourcing1688.
type CreateInput struct {
	SourceURL    string           `json:"source_url" binding:"required"`
	Title        *string          `json:"title"`
	Price        *float64         `json:"price"`
	MOQ          *int             `json:"moq"`
	SupplierName string           `json:"supplier_name"`
	ShopURL      *string          `json:"shop_url"`
	ShopLocation *string          `json:"shop_location"`
	Description  *string          `json:"description"`
	ProductID    *int64           `json:"product_id"`
	SupplierID   *int64           `json:"supplier_id"`
	CollectedBy  *string          `json:"collected_by"`
	Status       string           `json:"status"`
	RawData      *json.RawMessage `json:"raw_data"`
}

// UpdateInput allows partial updates.
type UpdateInput struct {
	SourceURL    *string          `json:"source_url"`
	Title        *string          `json:"title"`
	Price        *float64         `json:"price"`
	MOQ          *int             `json:"moq"`
	SupplierName *string          `json:"supplier_name"`
	ShopURL      *string          `json:"shop_url"`
	ShopLocation *string          `json:"shop_location"`
	Description  *string          `json:"description"`
	ProductID    *int64           `json:"product_id"`
	SupplierID   *int64           `json:"supplier_id"`
	CollectedBy  *string          `json:"collected_by"`
	Status       *string          `json:"status"`
	RawData      *json.RawMessage `json:"raw_data"`
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
