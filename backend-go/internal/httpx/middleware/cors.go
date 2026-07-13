package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
// When cfg.CORS.AllowedOrigins is empty or "*", all origins are allowed (dev mode).
// Otherwise, only the specified comma-separated origins are permitted.
func CORS(cfg *config.Config) gin.HandlerFunc {
	allowedOrigins := parseOrigins(cfg.CORS.AllowedOrigins)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		allowOrigin := "*"
		if len(allowedOrigins) > 0 && allowedOrigins[0] != "*" {
			allowOrigin = matchOrigin(origin, allowedOrigins)
			if allowOrigin == "" && isExtensionCORSPath(c.Request.URL.Path) {
				if _, ok := ChromeExtensionIDFromOrigin(origin); ok {
					allowOrigin = origin
				}
			}
			if allowOrigin == "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": "该来源不能访问此接口"})
				return
			}
		}

		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Approval-ID, Idempotency-Key")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		// Credentials support (required when origin is not *)
		if allowOrigin != "*" {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ChromeExtensionIDFromOrigin returns the installed extension ID from a
// browser Origin. Chrome IDs are exactly 32 lower-case characters from a-p.
func ChromeExtensionIDFromOrigin(origin string) (string, bool) {
	const prefix = "chrome-extension://"
	if !strings.HasPrefix(origin, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(origin, prefix)
	if len(id) != 32 {
		return "", false
	}
	for _, ch := range id {
		if ch < 'a' || ch > 'p' {
			return "", false
		}
	}
	return id, true
}

func isExtensionCORSPath(path string) bool {
	switch path {
	case "/api/v1/auth/extension-pairings/claim",
		"/api/v1/auth/extension-pairings/exchange",
		"/api/v1/auth/extension-devices/refresh":
		return true
	default:
		return strings.HasPrefix(path, "/api/v1/extension/")
	}
}

// parseOrigins splits a comma-separated origin string into a slice.
// Returns nil if input is empty or "*".
func parseOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// matchOrigin checks if the request origin is in the allowed list.
// Returns the matched origin (for Vary: Origin header support) or empty string.
func matchOrigin(origin string, allowed []string) string {
	if origin == "" {
		// No Origin header — non-browser client, allow first listed origin
		return allowed[0]
	}
	for _, a := range allowed {
		if a == origin {
			return origin
		}
	}
	return ""
}
