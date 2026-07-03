package shipping

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers shipping routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	g := rg.Group("/shipping")
	{
		// Quote
		g.POST("/quote", h.Quote)

		// Providers
		g.GET("/providers", h.ListProviders)
		g.GET("/providers/:id", h.GetProvider)
		g.POST("/providers", h.CreateProvider)
		g.PUT("/providers/:id", h.UpdateProvider)
		g.DELETE("/providers/:id", h.DeleteProvider)

		// Channels
		g.GET("/channels", h.ListChannels)
		g.GET("/channels/:id", h.GetChannel)
		g.POST("/channels", h.CreateChannel)
		g.PUT("/channels/:id", h.UpdateChannel)
		g.DELETE("/channels/:id", h.DeleteChannel)

		// Zones
		g.GET("/zones", h.ListZones)
		g.POST("/zones", h.CreateZone)
		g.DELETE("/zones/:id", h.DeleteZone)

		// Quote rules
		g.GET("/rules", h.ListRules)
		g.POST("/rules", h.CreateRule)
		g.DELETE("/rules/:id", h.DeleteRule)

		// Bill batches
		g.GET("/bill-batches", h.ListBillBatches)
		g.GET("/bill-batches/:id", h.GetBillBatch)
		g.POST("/bill-batches", h.CreateBillBatch)
		g.DELETE("/bill-batches/:id", h.DeleteBillBatch)
		g.GET("/bill-batches/:id/items", h.ListBillItems)

		// Phase 5: Carrier API endpoints (mock-backed)
		g.GET("/carriers", h.ListCarriers)
		g.POST("/carriers/:code/quote", h.CarrierQuote)
	}
}
