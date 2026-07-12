package platformtruth

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/integrationtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func registerForIntegration(rg *gin.RouterGroup, _ *gorm.DB, _ *zap.Logger) {
	RegisterRoutes(rg)
}

func TestPlatformTruthRequiresOwnerAuthentication(t *testing.T) {
	ts := integrationtest.NewTestServer(t, registerForIntegration)
	defer ts.Close()

	resp := ts.Get(t, "/api/v1/platform-truth", "")
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", resp.StatusCode)
	}

	resp = ts.Get(t, "/api/v1/platform-truth", ts.Login(t))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticated status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Code int      `json:"code"`
		Data Contract `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || len(body.Data.DomainDispositions) == 0 {
		t.Fatalf("invalid contract response: code=%d domains=%d", body.Code, len(body.Data.DomainDispositions))
	}
}
