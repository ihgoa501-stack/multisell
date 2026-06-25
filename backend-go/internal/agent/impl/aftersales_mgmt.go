// Package impl provides concrete agent implementations.
//
// AftersalesMgmtAgent implements A11 Aftersales Management business logic.
// It handles return analysis, refund decision automation, dispute management,
// and aggregated KPI reporting with anomaly alerts.
package impl

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AftersalesMgmtAgent implements A11 Aftersales Management logic.
// It handles return analysis, refund decision automation, dispute management,
// and aggregated reporting for aftersales operations.
type AftersalesMgmtAgent struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewAftersalesMgmtAgent creates a new AftersalesMgmtAgent.
func NewAftersalesMgmtAgent(db *gorm.DB, logger *zap.Logger) *AftersalesMgmtAgent {
	return &AftersalesMgmtAgent{db: db, logger: logger}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "return_analysis"   — aggregated return reason analysis and problem SKU detection
//   - "refund_decision"   — evaluate a single refund or scan pending refunds
//   - "dispute_manage"    — platform dispute monitoring and response recommendation
//   - "aftersales_report" — KPI aggregation with trend and anomaly alerts
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *AftersalesMgmtAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "return_analysis":
		return a.returnAnalysis(ctx)
	case "refund_decision":
		return a.refundDecision(ctx)
	case "dispute_manage":
		return a.disputeManage(ctx)
	case "aftersales_report":
		return a.aftersalesReport(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("未知决策点: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}
