package approval

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/response"
)

// ---------------------------------------------------------------------------
// Service tests
// ---------------------------------------------------------------------------

func TestApproval_CreateAndGet(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	req, err := s.Create(&CreateApprovalInput{
		ProductID:   100,
		RequestType: "publish",
		Requester:   "A3",
		Reason:      "low confidence",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if req.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if req.Status != "pending" {
		t.Errorf("expected status pending, got %s", req.Status)
	}

	got, err := s.Get(req.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ProductID != 100 || got.RequestType != "publish" {
		t.Errorf("got ProductID=%d RequestType=%s", got.ProductID, got.RequestType)
	}
}

func TestApproval_Get_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	_, err := s.Get(99999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestApproval_List(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	for _, r := range []CreateApprovalInput{
		{ProductID: 1, RequestType: "publish", Requester: "A3"},
		{ProductID: 2, RequestType: "price_change", Requester: "A3"},
		{ProductID: 3, RequestType: "publish", Requester: "A8"},
	} {
		s.Create(&r)
	}

	all, total, _ := s.List(1, 10, "", "")
	if total != 3 || len(all) != 3 {
		t.Errorf("expected 3 total, got %d / %d", total, len(all))
	}

	filtered, ftotal, _ := s.List(1, 10, "pending", "publish")
	if ftotal != 2 {
		t.Errorf("expected 2 pending publish, got %d", ftotal)
	}
	_ = filtered

	// Empty result
	empty, _, _ := s.List(1, 10, "approved", "")
	if len(empty) != 0 {
		t.Errorf("expected empty list for approved, got %d", len(empty))
	}
}

func TestApproval_Review(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	req, _ := s.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "publish", Requester: "A3",
	})

	reviewed, err := s.Review(req.ID, &ReviewApprovalInput{
		Action:     "approve",
		Reviewer:   "user-1",
		ReviewNote: "looks good",
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if reviewed.Status != "approved" {
		t.Errorf("expected approved, got %s", reviewed.Status)
	}
	if reviewed.Reviewer != "user-1" {
		t.Errorf("expected reviewer user-1, got %s", reviewed.Reviewer)
	}

	// Re-review should fail
	_, err = s.Review(req.ID, &ReviewApprovalInput{Action: "reject", Reviewer: "user-2"})
	if err == nil {
		t.Fatal("expected error reviewing already-reviewed request")
	}
}

func TestApproval_Review_Reject(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	req, _ := s.Create(&CreateApprovalInput{
		ProductID: 1, RequestType: "publish", Requester: "A3",
	})

	reviewed, err := s.Review(req.ID, &ReviewApprovalInput{
		Action: "reject", Reviewer: "user-1", ReviewNote: "not now",
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if reviewed.Status != "rejected" {
		t.Errorf("expected rejected, got %s", reviewed.Status)
	}
	if reviewed.ReviewNote != "not now" {
		t.Errorf("expected note 'not now', got %s", reviewed.ReviewNote)
	}
}

func TestApproval_Review_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	_, err := s.Review(99999, &ReviewApprovalInput{Action: "approve", Reviewer: "u"})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestApproval_MyPending(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	for i := 1; i <= 3; i++ {
		s.Create(&CreateApprovalInput{
			ProductID: int64(i), RequestType: "publish", Requester: "A3",
		})
	}

	items, total, _ := s.MyPending(1, 10)
	if total != 3 || len(items) != 3 {
		t.Errorf("expected 3 pending, got %d / %d", total, len(items))
	}
}

func TestApproval_Stats(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	s.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "A3"})
	s.Create(&CreateApprovalInput{ProductID: 2, RequestType: "publish", Requester: "A3"})
	r3, _ := s.Create(&CreateApprovalInput{ProductID: 3, RequestType: "price_change", Requester: "A8"})
	s.Review(r3.ID, &ReviewApprovalInput{Action: "approve", Reviewer: "u1"})

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.PendingCount != 2 {
		t.Errorf("expected 2 pending, got %d", stats.PendingCount)
	}
	if stats.ApprovedCount != 1 {
		t.Errorf("expected 1 approved, got %d", stats.ApprovedCount)
	}
	if stats.TotalCount != 3 {
		t.Errorf("expected 3 total, got %d", stats.TotalCount)
	}
}

func TestApproval_AutoEscalate(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	s := NewService(db, dbtest.NewLogger(t))

	// Create a request that's 48 hours old
	old := &ApprovalRequest{
		ProductID: 1, RequestType: "publish", Requester: "A3",
		Status: "pending", CreatedAt: time.Now().Add(-48 * time.Hour),
	}
	db.Create(old)

	// Create a recent request
	s.Create(&CreateApprovalInput{ProductID: 2, RequestType: "publish", Requester: "A3"})

	items, err := s.AutoEscalate()
	if err != nil {
		t.Fatalf("AutoEscalate: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 escalated, got %d", len(items))
	}
	if items[0].ProductID != 1 {
		t.Errorf("expected product 1 (old), got %d", items[0].ProductID)
	}
}

// ---------------------------------------------------------------------------
// Handler tests
// ---------------------------------------------------------------------------

func setupHandler(t *testing.T) (*Handler, *Service) {
	t.Helper()
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t))
	return NewHandler(svc), svc
}

func ginCtx(method, path, body string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return w, c
}

func TestHandler_CreateApproval(t *testing.T) {
	h, _ := setupHandler(t)

	w, c := ginCtx("POST", "/api/v1/approval", `{
		"product_id": 1, "request_type": "publish", "requester": "A3"
	}`)
	h.CreateApproval(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp response.Result
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestHandler_CreateApproval_Validation(t *testing.T) {
	h, _ := setupHandler(t)

	w, c := ginCtx("POST", "/api/v1/approval", `{}`)
	h.CreateApproval(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_GetApproval(t *testing.T) {
	h, s := setupHandler(t)
	req, _ := s.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "A3"})

	w, c := ginCtx("GET", "/api/v1/approval/"+dbtest.IToA(req.ID), "")
	c.Params = []gin.Param{{Key: "id", Value: dbtest.IToA(req.ID)}}
	h.GetApproval(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetApproval_NotFound(t *testing.T) {
	h, _ := setupHandler(t)

	w, c := ginCtx("GET", "/api/v1/approval/99999", "")
	c.Params = []gin.Param{{Key: "id", Value: "99999"}}
	h.GetApproval(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandler_GetApproval_InvalidID(t *testing.T) {
	h, _ := setupHandler(t)

	w, c := ginCtx("GET", "/api/v1/approval/abc", "")
	c.Params = []gin.Param{{Key: "id", Value: "abc"}}
	h.GetApproval(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid ID, got %d", w.Code)
	}
}

func TestHandler_ReviewApproval(t *testing.T) {
	h, s := setupHandler(t)
	req, _ := s.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "A3"})

	w, c := ginCtx("PUT", "/api/v1/approval/"+dbtest.IToA(req.ID)+"/review", `{
		"action": "approve", "reviewer": "user-1"
	}`)
	c.Params = []gin.Param{{Key: "id", Value: dbtest.IToA(req.ID)}}
	h.ReviewApproval(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ReviewApproval_InvalidAction(t *testing.T) {
	h, s := setupHandler(t)
	req, _ := s.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "A3"})

	w, c := ginCtx("PUT", "/api/v1/approval/"+dbtest.IToA(req.ID)+"/review", `{
		"action": "invalid", "reviewer": "user-1"
	}`)
	c.Params = []gin.Param{{Key: "id", Value: dbtest.IToA(req.ID)}}
	h.ReviewApproval(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid action, got %d", w.Code)
	}
}

func TestHandler_ListApprovals(t *testing.T) {
	h, s := setupHandler(t)
	s.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "A3"})

	w, c := ginCtx("GET", "/api/v1/approval?page=1&size=10", "")
	h.ListApprovals(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_MyPending(t *testing.T) {
	h, s := setupHandler(t)
	s.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "A3"})

	w, c := ginCtx("GET", "/api/v1/approval/my?page=1&size=10", "")
	h.MyPending(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ApprovalStats(t *testing.T) {
	h, s := setupHandler(t)
	s.Create(&CreateApprovalInput{ProductID: 1, RequestType: "publish", Requester: "A3"})

	w, c := ginCtx("GET", "/api/v1/approval/stats", "")
	h.ApprovalStats(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Subscriber tests
// ---------------------------------------------------------------------------

func TestNewAgentDecisionSubscriber_HighConfidence(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	logger := dbtest.NewLogger(t)

	handler := NewAgentDecisionSubscriber(db, logger)
	evt := eventbus.Event{
		Topic: "agent.decided.A3.acos_analysis",
		Payload: map[string]interface{}{
			"confidence": 0.85,
			"agent_id":   "A3",
		},
	}
	err := handler(nil, evt)
	if err != nil {
		t.Fatalf("subscriber handler: %v", err)
	}

	// No approval should be created for high confidence
	var count int64
	db.Model(&ApprovalRequest{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 approval requests for high confidence, got %d", count)
	}
}

func TestNewAgentDecisionSubscriber_LowConfidence(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	logger := dbtest.NewLogger(t)

	handler := NewAgentDecisionSubscriber(db, logger)
	evt := eventbus.Event{
		Topic: "agent.decided.A3.acos_analysis",
		Payload: map[string]interface{}{
			"confidence": 0.45,
			"agent_id":   "A3",
			"product_id": float64(42),
		},
	}
	err := handler(nil, evt)
	if err != nil {
		t.Fatalf("subscriber handler: %v", err)
	}

	var reqs []ApprovalRequest
	db.Find(&reqs)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 approval request, got %d", len(reqs))
	}
	if reqs[0].ProductID != 42 {
		t.Errorf("expected ProductID 42, got %d", reqs[0].ProductID)
	}
	if reqs[0].Requester != "A3" {
		t.Errorf("expected Requester A3, got %s", reqs[0].Requester)
	}
}

func TestNewAgentDecisionSubscriber_NoConfidence(t *testing.T) {
	db := dbtest.NewDB(t, &ApprovalRequest{})
	logger := dbtest.NewLogger(t)

	handler := NewAgentDecisionSubscriber(db, logger)
	evt := eventbus.Event{
		Topic: "agent.decided.A3.acos_analysis",
		Payload: map[string]interface{}{
			"agent_id": "A3",
		},
	}
	err := handler(nil, evt)
	if err != nil {
		t.Fatalf("subscriber handler: %v", err)
	}

	var count int64
	db.Model(&ApprovalRequest{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 with no confidence, got %d", count)
	}
}

func TestMapDecisionPointToRequestType(t *testing.T) {
	tests := []struct {
		decisionPoint string
		want          string
	}{
		{"stock_alert", "publish"},
		{"listing_optimize", "content_update"},
		{"price_review", "price_change"},
		{"profit_check", "price_change"},
		{"discount_risk_check", "price_change"},
		{"compliance_check", "content_update"},
		{"unknown", "publish"},
	}
	for _, tc := range tests {
		got := mapDecisionPointToRequestType(tc.decisionPoint)
		if got != tc.want {
			t.Errorf("mapDecisionPointToRequestType(%q) = %q, want %q", tc.decisionPoint, got, tc.want)
		}
	}
}

func TestService_FindApprovedByTarget(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ApprovalRequest{
		ProductID:   101,
		RequestType: "publish",
		Requester:   "A8",
		Status:      "approved",
		TargetType:  "listing_task",
		TargetID:    55,
		Reason:      "资料完整且利润达标",
	})

	req, err := svc.FindApprovedByTarget("listing_task", 55, "publish")
	if err != nil {
		t.Fatalf("FindApprovedByTarget: %v", err)
	}
	if req.TargetType != "listing_task" || req.TargetID != 55 || req.Status != "approved" {
		t.Fatalf("unexpected approval: %+v", req)
	}
}

func TestService_FindApprovedByTargetRejectsPending(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ApprovalRequest{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ApprovalRequest{
		ProductID:   101,
		RequestType: "publish",
		Requester:   "A8",
		Status:      "pending",
		TargetType:  "listing_task",
		TargetID:    55,
	})

	_, err := svc.FindApprovedByTarget("listing_task", 55, "publish")
	if err == nil {
		t.Fatal("expected no approved approval for pending request")
	}
}
