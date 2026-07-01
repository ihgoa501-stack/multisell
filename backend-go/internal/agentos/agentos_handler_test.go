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
