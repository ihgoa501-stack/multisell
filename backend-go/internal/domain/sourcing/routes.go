package sourcing

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers sourcing routes on the given router group.
// bridge is the ToolBridge for fetching product data; events is an optional EventPublisher.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, bridge ToolBridge, events EventPublisher) {
	svc := NewService(db, logger, bridge, events)
	h := NewHandler(svc)

	group := rg.Group("/sourcing")
	{
		group.POST("/fetch", h.Fetch)
		group.GET("/recommendations", h.ListRecommendations)
	}
}
