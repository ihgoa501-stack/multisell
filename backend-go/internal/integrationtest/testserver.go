// Package integrationtest provides shared test utilities for API integration tests.
//
// It creates an isolated HTTP test server with in-memory SQLite DB, full Gin
// middleware stack (CORS, RequestID, Auth), and auth routes so that domain
// integration tests can exercise the full HTTP -> route -> handler -> service -> DB path.
//
// Usage:
//
//	import "github.com/lingmirror/backend-go/internal/integrationtest"
//
//	func TestMyRoutes(t *testing.T) {
//	    ts := integrationtest.NewTestServer(t, domain.RegisterRoutes, &domain.Model{})
//	    defer ts.Close()
//	    token := ts.Login(t)
//	    resp := ts.Get(t, "/api/v1/domain/path", token)
//	    defer resp.Body.Close()
//	    // assert on resp
//	}
package integrationtest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/auth"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TestServer wraps an httptest.Server with helper methods for integration tests.
type TestServer struct {
	*httptest.Server
	DB     *gorm.DB
	Config *config.Config
}

// NewTestServer creates an isolated HTTP test server with an in-memory SQLite DB,
// full middleware stack (CORS, RequestID, Auth), and auth routes.
//
// The registerRoutes callback is called with the JWT-protected route group, so the
// domain can register its routes directly. Additional models are auto-migrated.
func NewTestServer(t *testing.T, registerRoutes func(*gin.RouterGroup, *gorm.DB, *zap.Logger), models ...interface{}) *TestServer {
	t.Helper()

	gin.SetMode(gin.TestMode)

	// Always include auth.User for login/register to work.
	allModels := append([]interface{}{&auth.User{}}, models...)
	db := dbtest.NewDB(t, allModels...)
	logger := zap.NewNop()
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:             "integration-test-secret",
			ExpiryHours:        1,
			RefreshExpiryHours: 24,
		},
	}

	r := gin.New()
	r.Use(middleware.CORS(cfg))
	r.Use(middleware.RequestID())

	api := r.Group("/api/v1")
	auth.RegisterRoutes(api, db, cfg, logger)

	protected := api.Group("")
	protected.Use(middleware.Auth(cfg))
	registerRoutes(protected, db, logger)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &TestServer{
		Server: srv,
		DB:     db,
		Config: cfg,
	}
}

// Login registers a test user and returns a JWT access token.
func (ts *TestServer) Login(t *testing.T) string {
	t.Helper()
	svc := auth.NewService(ts.DB, ts.Config, zap.NewNop())

	_, err := svc.Register("testuser", "password123", "Test User", "test@example.com", "user")
	if err != nil {
		t.Fatalf("register test user: %v", err)
	}

	access, _, _, err := svc.Login("testuser", "password123")
	if err != nil {
		t.Fatalf("login test user: %v", err)
	}
	return access
}

// Get sends a GET request with an optional Bearer token.
func (ts *TestServer) Get(t *testing.T, path, token string) *http.Response {
	return ts.doRequest(t, http.MethodGet, path, nil, token)
}

// Post sends a POST request with a JSON body and optional Bearer token.
func (ts *TestServer) Post(t *testing.T, path string, body interface{}, token string) *http.Response {
	return ts.doRequest(t, http.MethodPost, path, body, token)
}

// Put sends a PUT request with a JSON body and optional Bearer token.
func (ts *TestServer) Put(t *testing.T, path string, body interface{}, token string) *http.Response {
	return ts.doRequest(t, http.MethodPut, path, body, token)
}

// Delete sends a DELETE request with an optional Bearer token.
func (ts *TestServer) Delete(t *testing.T, path, token string) *http.Response {
	return ts.doRequest(t, http.MethodDelete, path, nil, token)
}

func (ts *TestServer) doRequest(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, ts.URL+path, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}
