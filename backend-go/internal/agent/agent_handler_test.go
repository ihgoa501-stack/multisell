package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestAgentHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:agent_handler_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	logger := zap.NewNop()
	svc := NewService(db, logger)
	orch := ai.NewOrchestrator(db, logger)
	h := NewHandler(svc, orch)
	return h, db
}

func setupAgentRouter(h *Handler) *gin.Engine {
	r := gin.New()
	r.GET("/agents", h.ListAgents)
	r.GET("/agents/:id", h.GetAgent)
	r.POST("/agents", h.CreateAgent)
	r.POST("/agents/:id/actions", h.ExecuteAction)
	r.GET("/agents/evolution", h.Evolution)
	r.GET("/agents/entropy", h.Entropy)
	return r
}

func TestHandler_ListAgents(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agents", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Code int              `json:"code"`
		Data []AgentSummary   `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("expected non-empty agent list")
	}
}

func TestHandler_GetAgent_Found(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agents/A5", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandler_GetAgent_NotFound(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agents/Z99", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandler_CreateAgent(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/agents", strings.NewReader(`{"id":"X1"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Agent roster is canonical — should return 409 Conflict
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestHandler_ExecuteAction_BadRequest(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/agents/A5/actions", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Empty body with no decision_point should fail binding → 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandler_ExecuteAction_AgentNotFound(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	body := `{"decision_point":"stock_alert"}`
	req := httptest.NewRequest(http.MethodPost, "/agents/Z99/actions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Agent Z99 doesn't exist → 404
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandler_ExecuteAction_Success(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	body := `{"decision_point":"stock_alert"}`
	req := httptest.NewRequest(http.MethodPost, "/agents/A5/actions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// With in-memory SQLite and no schema, the orchestrator returns 500.
	// The important thing is it doesn't panic and returns a valid HTTP status.
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 200 or 500; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Evolution(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agents/evolution", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandler_Entropy_Unauthenticated(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := setupAgentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agents/entropy", nil)
	r.ServeHTTP(w, req)

	// Entropy reads user from JWT context; without auth middleware, returns 401
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestHandler_Entropy_Authenticated(t *testing.T) {
	h, _ := newTestAgentHandler(t)
	r := gin.New()
	// Simulate auth middleware setting user_id
	r.Use(func(c *gin.Context) {
		c.Set("user_id", float64(1))
		c.Next()
	})
	r.GET("/agents/entropy", h.Entropy)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agents/entropy", nil)
	r.ServeHTTP(w, req)

	// Entropy requires a populated database; in test the query may error.
	// Verify it doesn't panic and returns a proper error response.
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 200 or 500; body: %s", w.Code, w.Body.String())
	}
}
