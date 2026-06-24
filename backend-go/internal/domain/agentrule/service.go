package agentrule

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides CRUD and Evaluate operations for personal rules.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new personal rule service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns enabled rules matching the given filters, sorted by priority
// (highest first) then by creation time (newest first).
func (s *Service) List(userID int64, agentID, decisionPoint, ruleType string) ([]PersonalRule, error) {
	q := s.db.Where("user_id = ? AND enabled = ?", userID, true)

	if agentID != "" {
		q = q.Where("agent_id = ?", agentID)
	}
	if decisionPoint != "" {
		q = q.Where("decision_point = ?", decisionPoint)
	}
	if ruleType != "" {
		q = q.Where("rule_type = ?", ruleType)
	}

	var rules []PersonalRule
	if err := q.Order("priority DESC, created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []PersonalRule{}
	}
	return rules, nil
}

// Create inserts a new rule.
func (s *Service) Create(input *CreateRuleInput) (*PersonalRule, error) {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	rule := &PersonalRule{
		UserID:        input.UserID,
		AgentID:       input.AgentID,
		DecisionPoint: input.DecisionPoint,
		RuleType:      input.RuleType,
		Name:          input.Name,
		Conditions:    input.Conditions,
		Effect:        input.Effect,
		Priority:      input.Priority,
		Enabled:       enabled,
		Description:   input.Description,
	}
	if err := s.db.Create(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

// Update modifies an existing rule with non-nil fields from input.
func (s *Service) Update(id int64, input *UpdateRuleInput) (*PersonalRule, error) {
	var rule PersonalRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if input.DecisionPoint != nil {
		updates["decision_point"] = *input.DecisionPoint
	}
	if input.RuleType != nil {
		updates["rule_type"] = *input.RuleType
	}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Conditions != nil {
		updates["conditions"] = *input.Conditions
	}
	if input.Effect != nil {
		updates["effect"] = *input.Effect
	}
	if input.Priority != nil {
		updates["priority"] = *input.Priority
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	updates["updated_at"] = time.Now()

	if err := s.db.Model(&rule).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload the updated record.
	if err := s.db.First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// Delete hard-deletes a rule by ID.
func (s *Service) Delete(id int64) error {
	result := s.db.Delete(&PersonalRule{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("规则不存在")
	}
	return nil
}

// ToggleEnabled flips the enabled flag for a rule.
func (s *Service) ToggleEnabled(id int64) (*PersonalRule, error) {
	var rule PersonalRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return nil, err
	}

	newEnabled := !rule.Enabled
	if err := s.db.Model(&rule).Update("enabled", newEnabled).Error; err != nil {
		return nil, err
	}

	rule.Enabled = newEnabled
	return &rule, nil
}

// Evaluate applies matching rules to the given agent output.
// It returns the modified output and a list of applied rules.
func (s *Service) Evaluate(userID int64, agentID, decisionPoint string, output map[string]interface{}) (*EvaluateResult, error) {
	rules, err := s.List(userID, agentID, decisionPoint, "")
	if err != nil {
		return nil, err
	}

	result := &EvaluateResult{
		Output:       deepCopyMap(output),
		AppliedRules: make([]AppliedRule, 0),
	}

	for _, rule := range rules {
		// Parse condition.
		matched, err := evaluateCondition(result.Output, rule.Conditions)
		if err != nil {
			s.logger.Warn("规则条件解析失败，跳过",
				zap.Int64("rule_id", rule.ID),
				zap.Error(err),
			)
			continue
		}
		if !matched {
			continue
		}

		// Apply the effect based on rule type.
		applied := applyRuleEffect(result.Output, &rule)
		result.AppliedRules = append(result.AppliedRules, *applied)

		// Veto rules that block stop further evaluation.
		if applied.Blocked {
			result.Blocked = true
			result.BlockReason = applied.BlockReason
			break
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Condition evaluation engine
// ---------------------------------------------------------------------------

// evaluateCondition parses and evaluates a JSON condition against the output map.
// The condition JSON format: {"field": "key", "operator": "gt", "value": 0.05}
// Returns true if the condition matches (or no condition is set).
func evaluateCondition(output map[string]interface{}, condJSON json.RawMessage) (bool, error) {
	if len(condJSON) == 0 || string(condJSON) == "{}" || string(condJSON) == "null" {
		return true, nil
	}

	var cond Condition
	if err := json.Unmarshal(condJSON, &cond); err != nil {
		return false, fmt.Errorf("解析条件失败: %w", err)
	}

	// Resolve the field value from output using dot-path.
	actual, found := resolvePath(output, cond.Field)
	if !found {
		return false, nil
	}

	return compareValues(actual, cond.Operator, cond.Value), nil
}

// resolvePath traverses a map using dot-separated path keys.
// "price.margin" -> output["price"]["margin"].
func resolvePath(data map[string]interface{}, path string) (interface{}, bool) {
	path = strings.TrimPrefix(path, "$.")
	if path == "" {
		return data, true
	}
	parts := strings.Split(path, ".")
	current := interface{}(data)
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// toFloat64 attempts to convert a value to float64 for numeric comparison.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case int16:
		return float64(val), true
	case int8:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint64:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint8:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	}
	return 0, false
}

// compareValues compares actual vs target using the given operator.
func compareValues(actual interface{}, op string, target interface{}) bool {
	switch op {
	case "eq":
		return valuesEqual(actual, target)
	case "neq":
		return !valuesEqual(actual, target)
	case "gt":
		a, aOK := toFloat64(actual)
		t, tOK := toFloat64(target)
		return aOK && tOK && a > t
	case "gte":
		a, aOK := toFloat64(actual)
		t, tOK := toFloat64(target)
		return aOK && tOK && a >= t
	case "lt":
		a, aOK := toFloat64(actual)
		t, tOK := toFloat64(target)
		return aOK && tOK && a < t
	case "lte":
		a, aOK := toFloat64(actual)
		t, tOK := toFloat64(target)
		return aOK && tOK && a <= t
	case "in":
		return valuesIn(actual, target)
	case "contains":
		return valuesContains(actual, target)
	}
	return false
}

// valuesEqual does a best-effort equality check handling mixed types.
func valuesEqual(a, b interface{}) bool {
	// Try numeric comparison first.
	aF, aOK := toFloat64(a)
	bF, bOK := toFloat64(b)
	if aOK && bOK {
		return aF == bF
	}
	// Fall back to string comparison.
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// valuesIn checks if actual is one of the items in target (which should be a slice).
func valuesIn(actual interface{}, target interface{}) bool {
	switch t := target.(type) {
	case []interface{}:
		for _, item := range t {
			if valuesEqual(actual, item) {
				return true
			}
		}
	case []string:
		aStr := fmt.Sprintf("%v", actual)
		for _, item := range t {
			if aStr == item {
				return true
			}
		}
	case []float64:
		aF, aOK := toFloat64(actual)
		if !aOK {
			return false
		}
		for _, item := range t {
			if aF == item {
				return true
			}
		}
	case []int64:
		aF, aOK := toFloat64(actual)
		if !aOK {
			return false
		}
		for _, item := range t {
			if aF == float64(item) {
				return true
			}
		}
	case []int:
		aF, aOK := toFloat64(actual)
		if !aOK {
			return false
		}
		for _, item := range t {
			if aF == float64(item) {
				return true
			}
		}
	}
	return false
}

// valuesContains checks if the string representation of actual contains target.
func valuesContains(actual, target interface{}) bool {
	aStr := fmt.Sprintf("%v", actual)
	tStr := fmt.Sprintf("%v", target)
	return strings.Contains(aStr, tStr)
}

// ---------------------------------------------------------------------------
// Effect application
// ---------------------------------------------------------------------------

// applyRuleEffect applies a matched rule's effect to the output map based on
// rule type and returns the AppliedRule record.
func applyRuleEffect(output map[string]interface{}, rule *PersonalRule) *AppliedRule {
	applied := &AppliedRule{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		RuleType: rule.RuleType,
		Matched:  true,
	}

	if len(rule.Effect) == 0 || string(rule.Effect) == "{}" || string(rule.Effect) == "null" {
		return applied
	}

	var effect map[string]interface{}
	if err := json.Unmarshal(rule.Effect, &effect); err != nil {
		return applied
	}

	switch rule.RuleType {
	case "veto":
		applied.Blocked = true
		output["_blocked"] = true
		if reason, ok := effect["reason"].(string); ok {
			output["_block_reason"] = reason
			applied.BlockReason = reason
		}

	case "threshold":
		for k, v := range effect {
			output[k] = v
		}
		applied.Action = "threshold_adjusted"

	case "strategy":
		for k, v := range effect {
			output[k] = v
		}
		applied.Action = "strategy_overridden"

	case "style":
		// Merge style parameters into output["_style"].
		existing, _ := output["_style"].(map[string]interface{})
		if existing == nil {
			existing = make(map[string]interface{})
		}
		for k, v := range effect {
			existing[k] = v
		}
		output["_style"] = existing
		applied.Action = "style_modified"
	}

	return applied
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

// deepCopyMap creates a deep copy of a map[string]interface{}.
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	data, err := json.Marshal(src)
	if err != nil {
		return src
	}
	var dst map[string]interface{}
	if err := json.Unmarshal(data, &dst); err != nil {
		return src
	}
	return dst
}
