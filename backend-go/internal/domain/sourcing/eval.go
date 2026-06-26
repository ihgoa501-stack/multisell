// Package sourcing provides domain logic for the A8 sourcing agent,
// including LLM-quality evaluation, profit calculation, and competitive analysis.
package sourcing

import "unicode/utf8"

// QualityReport holds the result of an LLM-quality evaluation for a
// product page. This struct is used as the deterministic baseline; when
// real LLM calls are added later their output can be compared against it.
type QualityReport struct {
	Score     int              `json:"score"`
	Breakdown QualityBreakdown `json:"breakdown"`
	Warnings  []string         `json:"warnings,omitempty"`
}

// QualityBreakdown breaks the total score down into the per-dimension
// contributions and any penalties applied.
type QualityBreakdown struct {
	Title        int `json:"title"`
	Images       int `json:"images"`
	Supplier     int `json:"supplier"`
	Description  int `json:"description"`
	SpecRichness int `json:"spec_richness"`
	Logistics    int `json:"logistics"`
	Penalty      int `json:"penalty"`
}

// QualityEvaluator evaluates product page quality using deterministic
// scoring rules that simulate what an LLM quality-assessment call would
// return. It provides a reliable test baseline so that when real LLM
// calls are swapped in later the two sets of scores can be compared.
type QualityEvaluator struct{}

// NewQualityEvaluator creates a new QualityEvaluator.
func NewQualityEvaluator() *QualityEvaluator {
	return &QualityEvaluator{}
}

// EvaluateQuality runs the deterministic scoring rules against the given
// PageData and returns a QualityReport with a 0-100 score, dimension
// breakdown, and any warnings about missing or suspicious fields.
func (e *QualityEvaluator) EvaluateQuality(page *PageData) *QualityReport {
	if page == nil {
		return &QualityReport{
			Score:    0,
			Warnings: []string{"nil page data"},
		}
	}

	var warnings []string

	titleScore := evaluateTitle(page.Title)
	imageScore := evaluateImages(len(page.Images))
	supplierScore := evaluateSupplier(page.SupplierScore)
	descScore := evaluateDescription(page.Description)
	specScore := evaluateSpecs(len(page.SpecVariants))
	logisticsScore := evaluatePackage(
		page.PackageWeight,
		page.PackageLength,
		page.PackageWidth,
		page.PackageHeight,
		page.FreightCNY,
	)

	baseScore := titleScore + imageScore + supplierScore + descScore + specScore + logisticsScore

	// Apply penalties for missing critical fields.
	penalty := 0
	if page.Price <= 0 {
		penalty += 15
		warnings = append(warnings, "missing price")
	}
	if len(page.Images) == 0 {
		penalty += 10
		warnings = append(warnings, "no product images")
	}
	if utf8.RuneCountInString(page.Description) == 0 {
		penalty += 10
		warnings = append(warnings, "missing product description")
	}
	if page.SupplierName == "" {
		penalty += 5
		warnings = append(warnings, "no supplier name")
	}
	if page.SupplierScore == nil {
		warnings = append(warnings, "no supplier trust score")
	}

	finalScore := baseScore - penalty
	if finalScore < 0 {
		finalScore = 0
	}
	if finalScore > 100 {
		finalScore = 100
	}

	return &QualityReport{
		Score: finalScore,
		Breakdown: QualityBreakdown{
			Title:        titleScore,
			Images:       imageScore,
			Supplier:     supplierScore,
			Description:  descScore,
			SpecRichness: specScore,
			Logistics:    logisticsScore,
			Penalty:      penalty,
		},
		Warnings: warnings,
	}
}

// evaluateTitle returns up to 20 points based on title length in runes.
func evaluateTitle(title string) int {
	switch l := utf8.RuneCountInString(title); {
	case l >= 20:
		return 20
	case l >= 10:
		return 15
	case l >= 5:
		return 10
	default:
		return 5
	}
}

// evaluateImages returns up to 20 points based on the number of images.
func evaluateImages(count int) int {
	switch {
	case count >= 5:
		return 20
	case count >= 3:
		return 15
	case count >= 1:
		return 10
	default:
		return 0
	}
}

// evaluateSupplier returns up to 20 points based on the supplier trust score.
func evaluateSupplier(score *int) int {
	if score == nil {
		return 0
	}
	switch s := *score; {
	case s >= 80:
		return 20
	case s >= 60:
		return 15
	case s >= 40:
		return 10
	default:
		return 5
	}
}

// evaluateDescription returns up to 20 points based on description length in runes.
func evaluateDescription(desc string) int {
	switch l := utf8.RuneCountInString(desc); {
	case l >= 200:
		return 20
	case l >= 100:
		return 15
	case l >= 30:
		return 10
	case l > 0:
		return 5
	default:
		return 0
	}
}

// evaluateSpecs returns up to 15 points based on the number of spec variants,
// rewarding product pages that offer multiple options.
func evaluateSpecs(count int) int {
	switch {
	case count >= 5:
		return 15
	case count >= 3:
		return 12
	case count >= 1:
		return 8
	default:
		return 0
	}
}

// evaluatePackage returns up to 10 points when package weight, dimensions,
// and/or freight cost are present — signalling a more complete listing.
func evaluatePackage(weight, length, width, height, freight *float64) int {
	hasWeight := weight != nil
	hasDims := length != nil && width != nil && height != nil
	hasFreight := freight != nil

	switch {
	case hasWeight && hasDims && hasFreight:
		return 10
	case hasWeight && hasDims:
		return 7
	case hasWeight:
		return 3
	default:
		return 0
	}
}
