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
	"fmt"
	"math"
	"strings"

	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Required and optional context field names ----------

var (
	discountCheckRequiredFields          = []string{"sku_code", "cost_price", "selling_price"}
	promotionValidationRequiredFields    = []string{"promotion", "selling_price"}
)

// ---------- DiscountRiskAgent ----------

// DiscountRiskAgent implements G3 Discount Risk logic.
// It simulates multi-discount stacking, validates individual promotions,
// checks platform price risk, and determines whether a discount should
// be blocked, warned, or allowed based on margin impact.
type DiscountRiskAgent struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewDiscountRiskAgent creates a new DiscountRiskAgent.
// The db handle is used for optional DB enrichment (SKU lookup).
func NewDiscountRiskAgent(db *gorm.DB, logger *zap.Logger) *DiscountRiskAgent {
	return &DiscountRiskAgent{db: db, logger: logger}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "discount_check"       — multi-discount stacking simulation and margin risk analysis
//   - "promotion_validation" — single promotion validation with special event handling
//   - "discount_risk_check"  — alias for discount_check
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high/critical), error.
func (a *DiscountRiskAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "discount_check", "discount_risk_check":
		return a.checkDiscountRisk(decisionPoint, ctx)
	case "promotion_validation":
		return a.validatePromotion(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: checkDiscountRisk (core multi-discount risk) ----------

// checkDiscountRisk is the core discount stacking risk logic. It:
//   - Validates required fields (sku_code, cost_price, selling_price)
//   - Enriches context from DB (SKU lookup)
//   - Simulates multiple discount stacking to produce a final price
//   - Computes gross profit and gross margin
//   - Applies block/warn/allow decision logic
//   - Performs platform-specific price floor checks
//   - Returns structured output with alerts, discount details, and margin info
func (a *DiscountRiskAgent) checkDiscountRisk(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Enrich context from DB if possible.
	a.fillSkuFromDB(ctx)

	// Validate required fields.
	if missing := missingFields(ctx, discountCheckRequiredFields); len(missing) > 0 {
		return insufficientData(decisionPoint, missing), 0.0, "low", nil
	}

	skuCode := safeString(ctx["sku_code"])
	asin := safeString(ctx["asin"])
	sellingPrice := safeFloat(ctx["selling_price"])
	costPrice := safeFloat(ctx["cost_price"])
	platform := safeString(ctx["platform"], "unknown")
	minMarginThreshold := safeFloat(ctx["min_margin_threshold"], 10.0)

	// Extract active discounts from context.
	activeDiscounts := extractDiscountList(ctx["active_discounts"])

	// Simulate multi-discount stacking.
	finalPrice, discountDetails := a.simulateDiscounts(sellingPrice, activeDiscounts)

	// Compute gross profit and margin.
	grossProfit := round2(finalPrice - costPrice)
	grossMargin := 0.0
	if finalPrice > 0 {
		grossMargin = round2(grossProfit / finalPrice * 100)
	}

	// Total discount rate.
	totalDiscountRate := 0.0
	if sellingPrice > 0 {
		totalDiscountRate = round2((1 - finalPrice/sellingPrice) * 100)
	}

	// ---- Decision logic ----
	blocked := false
	action := "allow"
	reason := ""
	alerts := make([]map[string]interface{}, 0)
	riskLevel = "low"

	switch {
	case finalPrice <= 0:
		action = "block"
		blocked = true
		reason = fmt.Sprintf("折后价 ¥%.2f ≤ ¥0，售价无效", finalPrice)
		alerts = append(alerts, map[string]interface{}{
			"level":   "critical",
			"message": reason,
		})
		confidence = 0.99
		riskLevel = "critical"
	case finalPrice < costPrice:
		action = "block"
		blocked = true
		reason = fmt.Sprintf(
			"折后价 ¥%.2f < 成本价 ¥%.2f，亏损 ¥%.2f，已自动阻断",
			finalPrice, costPrice, absf(grossProfit),
		)
		alerts = append(alerts, map[string]interface{}{
			"level":       "error",
			"message":     reason,
			"gross_loss":  round2(absf(grossProfit)),
		})
		confidence = 0.97
		riskLevel = "high"
	case finalPrice < costPrice*1.1:
		action = "warn"
		marginAboveCost := (finalPrice - costPrice) / costPrice * 100
		reason = fmt.Sprintf(
			"折后价 ¥%.2f 仅高于成本价 %.1f%%，低于安全阈值 (成本×1.1 = ¥%.2f)，建议人工复核",
			finalPrice, marginAboveCost, costPrice*1.1,
		)
		alerts = append(alerts, map[string]interface{}{
			"level":          "warning",
			"message":        reason,
			"min_safe_price": round2(costPrice * 1.1),
		})
		confidence = 0.90
		riskLevel = "medium"
	case grossMargin < minMarginThreshold:
		action = "warn"
		reason = fmt.Sprintf(
			"毛利率 %.2f%% < 最低阈值 %.2f%%，建议优化折扣或提价",
			grossMargin, minMarginThreshold,
		)
		alerts = append(alerts, map[string]interface{}{
			"level":     "warning",
			"message":   reason,
			"threshold": minMarginThreshold,
		})
		confidence = 0.85
		riskLevel = "medium"
	default:
		reason = fmt.Sprintf("折后毛利率 %.2f%%，高于阈值 %.2f%%，放行", grossMargin, minMarginThreshold)
		confidence = 0.85
		riskLevel = "low"
	}

	// ---- Platform price floor risk check ----
	platformAlert := a.checkPlatformPriceRisk(platform, finalPrice, sellingPrice)
	if platformAlert != nil {
		alerts = append(alerts, platformAlert)
		confidence = math.Max(0.80, confidence-0.05)
	}

	output = map[string]interface{}{
		"action":              action,
		"blocked":             blocked,
		"reason":              reason,
		"ai_explanation":      "", // stub; LLM explanation not yet wired

		"final_price":         round2(finalPrice),
		"original_price":      sellingPrice,
		"cost_price":          costPrice,
		"gross_profit":        grossProfit,
		"gross_margin":        grossMargin,
		"total_discount_rate": totalDiscountRate,
		"discount_count":      len(activeDiscounts),
		"discount_details":    discountDetails,

		"sku_code":             skuCode,
		"asin":                 asin,
		"platform":             platform,
		"min_margin_threshold": minMarginThreshold,

		"alerts":         alerts,
		"confidence":     confidence,
		"decision_point": decisionPoint,
		"status":         action,
	}

	return output, confidence, riskLevel, nil
}

// ---------- Decision point: promotionValidation ----------

// validatePromotion validates a single promotion against current pricing.
// It handles special event logic (e.g., Prime Day) where stricter or
// relaxed rules may apply depending on the scenario.
func (a *DiscountRiskAgent) validatePromotion(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Validate required fields.
	if missing := missingFields(ctx, promotionValidationRequiredFields); len(missing) > 0 {
		return insufficientData("promotion_validation", missing), 0.0, "low", nil
	}

	// Extract promotion details.
	promotion, ok := ctx["promotion"].(map[string]interface{})
	if !ok {
		return insufficientData("promotion_validation", []string{"promotion"}), 0.0, "low", nil
	}

	sellingPrice := safeFloat(ctx["selling_price"])
	costPrice := safeFloat(ctx["cost_price"])
	platform := safeString(ctx["platform"], "unknown")
	isPrimeDay := false
	if v, ok := ctx["is_prime_day"]; ok {
		if b, ok := v.(bool); ok {
			isPrimeDay = b
		}
	}

	pType := safeString(promotion["type"])
	pValue := safeFloat(promotion["value"])

	// Compute single-discount final price.
	finalPrice, detailStr := a.applySingleDiscount(sellingPrice, pType, pValue, promotion)

	// Compute gross profit and margin.
	grossProfit := round2(finalPrice - costPrice)
	grossMargin := 0.0
	if finalPrice > 0 {
		grossMargin = round2(grossProfit / finalPrice * 100)
	}

	alerts := make([]map[string]interface{}, 0)
	blocked := false
	action := "allow"
	reason := ""

	// Decision logic with Prime Day special handling.
	switch {
	case isPrimeDay:
		if finalPrice < costPrice {
			action = "warn"
			reason = fmt.Sprintf("大促期间折后价 ¥%.2f < 成本 ¥%.2f，请运营确认是否继续", finalPrice, costPrice)
			confidence = 0.85
			riskLevel = "medium"
		} else {
			action = "allow_special"
			reason = fmt.Sprintf("大促期间特殊放行，毛利率 %.2f%%", grossMargin)
			confidence = 0.90
			riskLevel = "low"
		}
	case finalPrice <= 0:
		action = "block"
		blocked = true
		reason = "促销后售价 ≤ ¥0，已自动阻断"
		confidence = 0.99
		riskLevel = "critical"
	case finalPrice < costPrice:
		action = "block"
		blocked = true
		reason = fmt.Sprintf("促销后售价 ¥%.2f < 成本 ¥%.2f，已阻断", finalPrice, costPrice)
		confidence = 0.97
		riskLevel = "high"
	case finalPrice < costPrice*1.1:
		action = "warn"
		reason = fmt.Sprintf("促销后毛利率仅 %.2f%%，低于安全阈值", grossMargin)
		confidence = 0.88
		riskLevel = "medium"
	default:
		action = "allow"
		reason = fmt.Sprintf("促销后毛利率 %.2f%%，放行", grossMargin)
		confidence = 0.92
		riskLevel = "low"
	}

	output = map[string]interface{}{
		"action":   action,
		"blocked":  blocked,
		"reason":   reason,

		"final_price":    round2(finalPrice),
		"original_price": sellingPrice,
		"cost_price":     costPrice,
		"gross_profit":   grossProfit,
		"gross_margin":   grossMargin,

		"discount_type":        pType,
		"discount_value":       pValue,
		"discount_description": detailStr,

		"platform":     platform,
		"is_prime_day": isPrimeDay,

		"alerts":         alerts,
		"confidence":     confidence,
		"decision_point": "promotion_validation",
		"status":         action,
	}

	return output, confidence, riskLevel, nil
}

// ---------- Internal simulation methods ----------

// simulateDiscounts applies multiple discount types sequentially and returns the
// final price and a list of discount detail records.
//
// Strategy (ported from Python G3 agent):
//   - Percentage discounts (percentage/coupon/promotion/member_discount) are
//     applied as compound: each percentage reduces the running price.
//   - Fixed-amount discounts (fixed/fixed_amount) subtract from the running price.
//   - BuyXGetY (buy_x_get_y/bogo) is converted to an effective percentage rate.
//   - percentage_no_compound discounts are recorded but not applied (non-stacking).
func (a *DiscountRiskAgent) simulateDiscounts(basePrice float64, discounts []map[string]interface{}) (float64, []map[string]interface{}) {
	price := basePrice
	details := make([]map[string]interface{}, 0)

	for _, d := range discounts {
		dType := strings.ToLower(safeString(d["type"]))
		dValue := safeFloat(d["value"])

		switch {
		case dType == "percentage" || dType == "coupon" || dType == "promotion" || dType == "member_discount":
			// Compound percentage discount.
			if dValue > 0 && dValue < 100 {
				discountAmount := price * dValue / 100
				price -= discountAmount
				details = append(details, map[string]interface{}{
					"type":            dType,
					"value":           dValue,
					"unit":            "%",
					"description":     fmt.Sprintf("%s %.0f%% 折扣", dType, dValue),
					"discount_amount": round2(discountAmount),
				})
			}

		case dType == "fixed" || dType == "fixed_amount":
			// Fixed amount discount.
			if dValue > 0 {
				discountAmount := math.Min(dValue, price)
				price -= discountAmount
				details = append(details, map[string]interface{}{
					"type":            "fixed_amount",
					"value":           dValue,
					"unit":            "¥",
					"description":     fmt.Sprintf("固定减免 ¥%.2f", dValue),
					"discount_amount": round2(discountAmount),
				})
			}

		case dType == "buy_x_get_y" || dType == "bogo":
			// Buy X Get Y — convert to effective discount rate.
			buyQty := safeFloat(d["buy_qty"])
			if buyQty <= 0 {
				buyQty = safeFloat(d["buy"], 2)
			}
			freeQty := safeFloat(d["free_qty"])
			if freeQty <= 0 {
				freeQty = safeFloat(d["free"], 1)
			}
			if buyQty > 0 && freeQty > 0 {
				effectiveRate := freeQty / (buyQty + freeQty) * 100
				discountAmount := price * effectiveRate / 100
				price -= discountAmount
				details = append(details, map[string]interface{}{
					"type":            "buy_x_get_y",
					"buy_qty":         buyQty,
					"free_qty":        freeQty,
					"effective_rate":  round2(effectiveRate),
					"description":     fmt.Sprintf("买%.0f送%.0f (等效 %.1f%% 折扣)", buyQty, freeQty, effectiveRate),
					"discount_amount": round2(discountAmount),
				})
			}

		case dType == "percentage_no_compound":
			// Non-compound percentage discount: record but do not apply.
			if dValue > 0 && dValue < 100 {
				details = append(details, map[string]interface{}{
					"type":            "percentage_no_compound",
					"value":           dValue,
					"unit":            "%",
					"description":     fmt.Sprintf("独立折扣 %.0f%%（非叠加）", dValue),
					"discount_amount": 0.0,
				})
			}
		}
	}

	// Clamp to zero.
	if price < 0 {
		price = 0
	}

	return price, details
}

// applySingleDiscount applies a single promotion discount and returns the final
// price and a human-readable description string.
//
// Supported types:
//   - percentage: base * (1 - d_value/100)
//   - fixed: base - d_value
//   - buy_x_get_y: base * (1 - free_qty/(buy_qty+free_qty))
//   - others: unchanged with "unknown" description
func (a *DiscountRiskAgent) applySingleDiscount(basePrice float64, dType string, dValue float64, promotion map[string]interface{}) (float64, string) {
	var finalPrice float64
	var detail string

	switch dType {
	case "percentage":
		finalPrice = basePrice * (1 - dValue/100)
		detail = fmt.Sprintf("%.0f%% 折扣", dValue)
	case "fixed":
		finalPrice = basePrice - dValue
		detail = fmt.Sprintf("减 ¥%.2f", dValue)
	case "buy_x_get_y":
		buyQty := safeFloat(promotion["buy_qty"])
		if buyQty <= 0 {
			buyQty = safeFloat(promotion["buy"], 2)
		}
		freeQty := safeFloat(promotion["free_qty"])
		if freeQty <= 0 {
			freeQty = safeFloat(promotion["free"], 1)
		}
		effectiveRate := 0.0
		if buyQty+freeQty > 0 {
			effectiveRate = freeQty / (buyQty + freeQty)
		}
		finalPrice = basePrice * (1 - effectiveRate)
		detail = fmt.Sprintf("买%.0f送%.0f", buyQty, freeQty)
	default:
		finalPrice = basePrice
		detail = "未知促销类型"
	}

	if finalPrice < 0 {
		finalPrice = 0
	}

	return finalPrice, detail
}

// ---------- Platform risk check ----------

// checkPlatformPriceRisk checks whether the final price falls below the
// platform's typical minimum price threshold (70% of original price).
// Only applies to known risk platforms.
func (a *DiscountRiskAgent) checkPlatformPriceRisk(platform string, finalPrice float64, originalPrice float64) map[string]interface{} {
	riskPlatforms := map[string]bool{
		"amazon":  true,
		"walmart": true,
		"ebay":    true,
		"shopify": true,
	}
	if !riskPlatforms[strings.ToLower(platform)] {
		return nil
	}
	if originalPrice > 0 && finalPrice < originalPrice*0.7 {
		ratio := math.Round(finalPrice / originalPrice * 100)
		return map[string]interface{}{
			"level": "info",
			"message": fmt.Sprintf(
				"折后价仅为原价的 %.0f%%，请确认该平台允许的最低折扣幅度", ratio,
			),
		}
	}
	return nil
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

// ---------- Helpers ----------

// extractDiscountList safely extracts a slice of discount maps from the
// context value, which arrives as []interface{} from JSON unmarshalling.
func extractDiscountList(v interface{}) []map[string]interface{} {
	if v == nil {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result
}
