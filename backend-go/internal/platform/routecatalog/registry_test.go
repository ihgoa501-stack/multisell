package routecatalog

import (
	"testing"
)

func TestGetActionType(t *testing.T) {
	tests := []struct {
		method     string
		path       string
		wantAction string
		wantHigh   bool
	}{
		{"PUT", "/api/v1/price/123", "price_update", true},
		{"POST", "/api/v1/price", "price_update", true},
		{"PATCH", "/api/v1/price/123", "price_update", true},
		{"PUT", "/api/v1/inventory/123", "sync_inventory", true},
		{"POST", "/api/v1/inventory/adjust", "sync_inventory", true},
		{"POST", "/api/v1/order/cancel/123", "order_cancel", true},
		{"POST", "/api/v1/order/refund/123", "refund_issue", true},
		{"PUT", "/api/v1/integrations/credentials", "credential_change", true},
		{"GET", "/api/v1/price", "", false},
		{"GET", "/api/v1/products", "", false},
		{"POST", "/api/v1/products", "", false}, // not high-risk per registry
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			got := GetActionType(tt.method, tt.path)
			if (got != "") != tt.wantHigh {
				t.Errorf("GetActionType(%q, %q) = %q, want high_risk=%v", tt.method, tt.path, got, tt.wantHigh)
			}
			if got != tt.wantAction {
				t.Errorf("GetActionType(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.wantAction)
			}
			if IsHighRisk(tt.method, tt.path) != tt.wantHigh {
				t.Errorf("IsHighRisk(%q, %q) = %v, want %v", tt.method, tt.path, !tt.wantHigh, tt.wantHigh)
			}
		})
	}
}

func TestUnknownRoutesNotHighRisk(t *testing.T) {
	routes := []struct{ method, path string }{
		{"GET", "/api/v1/health"},
		{"POST", "/api/v1/auth/login"},
		{"GET", "/api/v1/products"},
		{"POST", "/api/v1/candidate"},
	}

	for _, r := range routes {
		if IsHighRisk(r.method, r.path) {
			t.Errorf("unexpected high-risk for %s %s", r.method, r.path)
		}
	}
}

func TestAllBindingsHaveActionTypes(t *testing.T) {
	for _, b := range AllBindings() {
		if b.ActionType == "" {
			t.Errorf("binding %s %s has empty action type", b.Method, b.PathPrefix)
		}
		if b.Method == "" {
			t.Errorf("binding for %s has empty method", b.ActionType)
		}
		if b.PathPrefix == "" {
			t.Errorf("binding for %s has empty path prefix", b.ActionType)
		}
	}
}
