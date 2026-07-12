package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxRequestBody bounds every request body before any handler parses JSON,
// multipart data, or raw webhook payloads. Caddy enforces the same outer limit;
// this remains necessary for direct internal traffic and defense in depth.
func MaxRequestBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "request body limit is not configured"})
			return
		}
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"code": 413, "message": "request body exceeds the configured limit"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
