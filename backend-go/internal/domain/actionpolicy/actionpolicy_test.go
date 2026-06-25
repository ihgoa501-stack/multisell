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
	db, err := gorm.Open(sqlite.Open("file:actionpolicy_test?mode=memory&cache=shared"), &gorm.Config{})
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
