package agentrule

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

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
