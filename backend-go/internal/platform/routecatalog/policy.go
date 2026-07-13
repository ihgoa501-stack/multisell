package routecatalog

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
)

// MutationClass is the explicit security treatment for an HTTP mutation.
type MutationClass string

const (
	MutationPublic   MutationClass = "public"
	MutationStandard MutationClass = "standard"
	MutationHigh     MutationClass = "high"
)

// MutationPolicy is the reviewed, fail-closed classification of one route.
// Standard mutations are authenticated and synchronously audited by the global
// middleware stack. High mutations additionally require a bound, one-time
// approval and an idempotency key. Public mutations are restricted to the auth
// session lifecycle and are still covered by rate limiting and audit policy.
type MutationPolicy struct {
	Class      MutationClass
	Method     string
	Path       string
	ActionType string
	Source     string
}

// RuntimeRoute is the framework-independent shape of a registered route.
type RuntimeRoute struct {
	Method string
	Path   string
}

//go:embed mutation_policy.tsv
var mutationPolicyData string

var mutationPolicies map[string]MutationPolicy

var optionalRuntimePolicies = map[string]struct{}{
	"POST:/api/v1/mock/seed":                                            {},
	"POST:/api/v1/owner/suggestions/:id/feedback":                       {},
	"POST:/api/v1/platform-integrations/mock/seed":                      {},
	"POST:/api/v1/shipping/carriers/:code/quote":                        {},
	"POST:/api/v1/content/generate":                                     {},
	"POST:/api/v1/content/validate":                                     {},
	"PUT:/api/v1/settings/llm":                                          {},
	"POST:/api/v1/sourcing/fetch":                                       {},
	"POST:/api/v1/agent-learning/evaluate":                              {},
	"POST:/api/v1/agent-learning/recalculate":                           {},
	"POST:/api/v1/agent-rules":                                          {},
	"DELETE:/api/v1/agent-rules/:id":                                    {},
	"PUT:/api/v1/agent-rules/:id":                                       {},
	"POST:/api/v1/agent-rules/:id/toggle":                               {},
	"POST:/api/v1/agent-rules/evaluate":                                 {},
	"POST:/api/v1/agents":                                               {},
	"POST:/api/v1/agents/:id/actions":                                   {},
	"POST:/api/v1/agents/rules":                                         {},
	"DELETE:/api/v1/agents/rules/:id":                                   {},
	"PUT:/api/v1/agents/rules/:id":                                      {},
	"POST:/api/v1/agents/rules/apply":                                   {},
	"POST:/api/v1/entropy/defense":                                      {},
	"POST:/api/v1/evolution/nudges/:id/accept":                          {},
	"POST:/api/v1/evolution/nudges/:id/dismiss":                         {},
	"POST:/api/v1/evolution/nudges/evaluate":                            {},
	"POST:/api/v1/metabolism/dry-run":                                   {},
	"POST:/api/v1/metabolism/execute":                                   {},
	"POST:/api/v1/orchestration/pipeline/config":                        {},
	"POST:/api/v1/orchestration/products/:id/pipeline/start":            {},
	"POST:/api/v1/orchestration/products/:id/pipeline/step/:step/retry": {},
	"PUT:/api/v1/trust-scores/:agent_id/level":                          {},
	"POST:/api/v1/trust-scores/auto-upgrade":                            {},
	"POST:/api/v1/trust-scores/eligible":                                {},
	"POST:/api/v1/trust-scores/recalculate":                             {},
}

func init() {
	policies, err := parseMutationPolicies(mutationPolicyData)
	if err != nil {
		panic(err)
	}
	mutationPolicies = policies
}

