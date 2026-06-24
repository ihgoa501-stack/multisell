package rbac

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lingmirror/backend-go/internal/config"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDBCounter atomic.Int64

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := testDBCounter.Add(1)
	dsn := fmt.Sprintf("file:rbac_test_%d?mode=memory&cache=shared", n)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Role{}, &Permission{}, &UserRole{}, &RolePermission{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func init() {
	gin.SetMode(gin.TestMode)
}

// ===================== Service-level tests =====================

func TestRBAC_CreateRole_And_Get(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	r := &Role{Name: "管理员", Code: "admin", Description: "全部权限", Status: 1}
	if err := svc.CreateRole(r); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	fetched, err := svc.GetRole(r.ID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if fetched.Code != "admin" || fetched.Name != "管理员" {
		t.Fatalf("fetched = %+v", fetched)
	}
}

func TestRBAC_ListRoles_StatusFilter(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	for _, code := range []string{"r1", "r2", "r3"} {
		if err := svc.CreateRole(&Role{Name: code, Code: code, Status: 1}); err != nil {
			t.Fatalf("CreateRole %s: %v", code, err)
		}
	}
	// Disable r3. We can't set Status=0 at create time because GORM's
	// `default:1` tag treats the zero value as unset, so update it after.
	if err := db.Model(&Role{}).Where("code = ?", "r3").Update("status", 0).Error; err != nil {
		t.Fatalf("disable r3: %v", err)
	}

	// status=-1 means no filter.
	all, total, err := svc.ListRoles(-1, 1, 20)
	if err != nil {
		t.Fatalf("ListRoles all: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Fatalf("expected 3, got total=%d len=%d", total, len(all))
	}

	// status=1 means active only.
	active, total, err := svc.ListRoles(1, 1, 20)
	if err != nil {
		t.Fatalf("ListRoles active: %v", err)
	}
	if total != 2 || len(active) != 2 {
		t.Fatalf("expected 2 active, got total=%d len=%d", total, len(active))
	}
}

func TestRBAC_UpdateRole(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	r := &Role{Name: "Old", Code: "old", Status: 1}
	if err := svc.CreateRole(r); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	r.Name = "New"
	r.Description = "updated"
	if err := svc.UpdateRole(r); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}

	fetched, err := svc.GetRole(r.ID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if fetched.Name != "New" || fetched.Description != "updated" {
		t.Fatalf("fetched = %+v", fetched)
	}
}

func TestRBAC_DeleteRole_Cascades(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	r := &Role{Name: "Temp", Code: "temp", Status: 1}
	p := &Permission{Name: "Read", Code: "read"}
	if err := svc.CreateRole(r); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if err := svc.CreatePermission(p); err != nil {
		t.Fatalf("CreatePermission: %v", err)
	}
	if err := svc.AssignRolePermissions(r.ID, []int64{p.ID}); err != nil {
		t.Fatalf("AssignRolePermissions: %v", err)
	}
	if err := svc.AssignUserRoles(100, []int64{r.ID}); err != nil {
		t.Fatalf("AssignUserRoles: %v", err)
	}

	if err := svc.DeleteRole(r.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	// Role gone.
	if _, err := svc.GetRole(r.ID); err == nil {
		t.Fatal("expected GetRole to fail after delete")
	}
	// Cascade: role_permission cleared.
	perms, err := svc.GetRolePermissions(r.ID)
	if err != nil {
		t.Fatalf("GetRolePermissions: %v", err)
	}
	if len(perms) != 0 {
		t.Fatalf("expected 0 perms after delete, got %d", len(perms))
	}
	// Cascade: user_role cleared.
	roles, err := svc.GetUserRoles(100)
	if err != nil {
		t.Fatalf("GetUserRoles: %v", err)
	}
	if len(roles) != 0 {
		t.Fatalf("expected 0 roles after delete, got %d", len(roles))
	}
}

func TestRBAC_AssignUserRoles_Replace(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	r1 := &Role{Name: "r1", Code: "r1"}
	r2 := &Role{Name: "r2", Code: "r2"}
	r3 := &Role{Name: "r3", Code: "r3"}
	for _, r := range []*Role{r1, r2, r3} {
		if err := svc.CreateRole(r); err != nil {
			t.Fatalf("CreateRole: %v", err)
		}
	}

	uid := int64(42)
	if err := svc.AssignUserRoles(uid, []int64{r1.ID, r2.ID}); err != nil {
		t.Fatalf("AssignUserRoles 1: %v", err)
	}
	roles, err := svc.GetUserRoles(uid)
	if err != nil {
		t.Fatalf("GetUserRoles: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(roles))
	}

	// Replace assignment.
	if err := svc.AssignUserRoles(uid, []int64{r3.ID}); err != nil {
		t.Fatalf("AssignUserRoles 2: %v", err)
	}
	roles, err = svc.GetUserRoles(uid)
	if err != nil {
		t.Fatalf("GetUserRoles: %v", err)
	}
	if len(roles) != 1 || roles[0].ID != r3.ID {
		t.Fatalf("expected 1 role (r3), got %+v", roles)
	}
}

func TestRBAC_GetUserPermissions_Aggregation(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	adminRole := &Role{Name: "admin", Code: "admin"}
	editorRole := &Role{Name: "editor", Code: "editor"}
	for _, r := range []*Role{adminRole, editorRole} {
		if err := svc.CreateRole(r); err != nil {
			t.Fatalf("CreateRole: %v", err)
		}
	}
	permRead := &Permission{Name: "Read", Code: "read", Module: "common"}
	permWrite := &Permission{Name: "Write", Code: "write", Module: "common"}
	permDelete := &Permission{Name: "Delete", Code: "delete", Module: "common"}
	for _, p := range []*Permission{permRead, permWrite, permDelete} {
		if err := svc.CreatePermission(p); err != nil {
			t.Fatalf("CreatePermission: %v", err)
		}
	}

	// admin: read, write, delete.
	if err := svc.AssignRolePermissions(adminRole.ID, []int64{permRead.ID, permWrite.ID, permDelete.ID}); err != nil {
		t.Fatalf("AssignRolePermissions admin: %v", err)
	}
	// editor: read, write (overlaps with admin).
	if err := svc.AssignRolePermissions(editorRole.ID, []int64{permRead.ID, permWrite.ID}); err != nil {
		t.Fatalf("AssignRolePermissions editor: %v", err)
	}

	uid := int64(7)
	if err := svc.AssignUserRoles(uid, []int64{adminRole.ID, editorRole.ID}); err != nil {
		t.Fatalf("AssignUserRoles: %v", err)
	}

	codes, err := svc.GetUserPermissions(uid)
	if err != nil {
		t.Fatalf("GetUserPermissions: %v", err)
	}
	got := map[string]bool{}
	for _, c := range codes {
		got[c] = true
	}
	// Distinct union of read, write, delete (3 unique despite overlap).
	if len(codes) != 3 {
		t.Fatalf("expected 3 distinct permissions, got %d: %v", len(codes), codes)
	}
	for _, want := range []string{"read", "write", "delete"} {
		if !got[want] {
			t.Fatalf("missing permission %q in %v", want, codes)
		}
	}
}

func TestRBAC_GetUserPermissions_NoAssignment(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	codes, err := svc.GetUserPermissions(999)
	if err != nil {
		t.Fatalf("GetUserPermissions: %v", err)
	}
	if len(codes) != 0 {
		t.Fatalf("expected 0 permissions, got %v", codes)
	}
}

func TestRBAC_DeletePermission_Cascades(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	r := &Role{Name: "r", Code: "r"}
	p := &Permission{Name: "p", Code: "p"}
	if err := svc.CreateRole(r); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if err := svc.CreatePermission(p); err != nil {
		t.Fatalf("CreatePermission: %v", err)
	}
	if err := svc.AssignRolePermissions(r.ID, []int64{p.ID}); err != nil {
		t.Fatalf("AssignRolePermissions: %v", err)
	}

	if err := svc.DeletePermission(p.ID); err != nil {
		t.Fatalf("DeletePermission: %v", err)
	}
	perms, err := svc.GetRolePermissions(r.ID)
	if err != nil {
		t.Fatalf("GetRolePermissions: %v", err)
	}
	if len(perms) != 0 {
		t.Fatalf("expected 0 perms after delete, got %d", len(perms))
	}
}

// ===================== Middleware-level RBAC tests =====================
//
// The RBAC module ships no role-check middleware of its own (enforcement
// happens at the route layer). These tests simulate that layer with a small
// requireRole helper so we can verify the user-facing behavior: admin bypass,
// 403 on missing/insufficient role, 200 when authorized.

// authContext is a test middleware that mirrors middleware.Auth but also
// extracts the role and username claims into the gin context so requireRole
// has something to read.
func authContext(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing token"})
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "bad scheme"})
			return
		}
		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.Secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "bad claims"})
			return
		}
		if v, ok := claims["user_id"]; ok {
			c.Set("user_id", v)
		}
		if v, ok := claims["role"]; ok {
			if s, ok := v.(string); ok {
				c.Set("role", s)
			}
		}
		if v, ok := claims["username"]; ok {
			if s, ok := v.(string); ok {
				c.Set("username", s)
			}
		}
		c.Next()
	}
}

