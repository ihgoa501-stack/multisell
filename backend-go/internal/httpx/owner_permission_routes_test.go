package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/businessfeedback"
	"github.com/lingmirror/backend-go/internal/domain/purchase"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/rbac"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func ownerGateDB(t *testing.T) *gorm.DB {
	db := dbtest.NewDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{}, &purchase.Authority{})
	roles := []rbac.Role{{ID: 1, Code: "owner", Name: "Owner", Status: 1}, {ID: 2, Code: "ops", Name: "Ops", Status: 1}}
	perms := []rbac.Permission{{ID: 1, Code: "purchase.owner"}, {ID: 2, Code: "business_feedback.owner"}, {ID: 3, Code: "aftersales.owner"}}
	if e := db.Create(&roles).Error; e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&perms).Error; e != nil {
		t.Fatal(e)
	}
	for id := int64(1); id <= 3; id++ {
		if e := db.Create(&rbac.RolePermission{RoleID: 1, PermissionID: id}).Error; e != nil {
			t.Fatal(e)
		}
	}
	if e := db.Create(&rbac.UserRole{UserID: 11, RoleID: 1}).Error; e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&rbac.UserRole{UserID: 22, RoleID: 2}).Error; e != nil {
		t.Fatal(e)
	}
	return db
}
func serveAs(db *gorm.DB, user int64, method, path string, register func(*gin.RouterGroup, *gorm.DB)) *httptest.ResponseRecorder {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", user); c.Next() })
	register(r.Group("/api/v1"), db)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestAuthoritativeDomainRoutesEnforceOwnerPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path string
		register           func(*gin.RouterGroup, *gorm.DB)
	}{
		{"purchase", http.MethodGet, "/api/v1/purchase/authorities", func(g *gin.RouterGroup, db *gorm.DB) { purchase.RegisterRoutes(g, db, zap.NewNop(), nil) }},
		{"business_feedback", http.MethodPost, "/api/v1/business-feedback/actions", func(g *gin.RouterGroup, db *gorm.DB) {
			businessfeedback.RegisterRoutes(g, db, zap.NewNop(), command.NewDispatcher(zap.NewNop()), nil, nil)
		}},
		{"aftersales", http.MethodPost, "/api/v1/aftersales/resolutions", func(g *gin.RouterGroup, db *gorm.DB) { aftersales.RegisterRoutes(g, db, zap.NewNop(), nil) }},
	} {
		t.Run(tc.name+"_ops_denied", func(t *testing.T) {
			db := ownerGateDB(t)
			w := serveAs(db, 22, tc.method, tc.path, tc.register)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
		t.Run(tc.name+"_owner_allowed", func(t *testing.T) {
			db := ownerGateDB(t)
			w := serveAs(db, 11, tc.method, tc.path, tc.register)
			if w.Code == http.StatusForbidden || w.Code == http.StatusUnauthorized {
				t.Fatalf("Owner blocked: status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
