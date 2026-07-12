package demandcase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/rbac"
	"go.uber.org/zap"
)

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &DemandCase{}, &DemandEvidence{}, &DemandVerdict{}, &ResearchBatch{}, &ResearchSnapshot{}, &MarketOwnerDecision{}, &ProductOpportunity{}, &ProductOpportunityDecision{}, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})
	role := rbac.Role{Name: "Owner", Code: "owner-test", Status: 1}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&rbac.UserRole{UserID: 9, RoleID: role.ID}).Error; err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"market.read", "market.write", "market.decide"} {
		permission := rbac.Permission{Name: code, Code: code, Module: "market"}
		if err := db.Create(&permission).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&rbac.RolePermission{RoleID: role.ID, PermissionID: permission.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(func(c *gin.Context) { c.Set("user_id", int64(9)); c.Next() })
	RegisterRoutes(g, db, zap.NewNop())
	return r
}

func TestCandidateMarketAPIAndOwnerDecisionCard(t *testing.T) {
	r := testRouter(t)
	body := `{"region":"DE","consumer":"城市养猫家庭","need_scenario":"短途出行饮水","sales_channel":"独立站","target_locale":"de-DE","stop_condition":"费用无法核实时停止"}`
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
