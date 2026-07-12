// Package routecatalog provides the route-to-action binding registry.
//
// Each entry links a Gin route template (method + full path template) to an
// action type from the actioncatalog. The approval middleware uses exact
// matching against c.FullPath() to determine whether a request needs
// pre-execution approval.
//
// New high-risk routes MUST be registered here. If CI finds a high-risk route
// that is NOT in this registry, it fails the build.
//
// Routes not in this registry are allowed through (safe default — existing
// non-high-risk routes are unaffected). Only mutating methods
// (POST/PUT/PATCH/DELETE) are gated by the middleware.
//
// CI check: check_audit_coverage.sh step [5/5] cross-references registered
// Gin routes against this catalog and warns about any unregistered routes.
// Currently warning-only; make it a hard error once all routes are audited.
package routecatalog

// Binding links a Gin route template to an action type.
type Binding struct {
	Method      string // GET, POST, PUT, PATCH, DELETE
	PathPattern string // Gin route template, e.g. "/api/v1/prices/:id"
	ActionType  string // actioncatalog action type, e.g. "price_update"
	Description string // human-readable description
}

// matchTable is the pre-built lookup table: "METHOD:/api/v1/path" -> actionType.
var matchTable map[string]string

func init() {
	matchTable = make(map[string]string, len(DefaultBindings()))
	for _, b := range DefaultBindings() {
		matchTable[b.Method+":"+b.PathPattern] = b.ActionType
	}
}

