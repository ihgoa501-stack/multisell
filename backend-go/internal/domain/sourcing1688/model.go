package sourcing1688

import (
	"encoding/json"
	"time"
)

// Sourcing1688Product maps to "sourcing_1688_product".
// Column definitions match migration 000001_init_schema.up.sql.
type Sourcing1688Product struct {
	ID                       int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourceURL                string           `gorm:"column:source_url;uniqueIndex;not null" json:"source_url"`
	SourceOfferID            string           `gorm:"column:source_offer_id" json:"source_offer_id,omitempty"`
	SourceProductFingerprint string           `gorm:"column:source_product_fingerprint" json:"source_product_fingerprint,omitempty"`
	SupplierBusinessID       string           `gorm:"column:supplier_business_id" json:"supplier_business_id,omitempty"`
	Title                    *string          `gorm:"column:title" json:"title,omitempty"`
	Price                    *float64         `gorm:"column:price;type:numeric(10,2)" json:"price,omitempty"`
	MOQ                      int              `gorm:"column:moq;default:1" json:"moq"`
	SupplierName             string           `gorm:"column:supplier_name" json:"supplier_name"`
	ShopURL                  *string          `gorm:"column:shop_url" json:"shop_url,omitempty"`
	ShopLocation             *string          `gorm:"column:shop_location" json:"shop_location,omitempty"`
	Images                   *json.RawMessage `gorm:"column:images;type:jsonb" json:"images,omitempty"`
	Attributes               *json.RawMessage `gorm:"column:attributes;type:jsonb" json:"attributes,omitempty"`
	SkuVariants              *json.RawMessage `gorm:"column:sku_variants;type:jsonb" json:"sku_variants,omitempty"`
	Description              *string          `gorm:"column:description" json:"description,omitempty"`
	PackageLengthCm          *float64         `gorm:"column:package_length_cm;type:numeric(10,2)" json:"package_length_cm,omitempty"`
	PackageWidthCm           *float64         `gorm:"column:package_width_cm;type:numeric(10,2)" json:"package_width_cm,omitempty"`
	PackageHeightCm          *float64         `gorm:"column:package_height_cm;type:numeric(10,2)" json:"package_height_cm,omitempty"`
	PackageWeightKg          *float64         `gorm:"column:package_weight_kg;type:numeric(10,2)" json:"package_weight_kg,omitempty"`
	RawData                  *json.RawMessage `gorm:"column:raw_data;type:jsonb" json:"raw_data,omitempty"`
	Status                   string           `gorm:"column:status;default:collected" json:"status"`
	ProductID                *int64           `gorm:"column:product_id" json:"product_id,omitempty"`
	SupplierID               *int64           `gorm:"column:supplier_id" json:"supplier_id,omitempty"`
	CollectedBy              *string          `gorm:"column:collected_by" json:"collected_by,omitempty"`
	ImportedBy               *string          `gorm:"column:imported_by" json:"imported_by,omitempty"`
	ImportedAt               *time.Time       `gorm:"column:imported_at" json:"imported_at,omitempty"`
	DemandCaseID             *int64           `gorm:"column:demand_case_id" json:"demand_case_id,omitempty"`
	ExperimentID             *string          `gorm:"column:experiment_id" json:"experiment_id,omitempty"`
	SnapshotID               *int64           `gorm:"column:snapshot_id" json:"snapshot_id,omitempty"`
	ReviewedBy               *int64           `gorm:"column:reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt               *time.Time       `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	ReviewNotes              string           `gorm:"column:review_notes" json:"review_notes,omitempty"`
	LifecycleStatus          string           `gorm:"column:lifecycle_status;default:pending_review" json:"lifecycle_status"`
	LifecycleActorID         *int64           `gorm:"column:lifecycle_actor_id" json:"lifecycle_actor_id,omitempty"`
	LifecycleReason          string           `gorm:"column:lifecycle_reason" json:"lifecycle_reason,omitempty"`
	LifecycleUpdatedAt       *time.Time       `gorm:"column:lifecycle_updated_at" json:"lifecycle_updated_at,omitempty"`
	CreatedAt                time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// Sourcing1688Snapshot is an immutable copy of the exact source payload used
// for an Owner decision. Normal update flows never modify snapshot rows.
type Sourcing1688Snapshot struct {
	ID                         int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourcingProductID          int64           `gorm:"column:sourcing_product_id;not null;uniqueIndex:ux_sourcing_snapshot_hash" json:"sourcing_product_id"`
	SourceURL                  string          `gorm:"column:source_url;not null" json:"source_url"`
	CollectedAt                time.Time       `gorm:"column:collected_at;not null" json:"collected_at"`
	CollectedBy                int64           `gorm:"column:collected_by;not null" json:"collected_by"`
	Driver                     string          `gorm:"column:driver;not null" json:"driver"`
	ParserVersion              string          `gorm:"column:parser_version;not null" json:"parser_version"`
	RawPayload                 json.RawMessage `gorm:"column:raw_payload;type:jsonb;not null" json:"raw_payload"`
	RawSHA256                  string          `gorm:"column:raw_sha256;size:64;not null;uniqueIndex:ux_sourcing_snapshot_hash" json:"raw_sha256"`
	ObservedTitle              *string         `gorm:"column:observed_title" json:"observed_title,omitempty"`
	ObservedPrice              *float64        `gorm:"column:observed_price" json:"observed_price,omitempty"`
	ObservedMOQ                int             `gorm:"column:observed_moq" json:"observed_moq"`
	ObservedSupplier           string          `gorm:"column:observed_supplier" json:"observed_supplier,omitempty"`
	ObservedSupplierBusinessID string          `gorm:"column:observed_supplier_business_id" json:"observed_supplier_business_id,omitempty"`
	ProductFingerprint         string          `gorm:"column:product_fingerprint" json:"product_fingerprint,omitempty"`
	CreatedAt                  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Sourcing1688Snapshot) TableName() string { return "sourcing_1688_snapshot" }

type CaptureInput struct {
	DemandCaseID       int64           `json:"demand_case_id" binding:"required"`
	ExperimentID       string          `json:"experiment_id" binding:"required"`
	SourceURL          string          `json:"source_url" binding:"required"`
	CollectedAt        time.Time       `json:"collected_at" binding:"required"`
	CollectedBy        int64           `json:"collected_by"`
	Driver             string          `json:"driver" binding:"required"`
	ParserVersion      string          `json:"parser_version" binding:"required"`
	RawPayload         json.RawMessage `json:"raw_payload" binding:"required"`
	Title              *string         `json:"title"`
	Price              *float64        `json:"price"`
	MOQ                *int            `json:"moq"`
	SupplierName       string          `json:"supplier_name"`
	SupplierBusinessID string          `json:"supplier_business_id" binding:"required"`
	Images             json.RawMessage `json:"images"`
	SkuVariants        json.RawMessage `json:"sku_variants"`
}

type ReviewInput struct {
	ReviewedBy int64  `json:"reviewed_by"`
	Notes      string `json:"notes" binding:"required"`
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
