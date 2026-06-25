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
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LogisticsOpsAgent implements A10 logistics operations logic.
type LogisticsOpsAgent struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewLogisticsOpsAgent creates a new LogisticsOpsAgent with DB access.
func NewLogisticsOpsAgent(db *gorm.DB, logger *zap.Logger) *LogisticsOpsAgent {
	return &LogisticsOpsAgent{db: db, logger: logger}
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
func (a *LogisticsOpsAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "carrier_compare":
		return a.carrierCompare(ctx)
	case "shipping_bill_audit":
		return a.shippingBillAudit(ctx)
	case "carrier_performance":
		return a.carrierPerformance(ctx)
	case "logistics_route_opt":
		return a.logisticsRouteOpt(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("未知决策点: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}
