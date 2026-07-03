package candidate

// computeCompleteness evaluates a CandidateProduct's field completeness.
// Returns the overall status and a list of missing field names.
//
// Status progression:
//
//	incomplete     — missing core fields (title, purchase_price, main_image)
//	needs_review   — has core fields, missing supplier or package info
//	research_ready — has core + supplier + package, can proceed to profit check
//	listing_ready  — all 11 fields present
func computeCompleteness(p *CandidateProduct) (status string, missingFields []string) {
	// Check all 11 fields
	checks := []struct {
		key    string
		exists bool
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
		if !c.exists {
			missingFields = append(missingFields, c.key)
		}
	}

	// 3-tier classification
	hasCore := p.Title != "" && p.PurchasePrice > 0 && p.MainImage != ""
	hasSupplier := p.SupplierID != nil && *p.SupplierID > 0
	hasPackage := p.PackageWeightKg > 0 || (p.PackageLengthCm > 0 && p.PackageWidthCm > 0 && p.PackageHeightCm > 0)
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
