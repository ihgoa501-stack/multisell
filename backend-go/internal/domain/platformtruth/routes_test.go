package platformtruth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPlatformTruthRouteIsReadOnlyAndReturnsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	RegisterRoutes(g)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/platform-truth", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body=%s", w.Code, w.Body.String())
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, "/api/v1/platform-truth", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s unexpectedly available: status=%d", method, w.Code)
		}
	}
}
