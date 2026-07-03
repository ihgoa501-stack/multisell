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
		g.POST("/bill-batches/import", h.ImportBill)
		g.DELETE("/bill-batches/:id", h.DeleteBillBatch)
		g.GET("/bill-batches/:id/items", h.ListBillItems)
		g.POST("/bill-batches/:id/reconcile", h.ReconcileBillBatch)
		g.GET("/bill-batches/:id/anomalies", h.ListBillAnomalies)

		// Phase 1: Fulfillment Intelligence OS
		g.POST("/quote-unified", h.QuoteUnified)
		g.POST("/snapshots", h.CreateSnapshot)
		g.GET("/snapshots", h.ListSnapshots)
		g.GET("/snapshots/:orderId", h.GetSnapshot)
		g.PUT("/bill-items/:id/review", h.ReviewBillItem)
		g.GET("/rules/:id/versions", h.ListRuleVersions)

		// Phase 3: Fulfillment tracking
		g.POST("/tracking", h.CreateTracking)
		g.GET("/tracking", h.ListTracking)
		g.GET("/tracking/:orderId", h.GetTracking)
		g.PUT("/tracking/:id/event", h.UpdateTrackingEvent)
		g.PUT("/tracking/:id/exception", h.MarkTrackingException)

		// Phase 4: Carrier performance
		g.GET("/carrier-performance", h.GetCarrierPerformance)

		// Phase 5: Carrier API (mock-backed)
		g.GET("/carriers", h.ListCarriers)
		g.POST("/carriers/:code/quote", h.CarrierQuote)
	}
}
