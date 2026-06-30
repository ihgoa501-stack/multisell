package mock

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers mock data routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	r := rg.Group("/mock")
	{
		r.POST("/seed", h.Seed)
		r.GET("/orders", h.ListOrders)
		r.GET("/settlements", h.ListSettlements)
		r.GET("/sync-statuses", h.SyncStatuses)
	}
}
