package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
)

func TestRegistrationGateFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		cfg    *config.Config
		want   int
		called bool
	}{
		{name: "nil config", cfg: nil, want: http.StatusForbidden},
		{name: "release disabled", cfg: &config.Config{Server: config.ServerConfig{Mode: "release"}}, want: http.StatusForbidden},
		{name: "enabled", cfg: &config.Config{JWT: config.JWTConfig{RegistrationEnabled: true}}, want: http.StatusNoContent, called: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			r := gin.New()
			r.POST("/register", registrationGate(tc.cfg), func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/register", nil))
			if w.Code != tc.want || called != tc.called {
				t.Fatalf("status=%d called=%v", w.Code, called)
			}
		})
	}
}
