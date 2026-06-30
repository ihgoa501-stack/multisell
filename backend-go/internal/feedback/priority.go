package feedback

import (
	"math"
)

// PriorityCalculator computes a submission's priority score (0-100)
// using multi-dimensional weighting.
type PriorityCalculator struct{}

// NewPriorityCalculator creates a new calculator.
func NewPriorityCalculator() *PriorityCalculator {
	return &PriorityCalculator{}
}

// Calculate computes the priority score for a submission.
// Parameters:
//
//	baseScore: from severity + feedbackType (0-50)
//	voteCount: total upvotes
//	commentCount: total comments
//	ageHours: hours since submission creation
//	aiConfidence: AI classification confidence (0-1)
func (c *PriorityCalculator) Calculate(baseScore, voteCount, commentCount int, ageHours float64, aiConfidence float64) int {
	score := baseScore

	// Vote boost: every 5 upvotes = +5, max +25
	voteBoost := (voteCount / 5) * 5
	if voteBoost > 25 {
		voteBoost = 25
	}
	score += voteBoost

	// Comment activity: every 3 comments = +2, max +10
	commentBoost := (commentCount / 3) * 2
	if commentBoost > 10 {
		commentBoost = 10
	}
	score += commentBoost

	// AI confidence boost: confidence > 0.7 = +10, > 0.9 = +15
	if aiConfidence > 0.9 {
		score += 15
	} else if aiConfidence > 0.7 {
		score += 10
	} else if aiConfidence > 0.5 {
		score += 5
	}

	// Recency bonus: newer feedback gets higher score
	// First 24h: +20, first 7 days: +10, first 30 days: +5
	if ageHours <= 24 {
		score += 20
	} else if ageHours <= 24*7 {
		score += 10
	} else if ageHours <= 24*30 {
		score += 5
	} else {
		// Old feedback decays by -2 per month after 30 days
		months := ageHours / (24 * 30)
		decay := int(math.Floor(months)) * 2
		score -= decay
	}

	// Cap at 0-100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// BaseScore computes the initial score from severity and feedback type.
func BaseScore(severity, feedbackType string) int {
	score := 0

	switch severity {
	case "critical":
		score += 30
	case "major":
		score += 20
	case "minor":
		score += 8
	case "trivial":
		score += 3
	default:
		score += 12
	}

	switch feedbackType {
	case TypeBug:
		score += 20
	case TypeFeature:
		score += 15
	case TypeImprovement:
		score += 10
	}

	return score
}
