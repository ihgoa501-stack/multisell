package producthub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newMasterService(t *testing.T) *MasterService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{})
	return NewMasterService(db, zap.NewNop())
}

func setupMasterRouter(t *testing.T) (*gin.Engine, *MasterService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := newMasterService(t)
	h := NewMasterHandler(svc)
	r := gin.New()
	rg := r.Group("/api/v1/product-hub")
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.POST("", h.Create)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
	rg.POST("/:id/transition", h.TransitionLifecycle)
	return r, svc
}

func TestMasterCreateAndGet(t *testing.T) {
	r, _ := setupMasterRouter(t)

	body := `{"name":"Test Product","owner_id":1,"business_model":"catalog","target_market":"US"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/product-hub", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Data["name"] != "Test Product" {
		t.Fatalf("expected name 'Test Product', got '%v'", resp.Data["name"])
	}
	if resp.Data["lifecycle_status"] != LifecycleIdea {
		t.Fatalf("expected default lifecycle '%s', got '%v'", LifecycleIdea, resp.Data["lifecycle_status"])
	}
}

func TestMasterLifecycleTransition(t *testing.T) {
	r, svc := setupMasterRouter(t)
	ctx := t.Context()

	p := &ProductMaster{Name: "Lifecycle Test", OwnerID: 1}
	if err := svc.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	// Transition idea -> researching
	body := `{"status":"researching"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/product-hub/%d/transition", p.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for idea->researching, got %d: %s", w.Code, w.Body.String())
	}

	// Transition researching -> sampling
	body = `{"status":"sampling"}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/product-hub/%d/transition", p.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for researching->sampling, got %d: %s", w.Code, w.Body.String())
	}

	p2, _ := svc.GetByID(ctx, p.ID)
	if p2.LifecycleStatus != LifecycleSampling {
		t.Fatalf("expected '%s', got '%s'", LifecycleSampling, p2.LifecycleStatus)
	}
}
