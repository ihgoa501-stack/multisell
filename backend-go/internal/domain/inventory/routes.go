package inventory

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers inventory routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/inventory")
	{
		// Inventory CRUD + stock operations
		group.GET("", h.List)
		group.GET("/:id", h.Get)
		group.PUT("/:id", h.Update)
		group.POST("/:id/lock", h.Lock)
		group.POST("/:id/unlock", h.Unlock)
		group.GET("/logs", h.ListLogs)

		// Per-SKU warehouse inventory
		group.GET("/sku/:sku_id/warehouses", h.ListInventoryBySku)

		// Bin locations
		group.GET("/locations", h.ListLocations)

		// Inventory transfers
		group.GET("/transfers", h.ListTransfers)

		// Cross-platform inventory sync (oversell prevention)
		group.POST("/sync-cross-platform/:productId", h.SyncCrossPlatform)
		group.GET("/oversell-report", h.OversellReport)

		// Safety stock config (Issue #201)
		group.GET("/safety-config/:sku_id", h.GetSafetyConfig)
		group.PUT("/safety-config/:sku_id", h.UpsertSafetyConfig)
		group.GET("/safety-configs", h.ListSafetyConfigs)

		// Multi-platform allocation (Issue #201)
		group.GET("/allocate/:sku_id", h.AllocateStock)

		// Dead stock analysis (Issue #201)
		group.POST("/dead-stock/analyze", h.IdentifyDeadStock)
		group.GET("/dead-stock/logs", h.ListDeadStockLogs)
	}
}
