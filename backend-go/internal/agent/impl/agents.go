// Package impl provides concrete implementations of all registered AI agents.
// Each agent satisfies the Agent interface.
package impl

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Agent is the interface that all agent implementations must satisfy.
type Agent interface {
	Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error)
}

// All returns all registered agent implementations keyed by agent ID.
// The orchestrator uses this to dispatch real business logic instead of stubs.
func All(db *gorm.DB, logger *zap.Logger) map[string]Agent {
	return map[string]Agent{
		"A1":  NewProductScoutAgent(),
		"A2":  NewListingOptimizerAgent(),
		"A3":  NewAdAdviceAgent(),
		"A4":  NewCustomerServiceAgent(),
		"A5":  NewInventoryAlertAgent(db, logger),
		"A6":  NewProfitWatchAgent(db, logger),
		"A7":  NewComplianceGuardAgent(),
		"A8":  NewSourcingAgent(),
		"A9":  NewBatchOpsAgent(),
		"A10": NewLogisticsOpsAgent(db, logger),
		"A11": NewAftersalesMgmtAgent(db, logger),
		"G0":  NewCoordinatorAgent(db, logger),
		"G1":  NewDashboardAgent(),
		"G2":  NewWarehouseCustomsAgent(),
		"G3":  NewDiscountRiskAgent(db, logger),
	}
}
