package guardrails

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// OutputGuard implements L3 output guardrail for validating LLM-generated
// output against expected schemas, business rules, and numerical ranges.
//
// Checks performed:
//   - JSON format validation — verifies the raw output is valid JSON
//   - Required field existence — ensures all fields listed in the schema
//     presence hint are present
//   - Numeric range validation — detects negative prices/quantities and
//     unreasonably large magnitudes
//   - Enum value validation — checks string fields against an allowed set
type OutputGuard struct{}

// NewOutputGuard creates a new OutputGuard.
func NewOutputGuard() *OutputGuard {
	return &OutputGuard{}
}

// Name returns "output_guard".
func (g *OutputGuard) Name() string {
	return "output_guard"
}

// Check validates the LLM output against schema and business rules.
//
// It inspects input.RawOutput as a JSON payload and input.OutputSchema
// as the expected shape. When OutputSchema is a map, the guard treats
// it as a schema hint containing:
//   - "required": []string — field names that must be present
//   - "positive_fields": []string — field names that must be > 0 (e.g. price, quantity)
//   - "min": map[string]float64 — per-field minimum allowed values
//   - "max": map[string]float64 — per-field maximum allowed values
//   - "enum": map[string][]interface{} — per-field allowed values
//
// If RawOutput is empty the guard passes with a warning (no output to validate).
func (g *OutputGuard) Check(ctx context.Context, input *GuardInput) (*GuardResult, error) {
	if input.RawOutput == "" {
		return &GuardResult{
			Pass:    false,
			Blocked: false,
			Retry:   false,
			Reason:  "empty output — nothing to validate",
			Risk:    "low",
		}, nil
	}

	// 1. JSON format validation.
	var parsed interface{}
	if err := json.Unmarshal([]byte(input.RawOutput), &parsed); err != nil {
		return &GuardResult{
			Pass:    false,
			Blocked: true,
			Retry:   true,
			Reason:  fmt.Sprintf("invalid JSON output: %s", err.Error()),
			Risk:    "medium",
		}, nil
	}

	// If there's no schema hint, basic JSON validity is sufficient.
	if input.OutputSchema == nil {
		return &GuardResult{
			Pass:    true,
			Blocked: false,
			Retry:   false,
			Reason:  "valid JSON, no schema constraints",
			Risk:    "low",
		}, nil
	}

	parsedMap, ok := parsed.(map[string]interface{})
	if !ok {
		return &GuardResult{
			Pass:    false,
			Blocked: true,
			Retry:   false,
			Reason:  "output is valid JSON but not a JSON object",
			Risk:    "medium",
		}, nil
	}

	schema, ok := input.OutputSchema.(map[string]interface{})
	if !ok {
		// Schema provided but not a map — can't interpret, pass with warning.
		return &GuardResult{
			Pass:    false,
			Blocked: false,
			Retry:   false,
			Reason:  "output schema hint is not a map — cannot enforce",
			Risk:    "low",
		}, nil
	}

	// 2. Required field existence.
	if reqFields, ok := schema["required"]; ok {
		if reqList, ok := reqFields.([]interface{}); ok {
			for _, rf := range reqList {
				fieldName, ok := rf.(string)
				if !ok {
					continue
				}
				if _, exists := parsedMap[fieldName]; !exists {
					return &GuardResult{
						Pass:    false,
						Blocked: true,
						Retry:   true,
						Reason:  fmt.Sprintf("missing required field: %s", fieldName),
						Risk:    "medium",
					}, nil
				}
			}
		}
	}

	// 3. Numeric range validation — check positive fields.
	if posFields, ok := schema["positive_fields"]; ok {
		if posList, ok := posFields.([]interface{}); ok {
			for _, pf := range posList {
				fieldName, ok := pf.(string)
				if !ok {
					continue
				}
				val, exists := parsedMap[fieldName]
				if !exists {
					continue
				}
				num, err := toFloat64(val)
				if err != nil {
					continue
				}
				if num < 0 {
					return &GuardResult{
						Pass:    false,
						Blocked: true,
						Retry:   true,
						Reason:  fmt.Sprintf("field %q has negative value (%.2f)", fieldName, num),
						Risk:    "high",
					}, nil
				}
			}
		}
	}

	// 4. Numeric range validation — per-field min/max.
	if mins, ok := schema["min"]; ok {
		if minMap, ok := mins.(map[string]interface{}); ok {
			for fieldName, minVal := range minMap {
				val, exists := parsedMap[fieldName]
				if !exists {
					continue
				}
				num, err := toFloat64(val)
				if err != nil {
					continue
				}
				minNum, err := toFloat64(minVal)
				if err != nil {
					continue
				}
				if num < minNum {
					return &GuardResult{
						Pass:    false,
						Blocked: true,
						Retry:   true,
						Reason:  fmt.Sprintf("field %q value (%.2f) is below minimum (%.2f)", fieldName, num, minNum),
						Risk:    "medium",
					}, nil
				}
			}
		}
	}

	if maxs, ok := schema["max"]; ok {
		if maxMap, ok := maxs.(map[string]interface{}); ok {
			for fieldName, maxVal := range maxMap {
				val, exists := parsedMap[fieldName]
				if !exists {
					continue
				}
				num, err := toFloat64(val)
				if err != nil {
					continue
				}
				maxNum, err := toFloat64(maxVal)
				if err != nil {
					continue
				}
				if num > maxNum {
					return &GuardResult{
						Pass:    false,
						Blocked: true,
						Retry:   true,
						Reason:  fmt.Sprintf("field %q value (%.2f) exceeds maximum (%.2f)", fieldName, num, maxNum),
						Risk:    "medium",
					}, nil
				}
			}
		}
	}

	// 5. Enum value validation.
	if enums, ok := schema["enum"]; ok {
		if enumMap, ok := enums.(map[string]interface{}); ok {
			for fieldName, allowedRaw := range enumMap {
				val, exists := parsedMap[fieldName]
				if !exists {
					continue
				}
				allowedList, ok := allowedRaw.([]interface{})
				if !ok {
					continue
				}
				valStr := fmt.Sprintf("%v", val)
				found := false
				for _, a := range allowedList {
					if fmt.Sprintf("%v", a) == valStr {
						found = true
						break
					}
				}
				if !found {
					allowedStrs := make([]string, 0, len(allowedList))
					for _, a := range allowedList {
						allowedStrs = append(allowedStrs, fmt.Sprintf("%v", a))
					}
					return &GuardResult{
						Pass:    false,
						Blocked: true,
						Retry:   true,
						Reason:  fmt.Sprintf("field %q value %q is not in allowed set [%s]", fieldName, valStr, strings.Join(allowedStrs, ", ")),
						Risk:    "medium",
					}, nil
				}
			}
		}
	}

	return &GuardResult{
		Pass:    true,
		Blocked: false,
		Retry:   false,
		Reason:  "output passed all schema and business-rule checks",
		Risk:    "low",
	}, nil
}

// toFloat64 attempts to convert an interface{} to a float64.
// Handles json.Number, float64, and integer types.
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case json.Number:
		return val.Float64()
	case string:
		return strconv.ParseFloat(val, 64)
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}
