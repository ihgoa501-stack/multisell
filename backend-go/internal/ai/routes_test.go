package ai

import (
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestLegacyRoutesAreReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterLegacyReadRoutes(r.Group("/api/v1"), nil, zap.NewNop())

	want := map[string]bool{
		"GET /api/v1/ai/traces":           true,
		"GET /api/v1/ai/actions":          true,
		"GET /api/v1/ai/traces/:trace_id": true,
		"GET /api/v1/ai/actions/:id":      true,
	}
	for _, route := range r.Routes() {
		delete(want, route.Method+" "+route.Path)
		if route.Method != "GET" {
			t.Fatalf("legacy AI mutation route still registered: %s %s", route.Method, route.Path)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing legacy audit routes: %v", want)
	}
}
