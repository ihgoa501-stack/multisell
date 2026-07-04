package supplychain

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers supply chain flow and tracking routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, bus *eventbus.Bus) {
	svc := NewService(db, logger, bus)
	h := NewHandler(svc)

	group := rg.Group("/supply-chain/flows")
	{
		group.GET("", h.List)
		group.GET("/:id", h.Get)
		group.GET("/:id/events", h.GetEvents)
		group.POST("", h.Create)
		group.PUT("/:id", h.Update)
	}

	// Tracking routes
	trackingSvc := NewTrackingService(db, logger)
	th := NewTrackingHandler(trackingSvc).SetCarrierClient(NewMockCarrierClient())

	trackingGroup := rg.Group("/supply-chain/tracking")
	{
		trackingGroup.GET("", th.List)
		trackingGroup.GET("/:id", th.Get)
		trackingGroup.GET("/flow/:flowId", th.GetByFlow)
		trackingGroup.POST("", th.Create)
		trackingGroup.PUT("/:id/status", th.UpdateStatus)
		trackingGroup.POST("/:id/sync", th.SyncFromCarrier)
	}
}
