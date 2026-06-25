// Package impl provides concrete agent implementations.
//
// DashboardAgent implements G1 Dashboard business logic ported from
// backend/app/agent/agents/dashboard.py (Python FastAPI codebase).
//
// Design docs: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md section 7.1.5
//   - Dashboard overview — returns a message directing to the dedicated API endpoint
//   - G1's dashboard data is provided via dedicated API endpoint (not via Decide)
//   - This agent is a placeholder for future extensibility
package impl

import (
	"fmt"
)

// DashboardAgent implements G1 Dashboard logic.
//
// Decision points:
//   - "dashboard_overview" — returns a message directing to the API endpoint
type DashboardAgent struct{}

// NewDashboardAgent creates a new DashboardAgent.
func NewDashboardAgent() *DashboardAgent {
	return &DashboardAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "dashboard_overview"
func (a *DashboardAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if decisionPoint == "dashboard_overview" {
		output = map[string]interface{}{
			"message":    "请使用 /api/agents/dashboard 端点获取驾驶舱数据",
			"confidence": 0.0,
		}
		return output, 0.0, "low", nil
	}
	return map[string]interface{}{
		"status":         "unknown",
		"decision_point": decisionPoint,
		"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
	}, 0.0, "low", nil
}
