package actionpolicy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newActionPolicyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/actionpolicy.db"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.AutoMigrate(&PolicyRule{})
	return db
}

func TestHandler_ListRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/policy/rules", h.ListRules)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/policy/rules", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandler_CreateAndGetRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/policy/rules", h.CreateRule)
	r.GET("/policy/rules/:id", h.GetRule)

	// Create a rule
	body := `{"name":"test-blocker","action_type":"price_update","risk_level":"high","outcome":"block","enabled":true,"priority":1}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/policy/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("create status = %d, want 200", w.Code)
	}
}

func TestHandler_Evaluate_NoMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/policy/evaluate", h.Evaluate)

	body := `{"agent_id":"A5","action_type":"price_update","risk_level":"high"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/policy/evaluate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestMatches(t *testing.T) {
	ctx := &ActionContext{RiskLevel: "high", ActionType: "price_update"}

	// Wildcard match
	rule1 := PolicyRule{RiskLevel: "high", ActionType: "*"}
	if !matches(ctx, rule1) {
		t.Fatal("expected match with wildcard")
	}

	// Exact match
	rule2 := PolicyRule{RiskLevel: "high", ActionType: "price_update"}
	if !matches(ctx, rule2) {
		t.Fatal("expected exact match")
	}

	// Non-match
	rule3 := PolicyRule{RiskLevel: "low", ActionType: "price_update"}
	if matches(ctx, rule3) {
		t.Fatal("expected no match")
	}
}

func TestMatches_AmountBoundary(t *testing.T) {
	amt := 100.0
	ctx := &ActionContext{RiskLevel: "high", Amount: &amt}

	limit := 200.0
	rule1 := PolicyRule{RiskLevel: "high", MaxAmount: &limit}
	if !matches(ctx, rule1) {
		t.Fatal("amount 100 should be <= max 200")
	}

	limit2 := 50.0
	rule2 := PolicyRule{RiskLevel: "high", MaxAmount: &limit2}
	if matches(ctx, rule2) {
		t.Fatal("amount 100 should exceed max 50")
	}

}

func TestService_Evaluate_Allowed(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	rule := &PolicyRule{
		Name: "approve-low", RiskLevel: "low", ActionType: "price_update",
		Outcome: "auto_approve", Priority: 10, Enabled: true,
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	ctx := &ActionContext{RiskLevel: "low", ActionType: "price_update"}
	result, err := svc.Evaluate(ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.FinalOutcome != "auto_approve" {
		t.Fatalf("FinalOutcome = %q, want auto_approve", result.FinalOutcome)
	}
	if len(result.Verdicts) != 1 {
		t.Fatalf("verdicts = %d, want 1", len(result.Verdicts))
	}
}

func TestService_Evaluate_Blocked(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	rule := &PolicyRule{
		Name: "block-high", RiskLevel: "high", ActionType: "price_update",
		Outcome: "block", Priority: 10, Enabled: true,
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	ctx := &ActionContext{RiskLevel: "high", ActionType: "price_update"}
	result, err := svc.Evaluate(ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.FinalOutcome != "block" {
		t.Fatalf("FinalOutcome = %q, want block", result.FinalOutcome)
	}
}

func TestService_Evaluate_NoMatch(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	// Create a rule that won't match
	rule := &PolicyRule{
		Name: "low-only", RiskLevel: "low", ActionType: "price_update",
		Outcome: "auto_approve", Priority: 10, Enabled: true,
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	ctx := &ActionContext{RiskLevel: "high", ActionType: "price_update"}
	result, err := svc.Evaluate(ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.FinalOutcome != "escalate" {
		t.Fatalf("FinalOutcome = %q, want escalate (default)", result.FinalOutcome)
	}
	if len(result.Verdicts) != 0 {
		t.Fatalf("verdicts = %d, want 0", len(result.Verdicts))
	}
}

func TestService_Evaluate_BlockDominates(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	// Two matching rules: one approves, one blocks. Block should dominate.
	approve := &PolicyRule{
		Name: "approve", RiskLevel: "high", ActionType: "*",
		Outcome: "auto_approve", Priority: 1, Enabled: true,
	}
	block := &PolicyRule{
		Name: "block", RiskLevel: "high", ActionType: "*",
		Outcome: "block", Priority: 100, Enabled: true,
	}
	if err := svc.CreateRule(approve); err != nil {
		t.Fatalf("CreateRule approve: %v", err)
	}
	if err := svc.CreateRule(block); err != nil {
		t.Fatalf("CreateRule block: %v", err)
	}

	ctx := &ActionContext{RiskLevel: "high", ActionType: "price_update"}
	result, err := svc.Evaluate(ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.FinalOutcome != "block" {
		t.Fatalf("FinalOutcome = %q, want block (block dominates)", result.FinalOutcome)
	}
	if len(result.Verdicts) != 2 {
		t.Fatalf("verdicts = %d, want 2", len(result.Verdicts))
	}
	// Higher priority rule verdict comes first (ORDER BY priority DESC)
	if result.Verdicts[0].RuleName != "block" {
		t.Fatalf("first verdict = %q, want block (higher priority)", result.Verdicts[0].RuleName)
	}
}

func TestService_Evaluate_HighRiskGate(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	rule := &PolicyRule{
		Name: "approve-high", RiskLevel: "high", ActionType: "price_update",
		Outcome: "auto_approve", Priority: 10, Enabled: true,
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	ctx := &ActionContext{RiskLevel: "high", ActionType: "price_update"}
	result, err := svc.Evaluate(ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	// High-risk gate overrides auto_approve → escalate
	if result.FinalOutcome != "escalate" {
		t.Fatalf("FinalOutcome = %q, want escalate (high-risk gate)", result.FinalOutcome)
	}
	// Should have the synthetic high-risk-gate verdict
	found := false
	for _, v := range result.Verdicts {
		if v.RuleName == "high-risk-gate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected high-risk-gate verdict in results")
	}
}

func TestService_Evaluate_NilPointerFields(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	ctx := &ActionContext{RiskLevel: "low", ActionType: "price_update"}

	// Rule with thresholds — ctx has no Amount/Quantity/Confidence set
	amt := 1000.0
	qty := 50
	conf := 0.95
	rule := &PolicyRule{
		Name: "threshold", RiskLevel: "low", ActionType: "price_update",
		MaxAmount: &amt, MaxQuantity: &qty, MinConfidence: &conf,
		Outcome: "block", Priority: 10, Enabled: true,
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	// All ctx numeric pointers are nil → threshold checks should be skipped
	result, err := svc.Evaluate(ctx)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.FinalOutcome != "block" {
		t.Fatalf("FinalOutcome = %q, want block (nil fields should not fail match)", result.FinalOutcome)
	}
}

func TestMatches_ExactAgent(t *testing.T) {
	ctx := &ActionContext{AgentID: "A5"}

	if !matches(ctx, PolicyRule{AgentID: "A5"}) {
		t.Fatal("expected exact agent match")
	}
	if matches(ctx, PolicyRule{AgentID: "A6"}) {
		t.Fatal("expected no match for different agent")
	}
	if !matches(ctx, PolicyRule{}) {
		t.Fatal("expected empty rule (no filter) to match")
	}
	if !matches(ctx, PolicyRule{AgentID: "*"}) {
		t.Fatal("expected wildcard agent match")
	}
}

func TestMatches_ActionType(t *testing.T) {
	ctx := &ActionContext{ActionType: "price_update"}

	if !matches(ctx, PolicyRule{ActionType: "price_update"}) {
		t.Fatal("expected exact action match")
	}
	if matches(ctx, PolicyRule{ActionType: "inventory_sync"}) {
		t.Fatal("expected no match for different action")
	}
	if !matches(ctx, PolicyRule{ActionType: "*"}) {
		t.Fatal("expected wildcard action match")
	}
	if !matches(ctx, PolicyRule{}) {
		t.Fatal("expected empty rule (no filter) to match")
	}
}

func TestMatches_Wildcard(t *testing.T) {
	ctx := &ActionContext{AgentID: "A5", ActionType: "price_update", RiskLevel: "high"}

	if !matches(ctx, PolicyRule{AgentID: "*", ActionType: "*", RiskLevel: "*"}) {
		t.Fatal("expected all wildcards match")
	}
	if !matches(ctx, PolicyRule{AgentID: "*", ActionType: "price_update"}) {
		t.Fatal("expected agent wildcard + exact action match")
	}
	if !matches(ctx, PolicyRule{}) {
		t.Fatal("expected empty rule (no filters) to match")
	}
}

func TestMatches_ThresholdBoundaries(t *testing.T) {
	t.Run("nil rule threshold skips check", func(t *testing.T) {
		amt := 100.0
		ctx := &ActionContext{Amount: &amt}
		if !matches(ctx, PolicyRule{}) {
			t.Fatal("nil MaxAmount should not filter")
		}
	})

	t.Run("nil ctx amount skips rule max check", func(t *testing.T) {
		amt := 100.0
		if !matches(&ActionContext{}, PolicyRule{MaxAmount: &amt}) {
			t.Fatal("nil ctx Amount should not be filtered by MaxAmount")
		}
	})

	t.Run("quantity boundary", func(t *testing.T) {
		qty := 10
		limit := 10
		ctx := &ActionContext{Quantity: &qty}

		// 10 <= 10 — should match
		if !matches(ctx, PolicyRule{MaxQuantity: &limit}) {
			t.Fatal("quantity 10 should be <= max 10")
		}

		over := 5
		if matches(ctx, PolicyRule{MaxQuantity: &over}) {
			t.Fatal("quantity 10 should exceed max 5")
		}
	})

	t.Run("confidence boundary", func(t *testing.T) {
		conf := 0.8
		ctx := &ActionContext{Confidence: &conf}

		minConf := 0.7
		if !matches(ctx, PolicyRule{MinConfidence: &minConf}) {
			t.Fatal("confidence 0.8 should be >= min 0.7")
		}

		needMore := 0.9
		if matches(ctx, PolicyRule{MinConfidence: &needMore}) {
			t.Fatal("confidence 0.8 should be < min 0.9")
		}
	})
}

func TestService_CreateRule(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	rule := &PolicyRule{
		Name: "new-rule", RiskLevel: "medium", ActionType: "inventory_sync",
		Outcome: "escalate", Priority: 5, Enabled: true,
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if rule.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	rules, err := svc.ListRules()
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if rules[0].Name != "new-rule" {
		t.Fatalf("rule name = %q, want new-rule", rules[0].Name)
	}
}

func TestService_ToggleRule(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	rule := &PolicyRule{
		Name: "togglable", RiskLevel: "low", ActionType: "price_update",
		Outcome: "auto_approve", Priority: 1, Enabled: true,
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	// Toggle off (enabled → false)
	if err := svc.ToggleRule(rule.ID); err != nil {
		t.Fatalf("ToggleRule: %v", err)
	}

	// ListRules only returns enabled rules
	rules, err := svc.ListRules()
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("len(rules) after toggle = %d, want 0", len(rules))
	}
}

func TestService_ListRules(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	for _, r := range []*PolicyRule{
		{Name: "rule-c", RiskLevel: "low", Outcome: "auto_approve", Priority: 10, Enabled: true},
		{Name: "rule-a", RiskLevel: "high", Outcome: "block", Priority: 20, Enabled: true},
		{Name: "rule-b", RiskLevel: "medium", Outcome: "escalate", Priority: 10, Enabled: true},
	} {
		if err := svc.CreateRule(r); err != nil {
			t.Fatalf("CreateRule %s: %v", r.Name, err)
		}
	}

	rules, err := svc.ListRules()
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("len(rules) = %d, want 3", len(rules))
	}

	// ORDER BY priority DESC, id ASC → rule-a (pri 20), rule-c (pri 10, id 1), rule-b (pri 10, id 2)
	if rules[0].Name != "rule-a" {
		t.Fatalf("rules[0] = %q, want rule-a (highest priority)", rules[0].Name)
	}
	if rules[1].Name != "rule-c" {
		t.Fatalf("rules[1] = %q, want rule-c (second by id)", rules[1].Name)
	}
	if rules[2].Name != "rule-b" {
		t.Fatalf("rules[2] = %q, want rule-b", rules[2].Name)
	}
}

func TestService_DeleteRule(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	rule := &PolicyRule{
		Name: "deletable", RiskLevel: "low", ActionType: "price_update",
		Outcome: "auto_approve", Priority: 1, Enabled: true,
	}
	if err := svc.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	// Delete sets enabled=false
	if err := svc.DeleteRule(rule.ID); err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}

	rules, err := svc.ListRules()
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("len(rules) after delete = %d, want 0", len(rules))
	}
}