// requireRole is a test-only RBAC enforcement middleware. "admin" bypasses
// all checks; otherwise the user's role must be in the allowed set.
func requireRole(allowed ...string) gin.HandlerFunc {
	set := make(map[string]bool, len(allowed))
	for _, r := range allowed {
		set[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)
		if roleStr == "admin" {
			c.Next()
			return
		}
		if !set[roleStr] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "forbidden: requires " + strings.Join(allowed, "|"),
			})
			return
		}
		c.Next()
	}
}

// makeToken builds a JWT with the given role using the test secret.
func makeToken(t *testing.T, cfg *config.Config, userID int64, username, role string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"type":     "access",
		"iat":      now.Unix(),
		"exp":      now.Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func newRBACApp(cfg *config.Config, allowed ...string) *gin.Engine {
	r := gin.New()
	r.Use(authContext(cfg))
	r.GET("/admin/panel", requireRole(allowed...), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRBAC_AdminBypass(t *testing.T) {
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret"}}
	// Only "operator" is explicitly allowed, but admin should bypass.
	app := newRBACApp(cfg, "operator")

	tok := makeToken(t, cfg, 1, "root", "admin")
	req := httptest.NewRequest(http.MethodGet, "/admin/panel", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("admin should bypass; got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRBAC_UserNoPermission(t *testing.T) {
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret"}}
	app := newRBACApp(cfg, "admin")

	tok := makeToken(t, cfg, 2, "alice", "user")
	req := httptest.NewRequest(http.MethodGet, "/admin/panel", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("user should be 403; got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRBAC_HasPermission(t *testing.T) {
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret"}}
	app := newRBACApp(cfg, "operator", "admin")

	tok := makeToken(t, cfg, 3, "bob", "operator")
	req := httptest.NewRequest(http.MethodGet, "/admin/panel", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("operator should pass; got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRBAC_NoRole(t *testing.T) {
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret"}}
	app := newRBACApp(cfg, "admin")

	// Token carries no role claim at all.
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  4,
		"username": "carol",
		"type":     "access",
		"iat":      now.Unix(),
		"exp":      now.Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := tok.SignedString([]byte(cfg.JWT.Secret))

	req := httptest.NewRequest(http.MethodGet, "/admin/panel", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("no role should be 403; got %d body=%s", w.Code, w.Body.String())
	}
}
