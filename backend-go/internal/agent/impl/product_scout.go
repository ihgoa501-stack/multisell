// Package impl provides concrete agent implementations.
//
// ProductScoutAgent implements A1 Product Scout business logic ported from
// backend/app/agent/agents/product_scout.py (Python FastAPI codebase).
//
// Design docs: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent1
//   - Multi-dimension product scoring (demand / competition / margin / trend)
//   - Input: category, marketplace, candidate product list
//   - Output: scored and ranked candidate list
package impl

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// ---------- Context field names ----------

var productScoutRequiredFields = []string{"category", "marketplace"}

// ---------- candidateItem represents a product candidate with scoring data ----------

type candidateItem struct {
	Name         string
	SearchVolume float64
	TrendGrowth  float64
	ReviewCount  float64
	Price        float64
	Cost         float64
}

// ---------- ProductScoutAgent ----------

// ProductScoutAgent implements A1 Product Scout logic.
//
// Decision points:
//   - "product_scout" — scores and ranks product candidates by multi-dimension formula
//   - "market_analysis" — returns a market analysis summary
//
// When a ToolRegistry is configured via SetToolRegistry, the agent delegates
// through it, gaining hook-based middleware, circuit-breaker protection, and
// LLM function-calling discoverability.
type ProductScoutAgent struct {
	registry *toolregistry.ToolRegistry
}

// NewProductScoutAgent creates a new ProductScoutAgent.
// Call SetToolRegistry to enable ToolRegistry-based dispatch.
func NewProductScoutAgent() *ProductScoutAgent {
	return &ProductScoutAgent{}
}

// SetToolRegistry configures the ToolRegistry for this agent and registers its
// decision points as discoverable tools. After this call, Decide() will delegate
// through the ToolRegistry.
func (a *ProductScoutAgent) SetToolRegistry(registry *toolregistry.ToolRegistry) {
	if registry == nil {
		return
	}
	a.registry = registry
	a.registerTools()
}

