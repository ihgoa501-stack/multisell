package shipping

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/rbac"
	"go.uber.org/zap"
)

func TestMockCarrierRoutesAreDevelopmentOnly(t *testing.T) {
	for _, tc := range []struct {
		name      string
		allowMock bool
		want      bool
	}{{"production", false, false}, {"development", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			RegisterRoutes(router.Group("/api/v1"), dbtest.NewDB(t), zap.NewNop(), tc.allowMock)
			found := false
			for _, route := range router.Routes() {
				if route.Path == "/api/v1/shipping/carriers" || route.Path == "/api/v1/shipping/carriers/:code/quote" {
					found = true
				}
			}
			if found != tc.want {
				t.Fatalf("mock carrier route registered=%v, want %v", found, tc.want)
			}
		})
	}
}

func TestShippingMutationRequiresWritePermission(t *testing.T) {
	db := dbtest.NewDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.RolePermission{}, &rbac.UserRole{}, &ShippingProvider{})
	viewer := rbac.Role{Name: "viewer", Code: "viewer", Status: 1}
	owner := rbac.Role{Name: "owner", Code: "owner", Status: 1}
	read := rbac.Permission{Name: "shipping read", Code: "shipping.read", Module: "shipping"}
	write := rbac.Permission{Name: "shipping write", Code: "shipping.write", Module: "shipping"}
	db.Create(&viewer)
	db.Create(&owner)
	db.Create(&read)
	db.Create(&write)
	db.Create(&rbac.UserRole{UserID: 7, RoleID: viewer.ID})
	db.Create(&rbac.UserRole{UserID: 8, RoleID: owner.ID})
	db.Create(&rbac.RolePermission{RoleID: viewer.ID, PermissionID: read.ID})
	db.Create(&rbac.RolePermission{RoleID: owner.ID, PermissionID: read.ID})
	db.Create(&rbac.RolePermission{RoleID: owner.ID, PermissionID: write.ID})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		userID, _ := strconv.ParseInt(c.GetHeader("X-Test-User"), 10, 64)
		c.Set("user_id", userID)
		c.Next()
	})
	RegisterRoutes(router.Group("/api/v1"), db, zap.NewNop(), false)

	readRequest := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/providers", nil)
	readRequest.Header.Set("X-Test-User", "7")
	readResponse := httptest.NewRecorder()
	router.ServeHTTP(readResponse, readRequest)
	if readResponse.Code != http.StatusOK {
		t.Fatalf("shipping.read GET status=%d, want 200", readResponse.Code)
	}

	writeRequest := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/providers", strings.NewReader(`{}`))
	writeRequest.Header.Set("Content-Type", "application/json")
	writeRequest.Header.Set("X-Test-User", "7")
	writeResponse := httptest.NewRecorder()
	router.ServeHTTP(writeResponse, writeRequest)
	if writeResponse.Code != http.StatusForbidden {
		t.Fatalf("shipping.read POST status=%d, want 403", writeResponse.Code)
	}

	ownerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/providers", strings.NewReader(`{}`))
	ownerRequest.Header.Set("Content-Type", "application/json")
	ownerRequest.Header.Set("X-Test-User", "8")
	allowedResponse := httptest.NewRecorder()
	router.ServeHTTP(allowedResponse, ownerRequest)
	if allowedResponse.Code == http.StatusForbidden {
		t.Fatal("Owner shipping.write permission did not reach the handler")
	}
}

func TestShippingWriteMigrationRestrictsGrantToOwnerAdmin(t *testing.T) {
	body, err := os.ReadFile("../../../migrations/000153_shipping_owner_write_permission.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, required := range []string{
		"p.code = 'shipping.write'",
		"r.code NOT IN ('owner', 'admin') OR r.status <> 1",
		"r.code IN ('owner', 'admin')",
		"r.status = 1",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("shipping.write migration missing %q", required)
		}
	}
}
