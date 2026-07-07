package listing

import (
	"fmt"
)

// PlatformComparisonResult holds per-platform listing comparison data for a product.
type PlatformComparisonResult struct {
	PlatformID       int64   `json:"platform_id"`
	PlatformName     string  `json:"platform_name"`
	EstimatedProfit  float64 `json:"estimated_profit"`
	ProfitMargin     float64 `json:"profit_margin"`
	TotalCost        float64 `json:"total_cost"`
	FeeAmount        float64 `json:"fee_amount"`
	LogisticsCost    float64 `json:"logistics_cost"`
	RiskLevel        string  `json:"risk_level"`
	Decision         string  `json:"decision"`
	Confidence       float64 `json:"confidence"`
	Suggested        bool    `json:"suggested"`
	RecommendationID int64   `json:"recommendation_id"`
	FeedbackStatus   string  `json:"feedback_status"`
}

// GetPlatformComparison returns per-platform comparison data for a product.
// It joins listing_recommendation with listing_task to resolve platform context,
// and overlays profit_summary cost data.
//
// ponytail: profit_summary is product-level, not per-platform. The cost data is
// the same across all platforms for a given product. Per-platform profit data
// would require a schema change (platform_id on profit_summary or listing_recommendation).
func (s *Service) GetPlatformComparison(productID int64) ([]PlatformComparisonResult, error) {
	// Query recommendations with platform context via listing_task join.
	type recRow struct {
		PlatformID      int64
		PlatformName    string
		RecID           int64
		Decision        string
		Confidence      float64
		ProfitMargin    float64
		EstimatedProfit float64
		FeedbackStatus  string
	}
	var rows []recRow
	if err := s.db.Table("listing_recommendation lr").
		Select(`lt.platform_id, p.name as platform_name, lr.id as rec_id, lr.decision,
			lr.confidence, lr.profit_margin, lr.estimated_profit, lr.feedback_status`).
		Joins("JOIN listing_task lt ON lt.id = lr.created_listing_task_id").
		Joins("JOIN platform p ON p.id = lt.platform_id").
		Where("lr.product_id = ?", productID).
		Order("lr.id DESC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query platform comparison: %w", err)
	}

	// Get profit_summary cost data (product-level — same for all platforms).
	type profitRow struct {
		TotalCost    float64
		PlatformFee  float64
		ShippingCost float64
	}
	var ps profitRow
	s.db.Table("profit_summary").
		Select("total_cost, platform_fee, shipping_cost").
		Where("product_id = ?", productID).
		Order("id DESC").
		First(&ps)

	// Deduplicate by platform_id (keep the latest recommendation per platform).
	seen := make(map[int64]bool)
	results := make([]PlatformComparisonResult, 0, len(rows))
	for _, r := range rows {
		if seen[r.PlatformID] {
			continue
		}
		seen[r.PlatformID] = true

		riskLevel := "unknown"
		switch {
		case r.ProfitMargin >= 15:
			riskLevel = "low"
		case r.ProfitMargin >= 0:
			riskLevel = "medium"
		default:
			riskLevel = "high"
		}

		results = append(results, PlatformComparisonResult{
			PlatformID:       r.PlatformID,
			PlatformName:     r.PlatformName,
			EstimatedProfit:  r.EstimatedProfit,
			ProfitMargin:     r.ProfitMargin,
			TotalCost:        ps.TotalCost,
			FeeAmount:        ps.PlatformFee,
			LogisticsCost:    ps.ShippingCost,
			RiskLevel:        riskLevel,
			Decision:         r.Decision,
			Confidence:       r.Confidence,
			Suggested:        r.Decision == "list",
			RecommendationID: r.RecID,
			FeedbackStatus:   r.FeedbackStatus,
		})
	}

	return results, nil
}
