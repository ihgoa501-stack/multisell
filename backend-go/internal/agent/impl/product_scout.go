// Package impl provides concrete agent implementations.
//
// ProductScoutAgent implements A1 Product Scout business logic ported from
// backend/app/agent/agents/product_scout.py (Python FastAPI codebase).
//
// Design docs: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent1
//   - Multi-dimension product scoring (demand / competition / margin / trend)
//   - Input: category, marketplace, candidate product list
//   - Output: scored and ranked candidate list
package impl

import (
	"fmt"
	"sort"
)

// ---------- Context field names ----------

var productScoutRequiredFields = []string{"category", "marketplace"}

// ---------- candidateItem represents a product candidate with scoring data ----------

type candidateItem struct {
	Name            string
	SearchVolume    float64
	TrendGrowth     float64
	ReviewCount     float64
	Price           float64
	Cost            float64
}

// ---------- ProductScoutAgent ----------

// ProductScoutAgent implements A1 Product Scout logic.
//
// Decision points:
//   - "product_scout" — scores and ranks product candidates by multi-dimension formula
//   - "market_analysis" — returns a market analysis summary
type ProductScoutAgent struct{}

// NewProductScoutAgent creates a new ProductScoutAgent.
func NewProductScoutAgent() *ProductScoutAgent {
	return &ProductScoutAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "product_scout"
//   - "market_analysis"
func (a *ProductScoutAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "product_scout":
		return a.scout(ctx)
	case "market_analysis":
		return a.analyzeMarket(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: product_scout ----------

// scout scores and ranks product candidates using the multi-dimension formula:
//
//	score = demand*30 + growth*25 + competition*20 + margin*25
//
// where:
//   - demand = search_volume * 0.01
//   - growth = trend_growth / 100
//   - competition = max(0, 1 - review_count / 1000)
//   - margin = (price - cost) / price (when price > 0)
//
// Required context fields: category, marketplace
// Required candidates list: each item must have name, price, cost, search_volume,
// trend_growth, and review_count.
//
// Returns top-20 scored candidates sorted by score descending.
func (a *ProductScoutAgent) scout(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, productScoutRequiredFields); len(missing) > 0 {
		return insufficientData("product_scout", missing), 0.0, "low", nil
	}

	category := safeString(ctx["category"], "")
	marketplace := safeString(ctx["marketplace"], "US")

	candidatesInput := ctx["candidates"]
	if candidatesInput == nil {
		return insufficientData("product_scout", []string{"candidates"}), 0.0, "low", nil
	}

	rawList, ok := candidatesInput.([]interface{})
	if !ok || len(rawList) == 0 {
		return insufficientData("product_scout", []string{"candidates"}), 0.0, "low", nil
	}

	type scoredCandidate struct {
		Name             string  `json:"name"`
		Score            float64 `json:"score"`
		DemandScore      float64 `json:"demand_score"`
		CompetitionScore float64 `json:"competition_score"`
		MarginScore      float64 `json:"margin_score"`
		TrendScore       float64 `json:"trend_score"`
		EstimatedMargin  float64 `json:"estimated_margin"`
		RiskFlags        []string `json:"risk_flags"`
	}

	scored := make([]scoredCandidate, 0, len(rawList))

	for _, raw := range rawList {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		demand := safeFloat(item["search_volume"], 0) * 0.01
		growth := safeFloat(item["trend_growth"], 0) / 100.0
		competition := max(0, 1-safeFloat(item["review_count"], 0)/1000.0)
		price := safeFloat(item["price"], 0)
		cost := safeFloat(item["cost"], 0)

		var margin float64
		if price > 0 {
			margin = (price - cost) / price
		}

		score := round1(demand*30 + growth*25 + competition*20 + margin*25)

		riskFlags := make([]string, 0)
		if competition < 0.3 {
			riskFlags = append(riskFlags, "高竞争")
		}

		scored = append(scored, scoredCandidate{
			Name:             safeString(item["name"], ""),
			Score:            score,
			DemandScore:      round1(demand * 100),
			CompetitionScore: round1(competition * 100),
			MarginScore:      round1(margin * 100),
			TrendScore:       round1(growth * 100),
			EstimatedMargin:  round1(margin * 100),
			RiskFlags:        riskFlags,
		})
	}

	// Sort by score descending (mirrors Python: scored.sort(key=lambda x: x["score"], reverse=True)).
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Return top 20.
	topN := 20
	if len(scored) < topN {
		topN = len(scored)
	}

	output = map[string]interface{}{
		"category":      category,
		"marketplace":   marketplace,
		"candidates":    scored[:topN],
		"total_scanned": len(scored),
		"confidence":    0.85,
	}
	return output, 0.85, "low", nil
}

// ---------- Decision point: market_analysis ----------

// analyzeMarket returns a high-level market analysis summary.
func (a *ProductScoutAgent) analyzeMarket(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	output = map[string]interface{}{
		"category":             safeString(ctx["category"], ""),
		"marketplace":          safeString(ctx["marketplace"], "US"),
		"market_size_estimate": "medium",
		"trend_direction":      safeString(ctx["trend"], "stable"),
		"confidence":           0.80,
	}
	return output, 0.80, "low", nil
}
