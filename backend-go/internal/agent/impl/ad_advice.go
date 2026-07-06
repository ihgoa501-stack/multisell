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
	"context"
	"fmt"
	"math"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/aios/toolregistry/tools"
)

// ---------- Required and optional context field names ----------

var (
	acosAnalysisRequiredFields   = []string{"campaign_id", "spend", "sales"}
	adOptimizationRequiredFields = []string{"campaign_id"}
)

// ---------- AdAdviceAgent ----------

// AdAdviceAgent implements A3 Ad Advice logic.
// It registers its tools on the ToolRegistry and dispatches decisions through it.
//
// Decision points:
//   - "acos_analysis" — analyzes ACoS metrics, detects anomalies, provides bid suggestions
//   - "ad_optimization" — suggests optimization actions (negative keywords, bid, budget)
type AdAdviceAgent struct {
	registry *toolregistry.ToolRegistry
}

// NewAdAdviceAgent creates a new AdAdviceAgent without a ToolRegistry.
// Use NewAdAdviceAgentWithRegistry to create one that dispatches through
// the ToolRegistry. This no-arg constructor is kept for backward compatibility
// with existing callers (agents.go, tests) — it falls back to direct execution.
func NewAdAdviceAgent() *AdAdviceAgent {
	return &AdAdviceAgent{}
}

// NewAdAdviceAgentWithRegistry creates a new AdAdviceAgent that registers its
// tools on the provided ToolRegistry and dispatches decisions through it.
func NewAdAdviceAgentWithRegistry(registry *toolregistry.ToolRegistry) *AdAdviceAgent {
	a := &AdAdviceAgent{registry: registry}
	a.registerTools()
	return a
}

// registerTools registers the A3 domain tools on the ToolRegistry with
// handlers that wrap the existing business logic.
func (a *AdAdviceAgent) registerTools() {
	specs := tools.AdAdviceTools()
	for i := range specs {
		spec := &specs[i]
		switch spec.Name {
		case "acos.analyze":
			spec.Handler = a.handleAcosAnalyze
		case "ad.optimize":
			spec.Handler = a.handleAdOptimize
		}
		a.registry.Register(spec)
	}
}

// handleAcosAnalyze is the ToolRegistry handler for "acos.analyze".
// It delegates to the existing analyzeAcos method and converts the
// multi-return output into a single interface{} response.
func (a *AdAdviceAgent) handleAcosAnalyze(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	output, _, _, _ := a.analyzeAcos(input)
	return output, nil
}

