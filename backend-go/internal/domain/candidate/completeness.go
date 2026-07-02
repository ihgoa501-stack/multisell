package candidate

// computeCompleteness evaluates a CandidateProduct's field completeness.
// Returns the overall status and a list of missing field names.
// "ready_for_profit_check" means all essential fields are present;
// "incomplete" means one or more critical fields are missing.
func computeCompleteness(p *CandidateProduct) (status string, missingFields []string) {
	if p.Title == "" {
		missingFields = append(missingFields, "title")
	}
	if p.PurchasePrice <= 0 {
		missingFields = append(missingFields, "purchase_price")
	}
	if p.MainImage == "" {
		missingFields = append(missingFields, "main_image")
	}
	if p.PackageWeightKg <= 0 {
		missingFields = append(missingFields, "package_weight_kg")
	}
	if p.PackageLengthCm <= 0 {
		missingFields = append(missingFields, "package_length_cm")
	}
	if p.PackageWidthCm <= 0 {
		missingFields = append(missingFields, "package_width_cm")
	}
	if p.PackageHeightCm <= 0 {
		missingFields = append(missingFields, "package_height_cm")
	}
	if p.SupplierID == nil || *p.SupplierID == 0 {
		missingFields = append(missingFields, "supplier_id")
	}

	if len(missingFields) > 0 {
		return "incomplete", missingFields
	}
	return "ready_for_profit_check", nil
}
