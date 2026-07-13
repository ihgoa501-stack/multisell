package listing

import (
	"encoding/json"
	"time"
)

// ProductListing maps to the "product_listing" table — per-platform publish record.
type ProductListing struct {
	ID                int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID         int64           `gorm:"column:product_id;not null" json:"product_id"`
	PlatformID        int64           `gorm:"column:platform_id;not null" json:"platform_id"`
	PlatformProductID string          `gorm:"column:platform_product_id" json:"platform_product_id"`
	PlatformSKU       string          `gorm:"column:platform_sku" json:"platform_sku"`
	Status            string          `gorm:"column:status;default:draft" json:"status"`
	PlatformURL       string          `gorm:"column:platform_url" json:"platform_url"`
	SyncMessage       string          `gorm:"column:sync_message" json:"sync_message"`
	PublishedData     json.RawMessage `gorm:"column:published_data;type:jsonb" json:"published_data,omitempty"`
	LastSyncAt        *time.Time      `gorm:"column:last_sync_at" json:"last_sync_at,omitempty"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName explicitly sets the table name (no AutoMigrate).
func (ProductListing) TableName() string { return "product_listing" }

// CreateListingInput is the payload for creating a listing.
type CreateListingInput struct {
	ProductID         int64           `json:"product_id" binding:"required"`
	PlatformID        int64           `json:"platform_id" binding:"required"`
	PlatformProductID string          `json:"platform_product_id"`
	PlatformSKU       string          `json:"platform_sku"`
	Status            string          `json:"status"`
	PlatformURL       string          `json:"platform_url"`
	PublishedData     json.RawMessage `json:"published_data"`
}

// UpdateListingInput is the payload for updating a listing.
type UpdateListingInput struct {
	PlatformProductID *string          `json:"platform_product_id"`
	PlatformSKU       *string          `json:"platform_sku"`
	Status            *string          `json:"status"`
	PlatformURL       *string          `json:"platform_url"`
	SyncMessage       *string          `json:"sync_message"`
	PublishedData     *json.RawMessage `json:"published_data"`
}

// PublishProductInput is the body for POST /listing/products/:product_id/publish/:platform_id.
type PublishProductInput struct {
	ExternalID string `json:"external_id"`
	ListingURL string `json:"listing_url"`
	Status     string `json:"status"`
}

// CreateTasksFromDecisionsInput is the body for POST /listing/listing-tasks/from-decisions.
type CreateTasksFromDecisionsInput struct {
	DecisionIDs []int64 `json:"decision_ids" binding:"required"`
}

// CancelTaskInput is the body for POST /listing/listing-tasks/:task_id/cancel.
type CancelTaskInput struct {
	Reason string `json:"reason"`
}

// ListingSuggestion is a structured suggestion generated for a candidate product.
type ListingSuggestion struct {
	CandidateID    uint            `json:"candidate_id"`
	Title          string          `json:"title"`
	CategoryPath   string          `json:"category_path"`
	SuggestedPrice float64         `json:"suggested_price"`
	SuggestedStock int             `json:"suggested_stock"`
	PlatformFields []PlatformField `json:"platform_fields"`
	RiskLevel      string          `json:"risk_level"`
	CreatedAt      time.Time       `json:"created_at"`
}

// PlatformField is a platform-specific field in a listing suggestion.
type PlatformField struct {
	Platform  string `json:"platform"`
	FieldName string `json:"field_name"`
	Value     string `json:"value"`
}

// SuggestInput is the payload for generating a listing suggestion.
type SuggestInput struct {
	CandidateID uint `json:"candidate_id" binding:"required"`
}