// registerTools registers the agent's decision points as tools in the ToolRegistry.
func (a *ProductScoutAgent) registerTools() {
	if a.registry == nil {
		return
	}
	a.registry.Register(&toolregistry.Tool{
		Name:        "product_scout",
		Version:     "1.0.0",
		Description: "选品打分——多维度（需求/竞争/利润/趋势）对候选商品打分排序，返回 Top-20 结果",
		Squad:       "growth",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "选品打分参数",
			Properties: map[string]*toolregistry.Schema{
				"category":    {Type: "string", Description: "商品类目"},
				"marketplace": {Type: "string", Description: "目标市场（如 US/JP/EU）"},
				"candidates": {
					Type:        "array",
					Description: "候选商品列表",
					Items: &toolregistry.Schema{
						Type: "object",
						Properties: map[string]*toolregistry.Schema{
							"name":          {Type: "string", Description: "商品名称"},
							"price":         {Type: "number", Description: "售价"},
							"cost":          {Type: "number", Description: "成本"},
							"search_volume": {Type: "number", Description: "搜索量"},
							"trend_growth":  {Type: "number", Description: "趋势增长率(%)"},
							"review_count":  {Type: "number", Description: "竞品评论数"},
						},
					},
				},
			},
			Required: []string{"category", "marketplace", "candidates"},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "选品排序结果，包含 Top-20 商品及其多维度评分和风险标记",
		},
		RequiredPermissions: []string{"growth:read:product_scout"},
		RiskLevel:           toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.scout(input)
			return map[string]interface{}{
				"output":     output,
				"confidence": confidence,
				"risk_level": riskLevel,
			}, nil
		},
	})

	a.registry.Register(&toolregistry.Tool{
		Name:        "market_analysis",
		Version:     "1.0.0",
		Description: "市场分析——快速评估品类市场概况（市场规模、趋势方向、置信度）",
		Squad:       "growth",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "市场分析参数",
			Properties: map[string]*toolregistry.Schema{
				"category":    {Type: "string", Description: "商品类目"},
				"marketplace": {Type: "string", Description: "目标市场（如 US/JP/EU，默认 US）"},
				"trend":       {Type: "string", Description: "趋势方向（可选，如 stable/rising/declining）"},
			},
			Required: []string{"category"},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "市场分析概要，包含类目、市场规模评估、趋势方向和置信度",
		},
		RequiredPermissions: []string{"growth:read:market_analysis"},
		RiskLevel:           toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.analyzeMarket(input)
			return map[string]interface{}{
				"output":     output,
				"confidence": confidence,
				"risk_level": riskLevel,
			}, nil
		},
	})

	a.registry.Register(&toolregistry.Tool{
		Name:        "product_research",
		Version:     "1.0.0",
		Description: "选品调研——给定类目和目标市场，生成调研假设方向、关键词建议、风险提示和待采集数据清单",
		Squad:       "growth",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "选品调研参数",
			Properties: map[string]*toolregistry.Schema{
				"category":        {Type: "string", Description: "商品类目"},
				"target_market":   {Type: "string", Description: "目标市场（如 US/RU/JP/EU）"},
				"target_platform": {Type: "string", Description: "目标平台（如 Ozon/Amazon/Shopee）"},
				"constraints":     {Type: "string", Description: "约束条件（可选，如低重量/小件）"},
			},
			Required: []string{"category", "target_market", "target_platform"},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "选品调研结果，包含推荐方向、理由、风险、关键词、待采集数据和警告",
		},
		RequiredPermissions: []string{"growth:read:product_research"},
		RiskLevel:           toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.productResearch(input)
			return map[string]interface{}{
				"output":     output,
				"confidence": confidence,
				"risk_level": riskLevel,
			}, nil
		},
	})

	a.registry.Register(&toolregistry.Tool{
		Name:        "supplier_discovery",
		Version:     "1.0.0",
		Description: "供应商发现——根据调研方向或关键词生成 1688 搜索页面、采集指令和供应商筛选规则",
		Squad:       "growth",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "供应商发现参数",
			Properties: map[string]*toolregistry.Schema{
				"keywords":   {Type: "array", Items: &toolregistry.Schema{Type: "string"}, Description: "搜索关键词列表"},
				"directions": {Type: "array", Items: &toolregistry.Schema{Type: "object"}, Description: "来自 product_research 的方向列表（含 name 字段）"},
				"category":   {Type: "string", Description: "类目名称（当无关键词时的回退）"},
			},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "供应商发现结果，包含 1688 搜索页面 URL、采集指令、筛选规则",
		},
		RequiredPermissions: []string{"growth:read:supplier_discovery"},
		RiskLevel:           toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.supplierDiscovery(input)
			return map[string]interface{}{
				"output":     output,
				"confidence": confidence,
				"risk_level": riskLevel,
			}, nil
		},
	})
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// When a ToolRegistry is configured, the agent delegates via registry.Call(),
// which applies hooks, circuit breakers, and schema validation.
// Without a ToolRegistry, the agent falls back to direct internal dispatch.
//
// Supported decision points:
//   - "product_scout"
//   - "market_analysis"
//   - "product_research" — generate research hypotheses for a category/market
//   - "supplier_discovery" — generate 1688 collection plans from keywords/directions
func (a *ProductScoutAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if a.registry != nil {
		return a.decideViaRegistry(ctx, decisionPoint, params)
	}
	return a.decideDirect(decisionPoint, params)
}

// decideViaRegistry delegates the decision through the ToolRegistry's hook chain.
func (a *ProductScoutAgent) decideViaRegistry(callCtx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	result, callErr := a.registry.Call(callCtx, decisionPoint, params)
	if callErr != nil {
		// Tool not found or hook rejected; fall through to the direct dispatch table.
		return a.decideDirect(decisionPoint, params)
	}

	// Unwrap the tool output envelope.
	env, ok := result.(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          "unexpected tool output format",
		}, 0.0, "low", nil
	}

	if out, ok := env["output"].(map[string]interface{}); ok {
		output = out
	}
	if conf, ok := env["confidence"].(float64); ok {
		confidence = conf
	}
	if risk, ok := env["risk_level"].(string); ok {
		riskLevel = risk
	}
	return output, confidence, riskLevel, nil
}

