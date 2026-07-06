package personalrule

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create
	rule := &PersonalRule{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		RuleType:      "threshold",
		RuleName:      "低库存阈值",
		RuleCondition: json.RawMessage(`{"field":"stock","op":"lt","value":10}`),
		RuleAction:    json.RawMessage(`{"action":"notify"}`),
		Priority:      100,
		Source:        "manual",
	}
	err := svc.CreateRule(rule)
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if rule.ID == 0 {
		t.Fatal("ID should be set")
	}

	// ListRules
	rules, err := svc.ListRules(1, "A5", "stock_alert")
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("rules = %d (expected 1)", len(rules))
	}
	if rules[0].RuleName != "低库存阈值" {
		t.Fatalf("RuleName = %s", rules[0].RuleName)
	}

	// UpdateRule - change status
	rule.Status = "inactive"
	err = svc.UpdateRule(rule)
	if err != nil {
		t.Fatalf("UpdateRule: %v", err)
	}

	rules, _ = svc.ListRules(1, "A5", "stock_alert")
	if len(rules) != 0 {
		t.Fatalf("rules after deactivation = %d (expected 0)", len(rules))
	}

	// Verify mark change was recorded
	var changes []RuleMarkChange
	db.Find(&changes)
	if len(changes) == 0 {
		t.Fatal("expected RuleMarkChange to be recorded")
	}

	// DeleteRule
	err = svc.DeleteRule(rule.ID)
	if err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}
}

func TestService_ApplyRules(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create an override rule
	svc.CreateRule(&PersonalRule{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		RuleType:      "override",
		RuleName:      "价格上限",
		RuleCondition: json.RawMessage(`{"field":"price","op":"gt","value":100}`),
		RuleAction:    json.RawMessage(`{"price":99}`),
		Source:        "manual",
	})

	output, err := svc.ApplyRules(&ApplyRulesInput{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		Output:        map[string]interface{}{"price": float64(150)},
	})
	if err != nil {
		t.Fatalf("ApplyRules: %v", err)
	}
	if len(output.Results) != 1 {
		t.Fatalf("results = %d", len(output.Results))
	}
	if output.Blocked {
		t.Fatal("should not be blocked")
	}
	// Price should be adjusted
	if output.Output["price"] != float64(99) {
		t.Fatalf("price = %v", output.Output["price"])
	}
}

func TestService_ApplyRules_NoMatch(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreateRule(&PersonalRule{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		RuleType:      "override",
		RuleName:      "价格上限",
		RuleCondition: json.RawMessage(`{"field":"price","op":"gt","value":200}`),
		RuleAction:    json.RawMessage(`{"price":99}`),
		Source:        "manual",
	})

	output, err := svc.ApplyRules(&ApplyRulesInput{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		Output:        map[string]interface{}{"price": float64(100)},
	})
	if err != nil {
		t.Fatalf("ApplyRules: %v", err)
	}
	if len(output.Results) != 0 {
		t.Fatalf("results = %d (expected 0)", len(output.Results))
	}
}

