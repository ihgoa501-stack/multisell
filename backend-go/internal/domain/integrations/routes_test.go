package integrations

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func TestIntegrationRoutesRetireLegacyWriteBack(t *testing.T) {
	for _, allowMock := range []bool{false, true} {
		t.Run(map[bool]string{false: "formal", true: "development"}[allowMock], func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			RegisterRoutes(router.Group("/api/v1"), dbtest.NewDB(t), zap.NewNop(), nil, allowMock)

			for _, route := range router.Routes() {
				switch route.Path {
				case "/api/v1/platform-integrations/publish-to-ozon",
					"/api/v1/platform-integrations/write-back",
					"/api/v1/platform-integrations/write-back/:ref-id/retry":
					t.Fatalf("retired route is still registered: %s %s", route.Method, route.Path)
				}
			}
		})
	}
}

func TestIntegrationMockSeedIsDevelopmentOnly(t *testing.T) {
	for _, tc := range []struct {
		name      string
		allowMock bool
		wantSeed  bool
	}{
		{name: "formal", allowMock: false, wantSeed: false},
		{name: "development", allowMock: true, wantSeed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			RegisterRoutes(router.Group("/api/v1"), dbtest.NewDB(t), zap.NewNop(), nil, tc.allowMock)

			found := false
			for _, route := range router.Routes() {
				if route.Method == http.MethodPost && route.Path == "/api/v1/platform-integrations/mock/seed" {
					found = true
				}
			}
			if found != tc.wantSeed {
				t.Fatalf("mock seed registered = %v, want %v", found, tc.wantSeed)
			}
		})
	}
}