// decideDirect is the legacy direct-dispatch path (no ToolRegistry configured).
func (a *ProductScoutAgent) decideDirect(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "product_scout":
		return a.scout(ctx)
	case "market_analysis":
		return a.analyzeMarket(ctx)
	case "product_research":
		return a.productResearch(ctx)
	case "supplier_discovery":
		return a.supplierDiscovery(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: product_scout ----------

// scout scores and ranks product candidates using the multi-dimension formula:
//
//	score = demand*30 + growth*25 + competition*20 + margin*25
//
// where:
//   - demand = search_volume * 0.01
//   - growth = trend_growth / 100
//   - competition = max(0, 1 - review_count / 1000)
//   - margin = (price - cost) / price (when price > 0)
//
// Required context fields: category, marketplace
// Required candidates list: each item must have name, price, cost, search_volume,
// trend_growth, and review_count.
//
// Returns top-20 scored candidates sorted by score descending.
func (a *ProductScoutAgent) scout(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, productScoutRequiredFields); len(missing) > 0 {
		return insufficientData("product_scout", missing), 0.0, "low", nil
	}

	category := safeString(ctx["category"], "")
	marketplace := safeString(ctx["marketplace"], "US")

	candidatesInput := ctx["candidates"]
	if candidatesInput == nil {
		return insufficientData("product_scout", []string{"candidates"}), 0.0, "low", nil
	}

	rawList, ok := candidatesInput.([]interface{})
	if !ok || len(rawList) == 0 {
		return insufficientData("product_scout", []string{"candidates"}), 0.0, "low", nil
	}

	type scoredCandidate struct {
		Name             string   `json:"name"`
		Score            float64  `json:"score"`
		DemandScore      float64  `json:"demand_score"`
		CompetitionScore float64  `json:"competition_score"`
		MarginScore      float64  `json:"margin_score"`
		TrendScore       float64  `json:"trend_score"`
		EstimatedMargin  float64  `json:"estimated_margin"`
		RiskFlags        []string `json:"risk_flags"`
	}

	scored := make([]scoredCandidate, 0, len(rawList))

	for _, raw := range rawList {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		demand := safeFloat(item["search_volume"], 0) * 0.01
		growth := safeFloat(item["trend_growth"], 0) / 100.0
		competition := max(0, 1-safeFloat(item["review_count"], 0)/1000.0)
		price := safeFloat(item["price"], 0)
		cost := safeFloat(item["cost"], 0)

		var margin float64
		if price > 0 {
			margin = (price - cost) / price
		}

		score := round1(demand*30 + growth*25 + competition*20 + margin*25)

		riskFlags := make([]string, 0)
		if competition < 0.3 {
			riskFlags = append(riskFlags, "高竞争")
		}

		scored = append(scored, scoredCandidate{
			Name:             safeString(item["name"], ""),
			Score:            score,
			DemandScore:      round1(demand * 100),
			CompetitionScore: round1(competition * 100),
			MarginScore:      round1(margin * 100),
			TrendScore:       round1(growth * 100),
			EstimatedMargin:  round1(margin * 100),
			RiskFlags:        riskFlags,
		})
	}

	// Sort by score descending (mirrors Python: scored.sort(key=lambda x: x["score"], reverse=True)).
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Return top 20.
	topN := 20
	if len(scored) < topN {
		topN = len(scored)
	}

	output = map[string]interface{}{
		"category":      category,
		"marketplace":   marketplace,
		"candidates":    scored[:topN],
		"total_scanned": len(scored),
		"confidence":    0.85,
	}
	return output, 0.85, "low", nil
}

// ---------- Decision point: market_analysis ----------

// analyzeMarket returns a high-level market analysis summary.
func (a *ProductScoutAgent) analyzeMarket(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	output = map[string]interface{}{
		"category":             safeString(ctx["category"], ""),
		"marketplace":          safeString(ctx["marketplace"], "US"),
		"market_size_estimate": "medium",
		"trend_direction":      safeString(ctx["trend"], "stable"),
		"confidence":           0.80,
	}
	return output, 0.80, "low", nil
}

// ---------- Decision point: product_research ----------

// productRequiredFields are the fields needed for productResearch.
var productResearchRequiredFields = []string{"category", "target_market", "target_platform"}

// productResearch generates structured research directions for a given category
// and target market. It does not access real market data — it produces research
// hypotheses with explicit confidence levels and uncertainty notes.
//
// Input: category, target_market, target_platform, constraints (optional)
// Output: recommended directions with reasoning, risks, keywords, data gaps, confidence
//
// Ponytail: rule-based category→direction mapping for initial coverage.
// Replace with LLM-driven research when real market data feeds are available.
func (a *ProductScoutAgent) productResearch(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, productResearchRequiredFields); len(missing) > 0 {
		return insufficientData("product_research", missing), 0.0, "low", nil
	}

	category := safeString(ctx["category"], "")
	targetMarket := safeString(ctx["target_market"], "US")
	targetPlatform := safeString(ctx["target_platform"], "")
	constraints := safeString(ctx["constraints"], "")

	// Build directions based on category heuristics.
	// ponytail: static direction templates, expand with LLM/research when real
	// category→direction mappings from market analysis are available.
	directions := defaultDirections(category, targetMarket, targetPlatform)

	// Collect data needs across all directions.
	dataNeeded := uniqueDataNeeds(directions)

	output = map[string]interface{}{
		"status":                 "research_ready",
		"category":               category,
		"target_market":          targetMarket,
		"target_platform":        targetPlatform,
		"constraints_used":       constraints,
		"recommended_directions": directions,
		"data_needed":            dataNeeded,
		"warnings": []string{
			"这是调研假设，不是确定经营结论。所有方向需要采集真实数据进行验证。",
			"This is a research hypothesis, not a business conclusion. All directions require real data collection for validation.",
		},
	}
	return output, 0.65, "low", nil
}

