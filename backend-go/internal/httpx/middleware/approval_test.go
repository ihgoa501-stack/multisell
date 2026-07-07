package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
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
