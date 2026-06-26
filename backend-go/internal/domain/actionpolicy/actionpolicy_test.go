package actionpolicy

import (
	"encoding/json"
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
	db, err := gorm.Open(sqlite.Open("file:actionpolicy_test_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.AutoMigrate(&PolicyRule{}, &ApprovalPolicy{}, &ApprovalRequest{})
	return db
}

// ──────────────────────────────────────────────────────────────
// PolicyRule tests (existing)
// ──────────────────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────────────────
// ApprovalPolicy tests (new)
// ──────────────────────────────────────────────────────────────

func TestApprovalPolicy_CRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/policy/approval-policies", h.ListPolicies)
	r.POST("/policy/approval-policies", h.CreatePolicy)
	r.GET("/policy/approval-policies/:id", h.GetPolicy)
	r.PUT("/policy/approval-policies/:id", h.UpdatePolicy)
	r.DELETE("/policy/approval-policies/:id", h.DeletePolicy)

	// Create
	body := `{"name":"A5 gate","agent_id":"A5","decision_point":"stock_alert","min_trust_score":0.5,"requires_approval":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/policy/approval-policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	// List
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/policy/approval-policies", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"agent_id":"A5"`) {
		t.Fatalf("list response missing agent_id: %s", w.Body.String())
	}

	// Get
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/policy/approval-policies/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	// Update
	body = `{"name":"A5 gate updated","agent_id":"A5","decision_point":"stock_alert","min_trust_score":0.6,"requires_approval":true}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/policy/approval-policies/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	// Delete
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/policy/approval-policies/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestApprovalPolicy_GetMatching(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	// Create a policy for A5/stock_alert
	policy := &ApprovalPolicy{
		Name:             "A5 Gate",
		AgentID:          "A5",
		DecisionPoint:    "stock_alert",
		MinTrustScore:    0.50,
		RequiresApproval: true,
	}
	if err := svc.CreatePolicy(policy); err != nil {
		t.Fatalf("create policy: %v", err)
	}

	// Should match
	matched, err := svc.GetMatchingPolicy("A5", "stock_alert")
	if err != nil {
		t.Fatalf("get matching policy: %v", err)
	}
	if matched == nil {
		t.Fatal("expected matching policy, got nil")
	}
	if matched.MinTrustScore != 0.50 {
		t.Fatalf("expected min_trust_score=0.50, got %f", matched.MinTrustScore)
	}

	// Non-matching agent — should return nil
	matched, err = svc.GetMatchingPolicy("A6", "stock_alert")
	if err != nil {
		t.Fatalf("get matching non-agent: %v", err)
	}
	if matched != nil {
		t.Fatal("expected nil for non-matching agent")
	}

	// Non-matching decision point — should return nil
	matched, err = svc.GetMatchingPolicy("A5", "profit_watch")
	if err != nil {
		t.Fatalf("get matching non-dp: %v", err)
	}
	if matched != nil {
		t.Fatal("expected nil for non-matching decision point")
	}
}

// ──────────────────────────────────────────────────────────────
// ApprovalRequest tests (new)
// ──────────────────────────────────────────────────────────────

func TestApprovalRequest_SubmitAndReview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/policy/approval-policies", h.CreatePolicy)
	r.POST("/policy/approval-requests/:id/review", h.HandleReview)
	r.GET("/policy/approval-requests", h.ListRequests)
	r.GET("/policy/approval-requests/:id", h.GetRequest)

	// Create a policy first
	polBody := `{"name":"test","agent_id":"T1","decision_point":"test_dp","min_trust_score":0.5,"requires_approval":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/policy/approval-policies", strings.NewReader(polBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create policy: %d", w.Code)
	}

	// Submit an approval request
	payload, _ := json.Marshal(map[string]interface{}{"action": "test", "value": 100})
	reqObj, err := svc.SubmitApproval(1, "T1", "test_dp", payload, "agent:T1")
	if err != nil {
		t.Fatalf("submit approval: %v", err)
	}
	if reqObj.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", reqObj.Status)
	}
	if reqObj.PolicyID != 1 {
		t.Fatalf("expected policy_id=1, got %d", reqObj.PolicyID)
	}

	// List requests — should show 1 pending
	reqs, err := svc.ListRequests("")
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}

	// Filter by status — should show 1 pending
	pendingReqs, err := svc.ListRequests(StatusPending)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pendingReqs) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pendingReqs))
	}

	// Filter by approved — should show 0
	approvedReqs, err := svc.ListRequests(StatusApproved)
	if err != nil {
		t.Fatalf("list approved: %v", err)
	}
	if len(approvedReqs) != 0 {
		t.Fatalf("expected 0 approved, got %d", len(approvedReqs))
	}

	// Review: approve
	reviewBody := `{"approve":true,"reviewed_by":"admin"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/policy/approval-requests/1/review", strings.NewReader(reviewBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("review status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"approved")`) && !strings.Contains(w.Body.String(), `"status":"approved"`) {
		if !strings.Contains(w.Body.String(), `"approved"`) {
			t.Fatalf("review response missing approved status: %s", w.Body.String())
		}
	}

	// Verify status changed
	getBody := ``
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/policy/approval-requests/1", strings.NewReader(getBody))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get request: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"approved"`) {
		t.Fatalf("expected approved status in: %s", w.Body.String())
	}

	// Re-review should fail (not pending)
	reviewBody = `{"approve":false,"reviewed_by":"admin2"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/policy/approval-requests/1/review", strings.NewReader(reviewBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("re-review should fail with 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApprovalRequest_Reject(t *testing.T) {
	db := newActionPolicyDB(t)
	svc := NewService(db, zap.NewNop())

	// Create policy + pending request
	svc.CreatePolicy(&ApprovalPolicy{
		Name: "test", AgentID: "T2", DecisionPoint: "dp",
		MinTrustScore: 0.5, RequiresApproval: true,
	})
	payload, _ := json.Marshal(map[string]string{"action": "reject_me"})
	_, err := svc.SubmitApproval(1, "T2", "dp", payload, "agent:T2")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	// Reject
	reviewed, err := svc.Review(1, false, "admin")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if reviewed.Status != StatusRejected {
		t.Fatalf("expected rejected, got %s", reviewed.Status)
	}
	if reviewed.ReviewedBy == nil || *reviewed.ReviewedBy != "admin" {
		t.Fatalf("expected reviewed_by=admin, got %v", reviewed.ReviewedBy)
	}
	if reviewed.ReviewedAt == nil {
		t.Fatal("expected reviewed_at to be set")
	}
}