func parseMutationPolicies(data string) (map[string]MutationPolicy, error) {
	policies := make(map[string]MutationPolicy)
	for lineNumber, raw := range strings.Split(data, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 5 {
			return nil, fmt.Errorf("mutation policy line %d: expected 5 fields", lineNumber+1)
		}
		policy := MutationPolicy{
			Class: MutationClass(fields[0]), Method: fields[1], Path: fields[2],
			ActionType: fields[3], Source: fields[4],
		}
		if policy.Class != MutationPublic && policy.Class != MutationStandard && policy.Class != MutationHigh {
			return nil, fmt.Errorf("mutation policy line %d: invalid class %q", lineNumber+1, policy.Class)
		}
		if policy.Method != "POST" && policy.Method != "PUT" && policy.Method != "PATCH" && policy.Method != "DELETE" {
			return nil, fmt.Errorf("mutation policy line %d: invalid method %q", lineNumber+1, policy.Method)
		}
		if !strings.HasPrefix(policy.Path, "/api/v1/") {
			return nil, fmt.Errorf("mutation policy line %d: invalid path %q", lineNumber+1, policy.Path)
		}
		if policy.Class == MutationHigh && (policy.ActionType == "" || policy.ActionType == "-") {
			return nil, fmt.Errorf("mutation policy line %d: high route lacks action", lineNumber+1)
		}
		if policy.Class != MutationHigh && policy.ActionType != "-" {
			return nil, fmt.Errorf("mutation policy line %d: non-high route has action", lineNumber+1)
		}
		key := policy.Method + ":" + policy.Path
		if _, exists := policies[key]; exists {
			return nil, fmt.Errorf("mutation policy line %d: duplicate %s", lineNumber+1, key)
		}
		policies[key] = policy
	}
	return policies, nil
}

// GetMutationPolicy returns the explicit policy for an exact Gin route
// template. An absent policy is a configuration error and must fail CI or
// fail closed in release mode; callers must not silently infer "standard".
func GetMutationPolicy(method, path string) (MutationPolicy, bool) {
	policy, ok := mutationPolicies[method+":"+path]
	return policy, ok
}

// ResolveMutationPolicy matches an actual URL path against the explicitly
// classified Gin templates. It is used before Gin exposes Context.FullPath().
func ResolveMutationPolicy(method, requestPath string) (MutationPolicy, bool) {
	requestParts := splitPath(requestPath)
	bestSpecificity := -1
	var best MutationPolicy
	ambiguous := false
	for _, policy := range mutationPolicies {
		if policy.Method != method {
			continue
		}
		patternParts := splitPath(policy.Path)
		if len(patternParts) != len(requestParts) {
			continue
		}
		matched := true
		specificity := 0
		for i, patternPart := range patternParts {
			if strings.HasPrefix(patternPart, ":") {
				if requestParts[i] == "" {
					matched = false
					break
				}
				continue
			}
			specificity++
			if patternPart != requestParts[i] {
				matched = false
				break
			}
		}
		if !matched || specificity < bestSpecificity {
			continue
		}
		if specificity == bestSpecificity && best.Path != "" && best.Path != policy.Path {
			ambiguous = true
			continue
		}
		best = policy
		bestSpecificity = specificity
		ambiguous = false
	}
	if bestSpecificity < 0 || ambiguous {
		return MutationPolicy{}, false
	}
	return best, true
}

// MutationPolicies returns a copy for validation and documentation tooling.
func MutationPolicies() []MutationPolicy {
	result := make([]MutationPolicy, 0, len(mutationPolicies))
	for _, policy := range mutationPolicies {
		result = append(result, policy)
	}
	return result
}

// ValidateRuntimeRoutes proves that the actual router and the reviewed policy
// inventory are identical for mutations. It checks both directions so neither
// an unclassified live route nor a stale policy can be hidden by source scans.
func ValidateRuntimeRoutes(routes []RuntimeRoute) error {
	runtime := make(map[string]struct{})
	var violations []string
	for _, route := range routes {
		if route.Method != "POST" && route.Method != "PUT" && route.Method != "PATCH" && route.Method != "DELETE" {
			continue
		}
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		key := route.Method + ":" + route.Path
		runtime[key] = struct{}{}
		if _, ok := mutationPolicies[key]; !ok {
			violations = append(violations, fmt.Sprintf("runtime mutation lacks explicit policy: %s %s", route.Method, route.Path))
		}
	}
	for key, policy := range mutationPolicies {
		if _, ok := runtime[key]; !ok {
			if _, optional := optionalRuntimePolicies[key]; optional {
				continue
			}
			violations = append(violations, fmt.Sprintf("mutation policy has no runtime route: %s %s", policy.Method, policy.Path))
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		return fmt.Errorf("%d mutation route policy violations:\n%s", len(violations), strings.Join(violations, "\n"))
	}
	return nil
}
