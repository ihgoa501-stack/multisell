package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers auth routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg *config.Config, logger *zap.Logger) {
	svc := NewService(db, cfg, logger)
	h := NewHandler(svc, cfg, logger)

	auth := rg.Group("/auth")
	{
		auth.POST("/login", loginLimiter.Limit(), h.Login)
		auth.POST("/register", registrationGate(cfg), registerLimiter.Limit(), h.Register)
		auth.POST("/refresh", refreshLimiter.Limit(), h.Refresh)
		auth.POST("/extension-pairings/claim", refreshLimiter.Limit(), h.ClaimExtensionPairing)
		auth.POST("/extension-pairings/exchange", refreshLimiter.Limit(), h.ExchangeExtensionPairing)
		auth.POST("/extension-devices/refresh", refreshLimiter.Limit(), h.RefreshExtensionDevice)
		auth.GET("/me", middleware.Auth(cfg), h.CurrentUser)
		auth.POST("/extension-pairings", middleware.Auth(cfg), h.CreateExtensionPairing)
		auth.GET("/extension-pairings/:pairingId", middleware.Auth(cfg), h.GetExtensionPairing)
		auth.POST("/extension-pairings/:pairingId/confirm", middleware.Auth(cfg), h.ConfirmExtensionPairing)
		auth.GET("/extension-devices", middleware.Auth(cfg), h.ListExtensionDevices)
		auth.DELETE("/extension-devices/:deviceId", middleware.Auth(cfg), h.RevokeExtensionDevice)
		auth.POST("/logout", middleware.Auth(cfg), h.Logout)
		auth.POST("/logout-all", middleware.Auth(cfg), h.LogoutAll)
	}
}

func registrationGate(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg == nil || (cfg.Server.Mode == "release" && !cfg.JWT.RegistrationEnabled) {
			c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "public registration is disabled"})
			return
		}
		c.Next()
	}
}

// loginLimiter limits login attempts to 10 per minute per IP.
var loginLimiter = middleware.NewRateLimiter(10, 1*time.Minute)

// registerLimiter limits registration attempts to 5 per minute per IP.
var registerLimiter = middleware.NewRateLimiter(5, 1*time.Minute)

// refreshLimiter limits token refresh attempts to 20 per minute per IP.
var refreshLimiter = middleware.NewRateLimiter(20, 1*time.Minute)
