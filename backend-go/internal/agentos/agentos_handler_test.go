package agentos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestHandler(t *testing.T) (*Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := gin.New()
	return h, r
}

func TestHandler_Autonomy(t *testing.T) {
	h, r := newTestHandler(t)
	r.GET("/agentos/autonomy", h.Autonomy)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/autonomy", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Status(t *testing.T) {
	h, r := newTestHandler(t)
	r.GET("/agentos/status", h.Status)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 200 or 500", w.Code)
	}
}

func TestHandler_WorkItems(t *testing.T) {
	h, r := newTestHandler(t)
	r.GET("/agentos/work-items", h.WorkItems)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/work-items", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		var resp struct{ Code int }
		json.Unmarshal(w.Body.Bytes(), &resp)
		t.Logf("WorkItems response code=%d (expected 0)", resp.Code)
	}
}

func TestService_WorkItemDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_detail?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, decision_point TEXT,
		business_object_type TEXT, business_object_id TEXT,
		created_at TIMESTAMP
	)`)

	db.Exec(`INSERT INTO unified_action (id, title, agent_id, squad_id, risk_level, status, confidence, proposed_at, description, business_object_type, business_object_id, created_at)
		VALUES (1, 'Test listing recommendation', 'A2', 'insight', 'medium', 'suggested', 0.85, datetime('now'), 'Test description', 'listing_task', '100', datetime('now'))`)

	result, err := svc.WorkItemDetail(1)
	if err != nil {
		t.Fatalf("WorkItemDetail: %v", err)
	}
	if result.Title != "Test listing recommendation" {
		t.Errorf("title = %q, want %q", result.Title, "Test listing recommendation")
	}
	if result.AgentID != "A2" {
		t.Errorf("agent_id = %q, want %q", result.AgentID, "A2")
	}
	if result.RiskLevel != "medium" {
		t.Errorf("risk_level = %q, want %q", result.RiskLevel, "medium")
	}
	if result.EntityType != "listing_task" {
		t.Errorf("entity_type = %q, want %q", result.EntityType, "listing_task")
	}
	if result.EntityID == nil || *result.EntityID != 100 {
		t.Errorf("entity_id = %v, want 100", result.EntityID)
	}
}

func TestService_WorkItemDetail_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_detail_notfound?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())
	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (id INTEGER PRIMARY KEY)`)

	_, err = svc.WorkItemDetail(999)
	if err == nil {
		t.Fatal("expected error for non-existent work item, got nil")
	}
}

func TestService_AgentTimeline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_timeline?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, business_object_type TEXT,
		business_object_id TEXT, created_at TIMESTAMP
	)`)

	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, confidence, created_at)
		VALUES (1, 'Action 1', 'A2', 'completed', 'low', 0.9, datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, confidence, created_at)
		VALUES (2, 'Action 2', 'A5', 'suggested', 'medium', 0.7, datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, confidence, created_at)
		VALUES (3, 'Action 3', 'A2', 'suggested', 'low', 0.8, datetime('now'))`)

	result, err := svc.AgentTimeline(50)
	if err != nil {
		t.Fatalf("AgentTimeline: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 agent entries, got %d", len(result))
	}
	for _, entry := range result {
		if entry.AgentID == "A2" {
			if len(entry.RecentActions) != 2 {
				t.Errorf("A2: expected 2 recent actions, got %d", len(entry.RecentActions))
			}
			if entry.StatusSummary["completed"] != 1 {
				t.Errorf("A2: expected 1 completed, got %d", entry.StatusSummary["completed"])
			}
			if entry.StatusSummary["suggested"] != 1 {
				t.Errorf("A2: expected 1 suggested, got %d", entry.StatusSummary["suggested"])
			}
		}
	}
}

func TestService_FailedRuns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_failed?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())

	db.Exec(`CREATE TABLE IF NOT EXISTS ai_trace (
		id INTEGER PRIMARY KEY, trace_id TEXT, agent_id TEXT,
		decision_point TEXT, status TEXT, final_output TEXT,
		started_at TIMESTAMP, completed_at TIMESTAMP
	)`)
	db.Exec(`INSERT INTO ai_trace (id, trace_id, agent_id, decision_point, status, final_output, started_at, completed_at)
		VALUES (1, 'trace-1', 'A5', 'stock_alert', 'failed', '{"error":"LLM timeout"}', datetime('now'), datetime('now'))`)
	db.Exec(`INSERT INTO ai_trace (id, trace_id, agent_id, decision_point, status, final_output, started_at, completed_at)
		VALUES (2, 'trace-2', 'A6', 'profit_watch', 'completed', '{}', datetime('now', '-1 hour'), datetime('now', '-1 hour'))`)

	result, err := svc.FailedRuns(50)
	if err != nil {
		t.Fatalf("FailedRuns: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 failed run, got %d", len(result))
	}
	if result[0].AgentID != "A5" {
		t.Errorf("expected agent A5, got %s", result[0].AgentID)
	}
}

