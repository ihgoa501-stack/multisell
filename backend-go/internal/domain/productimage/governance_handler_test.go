package productimage

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func governanceRouter(t *testing.T, owner int64) (*gin.Engine, *Service) {
	t.Helper()
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &RightsGrant{}, &Review{}, &CostEntry{})
	svc := NewService(db, zap.NewNop(), nil)
	h := NewGovernanceHandler(svc)
	r := gin.New()
	g := r.Group("/product-images")
	g.Use(func(c *gin.Context) { c.Set("user_id", owner) })
	g.POST("/rights-grants", h.CreateRights)
	g.GET("/rights-grants", h.ListRights)
	g.POST("/rights-grants/:grant_id/revocations", h.RevokeRights)
	g.POST("/tasks/:id/reviews", h.CreateReview)
	g.GET("/tasks/:id/reviews", h.ListReviews)
	g.POST("/tasks/:id/costs", h.CreateCost)
	g.GET("/tasks/:id/costs", h.ListCosts)
	return r, svc
}

func TestGovernanceAPIListsOnlyOwnerWithPagination(t *testing.T) {
	r, svc := governanceRouter(t, 42)
	now := time.Now().UTC()
	for i, owner := range []int64{42, 42, 99} {
		g := RightsGrant{OwnerID: owner, AssetSHA: strings.Repeat("b", 64), Purpose: "listing_main", Jurisdiction: "*", Channel: "ozon", Provider: "deterministic", Region: "local", Grantor: "owner", RightsChain: "evidence", EvidenceSHA: strings.Repeat("e", 64), OwnerVerified: true, ValidFrom: now, IdempotencyKey: fmt.Sprintf("k-%d", i), RequestHash: strings.Repeat("a", 64), Version: 1}
		if err := svc.db.Create(&g).Error; err != nil {
			t.Fatal(err)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/product-images/rights-grants?page=1&size=1", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"total":2`) || strings.Count(w.Body.String(), `"asset_sha256"`) != 1 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestGovernanceAPIRejectsOrdinaryActualReview(t *testing.T) {
	r, svc := governanceRouter(t, 42)
	task := readyGovernanceTask(t, svc, 42, "deterministic")
	body := fmt.Sprintf(`{"asset_sha256":"%s","purpose":"listing_main","channel":"ozon","product_authenticity":"passed","rights":"passed","channel_rules":"passed","claims_scene":"passed","technical_visual":"passed","evidence_sha256":"%s","evidence_truth":"actual","idempotency_key":"review","expected_version":1}`, task.OutputBlobID, strings.Repeat("e", 64))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/product-images/tasks/%d/reviews", task.ID), strings.NewReader(body)))
	if w.Code != 422 || !strings.Contains(w.Body.String(), "VALIDATION_ERROR") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestGovernanceAPIEnforcesExpectedVersionAndStrictDecimal(t *testing.T) {
	r, svc := governanceRouter(t, 42)
	task := readyGovernanceTask(t, svc, 42, "openai")
	body := fmt.Sprintf(`{"kind":"estimated","category":"provider","provider":"openai","amount":"1e2","currency":"USD","exchange_rate":"7.2","exchange_rate_source":"owner","observed_at":"%s","billing_status":"estimated","idempotency_key":"cost","expected_version":1}`, time.Now().UTC().Format(time.RFC3339))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/product-images/tasks/%d/costs", task.ID), strings.NewReader(body)))
	if w.Code != 422 {
		t.Fatalf("decimal status=%d body=%s", w.Code, w.Body.String())
	}
	body = strings.Replace(body, `"amount":"1e2"`, `"amount":"1.20"`, 1)
	body = strings.Replace(body, `"expected_version":1`, `"expected_version":2`, 1)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/product-images/tasks/%d/costs", task.ID), strings.NewReader(body)))
	if w.Code != 409 || !strings.Contains(w.Body.String(), "VERSION_CONFLICT") {
		t.Fatalf("version status=%d body=%s", w.Code, w.Body.String())
	}
}
