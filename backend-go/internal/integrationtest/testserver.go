// Package integrationtest provides a shared test server for HTTP integration tests.
//
// Usage:
//
//	import "github.com/lingmirror/backend-go/internal/integrationtest"
//
//	func TestMyRoutes(t *testing.T) {
//	    ts := integrationtest.NewTestServer(t, domain.RegisterRoutes, &domain.Model{})
//	    defer ts.Close()
//
//	    token := ts.Login(t)
//	    resp := ts.Get(t, "/api/v1/domain/path", token)
//	    defer resp.Body.Close()
//	    // assert resp.StatusCode == 200
//	}
package integrationtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/auth"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TestServer wraps an httptest.Server with a test database, config, and convenience
// methods for common HTTP operations and authentication.
type TestServer struct {
	server *httptest.Server
	db     *gorm.DB
	cfg    *config.Config
}

// NewTestServer creates a test server with:
//   - An in-memory SQLite database (via dbtest) with all given models auto-migrated
//   - A minimal config with a known JWT secret
//   - CORS and RequestID middleware
//   - Auth routes (login/register) registered on /api/v1 (public)
//   - The domain's register function called on the JWT-protected group
func NewTestServer(t testing.TB, register func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger), models ...interface{}) *TestServer {
	t.Helper()

	// Ensure auth.User is always migrated (needed for Login/Register).
	allModels := make([]interface{}, 0, len(models)+1)
	allModels = append(allModels, &auth.User{})
	allModels = append(allModels, models...)
	db := dbtest.NewDB(t, allModels...)
	logger := dbtest.NewLogger(t)

	// Build a minimal config with a known JWT secret.
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:             "test-secret-key-for-jwt",
			ExpiryHours:        24,
			RefreshExpiryHours: 720,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: "*",
		},
		Server: config.ServerConfig{
			Mode: "test",
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS(cfg))
	r.Use(middleware.RequestID())

	// API v1 group
	api := r.Group("/api/v1")

	// Auth routes (public — login, register, refresh)
	auth.RegisterRoutes(api, db, cfg, logger)

	// Protected routes (require JWT)
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg))

	// Register domain routes on the protected group
	register(protected, db, logger)

	server := httptest.NewServer(r)

	return &TestServer{
		server: server,
		db:     db,
		cfg:    cfg,
	}
}

// Close shuts down the test server.
func (ts *TestServer) Close() {
	ts.server.Close()
}

// GetDB returns the test in-memory database.
func (ts *TestServer) GetDB() *gorm.DB {
	return ts.db
}

// GetURL returns the test server's base URL.
func (ts *TestServer) GetURL() string {
	return ts.server.URL
}

// Get performs an authenticated or unauthenticated GET request.
// Pass an empty token for unauthenticated requests.
func (ts *TestServer) Get(t testing.TB, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", ts.server.URL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest GET %s: %v", path, err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

// Post performs an authenticated or unauthenticated POST request.
// body is a JSON string; pass an empty token for unauthenticated requests.
func (ts *TestServer) Post(t testing.TB, path, body, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", ts.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest POST %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// Put performs an authenticated or unauthenticated PUT request.
func (ts *TestServer) Put(t testing.TB, path, body, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("PUT", ts.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest PUT %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", path, err)
	}
	return resp
}

// Delete performs an authenticated or unauthenticated DELETE request.
func (ts *TestServer) Delete(t testing.TB, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("DELETE", ts.server.URL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest DELETE %s: %v", path, err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", path, err)
	}
	return resp
}

// loginResponseData is the expected shape of the login response data field.
type loginResponseData struct {
	AccessToken string `json:"access_token"`
}

// Login registers a test user and logs in to obtain a JWT access token.
func (ts *TestServer) Login(t testing.TB) string {
	t.Helper()

	// Register a test user (ignore error — the user may already exist).
	_ = ts.Post(t, "/api/v1/auth/register", `{"username":"testuser","password":"test123456"}`, "")

	// Login to get the token.
	resp := ts.Post(t, "/api/v1/auth/login", `{"username":"testuser","password":"test123456"}`, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Login failed with status %d", resp.StatusCode)
	}

	var wrapper struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    loginResponseData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if wrapper.Code != 0 {
		t.Fatalf("Login failed: code=%d message=%s", wrapper.Code, wrapper.Message)
	}
	return wrapper.Data.AccessToken
}