// ---------- Decision point: supplier_discovery ----------

var supplierDiscoveryCommonKeywords = map[string][]string{
	"厨房收纳":  {"厨房收纳", "免打孔置物架", "厨房整理架", "锅盖架", "调料架"},
	"家居收纳":  {"收纳盒", "储物箱", "桌面收纳", "化妆品收纳", "抽屉分隔"},
	"厨房小工具": {"厨房计时器", "削皮器", "切菜器", "捣蒜器", "开瓶器"},
	"浴室用品":  {"浴室置物架", "牙刷架", "毛巾架", "浴巾", "防滑垫"},
	"办公用品":  {"桌面文具收纳", "笔记本", "便签纸", "笔筒", "文件架"},
	"手机配件":  {"手机壳", "手机支架", "充电线", "车载手机架", "屏幕保护膜"},
	"家居装饰":  {"墙贴", "桌布", "花瓶", "仿真花", "装饰画"},
	"小家电":   {"便携榨汁杯", "迷你加湿器", "USB风扇", "手持挂烫机", "电动牙刷"},
	"宠物用品":  {"猫抓板", "狗玩具", "宠物梳子", "猫碗架", "宠物出行包"},
	"运动户外":  {"瑜伽垫", "阻力带", "运动水壶", "握力器", "跳绳"},
}

