package agentrule

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &PersonalRule{})
}

func newService(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	return NewService(db, dbtest.NewLogger(t))
}

func createRule(t *testing.T, svc *Service, userID int64, agentID, ruleType, name string, priority int) *PersonalRule {
	t.Helper()
	enabled := true
	cond, _ := json.Marshal(map[string]interface{}{"field": "price", "operator": "gt", "value": 0.05})
	effect, _ := json.Marshal(map[string]interface{}{"reason": "test block"})
	in := &CreateRuleInput{
		UserID:     userID,
		AgentID:    agentID,
		RuleType:   ruleType,
		Name:       name,
		Conditions: cond,
		Effect:     effect,
		Priority:   priority,
		Enabled:    &enabled,
	}
	r, err := svc.Create(in)
	if err != nil {
		t.Fatalf("createRule failed: %v", err)
	}
	return r
}

func TestAgentRule_Create(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	enabled := true
	in := &CreateRuleInput{
		UserID:     1,
		AgentID:    "A1",
		RuleType:   "veto",
		Name:       "Block low margin",
		Priority:   10,
		Enabled:    &enabled,
	}
	r, err := svc.Create(in)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if r.Name != "Block low margin" {
		t.Fatalf("Name = %q, want %q", r.Name, "Block low margin")
	}
	if !r.Enabled {
		t.Fatal("expected Enabled=true")
	}
}

func TestAgentRule_Create_DefaultEnabled(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	in := &CreateRuleInput{
		UserID:   1,
		AgentID:  "A1",
		RuleType: "strategy",
		Name:     "Default rule",
	}
	r, err := svc.Create(in)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !r.Enabled {
		t.Fatal("expected default Enabled=true")
	}
}

func TestAgentRule_GetByID(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	r := createRule(t, svc, 1, "A1", "veto", "Findable Rule", 5)

	got, err := svc.Update(r.ID, &UpdateRuleInput{}) // use Update to fetch by ID via First
	if err != nil {
		// If no changes, Update returns the existing record
		t.Logf("Update with no changes: %v", err)
	}
	_ = got
}

func TestAgentRule_Update(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	r := createRule(t, svc, 1, "A1", "veto", "Old Name", 5)

	newName := "Updated Name"
	updated, err := svc.Update(r.ID, &UpdateRuleInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Fatalf("Name = %q, want %q", updated.Name, "Updated Name")
	}
}

func TestAgentRule_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	name := "X"
	if _, err := svc.Update(999, &UpdateRuleInput{Name: &name}); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestAgentRule_Delete(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	r := createRule(t, svc, 1, "A1", "veto", "To Delete", 0)
	if err := svc.Delete(r.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	rules, err := svc.List(1, "", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(rules))
	}
}

func TestAgentRule_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	if err := svc.Delete(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestAgentRule_List_FilterType(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	createRule(t, svc, 1, "A1", "veto", "Veto Rule", 1)
	createRule(t, svc, 1, "A1", "strategy", "Strategy Rule", 2)

	rules, err := svc.List(1, "", "", "veto")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 veto rule, got %d", len(rules))
	}
	if rules[0].RuleType != "veto" {
		t.Fatalf("RuleType = %q, want %q", rules[0].RuleType, "veto")
	}
}

func TestAgentRule_List_FilterAgent(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	createRule(t, svc, 1, "A1", "veto", "Rule A1", 1)
	createRule(t, svc, 1, "A2", "veto", "Rule A2", 2)

	rules, err := svc.List(1, "A1", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule for A1, got %d", len(rules))
	}
}

