// Package impl provides concrete agent implementations.
//
// ListingOptimizerAgent implements A2 Listing Optimizer business logic ported
// from backend/app/agent/agents/listing_optimizer.py (Python FastAPI codebase).
//
// Design docs: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent2
//   - Keyword strategy + competitive deconstruction + copy generation
//   - Input: product info and competitor data
//   - Output: optimized listing (title, bullets, search terms, keyword research)
package impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// ---------- ListingOptimizerAgent ----------

// ListingOptimizerAgent implements A2 Listing Optimizer logic.
//
// Decision points (P3 upgrades #195):
//   - "listing_optimize" — generates optimized title, bullets, search terms
//   - "keyword_research" — expands seed keywords into broader keyword candidates
//   - "title_optimization" — optimizes product title for a marketplace
//   - "keyword_optimization" — optimizes keyword selection by volume/competition
//   - "search_trend_analysis" — analyzes search trends for keywords
//   - "multi_platform_seo" — generates platform-specific SEO recommendations
type ListingOptimizerAgent struct{}

// NewListingOptimizerAgent creates a new ListingOptimizerAgent.
func NewListingOptimizerAgent() *ListingOptimizerAgent {
	return &ListingOptimizerAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
func (a *ListingOptimizerAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "listing_optimize":
		result, callErr := toolregistry.DefaultRegistry.Call(ctx, "listing.optimize", params)
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
		return output, 0.0, "low", nil
	case "keyword_research":
		return a.researchKeywords(params)
	case "title_optimization":
		return a.optimizeTitle(params)
	case "keyword_optimization":
		return a.optimizeKeywords(params)
	case "search_trend_analysis":
		return a.analyzeSearchTrends(params)
	case "multi_platform_seo":
		return a.multiPlatformSEO(params)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- (existing) keyword_research ----------

func (a *ListingOptimizerAgent) researchKeywords(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	seed := parseStringList(ctx["seed_keywords"])
	if len(seed) == 0 {
		return insufficientData("keyword_research", []string{"seed_keywords"}), 0.0, "low", nil
	}

	suffixes := []string{"for", "with", "best"}
	expanded := make([]string, 0, len(seed)*len(suffixes))
	for _, s := range seed {
		for _, t := range suffixes {
			expanded = append(expanded, s+" "+t)
		}
	}

	output = map[string]interface{}{
		"seed":        seed,
		"expanded":    expanded,
		"total_found": len(seed) * 3,
	}
	return output, 0.80, "low", nil
}

// ---------- P3: title_optimization (#195) ----------

// optimizeTitle optimizes a product title for a given marketplace.
// Considers character limits, keyword placement, and platform best practices.
func (a *ListingOptimizerAgent) optimizeTitle(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	title := safeString(ctx["title"], "")
	marketplace := safeString(ctx["marketplace"], "US")
	category := safeString(ctx["category"], "")
	keywords := parseStringList(ctx["keywords"])

	if title == "" {
		return insufficientData("title_optimization", []string{"title"}), 0.0, "low", nil
	}

	// Platform-specific title length limits (approximate).
	charLimit := 200 // default
	platformMax := map[string]int{
		"US": 200, "Amazon": 200, "DE": 150, "FR": 150,
		"UK": 200, "Ozon": 200, "Shopee": 120, "Lazada": 120,
	}
	if limit, ok := platformMax[marketplace]; ok {
		charLimit = limit
	}

	// Build optimization suggestions.
	suggestions := make([]string, 0)
	if len(title) > charLimit {
		suggestions = append(suggestions, fmt.Sprintf("标题超出 %d 字符限制（当前 %d 字符），建议精简", charLimit, len(title)))
	} else if len(title) < 80 {
		suggestions = append(suggestions, fmt.Sprintf("标题过短（当前 %d 字符），建议扩充至 80-120 字符以包含更多关键词", len(title)))
	}

	// Suggest front-loading important keywords.
	suggestions = append(suggestions, "建议将核心关键词放在标题前 80 个字符内")

	if len(keywords) > 0 {
		missing := make([]string, 0)
		for _, kw := range keywords {
			if !strings.Contains(strings.ToLower(title), strings.ToLower(kw)) {
				missing = append(missing, kw)
			}
		}
		if len(missing) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("以下关键词未包含在标题中: %s", strings.Join(missing, ", ")))
		}
	}

	if category != "" {
		suggestions = append(suggestions, fmt.Sprintf("建议在标题中包含品类关键词 '%s' 以提高搜索匹配度", category))
	}

	output = map[string]interface{}{
		"title":           title,
		"marketplace":     marketplace,
		"char_count":      len(title),
		"char_limit":      charLimit,
		"optimized_title": title, // ponytail: return original; LLM expansion when wired
		"suggestions":     suggestions,
		"total_suggestions": len(suggestions),
	}
	return output, 0.75, "low", nil
}

