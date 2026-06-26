package sourcing

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers sourcing routes on the given router group.
// events is an optional EventPublisher; if nil, events will not be published.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, events EventPublisher) {
	svc := NewService(db, logger, nil, events)
	h := NewHandler(svc)

	group := rg.Group("/sourcing")
	{
		group.POST("/fetch", h.Fetch)
		group.GET("/recommendations", h.ListRecommendations)
	}
}
