// Package impl provides concrete agent implementations.
//
// ProfitWatchAgent implements A6 Profit Watch business logic ported from
// backend/app/agent/agents/profit_watch.py (Python FastAPI codebase).
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

	skuCode := safeString(ctx["sku_code"])
	sellingPrice := safeFloat(ctx["selling_price"])
	costPrice := safeFloat(ctx["cost_price"])
	platform := safeString(ctx["platform"], "unknown")
	country := safeString(ctx["country"], "unknown")
	minMarginThreshold := safeFloat(ctx["min_margin_threshold"], 15.0)

	// ── Fee calculation ──

	// Platform commission.
	platformFeeRate := safeFloat(ctx["platform_fee_rate"], 0)
	platformFee := safeFloat(ctx["platform_fee"], 0)
	if platformFee == 0 && platformFeeRate > 0 {
		platformFee = sellingPrice * platformFeeRate / 100
	}
	platformFeeR := round2(platformFee)

	// Fixed fee.
	fixedFee := round2(safeFloat(ctx["fixed_fee"], 0))

	// Shipping fee.
	shippingFee := round2(safeFloat(ctx["shipping_fee"], 0))

	// Discount amortization.
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
		}
	} else if _, ok := ctx["discount_rate"]; ok {
		discountRate = safeFloat(ctx["discount_rate"], 0)
	}
	discountRateR := round2(discountRate)
	discountAmount := sellingPrice * discountRate / 100
	discountAmountR := round2(discountAmount)

	// Ad cost per unit.
	adCost := round2(safeFloat(ctx["ad_cost_per_unit"], 0))

	// Refund cost (estimated by refund rate applied to selling price).
	refundRate := safeFloat(ctx["refund_rate"], 0)
	refundCost := round2(sellingPrice * refundRate / 100)

	// Total fees.
	totalFees := platformFeeR + fixedFee + shippingFee + discountAmountR + adCost + refundCost
	feesTotal := round2(totalFees)

	// ── Profit calculation ──
	effectiveRevenue := sellingPrice - discountAmount
	profitPerUnit := round2(effectiveRevenue - costPrice - feesTotal)
	grossMargin := 0.0
	if effectiveRevenue > 0 {
		grossMargin = round2(profitPerUnit / effectiveRevenue * 100)
	}

	// ── Risk assessment ──
	isLoss := profitPerUnit < 0
	belowThreshold := grossMargin < minMarginThreshold

	anomalyReason := ""
	optimizationSuggestions := []string{}

	if isLoss {
		anomalyReason = fmt.Sprintf(
			"单件亏损 ¥%.2f，营收 ¥%.2f 不足以覆盖成本 ¥%.2f + 费用 ¥%.2f",
			absf(profitPerUnit), effectiveRevenue, costPrice, feesTotal,
		)
		optimizationSuggestions = []string{
			"考虑提高售价",
			"降低采购成本",
			"减少折扣力度",
			"优化物流渠道降低成本",
		}
		confidence = 0.95
		riskLevel = "high"
	} else if belowThreshold {
		anomalyReason = fmt.Sprintf(
			"毛利率 %.2f%% 低于阈值 %.2f%%，建议优化成本结构",
			grossMargin, minMarginThreshold,
		)
		optimizationSuggestions = []string{
			"适当提高售价",
			"检查平台佣金是否有优化空间",
			"评估广告成本是否可控",
		}
		confidence = 0.88
		riskLevel = "medium"
	} else {
		anomalyReason = "毛利率正常，在安全范围内"
		optimizationSuggestions = []string{"维持当前策略，定期监控"}
		confidence = 0.85
		riskLevel = "low"
	}

	// ── Fee-ratio warnings ──
	feeWarnings := []string{}
	costRatioThreshold := 0.5
	if effectiveRevenue > 0 && feesTotal > effectiveRevenue*costRatioThreshold {
		feeWarnings = append(feeWarnings,
			fmt.Sprintf("总费用占比过高(%.0f%%)", feesTotal/effectiveRevenue*100))
	}
	if effectiveRevenue > 0 && platformFee > effectiveRevenue*0.2 {
		feeWarnings = append(feeWarnings,
			fmt.Sprintf("平台佣金(%.2f)占比较高", platformFee))
	}
	if effectiveRevenue > 0 && shippingFee > effectiveRevenue*0.25 {
		feeWarnings = append(feeWarnings,
			fmt.Sprintf("物流费用(%.2f)占比较高", shippingFee))
	}

	// ── Status determination ──
	status := "allow"
	if isLoss {
		status = "block"
	} else if belowThreshold {
		status = "warn"
	}

	feeBreakdown := map[string]interface{}{
		"platform_fee": platformFeeR,
		"fixed_fee":    fixedFee,
		"shipping_fee": shippingFee,
		"discount":     discountAmountR,
		"ad_cost":      adCost,
		"refund_cost":  refundCost,
		"total":        feesTotal,
	}

	output = map[string]interface{}{
		"profit_check_status":      status,
		"sku_code":                 skuCode,
		"platform":                 platform,
		"country":                  country,
		"selling_price":            sellingPrice,
		"cost_price":               costPrice,
		"effective_revenue":        round2(effectiveRevenue),
		"discount_rate":            discountRateR,
		"profit_per_unit":          profitPerUnit,
		"gross_margin":             grossMargin,
		"min_margin_threshold":     minMarginThreshold,
		"is_loss":                  isLoss,
		"below_threshold":          belowThreshold,
		"fee_breakdown":            feeBreakdown,
		"fee_warnings":             feeWarnings,
		"anomaly_reason":           anomalyReason,
		"optimization_suggestions": optimizationSuggestions,
		"confidence":               confidence,
		"decision_point":           decisionPoint,
	}

	return output, confidence, riskLevel, nil
}