// ---------- P3: keyword_optimization (#195) ----------

// optimizeKeywords analyzes keywords for search volume and competition level.
func (a *ListingOptimizerAgent) optimizeKeywords(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	keywords := parseStringList(ctx["keywords"])
	marketplace := safeString(ctx["marketplace"], "US")
	category := safeString(ctx["category"], "")

	if len(keywords) == 0 {
		return insufficientData("keyword_optimization", []string{"keywords"}), 0.0, "low", nil
	}

	// ponytail: keyword analysis is stub-level; upgrade with real search volume
	// API (e.g., Helium 10, Jungle Scout) when connected.
	analyzed := make([]map[string]interface{}, 0, len(keywords))
	for _, kw := range keywords {
		analyzed = append(analyzed, map[string]interface{}{
			"keyword":             kw,
			"estimated_volume":    "medium",     // stub
			"competition_level":   "medium",     // stub
			"recommended":         len(kw) >= 2, // filter too-short keywords
			"platform_relevance":  marketplace,
		})
	}

	highPriority := make([]string, 0)
	lowPriority := make([]string, 0)
	for _, a := range analyzed {
		if a["recommended"].(bool) {
			highPriority = append(highPriority, a["keyword"].(string))
		} else {
			lowPriority = append(lowPriority, a["keyword"].(string))
		}
	}

	output = map[string]interface{}{
		"marketplace":     marketplace,
		"category":        category,
		"analyzed":        analyzed,
		"high_priority":   highPriority,
		"low_priority":    lowPriority,
		"total_keywords":  len(keywords),
	}
	return output, 0.70, "low", nil
}

// ---------- P3: search_trend_analysis (#195) ----------

// analyzeSearchTrends analyzes search trends for a keyword or category.
func (a *ListingOptimizerAgent) analyzeSearchTrends(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	keyword := safeString(ctx["keyword"], "")
	category := safeString(ctx["category"], "")
	marketplace := safeString(ctx["marketplace"], "US")

	if keyword == "" && category == "" {
		return insufficientData("search_trend_analysis", []string{"keyword", "category"}), 0.0, "low", nil
	}

	// ponytail: stub trend data; replace with real Google Trends / platform
	// search term report API integration when available.
	output = map[string]interface{}{
		"keyword":           keyword,
		"category":          category,
		"marketplace":       marketplace,
		"trend_direction":   "stable", // "rising", "stable", "declining"
		"search_volume":     "medium",
		"seasonal_peak":     "Q4",
		"related_searches":  []string{},
		"recommendation":    "基于现有数据，该关键词搜索趋势稳定，建议持续优化 Listing",
	}
	return output, 0.60, "low", nil
}

// ---------- P3: multi_platform_seo (#195) ----------

// multiPlatformSEO generates SEO recommendations for multiple platforms.
func (a *ListingOptimizerAgent) multiPlatformSEO(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	title := safeString(ctx["title"], "")
	description := safeString(ctx["description"], "")
	keywords := parseStringList(ctx["keywords"])
	targetPlatforms := parseStringList(ctx["target_platforms"])
	if len(targetPlatforms) == 0 {
		targetPlatforms = []string{"Amazon", "Ozon", "Shopee", "Lazada"}
	}

	if title == "" {
		return insufficientData("multi_platform_seo", []string{"title"}), 0.0, "low", nil
	}

	platformTips := map[string]string{
		"Amazon": "使用品牌+核心词+属性+卖点格式，充分利用 200 字符，包含后端搜索词",
		"Ozon":   "支持多语言标题，建议俄语翻译，关键词前置，Ozon 搜索权重对标题前 50 字符敏感",
		"Shopee": "标题控制在 120 字符以内，使用简洁关键词，突出价格和促销信息",
		"Lazada": "Lazada 东南亚市场建议使用英文标题+本地语言关键词，包含品类关键词",
		"eBay":   "使用描述性标题，包含 UPC/EAN 编码以提高搜索匹配度",
		"Walmart": "标题控制在 100 字符以内，品牌前置，避免促销词汇",
	}

	platformResults := make([]map[string]interface{}, 0, len(targetPlatforms))
	for _, p := range targetPlatforms {
		tip := platformTips[p]
		if tip == "" {
			tip = "标准 SEO 建议：标题包含核心关键词和卖点，描述包含长尾词"
		}
		platformResults = append(platformResults, map[string]interface{}{
			"platform":  p,
			"tips":      tip,
			"keywords":  keywords,
		})
	}

	output = map[string]interface{}{
		"title":             title,
		"description":       description,
		"target_platforms":  targetPlatforms,
		"platform_seo":      platformResults,
	}
	return output, 0.70, "low", nil
}

// ---------- Helper ----------

func parseStringList(v interface{}) []string {
	if v == nil {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if ok {
			out = append(out, s)
		}
	}
	return out
}
