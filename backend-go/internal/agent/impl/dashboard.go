// Package impl provides concrete agent implementations.
//
// DashboardAgent implements G1 Dashboard business logic ported from
// backend/app/agent/agents/dashboard.py (Python FastAPI codebase).
//
// Design docs: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md section 7.1.5
//   - Dashboard overview — calls the dashboard.overview tool via ToolRegistry
//   - G1's dashboard data is provided via dedicated API endpoint (not via Decide)
//   - This agent delegates logic to the registered dashboard tool
package impl

import (
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// DashboardAgent implements G1 Dashboard logic.
//
// Decision points:
//   - "dashboard_overview" — calls the dashboard.overview tool via ToolRegistry
type DashboardAgent struct{}

// NewDashboardAgent creates a new DashboardAgent.
func NewDashboardAgent() *DashboardAgent {
	return &DashboardAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "dashboard_overview" — delegates to toolregistry.DefaultRegistry.Call()
func (a *DashboardAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if decisionPoint == "dashboard_overview" {
		result, callErr := toolregistry.DefaultRegistry.Call(ctx, "dashboard.overview", params)
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
	}
	return map[string]interface{}{
		"status":         "unknown",
		"decision_point": decisionPoint,
		"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
	}, 0.0, "low", nil
}