// supplierDiscovery generates actionable collection plans for finding products
// on 1688 based on research directions. Output is meant for human execution
// through the browser extension (#186/#191 flow).
//
// Input: directions (from product_research), or keywords directly
// Output: 1688 search URLs, supplier filter rules, collection instructions
func (a *ProductScoutAgent) supplierDiscovery(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	keywords := resolveKeywords(ctx)

	if len(keywords) == 0 {
		return map[string]interface{}{
			"status":   "needs_keywords",
			"message":  "需要关键词或选品调研方向来生成搜索页面。请先运行 product_research 或提供 keywords 参数。",
			"warnings": []string{"这是调研建议，不是确定经营结论"},
		}, 0.0, "low", nil
	}

	pages := make([]map[string]interface{}, 0, len(keywords))
	for _, kw := range keywords {
		encoded := url.QueryEscape(kw)
		pages = append(pages, map[string]interface{}{
			"type":   "search",
			"url":    "https://s.1688.com/selloffer/offer_search.htm?keywords=" + encoded,
			"reason": "搜索「" + kw + "」候选商品",
		})
	}

	output = map[string]interface{}{
		"status":          "collection_plan_ready",
		"source_platform": "1688",
		"search_keywords": keywords,
		"suggested_pages": pages,
		"supplier_filter_rules": []string{
			"优先看有成交记录的店铺（显示成交笔数）",
			"优先看支持小起订量（如 2-10 件起批）",
			"优先看主图和规格完整的商品",
			"优先看店铺评分 4.5 以上的卖家",
			"优先看有实力商家/诚信通标识的店铺",
			"注意商品是否支持跨境/一件代发",
		},
		"collection_instructions": []string{
			"由用户手动打开上述 1688 搜索页面",
			"使用 Chrome 扩展「采集当前页商品列表」采集合集结果（#191 流程）",
			"列表页结果进入 CollectLead 待处理",
			"对感兴趣的商品点开详情页，用扩展采集商品详情（#186 流程）",
			"详情页结果进入 CandidateProduct 进行完整度评估",
		},
		"warnings": []string{
			"这是调研假设，不是确定经营结论。所有页面需要人工确认后再采集。",
			"1688 链接可能随时间失效，需要在采集前确认页面可访问。",
		},
	}
	return output, 0.6, "low", nil
}

// ---------- Helpers ----------

// defaultDirections returns research directions for a given category.
// ponytail: static map, load from config or DB when direction catalog grows.
func defaultDirections(category, market, platform string) []map[string]interface{} {
	switch category {
	case "家居", "home", "家居用品":
		return []map[string]interface{}{
			{
				"name":              "厨房收纳小件",
				"why":               "厨房收纳是跨境电商长尾刚需，SKU丰富，重量轻适合小包直邮，体积小运费低，客单价适中（$5-20）",
				"target_price_band": "$5-$20 (零售)",
				"risk_notes":        []string{"部分品类竞争激烈（如调料架）", "需确认目标市场的厨房标准尺寸（美标/欧标）", "部分商品可能有专利风险"},
				"keywords":          []string{"kitchen storage", "kitchen organizer", "spice rack", "cabinet organizer", "over sink dish rack"},
				"data_needed":       []string{"1688 采购价", "重量", "尺寸", "平台竞品价格和评价数", "品类搜索趋势"},
				"confidence":        0.65,
			},
			{
				"name":              "浴室收纳与配件",
				"why":               "浴室收纳与厨房收纳同源供应链，重量轻体积小，跨境物流友好。客单价和利润空间与厨房收纳类似",
				"target_price_band": "$5-$25 (零售)",
				"risk_notes":        []string{"需确认材质是否防锈防水", "部分商品的包装尺寸可能导致物流成本偏高", "品牌商品专利风险"},
				"keywords":          []string{"bathroom storage", "shower caddy", "toothbrush holder", "soap dispenser", "bath mat"},
				"data_needed":       []string{"1688 采购价", "重量", "尺寸", "平台竞品价格", "材质选择"},
				"confidence":        0.55,
			},
			{
				"name":              "桌面收纳与办公小件",
				"why":               "收纳类延伸品类，供应链成熟。适合 With 办公/学生场景，与家居收纳联合采购降低单位物流成本",
				"target_price_band": "$3-$15 (零售)",
				"risk_notes":        []string{"低价位商品利润空间有限", "需靠多件组合提升客单价", "部分商品需要 FBA 物流方案"},
				"keywords":          []string{"desk organizer", "pencil case", "document holder", "cable organizer", "phone stand"},
				"data_needed":       []string{"1688 采购价", "重量", "组合装可能性"},
				"confidence":        0.5,
			},
		}
	case "宠物用品", "pet", "pets":
		return []map[string]interface{}{
			{
				"name":              "猫用品小件",
				"why":               "宠物用品在 Ozon 等新兴平台增长快，中国供应链优势明显。猫咪用品客单价适中，重复购买率高",
				"target_price_band": "$5-$30 (零售)",
				"risk_notes":        []string{"需确认材质安全认证", "部分商品有运输尺寸限制", "需关注目的国宠物用品进口规定"},
				"keywords":          []string{"cat toys", "cat bed", "cat scratching post", "cat bowl", "pet grooming"},
				"data_needed":       []string{"1688 采购价", "重量", "材质认证要求", "平台相关品类增长数据"},
				"confidence":        0.55,
			},
			{
				"name":              "狗出行配件",
				"why":               "狗出行配件（牵引绳、胸背带、便携水碗等）是高频需求，重量轻，适合跨境小包",
				"target_price_band": "$8-$25 (零售)",
				"risk_notes":        []string{"尺寸规格多，管理复杂度高", "需确认安全标准", "品牌配件需避免侵权"},
				"keywords":          []string{"dog leash", "dog harness", "dog collar", "dog travel bowl", "pet carrier"},
				"data_needed":       []string{"1688 采购价", "重量", "规格 SKU 管理"},
				"confidence":        0.5,
			},
		}
	default:
		return []map[string]interface{}{
			{
				"name":              category + " 子类目调研方向",
				"why":               "通用类目需要更多市场数据来确定具体方向。建议先采集该品类在目标平台的搜索结果，了解市场供给情况",
				"target_price_band": "待采集后确定",
				"risk_notes":        []string{"该品类暂无预设方向", "建议先运行品类关键词搜索采集"},
				"keywords":          []string{category},
				"data_needed":       []string{"平台搜索结果", "竞品价格区间", "商品数量统计"},
				"confidence":        0.3,
			},
		}
	}
}

