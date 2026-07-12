package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestRequestTypeMatchesAction_ExplicitMapping(t *testing.T) {
	tests := []struct {
		requestType string
		actionType  string
		want        bool
	}{
		{"publish", "auto_publish", true},
		{"publish", "listing_optimize", true},
		{"price_change", "price_update", true},
		{"permission", "permission_change", true},
		{"sourcing_1688_publish", "auto_publish", true},
		{"price", "price_update", false},
		{"unknown", "price_update", false},
	}

	for _, tt := range tests {
		t.Run(tt.requestType+"_"+tt.actionType, func(t *testing.T) {
			got := requestTypeMatchesAction(tt.requestType, tt.actionType)
			if got != tt.want {
				t.Fatalf("requestTypeMatchesAction(%q, %q) = %v, want %v", tt.requestType, tt.actionType, got, tt.want)
			}
		})
	}
}

func approvedHTTPFixture(t *testing.T) (*gin.Engine, *approval.ApprovalRequest, *gorm.DB, *int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &approval.ApprovalRequest{}, &approval.ApprovalExecution{})
	future := time.Now().Add(time.Hour)
	req := &approval.ApprovalRequest{RequestType: "publish", Requester: "owner", Status: approval.StatusApproved, TargetType: "listing", TargetID: 42, ExpiresAt: &future}
	if err := db.Create(req).Error; err != nil {
		t.Fatal(err)
	}
	calls := 0
	r := gin.New()
	r.Use(ApprovalRequired(db, zap.NewNop()))
	r.POST("/api/v1/listings/:id/publish", func(c *gin.Context) {
		calls++
		if c.Query("fail") == "true" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r, req, db, &calls
}

func executeApprovedHTTP(r *gin.Engine, approvalID int64, key, suffix string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings/42/publish"+suffix, nil)
	req.Header.Set("X-Approval-ID", strconv.FormatInt(approvalID, 10))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestApprovalRequiredConsumesSuccessfulHTTPApprovalOnce(t *testing.T) {
	r, approvalReq, db, calls := approvedHTTPFixture(t)
	if got := executeApprovedHTTP(r, approvalReq.ID, "http-publish-42", ""); got.Code != http.StatusOK {
		t.Fatalf("first execution status=%d body=%s", got.Code, got.Body.String())
	}
	if got := executeApprovedHTTP(r, approvalReq.ID, "http-publish-42", ""); got.Code != http.StatusConflict {
		t.Fatalf("same-key reexecution status=%d", got.Code)
	}
	if got := executeApprovedHTTP(r, approvalReq.ID, "http-publish-other", ""); got.Code != http.StatusConflict {
		t.Fatalf("different-key reuse status=%d", got.Code)
	}
	if *calls != 1 {
		t.Fatalf("handler calls=%d, want 1", *calls)
	}
	var execution approval.ApprovalExecution
	if err := db.First(&execution, "approval_id = ?", approvalReq.ID).Error; err != nil || execution.State != approval.ExecutionSucceeded {
		t.Fatalf("execution=%+v err=%v", execution, err)
	}
}

func TestApprovalRequiredFailedHTTPExecutionRetriesSameKey(t *testing.T) {
	r, approvalReq, _, calls := approvedHTTPFixture(t)
	if got := executeApprovedHTTP(r, approvalReq.ID, "http-retry-42", "?fail=true"); got.Code != http.StatusInternalServerError {
		t.Fatalf("failed execution status=%d", got.Code)
	}
	if got := executeApprovedHTTP(r, approvalReq.ID, "http-retry-42", ""); got.Code != http.StatusOK {
		t.Fatalf("same-key retry status=%d body=%s", got.Code, got.Body.String())
	}
	if *calls != 2 {
		t.Fatalf("handler calls=%d, want 2", *calls)
	}
}

func TestApprovalRequiredRejectsMissingIdempotencyKey(t *testing.T) {
	r, approvalReq, _, calls := approvedHTTPFixture(t)
	if got := executeApprovedHTTP(r, approvalReq.ID, "", ""); got.Code != http.StatusBadRequest {
		t.Fatalf("missing key status=%d", got.Code)
	}
	if *calls != 0 {
		t.Fatal("handler ran without idempotency key")
	}
}

func TestApprovalRequiredAllowsExplicitStandardMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &approval.ApprovalRequest{}, &approval.ApprovalExecution{})
	r := gin.New()
	r.Use(ApprovalRequired(db, zap.NewNop()))
	r.POST("/api/v1/xiao-q/messages", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/xiao-q/messages", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("standard mutation status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestApprovalRequiredFailsClosedForUnclassifiedMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &approval.ApprovalRequest{}, &approval.ApprovalExecution{})
	calls := 0
	r := gin.New()
	r.Use(ApprovalRequired(db, zap.NewNop()))
	r.POST("/api/v1/unclassified/write", func(c *gin.Context) { calls++; c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/unclassified/write", nil))
	if w.Code != http.StatusInternalServerError || calls != 0 {
		t.Fatalf("unclassified mutation status=%d calls=%d body=%s", w.Code, calls, w.Body.String())
	}
}

func TestApprovalRequiredUsesExplicitRouteTargetMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &approval.ApprovalRequest{}, &approval.ApprovalExecution{})
	future := time.Now().Add(time.Hour)
	req := &approval.ApprovalRequest{
		ProductID: 123, RequestType: "sourcing_1688_publish", Requester: "owner",
		Status: approval.StatusApproved, TargetType: "sourcing_publish_attempt",
		TargetID: 77, ExpiresAt: &future,
	}
	if err := db.Create(req).Error; err != nil {
		t.Fatal(err)
	}
	calls := 0
	r := gin.New()
	r.Use(ApprovalRequired(db, zap.NewNop()))
	r.POST("/api/v1/sourcing-1688/:id/publish-requests/:attemptId/execute", func(c *gin.Context) {
		calls++
		c.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/sourcing-1688/55/publish-requests/77/execute", nil)
	httpReq.Header.Set("X-Approval-ID", strconv.FormatInt(req.ID, 10))
	httpReq.Header.Set("Idempotency-Key", "publish-attempt-77")
	r.ServeHTTP(w, httpReq)
	if w.Code != http.StatusNoContent || calls != 1 {
		t.Fatalf("explicit target route status=%d calls=%d body=%s", w.Code, calls, w.Body.String())
	}
	var execution approval.ApprovalExecution
	if err := db.First(&execution, "approval_id = ?", req.ID).Error; err != nil {
		t.Fatal(err)
	}
	if execution.TargetType != "sourcing_publish_attempt" || !strings.Contains(execution.TargetID, "target=77") {
		t.Fatalf("execution binding = %+v", execution)
	}
}

func TestApprovalMatchesRequestIDs_TypedBinding(t *testing.T) {
	tests := []struct {
		name string
		req  approval.ApprovalRequest
		ids  requestIdentity
		want bool
	}{
		{
			name: "target id matches primary route id",
			req:  approval.ApprovalRequest{ProductID: 999, TargetID: 42},
			ids:  requestIdentity{PrimaryID: 42, TargetID: 42},
			want: true,
		},
		{
			name: "target id mismatch rejected",
			req:  approval.ApprovalRequest{TargetID: 42},
			ids:  requestIdentity{PrimaryID: 43, TargetID: 43},
			want: false,
		},
		{
			name: "product id cannot match unrelated primary id",
			req:  approval.ApprovalRequest{ProductID: 42},
			ids:  requestIdentity{PrimaryID: 42},
			want: false,
		},
		{
			name: "product id matches product identity",
			req:  approval.ApprovalRequest{ProductID: 42},
			ids:  requestIdentity{PrimaryID: 42, ProductID: 42},
			want: true,
		},
		{
			name: "entity id mismatch rejected even when target id matches",
			req:  approval.ApprovalRequest{TargetID: 42, EntityID: 77},
			ids:  requestIdentity{PrimaryID: 42, TargetID: 42, EntityID: 78},
			want: false,
		},
		{
			name: "declared id with no verifiable request id rejected",
			req:  approval.ApprovalRequest{TargetID: 42},
			ids:  requestIdentity{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := approvalMatchesRequestIDs(&tt.req, tt.ids)
			if got != tt.want {
				t.Fatalf("approvalMatchesRequestIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveRequestIDsReadsTypedIDsAndRestoresBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/listing/products/42/publish/3?target_id=55", strings.NewReader(`{"entityId":77,"sku_id":88}`))
	c.Params = gin.Params{
		{Key: "product_id", Value: "42"},
		{Key: "platform_id", Value: "3"},
	}

	ids := resolveRequestIDs(c, "listing")
	if ids.ProductID != 42 {
		t.Fatalf("ProductID = %d, want 42", ids.ProductID)
	}
	if ids.TargetID != 55 {
		t.Fatalf("TargetID = %d, want 55", ids.TargetID)
	}
	if ids.EntityID != 77 {
		t.Fatalf("EntityID = %d, want 77", ids.EntityID)
	}
	if ids.SkuID != 88 {
		t.Fatalf("SkuID = %d, want 88", ids.SkuID)
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if string(body) != `{"entityId":77,"sku_id":88}` {
		t.Fatalf("body was not restored, got %q", string(body))
	}
}
