// Package toolbridge provides a driver abstraction for collecting structured
// page/product data from external URLs. It routes fetch requests to the best
// available driver (e.g., a browser extension, Playwright) with automatic
// fallback on failure.
package toolbridge

import (
	"encoding/json"
	"time"
)

// PageData holds structured product data scraped from a source URL.
type PageData struct {
	SourceURL    string    `json:"source_url"`
	Title        string    `json:"title"`
	PriceCNY     float64   `json:"price_cny"`
	PriceMinCNY  *float64  `json:"price_min_cny,omitempty"`
	PriceMaxCNY  *float64  `json:"price_max_cny,omitempty"`
	MOQ          int       `json:"moq"`
	Images       []string  `json:"images,omitempty"`

	SpecVariants []SpecVariant `json:"spec_variants,omitempty"`

	SupplierName  string `json:"supplier_name"`
	SupplierScore *int   `json:"supplier_score,omitempty"`

	WeightKg        *float64 `json:"weight_kg,omitempty"`
	PackageLengthCm *float64 `json:"package_length_cm,omitempty"`
	PackageWidthCm  *float64 `json:"package_width_cm,omitempty"`
	PackageHeightCm *float64 `json:"package_height_cm,omitempty"`

	Description string          `json:"description,omitempty"`
	RawData     json.RawMessage `json:"raw_data,omitempty"`
	CollectedAt time.Time       `json:"collected_at"`
	Driver      string          `json:"driver"`
}

// SpecVariant describes a specific variant of a product.
type SpecVariant struct {
	Spec  string  `json:"spec"`
	Price float64 `json:"price"`
	Stock int     `json:"stock,omitempty"`
}