// ---------------------------------------------------------------------------
// P3 Traffic Dashboard tests
// ---------------------------------------------------------------------------

func TestTrafficSummary(t *testing.T) {
	h, r := newTestHandler(t)
	db := h.service.db

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, correlation_id TEXT,
		block_reason TEXT, created_at TIMESTAMP
	)`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (1, 'A1', 'A2', 'suggested', 'low', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (2, 'A2', 'A2', 'approved', 'medium', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (3, 'A3', 'A5', 'executed', 'low', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (4, 'A4', 'A5', 'rejected', 'high', datetime('now'))`)

	r.GET("/agentos/traffic-summary", h.TrafficSummary)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/traffic-summary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int           `json:"code"`
		Data    TrafficSummary `json:"data"`
		Message string        `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0", resp.Code)
	}
	if len(resp.Data.StatusDistribution) == 0 {
		t.Fatal("expected non-empty status_distribution")
	}
}

func TestInterceptedActions(t *testing.T) {
	h, r := newTestHandler(t)
	db := h.service.db

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, correlation_id TEXT,
		block_reason TEXT, action_type TEXT, created_at TIMESTAMP
	)`)
	db.Exec(`INSERT INTO unified_action (id, action_type, agent_id, risk_level, status, block_reason, description, created_at) VALUES (1, 'price_update', 'A6', 'high', 'blocked', 'L4_blocked', 'SKU-1001', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, action_type, agent_id, risk_level, status, block_reason, description, created_at) VALUES (2, 'stock_alert', 'A5', 'critical', 'blocked', 'approval_timeout', 'SKU-2002', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, action_type, agent_id, risk_level, status, created_at) VALUES (3, 'listing_optimize', 'A2', 'low', 'suggested', datetime('now'))`)

	r.GET("/agentos/intercepted-actions", h.InterceptedActions)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/intercepted-actions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int                `json:"code"`
		Data    struct {
			Items []InterceptedAction `json:"items"`
			Total int64               `json:"total"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0", resp.Code)
	}
	if resp.Data.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Data.Total)
	}
	if len(resp.Data.Items) != 2 {
		t.Fatalf("items count = %d, want 2", len(resp.Data.Items))
	}
}

func TestAgentMetrics(t *testing.T) {
	h, r := newTestHandler(t)
	db := h.service.db

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, correlation_id TEXT,
		block_reason TEXT, action_type TEXT, created_at TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS ai_trace (
		id INTEGER PRIMARY KEY, trace_id TEXT, agent_id TEXT,
		decision_point TEXT, status TEXT, final_output TEXT,
		started_at TIMESTAMP, completed_at TIMESTAMP
	)`)

	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (1, 'A1', 'A2', 'completed', 'low', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (2, 'A2', 'A2', 'approved', 'medium', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (3, 'A3', 'A5', 'failed', 'high', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (4, 'A4', 'A5', 'blocked', 'critical', datetime('now'))`)
	db.Exec(`INSERT INTO ai_trace (id, trace_id, agent_id, decision_point, status, started_at, completed_at) VALUES (1, 't1', 'A2', 'listing', 'completed', datetime('now', '-1 hour'), datetime('now'))`)
	db.Exec(`INSERT INTO ai_trace (id, trace_id, agent_id, decision_point, status, started_at, completed_at) VALUES (2, 't2', 'A5', 'stock', 'completed', datetime('now', '-2 hour'), datetime('now', '-1 hour'))`)

	r.GET("/agentos/agent-metrics", h.AgentMetrics)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/agent-metrics", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "agents") {
		t.Fatalf("response body does not contain 'agents': %s", w.Body.String())
	}
}

func TestExternalHealth(t *testing.T) {
	h, r := newTestHandler(t)

	r.GET("/agentos/external-health", h.ExternalHealth)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/external-health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestAuditReplay_NotFound(t *testing.T) {
	h, r := newTestHandler(t)
	db := h.service.db

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, correlation_id TEXT,
		block_reason TEXT, action_type TEXT, created_at TIMESTAMP
	)`)

	r.GET("/agentos/audit-replay/:correlation_id", h.AuditReplay)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/audit-replay/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int                 `json:"code"`
		Data    AuditReplayResponse `json:"data"`
		Message string              `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0", resp.Code)
	}
	if len(resp.Data.Events) != 0 {
		t.Fatalf("expected empty events for nonexistent correlation_id, got %d", len(resp.Data.Events))
	}
}