// DefaultBindings returns all route-to-action bindings.
// These are the EXACT Gin route templates (use c.FullPath() to match).
func DefaultBindings() []Binding {
	return []Binding{
		// ── Price mutations ──
		{Method: "POST", PathPattern: "/api/v1/prices", ActionType: "price_update", Description: "设置售价"},
		{Method: "PUT", PathPattern: "/api/v1/prices/:id", ActionType: "price_update", Description: "更新售价"},
		{Method: "DELETE", PathPattern: "/api/v1/prices/:id", ActionType: "price_update", Description: "删除价格"},
		{Method: "POST", PathPattern: "/api/v1/competitor-prices", ActionType: "price_update", Description: "创建竞品价格"},
		{Method: "DELETE", PathPattern: "/api/v1/competitor-prices/:id", ActionType: "price_update", Description: "删除竞品价格"},
		{Method: "POST", PathPattern: "/api/v1/pricing-recommendations/:id/apply", ActionType: "price_update", Description: "应用定价建议"},
		{Method: "POST", PathPattern: "/api/v1/pricing-recommendations/generate", ActionType: "price_update", Description: "生成定价建议"},
		{Method: "POST", PathPattern: "/api/v1/pricing-strategies", ActionType: "price_update", Description: "保存定价策略"},
		{Method: "PUT", PathPattern: "/api/v1/pricing-strategies/:id", ActionType: "price_update", Description: "更新定价策略"},
		{Method: "DELETE", PathPattern: "/api/v1/pricing-strategies/:id", ActionType: "price_update", Description: "删除定价策略"},

		// ── Order mutations ──
		{Method: "POST", PathPattern: "/api/v1/order", ActionType: "order_cancel", Description: "创建订单"},
		{Method: "PUT", PathPattern: "/api/v1/order/:id", ActionType: "order_cancel", Description: "更新订单"},
		{Method: "DELETE", PathPattern: "/api/v1/order/:id", ActionType: "order_cancel", Description: "删除订单"},
		{Method: "POST", PathPattern: "/api/v1/order/:id/status", ActionType: "order_cancel", Description: "更新订单状态"},

		// ── Inventory mutations ──
		{Method: "PUT", PathPattern: "/api/v1/inventory/:id", ActionType: "sync_inventory", Description: "更新库存"},
		{Method: "POST", PathPattern: "/api/v1/inventory/:id/lock", ActionType: "sync_inventory", Description: "锁定库存"},
		{Method: "POST", PathPattern: "/api/v1/inventory/:id/unlock", ActionType: "sync_inventory", Description: "解锁库存"},
		{Method: "POST", PathPattern: "/api/v1/inventory/sync-cross-platform/:productId", ActionType: "sync_inventory", Description: "跨平台库存同步"},
		{Method: "PUT", PathPattern: "/api/v1/inventory/safety-config/:sku_id", ActionType: "sync_inventory", Description: "更新安全库存配置"},
		{Method: "POST", PathPattern: "/api/v1/inventory/dead-stock/analyze", ActionType: "sync_inventory", Description: "分析滞销库存"},

		// ── Platform integrations mutations ──
		{Method: "POST", PathPattern: "/api/v1/platform-integrations", ActionType: "credential_change", Description: "创建平台集成"},
		{Method: "PUT", PathPattern: "/api/v1/platform-integrations/:id", ActionType: "credential_change", Description: "更新平台集成"},
		{Method: "DELETE", PathPattern: "/api/v1/platform-integrations/:id", ActionType: "credential_change", Description: "删除平台集成"},
		{Method: "POST", PathPattern: "/api/v1/platform-integrations/publish-to-ozon", ActionType: "auto_publish", Description: "发布到 Ozon"},
		{Method: "POST", PathPattern: "/api/v1/platform-integrations/write-back", ActionType: "auto_publish", Description: "回写到平台"},
		{Method: "POST", PathPattern: "/api/v1/platform-integrations/write-back/:ref-id/retry", ActionType: "auto_publish", Description: "重试回写"},
		{Method: "PUT", PathPattern: "/api/v1/platform-integrations/:id/mode", ActionType: "credential_change", Description: "修改集成模式"},
		{Method: "POST", PathPattern: "/api/v1/platform-integrations/:id/test", ActionType: "credential_change", Description: "测试连接"},
		{Method: "POST", PathPattern: "/api/v1/platform-integrations/:id/sync", ActionType: "sync_inventory", Description: "同步平台数据"},
		{Method: "POST", PathPattern: "/api/v1/platform-integrations/:id/categories", ActionType: "credential_change", Description: "创建平台类目"},
		{Method: "POST", PathPattern: "/api/v1/platform-integrations/:id/attributes", ActionType: "credential_change", Description: "创建平台属性"},
		{Method: "POST", PathPattern: "/api/v1/platform-integrations/mock/seed", ActionType: "destructive_data_change", Description: "写入平台模拟数据"},

		// ── Settlement mutations ──
		{Method: "POST", PathPattern: "/api/v1/settlement", ActionType: "destructive_data_change", Description: "创建结算"},
		{Method: "PUT", PathPattern: "/api/v1/settlement/:id", ActionType: "destructive_data_change", Description: "更新结算"},
		{Method: "DELETE", PathPattern: "/api/v1/settlement/:id", ActionType: "destructive_data_change", Description: "删除结算"},
		{Method: "POST", PathPattern: "/api/v1/settlement/:id/reconcile", ActionType: "destructive_data_change", Description: "对账"},
		{Method: "POST", PathPattern: "/api/v1/settlement/:id/items", ActionType: "destructive_data_change", Description: "添加结算项"},
		{Method: "PUT", PathPattern: "/api/v1/settlement/items/:item_id/reconciliation", ActionType: "destructive_data_change", Description: "更新对账状态"},
		{Method: "POST", PathPattern: "/api/v1/settlement/recalculate", ActionType: "destructive_data_change", Description: "重算结算利润"},

		// ── Finance account/transaction mutations ──
		{Method: "POST", PathPattern: "/api/v1/finance/accounts", ActionType: "destructive_data_change", Description: "创建财务账户"},
		{Method: "PUT", PathPattern: "/api/v1/finance/accounts/:id", ActionType: "destructive_data_change", Description: "更新财务账户"},
		{Method: "DELETE", PathPattern: "/api/v1/finance/accounts/:id", ActionType: "destructive_data_change", Description: "删除财务账户"},
		{Method: "POST", PathPattern: "/api/v1/finance/transactions", ActionType: "destructive_data_change", Description: "创建财务流水"},
		{Method: "POST", PathPattern: "/api/v1/finance/orders/:order_id/ledger/rebuild", ActionType: "destructive_data_change", Description: "重建订单账本"},
		{Method: "POST", PathPattern: "/api/v1/finance/profit/calculate", ActionType: "destructive_data_change", Description: "利润测算"},
		{Method: "POST", PathPattern: "/api/v1/finance/profit/batch-calculate", ActionType: "destructive_data_change", Description: "批量利润测算"},
		{Method: "POST", PathPattern: "/api/v1/finance/mock", ActionType: "destructive_data_change", Description: "财务模拟"},

		// ── RBAC mutations ──
		{Method: "POST", PathPattern: "/api/v1/rbac/roles", ActionType: "permission_change", Description: "创建角色"},
		{Method: "PUT", PathPattern: "/api/v1/rbac/roles/:id", ActionType: "permission_change", Description: "更新角色"},
		{Method: "DELETE", PathPattern: "/api/v1/rbac/roles/:id", ActionType: "permission_change", Description: "删除角色"},
		{Method: "POST", PathPattern: "/api/v1/rbac/roles/:id/permissions", ActionType: "permission_change", Description: "分配角色权限"},
		{Method: "POST", PathPattern: "/api/v1/rbac/permissions", ActionType: "permission_change", Description: "创建权限"},
		{Method: "PUT", PathPattern: "/api/v1/rbac/permissions/:id", ActionType: "permission_change", Description: "更新权限"},
		{Method: "DELETE", PathPattern: "/api/v1/rbac/permissions/:id", ActionType: "permission_change", Description: "删除权限"},
		{Method: "DELETE", PathPattern: "/api/v1/rbac/users/:id/roles", ActionType: "permission_change", Description: "分配用户角色"},

		// ── Listings / Listing task mutations ──
		{Method: "POST", PathPattern: "/api/v1/listings", ActionType: "listing_optimize", Description: "创建 listing"},
		{Method: "PUT", PathPattern: "/api/v1/listings/:id", ActionType: "listing_optimize", Description: "更新 listing"},
		{Method: "DELETE", PathPattern: "/api/v1/listings/:id", ActionType: "listing_optimize", Description: "删除 listing"},
		{Method: "POST", PathPattern: "/api/v1/listings/:id/publish", ActionType: "auto_publish", Description: "发布 listing"},
		{Method: "POST", PathPattern: "/api/v1/listings/:id/sync", ActionType: "listing_optimize", Description: "同步 listing"},
		{Method: "POST", PathPattern: "/api/v1/listings/suggest", ActionType: "listing_optimize", Description: "生成 listing 建议"},
		{Method: "POST", PathPattern: "/api/v1/listing", ActionType: "listing_optimize", Description: "创建 listing chain"},
		{Method: "POST", PathPattern: "/api/v1/listing/products/:product_id/publish/:platform_id", ActionType: "auto_publish", Description: "发布产品到平台"},
		{Method: "POST", PathPattern: "/api/v1/listing/listing-tasks/:task_id/publish", ActionType: "auto_publish", Description: "发布 listing 任务"},
		{Method: "POST", PathPattern: "/api/v1/listing/listing-tasks/:task_id/recheck", ActionType: "listing_optimize", Description: "重新检查 listing 任务"},
		{Method: "POST", PathPattern: "/api/v1/listing/listing-tasks/:task_id/cancel", ActionType: "listing_optimize", Description: "取消 listing 任务"},
		{Method: "POST", PathPattern: "/api/v1/listing/listing-tasks/from-decisions", ActionType: "listing_optimize", Description: "从决策创建任务"},
		{Method: "POST", PathPattern: "/api/v1/listing-tasks", ActionType: "listing_optimize", Description: "创建 listing 任务"},
		{Method: "PUT", PathPattern: "/api/v1/listing-tasks/:id", ActionType: "listing_optimize", Description: "更新 listing 任务"},
		{Method: "DELETE", PathPattern: "/api/v1/listing-tasks/:id", ActionType: "listing_optimize", Description: "删除 listing 任务"},
		{Method: "POST", PathPattern: "/api/v1/listing-tasks/from-suggestion", ActionType: "listing_optimize", Description: "从建议创建任务"},
		{Method: "POST", PathPattern: "/api/v1/listing-tasks/:id/items", ActionType: "listing_optimize", Description: "创建 listing 任务条目"},
		{Method: "PUT", PathPattern: "/api/v1/listing-tasks/:id/items/:item_id", ActionType: "listing_optimize", Description: "更新 listing 任务条目"},
		{Method: "DELETE", PathPattern: "/api/v1/listing-tasks/:id/items/:item_id", ActionType: "listing_optimize", Description: "删除 listing 任务条目"},
		{Method: "POST", PathPattern: "/api/v1/listing-task/:task_id/execute", ActionType: "listing_optimize", Description: "执行 listing 任务"},
		{Method: "POST", PathPattern: "/api/v1/listing-task/retry-all", ActionType: "listing_optimize", Description: "重试所有失败"},
		{Method: "POST", PathPattern: "/api/v1/listing-task/:task_id/retry-failed", ActionType: "listing_optimize", Description: "重试失败项"},
		{Method: "POST", PathPattern: "/api/v1/listing-task/:task_id/feedback", ActionType: "listing_optimize", Description: "记录 listing 任务反馈"},
		{Method: "POST", PathPattern: "/api/v1/listing-task/:task_id/items/:item_id/retry", ActionType: "listing_optimize", Description: "重试单个条目"},

		// ── Platform / Store CRUD ──
		{Method: "POST", PathPattern: "/api/v1/platforms", ActionType: "destructive_data_change", Description: "创建平台"},
		{Method: "PUT", PathPattern: "/api/v1/platforms/:id", ActionType: "destructive_data_change", Description: "更新平台"},
		{Method: "DELETE", PathPattern: "/api/v1/platforms/:id", ActionType: "destructive_data_change", Description: "删除平台"},
		{Method: "POST", PathPattern: "/api/v1/stores", ActionType: "destructive_data_change", Description: "创建店铺"},
		{Method: "PUT", PathPattern: "/api/v1/stores/:id", ActionType: "destructive_data_change", Description: "更新店铺"},
		{Method: "DELETE", PathPattern: "/api/v1/stores/:id", ActionType: "destructive_data_change", Description: "删除店铺"},

		// ── Product Master / SKU CRUD ──
		{Method: "POST", PathPattern: "/api/v1/product-master", ActionType: "destructive_data_change", Description: "创建产品"},
		{Method: "PUT", PathPattern: "/api/v1/product-master/:id", ActionType: "destructive_data_change", Description: "更新产品"},
		{Method: "DELETE", PathPattern: "/api/v1/product-master/:id", ActionType: "destructive_data_change", Description: "删除产品"},
		{Method: "POST", PathPattern: "/api/v1/product-master/:id/specs", ActionType: "destructive_data_change", Description: "创建产品规格"},
		{Method: "PUT", PathPattern: "/api/v1/product-master/:id/specs/:spec_id", ActionType: "destructive_data_change", Description: "更新产品规格"},
		{Method: "DELETE", PathPattern: "/api/v1/product-master/:id/specs/:spec_id", ActionType: "destructive_data_change", Description: "删除产品规格"},
		{Method: "POST", PathPattern: "/api/v1/product-master/:id/specs/:spec_id/values", ActionType: "destructive_data_change", Description: "创建规格值"},
		{Method: "PUT", PathPattern: "/api/v1/spec-values/:id", ActionType: "destructive_data_change", Description: "更新规格值"},
		{Method: "DELETE", PathPattern: "/api/v1/spec-values/:id", ActionType: "destructive_data_change", Description: "删除规格值"},
		{Method: "POST", PathPattern: "/api/v1/skus", ActionType: "destructive_data_change", Description: "创建 SKU"},
		{Method: "PUT", PathPattern: "/api/v1/skus/:id", ActionType: "destructive_data_change", Description: "更新 SKU"},
		{Method: "DELETE", PathPattern: "/api/v1/skus/:id", ActionType: "destructive_data_change", Description: "删除 SKU"},

		// ── Aftersales mutations ──
		{Method: "POST", PathPattern: "/api/v1/aftersales", ActionType: "refund_issue", Description: "创建售后"},
		{Method: "PUT", PathPattern: "/api/v1/aftersales/:id", ActionType: "refund_issue", Description: "更新售后"},
		{Method: "DELETE", PathPattern: "/api/v1/aftersales/:id", ActionType: "refund_issue", Description: "删除售后"},
		{Method: "POST", PathPattern: "/api/v1/aftersales/:id/auto-decide", ActionType: "refund_issue", Description: "自动决策售后"},
		{Method: "POST", PathPattern: "/api/v1/aftersales/:id/approve", ActionType: "refund_issue", Description: "审批售后"},
		{Method: "POST", PathPattern: "/api/v1/aftersales/:id/reject", ActionType: "refund_issue", Description: "驳回售后"},
		{Method: "POST", PathPattern: "/api/v1/aftersales/:id/refund", ActionType: "refund_issue", Description: "执行退款"},
		{Method: "POST", PathPattern: "/api/v1/aftersales/:id/receive", ActionType: "refund_issue", Description: "确认收货"},
		{Method: "POST", PathPattern: "/api/v1/aftersales/disputes", ActionType: "refund_issue", Description: "创建售后争议"},
		{Method: "POST", PathPattern: "/api/v1/aftersales/disputes/:id/evaluate", ActionType: "refund_issue", Description: "评估争议"},
		{Method: "POST", PathPattern: "/api/v1/aftersales/disputes/:id/auto-decide", ActionType: "refund_issue", Description: "自动决策争议"},
		{Method: "PUT", PathPattern: "/api/v1/aftersales/disputes/:id/status", ActionType: "refund_issue", Description: "更新争议状态"},

		// ── Decision mutations ──
		{Method: "POST", PathPattern: "/api/v1/decision", ActionType: "agent_approve", Description: "创建决策"},
		{Method: "POST", PathPattern: "/api/v1/decision/:id/approve", ActionType: "agent_approve", Description: "审批决策"},
		{Method: "POST", PathPattern: "/api/v1/decision/:id/reject", ActionType: "agent_approve", Description: "驳回决策"},
	}
}

// GetActionType returns the action type for a given HTTP method and Gin full path.
// Returns empty string if no binding matches.
// fullPath should be the value of c.FullPath() — the Gin route template.
func GetActionType(method, fullPath string) string {
	return matchTable[method+":"+fullPath]
}

// IsHighRisk returns true if the route is registered in the high-risk catalog.
func IsHighRisk(method, fullPath string) bool {
	_, ok := matchTable[method+":"+fullPath]
	return ok
}

// ValidateRoute checks whether a given (method, fullPath) is registered in the
// catalog. Returns true and the action type if registered. Used by CI to verify
// coverage.
func ValidateRoute(method, fullPath string) (actionType string, registered bool) {
	at := matchTable[method+":"+fullPath]
	if at != "" {
		return at, true
	}
	return "", false
}