// handleAdOptimize is the ToolRegistry handler for "ad.optimize".
// It delegates to the existing suggestAdOptimization method.
func (a *AdAdviceAgent) handleAdOptimize(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	output, _, _, _ := a.suggestAdOptimization(input)
	return output, nil
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// When a ToolRegistry is available, the decision is routed through it by
// mapping decision points to tool names and calling through the registry.
// When no registry is configured, the agent falls back to direct method calls.
//
// Supported decision points:
//   - "acos_analysis" → tool "acos.analyze"
//   - "ad_optimization" → tool "ad.optimize"
func (a *AdAdviceAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "acos_analysis":
		return a.dispatchOrDirect(ctx, "acos.analyze", decisionPoint, params, a.analyzeAcos)
	case "ad_optimization":
		return a.dispatchOrDirect(ctx, "ad.optimize", decisionPoint, params, a.suggestAdOptimization)
	case "ad_strategy":
		return a.adStrategy(params)
	case "keyword_bidding":
		return a.keywordBidding(params)
	case "acos_optimization":
		return a.acosOptimization(params)
	case "budget_allocation":
		return a.budgetAllocation(params)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// dispatchOrDirect tries the ToolRegistry first; if unavailable, falls back to
// the provided direct handler.
func (a *AdAdviceAgent) dispatchOrDirect(
	callCtx context.Context,
	toolName string,
	_ string, // decisionPoint — reserved for future use
	params map[string]interface{},
	direct func(map[string]interface{}) (map[string]interface{}, float64, string, error),
) (map[string]interface{}, float64, string, error) {
	if a.registry != nil {
		result, callErr := a.registry.Call(callCtx, toolName, params)
		if callErr == nil {
			if m, ok := result.(map[string]interface{}); ok {
				// Extract confidence from the output map if present.
				conf := extractConfidence(m)
				return m, conf, "low", nil
			}
		}
		// Registry call failed — fall back to direct below.
		// Logging would go here if we had a logger on the agent.
	}
	return direct(params)
}

// extractConfidence looks for a "confidence" field in the output map.
// Returns 0.85 as default if not found.
func extractConfidence(m map[string]interface{}) float64 {
	if v, ok := m["confidence"]; ok {
		switch c := v.(type) {
		case float64:
			return c
		case int:
			return float64(c)
		}
	}
	return 0.85
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

// ---------- P3: ad_strategy (#194) ----------

// adStrategy recommends an ad campaign strategy based on product lifecycle and goals.
func (a *AdAdviceAgent) adStrategy(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	productStage := safeString(ctx["product_stage"], "launch") // launch, growth, mature
	budget := safeFloat(ctx["budget"], 0)
	goal := safeString(ctx["goal"], "sales") // sales, brand, acos_minimize
	marketplace := safeString(ctx["marketplace"], "US")

	strategies := make([]map[string]interface{}, 0)

	switch productStage {
	case "launch":
		strategies = append(strategies,
			map[string]interface{}{"type": "auto_targeting", "description": "自动定位广告，快速收集关键词数据，预算占比 60%", "budget_share": 60},
			map[string]interface{}{"type": "manual_broad", "description": "手动广泛匹配，测试核心关键词，预算占比 40%", "budget_share": 40},
		)
	case "growth":
		strategies = append(strategies,
			map[string]interface{}{"type": "manual_exact", "description": "精准匹配高转化关键词，预算占比 50%", "budget_share": 50},
			map[string]interface{}{"type": "auto_targeting", "description": "自动定位补充发现新词，预算占比 25%", "budget_share": 25},
			map[string]interface{}{"type": "product_targeting", "description": "商品定位广告，抢占竞品流量，预算占比 25%", "budget_share": 25},
		)
	case "mature":
		strategies = append(strategies,
			map[string]interface{}{"type": "manual_exact", "description": "精准匹配核心盈利关键词，预算占比 60%", "budget_share": 60},
			map[string]interface{}{"type": "retargeting", "description": "再营销广告，挽回流失客户，预算占比 25%", "budget_share": 25},
			map[string]interface{}{"type": "product_targeting", "description": "商品定位防御广告，预算占比 15%", "budget_share": 15},
		)
	default:
		strategies = append(strategies,
			map[string]interface{}{"type": "manual_exact", "description": "精准匹配广告", "budget_share": 100},
		)
	}

	var recommendedBudget float64
	if budget > 0 {
		recommendedBudget = budget
	} else {
		recommendedBudget = 200.0 // default daily budget
	}

	output = map[string]interface{}{
		"marketplace":       marketplace,
		"product_stage":     productStage,
		"goal":              goal,
		"recommended_budget": recommendedBudget,
		"strategies":        strategies,
		"confidence":        0.80,
	}
	return output, 0.80, "low", nil
}

// ---------- P3: keyword_bidding (#194) ----------

// keywordBidding provides CPC bid recommendations per keyword.
func (a *AdAdviceAgent) keywordBidding(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	keywords := ctx["keywords"]
	targetAcos := safeFloat(ctx["target_acos"], 30.0)
	budget := safeFloat(ctx["budget"], 0)

	if keywords == nil {
		return insufficientData("keyword_bidding", []string{"keywords"}), 0.0, "low", nil
	}

	bidSuggestions := make([]map[string]interface{}, 0)
	if termsList, ok := keywords.([]interface{}); ok {
		for _, raw := range termsList {
			term, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			kw := safeString(term["keyword"], "")
			spend := safeFloat(term["spend"], 0)
			sales := safeFloat(term["sales"], 0)
			currentBid := safeFloat(term["current_bid"], 0)

			acos := 0.0
			if sales > 0 {
				acos = round2(spend / sales * 100)
			}

			var suggestedBid float64
			var action string
			if acos > targetAcos && sales > 0 {
				idealAcos := targetAcos * 0.8
				if spend > 0 && currentBid > 0 {
					suggestedBid = round2(currentBid * idealAcos / acos)
				}
				action = "reduce_bid"
			} else if acos > 0 && acos <= targetAcos*0.5 {
				if currentBid > 0 {
					suggestedBid = round2(currentBid * 1.2)
				}
				action = "increase_bid"
			} else {
				suggestedBid = currentBid
				action = "maintain"
			}

			bidSuggestions = append(bidSuggestions, map[string]interface{}{
				"keyword":      kw,
				"current_bid":  currentBid,
				"suggested_bid": suggestedBid,
				"acos":         acos,
				"action":       action,
			})
		}
	}

	output = map[string]interface{}{
		"bid_suggestions": bidSuggestions,
		"target_acos":     targetAcos,
		"budget":          budget,
		"confidence":      0.85,
	}
	return output, 0.85, "low", nil
}

// ---------- P3: acos_optimization (#194) ----------

// acosOptimization provides ACOS optimization recommendations across campaigns.
func (a *AdAdviceAgent) acosOptimization(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	campaignsRaw := ctx["campaigns"]
	targetAcos := safeFloat(ctx["target_acos"], 30.0)
	maxBudget := safeFloat(ctx["max_budget"], 0)

	if campaignsRaw == nil {
		return insufficientData("acos_optimization", []string{"campaigns"}), 0.0, "low", nil
	}

	campaignResults := make([]map[string]interface{}, 0)
	if campaigns, ok := campaignsRaw.([]interface{}); ok {
		for _, raw := range campaigns {
			c, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			campaignID := safeString(c["campaign_id"], "")
			cSpend := safeFloat(c["spend"], 0)
			cSales := safeFloat(c["sales"], 0)

			cAcos := 0.0
			if cSales > 0 {
				cAcos = round2(cSpend / cSales * 100)
			}

			status := "healthy"
			var optimization []string
			if cAcos > targetAcos {
				status = "needs_optimization"
				reduction := round2((cAcos - targetAcos) / cAcos * 100)
				optimization = append(optimization, fmt.Sprintf("目标降低出价 %.0f%% 以将 ACOS 降至 %.1f%%", reduction, targetAcos))
			}
			if maxBudget > 0 && cSpend > maxBudget*0.9 {
				optimization = append(optimization, "预算即将耗尽，建议增加日预算")
			}

			campaignResults = append(campaignResults, map[string]interface{}{
				"campaign_id":  campaignID,
				"acos":         cAcos,
				"target_acos":  targetAcos,
				"status":       status,
				"optimization": optimization,
			})
		}
	}

	output = map[string]interface{}{
		"campaigns":      campaignResults,
		"target_acos":    targetAcos,
		"confidence":     0.85,
	}
	return output, 0.85, "low", nil
}

// ---------- P3: budget_allocation (#194) ----------

// budgetAllocation recommends budget distribution across campaigns.
func (a *AdAdviceAgent) budgetAllocation(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	totalBudget := safeFloat(ctx["total_budget"], 1000)
	campaignsRaw := ctx["campaigns"]
	goal := safeString(ctx["goal"], "sales")

	if campaignsRaw == nil {
		return insufficientData("budget_allocation", []string{"campaigns"}), 0.0, "low", nil
	}

	type campaignPerf struct {
		id    string
		spend float64
		sales float64
		acos  float64
	}
	var perfs []campaignPerf
	if campaigns, ok := campaignsRaw.([]interface{}); ok {
		for _, raw := range campaigns {
			c, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			cp := campaignPerf{
				id:    safeString(c["campaign_id"], ""),
				spend: safeFloat(c["spend"], 0),
				sales: safeFloat(c["sales"], 0),
			}
			if cp.sales > 0 {
				cp.acos = round2(cp.spend / cp.sales * 100)
			}
			perfs = append(perfs, cp)
		}
	}

	if len(perfs) == 0 {
		return insufficientData("budget_allocation", []string{"campaigns"}), 0.0, "low", nil
	}

	// Calculate total spend for proportion.
	var totalSpend float64
	for _, p := range perfs {
		totalSpend += p.spend
	}

	allocations := make([]map[string]interface{}, 0, len(perfs))
	if totalSpend > 0 {
		for _, p := range perfs {
			share := p.spend / totalSpend
			allocated := round2(totalBudget * share)
			var reason string
			if goal == "sales" && p.acos < 30 {
				reason = "高转化低ACOS，建议维持或增加预算"
			} else if p.acos > 50 {
				reason = "ACOS过高，建议减少预算或优化"
			} else {
				reason = "表现中等，建议维持当前预算"
			}
			allocations = append(allocations, map[string]interface{}{
				"campaign_id":      p.id,
				"current_share":    round2(share * 100),
				"allocated_budget": allocated,
				"acos":             p.acos,
				"reason":           reason,
			})
		}
	}

	output = map[string]interface{}{
		"total_budget":          totalBudget,
		"allocations":           allocations,
		"allocation_strategy":   goal,
		"confidence":            0.80,
	}
	return output, 0.80, "low", nil
}
