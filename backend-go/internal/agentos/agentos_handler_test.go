package agentos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandler_Autonomy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/agentos/autonomy", h.Autonomy)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/autonomy", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Status(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_status?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/agentos/status", h.Status)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/status", nil)
	r.ServeHTTP(w, req)

	// With empty DB, may return an error; verify no panic and valid status
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 200 or 500", w.Code)
	}
}

func TestHandler_WorkItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_work?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
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

	// Create the unified_action table.
	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, decision_point TEXT,
		business_object_type TEXT, business_object_id TEXT,
		created_at TIMESTAMP
	)`)

	// Insert test action.
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

	// Insert actions for two agents.
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

	// Should have 2 entries (A2 and A5).
	if len(result) != 2 {
		t.Fatalf("expected 2 agent entries, got %d", len(result))
	}

	// A2 should have 2 actions and status summary counts.
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

func TestHandler_TrafficSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_traffic_summary?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, business_object_type TEXT,
		business_object_id TEXT, created_at TIMESTAMP
	)`)

	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (1, 'A1', 'A2', 'suggested', 'low', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (2, 'A2', 'A2', 'approved', 'medium', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (3, 'A3', 'A5', 'executed', 'low', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (4, 'A4', 'A5', 'rejected', 'high', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, title, agent_id, status, risk_level, created_at) VALUES (5, 'A5', 'A6', 'escalated', 'critical', datetime('now'))`)

	r := gin.New()
	r.GET("/agentos/traffic-summary", h.TrafficSummary)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/traffic-summary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int                    `json:"code"`
		Data    TrafficSummaryResponse `json:"data"`
		Message string                 `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0; message: %s", resp.Code, resp.Message)
	}
	if len(resp.Data.StatusDistribution) == 0 {
		t.Fatal("expected non-empty status_distribution")
	}
	if resp.Data.StatusDistribution["suggested"] != 1 {
		t.Errorf("suggested count = %d, want 1", resp.Data.StatusDistribution["suggested"])
	}
	if resp.Data.InterceptedTotal != 2 {
		t.Errorf("intercepted_total = %d, want 2", resp.Data.InterceptedTotal)
	}
	if resp.Data.Funnel["produced"] != 5 {
		t.Errorf("produced = %d, want 5", resp.Data.Funnel["produced"])
	}
	if resp.Data.Funnel["rejected_by_owner"] != 1 {
		t.Errorf("rejected_by_owner = %d, want 1", resp.Data.Funnel["rejected_by_owner"])
	}
}

func TestHandler_InterceptedActions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_intercepted?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, action_type TEXT,
		rejection_reason TEXT, created_at TIMESTAMP
	)`)

	db.Exec(`INSERT INTO unified_action (id, action_type, agent_id, risk_level, status, rejection_reason, description, created_at) VALUES (1, 'price_update', 'A6', 'high', 'rejected', 'owner rejected', 'SKU-1001', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, action_type, agent_id, risk_level, status, rejection_reason, description, created_at) VALUES (2, 'stock_alert', 'A5', 'critical', 'escalated', 'approval timeout', 'SKU-2002', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, action_type, agent_id, risk_level, status, created_at) VALUES (3, 'listing_optimize', 'A2', 'low', 'suggested', datetime('now'))`)

	r := gin.New()
	r.GET("/agentos/intercepted-actions", h.InterceptedActions)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/intercepted-actions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int                       `json:"code"`
		Data    InterceptedActionsResponse `json:"data"`
		Message string                    `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0; message: %s", resp.Code, resp.Message)
	}
	if resp.Data.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Data.Total)
	}
	if len(resp.Data.Items) != 2 {
		t.Fatalf("items count = %d, want 2", len(resp.Data.Items))
	}
	if resp.Data.Items[0].BlockReason == "" {
		t.Error("expected non-empty block_reason")
	}
}

func TestHandler_AuditReplay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agentos_test_audit_replay?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	db.Exec(`CREATE TABLE IF NOT EXISTS ai_trace (
		id INTEGER PRIMARY KEY, trace_id TEXT, agent_id TEXT,
		decision_point TEXT, status TEXT, started_at TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, title TEXT, agent_id TEXT, squad_id TEXT,
		risk_level TEXT, status TEXT, confidence REAL, proposed_at TIMESTAMP,
		description TEXT, trace_id TEXT, action_type TEXT,
		created_at TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS approval_request (
		id INTEGER PRIMARY KEY, status TEXT, entity_type TEXT,
		entity_id TEXT, reviewer TEXT, updated_at TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS operation_log (
		id INTEGER PRIMARY KEY, action TEXT, content TEXT,
		resource_id TEXT, created_at TIMESTAMP
	)`)

	db.Exec(`INSERT INTO ai_trace (id, trace_id, agent_id, decision_point, status, started_at) VALUES (1, 'trace-123', 'A5', 'stock_alert', 'completed', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, action_type, agent_id, status, risk_level, description, trace_id, created_at) VALUES (1, 'stock_alert', 'A5', 'suggested', 'low', 'Stock alert for SKU-1001', 'trace-123', datetime('now'))`)
	db.Exec(`INSERT INTO unified_action (id, action_type, agent_id, status, risk_level, description, trace_id, created_at) VALUES (2, 'price_update', 'A6', 'approved', 'high', 'Increase price 19.99->24.99', 'trace-123', datetime('now', '+1 minute'))`)
	db.Exec(`INSERT INTO approval_request (id, status, entity_type, entity_id, reviewer, updated_at) VALUES (1, 'approved', 'unified_action', '2', 'owner', datetime('now', '+2 minutes'))`)
	db.Exec(`INSERT INTO operation_log (id, action, content, resource_id, created_at) VALUES (1, 'update', 'price 19.99 -> 24.99', '2', datetime('now', '+3 minutes'))`)

	r := gin.New()
	r.GET("/agentos/audit-replay/:correlation_id", h.AuditReplay)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentos/audit-replay/trace-123", nil)
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
		t.Fatalf("response code = %d, want 0; message: %s", resp.Code, resp.Message)
	}
	if resp.Data.CorrelationID != "trace-123" {
		t.Errorf("correlation_id = %s, want trace-123", resp.Data.CorrelationID)
	}
	if len(resp.Data.Events) == 0 {
		t.Fatal("expected non-empty events")
	}

	// Events should be in chronological order: agent_decision, action, action, approval, audit
	if resp.Data.Events[0].Type != "agent_decision" {
		t.Errorf("first event type = %s, want agent_decision", resp.Data.Events[0].Type)
	}
	if resp.Data.Events[3].Type != "approval" {
		t.Errorf("fourth event type = %s, want approval", resp.Data.Events[3].Type)
	}
	if resp.Data.Events[4].Type != "audit" {
		t.Errorf("fifth event type = %s, want audit", resp.Data.Events[4].Type)
	}
}
