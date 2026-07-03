package candidate

import "encoding/json"

// computeCompleteness evaluates a CandidateProduct's field completeness,
// respecting any fields the user has marked as intentionally skipped.
// Returns the overall status and a list of missing field names.
//
// Status progression:
//
//	incomplete     — missing core fields (title, purchase_price, main_image)
//	needs_review   — has core fields, missing supplier or package info
//	research_ready — has core + supplier + package, can proceed to profit check
//	listing_ready  — all 11 fields present (or skipped)
func computeCompleteness(p *CandidateProduct) (status string, missingFields []string) {
	// Parse skipped fields from the model
	skipped := make(map[string]bool)
	if len(p.SkippedFields) > 0 {
		var fields []string
		if err := json.Unmarshal(p.SkippedFields, &fields); err == nil {
			for _, f := range fields {
				skipped[f] = true
			}
		}
	}

	// Helper: a field "exists" if it has a value OR the user skipped it.
	exists := func(key string, hasValue bool) bool {
		return hasValue || skipped[key]
	}

	// Check all 11 fields
	checks := []struct {
		key      string
		hasValue bool
	}{
		{"title", p.Title != ""},
		{"purchase_price", p.PurchasePrice > 0},
		{"main_image", p.MainImage != ""},
		{"supplier_id", p.SupplierID != nil && *p.SupplierID > 0},
		{"package_weight_kg", p.PackageWeightKg > 0},
		{"package_length_cm", p.PackageLengthCm > 0},
		{"package_width_cm", p.PackageWidthCm > 0},
		{"package_height_cm", p.PackageHeightCm > 0},
		{"hs_code", p.HSCode != ""},
		{"target_sale_price", p.TargetSalePrice > 0},
		{"origin_country", p.OriginCountry != ""},
	}

	for _, c := range checks {
		if !exists(c.key, c.hasValue) {
			missingFields = append(missingFields, c.key)
		}
	}

	// 3-tier classification
	hasCore := exists("title", p.Title != "") &&
		exists("purchase_price", p.PurchasePrice > 0) &&
		exists("main_image", p.MainImage != "")
	hasSupplier := exists("supplier_id", p.SupplierID != nil && *p.SupplierID > 0)
	hasPackage := exists("package_weight_kg", p.PackageWeightKg > 0) ||
		(exists("package_length_cm", p.PackageLengthCm > 0) &&
			exists("package_width_cm", p.PackageWidthCm > 0) &&
			exists("package_height_cm", p.PackageHeightCm > 0))
	hasAll := len(missingFields) == 0

	switch {
	case !hasCore:
		return "incomplete", missingFields
	case !hasSupplier || !hasPackage:
		return "needs_review", missingFields
	case hasAll:
		return "listing_ready", nil
	default:
		return "research_ready", missingFields
	}
}

// parseSkippedFields converts the raw JSONB skipped_fields column
// into a lookup set for efficient O(1) membership checks.
func parseSkippedFields(p *CandidateProduct) map[string]bool {
	skipped := make(map[string]bool)
	if len(p.SkippedFields) > 0 {
		var fields []string
		if err := json.Unmarshal(p.SkippedFields, &fields); err == nil {
			for _, f := range fields {
				skipped[f] = true
			}
		}
	}
	return skipped
}
