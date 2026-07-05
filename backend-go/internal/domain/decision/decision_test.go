package decision

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// --- helpers ---

func newTestDB(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &PreListingDecision{})
	return NewService(db, zap.NewNop())
}

func newTestHandler(t *testing.T) (*Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := newTestDB(t)
	h := NewHandler(svc)
	r := gin.New()

	// Test middleware that simulates JWT auth context.
	r.Use(func(c *gin.Context) {
		if user := c.GetHeader("X-Test-User"); user != "" {
			c.Set("username", user)
		} else {
			c.Set("username", "test-admin")
		}
		c.Set("user_id", int64(1))
	})

	// IMPORTANT: /summary MUST be registered before /:id
	r.GET("/api/v1/decision", h.List)
	r.GET("/api/v1/decision/summary", h.Summary)
	r.GET("/api/v1/decision/:id", h.Get)
	r.POST("/api/v1/decision", h.Create)
	r.PUT("/api/v1/decision/:id", h.Update)
	r.DELETE("/api/v1/decision/:id", h.Delete)
	r.POST("/api/v1/decision/:id/approve", h.Approve)
	r.POST("/api/v1/decision/:id/reject", h.Reject)

	return h, r
}

func int64Ptr(v int64) *int64       { return &v }
func strPtr(v string) *string       { return &v }
func float64Ptr(v float64) *float64 { return &v }

// decodeIntoResult unmarshals JSON bytes into a response.Result.
func decodeIntoResult(t *testing.T, body []byte) response.Result {
	t.Helper()
	var res response.Result
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("failed to decode response.Result: %v\nbody: %s", err, string(body))
	}
	return res
}

// decodeIntoPageResult unmarshals JSON bytes into a response.PageResult.
func decodeIntoPageResult(t *testing.T, body []byte) response.PageResult {
	t.Helper()
	var res response.PageResult
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("failed to decode response.PageResult: %v\nbody: %s", err, string(body))
	}
	return res
}

// extractDecision decodes an embedded PreListingDecision from the Data field
// of a response.Result.
func extractDecision(t *testing.T, res response.Result) PreListingDecision {
	t.Helper()
	data, err := json.Marshal(res.Data)
	if err != nil {
		t.Fatalf("failed to marshal Data back to JSON: %v", err)
	}
	var d PreListingDecision
	if err := json.Unmarshal(data, &d); err != nil {
		t.Fatalf("failed to unmarshal Data into PreListingDecision: %v\njson: %s", err, string(data))
	}
	return d
}

// =============================================================================
// Service Tests
// =============================================================================

// TestCreateDecision verifies that a minimal create sets expected defaults.
func TestCreateDecision(t *testing.T) {
	svc := newTestDB(t)

	in := &CreateInput{
		SkuID:          100,
		PlatformID:     int64Ptr(1),
		CountryCode:    "US",
		Recommendation: "list",
		Reasoning:      "Good margins",
		TraceID:        "trace-abc",
	}

	d, err := svc.Create(in)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if d.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}
	if d.SkuID != 100 {
		t.Fatalf("SkuID = %d, want 100", d.SkuID)
	}
	if d.PlatformID == nil || *d.PlatformID != 1 {
		t.Fatalf("PlatformID = %v, want 1", d.PlatformID)
	}
	if d.CountryCode != "US" {
		t.Fatalf("CountryCode = %q, want US", d.CountryCode)
	}
	if d.Recommendation != "list" {
		t.Fatalf("Recommendation = %q, want list", d.Recommendation)
	}
	if d.Reasoning != "Good margins" {
		t.Fatalf("Reasoning = %q, want 'Good margins'", d.Reasoning)
	}
	if d.TraceID != "trace-abc" {
		t.Fatalf("TraceID = %q, want trace-abc", d.TraceID)
	}

	// Default values
	if d.Status != "pending" {
		t.Fatalf("Status = %q, want pending", d.Status)
	}
	if d.DecisionPoint != "pre_listing" {
		t.Fatalf("DecisionPoint = %q, want pre_listing", d.DecisionPoint)
	}
	if d.RiskLevel != "medium" {
		t.Fatalf("RiskLevel = %q, want medium", d.RiskLevel)
	}

	// Timestamps
	if d.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
	if d.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt is zero")
	}
	// DecidedAt should be nil for a new decision
	if d.DecidedAt != nil {
		t.Fatal("DecidedAt should be nil for new decision")
	}
}

