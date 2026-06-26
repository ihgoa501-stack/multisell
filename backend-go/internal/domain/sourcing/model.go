package sourcing

import (
	"encoding/json"
	"time"
)

// PageData is the unified structured product data returned by any ToolBridge driver.
type PageData struct {
	// Metadata
	SourceURL    string    `json:"source_url"`
	CollectedAt  time.Time `json:"collected_at"`
	Driver       string    `json:"driver"` // "plugin", "playwright", "api1688"

	// Core fields
	Title    string  `json:"title"`
	Price    float64 `json:"price_1688"`
	PriceMin *float64 `json:"price_min,omitempty"`
	PriceMax *float64 `json:"price_max,omitempty"`
	Currency string  `json:"currency"`
	MOQ      int     `json:"min_order_qty"`

	// Images
	Images     []string `json:"images"`
	ImageFirst string   `json:"image_first,omitempty"`

	// Specs
	SpecVariants []SpecVariant `json:"spec_variants,omitempty"`

	// Supplier
	SupplierName  string `json:"supplier_name"`
	SupplierID    string `json:"supplier_id_1688"`
	SupplierScore *int   `json:"supplier_score,omitempty"`

	// Description & attributes
	Description string            `json:"description,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`

	// Logistics
	PackageWeight *float64 `json:"package_weight_kg,omitempty"`
	PackageLength *float64 `json:"package_length_cm,omitempty"`
	PackageWidth  *float64 `json:"package_width_cm,omitempty"`
	PackageHeight *float64 `json:"package_height_cm,omitempty"`
	FreightCNY    *float64 `json:"freight_cny,omitempty"`

	// Raw data
	RawHTML *string          `json:"raw_html,omitempty"`
	RawJSON *json.RawMessage `json:"raw_json,omitempty"`
}

// SpecVariant represents a product specification variant.
type SpecVariant struct {
	Spec     string  `json:"spec"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	ImageURL string  `json:"image_url,omitempty"`
}

// FetchRequest is the payload for POST /sourcing/fetch.
type FetchRequest struct {
	URL string `json:"url" binding:"required"`
}

// Recommendation represents a scored product recommendation.
type Recommendation struct {
	ID              int64  `json:"id"`
	SourceURL       string `json:"source_url"`
	Title           string `json:"title"`
	SupplierName    string `json:"supplier_name"`
	Price           float64 `json:"price"`
	Score           int    `json:"score"`
	Status          string `json:"status"`
	ProductID1688   string `json:"product_id_1688"`
	ImageURL        string `json:"image_url,omitempty"`
	RecommendReason string `json:"recommend_reason,omitempty"`
	CreatedAt       string `json:"created_at"`
}
