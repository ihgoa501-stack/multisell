// Package impl provides concrete agent implementations.
//
// ProfitWatchAgent implements A6 Profit Watch. The agent enriches input
// context from the database (SKU lookup, platform fee rules) and then
// delegates the pure-computation business logic (fee breakdown, margin
// analysis, cost-optimization) to tools registered on
// toolregistry.DefaultRegistry.
//
// Design docs: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md section 7.1.4
//   - Input: SKU code, selling price, cost price, platform, country, fees
//   - Output: per-unit profit, gross margin, fee breakdown, loss risk, price suggestions
//   - Insufficient data returns insufficient_data status
package impl

import (
	"fmt"
	"math"
	"strings"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/platformfee"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Required and optional context field names ----------

var (
	profitWatchRequiredFields = []string{"sku_code", "selling_price", "cost_price"}
	profitWatchOptionalFields = []string{
		"platform", "country",
		"weight_kg", "length", "width", "height",
		"shipping_fee", "platform_fee", "platform_fee_rate",
		"fixed_fee", "discounts", "ad_cost_per_unit", "refund_rate",
	}
)

// ---------- ProfitWatchAgent ----------

// ProfitWatchAgent implements A6 Profit Watch logic.
// It computes per-SKU profit, fee breakdown, margin analysis,
// loss detection, and cost-optimization suggestions.
type ProfitWatchAgent struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewProfitWatchAgent creates a new ProfitWatchAgent.
// The db handle is used for optional DB enrichment (SKU lookup, platform fee rules).
func NewProfitWatchAgent(db *gorm.DB, logger *zap.Logger) *ProfitWatchAgent {
	return &ProfitWatchAgent{db: db, logger: logger}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "profit_check"  — per-SKU profit margin analysis and loss detection
//   - "profit_watch"  — alias for profit_check (continuous monitoring context)
//   - "cost_optimization" — cost structure analysis with price/cost suggestions
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *ProfitWatchAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "profit_check", "profit_watch":
		return a.checkProfit(decisionPoint, ctx)
	case "cost_optimization":
		return a.suggestCostOptimization(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Type-safe context accessors ----------

// safeFloat extracts a float64 from an interface{} value.
// Returns defaultVal (0.0) on nil, missing key, or conversion failure.
func safeFloat(v interface{}, defaultVal ...float64) float64 {
	def := 0.0
	if len(defaultVal) > 0 {
		def = defaultVal[0]
	}
	if v == nil {
		return def
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case uint32:
		return float64(val)
	}
	return def
}

// safeString extracts a string from an interface{} value.
// Returns defaultVal ("") on nil, missing key, or conversion failure.
func safeString(v interface{}, defaultVal ...string) string {
	def := ""
	if len(defaultVal) > 0 {
		def = defaultVal[0]
	}
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

// round2 rounds a float64 to 2 decimal places using Go's round-half-away-from-zero.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// missingFields returns context keys that are missing (absent, nil, or empty string).
func missingFields(ctx map[string]interface{}, fields []string) []string {
	var missing []string
	for _, f := range fields {
		v, ok := ctx[f]
		if !ok || v == nil {
			missing = append(missing, f)
		} else if s, ok := v.(string); ok && s == "" {
			missing = append(missing, f)
		}
	}
	return missing
}

// ---------- DB enrichment ----------

// fillSkuFromDB looks up the SKU record by code and fills missing context fields.
//
// This mirrors Python AgentDataService.fill_sku_context: it enriches the context
// with cost_price, selling_price, weight, and dimensions from the sku table when
// those fields are not already provided in the input context.
func (a *ProfitWatchAgent) fillSkuFromDB(ctx map[string]interface{}) {
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
	if _, ok := ctx["weight_kg"]; !ok {
		if w, exact := s.Weight.Float64(); exact {
			ctx["weight_kg"] = w
		}
	}
	if _, ok := ctx["sku_length"]; !ok {
		if l, exact := s.SkuLengthCm.Float64(); exact {
			ctx["sku_length"] = l
		}
	}
	if _, ok := ctx["sku_width"]; !ok {
		if w, exact := s.SkuWidthCm.Float64(); exact {
			ctx["sku_width"] = w
		}
	}
	if _, ok := ctx["sku_height"]; !ok {
		if h, exact := s.SkuHeightCm.Float64(); exact {
			ctx["sku_height"] = h
		}
	}
}

// fillPlatformFeeFromDB looks up active commission rules for the given platform/country
// and sets platform_fee_rate in the context if not already provided.
func (a *ProfitWatchAgent) fillPlatformFeeFromDB(ctx map[string]interface{}) {
	if a.db == nil {
		return
	}
	if _, ok := ctx["platform_fee_rate"]; ok {
		return
	}
	platform := safeString(ctx["platform"])
	if platform == "" {
		return
	}
	country := safeString(ctx["country"])
	if country == "" {
		return
	}
	// Find active commission-type fee rules.
	var rules []platformfee.PlatformFeeRule
	if err := a.db.Where("status = ? AND fee_type = ?", "active", "commission").
		Find(&rules).Error; err != nil {
		a.logger.Debug("platform fee rule lookup failed", zap.Error(err))
		return
	}
	for _, r := range rules {
		// Match by country code (exact or wildcard).
		if r.CountryCode == "" || strings.EqualFold(r.CountryCode, country) {
			if r.FeeRatePct > 0 {
				ctx["platform_fee_rate"] = r.FeeRatePct
				return
			}
		}
	}
}

// ---------- Decision point: checkProfit ----------

// checkProfit is the core profit check logic. It computes:
//   - Fee breakdown (platform, fixed, shipping, discount, ad cost, refund)
//   - Per-unit profit and gross margin
//   - Loss / below-threshold detection
//   - Optimization suggestions
//   - Fee-ratio warnings
func (a *ProfitWatchAgent) checkProfit(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Enrich context from DB if possible.
	a.fillSkuFromDB(ctx)
	a.fillPlatformFeeFromDB(ctx)

	// Validate required fields.
	if missing := missingFields(ctx, profitWatchRequiredFields); len(missing) > 0 {
		return insufficientData(decisionPoint, missing), 0.0, "low", nil
	}

		sellingPrice := safeFloat(ctx["selling_price"])

	// Compute discount_rate from discounts array if present (tools cannot
	// handle the complex discounts structure; we flatten it here).
	discountRate := 0.0
	if discounts, ok := ctx["discounts"]; ok {
		if discountList, ok := discounts.([]interface{}); ok && len(discountList) > 0 {
			for _, d := range discountList {
				if dMap, ok := d.(map[string]interface{}); ok {
					dVal := safeFloat(dMap["value"], 0)
					dType := strings.ToLower(safeString(dMap["type"], ""))
					switch dType {
					case "percentage", "coupon", "promotion", "member_discount":
						discountRate += dVal
					case "fixed", "fixed_amount":
						if sellingPrice > 0 {
							discountRate += dVal / sellingPrice * 100
						}
					}
				}
			}
			if discountRate > 100 {
				discountRate = 100
			}
			ctx["discount_rate"] = discountRate
		}
	}

	// Build the tool input from the enriched context.
	toolInput := map[string]interface{}{
		"sku_code":             ctx["sku_code"],
		"selling_price":        ctx["selling_price"],
		"cost_price":           ctx["cost_price"],
		"platform":             ctx["platform"],
		"country":              ctx["country"],
		"platform_fee_rate":    ctx["platform_fee_rate"],
		"platform_fee":         ctx["platform_fee"],
		"fixed_fee":            ctx["fixed_fee"],
		"shipping_fee":         ctx["shipping_fee"],
		"discount_rate":        ctx["discount_rate"],
		"ad_cost_per_unit":     ctx["ad_cost_per_unit"],
		"refund_rate":          ctx["refund_rate"],
		"min_margin_threshold": ctx["min_margin_threshold"],
	}

	// Delegate computation to the tool.
	rawResult, invokeErr := toolregistry.DefaultRegistry.Invoke("profit_watch.check_profit", toolInput)
	if invokeErr != nil {
		a.logger.Error("profit_watch.check_profit tool invocation failed", zap.Error(invokeErr))
		return map[string]interface{}{
			"status":         "error",
			"decision_point": decisionPoint,
			"error":          invokeErr.Error(),
		}, 0.0, "low", invokeErr
	}

	result, ok := rawResult.(map[string]interface{})
	if !ok {
		a.logger.Error("profit_watch.check_profit tool returned unexpected type")
		return map[string]interface{}{
			"status":         "error",
			"decision_point": decisionPoint,
			"error":          "tool returned unexpected type",
		}, 0.0, "low", fmt.Errorf("unexpected tool result type: %T", rawResult)
	}

	// Extract confidence and risk level from the tool result.
	confidence = safeFloat(result["confidence"], 0.85)
	riskLevel = safeString(result["risk_level"], "low")

	// Build the full output in the standard envelope.
	output = map[string]interface{}{
		"profit_check_status":      result["profit_check_status"],
		"sku_code":                 result["sku_code"],
		"platform":                 result["platform"],
		"country":                  result["country"],
		"selling_price":            result["selling_price"],
		"cost_price":               result["cost_price"],
		"effective_revenue":        result["effective_revenue"],
		"discount_rate":            result["discount_rate"],
		"profit_per_unit":          result["profit_per_unit"],
		"gross_margin":             result["gross_margin"],
		"min_margin_threshold":     result["min_margin_threshold"],
		"is_loss":                  result["is_loss"],
		"below_threshold":          result["below_threshold"],
		"fee_breakdown":            result["fee_breakdown"],
		"fee_warnings":             result["fee_warnings"],
		"anomaly_reason":           result["anomaly_reason"],
		"optimization_suggestions": result["optimization_suggestions"],
		"confidence":               confidence,
		"decision_point":           decisionPoint,
	}

	return output, confidence, riskLevel, nil
}

// ---------- Decision point: costOptimization ----------

// suggestCostOptimization analyzes the cost structure for a SKU and generates
// price-increase and/or cost-reduction suggestions by delegating to the
// profit_watch.cost_optimization tool.
func (a *ProfitWatchAgent) suggestCostOptimization(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Enrich context from DB if possible.
	a.fillSkuFromDB(ctx)

	// Validate required fields.
	if missing := missingFields(ctx, profitWatchRequiredFields); len(missing) > 0 {
		return insufficientData("cost_optimization", missing), 0.0, "low", nil
	}

	// Build the tool input from the enriched context.
	toolInput := map[string]interface{}{
		"sku_code":      ctx["sku_code"],
		"selling_price": ctx["selling_price"],
		"cost_price":    ctx["cost_price"],
		"target_margin": ctx["target_margin"],
	}

	// Delegate computation to the tool.
	rawResult, invokeErr := toolregistry.DefaultRegistry.Invoke("profit_watch.cost_optimization", toolInput)
	if invokeErr != nil {
		a.logger.Error("profit_watch.cost_optimization tool invocation failed", zap.Error(invokeErr))
		return map[string]interface{}{
			"status":         "error",
			"decision_point": "cost_optimization",
			"error":          invokeErr.Error(),
		}, 0.0, "low", invokeErr
	}

	result, ok := rawResult.(map[string]interface{})
	if !ok {
		a.logger.Error("profit_watch.cost_optimization tool returned unexpected type")
		return map[string]interface{}{
			"status":         "error",
			"decision_point": "cost_optimization",
			"error":          "tool returned unexpected type",
		}, 0.0, "low", fmt.Errorf("unexpected tool result type: %T", rawResult)
	}

	// Map the tool result to the output envelope.
	confidence = safeFloat(result["confidence"], 0.85)
	riskLevel = safeString(result["risk_level"], "medium")

	output = map[string]interface{}{
		"sku_code":       result["sku_code"],
		"current_margin": result["current_margin"],
		"target_margin":  result["target_margin"],
		"suggestions":    result["suggestions"],
		"confidence":     confidence,
		"decision_point": "cost_optimization",
	}

	return output, confidence, riskLevel, nil
}

// ---------- Helpers ----------

// insufficientData returns a structured map indicating the decision cannot
// proceed because required fields are missing.
func insufficientData(point string, missing []string) map[string]interface{} {
	return map[string]interface{}{
		"status":         "insufficient_data",
		"decision_point": point,
		"missing_fields": missing,
		"message":        fmt.Sprintf("缺少必要字段: %s，请补充完整数据", strings.Join(missing, ", ")),
		"confidence":     0.0,
	}
}

// absf returns the absolute value of a float64.
func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
