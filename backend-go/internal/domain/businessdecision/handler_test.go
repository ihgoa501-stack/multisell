package businessdecision

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/rbac"
)

func TestOwnerBusinessDecisionAPI(t *testing.T) {
	db := dbtest.NewDB(t, &orderIngest{}, &Case{}, &FactSnapshot{}, &AIRecommendation{}, &OwnerDecision{}, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})
	db.Create(&orderIngest{OwnerID: 9, AccountID: 1, PlatformCode: "ozon", ExternalEventID: "e", ExternalOrderID: "o", EventAction: "reserve", TruthStatus: "external_observed", ObservedAt: time.Now(), ProcessingStatus: "applied"})
	role := rbac.Role{Name: "Owner", Code: "decision-owner", Status: 1}
	db.Create(&role)
	db.Create(&rbac.UserRole{UserID: 9, RoleID: role.ID})
	for _, code := range []string{"market.read", "market.write", "market.decide"} {
		p := rbac.Permission{Name: code, Code: code, Module: "decision"}
		db.Create(&p)
		db.Create(&rbac.RolePermission{RoleID: role.ID, PermissionID: p.ID})
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(func(c *gin.Context) { c.Set("user_id", int64(9)); c.Next() })
	RegisterRoutes(g, db)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/business-decisions", strings.NewReader(`{"question":"是否补货","target":"避免断货","object_type":"platform_order_ingest","object_id":1,"unknowns":["持续需求"],"idempotency_key":"api-case"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/business-decisions/1", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "manifest_sha256") {
		t.Fatalf("get %d %s", w.Code, w.Body.String())
	}
}
