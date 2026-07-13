package productanalysis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestLegacyPrismTriggerIsNotRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	engine := gin.New()
	RegisterRoutes(engine.Group("/api/v1"), db, dbtest.NewLogger(t))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/product-analysis/trigger-prism", strings.NewReader(`{"image_url":"http://169.254.169.254/latest/meta-data","platform":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
