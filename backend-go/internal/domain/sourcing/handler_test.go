package sourcing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
)

func setupHandlerTest(t *testing.T) (*Service, *Handler, *mockToolBridge) {
	t.Helper()
	db := dbtest.NewDB(t, &sourcing1688.Sourcing1688Product{})
	bridge := &mockToolBridge{
		pageData: &PageData{
			SourceURL:    "https://detail.1688.com/offer/handler-test.html",
			Title:        "Handler Test Product with Adequate Title",
			Price:        75.0,
			Images:       []string{"img1.jpg", "img2.jpg", "img3.jpg"},
			SupplierName: "Handler Supplier",
		},
	}
	svc := NewService(db, dbtest.NewLogger(t), bridge, nil)
	h := NewHandler(svc)
	return svc, h, bridge
}

func TestHandler_Fetch_Success(t *testing.T) {
	t.Parallel()
	_, h, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"url": "https://detail.1688.com/offer/handler-test.html"}`
	c.Request = httptest.NewRequest("POST", "/sourcing/fetch", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Fetch(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Data["title"] != "Handler Test Product with Adequate Title" {
		t.Errorf("unexpected title: %v", resp.Data["title"])
	}
	score, ok := resp.Data["score"].(float64)
	if !ok || score < 1 {
		t.Errorf("expected valid score, got %v", resp.Data["score"])
	}
}

func TestHandler_Fetch_MissingURL(t *testing.T) {
	t.Parallel()
	_, h, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{}`
	c.Request = httptest.NewRequest("POST", "/sourcing/fetch", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Fetch(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Fetch_BridgeError(t *testing.T) {
	t.Parallel()
	_, h, bridge := setupHandlerTest(t)
	bridge.err = context.DeadlineExceeded

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"url": "https://detail.1688.com/offer/fail.html"}`
	c.Request = httptest.NewRequest("POST", "/sourcing/fetch", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Fetch(c)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ListRecommendations_Empty(t *testing.T) {
	t.Parallel()
	_, h, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sourcing/recommendations", nil)

	h.ListRecommendations(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Code    int              `json:"code"`
		Message string           `json:"message"`
		Data    []Recommendation `json:"data"`
		Total   int64            `json:"total"`
		Page    int              `json:"page"`
		Size    int              `json:"size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

func TestHandler_ListRecommendations_WithData(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &sourcing1688.Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil)

	// Pre-populate some recommendations directly in DB.
	products := []sourcing1688.Sourcing1688Product{
		{SourceURL: "https://example.com/1", SupplierName: "S1", Price1688: 10.0, ImageURL: "img1.jpg", Status: "recommended"},
		{SourceURL: "https://example.com/2", SupplierName: "S2", Price1688: 20.0, ImageURL: "img2.jpg", Status: "pending"},
	}
	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("create product: %v", err)
		}
	}

	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sourcing/recommendations", nil)

	h.ListRecommendations(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Code  int              `json:"code"`
		Data  []Recommendation `json:"data"`
		Total int64            `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Data))
	}
}
