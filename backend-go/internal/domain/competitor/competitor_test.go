package competitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/response"
)

func newService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &CompetitorProduct{}, &PriceSnapshot{})
	return NewService(db, dbtest.NewLogger(t))
}

// setupRouter creates a Gin engine with competitor routes.
func setupRouter(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &CompetitorProduct{}, &PriceSnapshot{})
	svc := NewService(db, dbtest.NewLogger(t))
	h := NewHandler(svc)
	r := gin.New()
	g := r.Group("/api/v1/competitors")
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
		g.POST("/:id/prices", h.RecordPrice)
		g.GET("/:id/prices", h.ListPrices)
		g.GET("/:id/trend", h.GetPriceTrend)
	}
	return r, svc
}

func TestCompetitor_CRUD(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	// Create
	cp, err := svc.Create(ctx, &CreateCompetitorInput{
		Name:     "Competitor A",
		Platform: "ozon",
		Category: "Electronics",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cp.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if cp.Platform != "ozon" {
		t.Fatalf("platform = %q, want ozon", cp.Platform)
	}

	// Get
	got, err := svc.GetByID(ctx, cp.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Competitor A" {
		t.Fatalf("name = %q, want Competitor A", got.Name)
	}

	// Update
	got.Name = "Competitor A Updated"
	if err := svc.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// List
	items, total, err := svc.List(ctx, 1, 20, "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	_ = items

	// Delete
	if err := svc.Delete(ctx, cp.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(ctx, cp.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestCompetitor_RecordPrice(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	cp, err := svc.Create(ctx, &CreateCompetitorInput{Name: "Test", Platform: "ozon"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	snapshot, err := svc.RecordPrice(ctx, cp.ID, &RecordPriceInput{
		Price:    99.99,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("RecordPrice: %v", err)
	}
	if snapshot.Price != 99.99 {
		t.Fatalf("price = %f, want 99.99", snapshot.Price)
	}

	prices, err := svc.ListPrices(ctx, cp.ID, 10)
	if err != nil {
		t.Fatalf("ListPrices: %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("expected 1 price, got %d", len(prices))
	}
}

func TestCompetitor_PriceTrend(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	cp, _ := svc.Create(ctx, &CreateCompetitorInput{Name: "Trend Test", Platform: "shopee"})
	svc.RecordPrice(ctx, cp.ID, &RecordPriceInput{Price: 100.00})
	svc.RecordPrice(ctx, cp.ID, &RecordPriceInput{Price: 110.00})
	svc.RecordPrice(ctx, cp.ID, &RecordPriceInput{Price: 105.00})

	trend, err := svc.GetPriceTrend(ctx, cp.ID)
	if err != nil {
		t.Fatalf("GetPriceTrend: %v", err)
	}
	if trend.MinPrice != 100.00 {
		t.Fatalf("min_price = %f, want 100.00", trend.MinPrice)
	}
	if trend.MaxPrice != 110.00 {
		t.Fatalf("max_price = %f, want 110.00", trend.MaxPrice)
	}
	if trend.AvgPrice != 105.00 {
		t.Fatalf("avg_price = %f, want 105.00", trend.AvgPrice)
	}
	if trend.CurrentPrice != 105.00 {
		t.Fatalf("current_price = %f, want 105.00", trend.CurrentPrice)
	}
}

func TestCompetitor_Handler(t *testing.T) {
	r, svc := setupRouter(t)
	ctx := context.Background()

	cp, _ := svc.Create(ctx, &CreateCompetitorInput{Name: "Handler Test", Platform: "lazada"})

	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/competitors", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("Get", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/competitors/"+dbtest.IToA(cp.ID), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("RecordPrice", func(t *testing.T) {
		body := `{"price":199.99,"currency":"USD"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/competitors/"+dbtest.IToA(cp.ID)+"/prices",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
		}
		var res response.Result
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if res.Code != 0 {
			t.Fatalf("code = %d, want 0", res.Code)
		}
	})
}

func TestCompetitor_CreateEmptyName(t *testing.T) {
	svc := newService(t)
	_, err := svc.Create(context.Background(), &CreateCompetitorInput{Name: "", Platform: "ozon"})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}
