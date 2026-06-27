package ai

import "strings"

// AgentSpec describes a registered AI agent.
type AgentSpec struct {
	ID             string   `json:"agent_id"`
	Name           string   `json:"name"`
	Squad          string   `json:"squad"`           // autonomous | governance | ops
	Autonomy       string   `json:"autonomy"`        // advisory | guided | autonomous | supervised
	DecisionPoints []string `json:"decision_points"`
	Description    string   `json:"description"`
	ModelHint      string   `json:"model_hint,omitempty"`
	RiskFloor      string   `json:"risk_floor,omitempty"` // minimum risk level for this agent's actions
}

// PrimaryDecisionPoint returns the first decision point for display.
func (a AgentSpec) PrimaryDecisionPoint() string {
	if len(a.DecisionPoints) == 0 {
		return ""
	}
	return a.DecisionPoints[0]
}

// AgentRegistry holds all registered agents.
type AgentRegistry struct {
	Agents []AgentSpec
	byID   map[string]AgentSpec
}

// DefaultRegistry returns the canonical LingMirror agent roster:
// A1-A7 (autonomous squad) + G0-G3 (governance squad) + A8/A10/A11.
func DefaultRegistry() *AgentRegistry {
	agents := []AgentSpec{
		{ID: "A1", Name: "Product Scout", Squad: "growth", Autonomy: "advisory",
			DecisionPoints: []string{"product_scout", "market_analysis"},
			Description:    "选品探路：市场机会扫描、新品推荐、竞品对标", ModelHint: "gpt-4o", RiskFloor: "low"},
		{ID: "A2", Name: "Listing Optimizer", Squad: "growth", Autonomy: "guided",
			DecisionPoints: []string{"listing_optimize", "keyword_research"},
			Description:    "Listing 优化：标题/描述/关键词自动重写", ModelHint: "gpt-4o", RiskFloor: "low"},
		{ID: "A3", Name: "Ad Advice", Squad: "growth", Autonomy: "advisory",
			DecisionPoints: []string{"acos_analysis", "ad_optimization"},
			Description:    "广告建议：ACOS 分析、投放优化", ModelHint: "gpt-4o-mini", RiskFloor: "low"},
		{ID: "A4", Name: "Customer Service", Squad: "growth", Autonomy: "autonomous",
			DecisionPoints: []string{"auto_reply", "intent_classify"},
			Description:    "客服自动化：意图分类、自动回复", ModelHint: "gpt-4o-mini", RiskFloor: "low"},
		{ID: "A5", Name: "Inventory Alert", Squad: "fulfillment", Autonomy: "supervised",
			DecisionPoints: []string{"stock_alert", "replenishment_plan", "logistics_choice"},
			Description:    "库存预警：缺货/补货/物流切换", ModelHint: "gpt-4o", RiskFloor: "medium"},
		{ID: "A6", Name: "Profit Watch", Squad: "risk", Autonomy: "supervised",
			DecisionPoints: []string{"profit_check", "cost_optimization", "profit_watch"},
			Description:    "利润看护：SKU 利润率、成本优化、亏损止血", ModelHint: "gpt-4o", RiskFloor: "medium"},
		{ID: "A7", Name: "Compliance Guard", Squad: "risk", Autonomy: "supervised",
			DecisionPoints: []string{"compliance_check", "certification_lookup"},
			Description:    "合规守门：商品合规、认证查询、禁售词", ModelHint: "gpt-4o", RiskFloor: "high"},
		{ID: "G1", Name: "Dashboard", Squad: "risk", Autonomy: "advisory",
			DecisionPoints: []string{"dashboard_overview"},
			Description:    "驾驶舱聚合：全局指标、趋势、异常汇总", ModelHint: "gpt-4o-mini", RiskFloor: "low"},
		{ID: "G2", Name: "Warehouse Customs", Squad: "fulfillment", Autonomy: "supervised",
			DecisionPoints: []string{"warehouse_routing", "customs_declare"},
			Description:    "仓储报关：仓库路由、报关单校验", ModelHint: "gpt-4o", RiskFloor: "high"},
		{ID: "G3", Name: "Discount Risk", Squad: "risk", Autonomy: "supervised",
			DecisionPoints: []string{"discount_check", "promotion_validation", "discount_risk_check"},
			Description:    "折扣风控：促销折扣风险、价格底线", ModelHint: "gpt-4o", RiskFloor: "high"},
		// G0 — Coordinator (supervisor, Phase 1)
		{ID: "G0", Name: "Coordinator", Squad: "governance", Autonomy: "supervised",
			DecisionPoints: []string{"system_health", "anomaly_escalation", "cross_squad_coordinate", "agent_audit"},
			Description:    "协调仲裁：系统健康、异常升级、跨Agent协作、Agent审计",
			ModelHint: "gpt-4o", RiskFloor: "high"},
		// A8 — Sourcing Agent (Phase 2)
		{ID: "A8", Name: "Sourcing Agent", Squad: "growth", Autonomy: "supervised",
			DecisionPoints: []string{"sourcing_recommend"},
			Description:    "AI 选品分析：1688 商品利润分析、采购推荐、选品到刊登全链路触发",
			ModelHint: "gpt-4o", RiskFloor: "medium"},
		// A10 — Logistics Optimization (Phase 2)
		{ID: "A10", Name: "Logistics Ops", Squad: "fulfillment", Autonomy: "supervised",
			DecisionPoints: []string{"carrier_compare", "shipping_bill_audit", "carrier_performance", "logistics_route_opt"},
			Description:    "物流优化：承运商比价、运费审计、绩效评估、仓配策略",
			ModelHint: "gpt-4o-mini", RiskFloor: "medium"},
		// A11 — Aftersales Management (Phase 2)
		{ID: "A11", Name: "Aftersales Mgmt", Squad: "settle", Autonomy: "supervised",
			DecisionPoints: []string{"return_analysis", "refund_decision", "dispute_manage", "aftersales_report"},
			Description:    "售后管理：退货分析、退款决策、纠纷处理、售后预警",
			ModelHint: "gpt-4o", RiskFloor: "medium"},
	}
	r := &AgentRegistry{Agents: agents, byID: make(map[string]AgentSpec, len(agents))}
	for _, a := range agents {
		r.byID[a.ID] = a
	}
	return r
}

// Get returns the agent spec by ID, case-insensitive.
func (r *AgentRegistry) Get(id string) (AgentSpec, bool) {
	a, ok := r.byID[strings.ToUpper(id)]
	return a, ok
}

// IDs returns all registered agent IDs in order.
func (r *AgentRegistry) IDs() []string {
	out := make([]string, 0, len(r.Agents))
	for _, a := range r.Agents {
		out = append(out, a.ID)
	}
	return out
}

// BySquad groups agents by squad.
func (r *AgentRegistry) BySquad() map[string][]AgentSpec {
	out := make(map[string][]AgentSpec)
	for _, a := range r.Agents {
		out[a.Squad] = append(out[a.Squad], a)
	}
	return out
}
