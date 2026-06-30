package owner

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers Owner cockpit routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	r := rg.Group("/owner")
	{
		r.GET("/risk-summary", h.RiskSummary)
		r.GET("/suggestions", h.Suggestions)
		r.GET("/platform-sync", h.PlatformSyncStatus)
	}
}