func TestListRules_ByAgent(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	cond := json.RawMessage(`{"field":"id","op":"gte","value":0}`)
	act := json.RawMessage(`{}`)

	svc.CreateRule(&PersonalRule{UserID: 1, AgentID: "A5", DecisionPoint: "stock_alert", RuleType: "threshold", RuleName: "A5规则", RuleCondition: cond, RuleAction: act, Source: "manual"})
	svc.CreateRule(&PersonalRule{UserID: 1, AgentID: "A6", DecisionPoint: "pricing", RuleType: "override", RuleName: "A6规则", RuleCondition: cond, RuleAction: act, Source: "manual"})

	// Filter by A5
	rules, err := svc.ListRules(1, "A5", "")
	if err != nil {
		t.Fatalf("ListRules(A5): %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("A5 rules = %d (expected 1)", len(rules))
	}
	if rules[0].AgentID != "A5" {
		t.Fatalf("wrong agent: %s", rules[0].AgentID)
	}

	// Filter by A6
	rules, err = svc.ListRules(1, "A6", "")
	if err != nil {
		t.Fatalf("ListRules(A6): %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("A6 rules = %d (expected 1)", len(rules))
	}
	if rules[0].AgentID != "A6" {
		t.Fatalf("wrong agent: %s", rules[0].AgentID)
	}
}

func TestListRules_ByDecisionPoint(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	cond := json.RawMessage(`{"field":"id","op":"gte","value":0}`)
	act := json.RawMessage(`{}`)

	svc.CreateRule(&PersonalRule{UserID: 1, AgentID: "A5", DecisionPoint: "stock_alert", RuleType: "threshold", RuleName: "库存规则", RuleCondition: cond, RuleAction: act, Source: "manual"})
	svc.CreateRule(&PersonalRule{UserID: 1, AgentID: "A5", DecisionPoint: "pricing", RuleType: "override", RuleName: "定价规则", RuleCondition: cond, RuleAction: act, Source: "manual"})

	// Filter by stock_alert
	rules, err := svc.ListRules(1, "", "stock_alert")
	if err != nil {
		t.Fatalf("ListRules(stock_alert): %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("stock_alert rules = %d (expected 1)", len(rules))
	}
	if rules[0].DecisionPoint != "stock_alert" {
		t.Fatalf("wrong decision_point: %s", rules[0].DecisionPoint)
	}

	// Filter by pricing
	rules, err = svc.ListRules(1, "", "pricing")
	if err != nil {
		t.Fatalf("ListRules(pricing): %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("pricing rules = %d (expected 1)", len(rules))
	}
	if rules[0].DecisionPoint != "pricing" {
		t.Fatalf("wrong decision_point: %s", rules[0].DecisionPoint)
	}
}

func TestListRules_Empty(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	rules, err := svc.ListRules(999, "A5", "stock_alert")
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("rules = %d (expected 0)", len(rules))
	}
}

func TestCreateRule_Fields(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	rule := &PersonalRule{
		UserID:        42,
		AgentID:       "A7",
		DecisionPoint: "compliance",
		RuleType:      "block",
		RuleName:      "合规检查",
		RuleCondition: json.RawMessage(`{"field":"risk_score","op":"gt","value":80}`),
		RuleAction:    json.RawMessage(`{"reason":"high risk"}`),
		Priority:      200,
		Source:        "ai_suggested",
		Status:        "draft",
		Confidence:    0.85,
	}

	err := svc.CreateRule(rule)
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if rule.ID == 0 {
		t.Fatal("ID should be set after Create")
	}

	// Read back and verify every field
	var saved PersonalRule
	if err := db.First(&saved, rule.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}

	if saved.UserID != 42 {
		t.Fatalf("UserID = %d", saved.UserID)
	}
	if saved.AgentID != "A7" {
		t.Fatalf("AgentID = %s", saved.AgentID)
	}
	if saved.DecisionPoint != "compliance" {
		t.Fatalf("DecisionPoint = %s", saved.DecisionPoint)
	}
	if saved.RuleType != "block" {
		t.Fatalf("RuleType = %s", saved.RuleType)
	}
	if saved.RuleName != "合规检查" {
		t.Fatalf("RuleName = %s", saved.RuleName)
	}
	if saved.Priority != 200 {
		t.Fatalf("Priority = %d", saved.Priority)
	}
	if saved.Source != "ai_suggested" {
		t.Fatalf("Source = %s", saved.Source)
	}
	if saved.Status != "draft" {
		t.Fatalf("Status = %s", saved.Status)
	}
	if saved.Confidence != 0.85 {
		t.Fatalf("Confidence = %f", saved.Confidence)
	}
}

func TestUpdateRule_Approved(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	rule := &PersonalRule{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		RuleType:      "threshold",
		RuleName:      "批准测试",
		RuleCondition: json.RawMessage(`{"field":"stock","op":"lt","value":5}`),
		RuleAction:    json.RawMessage(`{"action":"warn"}`),
		Source:        "manual",
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	// Update status to "approved"
	rule.Status = "approved"
	if err := svc.UpdateRule(rule); err != nil {
		t.Fatalf("UpdateRule: %v", err)
	}

	// Verify status persisted
	var saved PersonalRule
	if err := db.First(&saved, rule.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if saved.Status != "approved" {
		t.Fatalf("Status = %s (expected approved)", saved.Status)
	}

	// Verify RuleMarkChange was recorded
	var changes []RuleMarkChange
	db.Where("target_id = ? AND target_type = 'personal_rule'", rule.ID).Find(&changes)
	if len(changes) == 0 {
		t.Fatal("expected RuleMarkChange to be recorded for approved status change")
	}
	if string(changes[0].OldValue) != `"active"` {
		t.Fatalf("expected old value active, got %s", string(changes[0].OldValue))
	}
	if string(changes[0].NewValue) != `"approved"` {
		t.Fatalf("expected new value approved, got %s", string(changes[0].NewValue))
	}
}

func TestDeleteRule_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Delete a non-existent ID
	err := svc.DeleteRule(99999)
	if err == nil {
		t.Fatal("expected error when deleting non-existent rule")
	}
}

func TestApplyRules_Block(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreateRule(&PersonalRule{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		RuleType:      "block",
		RuleName:      "价格封禁",
		RuleCondition: json.RawMessage(`{"field":"price","op":"gt","value":1000}`),
		RuleAction:    json.RawMessage(`{"reason":"price exceeds max"}`),
		Source:        "manual",
	})

	output, err := svc.ApplyRules(&ApplyRulesInput{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		Output:        map[string]interface{}{"price": float64(1500)},
	})
	if err != nil {
		t.Fatalf("ApplyRules: %v", err)
	}
	if len(output.Results) != 1 {
		t.Fatalf("results = %d (expected 1)", len(output.Results))
	}
	if !output.Blocked {
		t.Fatal("expected blocked = true")
	}
	if output.AppliedRuleID == 0 {
		t.Fatal("expected AppliedRuleID to be set")
	}
	if output.Output["_blocked"] != true {
		t.Fatal("expected _blocked in output")
	}
}

func TestApplyRules_Confidence(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	cond := json.RawMessage(`{"field":"price","op":"gt","value":0}`)

	// Create two rules with different confidence values and priorities
	svc.CreateRule(&PersonalRule{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		RuleType:      "override",
		RuleName:      "低信心上限",
		RuleCondition: cond,
		RuleAction:    json.RawMessage(`{"price":50}`),
		Priority:      100,
		Confidence:    0.3,
		Source:        "manual",
	})
	svc.CreateRule(&PersonalRule{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		RuleType:      "override",
		RuleName:      "高信心上限",
		RuleCondition: cond,
		RuleAction:    json.RawMessage(`{"price":90}`),
		Priority:      200,
		Confidence:    0.9,
		Source:        "manual",
	})

	output, err := svc.ApplyRules(&ApplyRulesInput{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		Output:        map[string]interface{}{"price": float64(100)},
	})
	if err != nil {
		t.Fatalf("ApplyRules: %v", err)
	}
	// Both rules match; higher priority (200) applies first setting price=90,
	// then lower priority (100) applies setting price=50 (last write wins)
	if len(output.Results) != 2 {
		t.Fatalf("results = %d (expected 2)", len(output.Results))
	}
	if output.Output["price"] != float64(50) {
		t.Fatalf("final price = %v (expected 50)", output.Output["price"])
	}
	// Verify confidence values persisted on the rules
	rules, _ := svc.ListRules(1, "A5", "pricing")
	if len(rules) >= 1 && rules[0].RuleName == "高信心上限" && rules[0].Confidence != 0.9 {
		t.Fatalf("高信心上限 confidence = %f", rules[0].Confidence)
	}
}

func TestApplyRules_Enablement(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PersonalRule{}, &RuleMarkChange{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create a disabled rule
	svc.CreateRule(&PersonalRule{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		RuleType:      "override",
		RuleName:      "禁用的价格规则",
		RuleCondition: json.RawMessage(`{"field":"price","op":"gt","value":0}`),
		RuleAction:    json.RawMessage(`{"price":0}`),
		Status:        "inactive",
		Source:        "manual",
	})

	output, err := svc.ApplyRules(&ApplyRulesInput{
		UserID:        1,
		AgentID:       "A5",
		DecisionPoint: "pricing",
		Output:        map[string]interface{}{"price": float64(100)},
	})
	if err != nil {
		t.Fatalf("ApplyRules: %v", err)
	}
	if len(output.Results) != 0 {
		t.Fatalf("results = %d (expected 0 for disabled rule)", len(output.Results))
	}
	// Output should be unchanged
	if output.Output["price"] != float64(100) {
		t.Fatalf("price = %v (expected 100, unchanged)", output.Output["price"])
	}
}

func TestDeepCopyMap(t *testing.T) {
	t.Parallel()

	// nil map → nil
	if got := deepCopyMap(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}

	src := map[string]interface{}{
		"price":  float64(100),
		"status": "active",
		"meta": map[string]interface{}{
			"name": "test",
		},
	}

	dst := deepCopyMap(src)
	if dst == nil {
		t.Fatal("got nil copy")
	}

	// Modify top-level value in copy
	dst["price"] = float64(200)
	if src["price"] != float64(100) {
		t.Fatal("source price changed after modifying copy")
	}

	// Modify nested value in copy
	nestedDst, _ := dst["meta"].(map[string]interface{})
	nestedDst["name"] = "modified"
	nestedSrc, _ := src["meta"].(map[string]interface{})
	if nestedSrc["name"] != "test" {
		t.Fatal("source nested value changed after modifying copy — copy is not deep")
	}
}
