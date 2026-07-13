package shipping

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers shipping routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, allowMock bool) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	read := rg.Group("/shipping", middleware.RequirePermission(db, "shipping.read"))
	{
		read.POST("/quote", h.Quote)
		read.POST("/quote-unified", h.QuoteUnified)

		// Providers
		read.GET("/providers", h.ListProviders)
		read.GET("/providers/:id", h.GetProvider)

		// Channels
		read.GET("/channels", h.ListChannels)
		read.GET("/channels/:id", h.GetChannel)

		// Zones
		read.GET("/zones", h.ListZones)

		// Quote rules
		read.GET("/rules", h.ListRules)
		read.GET("/rules/:id/versions", h.ListRuleVersions)

		// Bill batches
		read.GET("/bill-batches", h.ListBillBatches)
		read.GET("/bill-batches/:id", h.GetBillBatch)
		read.GET("/bill-batches/:id/items", h.ListBillItems)
		read.GET("/bill-batches/:id/anomalies", h.ListBillAnomalies)

		// Phase 1: Fulfillment Intelligence OS
		read.GET("/snapshots", h.ListSnapshots)
		read.GET("/snapshots/:orderId", h.GetSnapshot)

		// Phase 3: Fulfillment tracking
		read.GET("/tracking", h.ListTracking)
		read.GET("/tracking/:orderId", h.GetTracking)

		// Phase 4: Carrier performance
		read.GET("/carrier-performance", h.GetCarrierPerformance)

		// Mock carriers are development fixtures, never production capabilities.
		if allowMock {
			read.GET("/carriers", h.ListCarriers)
			read.POST("/carriers/:code/quote", h.CarrierQuote)
		}
	}

	write := rg.Group("/shipping", middleware.RequirePermission(db, "shipping.write"))
	write.POST("/providers", h.CreateProvider)
	write.PUT("/providers/:id", h.UpdateProvider)
	write.DELETE("/providers/:id", h.DeleteProvider)
	write.POST("/channels", h.CreateChannel)
	write.PUT("/channels/:id", h.UpdateChannel)
	write.DELETE("/channels/:id", h.DeleteChannel)
	write.POST("/zones", h.CreateZone)
	write.DELETE("/zones/:id", h.DeleteZone)
	write.POST("/rules", h.CreateRule)
	write.DELETE("/rules/:id", h.DeleteRule)
	write.POST("/bill-batches", h.CreateBillBatch)
	write.POST("/bill-batches/import", h.ImportBill)
	write.DELETE("/bill-batches/:id", h.DeleteBillBatch)
	write.POST("/bill-batches/:id/reconcile", h.ReconcileBillBatch)
	write.POST("/snapshots", h.CreateSnapshot)
	write.PUT("/bill-items/:id/review", h.ReviewBillItem)
	write.POST("/tracking", h.CreateTracking)
	write.PUT("/tracking/:id/event", h.UpdateTrackingEvent)
	write.PUT("/tracking/:id/exception", h.MarkTrackingException)
}
