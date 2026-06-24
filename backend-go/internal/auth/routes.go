package auth

import (
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
		auth.POST("/login", h.Login)
		auth.POST("/register", h.Register)
		auth.POST("/refresh", h.Refresh)
		auth.GET("/me", middleware.Auth(cfg, db), h.CurrentUser)
	}
}
