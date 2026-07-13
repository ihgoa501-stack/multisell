package routecatalog

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
)

func TestMutationPolicyMatchesHighRiskRegistry(t *testing.T) {
	actions := actioncatalog.Default()
	seenHigh := 0
	for _, policy := range MutationPolicies() {
		action, registered := ValidateRoute(policy.Method, policy.Path)
		if policy.Class == MutationHigh {
			seenHigh++
			if !registered || action != policy.ActionType {
				t.Errorf("high policy %s %s action=%q registry=(%q,%v)", policy.Method, policy.Path, policy.ActionType, action, registered)
			}
			if _, exists := actions.Lookup(policy.ActionType); !exists {
				t.Errorf("high policy %s %s uses unknown action %q", policy.Method, policy.Path, policy.ActionType)
			}
			continue
		}
		if registered {
			t.Errorf("non-high policy %s %s is present in high-risk registry", policy.Method, policy.Path)
		}
	}
	if seenHigh != len(DefaultBindings()) {
		t.Fatalf("policy has %d high routes, registry has %d", seenHigh, len(DefaultBindings()))
	}
}

func TestMutationPolicyPublicRoutesAreOnlyAuthLifecycle(t *testing.T) {
	allowed := map[string]struct{}{
		"POST:/api/v1/auth/login": {}, "POST:/api/v1/auth/register": {},
		"POST:/api/v1/auth/refresh": {}, "POST:/api/v1/webhooks/:platform": {},
		"POST:/api/v1/auth/extension-pairings/claim": {}, "POST:/api/v1/auth/extension-pairings/exchange": {},
		"POST:/api/v1/auth/extension-devices/refresh": {},
		"POST:/api/v1/feedback/submissions":           {},
	}
	for _, policy := range MutationPolicies() {
		if policy.Class == MutationPublic {
			if _, ok := allowed[policy.Method+":"+policy.Path]; ok {
				continue
			}
			t.Errorf("unexpected public mutation: %s %s", policy.Method, policy.Path)
		}
	}
}

func TestResolveMutationPolicyActualPath(t *testing.T) {
	policy, ok := ResolveMutationPolicy("POST", "/api/v1/listings/42/publish")
	if !ok || policy.Class != MutationHigh || policy.ActionType != "auto_publish" {
		t.Fatalf("high policy = %+v, %v", policy, ok)
	}
	policy, ok = ResolveMutationPolicy("POST", "/api/v1/xiao-q/messages")
	if !ok || policy.Class != MutationStandard {
		t.Fatalf("standard policy = %+v, %v", policy, ok)
	}
	if _, ok := ResolveMutationPolicy("POST", "/api/v1/unclassified/write"); ok {
		t.Fatal("unclassified mutation resolved")
	}
}

func TestMutationPolicyHasCurrentFullInventory(t *testing.T) {
	// This number is not the security gate by itself; the source cross-check in
	// check_audit_coverage.sh catches additions, removals and path changes. It
	// makes accidental parser or embedded-file truncation immediately visible.
	if got := len(MutationPolicies()); got != 474 {
		t.Fatalf("expected 474 explicitly classified mutations, got %d", got)
	}
}

func TestAllDeletesAreHighRisk(t *testing.T) {
	for _, policy := range MutationPolicies() {
		if policy.Method == "DELETE" && policy.Class != MutationHigh {
			t.Errorf("destructive DELETE is not high risk: %s", policy.Path)
		}
	}
}

func TestSourcingMaterialMutationsHaveReviewedStandardRisk(t *testing.T) {
	// These mutations only organize Owner-private evidence or record an internal
	// eligibility fact. They perform no external write, purchase, publication,
	// credential change, deletion, or autonomous action. Domain gates still
	// enforce Owner/task isolation, immutable source hashes, approved rights and
	// controlled renditions. Real publication remains separately high-risk.
	paths := []struct{ method, path string }{
		{"POST", "/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets"},
		{"PATCH", "/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/order"},
		{"POST", "/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/archive"},
		{"POST", "/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/mark-used"},
		{"POST", "/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/rights-evidence"},
		{"POST", "/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/rights-evidence/:evidenceId/review"},
		{"POST", "/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/rights-evidence/:evidenceId/revoke"},
		{"POST", "/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/renditions"},
	}
	for _, route := range paths {
		policy, ok := ResolveMutationPolicy(route.method, route.path)
		if !ok || policy.Class != MutationStandard {
			t.Errorf("material route %s %s policy=%+v found=%v", route.method, route.path, policy, ok)
		}
		if _, high := ValidateRoute(route.method, route.path); high {
			t.Errorf("internal material organization unexpectedly requires external-action approval: %s %s", route.method, route.path)
		}
	}
}

func TestValidateRuntimeRoutesBidirectional(t *testing.T) {
	routes := make([]RuntimeRoute, 0, len(MutationPolicies()))
	for _, policy := range MutationPolicies() {
		routes = append(routes, RuntimeRoute{Method: policy.Method, Path: policy.Path})
	}
	if err := ValidateRuntimeRoutes(routes); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeRoutes(append(routes, RuntimeRoute{Method: "POST", Path: "/api/v1/unclassified/live"})); err == nil {
		t.Fatal("live unclassified mutation was accepted")
	}
	withoutRequired := make([]RuntimeRoute, 0, len(routes)-1)
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/api/v1/xiao-q/messages" {
			continue
		}
		withoutRequired = append(withoutRequired, route)
	}
	if err := ValidateRuntimeRoutes(withoutRequired); err == nil {
		t.Fatal("stale mutation policy was accepted")
	}
}
