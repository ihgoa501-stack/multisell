package personalrule

import (
	"fmt"
	"strings"
)

// resolvePath traverses a map using dot-separated path keys.
// "$.foo.bar" → output["foo"]["bar"].
// Leading "$." is stripped if present.
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
	}
	return 0, false
}

// matchesCondition evaluates a single condition against the agent output.
// It resolves the field path from output, then applies the operator.
func matchesCondition(output map[string]interface{}, c *Condition) bool {
	if c == nil {
		return true
	}

	// Resolve the field value from output using the path.
	actual, ok := resolvePath(output, c.Field)
	if !ok {
		return false
	}

	return compareValues(actual, c.Op, c.Value)
}

// compareValues compares actual vs target using the given operator.
func compareValues(actual interface{}, op string, target interface{}) bool {
	switch op {
	case "eq":
		return valuesEqual(actual, target)
	case "neq":
		return !valuesEqual(actual, target)
	case "gt":
		a, aOk := toFloat64(actual)
		t, tOk := toFloat64(target)
		return aOk && tOk && a > t
	case "gte":
		a, aOk := toFloat64(actual)
		t, tOk := toFloat64(target)
		return aOk && tOk && a >= t
	case "lt":
		a, aOk := toFloat64(actual)
		t, tOk := toFloat64(target)
		return aOk && tOk && a < t
	case "lte":
		a, aOk := toFloat64(actual)
		t, tOk := toFloat64(target)
		return aOk && tOk && a <= t
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
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if aOk && bOk {
		return af == bf
	}

	// Fall back to string comparison.
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr == bStr
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
		af, aOk := toFloat64(actual)
		if !aOk {
			return false
		}
		for _, item := range t {
			if af == item {
				return true
			}
		}
	case []int:
		af, aOk := toFloat64(actual)
		if !aOk {
			return false
		}
		for _, item := range t {
			if af == float64(item) {
				return true
			}
		}
	case []int64:
		af, aOk := toFloat64(actual)
		if !aOk {
			return false
		}
		for _, item := range t {
			if af == float64(item) {
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

// applyAction applies a matched rule's action to the output map.
// It returns the action result including whether the decision was blocked,
// and any notifications that should be attached.
//
// Supported action types:
//   - override: config contains key-value pairs to set in output
//   - modifier: config contains "field", "operation" (add/sub/mul/div), "value"
//   - block: marks the decision as blocked (no config needed)
//   - notify: config contains "message", "severity"
func applyAction(output map[string]interface{}, actionType string, actionConf map[string]interface{}) *RuleResult {
	r := &RuleResult{
		Matched: true,
	}

	switch actionType {
	case "override":
		for k, v := range actionConf {
			output[k] = v
		}

	case "modifier":
		field, _ := actionConf["field"].(string)
		if field == "" {
			break
		}
		operation, _ := actionConf["operation"].(string)
		rawVal, hasVal := actionConf["value"]
		if !hasVal {
			break
		}
		modVal, ok := toFloat64(rawVal)
		if !ok {
			break
		}

		// Resolve the field to modify.
		rawCurrent, found := resolvePath(output, field)
		if !found {
			break
		}
		currentVal, ok := toFloat64(rawCurrent)
		if !ok {
			break
		}

		var newVal float64
		switch operation {
		case "add":
			newVal = currentVal + modVal
		case "sub":
			newVal = currentVal - modVal
		case "mul":
			newVal = currentVal * modVal
		case "div":
			if modVal == 0 {
				break
			}
			newVal = currentVal / modVal
		case "percentage":
			newVal = currentVal * (1 + modVal/100)
		default:
			break
		}

		// Write the modified value back using the path.
		setPath(output, field, newVal)

	case "block":
		r.Blocked = true
		output["_blocked"] = true
		if reason, ok := actionConf["reason"].(string); ok {
			output["_block_reason"] = reason
		}

	case "notify":
		msg, _ := actionConf["message"].(string)
		severity, _ := actionConf["severity"].(string)
		if severity == "" {
			severity = "info"
		}
		notification := map[string]interface{}{
			"message":  msg,
			"severity": severity,
		}
		// Add to rule result.
		r.Notifications = append(r.Notifications, notification)

		// Also attach to output's alerts array.
		alerts, _ := output["alerts"].([]interface{})
		alerts = append(alerts, notification)
		output["alerts"] = alerts
	}

	return r
}

// setPath sets a value at a dotted path within a nested map.
// Creates intermediate maps as needed.
func setPath(data map[string]interface{}, path string, value interface{}) {
	path = strings.TrimPrefix(path, "$.")
	if path == "" {
		return
	}
	parts := strings.Split(path, ".")
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		next, ok := current[part]
		if !ok {
			nested := make(map[string]interface{})
			current[part] = nested
			current = nested
		} else if m, ok := next.(map[string]interface{}); ok {
			current = m
		} else {
			nested := make(map[string]interface{})
			current[part] = nested
			current = nested
		}
	}
}
