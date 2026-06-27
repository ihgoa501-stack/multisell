// Package impl provides concrete agent implementations.
//
// AftersalesMgmtAgent implements A11 Aftersales Management business logic.
// It handles return analysis, refund decision automation, dispute management,
// and aggregated KPI reporting with anomaly alerts.
package impl

import (
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AftersalesMgmtAgent implements A11 Aftersales Management logic.
// It handles return analysis, refund decision automation, dispute management,
// and aggregated reporting for aftersales operations.
type AftersalesMgmtAgent struct {
	db      *gorm.DB
	logger  *zap.Logger
	toolReg *toolregistry.ToolRegistry
}

// NewAftersalesMgmtAgent creates a new AftersalesMgmtAgent.
func NewAftersalesMgmtAgent(db *gorm.DB, logger *zap.Logger) *AftersalesMgmtAgent {
	a := &AftersalesMgmtAgent{db: db, logger: logger}
	a.toolReg = toolregistry.NewToolRegistry(logger)
	a.registerTools()
	return a
}

// registerTools registers the agent's decision point tools in its internal
// tool registry. Each tool wraps the corresponding business logic method
// as a toolregistry.Handler so that Decide dispatches through the registry.
func (a *AftersalesMgmtAgent) registerTools() {
	tools := []toolregistry.Tool{
		{
			Name:        "aftersales.agent.return_analysis",
			Version:     "1.0.0",
			Description: "退货分析",
			RiskLevel:   toolregistry.RiskLow,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.returnAnalysis(input)
				if err != nil {
					return nil, err
				}
				output["_confidence"] = confidence
				output["_risk_level"] = riskLevel
				return output, nil
			},
		},
		{
			Name:        "aftersales.agent.refund_decision",
			Version:     "1.0.0",
			Description: "退款决策",
			RiskLevel:   toolregistry.RiskHigh,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.refundDecision(input)
				if err != nil {
					return nil, err
				}
				output["_confidence"] = confidence
				output["_risk_level"] = riskLevel
				return output, nil
			},
		},
		{
			Name:        "aftersales.agent.dispute_manage",
			Version:     "1.0.0",
			Description: "争议管理",
			RiskLevel:   toolregistry.RiskMedium,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.disputeManage(input)
				if err != nil {
					return nil, err
				}
				output["_confidence"] = confidence
				output["_risk_level"] = riskLevel
				return output, nil
			},
		},
		{
			Name:        "aftersales.agent.aftersales_report",
			Version:     "1.0.0",
			Description: "售后KPI报告",
			RiskLevel:   toolregistry.RiskLow,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.aftersalesReport(input)
				if err != nil {
					return nil, err
				}
				output["_confidence"] = confidence
				output["_risk_level"] = riskLevel
				return output, nil
			},
		},
	}
	for _, t := range tools {
		a.toolReg.Register(&t)
	}
}

// callDecision dispatches a decision point through the tool registry,
// extracting confidence and risk level from the tool handler output.
func (a *AftersalesMgmtAgent) callDecision(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	result, err := a.toolReg.Call(context.Background(), "aftersales.agent."+decisionPoint, ctx)
	if err != nil {
		return nil, 0, "high", err
	}
	output = result.(map[string]interface{})
	confidence, _ = output["_confidence"].(float64)
	riskLevel, _ = output["_risk_level"].(string)
	delete(output, "_confidence")
	delete(output, "_risk_level")
	return output, confidence, riskLevel, nil
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
	case "return_analysis", "refund_decision", "dispute_manage", "aftersales_report":
		return a.callDecision(decisionPoint, ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("未知决策点: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}
