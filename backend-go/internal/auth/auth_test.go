package auth

import (
	"encoding/json"
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
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/rbac"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDBCounter gives each test an isolated in-memory SQLite DB so tests
// don't share state.
var testDBCounter atomic.Int64

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := testDBCounter.Add(1)
	dsn := fmt.Sprintf("file:auth_test_%d?mode=memory&cache=shared", n)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:             "test-secret",
			ExpiryHours:        1,
			RefreshExpiryHours: 24,
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

// ===================== Register =====================

func TestRegister_Success(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	user, err := svc.Register("alice", "password123", "Alice", "alice@example.com", "user")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if user.Username != "alice" {
		t.Fatalf("username = %s", user.Username)
	}
	if user.Role != "user" {
		t.Fatalf("role = %s", user.Role)
	}
	if user.Status != 1 {
		t.Fatalf("status = %d", user.Status)
	}
	if user.PasswordHash == "" {
		t.Fatal("expected password hash to be set")
	}

	vo := user.ToVO()
	if vo.Username != "alice" || vo.ID != user.ID {
		t.Fatalf("vo = %+v", vo)
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	if _, err := svc.Register("bob", "password123", "Bob", "", "user"); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	_, err := svc.Register("bob", "password456", "Bob2", "", "user")
	if err == nil {
		t.Fatal("expected duplicate username error")
	}
	if !strings.Contains(err.Error(), "已存在") {
		t.Fatalf("error = %v", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	_, err := svc.Register("carol", "12345", "Carol", "", "user")
	if err == nil {
		t.Fatal("expected short password error")
	}
	if !strings.Contains(err.Error(), "至少 8 位") {
		t.Fatalf("error = %v", err)
	}
}

func TestRegister_InvalidRole(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	user, err := svc.Register("dave", "password123", "Dave", "", "superadmin")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.Role != "user" {
		t.Fatalf("expected role to fall back to 'user', got %q", user.Role)
	}
}

func TestRegister_OperatorAssignedOpsRBACRole(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&rbac.Role{}, &rbac.UserRole{}); err != nil {
		t.Fatalf("automigrate rbac: %v", err)
	}
	ops := rbac.Role{Name: "Operations", Code: "ops", Status: 1}
	if err := db.Create(&ops).Error; err != nil {
		t.Fatalf("create ops role: %v", err)
	}

	svc := NewService(db, testConfig(), testLogger())
	user, err := svc.Register("operator1", "password123", "Operator", "", "operator")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	var link rbac.UserRole
	if err := db.Where("user_id = ? AND role_id = ?", user.ID, ops.ID).First(&link).Error; err != nil {
		t.Fatalf("expected operator user_role link: %v", err)
	}
}

func TestRegister_UserDoesNotGetOpsRBACRole(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&rbac.Role{}, &rbac.UserRole{}); err != nil {
		t.Fatalf("automigrate rbac: %v", err)
	}
	ops := rbac.Role{Name: "Operations", Code: "ops", Status: 1}
	if err := db.Create(&ops).Error; err != nil {
		t.Fatalf("create ops role: %v", err)
	}

	svc := NewService(db, testConfig(), testLogger())
	user, err := svc.Register("plain-user", "password123", "Plain", "", "user")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	var count int64
	if err := db.Model(&rbac.UserRole{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count user roles: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no RBAC role links for plain user, got %d", count)
	}
}

// ===================== Login =====================

func TestLogin_Success(t *testing.T) {
	db := newTestDB(t)
	cfg := testConfig()
	svc := NewService(db, cfg, testLogger())

	if _, err := svc.Register("eve", "password123", "Eve", "", "user"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	access, refresh, userVO, err := svc.Login("eve", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}
	if access == refresh {
		t.Fatal("access and refresh should differ")
	}
	if userVO == nil || userVO.Username != "eve" {
		t.Fatalf("userVO = %+v", userVO)
	}

	// Access token should parse with type=access.
	claims, err := svc.ParseToken(access)
	if err != nil {
		t.Fatalf("ParseToken access: %v", err)
	}
	if claims.Type != "access" {
		t.Fatalf("access claims.Type = %s", claims.Type)
	}
	if claims.Username != "eve" {
		t.Fatalf("access claims.Username = %s", claims.Username)
	}

	// Refresh token should parse with type=refresh.
	rclaims, err := svc.ParseToken(refresh)
	if err != nil {
		t.Fatalf("ParseToken refresh: %v", err)
	}
	if rclaims.Type != "refresh" {
		t.Fatalf("refresh claims.Type = %s", rclaims.Type)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	if _, err := svc.Register("frank", "password123", "", "", "user"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, _, _, err := svc.Login("frank", "wrong-password")
	if err == nil {
		t.Fatal("expected wrong password error")
	}
	if !strings.Contains(err.Error(), "用户名或密码错误") {
		t.Fatalf("error = %v", err)
	}
}

func TestLogin_DisabledUser(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	user, err := svc.Register("grace", "password123", "", "", "user")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Disable the user.
	if err := db.Model(user).Update("status", 0).Error; err != nil {
		t.Fatalf("update status: %v", err)
	}

	_, _, _, err = svc.Login("grace", "password123")
	if err == nil {
		t.Fatal("expected disabled user error")
	}
	if !strings.Contains(err.Error(), "禁用") {
		t.Fatalf("error = %v", err)
	}
}

// ===================== Refresh =====================

func TestRefresh_Success(t *testing.T) {
	db := newTestDB(t)
	cfg := testConfig()
	svc := NewService(db, cfg, testLogger())

	if _, err := svc.Register("heidi", "password123", "", "", "user"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, refresh, _, err := svc.Login("heidi", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	newAccess, newRefresh, userVO, err := svc.Refresh(refresh)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Fatal("expected non-empty tokens")
	}
	if newAccess == refresh {
		t.Fatal("new access should not equal old refresh")
	}
	if userVO == nil || userVO.Username != "heidi" {
		t.Fatalf("userVO = %+v", userVO)
	}

	// The new refresh token should also be usable.
	if _, _, _, err := svc.Refresh(newRefresh); err != nil {
		t.Fatalf("Refresh new: %v", err)
	}
}

func TestRefresh_WithAccessToken(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	if _, err := svc.Register("ivan", "password123", "", "", "user"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	access, _, _, err := svc.Login("ivan", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// Using an access token as a refresh token should fail (type mismatch).
	_, _, _, err = svc.Refresh(access)
	if err == nil {
		t.Fatal("expected error when using access token as refresh")
	}
	if !strings.Contains(err.Error(), "invalid refresh token") {
		t.Fatalf("error = %v", err)
	}
}

func TestRefresh_ExpiredToken(t *testing.T) {
	db := newTestDB(t)
	cfg := testConfig()
	svc := NewService(db, cfg, testLogger())

	user, err := svc.Register("judy", "password123", "", "", "user")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Hand-craft an expired refresh token signed with the same secret.
	now := time.Now()
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expired, err := tok.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	_, _, _, err = svc.Refresh(expired)
	if err == nil {
		t.Fatal("expected expired token error")
	}
	if !strings.Contains(err.Error(), "invalid refresh token") {
		t.Fatalf("error = %v", err)
	}
}

// ===================== ParseToken =====================

func TestParseToken_InvalidSignature(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	// Sign a token with a different secret.
	otherSvc := NewService(db, &config.Config{
		JWT: config.JWTConfig{Secret: "different-secret", ExpiryHours: 1},
	}, testLogger())
	user := &User{ID: 1, Username: "x", Role: "user"}
	tok, err := otherSvc.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	if _, err := svc.ParseToken(tok); err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestParseToken_ForgedToken(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testConfig(), testLogger())

	cases := []string{
		"",
		"not-a-token",
		"aaa.bbb.ccc",
		"eyJhbGciOiJIUzI1NiJ9.bogus.signature",
	}
	for _, c := range cases {
		if _, err := svc.ParseToken(c); err == nil {
			t.Fatalf("expected error for forged token %q", c)
		}
	}
}

// ===================== Middleware =====================

// newMiddlewareEngine builds a gin engine with the Auth middleware and a
// handler that echoes the user_id from context.
func newMiddlewareEngine(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Auth(cfg))
	r.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": uid})
	})
	return r
}

func TestMiddleware_NoToken(t *testing.T) {
	cfg := testConfig()
	r := newMiddlewareEngine(cfg)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestMiddleware_InvalidScheme(t *testing.T) {
	cfg := testConfig()
	r := newMiddlewareEngine(cfg)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	db := newTestDB(t)
	cfg := testConfig()
	svc := NewService(db, cfg, testLogger())

	user, err := svc.Register("kyle", "password123", "", "", "user")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	access, _, _, err := svc.Login("kyle", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	r := newMiddlewareEngine(cfg)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		UserID interface{} `json:"user_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// MapClaims decodes JSON numbers as float64.
	got, ok := resp.UserID.(float64)
	if !ok {
		t.Fatalf("expected float64 user_id, got %T (%+v)", resp.UserID, resp.UserID)
	}
	if int64(got) != user.ID {
		t.Fatalf("user_id = %v, want %d", got, user.ID)
	}
}

func TestMiddleware_SetsUserID(t *testing.T) {
	db := newTestDB(t)
	cfg := testConfig()
	svc := NewService(db, cfg, testLogger())

	user, err := svc.Register("larry", "password123", "", "", "user")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	access, _, _, err := svc.Login("larry", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	r := gin.New()
	r.Use(middleware.Auth(cfg))
	var captured interface{}
	r.GET("/probe", func(c *gin.Context) {
		captured, _ = c.Get("user_id")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	got, ok := captured.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T (%+v)", captured, captured)
	}
	if got != user.ID {
		t.Fatalf("user_id = %v, want %d", got, user.ID)
	}
}