// TestCreateDecision_WithAllFields verifies every field is persisted.
func TestCreateDecision_WithAllFields(t *testing.T) {
	svc := newTestDB(t)

	rev := 1000.0
	cost := 500.0
	shipping := 100.0
	pf := 80.0
	payf := 20.0
	other := 10.0
	profit := 290.0
	margin := 29.0
	conf := 0.85

	in := &CreateInput{
		SkuID:                 101,
		PlatformID:            int64Ptr(2),
		CountryCode:           "DE",
		DecisionPoint:         "pre_listing",
		EstimatedRevenue:      &rev,
		EstimatedProductCost:  &cost,
		EstimatedShippingCost: &shipping,
		EstimatedPlatformFee:  &pf,
		EstimatedPaymentFee:   &payf,
		EstimatedOtherFee:     &other,
		EstimatedProfit:       &profit,
		ProfitMargin:          &margin,
		ConfidenceScore:       &conf,
		RiskLevel:             "low",
		Recommendation:        "list",
		Reasoning:             "Strong numbers across the board",
		Status:                "approved",
		TraceID:               "trace-xyz",
	}

	d, err := svc.Create(in)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if d.SkuID != 101 {
		t.Fatalf("SkuID = %d, want 101", d.SkuID)
	}
	if d.PlatformID == nil || *d.PlatformID != 2 {
		t.Fatalf("PlatformID = %v, want 2", d.PlatformID)
	}
	if d.EstimatedRevenue != rev {
		t.Fatalf("EstimatedRevenue = %f, want %f", d.EstimatedRevenue, rev)
	}
	if d.EstimatedProductCost != cost {
		t.Fatalf("EstimatedProductCost = %f, want %f", d.EstimatedProductCost, cost)
	}
	if d.EstimatedShippingCost != shipping {
		t.Fatalf("EstimatedShippingCost = %f, want %f", d.EstimatedShippingCost, shipping)
	}
	if d.EstimatedPlatformFee != pf {
		t.Fatalf("EstimatedPlatformFee = %f, want %f", d.EstimatedPlatformFee, pf)
	}
	if d.EstimatedPaymentFee != payf {
		t.Fatalf("EstimatedPaymentFee = %f, want %f", d.EstimatedPaymentFee, payf)
	}
	if d.EstimatedOtherFee != other {
		t.Fatalf("EstimatedOtherFee = %f, want %f", d.EstimatedOtherFee, other)
	}
	if d.EstimatedProfit != profit {
		t.Fatalf("EstimatedProfit = %f, want %f", d.EstimatedProfit, profit)
	}
	if d.ProfitMargin != margin {
		t.Fatalf("ProfitMargin = %f, want %f", d.ProfitMargin, margin)
	}
	if d.ConfidenceScore != conf {
		t.Fatalf("ConfidenceScore = %f, want %f", d.ConfidenceScore, conf)
	}
	if d.RiskLevel != "low" {
		t.Fatalf("RiskLevel = %q, want low", d.RiskLevel)
	}
	if d.Recommendation != "list" {
		t.Fatalf("Recommendation = %q, want list", d.Recommendation)
	}
	if d.Reasoning != "Strong numbers across the board" {
		t.Fatalf("Reasoning = %q, want 'Strong numbers across the board'", d.Reasoning)
	}
	if d.Status != "approved" {
		t.Fatalf("Status = %q, want approved", d.Status)
	}
	if d.TraceID != "trace-xyz" {
		t.Fatalf("TraceID = %q, want trace-xyz", d.TraceID)
	}
}

