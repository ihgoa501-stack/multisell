package routecatalog

import (
	"os"
	"strings"
	"testing"
)

func TestGetActionType(t *testing.T) {
	tests := []struct {
		method     string
		fullPath   string // as returned by gin.Context.FullPath()
		wantAction string
	}{
		// Price
		{"POST", "/api/v1/prices", "price_update"},
		{"PUT", "/api/v1/prices/:id", "price_update"},
		{"DELETE", "/api/v1/prices/:id", "price_update"},
		{"POST", "/api/v1/competitor-prices", "price_update"},
		{"POST", "/api/v1/pricing-recommendations/:id/apply", "price_update"},

		// Order
		{"POST", "/api/v1/order", "order_cancel"},
		{"PUT", "/api/v1/order/:id", "order_cancel"},
		{"POST", "/api/v1/order/:id/status", "order_cancel"},

		// Inventory
		{"PUT", "/api/v1/inventory/:id", "sync_inventory"},
		{"POST", "/api/v1/inventory/sync-cross-platform/:productId", "sync_inventory"},

		// Integrations

		// RBAC
		{"POST", "/api/v1/rbac/roles", "permission_change"},
		{"PUT", "/api/v1/rbac/roles/:id", "permission_change"},
		{"POST", "/api/v1/rbac/permissions", "permission_change"},
		{"POST", "/api/v1/rbac/users/:id/roles", "permission_change"},
		{"DELETE", "/api/v1/rbac/users/:id/roles", ""},

		// Listings
		{"POST", "/api/v1/listings/:id/publish", "auto_publish"},
		{"POST", "/api/v1/listing/products/:product_id/publish/:platform_id", "auto_publish"},

		// Aftersales
		{"POST", "/api/v1/aftersales/:id/refund", "refund_issue"},
		{"POST", "/api/v1/aftersales/:id/approve", "refund_issue"},

		// Non-matching
		{"GET", "/api/v1/prices", ""},
		{"GET", "/api/v1/products", ""},
		{"POST", "/api/v1/products", ""},
		{"POST", "/api/v1/order/summary", ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.fullPath, func(t *testing.T) {
			got := GetActionType(tt.method, tt.fullPath)
			if got != tt.wantAction {
				t.Errorf("GetActionType(%q, %q) = %q, want %q", tt.method, tt.fullPath, got, tt.wantAction)
			}
			if (got != "") != IsHighRisk(tt.method, tt.fullPath) {
				t.Errorf("IsHighRisk(%q, %q) inconsistency", tt.method, tt.fullPath)
			}
		})
	}
}

func TestUnknownRoutesNotHighRisk(t *testing.T) {
	routes := []struct{ method, fullPath string }{
		{"GET", "/api/v1/health"},
		{"POST", "/api/v1/auth/login"},
		{"GET", "/api/v1/products"},
		{"POST", "/api/v1/candidate"},
		{"GET", "/api/v1/prices"},
		{"GET", "/api/v1/order/summary"},
	}

	for _, r := range routes {
		if IsHighRisk(r.method, r.fullPath) {
			t.Errorf("unexpected high-risk for %s %s", r.method, r.fullPath)
		}
	}
}

func TestAllBindingsHaveValidPaths(t *testing.T) {
	for _, b := range DefaultBindings() {
		if b.ActionType == "" {
			t.Errorf("binding %s %s has empty action type", b.Method, b.PathPattern)
		}
		if b.Method == "" {
			t.Errorf("binding for %s has empty method", b.ActionType)
		}
		if b.PathPattern == "" {
			t.Errorf("binding for %s has empty path", b.ActionType)
		}
		if !strings.HasPrefix(b.PathPattern, "/api/v1/") {
			t.Errorf("binding %s %s path must start with /api/v1/", b.Method, b.PathPattern)
		}
	}
}

func TestValidateRoute(t *testing.T) {
	at, ok := ValidateRoute("PUT", "/api/v1/prices/:id")
	if !ok || at != "price_update" {
		t.Errorf("ValidateRoute PUT /api/v1/prices/:id = (%q, %v), want (price_update, true)", at, ok)
	}

	at, ok = ValidateRoute("GET", "/api/v1/prices")
	if ok {
		t.Errorf("ValidateRoute GET /api/v1/prices should not be registered")
	}
}

func TestBindingsCount(t *testing.T) {
	n := len(DefaultBindings())
	if n < 90 {
		t.Fatalf("expected at least 90 high-risk route bindings, got %d — new routes added?", n)
	}
	if n > 220 {
		t.Fatalf("expected at most 220 high-risk route bindings, got %d — review if all need approval", n)
	}
}

func TestResolveActualRequestPath(t *testing.T) {
	pattern, action, ok := Resolve("POST", "/api/v1/listings/42/publish")
	if !ok || pattern != "/api/v1/listings/:id/publish" || action != "auto_publish" {
		t.Fatalf("Resolve() = %q %q %v", pattern, action, ok)
	}
	if _, _, ok := Resolve("GET", "/api/v1/listings/42/publish"); ok {
		t.Fatal("wrong method matched high-risk route")
	}
	if _, _, ok := Resolve("POST", "/api/v1/not-registered/42"); ok {
		t.Fatal("unknown path matched high-risk route")
	}
}

func TestMain(m *testing.M) {
	// init() already runs — matchTable is built.
	os.Exit(m.Run())
}
