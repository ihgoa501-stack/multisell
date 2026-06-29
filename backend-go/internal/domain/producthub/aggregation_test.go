package producthub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func TestAggregationAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductVariant{}, &ProductConcept{}, &SupplierOffer{}, &SampleRequest{}, &CostVersion{})
	aggr := NewAggregationService(db, zap.NewNop())
	h := NewHubHandler(aggr)

	// Create a product with data.
	ctx := t.Context()
	master := &ProductMaster{Name: "Aggregated Product", OwnerID: 1}
	if err := aggr.master.Create(ctx, master); err != nil {
		t.Fatal(err)
	}

	// Add a variant.
	if err := aggr.variant.Create(ctx, &ProductVariant{ProductMasterID: master.ID, SKUCode: "AGG-001"}); err != nil {
		t.Fatal(err)
	}

	// Set up router.
	r := gin.New()
	rg := r.Group("/api/v1/product-hub")
	rg.GET("/:id/hub", h.GetHub)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/product-hub/1/hub", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                `json:"code"`
		Data ProductHubProfile `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Data.Master.Name != "Aggregated Product" {
		t.Fatalf("expected 'Aggregated Product', got '%s'", resp.Data.Master.Name)
	}
	if len(resp.Data.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(resp.Data.Variants))
	}
	if resp.Data.Variants[0].SKUCode != "AGG-001" {
		t.Fatalf("expected 'AGG-001', got '%s'", resp.Data.Variants[0].SKUCode)
	}
	if len(resp.Data.Timeline) == 0 {
		t.Fatal("expected timeline events, got none")
	}
}
