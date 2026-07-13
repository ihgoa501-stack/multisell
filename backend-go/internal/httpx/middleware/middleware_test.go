package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/rbac"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDBCounter atomic.Int64

func newTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	n := testDBCounter.Add(1)
	dsn := fmt.Sprintf("file:middleware_test_%d?mode=memory&cache=shared", n)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("automigrate: %v", err)
		}
	}
	return db
}

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret",
		},
	}
}

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestOwnerAuthorityPermissionsRejectOperationalRoles(t *testing.T) {
	db := newTestDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})
	roles := []rbac.Role{{ID: 1, Code: "owner", Name: "Owner", Status: 1}, {ID: 2, Code: "admin", Name: "Admin", Status: 1}, {ID: 3, Code: "ops", Name: "Ops", Status: 1}, {ID: 4, Code: "viewer", Name: "Viewer", Status: 1}}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatal(err)
	}
	permissions := []rbac.Permission{{ID: 1, Code: "purchase.owner"}, {ID: 2, Code: "business_feedback.owner"}, {ID: 3, Code: "aftersales.owner"}}
	if err := db.Create(&permissions).Error; err != nil {
		t.Fatal(err)
	}
	for _, roleID := range []int64{1, 2} {
		for _, permissionID := range []int64{1, 2, 3} {
			if err := db.Create(&rbac.RolePermission{RoleID: roleID, PermissionID: permissionID}).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	for userID, roleID := range map[int64]int64{11: 1, 12: 2, 13: 3, 14: 4} {
		if err := db.Create(&rbac.UserRole{UserID: userID, RoleID: roleID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, permission := range []string{"purchase.owner", "business_feedback.owner", "aftersales.owner"} {
		for _, tc := range []struct {
			name   string
			userID int64
			want   int
		}{{"owner", 11, 200}, {"admin", 12, 200}, {"ops", 13, 403}, {"viewer", 14, 403}} {
			t.Run(permission+"/"+tc.name, func(t *testing.T) {
				r := gin.New()
				r.Use(func(c *gin.Context) { c.Set("user_id", tc.userID); c.Next() })
				r.POST("/authority", RequirePermission(db, permission), func(c *gin.Context) { c.Status(200) })
				w := httptest.NewRecorder()
				r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/authority", nil))
				if w.Code != tc.want {
					t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
				}
			})
		}
	}
}

func TestAuditPreservesRequestBodyLargerThanAuditSnippet(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.POST("/api/v1/sourcing-1688/private-collections", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil || len(body) != 4096 {
			c.String(http.StatusBadRequest, "body was truncated: %d", len(body))
			return
		}
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sourcing-1688/private-collections", strings.NewReader(strings.Repeat("x", 4096)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestExtensionAuthAcceptsOnlyActiveScopedDevice(t *testing.T) {
	const extensionID = "abcdefghijklmnopabcdefghijklmnop"
	db := newTestDB(t)
	for _, sql := range []string{
		`CREATE TABLE "user" (id INTEGER PRIMARY KEY, status INTEGER NOT NULL, role TEXT NOT NULL)`,
		`CREATE TABLE extension_device (device_id TEXT PRIMARY KEY, user_id INTEGER NOT NULL, extension_id TEXT NOT NULL, environment TEXT NOT NULL, scope TEXT NOT NULL, revoked_at DATETIME)`,
		`INSERT INTO "user" (id,status,role) VALUES (7,1,'owner')`,
		`INSERT INTO extension_device (device_id,user_id,extension_id,environment,scope) VALUES ('device-7',7,'` + extensionID + `','development','sourcing1688.collect')`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	cfg := testConfig()
	makeToken := func(tokenType string, scopes []string) string {
		claims := jwt.MapClaims{"type": tokenType, "user_id": float64(7), "device_id": "device-7", "environment": "development", "scopes": scopes,
			"iss": "lingmirror-extension:development", "aud": []string{"lingmirror-sourcing1688"}, "exp": time.Now().Add(time.Hour).Unix()}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token.Header["kid"] = cfg.JWT.EffectiveKeyID()
		signed, err := token.SignedString([]byte(cfg.JWT.Secret))
		if err != nil {
			t.Fatal(err)
		}
		return signed
	}
	run := func(token, origin string) int {
		r := gin.New()
		r.Use(ExtensionAuth(cfg, db, "sourcing1688.collect"))
		r.POST("/collect", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if got := run(makeToken("extension_access", []string{"sourcing1688.collect"}), "chrome-extension://"+extensionID); got != http.StatusNoContent {
		t.Fatalf("valid device status=%d", got)
	}
	if got := run(makeToken("access", []string{"sourcing1688.collect"}), "chrome-extension://"+extensionID); got != http.StatusUnauthorized {
		t.Fatalf("web token status=%d", got)
	}
	if got := run(makeToken("extension_access", []string{"other"}), "chrome-extension://"+extensionID); got != http.StatusForbidden {
		t.Fatalf("wrong scope status=%d", got)
	}
	if got := run(makeToken("extension_access", []string{"sourcing1688.collect"}), "chrome-extension://ponmlkjihgfedcbaponmlkjihgfedcba"); got != http.StatusUnauthorized {
		t.Fatalf("wrong extension origin status=%d", got)
	}
	if err := db.Exec(`UPDATE extension_device SET revoked_at = CURRENT_TIMESTAMP WHERE device_id = 'device-7'`).Error; err != nil {
		t.Fatal(err)
	}
	if got := run(makeToken("extension_access", []string{"sourcing1688.collect"}), "chrome-extension://"+extensionID); got != http.StatusUnauthorized {
		t.Fatalf("revoked device status=%d", got)
	}
}

// signToken signs a jwt.MapClaims with the test secret.
func signToken(t *testing.T, cfg *config.Config, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// ===================== Auth Middleware Tests =====================

func TestAuth_NoHeader(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg, _ := resp["message"].(string); !strings.Contains(msg, "missing") {
		t.Fatalf("message = %q, want containing 'missing'", msg)
	}
}

func TestAuth_InvalidScheme(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	var handlerCalled bool
	r.GET("/protected", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if handlerCalled {
		t.Fatal("handler was called despite invalid scheme")
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer garbage-token-that-cannot-parse")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuth_ExpiredToken(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	tok := signToken(t, cfg, jwt.MapClaims{
		"user_id": float64(1),
		"exp":     float64(time.Now().Add(-time.Hour).Unix()),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuth_WrongSecret(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Sign with a different secret
	otherCfg := &config.Config{JWT: config.JWTConfig{Secret: "other-secret"}}
	tok := signToken(t, otherCfg, jwt.MapClaims{
		"user_id": float64(1),
		"exp":     float64(time.Now().Add(time.Hour).Unix()),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuth_AcceptsExplicitPreviousKeyDuringRotation(t *testing.T) {
	cfg := testConfig()
	cfg.JWT.KeyID = "2026-07"
	cfg.JWT.Secret = "new-secret"
	cfg.JWT.PreviousKeysJSON = `{"2026-06":"old-secret"}`
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type": "access", "user_id": float64(1), "exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	token.Header["kid"] = "2026-06"
	signed, err := token.SignedString([]byte("old-secret"))
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(Auth(cfg))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("previous key token status=%d body=%s", w.Code, w.Body.String())
	}
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type": "access", "user_id": float64(1), "exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	legacySigned, _ := legacy.SignedString([]byte("old-secret"))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+legacySigned)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("legacy no-kid token status=%d", w.Code)
	}

	token.Header["kid"] = "unknown"
	signed, _ = token.SignedString([]byte("old-secret"))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unknown kid status=%d", w.Code)
	}
}

func TestAuth_SetsUserID_Float64Claim(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	var captured interface{}
	r.GET("/protected", func(c *gin.Context) {
		captured, _ = c.Get("user_id")
		c.Status(http.StatusOK)
	})

	tok := signToken(t, cfg, jwt.MapClaims{
		"type":    "access",
		"user_id": float64(42),
		"exp":     float64(time.Now().Add(time.Hour).Unix()),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	got, ok := captured.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T (%+v)", captured, captured)
	}
	if got != 42 {
		t.Fatalf("user_id = %d, want 42", got)
	}
}

func TestAuth_SetsUserID_Int64Claim(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	var captured interface{}
	r.GET("/protected", func(c *gin.Context) {
		captured, _ = c.Get("user_id")
		c.Status(http.StatusOK)
	})

	tok := signToken(t, cfg, jwt.MapClaims{
		"type":    "access",
		"user_id": int64(100),
		"exp":     float64(time.Now().Add(time.Hour).Unix()),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	got, ok := captured.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T (%+v)", captured, captured)
	}
	if got != 100 {
		t.Fatalf("user_id = %d, want 100", got)
	}
}

func TestAuth_MissingUserIDClaim(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	var handlerCalled bool
	r.GET("/protected", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })

	tok := signToken(t, cfg, jwt.MapClaims{
		"type": "access",
		"sub":  "test-subject",
		"exp":  float64(time.Now().Add(time.Hour).Unix()),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if handlerCalled {
		t.Fatal("handler was called without a numeric user identity")
	}
}

func TestAuth_NonNumericUserID(t *testing.T) {
	cfg := testConfig()
	r := gin.New()
	r.Use(Auth(cfg))
	var handlerCalled bool
	r.GET("/protected", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })

	tok := signToken(t, cfg, jwt.MapClaims{
		"type":    "access",
		"user_id": "alice",
		"exp":     float64(time.Now().Add(time.Hour).Unix()),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if handlerCalled {
		t.Fatal("handler was called with a non-numeric user identity")
	}
}

func TestAuth_RejectsNonHS256HMAC(t *testing.T) {
	cfg := testConfig()
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"type": "access", "user_id": float64(1), "exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	signed, err := token.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(Auth(cfg))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// ===================== Audit Middleware Tests =====================

func TestAudit_SkipsHealthCheck(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	var called bool
	r.GET("/api/health", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !called {
		t.Fatal("handler was not called")
	}
	var count int64
	db.Model(&operationlog.OperationLog{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 audit logs, got %d", count)
	}
}

func TestAudit_SkipsHealthz(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var count int64
	db.Model(&operationlog.OperationLog{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 audit logs for /healthz, got %d", count)
	}
}

func TestAudit_SkipsNonSensitiveGET(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.GET("/api/v1/public/list", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var count int64
	db.Model(&operationlog.OperationLog{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 audit logs, got %d", count)
	}
}

func TestAudit_SkipsOptions(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.OPTIONS("/api/v1/order", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/order", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var count int64
	db.Model(&operationlog.OperationLog{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 audit logs for OPTIONS, got %d", count)
	}
}

func TestAudit_RecordsMutationPOST(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.POST("/api/v1/order", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Set("username", "alice")
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/order",
		strings.NewReader(`{"sku":"ABC","qty":1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}

	var logs []operationlog.OperationLog
	db.Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Module != "order" {
		t.Errorf("module = %s, want 'order'", logs[0].Module)
	}
	if logs[0].Action != "create_order" {
		t.Errorf("action = %s, want 'create_order'", logs[0].Action)
	}
	if logs[0].Operator != "alice" {
		t.Errorf("operator = %s, want 'alice'", logs[0].Operator)
	}
	if logs[0].Result != "success" {
		t.Errorf("result = %s, want 'success'", logs[0].Result)
	}
	if logs[0].UserID != 1 {
		t.Errorf("user_id = %d, want 1", logs[0].UserID)
	}
}

func TestAudit_RecordsMutationPUT(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.PUT("/api/v1/order/:id", func(c *gin.Context) {
		c.Set("user_id", int64(2))
		c.Set("username", "bob")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/order/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var logs []operationlog.OperationLog
	db.Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != "update_id_order" {
		t.Errorf("action = %s, want 'update_id_order'", logs[0].Action)
	}
}

func TestAudit_RecordsMutationDELETE(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.DELETE("/api/v1/order/:id", func(c *gin.Context) {
		c.Set("user_id", int64(3))
		c.Set("username", "carol")
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/order/7", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var logs []operationlog.OperationLog
	db.Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != "delete_id_order" {
		t.Errorf("action = %s, want 'delete_id_order'", logs[0].Action)
	}
}

func TestAudit_RecordsMutationPATCH(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.PATCH("/api/v1/order/:id", func(c *gin.Context) {
		c.Set("user_id", int64(4))
		c.Set("username", "dave")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/order/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var logs []operationlog.OperationLog
	db.Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != "patch_id_order" {
		t.Errorf("action = %s, want 'patch_id_order'", logs[0].Action)
	}
}

func TestAudit_RecordsSensitiveGET_Finance(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.GET("/api/v1/finance/report", func(c *gin.Context) {
		c.Set("user_id", int64(5))
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/report", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var logs []operationlog.OperationLog
	db.Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log for sensitive GET, got %d", len(logs))
	}
	if logs[0].Module != "finance" {
		t.Errorf("module = %s, want 'finance'", logs[0].Module)
	}
	if logs[0].Action != "get_report_finance" {
		t.Errorf("action = %s, want 'get_report_finance'", logs[0].Action)
	}
}

func TestAudit_RecordsSensitiveGET_Orders(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.GET("/api/v1/orders/list", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var count int64
	db.Model(&operationlog.OperationLog{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 audit log for sensitive orders GET, got %d", count)
	}
}

func TestAudit_RecordsSensitiveGET_RBAC(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.GET("/api/v1/rbac/roles", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/roles", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var count int64
	db.Model(&operationlog.OperationLog{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 audit log for sensitive rbac GET, got %d", count)
	}
}

func TestAudit_RecordsFailure(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.POST("/api/v1/order", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "validation failed"})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}

	var logs []operationlog.OperationLog
	db.Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Result != "failure" {
		t.Errorf("result = %s, want 'failure'", logs[0].Result)
	}
}

func TestAudit_OperatorFallbackToUserID(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.POST("/api/v1/order", func(c *gin.Context) {
		c.Set("user_id", int64(5))
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var logs []operationlog.OperationLog
	db.Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Operator != "user:5" {
		t.Errorf("operator = %s, want 'user:5'", logs[0].Operator)
	}
}

func TestAudit_OperatorAnonymousWhenNoUserID(t *testing.T) {
	db := newTestDB(t, &operationlog.OperationLog{})
	r := gin.New()
	r.Use(Audit(db, testLogger()))
	r.POST("/api/v1/order", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var logs []operationlog.OperationLog
	db.Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Operator != "anonymous" {
		t.Errorf("operator = %s, want 'anonymous'", logs[0].Operator)
	}
}

// ===================== Audit Helper Function Tests =====================

func TestModuleFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/order/123", "order"},
		{"/api/v1/ai/actions/5/approve", "ai"},
		{"/api/v1/finance", "finance"},
		{"/api/v1/", "root"},
		{"/api/health", "health"},
		{"/api", "root"},
		{"/some-other-path", "some-other-path"},
		{"", "root"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := moduleFromPath(tt.path)
			if got != tt.want {
				t.Errorf("moduleFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestActionFromMethod(t *testing.T) {
	tests := []struct {
		name   string
		method string
		route  string
		want   string
	}{
		{"POST /api/v1/order", http.MethodPost, "/api/v1/order", "create_order"},
		{"PUT /api/v1/order/:id", http.MethodPut, "/api/v1/order/:id", "update_id_order"},
		{"PATCH /api/v1/order/:id", http.MethodPatch, "/api/v1/order/:id", "patch_id_order"},
		{"DELETE /api/v1/order/:id", http.MethodDelete, "/api/v1/order/:id", "delete_id_order"},
		{"POST /api/v1/ai/actions/:id/approve", http.MethodPost, "/api/v1/ai/actions/:id/approve", "create_approve_ai"},
		{"GET /api/v1/finance", http.MethodGet, "/api/v1/finance", "get_finance"},
		{"POST empty route", http.MethodPost, "", "create_unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actionFromMethod(tt.method, tt.route)
			if got != tt.want {
				t.Errorf("actionFromMethod(%q, %q) = %q, want %q", tt.method, tt.route, got, tt.want)
			}
		})
	}
}

func TestResourceIDFromCtx(t *testing.T) {
	r := gin.New()
	r.GET("/resource/:id", func(c *gin.Context) {
		got := resourceIDFromCtx(c)
		if got != "42" {
			t.Errorf("resourceIDFromCtx with :id = %q, want '42'", got)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/resource/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_ = w
}

func TestResourceIDFromCtx_ProductID(t *testing.T) {
	r := gin.New()
	r.GET("/product/:product_id", func(c *gin.Context) {
		got := resourceIDFromCtx(c)
		if got != "p-100" {
			t.Errorf("resourceIDFromCtx with :product_id = %q, want 'p-100'", got)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/product/p-100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_ = w
}

func TestResourceIDFromCtx_NoMatch(t *testing.T) {
	r := gin.New()
	r.GET("/plain", func(c *gin.Context) {
		got := resourceIDFromCtx(c)
		if got != "" {
			t.Errorf("resourceIDFromCtx with no params = %q, want ''", got)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/plain", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_ = w
}

func TestIsSensitivePath(t *testing.T) {
	sensitive := []string{"/api/v1/finance", "/api/v1/orders", "/api/v1/user"}
	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1/finance/summary", true},
		{"/api/v1/finance", true},
		{"/api/v1/orders/123", true},
		{"/api/v1/user/profile", true},
		{"/api/v1/products", false},
		{"/api/v1/health", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isSensitivePath(tt.path, sensitive)
			if got != tt.want {
				t.Errorf("isSensitivePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestComposeAuditContent(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/order", func(c *gin.Context) {
		content := composeAuditContent(c, `{"sku":"ABC"}`, http.StatusCreated)
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content), &parsed); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if parsed["path"] != "/api/v1/order" {
			t.Errorf("path = %v, want '/api/v1/order'", parsed["path"])
		}
		if _, ok := parsed["body"]; !ok {
			t.Errorf("expected 'body' in content for valid JSON, got keys: %v", parsed)
		}
		if status, _ := parsed["status"].(float64); status != 201 {
			t.Errorf("status = %v, want 201", status)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/order",
		strings.NewReader(`{"sku":"ABC"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_ = w
}

func TestComposeAuditContent_NonJSONBody(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/order", func(c *gin.Context) {
		content := composeAuditContent(c, "plain text body that is not JSON", http.StatusOK)
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content), &parsed); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := parsed["body_raw"]; !ok {
			t.Errorf("expected 'body_raw' for non-JSON body, got keys: %v", parsed)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/order",
		strings.NewReader("plain text"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_ = w
}

func TestComposeAuditContent_TruncatesLongNonJSON(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/order", func(c *gin.Context) {
		longBody := strings.Repeat("x", 300)
		content := composeAuditContent(c, longBody, http.StatusOK)
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content), &parsed); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		raw, _ := parsed["body_raw"].(string)
		if len(raw) > 260 {
			t.Errorf("body_raw too long: len=%d, want <=260 (256+ellipsis)", len(raw))
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/order",
		strings.NewReader(strings.Repeat("x", 300)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_ = w
}

func TestComposeAuditContent_WithError(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/order", func(c *gin.Context) {
		c.Set("error", "something went wrong")
		content := composeAuditContent(c, "", http.StatusInternalServerError)
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content), &parsed); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if parsed["error"] != "something went wrong" {
			t.Errorf("error = %v, want 'something went wrong'", parsed["error"])
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_ = w
}

// ===================== RBAC Middleware Tests =====================

func TestRBACMiddleware_NoUserID(t *testing.T) {
	db := newTestDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})
	r := gin.New()
	r.Use(RequirePermission(db, "order:read"))
	var handlerCalled bool
	r.GET("/protected", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if handlerCalled {
		t.Fatal("handler was called despite no user_id in context")
	}
}

func TestRBACMiddleware_InvalidUserIDType(t *testing.T) {
	db := newTestDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", true) // boolean is not the expected type
		c.Next()
	})
	r.Use(RequirePermission(db, "order:read"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRBACMiddleware_HasPermission(t *testing.T) {
	db := initRBACData(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.Use(RequirePermission(db, "order:read"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRBACMiddleware_HasAnyOfMultiplePermissions(t *testing.T) {
	db := initRBACData(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Next()
	})
	// Only "order:read" is assigned; passing both "order:delete" and "order:read"
	r.Use(RequirePermission(db, "order:delete", "order:read"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRBACMiddleware_MissingPermission(t *testing.T) {
	db := initRBACData(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.Use(RequirePermission(db, "order:delete"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg, _ := resp["message"].(string); !strings.Contains(msg, "insufficient") {
		t.Errorf("message = %q, want containing 'insufficient'", msg)
	}
}

func TestRBACMiddleware_UsesCachedPermissions(t *testing.T) {
	// Preset the permissions cache in a front middleware to verify that
	// RequirePermission uses the cached "_rbac_perms" without hitting the DB.
	db := newTestDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Set("_rbac_perms", []string{"order:read", "product:write"})
		c.Next()
	})
	r.Use(RequirePermission(db, "order:read"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRBACMiddleware_LoadPermissionError(t *testing.T) {
	db := newTestDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying sql.DB: %v", err)
	}
	sqlDB.Close()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.Use(RequirePermission(db, "order:read"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestRBACMiddleware_Float64UserID(t *testing.T) {
	db := initRBACData(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", float64(1))
		c.Next()
	})
	r.Use(RequirePermission(db, "order:read"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRBACMiddleware_IntUserID(t *testing.T) {
	db := initRBACData(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int(1))
		c.Next()
	})
	r.Use(RequirePermission(db, "order:read"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// initRBACData creates a test RBAC database with one role, one permission,
// and assigns user_id=1 to that role with that permission.
func initRBACData(t *testing.T) *gorm.DB {
	t.Helper()
	db := newTestDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})

	role := &rbac.Role{Name: "Operator", Code: "operator", Status: 1}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	perm := &rbac.Permission{Name: "Order Read", Code: "order:read", Module: "order"}
	if err := db.Create(perm).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}

	if err := db.Create(&rbac.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
		t.Fatalf("assign permission to role: %v", err)
	}

	if err := db.Create(&rbac.UserRole{UserID: 1, RoleID: role.ID}).Error; err != nil {
		t.Fatalf("assign role to user: %v", err)
	}

	return db
}

// ===================== RateLimit Middleware Tests =====================

func TestRateLimit_SingleRequestPasses(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestRateLimit_WithinLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	r := gin.New()
	r.Use(rl.Limit())
	var callCount int
	r.GET("/test", func(c *gin.Context) { callCount++; c.Status(http.StatusOK) })

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, w.Code)
		}
	}
	if callCount != 3 {
		t.Fatalf("handler called %d times, want 3", callCount)
	}
}

func TestRateLimit_ExceedsLimit(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	// First 2 requests should pass
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, w.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if code, _ := resp["code"].(float64); code != 429 {
		t.Errorf("response code = %v, want 429", code)
	}
	if msg, _ := resp["message"].(string); !strings.Contains(msg, "rate limit") {
		t.Errorf("message = %q, want containing 'rate limit'", msg)
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	// First IP uses its only allowed request
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first IP request 1: status = %d, want 200", w1.Code)
	}

	// Same IP should be blocked
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("first IP request 2: status = %d, want 429", w2.Code)
	}

	// Different IP should be allowed its own window
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "10.0.0.1:54321"
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("second IP: status = %d, want 200", w3.Code)
	}

	// Second IP's second request should also be blocked
	req4 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req4.RemoteAddr = "10.0.0.1:54321"
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusTooManyRequests {
		t.Fatalf("second IP request 2: status = %d, want 429", w4.Code)
	}
}

func TestRateLimit_WindowExpires(t *testing.T) {
	rl := NewRateLimiter(1, 100*time.Millisecond)
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	// First request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first: status = %d, want 200", w.Code)
	}

	// Second request immediately — should be blocked
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second (before window expiry): status = %d, want 429", w2.Code)
	}

	// Wait for window to expire
	time.Sleep(120 * time.Millisecond)

	// Third request — should be allowed again (new window)
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("third (after window expiry): status = %d, want 200", w3.Code)
	}
}

func TestRateLimit_CleanupExpiredEntries(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Make a request to create an entry
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first: status = %d, want 200", w.Code)
	}

	rl.mu.Lock()
	for _, entry := range rl.requests {
		entry.expireAt = time.Now().Add(-time.Second)
	}
	rl.lastCleanup = time.Now().Add(-2 * rl.window)
	rl.mu.Unlock()

	// A later request performs opportunistic cleanup before adding its own entry.
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.0.2.2:1234"
	r.ServeHTTP(httptest.NewRecorder(), req2)
	rl.mu.Lock()
	_, oldExists := rl.requests["192.0.2.1"]
	remaining := len(rl.requests)
	rl.mu.Unlock()
	if oldExists || remaining != 1 {
		t.Fatalf("expired entry cleanup old=%v remaining=%d", oldExists, remaining)
	}
}

func TestRateLimitBoundsDistinctIPMemory(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	rl.maxEntries = 1
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	first := httptest.NewRequest(http.MethodGet, "/test", nil)
	first.RemoteAddr = "192.0.2.1:1234"
	r.ServeHTTP(httptest.NewRecorder(), first)
	second := httptest.NewRequest(http.MethodGet, "/test", nil)
	second.RemoteAddr = "192.0.2.2:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, second)
	if w.Code != http.StatusTooManyRequests || len(rl.requests) != 1 {
		t.Fatalf("status=%d entries=%d", w.Code, len(rl.requests))
	}
}

// ===================== RequestID Middleware Tests =====================

func TestRequestID_GeneratesIfMissing(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	var rid string
	r.GET("/test", func(c *gin.Context) {
		if v, ok := c.Get("request_id"); ok {
			rid, _ = v.(string)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if rid == "" {
		t.Fatal("expected non-empty generated request_id")
	}
	if got := w.Header().Get("X-Request-ID"); got != rid {
		t.Errorf("response header X-Request-ID = %q, want %q", got, rid)
	}
}

func TestRequestID_PreservesExisting(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	var rid string
	r.GET("/test", func(c *gin.Context) {
		if v, ok := c.Get("request_id"); ok {
			rid, _ = v.(string)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "client-provided-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if rid != "client-provided-id" {
		t.Errorf("request_id = %q, want 'client-provided-id'", rid)
	}
	if got := w.Header().Get("X-Request-ID"); got != "client-provided-id" {
		t.Errorf("response header = %q, want 'client-provided-id'", got)
	}
}

func TestRequestID_PropagatesCorrelationAndRejectsUnsafeInput(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, eventbus.CorrelationIDFromContext(c.Request.Context()))
	})

	safe := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(requestIDHeader, "owner-flow:123")
	r.ServeHTTP(safe, req)
	if safe.Body.String() != "owner-flow:123" {
		t.Fatalf("correlation=%q", safe.Body.String())
	}

	unsafe := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(requestIDHeader, "bad\nlog-entry")
	r.ServeHTTP(unsafe, req)
	if unsafe.Body.String() == "bad\nlog-entry" || !safeRequestID.MatchString(unsafe.Body.String()) {
		t.Fatalf("unsafe request ID was not replaced: %q", unsafe.Body.String())
	}
}

// ===================== CORS Middleware Tests =====================

func TestCORS_AllowsAllByDefault(t *testing.T) {
	cfg := &config.Config{}
	r := gin.New()
	r.Use(CORS(cfg))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin = %q, want '*'", got)
	}
}

func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: "https://app.example.com"}}
	r := gin.New()
	r.Use(CORS(cfg))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Allow-Origin = %q, want 'https://app.example.com'", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials = %q, want 'true'", got)
	}
}

func TestCORS_RejectsUnconfiguredOrigin(t *testing.T) {
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: "https://app.example.com"}}
	r := gin.New()
	r.Use(CORS(cfg))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestCORS_AllowsChromeExtensionOnlyForExtensionRoutes(t *testing.T) {
	const origin = "chrome-extension://abcdefghijklmnopabcdefghijklmnop"
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: "https://owner.example.com"}}
	r := gin.New()
	r.Use(CORS(cfg))
	r.POST("/api/v1/auth/extension-pairings/claim", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.POST("/api/v1/auth/extension-pairings/exchange", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.POST("/api/v1/auth/extension-devices/refresh", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/v1/extension/sourcing-1688/requests/1", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/v1/orders", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	allowed := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/auth/extension-pairings/claim"},
		{http.MethodPost, "/api/v1/auth/extension-pairings/exchange"},
		{http.MethodPost, "/api/v1/auth/extension-devices/refresh"},
		{http.MethodGet, "/api/v1/extension/sourcing-1688/requests/1"},
	}
	for _, tc := range allowed {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Fatalf("path=%s status=%d allow-origin=%q", tc.path, w.Code, w.Header().Get("Access-Control-Allow-Origin"))
		}
	}
	bad := httptest.NewRequest(http.MethodPost, "/api/v1/auth/extension-pairings/claim", nil)
	bad.Header.Set("Origin", "chrome-extension://not-a-chrome-extension-id")
	badResponse := httptest.NewRecorder()
	r.ServeHTTP(badResponse, bad)
	if badResponse.Code != http.StatusForbidden {
		t.Fatalf("malformed extension origin status=%d", badResponse.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("ordinary route status=%d content-type=%q body=%s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
}

func TestCORS_OptionsReturnsNoContent(t *testing.T) {
	cfg := &config.Config{}
	r := gin.New()
	r.Use(CORS(cfg))
	r.OPTIONS("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	allowed := w.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(allowed, "X-Approval-ID") || !strings.Contains(allowed, "Idempotency-Key") {
		t.Fatalf("high-risk headers missing from CORS allowlist: %q", allowed)
	}
}

func TestCORS_MultipleOrigins(t *testing.T) {
	cfg := &config.Config{
		CORS: config.CORSConfig{AllowedOrigins: "https://app.example.com,https://admin.example.com"},
	}
	r := gin.New()
	r.Use(CORS(cfg))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("Origin", "https://app.example.com")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first origin: status = %d, want 200", w1.Code)
	}
	if got := w1.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Allow-Origin = %q, want 'https://app.example.com'", got)
	}

	// Second allowed origin
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("Origin", "https://admin.example.com")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second origin: status = %d, want 200", w2.Code)
	}
	if got := w2.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Errorf("Allow-Origin = %q, want 'https://admin.example.com'", got)
	}
}

// ===================== Recovery Middleware Tests =====================

func TestRecovery_CatchesPanic(t *testing.T) {
	logger := testLogger()
	r := gin.New()
	r.Use(Recovery(logger))
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

// ===================== parseOrigins / matchOrigin Tests =====================

func TestParseOrigins(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"*", nil},
		{"https://app.example.com", []string{"https://app.example.com"}},
		{"https://a.com,https://b.com", []string{"https://a.com", "https://b.com"}},
		{" https://a.com , https://b.com ", []string{"https://a.com", "https://b.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseOrigins(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseOrigins(%q) = len=%d %v, want len=%d %v",
					tt.input, len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("parseOrigins(%q) = %v, want %v", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestMatchOrigin(t *testing.T) {
	allowed := []string{"https://app.example.com", "https://admin.example.com"}

	t.Run("matching origin", func(t *testing.T) {
		got := matchOrigin("https://app.example.com", allowed)
		if got != "https://app.example.com" {
			t.Errorf("matchOrigin = %q, want 'https://app.example.com'", got)
		}
	})

	t.Run("non-matching origin", func(t *testing.T) {
		got := matchOrigin("https://evil.com", allowed)
		if got != "" {
			t.Errorf("matchOrigin = %q, want ''", got)
		}
	})

	t.Run("empty origin returns first allowed", func(t *testing.T) {
		got := matchOrigin("", allowed)
		if got != allowed[0] {
			t.Errorf("matchOrigin('') = %q, want %q", got, allowed[0])
		}
	})
}
