package httpx

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func TestAllowDevelopmentFixtures(t *testing.T) {
	for _, tc := range []struct {
		name        string
		environment string
		want        bool
	}{
		{name: "development", environment: "development", want: true},
		{name: "acceptance", environment: "acceptance", want: false},
		{name: "production", environment: "production", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{Server: config.ServerConfig{DeploymentEnvironment: tc.environment}}
			if got := allowDevelopmentFixtures(cfg); got != tc.want {
				t.Fatalf("allowDevelopmentFixtures() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFixtureRoutesAreDevelopmentOnly(t *testing.T) {
	for _, tc := range []struct {
		name  string
		allow bool
		want  bool
	}{
		{name: "acceptance_or_production", allow: false, want: false},
		{name: "development", allow: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			registerDevelopmentFixtureRoutes(router.Group("/api/v1"), dbtest.NewDB(t), zap.NewNop(), tc.allow)

			foundMockSeed := false
			foundOwner := false
			for _, route := range router.Routes() {
				if route.Method == http.MethodPost && route.Path == "/api/v1/mock/seed" {
					foundMockSeed = true
				}
				if route.Method == http.MethodGet && route.Path == "/api/v1/owner/risk-summary" {
					foundOwner = true
				}
			}
			if foundMockSeed != tc.want || foundOwner != tc.want {
				t.Fatalf("fixture routes: mock seed=%v owner=%v, want both %v", foundMockSeed, foundOwner, tc.want)
			}
		})
	}
}
