package demandcase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &DemandCase{}, &DemandEvidence{}, &DemandVerdict{}, &ResearchBatch{}, &ResearchSnapshot{}, &DataAccessRecord{}, &ProblemCase{}, &ProblemEvidence{})
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(func(c *gin.Context) {
		c.Set("user_id", int64(9))
		c.Set("_rbac_perms", []string{"ai.action"})
		c.Next()
	})
	RegisterRoutes(g, db, zap.NewNop())
	return r
}

func TestReviewedWildfireEventAPIIsIdempotent(t *testing.T) {
	r := testRouter(t)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/problem-cases/research/reviewed-wildfire-event-batch", nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"rejected":1`) || !strings.Contains(w.Body.String(), `"selected_channels":0`) {
			t.Fatalf("import %d status=%d body=%s", i, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problem-cases", nil))
	if w.Code != http.StatusOK || strings.Count(w.Body.String(), "us-ca-hoopa-2021-monument-fire-household-clean-air") != 1 || !strings.Contains(w.Body.String(), `"residual_barrier_status":"not_confirmed"`) {
		t.Fatalf("unexpected list: %s", w.Body.String())
	}
}

func TestCandidateMarketAPIAndOwnerDecisionCard(t *testing.T) {
	r := testRouter(t)
	body := `{"region":"DE","consumer":"城市养猫家庭","need_scenario":"短途出行饮水","sales_channel":"独立站","stop_condition":"费用无法核实时停止"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/demand-cases", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Data DemandCase `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Data.OwnerID != 9 || created.Data.ID == 0 {
		t.Fatalf("owner-bound case: %+v", created.Data)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/demand-cases/1/decision-card", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("card status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "evidence_missing") || !strings.Contains(w.Body.String(), "next_authority_or_cost") {
		t.Fatalf("unexpected card: %s", w.Body.String())
	}
}
