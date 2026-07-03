package guardrails

import (
	"encoding/json"
	"context"
	"fmt"
	"math"
	"sync"
)

// ExecutionRule defines a single threshold check in the execution guard.
// Rules are evaluated against the action's parameters extracted from
// GuardInput.ToolInput.
type ExecutionRule struct {
	// Name is a human-readable identifier (e.g. "purchase_amount_limit").
	Name string

	// MaxAmount is the maximum allowed monetary value. If the action's
	// "amount" parameter exceeds this threshold, the rule triggers.
	MaxAmount float64

	// MaxQuantity is the maximum allowed item count. If the action's
	// "quantity" parameter exceeds this threshold, the rule triggers.
	MaxQuantity int

	// RiskLevels specifies which action risk levels this rule applies to.
	// An empty slice means the rule applies to all risk levels.
	RiskLevels []string

	// ActionTypes specifies which action types this rule applies to
	// (e.g. "purchase", "replenish", "refund", "discount", "listing").
	// An empty slice means the rule applies to all action types.
	ActionTypes []string

	// RequireApproval, when true, signals that exceeding this threshold
	// requires human approval (the check returns a warning, not a block).
	RequireApproval bool
}

// ExecutionGuard implements L4 execution guardrail for checking financial
// thresholds, quantity limits, and risk-based approval requirements before
// an action is carried out.
//
// The guard evaluates ToolInput parameters:
//   - "amount" (float or numeric string) for monetary thresholds
//   - "quantity" (int or numeric string) for quantity thresholds
//   - "action_type" (string) to match against rule ActionTypes
//   - "risk_level" (string) to match against rule RiskLevels
type ExecutionGuard struct {
	mu    sync.RWMutex
	rules []ExecutionRule
}

// NewExecutionGuard creates an execution guard with the built-in rule set.
//
// Built-in rules:
//   - Purchase amount > 100000 → require_approval
//   - Replenish quantity > 10000 → block + require_approval
//   - Refund amount > 5000 → require_approval
//   - Discount rate > 50% → block
//   - New listing count > 100 → require_approval
func NewExecutionGuard() *ExecutionGuard {
	return &ExecutionGuard{
		rules: []ExecutionRule{
			{
				Name:            "purchase_amount_limit",
				MaxAmount:       100000,
				ActionTypes:     []string{"purchase"},
				RequireApproval: true,
			},
			{
				Name:        "replenish_quantity_limit",
				MaxQuantity:  10000,
				ActionTypes: []string{"replenish"},
			},
			{
				Name:            "refund_amount_limit",
				MaxAmount:       5000,
				ActionTypes:     []string{"refund"},
				RequireApproval: true,
			},
			{
				Name:        "discount_rate_limit",
				MaxAmount:    50, // 50% max discount
				ActionTypes: []string{"discount"},
			},
			{
				Name:            "listing_count_limit",
				MaxQuantity:     100,
				ActionTypes:     []string{"listing"},
				RequireApproval: true,
			},
		},
	}
}

// NewExecutionGuardWithRules creates an execution guard with a custom rule set.
func NewExecutionGuardWithRules(rules []ExecutionRule) *ExecutionGuard {
	if rules == nil {
		rules = []ExecutionRule{}
	}
	return &ExecutionGuard{
		rules: rules,
	}
}

// SetRules replaces the current rule set. Thread-safe.
func (g *ExecutionGuard) SetRules(rules []ExecutionRule) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if rules == nil {
		g.rules = []ExecutionRule{}
		return
	}
	g.rules = rules
}

// Rules returns a copy of the current rules. Thread-safe.
func (g *ExecutionGuard) Rules() []ExecutionRule {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]ExecutionRule, len(g.rules))
	copy(out, g.rules)
	return out
}

// SyncFromActionPolicy converts a list of action-policy-like rules to
// ExecutionRule format and replaces the current rule set.
// Each rule is a map with optional keys: "name"(string), "max_amount"(float64),
// "max_quantity"(int), "action_types"([]string), "risk_levels"([]string),
// "require_approval"(bool).
func (g *ExecutionGuard) SyncFromActionPolicy(rules []map[string]interface{}) {
	converted := make([]ExecutionRule, 0, len(rules))
	for _, r := range rules {
		er := ExecutionRule{}
		if v, ok := r["name"].(string); ok {
			er.Name = v
		}
		if v, ok := r["max_amount"].(float64); ok {
			er.MaxAmount = v
		}
		if v, ok := r["max_quantity"].(int); ok {
			er.MaxQuantity = v
		}
		if v, ok := r["require_approval"].(bool); ok {
			er.RequireApproval = v
		}
		if v, ok := r["action_types"].([]string); ok {
			er.ActionTypes = v
		}
		if v, ok := r["risk_levels"].([]string); ok {
			er.RiskLevels = v
		}
		converted = append(converted, er)
	}
	g.SetRules(converted)
}

