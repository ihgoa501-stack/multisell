// Package impl provides concrete agent implementations.
//
// SourcingAgent implements A8 Sourcing Agent business logic.
//
// Decision points:
//   - "sourcing_recommend" — analyze a 1688 product URL for sourcing viability,
//     estimate profit, and produce a recommendation for A2 listing_optimize
//     or for the user to review
package impl

import (
	"fmt"
	"strings"

	"github.com/lingmirror/backend-go/internal/domain/sourcing"
)

// ---------- Required context field names ----------

var sourcingRequiredFields = []string{
	"source_url", "price_1688", "weight_kg", "destination",
}

// ---------- SourcingAgent ----------

// SourcingAgent implements A8 Sourcing Agent logic.
// It handles the "sourcing_recommend" decision point — the entry point for
// AI-powered product sourcing analysis that kicks off the 1688→platform pipeline.
type SourcingAgent struct{}

// NewSourcingAgent creates a new SourcingAgent.
func NewSourcingAgent() *SourcingAgent {
	return &SourcingAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "sourcing_recommend" — profit analysis + sourcing recommendation
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *SourcingAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "sourcing_recommend":
		return a.recommend(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: sourcing_recommend ----------

// recommend analyzes sourcing viability and returns a structured recommendation.
//
// Required context fields: source_url, price_1688, weight_kg, destination
// Optional context fields: product_name, supplier_name, markup_pct, image_url
//
// Returns:
//   - profit_breakdown: detailed cost estimate and margin analysis
//   - status: "viable" if margin >= 15%, "marginal" if margin >= 5%, "unviable" otherwise
//   - action: what the agent suggests (review / escalate_to_optimizer / discard)
//   - message: human-readable analysis in Chinese
func (a *SourcingAgent) recommend(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Validate required fields.
	if missing := missingFields(ctx, sourcingRequiredFields); len(missing) > 0 {
		return insufficientData("sourcing_recommend", missing), 0.0, "low", nil
	}

	sourceURL := safeString(ctx["source_url"])
	price := safeFloat(ctx["price_1688"])
	weight := safeFloat(ctx["weight_kg"])
	dest := strings.ToUpper(safeString(ctx["destination"], "US"))
	productName := safeString(ctx["product_name"], "")
	markup := safeFloat(ctx["markup_pct"], 250.0)

	// Calculate profit.
	profit := sourcing.CalculateProfit(&sourcing.ProfitInput{
		SourcePriceCNY: price,
		WeightKg:       weight,
		Destination:    dest,
		MarkupPct:      markup,
	})

	// Determine status and action.
	status := "viable"
	action := "escalate_to_optimizer"
	var message string
	confidence = 0.85
	riskLevel = "low"

	switch {
	case profit.MarginPct >= 15:
		status = "viable"
		action = "escalate_to_optimizer"
		message = fmt.Sprintf(
			"该商品毛利率 %.1f%%，利润 ¥%.2f，利润率良好，建议推送至 A2 开启刊登优化流程。",
			profit.MarginPct, profit.ProfitCNY,
		)
		confidence = 0.90
		riskLevel = "low"
	case profit.MarginPct >= 5:
		status = "marginal"
		action = "review"
		message = fmt.Sprintf(
			"该商品毛利率 %.1f%%，利润 ¥%.2f，利润率偏低，建议人工评估后决定是否上架。",
			profit.MarginPct, profit.ProfitCNY,
		)
		confidence = 0.75
		riskLevel = "medium"
	default:
		status = "unviable"
		action = "discard"
		message = fmt.Sprintf(
			"该商品毛利率 %.1f%%，利润 ¥%.2f，不满足最低利润率要求(15%%)，建议放弃该选品。",
			profit.MarginPct, profit.ProfitCNY,
		)
		confidence = 0.95
		riskLevel = "low"
	}

	output = map[string]interface{}{
		"status":            status,
		"action":            action,
		"source_url":        sourceURL,
		"product_name":      productName,
		"profit_breakdown":  profit,
		"message":           message,
		"confidence":        confidence,
		"decision_point":    "sourcing_recommend",
	}

	return output, confidence, riskLevel, nil
}
