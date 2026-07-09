package loop

import (
	"encoding/json"
	"fmt"
	"time"
)

// GetRecommendationFeedbackSummary aggregates recommendation feedback for a product.
// Returns counts by feedback_status, derived rates, and individual review items.
func (s *Service) GetRecommendationFeedbackSummary(productID int64) (*RecommendationFeedbackSummary, error) {
	var recs []ListingRecommendation
	if err := s.db.Where("product_id = ?", productID).Order("id DESC").Find(&recs).Error; err != nil {
		return nil, fmt.Errorf("query listing recommendations: %w", err)
	}

	summary := &RecommendationFeedbackSummary{
		Reviews: make([]ReviewItem, 0, len(recs)),
	}
	var lastUpdated time.Time

	for _, r := range recs {
		// Parse failure_type from feedback_note JSON (V2 format).
		var failureType string
		if r.FeedbackStatus == "execution_failed" && r.FeedbackNote != "" {
			var noteData map[string]interface{}
			if err := json.Unmarshal([]byte(r.FeedbackNote), &noteData); err == nil {
				if ft, ok := noteData["failure_type"].(string); ok {
					failureType = ft
				}
			}
		}

		// Derive execution_success from feedback_status.
		var execSuccess *bool
		switch r.FeedbackStatus {
		case "executed":
			v := true
			execSuccess = &v
		case "execution_failed":
			v := false
			execSuccess = &v
		}

		summary.TotalRecommendations++
		switch r.FeedbackStatus {
		case "adopted":
			summary.AdoptedCount++
		case "rejected":
			summary.RejectedCount++
		case "executed":
			summary.ExecutedCount++
		case "execution_failed":
			summary.FailedCount++
		}

		summary.Reviews = append(summary.Reviews, ReviewItem{
			RecommendationID: r.ID,
			Decision:         r.Decision,
			Confidence:       r.Confidence,
			FeedbackStatus:   r.FeedbackStatus,
			ExecutionSuccess: execSuccess,
			FailureType:      failureType,
			CreatedAt:        r.CreatedAt,
		})

		if r.UpdatedAt.After(lastUpdated) {
			lastUpdated = r.UpdatedAt
		}
	}

	// AdoptRate: recommendations that were adopted (adopted, executed, or execution_failed) / total.
	if summary.TotalRecommendations > 0 {
		adopted := summary.AdoptedCount + summary.ExecutedCount + summary.FailedCount
		summary.AdoptRate = float64(adopted) / float64(summary.TotalRecommendations) * 100
		totalExec := summary.ExecutedCount + summary.FailedCount
		if totalExec > 0 {
			summary.SuccessRate = float64(summary.ExecutedCount) / float64(totalExec) * 100
		}
	}
	summary.LastUpdated = lastUpdated

	return summary, nil
}