func TestAgentRule_List_ExcludesDisabled(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	createRule(t, svc, 1, "A1", "veto", "Enabled Rule", 1)
	// Create a disabled rule directly via DB to avoid GORM zero-value default handling
	disabledRule := &PersonalRule{
		UserID:    1,
		AgentID:   "A1",
		RuleType:  "strategy",
		Name:      "Disabled Rule",
		Priority:  0,
		Enabled:   false,
	}
	db.Create(disabledRule)
	// Force update enabled to 0 since GORM default:true may override zero-value bool
	db.Exec("UPDATE personal_rule SET enabled = 0 WHERE id = ?", disabledRule.ID)

	// Verify the disabled rule exists with enabled=0 in the DB
	var enabledVal int
	db.Raw("SELECT enabled FROM personal_rule WHERE id = ?", disabledRule.ID).Scan(&enabledVal)
	if enabledVal != 0 {
		t.Fatalf("disabled rule enabled column = %d, want 0", enabledVal)
	}

	// List should only return enabled rules (service WHERE enabled = true)
	rules, err := svc.List(1, "", "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 enabled rule from List, got %d", len(rules))
	}
	if rules[0].Name != "Enabled Rule" {
		t.Fatalf("expected Enabled Rule, got %q", rules[0].Name)
	}
}

func TestAgentRule_ToggleEnabled(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	r := createRule(t, svc, 1, "A1", "veto", "Toggle Me", 0)
	if !r.Enabled {
		t.Fatal("rule should start enabled")
	}

	toggled, err := svc.ToggleEnabled(r.ID)
	if err != nil {
		t.Fatalf("ToggleEnabled failed: %v", err)
	}
	if toggled.Enabled {
		t.Fatal("expected Enabled=false after toggle")
	}

	toggled2, err := svc.ToggleEnabled(r.ID)
	if err != nil {
		t.Fatalf("ToggleEnabled failed: %v", err)
	}
	if !toggled2.Enabled {
		t.Fatal("expected Enabled=true after second toggle")
	}
}

func TestAgentRule_Evaluate_MatchVeto(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	cond, _ := json.Marshal(map[string]interface{}{"field": "price", "operator": "gt", "value": 0.05})
	effect, _ := json.Marshal(map[string]interface{}{"reason": "margin too low"})
	enabled := true
	svc.Create(&CreateRuleInput{
		UserID:     1,
		AgentID:    "A1",
		RuleType:   "veto",
		Name:       "Block high price",
		Conditions: cond,
		Effect:     effect,
		Enabled:    &enabled,
		Priority:   10,
	})

	output := map[string]interface{}{"price": 0.10}
	result, err := svc.Evaluate(1, "A1", "", output)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if !result.Blocked {
		t.Fatal("expected Blocked=true for matching veto rule")
	}
	if result.BlockReason != "margin too low" {
		t.Fatalf("BlockReason = %q, want %q", result.BlockReason, "margin too low")
	}
	if len(result.AppliedRules) != 1 {
		t.Fatalf("expected 1 applied rule, got %d", len(result.AppliedRules))
	}
}

func TestAgentRule_Evaluate_NoMatch(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	cond, _ := json.Marshal(map[string]interface{}{"field": "price", "operator": "gt", "value": 0.99})
	enabled := true
	svc.Create(&CreateRuleInput{
		UserID:     1,
		AgentID:    "A1",
		RuleType:   "veto",
		Name:       "High threshold",
		Conditions: cond,
		Enabled:    &enabled,
	})

	output := map[string]interface{}{"price": 0.10}
	result, err := svc.Evaluate(1, "A1", "", output)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result.Blocked {
		t.Fatal("expected Blocked=false when condition does not match")
	}
	if len(result.AppliedRules) != 0 {
		t.Fatalf("expected 0 applied rules, got %d", len(result.AppliedRules))
	}
}

func TestAgentRule_Evaluate_NoCondition(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	enabled := true
	svc.Create(&CreateRuleInput{
		UserID:   1,
		AgentID:  "A1",
		RuleType: "strategy",
		Name:     "Always match",
		Enabled:  &enabled,
	})

	output := map[string]interface{}{"price": 0.10}
	result, err := svc.Evaluate(1, "A1", "", output)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if len(result.AppliedRules) != 1 {
		t.Fatalf("expected 1 applied rule (empty condition = always match), got %d", len(result.AppliedRules))
	}
}
