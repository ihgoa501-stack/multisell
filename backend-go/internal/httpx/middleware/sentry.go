package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"go.uber.org/zap"
)

// Sentry returns a Gin middleware that captures panics and reports them to
// Sentry. It wraps the official sentry-go Gin integration and falls back to
// the local Recovery middleware when Sentry is not configured.
func Sentry(cfg *config.Config, logger *zap.Logger) gin.HandlerFunc {
	if cfg.Sentry.DSN == "" {
		// Sentry not configured; use the standard recovery middleware.
		return Recovery(logger)
	}

	// Use the official sentry-go Gin recovery handler, which captures
	// panics, enriches the event with HTTP context, and sends it to Sentry.
	// The sentrygin middleware already calls sentry.Flush internally with
	// a 2-second timeout for each panic.
	return sentrygin.New(sentrygin.Options{
		Repanic: true,
	})
}

// SentryErrorHandler is a Gin middleware that captures handled errors
// (returned via c.Error or c.Errors) and sends them to Sentry as non-fatal
// events. It does not alter the response.
func SentryErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		hub := sentry.GetHubFromContext(c.Request.Context())
		if hub == nil {
			hub = sentry.CurrentHub()
		}

		for _, err := range c.Errors {
			event := sentry.NewEvent()
			event.Level = sentry.LevelError
			event.Message = err.Error()
			event.Request = &sentry.Request{
				URL:    c.Request.URL.String(),
				Method: c.Request.Method,
			}

			hub.CaptureEvent(event)
		}
	}
}

// RecoveryWithSentry returns a middleware that recovers from panics, reports
// them to Sentry (if configured), and returns a generic 500 response without
// exposing Sentry internals.
func RecoveryWithSentry(cfg *config.Config, logger *zap.Logger) gin.HandlerFunc {
	if cfg.Sentry.DSN == "" {
		return Recovery(logger)
	}

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Report to Sentry
				hub := sentry.GetHubFromContext(c.Request.Context())
				if hub == nil {
					hub = sentry.CurrentHub()
				}
				hub.RecoverWithContext(c.Request.Context(), err)

				// Log locally
				logger.Error("panic recovered and reported to sentry",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
