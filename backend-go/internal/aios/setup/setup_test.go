package setup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/aios/runtime"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/platform/scheduler"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}
	t.Cleanup(func() { _ = logger.Sync() })
	return logger
}

// newGinEngine creates a Gin engine in test mode and returns the router group
// under /api/v1 that RegisterAIOSRoutes expects.
func newGinEngine() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api/v1")
	return r, rg
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestInitialize_AllComponentsNonNil verifies Initialize returns non-nil components.
func TestInitialize_AllComponentsNonNil(t *testing.T) {
	logger := newTestLogger(t)
	bus := eventbus.New(logger)

	cfg := Initialize(nil, bus, logger)
	if cfg == nil {
		t.Fatal("expected non-nil Config from Initialize")
	}

	tests := []struct {
		name string
		got  interface{}
	}{
		{"Runtime", cfg.Runtime},
		{"Bus", cfg.Bus},
		{"Registry", cfg.Registry},
		{"Guardrails", cfg.Guardrails},
		{"LLMGateway", cfg.LLMGateway},
		{"Memory", cfg.Memory},
		{"IPC", cfg.IPC},
		{"Pipeline", cfg.Pipeline},
		{"Observability", cfg.Observability},
	}
	for _, tt := range tests {
		if tt.got == nil {
			t.Errorf("Initialize: %s is nil", tt.name)
		}
	}

	if n := cfg.Registry.ToolCount(); n == 0 {
		t.Error("Initialize: ToolRegistry should have at least one tool registered")
	}
	if instances := cfg.Runtime.ListInstances(); len(instances) != 0 {
		t.Fatalf("legacy AIOS runtime registered %d agents", len(instances))
	}
}

// TestInitialize_WithDB ensures Initialize works with a nil *gorm.DB.
func TestInitialize_WithDB(t *testing.T) {
	logger := newTestLogger(t)
	bus := eventbus.New(logger)

	cfg := Initialize(nil, bus, logger)
	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}
	// DB may be nil since we passed nil — that's fine for bootstrap.
}

// TestRegisterAIOSRoutes_HealthEndpoint checks GET /api/v1/aios/health returns 200.
func TestRegisterAIOSRoutes_HealthEndpoint(t *testing.T) {
	logger := newTestLogger(t)
	bus := eventbus.New(logger)
	cfg := Initialize(nil, bus, logger)

	engine, rg := newGinEngine()
	RegisterAIOSRoutes(rg, cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/aios/health", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	expectedFields := []string{"status", "runtime", "tools", "guardrails", "agents", "observability"}
	for _, f := range expectedFields {
		if _, ok := body[f]; !ok {
			t.Errorf("response missing field %q", f)
		}
	}
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", body["status"])
	}
}

// TestRegisterAIOSRoutes_ToolsEndpoint checks GET /api/v1/aios/tools returns tool list.
func TestRegisterAIOSRoutes_ToolsEndpoint(t *testing.T) {
	logger := newTestLogger(t)
	bus := eventbus.New(logger)
	cfg := Initialize(nil, bus, logger)

	engine, rg := newGinEngine()
	RegisterAIOSRoutes(rg, cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/aios/tools", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if _, ok := body["tools"]; !ok {
		t.Error("response missing 'tools' field")
	}
	if _, ok := body["count"]; !ok {
		t.Error("response missing 'count' field")
	}
}

// TestRegisterAIOSRoutes_NoRouteConflict ensures AIOS routes coexist with
// existing routes without duplicate-route panics.
func TestRegisterAIOSRoutes_NoRouteConflict(t *testing.T) {
	logger := newTestLogger(t)
	bus := eventbus.New(logger)
	cfg := Initialize(nil, bus, logger)

	_, rg := newGinEngine()

	// Simulate domain route registrations like router.go.
	rg.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	rg.GET("/runtime/agents", func(c *gin.Context) { c.JSON(200, gin.H{}) })
	rg.POST("/commands", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	requirePanic(t, false, func() {
		RegisterAIOSRoutes(rg, cfg)
	})
}

// TestRegisterAIOSRoutes_AgentsEndpoint checks GET /api/v1/aios/runtime/agents.
func TestRegisterAIOSRoutes_AgentsEndpoint(t *testing.T) {
	logger := newTestLogger(t)
	bus := eventbus.New(logger)
	cfg := Initialize(nil, bus, logger)

	_ = cfg.Runtime.RegisterAgent(runtime.AgentManifest{
		ID:          "test-agent",
		Name:        "Test Agent",
		Squad:       "test",
		Version:     "1.0.0",
		Description: "agent for test",
		Triggers:    []runtime.TriggerDef{},
	})
	_ = cfg.Runtime.StartAgent("test-agent")

	engine, rg := newGinEngine()
	RegisterAIOSRoutes(rg, cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/aios/runtime/agents", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	agents, ok := body["agents"].([]interface{})
	if !ok {
		t.Fatalf("expected 'agents' array, got %T", body["agents"])
	}
	if len(agents) == 0 {
		t.Error("expected at least one agent in response")
	}
}

// TestSetupSchedulerAgentTriggers_NoPanic verifies the function does not panic.
func TestSetupSchedulerAgentTriggers_NoPanic(t *testing.T) {
	logger := newTestLogger(t)
	bus := eventbus.New(logger)
	rt := runtime.New(logger, bus)

	_ = rt.RegisterAgent(runtime.AgentManifest{
		ID:          "A5",
		Name:        "Stock Alert",
		Squad:       "fulfillment",
		Version:     "1.0.0",
		Description: "Monitors inventory levels",
		Triggers: []runtime.TriggerDef{
			{Type: "schedule", Interval: "15m", DecisionPoint: "stock_alert"},
		},
	})
	_ = rt.StartAgent("A5")

	sched := scheduler.New(bus, logger)

	requirePanic(t, false, func() {
		SetupSchedulerAgentTriggers(sched, rt, logger)
	})
}

// TestParseDuration verifies the interval string parser.
func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    bool
		wantDur int64
	}{
		{"15m", true, int64(15 * time.Minute)},
		{"1h", true, int64(time.Hour)},
		{"30s", true, int64(30 * time.Second)},
		{"6h", true, int64(6 * time.Hour)},
		{"", false, 0},
		{"invalid", false, 0},
		{"10", false, 0},
	}
	for _, tt := range tests {
		d, ok := parseDuration(tt.input)
		if ok != tt.want {
			t.Errorf("parseDuration(%q): got ok=%v, want %v", tt.input, ok, tt.want)
		}
		if ok && int64(d) != tt.wantDur {
			t.Errorf("parseDuration(%q): got %d ns, want %d ns", tt.input, int64(d), tt.wantDur)
		}
	}
}

// requirePanic verifies fn does (or does not) panic.
func requirePanic(t *testing.T, expectPanic bool, fn func()) {
	t.Helper()
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		fn()
	}()
	if panicked != expectPanic {
		t.Errorf("requirePanic(expectPanic=%v): panicked=%v", expectPanic, panicked)
	}
}

// Import guard for gorm used in Initialize signature.
var _ = (*gorm.DB)(nil)
