package middleware

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

const requestIDHeader = "X-Request-ID"

var safeRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// RequestID returns a middleware that generates a unique request ID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(requestIDHeader)
		if !safeRequestID.MatchString(rid) {
			rid = uuid.New().String()
		}
		c.Set("request_id", rid)
		c.Request = c.Request.WithContext(eventbus.WithCorrelationID(c.Request.Context(), rid))
		c.Header(requestIDHeader, rid)
		c.Next()
	}
}
