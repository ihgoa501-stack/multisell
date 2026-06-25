// Package impl provides concrete agent implementations.
//
// AdAdviceAgent implements A3 Ad Advice business logic ported from
// backend/app/agent/agents/ad_advice.py (Python FastAPI codebase).
//
// Design docs: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md section 7.1.6
//   - Ad campaign performance analysis
//   - ACoS anomaly detection (critical/warning)
//   - CPC bid suggestions
//   - Negative keyword recommendations
//   - Budget usage alerts
//   - No automatic price changes or ad pausing (advice only)
package impl

import (
	"fmt"
	"math"
)

// ---------- Required and optional context field names ----------

var (
	acosAnalysisRequiredFields   = []string{"campaign_id", "spend", "sales"}
	adOptimizationRequiredFields = []string{"campaign_id"}
)

// ---------- AdAdviceAgent ----------

// AdAdviceAgent implements A3 Ad Advice logic.
//
// Decision points:
//   - "acos_analysis" — analyzes ACoS metrics, detects anomalies, provides bid suggestions
//   - "ad_optimization" — suggests optimization actions (negative keywords, bid, budget)
type AdAdviceAgent struct{}

// NewAdAdviceAgent creates a new AdAdviceAgent.
func NewAdAdviceAgent() *AdAdviceAgent {
	return &AdAdviceAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "acos_analysis"
//   - "ad_optimization"
func (a *AdAdviceAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "acos_analysis":
		return a.analyzeAcos(ctx)
	case "ad_optimization":
		return a.suggestAdOptimization(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: acos_analysis ----------

// analyzeAcos performs a detailed ACoS analysis including:
//   - Metric calculation (ACoS, CTR, CVR, CPC, budget usage)
//   - ACoS anomaly detection (critical if ACOS > gross margin, warning if ACOS > target)
//   - Budget consumption alerts
//   - Inventory risk (low/out_of_stock)
//   - Low CTR / low CVR alerts
//   - CPC bid suggestion when ACOS exceeds target
//
// Required context fields: campaign_id, spend, sales
// Optional context fields: clicks, impressions, conversions, budget,
// inventory_status, gross_margin, target_acos, search_terms
func (a *AdAdviceAgent) analyzeAcos(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, acosAnalysisRequiredFields); len(missing) > 0 {
		return insufficientData("acos_analysis", missing), 0.0, "low", nil
	}

	campaignID := safeString(ctx["campaign_id"], "")
	skuCode := safeString(ctx["sku_code"], "")
	if skuCode == "" {
		skuCode = safeString(ctx["asin"], "")
	}
	spend := safeFloat(ctx["spend"], 0)
	sales := safeFloat(ctx["sales"], 0)
	clicks := safeFloat(ctx["clicks"], 0)
	impressions := safeFloat(ctx["impressions"], 0)
	conversions := safeFloat(ctx["conversions"], 0)
	budget := safeFloat(ctx["budget"], 0)
	inventoryStatus := safeString(ctx["inventory_status"], "normal")
	grossMargin := safeFloat(ctx["gross_margin"], 0)
	targetAcos := safeFloat(ctx["target_acos"], 30.0)

	// Calculate metrics.
	acos := 0.0
	if sales > 0 {
		acos = round2(spend / sales * 100)
	}
	ctr := 0.0
	if impressions > 0 {
		ctr = round2(clicks / impressions * 100)
	}
	conversionRate := 0.0
	if clicks > 0 {
		conversionRate = round2(conversions / clicks * 100)
	}
	cpc := 0.0
	if clicks > 0 {
		cpc = round2(spend / clicks)
	}
	budgetUsage := 0.0
	if budget > 0 {
		budgetUsage = round2(spend / budget * 100)
	}

	// Risk assessment.
	alerts := make([]map[string]interface{}, 0)
	suggestions := make([]string, 0)
	status := "normal"
	conf := 0.85
	riskLvl := "low"
	acosAbnormal := false

	// ACoS anomaly detection.
	if acos > targetAcos {
		acosAbnormal = true
		if grossMargin > 0 && acos > grossMargin {
			status = "critical"
			riskLvl = "high"
			conf = 0.95
			alerts = append(alerts, map[string]interface{}{
				"level":   "critical",
				"message": fmt.Sprintf("ACoS (%.2f%%) 超过毛利率 (%.2f%%)，广告亏损", acos, grossMargin),
			})
			suggestions = append(suggestions, "建议暂停或大幅降低广告出价")
		} else {
			status = "warning"
			riskLvl = "medium"
			conf = 0.88
			alerts = append(alerts, map[string]interface{}{
				"level":   "warning",
				"message": fmt.Sprintf("ACoS (%.2f%%) 超过目标阈值 (%.2f%%)", acos, targetAcos),
			})
			suggestions = append(suggestions, "建议降低广告出价或优化否定关键词")
		}
	}

	// Budget consumption alerts.
	if budget > 0 && budgetUsage > 90 {
		alerts = append(alerts, map[string]interface{}{
			"level":   "info",
			"message": fmt.Sprintf("预算已使用 %.2f%%，接近上限", budgetUsage),
		})
	} else if budget > 0 && budgetUsage < 10 {
		alerts = append(alerts, map[string]interface{}{
			"level":   "info",
			"message": fmt.Sprintf("预算使用率仅 %.2f%%，建议检查广告是否正常投放", budgetUsage),
		})
	}

	// Inventory-related risk.
	if inventoryStatus == "low" || inventoryStatus == "out_of_stock" {
		alerts = append(alerts, map[string]interface{}{
			"level":   "warning",
			"message": fmt.Sprintf("库存状态为 %s，建议暂停广告避免浪费", inventoryStatus),
		})
		suggestions = append(suggestions, "库存不足时暂停广告")
	}

	// Low CTR / CVR alerts.
	if impressions > 0 && ctr < 0.5 {
		alerts = append(alerts, map[string]interface{}{
			"level":   "info",
			"message": fmt.Sprintf("点击率 (%.2f%%) 偏低，建议优化主图或标题", ctr),
		})
	}
	if clicks > 0 && conversionRate < 5 {
		alerts = append(alerts, map[string]interface{}{
			"level":   "info",
			"message": fmt.Sprintf("转化率 (%.2f%%) 偏低，建议检查 Listing 或价格", conversionRate),
		})
	}

	// CPC bid suggestion.
	var bidSuggestion map[string]interface{}
	if acos > targetAcos && sales > 0 {
		idealAcos := targetAcos * 0.8 // Safety buffer.
		suggestedSpend := sales * idealAcos / 100
		if clicks > 0 {
			suggestedCPC := round2(suggestedSpend / clicks)
			currentCPC := cpc
			if suggestedCPC < currentCPC {
				reductionPct := round2((currentCPC - suggestedCPC) / currentCPC * 100)
				bidSuggestion = map[string]interface{}{
					"current_cpc":   currentCPC,
					"suggested_cpc": suggestedCPC,
					"reduction_pct": reductionPct,
					"description":   fmt.Sprintf("建议 CPC 从 ¥%.2f 降至 ¥%.2f", currentCPC, suggestedCPC),
				}
			}
		}
	}

	output = map[string]interface{}{
		"status":      status,
		"campaign_id": campaignID,
		"sku_code":    skuCode,
		"ai_explanation": "", // LLM explanation not yet wired in Go
		"metrics": map[string]interface{}{
			"acos":            acos,
			"target_acos":     targetAcos,
			"ctr":             ctr,
			"conversion_rate": conversionRate,
			"cpc":             cpc,
			"budget_usage":    budgetUsage,
			"spend":           spend,
			"sales":           sales,
			"clicks":          int(clicks),
			"impressions":     int(impressions),
			"conversions":     int(conversions),
		},
		"acos_abnormal":  acosAbnormal,
		"alerts":         alerts,
		"suggestions":    suggestions,
		"bid_suggestion": bidSuggestion,
		"confidence":     conf,
	}
	return output, conf, riskLvl, nil
}

// ---------- Decision point: ad_optimization ----------

// suggestAdOptimization suggests optimization actions including:
//   - Negative keywords (terms with >=10 clicks and ACOS > target*1.5)
//   - Bid reduction (when ACOS > target)
//   - Budget increase (when budget usage > 90%)
//
// Required context fields: campaign_id
// Optional context fields: spend, sales, clicks, search_terms, budget, target_acos
func (a *AdAdviceAgent) suggestAdOptimization(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, adOptimizationRequiredFields); len(missing) > 0 {
		return insufficientData("ad_optimization", missing), 0.0, "low", nil
	}

	campaignID := safeString(ctx["campaign_id"], "")
	spend := safeFloat(ctx["spend"], 0)
	sales := safeFloat(ctx["sales"], 0)
	_ = safeFloat(ctx["clicks"], 0)
	targetAcos := safeFloat(ctx["target_acos"], 30.0)

	acos := 0.0
	if sales > 0 {
		acos = round2(spend / sales * 100)
	}

	optimizationItems := make([]map[string]interface{}, 0)

	// Negative keyword suggestions.
	searchTerms := ctx["search_terms"]
	if searchTerms != nil {
		if termsList, ok := searchTerms.([]interface{}); ok {
			negativeKeywords := make([]map[string]interface{}, 0)
			for _, raw := range termsList {
				term, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				termSpend := safeFloat(term["spend"], 0)
				termSales := safeFloat(term["sales"], 0)
				termClicks := safeFloat(term["clicks"], 0)
				termAcos := 0.0
				if termSales > 0 {
					termAcos = round2(termSpend / termSales * 100)
				}

				if termClicks >= 10 && termAcos > targetAcos*1.5 {
					negativeKeywords = append(negativeKeywords, map[string]interface{}{
						"keyword": safeString(term["keyword"], ""),
						"clicks":  int(termClicks),
						"acos":    termAcos,
						"reason":  fmt.Sprintf("ACoS %.2f%% 远超目标 %.2f%%，建议添加为否定关键词", termAcos, targetAcos),
					})
				}
			}

			if len(negativeKeywords) > 0 {
				topNK := 10
				if len(negativeKeywords) < topNK {
					topNK = len(negativeKeywords)
				}
				optimizationItems = append(optimizationItems, map[string]interface{}{
					"type":        "negative_keyword",
					"items":       negativeKeywords[:topNK],
					"description": fmt.Sprintf("发现 %d 个高 ACOS 搜索词，建议添加否定关键词", len(negativeKeywords)),
				})
			}
		}
	}

	// Bid reduction suggestion.
	if acos > targetAcos {
		reduction := round2(math.Min((acos-targetAcos)/acos*100, 50))
		optimizationItems = append(optimizationItems, map[string]interface{}{
			"type":                  "bid_reduction",
			"description":           fmt.Sprintf("建议降低出价 %.1f%%（当前 ACoS %.2f%%，目标 %.2f%%）", reduction, acos, targetAcos),
			"suggested_reduction_pct": reduction,
		})
	}

	// Budget increase suggestion.
	budget := safeFloat(ctx["budget"], 0)
	if budget > 0 && spend > budget*0.9 {
		usagePct := math.Round(spend / budget * 100)
		optimizationItems = append(optimizationItems, map[string]interface{}{
			"type":        "budget_increase",
			"description": fmt.Sprintf("预算即将耗尽（已用 %.0f%%），如效果良好建议增加预算", usagePct),
		})
	}

	output = map[string]interface{}{
		"campaign_id":       campaignID,
		"current_acos":      acos,
		"target_acos":       targetAcos,
		"optimization_items": optimizationItems,
		"confidence":        0.85,
	}
	return output, 0.85, "low", nil
}
