// Package impl provides concrete agent implementations.
//
// ListingOptimizerAgent implements A2 Listing Optimizer business logic ported
// from backend/app/agent/agents/listing_optimizer.py (Python FastAPI codebase).
//
// Design docs: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent2
//   - Keyword strategy + competitive deconstruction + copy generation
//   - Input: product info and competitor data
//   - Output: optimized listing (title, bullets, search terms, keyword research)
package impl

import (
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// ---------- ListingOptimizerAgent ----------

// ListingOptimizerAgent implements A2 Listing Optimizer logic.
//
// Decision points:
//   - "listing_optimize" — generates optimized title, bullets, search terms,
//     and actionable suggestions based on product features and keywords
//   - "keyword_research" — expands seed keywords into broader keyword candidates
//     by appending common suffixes (for, with, best)
type ListingOptimizerAgent struct{}

// NewListingOptimizerAgent creates a new ListingOptimizerAgent.
func NewListingOptimizerAgent() *ListingOptimizerAgent {
	return &ListingOptimizerAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "listing_optimize"
//   - "keyword_research"
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *ListingOptimizerAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "listing_optimize":
		result, callErr := toolregistry.DefaultRegistry.Call(context.Background(), "listing.optimize", ctx)
		if callErr != nil {
			return map[string]interface{}{
				"status":         "error",
				"decision_point": decisionPoint,
				"error":          callErr.Error(),
			}, 0.0, "low", nil
		}
		output, ok := result.(map[string]interface{})
		if !ok {
			return map[string]interface{}{
				"status":         "error",
				"decision_point": decisionPoint,
				"error":          "unexpected tool result type",
			}, 0.0, "low", nil
		}
		return output, 0.0, "low", nil
	case "keyword_research":
		return a.researchKeywords(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: keyword_research ----------

// researchKeywords expands seed keywords into broader keyword candidates.
//
// Ported from Python _research_keywords:
//   Takes seed_keywords list and generates "S for", "S with", "S best" combos.
//
// Example: seed=["shoes","bags"] produces:
//   expanded=["shoes for","shoes with","shoes best","bags for","bags with","bags best"]
//   total_found = 6
func (a *ListingOptimizerAgent) researchKeywords(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	seed := parseStringList(ctx["seed_keywords"])
	if len(seed) == 0 {
		return insufficientData("keyword_research", []string{"seed_keywords"}), 0.0, "low", nil
	}

	suffixes := []string{"for", "with", "best"}
	expanded := make([]string, 0, len(seed)*len(suffixes))
	for _, s := range seed {
		for _, t := range suffixes {
			expanded = append(expanded, s+" "+t)
		}
	}

	output = map[string]interface{}{
		"seed":        seed,
		"expanded":    expanded,
		"total_found": len(seed) * 3,
	}

	return output, 0.80, "low", nil
}

// ---------- Helper ----------

// parseStringList extracts a []string from an interface{} value.
//
// Handles []interface{} (from JSON unmarshalling) where each element is a string.
// Returns nil when v is nil, not a list, or all elements are non-string.
func parseStringList(v interface{}) []string {
	if v == nil {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if ok {
			out = append(out, s)
		}
	}
	return out
}
