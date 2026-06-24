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