// Name returns "execution_guard".
func (g *ExecutionGuard) Name() string {
	return "execution_guard"
}

// Check evaluates all rules against the action parameters in ToolInput.
func (g *ExecutionGuard) Check(ctx context.Context, input *GuardInput) (*GuardResult, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if input.ToolInput == nil || len(input.ToolInput) == 0 {
		return &GuardResult{
			Pass:    true,
			Blocked: false,
			Retry:   false,
			Reason:  "no action parameters to check",
			Risk:    "low",
		}, nil
	}

	// Extract action type and risk level from ToolInput.
	actionType, _ := input.ToolInput["action_type"].(string)
	riskLevel, _ := input.ToolInput["risk_level"].(string)

	// Extract amount (monetary value) from ToolInput.
	// Supports "amount" field as float64, json.Number, int, or string.
	amount := 0.0
	if rawAmount, exists := input.ToolInput["amount"]; exists {
		switch v := rawAmount.(type) {
		case float64:
			amount = v
		case int:
			amount = float64(v)
		case int64:
			amount = float64(v)
		case json.Number:
			if f, err := v.Float64(); err == nil {
				amount = f
			}
		case string:
			if f, err := parseFloat64(v); err == nil {
				amount = f
			}
		}
	}

	// Extract quantity from ToolInput.
	// Supports "quantity" field as int, float64, json.Number, or string.
	quantity := 0
	if rawQty, exists := input.ToolInput["quantity"]; exists {
		switch v := rawQty.(type) {
		case float64:
			quantity = int(math.Round(v))
		case int:
			quantity = v
		case int64:
			quantity = int(v)
		case json.Number:
			if f, err := v.Float64(); err == nil {
				quantity = int(math.Round(f))
			}
		case string:
			if f, err := parseFloat64(v); err == nil {
				quantity = int(math.Round(f))
			}
		}
	}

	// Evaluate each rule.
	for _, rule := range g.rules {
		// Check action type filter.
		if len(rule.ActionTypes) > 0 {
			if !containsString(rule.ActionTypes, actionType) {
				continue
			}
		}

		// Check risk level filter.
		if len(rule.RiskLevels) > 0 {
			if !containsString(rule.RiskLevels, riskLevel) {
				continue
			}
		}

		// Check amount threshold.
		amountExceeded := rule.MaxAmount > 0 && amount > rule.MaxAmount
		// Check quantity threshold.
		quantityExceeded := rule.MaxQuantity > 0 && quantity > rule.MaxQuantity

		if !amountExceeded && !quantityExceeded {
			continue
		}

		// Determine the reason.
		var reason string
		if amountExceeded && quantityExceeded {
			reason = fmt.Sprintf("rule %q: amount (%.2f) exceeds max (%.2f) and quantity (%d) exceeds max (%d)",
				rule.Name, amount, rule.MaxAmount, quantity, rule.MaxQuantity)
		} else if amountExceeded {
			reason = fmt.Sprintf("rule %q: amount (%.2f) exceeds max (%.2f)",
				rule.Name, amount, rule.MaxAmount)
		} else {
			reason = fmt.Sprintf("rule %q: quantity (%d) exceeds max (%d)",
				rule.Name, quantity, rule.MaxQuantity)
		}

		if rule.RequireApproval {
			// Require approval — warn but don't block.
			return &GuardResult{
				Pass:    false,
				Blocked: false,
				Retry:   false,
				Reason:  reason + " — requires human approval",
				Risk:    "medium",
			}, nil
		}

		// Block the action outright.
		return &GuardResult{
			Pass:    false,
			Blocked: true,
			Retry:   false,
			Reason:  reason + " — blocked",
			Risk:    "high",
		}, nil
	}

	return &GuardResult{
		Pass:    true,
		Blocked: false,
		Retry:   false,
		Reason:  "all execution rules passed",
		Risk:    "low",
	}, nil
}

// parseFloat64 parses a string as a float64, handling common format variations.
func parseFloat64(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// containsString checks if a slice contains a string value.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
