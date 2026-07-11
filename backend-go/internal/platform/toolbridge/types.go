// Package toolbridge provides a driver abstraction for collecting structured
// page/product data from external URLs. It routes fetch requests to the best
// available driver (e.g., a browser extension, Playwright) with automatic
// fallback on failure.
package toolbridge

import (
	"context"
	"encoding/json"
	"time"
)

type ownerUserIDContextKey struct{}

// WithOwnerUserID binds a read-only browser collection request to the
// authenticated Owner whose extension is allowed to receive it.
func WithOwnerUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, ownerUserIDContextKey{}, userID)
}

// OwnerUserIDFromContext returns the authenticated Owner target for a tool call.
func OwnerUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(ownerUserIDContextKey{}).(int64)
	return userID, ok && userID > 0
}

// PageData holds structured product data scraped from a source URL.
type PageData struct {
	SourceURL   string   `json:"source_url"`
	Title       string   `json:"title"`
	PriceCNY    float64  `json:"price_cny"`
	PriceMinCNY *float64 `json:"price_min_cny,omitempty"`
	PriceMaxCNY *float64 `json:"price_max_cny,omitempty"`
	MOQ         int      `json:"moq"`
	Images      []string `json:"images,omitempty"`

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

// ListItemData is one product opportunity discovered on a marketplace or
// supplier search/list page. Detail collection happens in a separate step.
type ListItemData struct {
	Title      string `json:"title"`
	PriceRange string `json:"price_range"`
	DetailURL  string `json:"detail_url"`
	ImageURL   string `json:"image_url,omitempty"`
	RawText    string `json:"raw_text,omitempty"`
	RawHTML    string `json:"raw_html,omitempty"`
}

// ListPageData is the evidence returned from an automatically collected
// search/list page.
type ListPageData struct {
	PageURL     string          `json:"page_url"`
	CollectedAt time.Time       `json:"collected_at"`
	Items       []ListItemData  `json:"items"`
	Driver      string          `json:"driver"`
	RawData     json.RawMessage `json:"raw_data,omitempty"`
}

// UnmarshalJSON handles both content-script field names (price_1688, price_min, etc.)
// and canonical toolbridge field names (price_cny, price_min_cny, etc.).
// ponytail: naive map lookup for aliases — extend as new field names appear
func (p *PageData) UnmarshalJSON(data []byte) error {
	type alias PageData // prevent recursion
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = PageData(raw)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil // partial data still useful
	}
	if raw.PriceCNY == 0 {
		if v, ok := m["price_1688"]; ok {
			var f float64
			if json.Unmarshal(v, &f) == nil && f > 0 {
				p.PriceCNY = f
			}
		}
	}
	if raw.MOQ == 0 {
		if v, ok := m["min_order_qty"]; ok {
			var i int
			if json.Unmarshal(v, &i) == nil && i > 0 {
				p.MOQ = i
			}
		}
	}
	if raw.WeightKg == nil {
		if v, ok := m["package_weight_kg"]; ok {
			var f float64
			if json.Unmarshal(v, &f) == nil && f > 0 {
				p.WeightKg = &f
			}
		}
	}
	return nil
}

// SpecVariant describes a specific variant of a product.
type SpecVariant struct {
	Spec  string  `json:"spec"`
	Price float64 `json:"price"`
	Stock int     `json:"stock,omitempty"`
}
