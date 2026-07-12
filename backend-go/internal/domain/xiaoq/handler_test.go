package xiaoq

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/rbac"
)

func testHTTPRouter(t *testing.T, svc *Service, ownerID *int64) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if ownerID != nil {
		r.Use(func(c *gin.Context) { c.Set("user_id", *ownerID); c.Next() })
	}
	RegisterHandlerRoutes(r.Group("/api/v1"), NewHandler(svc))
	return r
}

func TestHTTPIdentityAndCapabilitiesExposeOnlyReadV1(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub", resp: &ai.LLMResponse{}})
	owner := int64(42)
	r := testHTTPRouter(t, svc, &owner)

	for _, path := range []string{"/api/v1/xiao-q/identity", "/api/v1/xiao-q/capabilities"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/xiao-q/capabilities", nil))
	if strings.Contains(w.Body.String(), "mutate") || !strings.Contains(w.Body.String(), CapabilityDemandCaseRead) || !strings.Contains(w.Body.String(), CapabilityDemandCaseDecisionRead) {
		t.Fatalf("unexpected capabilities: %s", w.Body.String())
	}
}

func TestHTTPMessageUsesJWTIdentityAndRequiresDemandCase(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub", resp: &ai.LLMResponse{}})
	owner := int64(42)
	r := testHTTPRouter(t, svc, &owner)

	body, _ := json.Marshal(map[string]interface{}{"message": "案件怎么样？", "demand_case_id": 7, "owner_id": 999})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/xiao-q/messages", bytes.NewReader(body)))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"agent_id":"xiao_q"`) || !strings.Contains(w.Body.String(), `"truth_status":"mock"`) {
		t.Fatalf("message status=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/xiao-q/messages", strings.NewReader(`{"message":"缺少案件"}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing demand_case_id status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPRejectsUnauthenticatedAndOverlongMessage(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub", resp: &ai.LLMResponse{}})
	r := testHTTPRouter(t, svc, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/xiao-q/identity", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", w.Code)
	}

	owner := int64(42)
	r = testHTTPRouter(t, svc, &owner)
	body, _ := json.Marshal(MessageInput{Message: strings.Repeat("长", MaxMessageRunes+1), DemandCaseID: 7})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/xiao-q/messages", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("overlong status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPTraceIsOwnerScoped(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub", resp: &ai.LLMResponse{}})
	result, err := svc.SendMessage(httptest.NewRequest(http.MethodGet, "/", nil).Context(), 42, MessageInput{Message: "查看", DemandCaseID: 7})
	if err != nil {
		t.Fatal(err)
	}
	other := int64(99)
	r := testHTTPRouter(t, svc, &other)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/xiao-q/traces/"+result.TraceID, nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-owner trace status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestRegisteredRoutesEnforceReadAndMessageWritePermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{}, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	role := rbac.Role{Name: "reader", Code: "reader", Status: 1}
	read := rbac.Permission{Name: "Agent read", Code: "agent.read", Module: "agent"}
	write := rbac.Permission{Name: "Agent write", Code: "agent.write", Module: "agent"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&read).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&write).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&rbac.UserRole{UserID: 42, RoleID: role.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&rbac.RolePermission{RoleID: role.ID, PermissionID: read.ID}).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(42)); c.Next() })
	RegisterRoutes(r.Group("/api/v1"), db, dbtest.NewLogger(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/xiao-q/identity", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("agent.read GET status=%d body=%s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/xiao-q/messages", strings.NewReader(`{"message":"查看","demand_case_id":7}`)))
	if w.Code != http.StatusForbidden {
		t.Fatalf("missing agent.write POST status=%d body=%s", w.Code, w.Body.String())
	}
}
