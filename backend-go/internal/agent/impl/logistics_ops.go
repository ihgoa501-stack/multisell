// Package impl provides concrete agent implementations.
//
// LogisticsOpsAgent implements A10 Logistics Operations business logic,
// covering carrier comparison, shipping bill auditing, carrier performance
// scoring, and logistics route optimization.
//
// Decision points:
//   - carrier_compare        — Compare carriers by cost, speed, and suitability
//   - shipping_bill_audit    — Audit shipping bills for overcharges
//   - carrier_performance    — Score carrier performance across dimensions
//   - logistics_route_opt    — Recommend logistics route mix optimization
//
// All outputs are in Chinese with confidence-adaptive graceful degradation.
package impl

import (
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// logisticsToolResult wraps the three return values from logistics tool handlers
// so they can pass through the toolregistry's single-return-value handler interface.
type logisticsToolResult struct {
	Output     map[string]interface{}
	Confidence float64
	RiskLevel  string
}

// LogisticsOpsAgent implements A10 logistics operations logic.
type LogisticsOpsAgent struct {
	db      *gorm.DB
	logger  *zap.Logger
	toolReg *toolregistry.ToolRegistry
}

// NewLogisticsOpsAgent creates a new LogisticsOpsAgent with DB access.
func NewLogisticsOpsAgent(db *gorm.DB, logger *zap.Logger) *LogisticsOpsAgent {
	a := &LogisticsOpsAgent{
		db:      db,
		logger:  logger,
		toolReg: toolregistry.NewToolRegistry(logger),
	}
	a.registerTools()
	a.registerFulfillmentAdvice()
	return a
}

// registerTools registers all logistics decision points as tools in the
// ToolRegistry so Decide() can delegate to it.
func (a *LogisticsOpsAgent) registerTools() {
	for _, t := range a.toolDefs() {
		a.toolReg.Register(t)
	}
}

// toolDefs returns the Tool definitions for each supported decision point.
func (a *LogisticsOpsAgent) toolDefs() []*toolregistry.Tool {
	return []*toolregistry.Tool{
		{
			Name:        "carrier_compare",
			Version:     "1.0.0",
			Description: "承运商比价——比较不同承运商的运费和时效，推荐性价比最优方案",
			Squad:       "logistics",
			RiskLevel:   toolregistry.RiskLow,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.carrierCompare(input)
				if err != nil {
					return nil, err
				}
				return &logisticsToolResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
			},
		},
		{
			Name:        "shipping_bill_audit",
			Version:     "1.0.0",
			Description: "运费审计——审计物流账单，比对实际运费与预估费用，识别异常收费",
			Squad:       "logistics",
			RiskLevel:   toolregistry.RiskMedium,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.shippingBillAudit(input)
				if err != nil {
					return nil, err
				}
				return &logisticsToolResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
			},
		},
		{
			Name:        "carrier_performance",
			Version:     "1.0.0",
			Description: "承运商绩效评分——对承运商按送达时效、破损率、成本等维度综合评分",
			Squad:       "logistics",
			RiskLevel:   toolregistry.RiskLow,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.carrierPerformance(input)
				if err != nil {
					return nil, err
				}
				return &logisticsToolResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
			},
		},
		{
			Name:        "logistics_route_opt",
			Version:     "1.0.0",
			Description: "物流路线优化——分析当前物流路线分配，基于目的地分布推荐调整方案",
			Squad:       "logistics",
			RiskLevel:   toolregistry.RiskLow,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.logisticsRouteOpt(input)
				if err != nil {
					return nil, err
				}
				return &logisticsToolResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
			},
		},
	}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - carrier_compare        — compare carriers by cost/speed
//   - shipping_bill_audit    — audit recent shipping bills
//   - carrier_performance    — score carrier performance
//   - logistics_route_opt    — recommend logistics route mix
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *LogisticsOpsAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	result, callErr := a.toolReg.Call(ctx, decisionPoint, params)
	if callErr != nil {
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("未知决策点: %s", decisionPoint),
		}, 0.0, "low", nil
	}
	lr, ok := result.(*logisticsToolResult)
	if !ok {
		return map[string]interface{}{
			"status":         "error",
			"decision_point": decisionPoint,
			"error":          "意外的返回类型",
		}, 0.0, "high", fmt.Errorf("unexpected result type from tool registry: %T", result)
	}
	return lr.Output, lr.Confidence, lr.RiskLevel, nil
}