// TestGetDecision covers found and not-found cases.
func TestGetDecision(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateInput{SkuID: 201, Recommendation: "list", TraceID: "get-test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Found
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("ID = %d, want %d", got.ID, created.ID)
	}
	if got.SkuID != 201 {
		t.Fatalf("SkuID = %d, want 201", got.SkuID)
	}
	if got.Recommendation != "list" {
		t.Fatalf("Recommendation = %q, want list", got.Recommendation)
	}
	if got.TraceID != "get-test" {
		t.Fatalf("TraceID = %q, want get-test", got.TraceID)
	}

	// Not found
	_, err = svc.Get(99999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestListDecisions tests empty list, filtering, search, and pagination.
func TestListDecisions(t *testing.T) {
	svc := newTestDB(t)

	// --- Empty list ---
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}

	// --- Seed data ---
	_, _ = svc.Create(&CreateInput{SkuID: 1, Recommendation: "list", Status: "pending", RiskLevel: "low", Reasoning: "Good product A", TraceID: "t1"})
	_, _ = svc.Create(&CreateInput{SkuID: 2, Recommendation: "skip", Status: "rejected", RiskLevel: "high", Reasoning: "Bad product B", TraceID: "t2"})
	_, _ = svc.Create(&CreateInput{SkuID: 3, Recommendation: "list", Status: "approved", RiskLevel: "low", Reasoning: "Good product C", TraceID: "t3"})
	_, _ = svc.Create(&CreateInput{SkuID: 1, Recommendation: "list", Status: "pending", RiskLevel: "medium", Reasoning: "Another for sku 1", TraceID: "t4"})

	// --- List all (no filter) ---
	items, total, err = svc.List(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
	if len(items) != 4 {
		t.Fatalf("len(items) = %d, want 4", len(items))
	}
	// Results ordered by id DESC
	if items[0].TraceID != "t4" {
		t.Fatalf("first item TraceID = %q, want t4 (desc order)", items[0].TraceID)
	}

	// --- Filter by sku_id ---
	sku1 := int64Ptr(1)
	items, total, err = svc.List(&common.Pagination{Page: 1, Size: 20}, &ListFilter{SkuID: sku1})
	if err != nil {
		t.Fatalf("List by sku_id failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("sku_id filter total = %d, want 2", total)
	}

	// --- Filter by status ---
	items, total, err = svc.List(&common.Pagination{Page: 1, Size: 20}, &ListFilter{Status: "pending"})
	if err != nil {
		t.Fatalf("List by status failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("status filter total = %d, want 2", total)
	}

	// --- Filter by risk_level ---
	items, total, err = svc.List(&common.Pagination{Page: 1, Size: 20}, &ListFilter{RiskLevel: "low"})
	if err != nil {
		t.Fatalf("List by risk_level failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("risk_level filter total = %d, want 2", total)
	}

	// --- Filter by platform_id (non-matching) ---
	pid := int64Ptr(99)
	items, total, err = svc.List(&common.Pagination{Page: 1, Size: 20}, &ListFilter{PlatformID: pid})
	if err != nil {
		t.Fatalf("List by platform_id failed: %v", err)
	}
	if total != 0 {
		t.Fatalf("platform_id filter total = %d, want 0", total)
	}

	items, total, err = svc.List(&common.Pagination{Page: 2, Size: 2}, nil)
	if err != nil {
		t.Fatalf("List page 2 size 2 failed: %v", err)
	}
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// --- Pagination: page beyond total ---
	items, total, err = svc.List(&common.Pagination{Page: 10, Size: 20}, nil)
	if err != nil {
		t.Fatalf("List page beyond total failed: %v", err)
	}
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

// TestUpdateDecision tests partial updates, no-op, and not-found error.
func TestUpdateDecision(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateInput{
		SkuID:          301,
		Recommendation: "list",
		Reasoning:      "Original reasoning",
		CountryCode:    "US",
		TraceID:        "update-test",
		RiskLevel:      "medium",
		Status:         "pending",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// --- Partial update: recommendation and country ---
	cc := "DE"
	rec := "skip"
	updated, err := svc.Update(created.ID, &UpdateInput{
		CountryCode:    &cc,
		Recommendation: &rec,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.CountryCode != "DE" {
		t.Fatalf("CountryCode = %q, want DE", updated.CountryCode)
	}
	if updated.Recommendation != "skip" {
		t.Fatalf("Recommendation = %q, want skip", updated.Recommendation)
	}
	// Unchanged fields must remain
	if updated.SkuID != 301 {
		t.Fatalf("SkuID changed to %d, want 301", updated.SkuID)
	}
	if updated.TraceID != "update-test" {
		t.Fatalf("TraceID changed to %q, want update-test", updated.TraceID)
	}
	if updated.Reasoning != "Original reasoning" {
		t.Fatalf("Reasoning changed to %q, want 'Original reasoning'", updated.Reasoning)
	}
	if updated.RiskLevel != "medium" {
		t.Fatalf("RiskLevel changed to %q, want medium", updated.RiskLevel)
	}
	if updated.Status != "pending" {
		t.Fatalf("Status changed to %q, want pending", updated.Status)
	}

	// --- Partial update with financial fields ---
	revenue := 999.99
	profit := 123.45
	updated2, err := svc.Update(created.ID, &UpdateInput{
		EstimatedRevenue: &revenue,
		EstimatedProfit:  &profit,
	})
	if err != nil {
		t.Fatalf("Update (financial) failed: %v", err)
	}
	if updated2.EstimatedRevenue != 999.99 {
		t.Fatalf("EstimatedRevenue = %f, want 999.99", updated2.EstimatedRevenue)
	}
	if updated2.EstimatedProfit != 123.45 {
		t.Fatalf("EstimatedProfit = %f, want 123.45", updated2.EstimatedProfit)
	}
	if updated2.CountryCode != "DE" {
		t.Fatalf("CountryCode changed to %q, want DE", updated2.CountryCode)
	}

	// --- Partial update with risk level and status ---
	rl := "high"
	st := "approved"
	updated3, err := svc.Update(created.ID, &UpdateInput{
		RiskLevel: &rl,
		Status:    &st,
	})
	if err != nil {
		t.Fatalf("Update (risk+status) failed: %v", err)
	}
	if updated3.RiskLevel != "high" {
		t.Fatalf("RiskLevel = %q, want high", updated3.RiskLevel)
	}
	if updated3.Status != "approved" {
		t.Fatalf("Status = %q, want approved", updated3.Status)
	}

	// --- No-op update (all nil fields) ---
	noop, err := svc.Update(created.ID, &UpdateInput{})
	if err != nil {
		t.Fatalf("Update (no-op) failed: %v", err)
	}
	if noop.Recommendation != "skip" {
		t.Fatalf("after no-op Recommendation = %q, want skip", noop.Recommendation)
	}

	// --- Update non-existent record ---
	_, err = svc.Update(99999, &UpdateInput{CountryCode: strPtr("FR")})
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestDeleteDecision covers successful delete and not-found.
func TestDeleteDecision(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateInput{SkuID: 401, TraceID: "delete-test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// --- Successful deletion ---
	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it is gone
	_, err = svc.Get(created.ID)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound after delete, got %v", err)
	}

	// --- Delete non-existent record ---
	err = svc.Delete(99999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound for non-existent delete, got %v", err)
	}
}

// TestApproveDecision verifies approval sets status, decided_by, and decided_at.
func TestApproveDecision(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateInput{SkuID: 501, Status: "pending", TraceID: "approve-test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.DecidedAt != nil {
		t.Fatal("DecidedAt should be nil before approval")
	}
	if created.DecidedBy != "" {
		t.Fatalf("DecidedBy = %q, want empty before approval", created.DecidedBy)
	}

	// --- Approve ---
	approved, err := svc.Approve(created.ID, &ApproveInput{DecidedBy: "admin"})
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}
	if approved.Status != "approved" {
		t.Fatalf("Status = %q, want approved", approved.Status)
	}
	if approved.DecidedBy != "admin" {
		t.Fatalf("DecidedBy = %q, want admin", approved.DecidedBy)
	}
	if approved.DecidedAt == nil {
		t.Fatal("DecidedAt is nil; expected a timestamp after approval")
	}
	if time.Since(*approved.DecidedAt) > 5*time.Second {
		t.Fatalf("DecidedAt (%v) is more than 5s in the past", approved.DecidedAt)
	}
	if approved.DecidedAt.Before(created.CreatedAt) {
		t.Fatalf("DecidedAt (%v) is before CreatedAt (%v)", approved.DecidedAt, created.CreatedAt)
	}

	// --- Approve non-existent record ---
	_, err = svc.Approve(99999, &ApproveInput{DecidedBy: "admin"})
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestRejectDecision verifies rejection sets status, decided_by, decided_at,
// and stores the reason in reasoning.
func TestRejectDecision(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateInput{SkuID: 601, Status: "pending", TraceID: "reject-test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// --- Reject ---
	rejected, err := svc.Reject(created.ID, &RejectInput{DecidedBy: "manager", Reason: "Margins too thin"})
	if err != nil {
		t.Fatalf("Reject failed: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("Status = %q, want rejected", rejected.Status)
	}
	if rejected.DecidedBy != "manager" {
		t.Fatalf("DecidedBy = %q, want manager", rejected.DecidedBy)
	}
	if rejected.DecidedAt == nil {
		t.Fatal("DecidedAt is nil; expected a timestamp after rejection")
	}
	if time.Since(*rejected.DecidedAt) > 5*time.Second {
		t.Fatalf("DecidedAt (%v) is more than 5s in the past", rejected.DecidedAt)
	}
	if !strings.Contains(rejected.Reasoning, "Margins too thin") {
		t.Fatalf("Reasoning = %q, should contain 'Margins too thin'", rejected.Reasoning)
	}

	// --- Reject non-existent record ---
	_, err = svc.Reject(99999, &RejectInput{DecidedBy: "manager", Reason: "N/A"})
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestSummary verifies aggregation counts by recommendation, risk_level, status.
func TestSummary(t *testing.T) {
	svc := newTestDB(t)

	// --- Empty database ---
	sum, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary failed on empty db: %v", err)
	}
	if sum.Total != 0 {
		t.Fatalf("Total = %d, want 0", sum.Total)
	}
	if len(sum.ByRecommendation) != 0 {
		t.Fatalf("ByRecommendation = %v, want empty", sum.ByRecommendation)
	}
	if len(sum.ByRiskLevel) != 0 {
		t.Fatalf("ByRiskLevel = %v, want empty", sum.ByRiskLevel)
	}
	if len(sum.ByStatus) != 0 {
		t.Fatalf("ByStatus = %v, want empty", sum.ByStatus)
	}

	// --- Seed various decisions ---
	// 3 approved, low risk, recommend "list"
	for i := 0; i < 3; i++ {
		_, _ = svc.Create(&CreateInput{
			SkuID:          int64(700 + i),
			Recommendation: "list",
			RiskLevel:      "low",
			Status:         "approved",
			TraceID:        fmt.Sprintf("s1-%d", i),
		})
	}
	// 2 pending, medium risk, recommend "list"
	for i := 0; i < 2; i++ {
		_, _ = svc.Create(&CreateInput{
			SkuID:          int64(710 + i),
			Recommendation: "list",
			RiskLevel:      "medium",
			Status:         "pending",
			TraceID:        fmt.Sprintf("s2-%d", i),
		})
	}
	// 1 rejected, high risk, recommend "skip"
	_, _ = svc.Create(&CreateInput{
		SkuID:          720,
		Recommendation: "skip",
		RiskLevel:      "high",
		Status:         "rejected",
		TraceID:        "s3",
	})

	sum, err = svc.Summary()
	if err != nil {
		t.Fatalf("Summary failed: %v", err)
	}

	// Total
	if sum.Total != 6 {
		t.Fatalf("Total = %d, want 6", sum.Total)
	}

	// By recommendation
	if sum.ByRecommendation["list"] != 5 {
		t.Fatalf("ByRecommendation[list] = %d, want 5", sum.ByRecommendation["list"])
	}
	if sum.ByRecommendation["skip"] != 1 {
		t.Fatalf("ByRecommendation[skip] = %d, want 1", sum.ByRecommendation["skip"])
	}

	// By risk level
	if sum.ByRiskLevel["low"] != 3 {
		t.Fatalf("ByRiskLevel[low] = %d, want 3", sum.ByRiskLevel["low"])
	}
	if sum.ByRiskLevel["medium"] != 2 {
		t.Fatalf("ByRiskLevel[medium] = %d, want 2", sum.ByRiskLevel["medium"])
	}
	if sum.ByRiskLevel["high"] != 1 {
		t.Fatalf("ByRiskLevel[high] = %d, want 1", sum.ByRiskLevel["high"])
	}

	// By status
	if sum.ByStatus["approved"] != 3 {
		t.Fatalf("ByStatus[approved] = %d, want 3", sum.ByStatus["approved"])
	}
	if sum.ByStatus["pending"] != 2 {
		t.Fatalf("ByStatus[pending] = %d, want 2", sum.ByStatus["pending"])
	}
	if sum.ByStatus["rejected"] != 1 {
		t.Fatalf("ByStatus[rejected] = %d, want 1", sum.ByStatus["rejected"])
	}
}

// =============================================================================
// Handler Tests
// =============================================================================

// TestHandler_List verifies GET /decision returns a paginated response.
func TestHandler_List(t *testing.T) {
	h, r := newTestHandler(t)

	// --- Empty list ---
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decision", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	pr := decodeIntoPageResult(t, w.Body.Bytes())
	if pr.Code != 0 {
		t.Fatalf("code = %d, want 0; message: %s", pr.Code, pr.Message)
	}
	if pr.Total != 0 {
		t.Fatalf("Total = %d, want 0", pr.Total)
	}
	if pr.Page != 1 {
		t.Fatalf("Page = %d, want 1", pr.Page)
	}
	if pr.Size != 20 {
		t.Fatalf("Size = %d, want 20", pr.Size)
	}

	// --- Seed data via service ---
	_, _ = h.service.Create(&CreateInput{SkuID: 10, TraceID: "hl1"})
	_, _ = h.service.Create(&CreateInput{SkuID: 20, TraceID: "hl2"})

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/decision", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	pr = decodeIntoPageResult(t, w.Body.Bytes())
	if pr.Total != 2 {
		t.Fatalf("Total = %d, want 2", pr.Total)
	}

	// --- With query parameters ---
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/decision?page=1&size=1", nil)
	r.ServeHTTP(w, req)

	pr = decodeIntoPageResult(t, w.Body.Bytes())
	if pr.Total != 2 {
		t.Fatalf("Total = %d, want 2", pr.Total)
	}
	if pr.Size != 1 {
		t.Fatalf("Size = %d, want 1", pr.Size)
	}
}

// TestHandler_Get covers found, not-found, and invalid-ID cases.
func TestHandler_Get(t *testing.T) {
	h, r := newTestHandler(t)

	// Create a record to fetch
	created, err := h.service.Create(&CreateInput{SkuID: 801, TraceID: "handler-get"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// --- Found ---
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/decision/%d", created.ID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	res := decodeIntoResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("code = %d, want 0; message: %s", res.Code, res.Message)
	}
	d := extractDecision(t, res)
	if d.SkuID != 801 {
		t.Fatalf("SkuID = %d, want 801", d.SkuID)
	}
	if d.TraceID != "handler-get" {
		t.Fatalf("TraceID = %q, want handler-get", d.TraceID)
	}

	// --- Not found (valid id but no record) ---
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/decision/99999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}
	res = decodeIntoResult(t, w.Body.Bytes())
	if res.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", res.Code)
	}
	if res.Message != "decision not found" {
		t.Fatalf("message = %q, want 'decision not found'", res.Message)
	}

	// --- Invalid ID (non-numeric) ---
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/decision/abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
	res = decodeIntoResult(t, w.Body.Bytes())
	if res.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", res.Code)
	}
	if res.Message != "invalid id" {
		t.Fatalf("message = %q, want 'invalid id'", res.Message)
	}
}

// TestHandler_Create verifies POST /decision creates and returns the record.
func TestHandler_Create(t *testing.T) {
	_, r := newTestHandler(t)

	body := `{"sku_id":901,"platform_id":3,"country_code":"FR","recommendation":"list","reasoning":"Good opportunity","trace_id":"hc1"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/decision", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	res := decodeIntoResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("code = %d, want 0; message: %s", res.Code, res.Message)
	}
	d := extractDecision(t, res)
	if d.SkuID != 901 {
		t.Fatalf("SkuID = %d, want 901", d.SkuID)
	}
	if d.PlatformID == nil || *d.PlatformID != 3 {
		t.Fatalf("PlatformID = %v, want 3", d.PlatformID)
	}
	if d.CountryCode != "FR" {
		t.Fatalf("CountryCode = %q, want FR", d.CountryCode)
	}
	if d.Status != "pending" {
		t.Fatalf("Status = %q, want pending", d.Status)
	}
	if d.DecisionPoint != "pre_listing" {
		t.Fatalf("DecisionPoint = %q, want pre_listing", d.DecisionPoint)
	}
	if d.RiskLevel != "medium" {
		t.Fatalf("RiskLevel = %q, want medium", d.RiskLevel)
	}
	if d.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// --- Create with missing required field (sku_id) ---
	badBody := `{"recommendation":"list"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/decision", strings.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}

	// --- Create with invalid JSON ---
	badBody = `not-json`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/decision", strings.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

// TestHandler_Update verifies PUT /decision/:id.
func TestHandler_Update(t *testing.T) {
	h, r := newTestHandler(t)

	// Create a record to update
	created, err := h.service.Create(&CreateInput{
		SkuID:          1001,
		Recommendation: "list",
		CountryCode:    "US",
		Reasoning:      "Initial reasoning",
		TraceID:        "hu1",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// --- Successful update ---
	body := `{"recommendation":"skip","country_code":"GB","estimated_revenue":1500.50}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/decision/%d", created.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	res := decodeIntoResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("code = %d, want 0; message: %s", res.Code, res.Message)
	}
	d := extractDecision(t, res)
	if d.Recommendation != "skip" {
		t.Fatalf("Recommendation = %q, want skip", d.Recommendation)
	}
	if d.CountryCode != "GB" {
		t.Fatalf("CountryCode = %q, want GB", d.CountryCode)
	}
	if d.EstimatedRevenue != 1500.50 {
		t.Fatalf("EstimatedRevenue = %f, want 1500.50", d.EstimatedRevenue)
	}
	if d.TraceID != "hu1" {
		t.Fatalf("TraceID changed to %q, want hu1", d.TraceID)
	}

	// --- Update non-existent record ---
	body = `{"recommendation":"skip"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/decision/99999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}

	// --- Update with invalid ID ---
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/decision/abc", strings.NewReader(`{"recommendation":"skip"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}

	// --- Update with invalid JSON ---
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/decision/%d", created.ID), strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

// TestHandler_Delete verifies DELETE /decision/:id.
func TestHandler_Delete(t *testing.T) {
	h, r := newTestHandler(t)

	// Create a record to delete
	created, err := h.service.Create(&CreateInput{SkuID: 1101, TraceID: "hd1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// --- Successful delete ---
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/decision/%d", created.ID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	res := decodeIntoResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("code = %d, want 0; message: %s", res.Code, res.Message)
	}

	// Verify deletion via the service
	_, err = h.service.Get(created.ID)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound after delete, got %v", err)
	}

	// --- Delete non-existent record ---
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/decision/99999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}
	res = decodeIntoResult(t, w.Body.Bytes())
	if res.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", res.Code)
	}
	if res.Message != "decision not found" {
		t.Fatalf("message = %q, want 'decision not found'", res.Message)
	}

	// --- Delete with invalid ID ---
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/decision/abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

// TestHandler_Approve verifies POST /decision/:id/approve.
func TestHandler_Approve(t *testing.T) {
	h, r := newTestHandler(t)

	// Create a pending decision
	created, err := h.service.Create(&CreateInput{SkuID: 1201, Status: "pending", TraceID: "ha1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// --- Successful approval ---
	body := `{"decided_by":"admin"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/decision/%d/approve", created.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	res := decodeIntoResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("code = %d, want 0; message: %s", res.Code, res.Message)
	}
	d := extractDecision(t, res)
	if d.Status != "approved" {
		t.Fatalf("Status = %q, want approved", d.Status)
	}
	if d.DecidedBy != "test-admin" {
		t.Fatalf("DecidedBy = %q, want test-admin", d.DecidedBy)
	}
	if d.DecidedAt == nil {
		t.Fatal("DecidedAt is nil after approve")
	}

	// --- Approve non-existent ---
	body = `{"decided_by":"admin"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/decision/99999/approve", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}

	// --- Approve with invalid ID ---
	body = `{"decided_by":"admin"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/decision/abc/approve", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

// TestHandler_Reject verifies POST /decision/:id/reject.
func TestHandler_Reject(t *testing.T) {
	h, r := newTestHandler(t)

	// Create a pending decision
	created, err := h.service.Create(&CreateInput{SkuID: 1301, Status: "pending", TraceID: "hr1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// --- Successful rejection ---
	body := `{"decided_by":"manager","reason":"Not profitable enough"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/decision/%d/reject", created.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	res := decodeIntoResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("code = %d, want 0; message: %s", res.Code, res.Message)
	}
	d := extractDecision(t, res)
	if d.Status != "rejected" {
		t.Fatalf("Status = %q, want rejected", d.Status)
	}
	if d.DecidedBy != "test-admin" {
		t.Fatalf("DecidedBy = %q, want test-admin", d.DecidedBy)
	}
	if d.DecidedAt == nil {
		t.Fatal("DecidedAt is nil after reject")
	}
	if !strings.Contains(d.Reasoning, "Not profitable enough") {
		t.Fatalf("Reasoning = %q, should contain 'Not profitable enough'", d.Reasoning)
	}

	// --- Reject non-existent ---
	body = `{"decided_by":"manager","reason":"No"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/decision/99999/reject", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", w.Code, w.Body.String())
	}

	// --- Reject with invalid ID ---
	body = `{"decided_by":"manager","reason":"No"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/decision/abc/reject", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

// TestHandler_Summary verifies GET /decision/summary returns aggregated counts.
func TestHandler_Summary(t *testing.T) {
	h, r := newTestHandler(t)

	// --- Empty summary ---
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/decision/summary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	res := decodeIntoResult(t, w.Body.Bytes())
	rawData, _ := json.Marshal(res.Data)
	var emptySummary Summary
	if err := json.Unmarshal(rawData, &emptySummary); err != nil {
		t.Fatalf("failed to decode Summary: %v", err)
	}
	if emptySummary.Total != 0 {
		t.Fatalf("Total = %d, want 0", emptySummary.Total)
	}

	// --- Seed data ---
	_, _ = h.service.Create(&CreateInput{SkuID: 1401, Recommendation: "list", RiskLevel: "low", Status: "approved", TraceID: "hs1"})
	_, _ = h.service.Create(&CreateInput{SkuID: 1402, Recommendation: "list", RiskLevel: "medium", Status: "pending", TraceID: "hs2"})
	_, _ = h.service.Create(&CreateInput{SkuID: 1403, Recommendation: "skip", RiskLevel: "high", Status: "rejected", TraceID: "hs3"})

	// --- Verify summary is routed correctly (not caught by :id) ---
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/decision/summary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	res = decodeIntoResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("code = %d, want 0; message: %s", res.Code, res.Message)
	}

	rawData, _ = json.Marshal(res.Data)
	var summary Summary
	if err := json.Unmarshal(rawData, &summary); err != nil {
		t.Fatalf("failed to decode Summary: %v", err)
	}

	if summary.Total != 3 {
		t.Fatalf("Total = %d, want 3", summary.Total)
	}
	if summary.ByRecommendation["list"] != 2 {
		t.Fatalf("ByRecommendation[list] = %d, want 2", summary.ByRecommendation["list"])
	}
	if summary.ByRecommendation["skip"] != 1 {
		t.Fatalf("ByRecommendation[skip] = %d, want 1", summary.ByRecommendation["skip"])
	}
	if summary.ByRiskLevel["low"] != 1 {
		t.Fatalf("ByRiskLevel[low] = %d, want 1", summary.ByRiskLevel["low"])
	}
	if summary.ByRiskLevel["medium"] != 1 {
		t.Fatalf("ByRiskLevel[medium] = %d, want 1", summary.ByRiskLevel["medium"])
	}
	if summary.ByRiskLevel["high"] != 1 {
		t.Fatalf("ByRiskLevel[high] = %d, want 1", summary.ByRiskLevel["high"])
	}
	if summary.ByStatus["approved"] != 1 {
		t.Fatalf("ByStatus[approved] = %d, want 1", summary.ByStatus["approved"])
	}
	if summary.ByStatus["pending"] != 1 {
		t.Fatalf("ByStatus[pending] = %d, want 1", summary.ByStatus["pending"])
	}
	if summary.ByStatus["rejected"] != 1 {
		t.Fatalf("ByStatus[rejected] = %d, want 1", summary.ByStatus["rejected"])
	}
}

// TestRoutes_SummaryBeforeID verifies that /summary is not captured by /:id.
func TestRoutes_SummaryBeforeID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &PreListingDecision{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	// Correct order: /summary before /:id
	r := gin.New()
	r.GET("/decision", h.List)
	r.GET("/decision/summary", h.Summary)
	r.GET("/decision/:id", h.Get)

	// /summary should be handled by Summary handler, not Get
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/decision/summary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("/decision/summary returned status %d, want 200; body: %s", w.Code, w.Body.String())
	}

	res := decodeIntoResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("code = %d, want 0 (summary handler); message: %s", res.Code, res.Message)
	}
}
