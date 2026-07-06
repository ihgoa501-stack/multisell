package agentrule

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// =============================================================================
// Path resolution
// =============================================================================

func TestResolvePath_DotNotation(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "deep",
			},
		},
		"x": 42,
	}

	v, found := resolvePath(data, "a.b.c")
	if !found || v != "deep" {
		t.Errorf("a.b.c: got (%v, %v), want (deep, true)", v, found)
	}

	v, found = resolvePath(data, "x")
	if !found || v != 42 {
		t.Errorf("x: got (%v, %v), want (42, true)", v, found)
	}

	// $ prefix is stripped
	v, found = resolvePath(data, "$.x")
	if !found || v != 42 {
		t.Errorf("$.x: got (%v, %v), want (42, true)", v, found)
	}

	// Empty path returns root
	_, found = resolvePath(data, "")
	if !found {
		t.Error("empty path should return root")
	}
}

func TestResolvePath_MissingKey(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"a": 1}

	_, found := resolvePath(data, "b")
	if found {
		t.Error("expected not found for missing key")
	}

	_, found = resolvePath(data, "a.b.c")
	if found {
		t.Error("expected not found for nested missing key")
	}

	_, found = resolvePath(data, "x.y")
	if found {
		t.Error("expected not found for missing root key")
	}
}

// =============================================================================
// Numeric conversion
// =============================================================================

func TestToFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input interface{}
		want  float64
		ok    bool
	}{
		{"float64", float64(3.14), 3.14, true},
		{"int", 100, 100.0, true},
		{"int64", int64(100), 100.0, true},
		{"int32", int32(100), 100.0, true},
		{"int8", int8(10), 10.0, true},
		{"uint", uint(100), 100.0, true},
		{"uint64", uint64(100), 100.0, true},
		{"string", "abc", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
		{"json.Number", json.Number("3.14"), 3.14, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Comparison operators
// =============================================================================

func TestCompareValues(t *testing.T) {
	t.Parallel()

	if !compareValues(float64(100), "eq", float64(100)) {
		t.Error("eq: 100 == 100 should be true")
	}
	if compareValues(float64(99), "eq", float64(100)) {
		t.Error("eq: 99 == 100 should be false")
	}
	if !compareValues(float64(99), "neq", float64(100)) {
		t.Error("neq: 99 != 100 should be true")
	}
	if compareValues(float64(100), "neq", float64(100)) {
		t.Error("neq: 100 != 100 should be false")
	}
	if !compareValues(float64(101), "gt", float64(100)) {
		t.Error("gt: 101 > 100 should be true")
	}
	if compareValues(float64(100), "gt", float64(100)) {
		t.Error("gt: 100 > 100 should be false")
	}
	if !compareValues(float64(100), "gte", float64(100)) {
		t.Error("gte: 100 >= 100 should be true")
	}
	if compareValues(float64(99), "gte", float64(100)) {
		t.Error("gte: 99 >= 100 should be false")
	}
	if !compareValues(float64(99), "lt", float64(100)) {
		t.Error("lt: 99 < 100 should be true")
	}
	if compareValues(float64(100), "lt", float64(100)) {
		t.Error("lt: 100 < 100 should be false")
	}
	if !compareValues(float64(100), "lte", float64(100)) {
		t.Error("lte: 100 <= 100 should be true")
	}
	if compareValues(float64(101), "lte", float64(100)) {
		t.Error("lte: 101 <= 100 should be false")
	}
	if !compareValues("a", "in", []interface{}{"a", "b", "c"}) {
		t.Error("in: 'a' in ['a','b','c'] should be true")
	}
	if compareValues("d", "in", []interface{}{"a", "b", "c"}) {
		t.Error("in: 'd' in ['a','b','c'] should be false")
	}
	if !compareValues("hello world", "contains", "world") {
		t.Error("contains: 'hello world' contains 'world' should be true")
	}
	if compareValues("hello world", "contains", "xyz") {
		t.Error("contains: 'hello world' contains 'xyz' should be false")
	}
	if compareValues(float64(100), "unknown", float64(100)) {
		t.Error("unknown operator should return false")
	}
}

// =============================================================================
// Condition evaluation
// =============================================================================

func TestEvaluateCondition_Equals(t *testing.T) {
	t.Parallel()

	matched, err := evaluateCondition(
		map[string]interface{}{"price": float64(100)},
		json.RawMessage(`{"field":"price","operator":"eq","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("expected match for price=100")
	}

	matched, err = evaluateCondition(
		map[string]interface{}{"price": float64(99)},
		json.RawMessage(`{"field":"price","operator":"eq","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Error("expected no match for price=99")
	}

	// Missing field
	matched, err = evaluateCondition(
		map[string]interface{}{"other": 1},
		json.RawMessage(`{"field":"price","operator":"eq","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Error("expected no match for missing field")
	}

	// Empty condition always matches
	matched, err = evaluateCondition(
		map[string]interface{}{"price": float64(100)},
		json.RawMessage(`{}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("empty condition should always match")
	}

	// Null condition always matches
	matched, err = evaluateCondition(
		map[string]interface{}{"price": float64(100)},
		json.RawMessage(`null`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("null condition should always match")
	}

	// Empty JSON always matches
	matched, err = evaluateCondition(
		map[string]interface{}{"price": float64(100)},
		json.RawMessage{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("empty condJSON should always match")
	}
}

func TestEvaluateCondition_GreaterThan(t *testing.T) {
	t.Parallel()

	matched, err := evaluateCondition(
		map[string]interface{}{"price": float64(101)},
		json.RawMessage(`{"field":"price","operator":"gt","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("expected match for 101 > 100")
	}

	matched, err = evaluateCondition(
		map[string]interface{}{"price": float64(100)},
		json.RawMessage(`{"field":"price","operator":"gt","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Error("expected no match for 100 > 100")
	}

	matched, err = evaluateCondition(
		map[string]interface{}{"price": float64(99)},
		json.RawMessage(`{"field":"price","operator":"gt","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Error("expected no match for 99 > 100")
	}
}

func TestEvaluateCondition_LessThan(t *testing.T) {
	t.Parallel()

	matched, err := evaluateCondition(
		map[string]interface{}{"price": float64(99)},
		json.RawMessage(`{"field":"price","operator":"lt","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("expected match for 99 < 100")
	}

	matched, err = evaluateCondition(
		map[string]interface{}{"price": float64(100)},
		json.RawMessage(`{"field":"price","operator":"lt","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Error("expected no match for 100 < 100")
	}
}

func TestEvaluateCondition_Contains(t *testing.T) {
	t.Parallel()

	// String contains
	matched, err := evaluateCondition(
		map[string]interface{}{"title": "big sale today"},
		json.RawMessage(`{"field":"title","operator":"contains","value":"sale"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("expected match for title contains 'sale'")
	}

	// No match
	matched, err = evaluateCondition(
		map[string]interface{}{"title": "no discount"},
		json.RawMessage(`{"field":"title","operator":"contains","value":"sale"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Error("expected no match for title not containing 'sale'")
	}
}

func TestEvaluateCondition_In(t *testing.T) {
	t.Parallel()

	matched, err := evaluateCondition(
		map[string]interface{}{"status": "active"},
		json.RawMessage(`{"field":"status","operator":"in","value":["active","pending"]}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("expected match for status='active' in ['active','pending']")
	}

	matched, err = evaluateCondition(
		map[string]interface{}{"status": "inactive"},
		json.RawMessage(`{"field":"status","operator":"in","value":["active","pending"]}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Error("expected no match for status='inactive'")
	}
}

func TestEvaluateCondition_MultipleFields(t *testing.T) {
	t.Parallel()

	// Multiple top-level fields, condition checks one
	output := map[string]interface{}{
		"name":   "product",
		"price":  float64(100),
		"status": "active",
	}

	matched, err := evaluateCondition(
		output,
		json.RawMessage(`{"field":"price","operator":"gt","value":50}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("expected match for price>50 among multiple fields")
	}

	// Another field from the same output
	matched, err = evaluateCondition(
		output,
		json.RawMessage(`{"field":"status","operator":"eq","value":"active"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("expected match for status='active'")
	}

	// Nested field via dot-path among multiple top-level branches
	nested := map[string]interface{}{
		"product": map[string]interface{}{
			"price": float64(100),
		},
		"shipping": map[string]interface{}{
			"cost": float64(10),
		},
	}
	matched, err = evaluateCondition(
		nested,
		json.RawMessage(`{"field":"product.price","operator":"eq","value":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Error("expected match for product.price=100 via dot-path")
	}
}

// =============================================================================
// Effect application
// =============================================================================

func TestApplyRuleEffect(t *testing.T) {
	t.Parallel()

	t.Run("empty effect", func(t *testing.T) {
		output := map[string]interface{}{"price": float64(100)}
		rule := &PersonalRule{ID: 1, Name: "noop", RuleType: "threshold"}
		result := applyRuleEffect(output, rule)
		if result.Action != "" {
			t.Errorf("expected empty action, got %s", result.Action)
		}
		if !result.Matched {
			t.Error("expected matched=true")
		}
	})

	t.Run("null effect", func(t *testing.T) {
		output := map[string]interface{}{"price": float64(100)}
		rule := &PersonalRule{
			ID: 1, Name: "null-effect", RuleType: "threshold",
			Effect: json.RawMessage(`null`),
		}
		result := applyRuleEffect(output, rule)
		if result.Action != "" {
			t.Errorf("expected empty action for null effect, got %s", result.Action)
		}
	})

	t.Run("veto", func(t *testing.T) {
		output := map[string]interface{}{"discount": float64(60)}
		rule := &PersonalRule{
			ID: 1, Name: "veto-rule", RuleType: "veto",
			Effect: json.RawMessage(`{"reason":"too high"}`),
		}
		result := applyRuleEffect(output, rule)
		if !result.Blocked {
			t.Error("expected blocked=true")
		}
		if result.BlockReason != "too high" {
			t.Errorf("expected block_reason='too high', got %q", result.BlockReason)
		}
		if v, _ := output["_blocked"].(bool); !v {
			t.Error("expected _blocked in output")
		}
		if v, _ := output["_block_reason"].(string); v != "too high" {
			t.Errorf("expected _block_reason='too high', got %q", v)
		}
	})

	t.Run("veto no reason field", func(t *testing.T) {
		output := map[string]interface{}{"discount": float64(60)}
		rule := &PersonalRule{
			ID: 1, Name: "veto-no-reason", RuleType: "veto",
			Effect: json.RawMessage(`{"code":"OVER_LIMIT"}`),
		}
		result := applyRuleEffect(output, rule)
		if !result.Blocked {
			t.Error("expected blocked=true")
		}
		if output["_block_reason"] != nil {
			t.Errorf("unexpected _block_reason: %v", output["_block_reason"])
		}
	})

	t.Run("threshold", func(t *testing.T) {
		output := map[string]interface{}{"price": float64(150)}
		rule := &PersonalRule{
			ID: 1, Name: "low-price", RuleType: "threshold",
			Effect: json.RawMessage(`{"price":99}`),
		}
		result := applyRuleEffect(output, rule)
		if result.Action != "threshold_adjusted" {
			t.Errorf("expected threshold_adjusted, got %s", result.Action)
		}
		if output["price"] != float64(99) {
			t.Errorf("expected price=99, got %v", output["price"])
		}
	})

	t.Run("strategy", func(t *testing.T) {
		output := map[string]interface{}{"strategy": "profit"}
		rule := &PersonalRule{
			ID: 1, Name: "volume-strategy", RuleType: "strategy",
			Effect: json.RawMessage(`{"strategy":"volume"}`),
		}
		result := applyRuleEffect(output, rule)
		if result.Action != "strategy_overridden" {
			t.Errorf("expected strategy_overridden, got %s", result.Action)
		}
		if output["strategy"] != "volume" {
			t.Errorf("expected strategy=volume, got %v", output["strategy"])
		}
	})

	t.Run("style with existing", func(t *testing.T) {
		output := map[string]interface{}{
			"_style": map[string]interface{}{"color": "red"},
		}
		rule := &PersonalRule{
			ID: 1, Name: "style-rule", RuleType: "style",
			Effect: json.RawMessage(`{"fontSize":14}`),
		}
		result := applyRuleEffect(output, rule)
		if result.Action != "style_modified" {
			t.Errorf("expected style_modified, got %s", result.Action)
		}
		style, _ := output["_style"].(map[string]interface{})
		if style == nil {
			t.Fatal("expected _style in output")
		}
		if style["fontSize"] != float64(14) {
			t.Errorf("expected fontSize=14, got %v", style["fontSize"])
		}
		if style["color"] != "red" {
			t.Errorf("expected color=red, got %v", style["color"])
		}
	})

	t.Run("style without existing", func(t *testing.T) {
		output := map[string]interface{}{}
		rule := &PersonalRule{
			ID: 1, Name: "style-new", RuleType: "style",
			Effect: json.RawMessage(`{"fontSize":14}`),
		}
		applyRuleEffect(output, rule)
		style, _ := output["_style"].(map[string]interface{})
		if style == nil {
			t.Fatal("expected _style to be created")
		}
		if style["fontSize"] != float64(14) {
			t.Errorf("expected fontSize=14, got %v", style["fontSize"])
		}
	})
}

// =============================================================================
// DB-backed service tests
// =============================================================================

func TestService_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create
	rule, err := svc.Create(&CreateRuleInput{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		RuleType:      "threshold",
		Name:          "低库存阈值",
		Priority:      100,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rule.ID == 0 {
		t.Fatal("ID should be set")
	}
	if rule.Name != "低库存阈值" {
		t.Fatalf("Name = %s", rule.Name)
	}
	if !rule.Enabled {
		t.Fatal("Enabled should default to true")
	}

	// List
	rules, err := svc.List(1, "A5", "stock_alert", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("rules = %d (expected 1)", len(rules))
	}

	// Update name
	updated, err := svc.Update(rule.ID, &UpdateRuleInput{
		Name: dbtest.StringPtr("低库存阈值-调整"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "低库存阈值-调整" {
		t.Fatalf("Name = %s", updated.Name)
	}

	// ToggleEnabled
	toggled, err := svc.ToggleEnabled(rule.ID)
	if err != nil {
		t.Fatalf("ToggleEnabled: %v", err)
	}
	if toggled.Enabled {
		t.Fatal("Expected disabled after toggle")
	}

	// Delete
	if err := svc.Delete(rule.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	rulesAfter, _ := svc.List(1, "A5", "stock_alert", "")
	if len(rulesAfter) != 0 {
		t.Fatalf("rules after delete = %d", len(rulesAfter))
	}
}

func TestService_Evaluate_NoMatch(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		UserID: 1, AgentID: "A5", DecisionPoint: "stock_alert",
		RuleType: "threshold", Name: "test",
		Conditions: []byte(`{"field":"price","operator":"gt","value":100}`),
		Effect:     []byte(`{"price":99}`),
	})

	result, err := svc.Evaluate(1, "A5", "stock_alert", map[string]interface{}{
		"price": float64(50),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result.AppliedRules) != 0 {
		t.Fatalf("expected 0 applied rules, got %d", len(result.AppliedRules))
	}
	if result.Blocked {
		t.Fatal("should not be blocked")
	}
	// Output should be a deep copy with original value
	if v, _ := result.Output["price"].(float64); v != float64(50) {
		t.Fatalf("expected price=50, got %v", result.Output["price"])
	}
}

func TestService_Evaluate(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		UserID: 1, AgentID: "A5", DecisionPoint: "stock_alert",
		RuleType: "threshold", Name: "调整价格",
		Conditions: []byte(`{"field":"price","operator":"gt","value":100}`),
		Effect:     []byte(`{"price":99}`),
	})

	result, err := svc.Evaluate(1, "A5", "stock_alert", map[string]interface{}{
		"price": float64(150),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result.AppliedRules) != 1 {
		t.Fatalf("applied rules = %d", len(result.AppliedRules))
	}
	if result.Blocked {
		t.Fatal("should not be blocked")
	}
	// Output should contain the threshold-adjusted price
	if result.Output["price"] != float64(99) {
		t.Fatalf("price = %v", result.Output["price"])
	}
}

func TestService_Evaluate_Veto(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		UserID: 1, AgentID: "G3", DecisionPoint: "discount_risk_check",
		RuleType: "veto", Name: "阻止超高折扣",
		Conditions: []byte(`{"field":"discount_pct","operator":"gt","value":50}`),
		Effect:     []byte(`{"reason":"折扣超过50%，需要人工审批"}`),
	})

	result, err := svc.Evaluate(1, "G3", "discount_risk_check", map[string]interface{}{
		"discount_pct": float64(60),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !result.Blocked {
		t.Fatal("expected blocked")
	}
	if result.BlockReason != "折扣超过50%，需要人工审批" {
		t.Fatalf("BlockReason = %s", result.BlockReason)
	}
}
