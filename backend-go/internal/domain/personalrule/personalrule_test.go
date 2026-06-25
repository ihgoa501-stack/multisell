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
