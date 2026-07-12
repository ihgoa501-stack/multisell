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
			if allowOrigin == "" {
				// Origin not allowed — reject
				c.AbortWithStatus(http.StatusForbidden)
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