// uniqueDataNeeds collects unique data-needed strings across directions.
func uniqueDataNeeds(directions []map[string]interface{}) []string {
	seen := make(map[string]bool)
	var out []string
	for _, d := range directions {
		if raw, ok := d["data_needed"].([]string); ok {
			for _, item := range raw {
				if !seen[item] {
					seen[item] = true
					out = append(out, item)
				}
			}
		}
	}
	return out
}

// resolveKeywords extracts search keywords from context.
// Prefers explicit "keywords" param, then falls back to direction names,
// finally maps to common keyword list.
func resolveKeywords(ctx map[string]interface{}) []string {
	// Explicit keywords override.
	if kw, ok := ctx["keywords"].([]string); ok && len(kw) > 0 {
		return kw
	}
	if raw, ok := ctx["keywords"].([]interface{}); ok && len(raw) > 0 {
		kw := make([]string, 0, len(raw))
		for _, r := range raw {
			if s, ok := r.(string); ok {
				kw = append(kw, s)
			}
		}
		if len(kw) > 0 {
			return kw
		}
	}

	// Try to extract direction names from research output.
	if dirs, ok := ctx["directions"].([]map[string]interface{}); ok {
		kw := make([]string, 0, len(dirs))
		for _, d := range dirs {
			if n, ok := d["name"].(string); ok {
				kw = append(kw, n)
			}
		}
		if len(kw) > 0 {
			return kw
		}
	}
	if raw, ok := ctx["directions"].([]interface{}); ok {
		kw := make([]string, 0, len(raw))
		for _, r := range raw {
			if m, ok := r.(map[string]interface{}); ok {
				if n, ok := m["name"].(string); ok {
					kw = append(kw, n)
				}
			}
		}
		if len(kw) > 0 {
			return kw
		}
	}

	// Try to extract from category.
	category := safeString(ctx["category"], "")
	if category != "" {
		if kw, ok := supplierDiscoveryCommonKeywords[category]; ok {
			return kw
		}
		// Use category as keyword itself.
		return []string{category}
	}

	// As a fallback, try "message" field.
	if msg := safeString(ctx["message"], ""); msg != "" {
		return []string{msg}
	}

	return nil
}
