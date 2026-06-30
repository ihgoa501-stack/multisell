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
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
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
//   - "sourcing_recommend" — delegates to toolregistry.DefaultRegistry.Call()
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *SourcingAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "sourcing_recommend":
		// Validate required fields before calling the tool.
		if missing := missingFields(params, sourcingRequiredFields); len(missing) > 0 {
			return insufficientData("sourcing_recommend", missing), 0.0, "low", nil
		}

		result, callErr := toolregistry.DefaultRegistry.Call(ctx, "sourcing.recommend", params)
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

	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}
