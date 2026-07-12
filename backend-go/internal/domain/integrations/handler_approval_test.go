package integrations

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBoundHTTPApprovalIDUsesOnlyGateContext(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	bodyID := int64(42)
	if _, err := boundHTTPApprovalID(c, &bodyID); err == nil {
		t.Fatal("untrusted body approval was accepted")
	}
	c.Set("approval_id", int64(42))
	if got, err := boundHTTPApprovalID(c, &bodyID); err != nil || got != 42 {
		t.Fatalf("got=%d err=%v", got, err)
	}
	other := int64(43)
	if _, err := boundHTTPApprovalID(c, &other); err == nil {
		t.Fatal("mismatched body and header approvals were accepted")
	}
}
