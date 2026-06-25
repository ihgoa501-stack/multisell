package actions

// Action represents an agent action type for the legacy-compatible action API.
// The new unified action center (internal/ai.UnifiedAction) is the system of
// record; this package preserves the old /agents/:id/actions surface by
// mapping onto the same lifecycle.
type Action struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	RequiresApproval bool   `json:"requires_approval"`
	RiskLevel        string `json:"risk_level"`
}

// Registry holds all available agent action types. Populated by init() with
// the canonical LingMirror action vocabulary.
var Registry = map[string]Action{}

// Register adds an action to the registry.
func Register(name string, action Action) {
	Registry[name] = action
}

func init() {
	// Listing / optimization actions (advisory, low risk).
	Register("listing_optimize", Action{Name: "listing_optimize", Description: "重写 listing 标题/描述/关键词", RequiresApproval: false, RiskLevel: "low"})
	Register("keyword_research", Action{Name: "keyword_research", Description: "生成关键词建议", RequiresApproval: false, RiskLevel: "low"})
	Register("acos_analysis", Action{Name: "acos_analysis", Description: "广告 ACOS 分析", RequiresApproval: false, RiskLevel: "low"})
	Register("product_scout", Action{Name: "product_scout", Description: "选品建议", RequiresApproval: false, RiskLevel: "low"})
	Register("market_analysis", Action{Name: "market_analysis", Description: "市场趋势分析", RequiresApproval: false, RiskLevel: "low"})
	Register("auto_reply", Action{Name: "auto_reply", Description: "客服自动回复", RequiresApproval: false, RiskLevel: "low"})
	Register("intent_classify", Action{Name: "intent_classify", Description: "客服意图分类", RequiresApproval: false, RiskLevel: "low"})
	Register("dashboard_overview", Action{Name: "dashboard_overview", Description: "驾驶舱聚合", RequiresApproval: false, RiskLevel: "low"})

	// Operational actions (guided, medium risk, need approval).
	Register("replenishment_plan", Action{Name: "replenishment_plan", Description: "生成补货计划", RequiresApproval: true, RiskLevel: "medium"})
	Register("logistics_choice", Action{Name: "logistics_choice", Description: "切换物流渠道", RequiresApproval: true, RiskLevel: "medium"})
	Register("cost_optimization", Action{Name: "cost_optimization", Description: "成本优化建议", RequiresApproval: true, RiskLevel: "medium"})
	Register("ad_optimization", Action{Name: "ad_optimization", Description: "广告投放优化", RequiresApproval: true, RiskLevel: "medium"})

	// High-risk actions (supervised, need approval).
	Register("stock_alert", Action{Name: "stock_alert", Description: "库存预警 + 自动补货", RequiresApproval: true, RiskLevel: "medium"})
	Register("profit_check", Action{Name: "profit_check", Description: "利润率检查 + 调价建议", RequiresApproval: true, RiskLevel: "medium"})
	Register("profit_watch", Action{Name: "profit_watch", Description: "亏损 SKU 止损", RequiresApproval: true, RiskLevel: "high"})
	Register("compliance_check", Action{Name: "compliance_check", Description: "合规检查", RequiresApproval: true, RiskLevel: "high"})
	Register("certification_lookup", Action{Name: "certification_lookup", Description: "认证查询", RequiresApproval: true, RiskLevel: "high"})
	Register("discount_check", Action{Name: "discount_check", Description: "折扣风控", RequiresApproval: true, RiskLevel: "high"})
	Register("promotion_validation", Action{Name: "promotion_validation", Description: "促销校验", RequiresApproval: true, RiskLevel: "high"})
	Register("discount_risk_check", Action{Name: "discount_risk_check", Description: "折扣风险检查", RequiresApproval: true, RiskLevel: "high"})
	Register("warehouse_routing", Action{Name: "warehouse_routing", Description: "仓库路由", RequiresApproval: true, RiskLevel: "high"})
	Register("customs_declare", Action{Name: "customs_declare", Description: "报关单校验", RequiresApproval: true, RiskLevel: "high"})
}

// List returns all registered actions as a slice.
func List() []Action {
	out := make([]Action, 0, len(Registry))
	for _, a := range Registry {
		out = append(out, a)
	}
	return out
}

// Get returns an action by name.
func Get(name string) (Action, bool) {
	a, ok := Registry[name]
	return a, ok
}