// ---------- Decision point: costOptimization ----------

// suggestCostOptimization analyzes the cost structure for a SKU and generates
// price-increase and/or cost-reduction suggestions when the current margin
// falls below the target margin.
func (a *ProfitWatchAgent) suggestCostOptimization(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Enrich context from DB if possible.
	a.fillSkuFromDB(ctx)

	// Validate required fields.
	if missing := missingFields(ctx, profitWatchRequiredFields); len(missing) > 0 {
		return insufficientData("cost_optimization", missing), 0.0, "low", nil
	}

	skuCode := safeString(ctx["sku_code"])
	sellingPrice := safeFloat(ctx["selling_price"])
	costPrice := safeFloat(ctx["cost_price"])

	currentMargin := 0.0
	if sellingPrice > 0 {
		currentMargin = round2((sellingPrice - costPrice) / sellingPrice * 100)
	}
	targetMargin := safeFloat(ctx["target_margin"], 20.0)

	suggestions := make([]map[string]interface{}, 0)

	if currentMargin < targetMargin {
		// Price increase: calculate the revenue needed to hit target margin.
		neededRevenue := costPrice / (1 - targetMargin/100)
		priceSuggest := round2(neededRevenue)
		if priceSuggest > sellingPrice {
			increasePct := round2((priceSuggest - sellingPrice) / sellingPrice * 100)
			suggestions = append(suggestions, map[string]interface{}{
				"type":           "price_increase",
				"current_price":  sellingPrice,
				"suggested_price": priceSuggest,
				"increase_pct":   increasePct,
				"description": fmt.Sprintf("提价至 ¥%.2f 可达到 %.2f%% 毛利率",
					priceSuggest, targetMargin),
			})
		}

		// Cost reduction: calculate the max cost that hits target margin.
		neededCost := sellingPrice * (1 - targetMargin/100)
		if neededCost < costPrice {
			reductionPct := round2((costPrice - neededCost) / costPrice * 100)
			suggestions = append(suggestions, map[string]interface{}{
				"type":          "cost_reduction",
				"current_cost":  costPrice,
				"target_cost":   round2(neededCost),
				"reduction_pct": reductionPct,
				"description":   fmt.Sprintf("采购成本需降至 ¥%.2f 以下", round2(neededCost)),
			})
		}
	}

	output = map[string]interface{}{
		"sku_code":       skuCode,
		"current_margin": currentMargin,
		"target_margin":  targetMargin,
		"suggestions":    suggestions,
		"confidence":     0.85,
		"decision_point": "cost_optimization",
	}

	return output, 0.85, "medium", nil
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
