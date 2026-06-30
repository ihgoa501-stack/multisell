// Package impl provides concrete agent implementations.
//
// DiscountRiskAgent implements G3 Discount Risk business logic ported from
// backend/app/agent/agents/discount_risk.py (Python FastAPI codebase).
//
// Design docs: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md section 7.1.3
//   - Input: SKU code, cost price, selling price, active discount list, platform, minimum margin threshold
//   - Simulation: stacking of percentage, fixed, and BXGY discounts
//   - Output: final discounted price, gross profit, gross margin, block/warn/allow decision, alerts
//   - Insufficient data returns insufficient_data status
package impl

import (
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- DiscountRiskAgent ----------

// DiscountRiskAgent implements G3 Discount Risk logic.
// It delegates to the tool registry for discount_check, promotion_validation,
// and discount_risk_check decision points.
type DiscountRiskAgent struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewDiscountRiskAgent creates a new DiscountRiskAgent.
// The db handle is used for optional DB enrichment (SKU lookup).
func NewDiscountRiskAgent(db *gorm.DB, logger *zap.Logger) *DiscountRiskAgent {
	return &DiscountRiskAgent{db: db, logger: logger}
}

// callTool delegates a decision point to the tool registry and extracts
// the output map, confidence, and risk level from the result.
func (a *DiscountRiskAgent) callTool(callCtx context.Context, toolName string, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {

	result, callErr := toolregistry.DefaultRegistry.Call(callCtx, toolName, params)
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

	// Extract confidence and risk level from the output map.
	if c, ok := output["confidence"].(float64); ok {
		confidence = c
	}
	if r, ok := output["risk_level"].(string); ok {
		riskLevel = r
	} else {
		riskLevel = "low"
	}

	return output, confidence, riskLevel, nil
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "discount_check"       — multi-discount stacking simulation and margin risk analysis
//   - "promotion_validation" — single promotion validation with special event handling
//   - "discount_risk_check"  — alias for discount_check
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high/critical), error.
func (a *DiscountRiskAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "discount_check":
		// Optional DB enrichment before delegating to tool.
		a.fillSkuFromDB(params)
		return a.callTool(ctx, "discount.check", decisionPoint, params)
	case "discount_risk_check":
		// Optional DB enrichment before delegating to tool.
		a.fillSkuFromDB(params)
		return a.callTool(ctx, "discount.risk_check", decisionPoint, params)
	case "promotion_validation":
		return a.callTool(ctx, "discount.validate", decisionPoint, params)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- DB enrichment ----------

// fillSkuFromDB looks up the SKU record by code and fills missing context fields.
func (a *DiscountRiskAgent) fillSkuFromDB(ctx map[string]interface{}) {
	if a.db == nil {
		return
	}
	skuCode := safeString(ctx["sku_code"])
	if skuCode == "" {
		return
	}
	var s sku.Sku
	if err := a.db.Where("code = ?", skuCode).First(&s).Error; err != nil {
		a.logger.Debug("sku db lookup failed",
			zap.String("sku_code", skuCode),
			zap.Error(err),
		)
		return
	}
	a.logger.Debug("sku db enrichment",
		zap.String("sku_code", skuCode),
		zap.Int64("sku_id", s.ID),
	)

	// Only fill fields not already present in context.
	if _, ok := ctx["selling_price"]; !ok {
		if price, exact := s.Price.Float64(); exact {
			ctx["selling_price"] = price
		}
	}
	if _, ok := ctx["cost_price"]; !ok {
		if cost, exact := s.CostPrice.Float64(); exact {
			ctx["cost_price"] = cost
		}
	}
}
