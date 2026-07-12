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
	RawHTML     string          `json:"raw_html,omitempty"`
	// RawResponse preserves the exact JSON bytes of the extension's data object.
	// It is transport evidence and must never be serialized back into that object.
	RawResponse json.RawMessage `json:"-"`
	CollectedAt time.Time       `json:"collected_at"`
	Driver      string          `json:"driver"`
	// ParserVersion and SupplierBusinessID must come from the collector response;
	// controlled evidence capture rejects responses that omit either identity.
	ParserVersion      string `json:"parser_version,omitempty"`
	SupplierBusinessID string `json:"supplier_business_id,omitempty"`
	// CollectionRequestID is assigned by the server-side plugin driver after a
	// matching authenticated extension response. The extension cannot choose it.
	CollectionRequestID string `json:"collection_request_id,omitempty"`
}

// UnmarshalJSON accepts both the canonical ToolBridge field names and the
// browser-extension protocol names. The latter predate this Go type and use
// price_1688/min_order_qty/package_* keys.
func (p *PageData) UnmarshalJSON(data []byte) error {
	type pageDataAlias PageData
	aux := struct {
		*pageDataAlias
		Price1688      *float64 `json:"price_1688"`
		MinOrderQty    *int     `json:"min_order_qty"`
		PackageWeight  *float64 `json:"package_weight_kg"`
		PackageLength  *float64 `json:"package_length_cm"`
		PackageWidth   *float64 `json:"package_width_cm"`
		PackageHeight  *float64 `json:"package_height_cm"`
		SupplierID1688 string   `json:"supplier_id_1688"`
	}{pageDataAlias: (*pageDataAlias)(p)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if p.PriceCNY == 0 && aux.Price1688 != nil {
		p.PriceCNY = *aux.Price1688
	}
	if p.MOQ == 0 && aux.MinOrderQty != nil {
		p.MOQ = *aux.MinOrderQty
	}
	if p.WeightKg == nil {
		p.WeightKg = aux.PackageWeight
	}
	if p.PackageLengthCm == nil {
		p.PackageLengthCm = aux.PackageLength
	}
	if p.PackageWidthCm == nil {
		p.PackageWidthCm = aux.PackageWidth
	}
	if p.PackageHeightCm == nil {
		p.PackageHeightCm = aux.PackageHeight
	}
	if p.SupplierBusinessID == "" {
		p.SupplierBusinessID = aux.SupplierID1688
	}
	return nil
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

// SpecVariant describes a specific variant of a product.
type SpecVariant struct {
	Spec  string  `json:"spec"`
	Price float64 `json:"price"`
	Stock int     `json:"stock,omitempty"`
}
