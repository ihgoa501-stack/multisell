package aftersales

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLegacyDisputeHTTPFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(method, "/api/v1/aftersales/disputes/1", nil)
			legacyDisputeDisabled(c)
			if w.Code != http.StatusPreconditionRequired {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
