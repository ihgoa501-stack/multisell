package httpx

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func TestRouterBuildsWithoutLegacyAgentRuntime(t *testing.T) {
	db := dbtest.NewDB(t)
	app := NewRouter(db, &config.Config{
		Server: config.ServerConfig{Mode: gin.TestMode, DeploymentEnvironment: "acceptance"},
		JWT:    config.JWTConfig{Secret: "test-secret"},
	}, zap.NewNop())
	defer func() {
		app.Cancel()
		app.Scheduler.Shutdown()
		app.Bus.Stop()
	}()

	allowedLegacyAuditRoutes := map[string]struct{}{
		"GET /api/v1/ai/traces":           {},
		"GET /api/v1/ai/traces/:trace_id": {},
		"GET /api/v1/ai/actions":          {},
		"GET /api/v1/ai/actions/:id":      {},
	}
	retiredPrefixes := []string{
		"/api/v1/actions", "/api/v1/content", "/api/v1/agent-upgrades",
		"/api/v1/agent-learning", "/api/v1/agent-rules", "/api/v1/agents",
		"/api/v1/agentos", "/api/v1/entropy", "/api/v1/evolution",
		"/api/v1/metabolism", "/api/v1/orchestration", "/api/v1/trust-scores",
		"/api/v1/settings/llm", "/api/v1/sourcing/",
	}
	for _, route := range app.Engine.Routes() {
		if strings.HasPrefix(route.Path, "/api/v1/ai/") {
			if _, ok := allowedLegacyAuditRoutes[route.Method+" "+route.Path]; !ok {
				t.Fatalf("legacy AI runtime route registered: %s %s", route.Method, route.Path)
			}
		}
		for _, prefix := range retiredPrefixes {
			if strings.HasPrefix(route.Path, prefix) {
				t.Fatalf("retired Agent subsystem route registered: %s %s", route.Method, route.Path)
			}
		}
	}

	for _, task := range app.Scheduler.RegisteredTasks() {
		id := task.AgentID
		legacyNumberedAgent := len(id) > 1 && (id[0] == 'A' || id[0] == 'G') && id[1] >= '0' && id[1] <= '9'
		if legacyNumberedAgent || id == "M1" || id == "trustscore" || id == "entropy" || id == "agentos" || id == "orch" {
			t.Fatalf("legacy Agent scheduled task registered: %s (%s)", task.ID, task.AgentID)
		}
	}
}
