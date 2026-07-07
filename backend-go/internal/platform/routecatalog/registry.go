// Package routecatalog provides the route-to-action binding registry.
//
// Each entry links an HTTP route pattern (method + Gin template path) to an
// action type from the actioncatalog. The approval middleware uses this to
// determine whether a request needs pre-execution approval.
//
// Routes with RiskHigh in the actioncatalog or RequireApproval=true are
// gated. Unregistered routes are allowed through (safe default); the CI check
// (check_audit_coverage.sh) warns when high-risk routes lack a binding.
package routecatalog

import (
	"strings"
)

// Binding links an HTTP route to an action type.
type Binding struct {
	Method      string // GET, POST, PUT, PATCH, DELETE
	PathPrefix  string // Gin path prefix, e.g. "/api/v1/price"
	ActionType  string // actioncatalog action type, e.g. "price_update"
	Description string // human-readable description
}

// AllBindings returns the complete route-to-action binding list.
// CI reads this list to verify that all high-risk routes are registered.
func AllBindings() []Binding {
	return []Binding{
		// ── Price (high risk) ──
		{Method: "PUT", PathPrefix: "/api/v1/price", ActionType: "price_update", Description: "更新售价"},
		{Method: "POST", PathPrefix: "/api/v1/price", ActionType: "price_update", Description: "创建定价"},
		{Method: "PATCH", PathPrefix: "/api/v1/price", ActionType: "price_update", Description: "部分更新售价"},

		// ── Inventory (high risk) ──
		{Method: "PUT", PathPrefix: "/api/v1/inventory", ActionType: "sync_inventory", Description: "更新库存"},
		{Method: "PATCH", PathPrefix: "/api/v1/inventory", ActionType: "sync_inventory", Description: "部分更新库存"},
		{Method: "POST", PathPrefix: "/api/v1/inventory", ActionType: "sync_inventory", Description: "创建库存记录"},

		// ── Orders (high risk) ──
		{Method: "POST", PathPrefix: "/api/v1/order/cancel", ActionType: "order_cancel", Description: "取消订单"},
		{Method: "POST", PathPrefix: "/api/v1/order/refund", ActionType: "refund_issue", Description: "发起退款"},
		{Method: "PATCH", PathPrefix: "/api/v1/order", ActionType: "order_update", Description: "更新订单"},
		{Method: "DELETE", PathPrefix: "/api/v1/order", ActionType: "order_cancel", Description: "删除订单"},

		// ── Platform Integrations publish (high risk) ──
		{Method: "POST", PathPrefix: "/api/v1/listings", ActionType: "listing_optimize", Description: "创建/发布 listing"},
		{Method: "PUT", PathPrefix: "/api/v1/listings", ActionType: "listing_optimize", Description: "更新 listing"},
		{Method: "POST", PathPrefix: "/api/v1/list", ActionType: "list_generation", Description: "生成 listing 并发布"},
		{Method: "POST", PathPrefix: "/api/v1/integrations/platforms/publish", ActionType: "auto_publish", Description: "平台发布"},

		// ── Permissions (high risk) ──
		{Method: "POST", PathPrefix: "/api/v1/rbac", ActionType: "permission_change", Description: "创建 RBAC 规则"},
		{Method: "PUT", PathPrefix: "/api/v1/rbac", ActionType: "permission_change", Description: "修改权限"},
		{Method: "DELETE", PathPrefix: "/api/v1/rbac", ActionType: "permission_change", Description: "删除权限"},

		// ── Credentials (high risk) ──
		{Method: "PUT", PathPrefix: "/api/v1/integrations/credentials", ActionType: "credential_change", Description: "修改平台凭证"},
		{Method: "POST", PathPrefix: "/api/v1/integrations/credentials", ActionType: "credential_change", Description: "创建平台凭证"},
		{Method: "DELETE", PathPrefix: "/api/v1/integrations/credentials", ActionType: "credential_change", Description: "删除平台凭证"},
		{Method: "PUT", PathPrefix: "/api/v1/auth", ActionType: "credential_change", Description: "修改登录凭证"},

		// ── Settlements & Finance (high risk) ──
		{Method: "POST", PathPrefix: "/api/v1/settlement", ActionType: "destructive_data_change", Description: "创建结算"},
		{Method: "PUT", PathPrefix: "/api/v1/settlement", ActionType: "destructive_data_change", Description: "更新结算"},
		{Method: "POST", PathPrefix: "/api/v1/finance", ActionType: "destructive_data_change", Description: "财务操作"},

		// ── Agent actions (via HTTP) ──
		{Method: "POST", PathPrefix: "/api/v1/agents/approve", ActionType: "agent_approve", Description: "审批 Agent 动作"},
	}
}

// GetActionType returns the action type for a given HTTP method and path.
// Returns empty string if no binding matches.
func GetActionType(method, path string) string {
	for _, b := range AllBindings() {
		if b.Method != method {
			continue
		}
		if strings.HasPrefix(path, b.PathPrefix) {
			return b.ActionType
		}
	}
	return ""
}

// IsHighRisk returns true if the route is registered as a high-risk action.
func IsHighRisk(method, path string) bool {
	return GetActionType(method, path) != ""
}
