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

		// DEPRECATED: Warehouse management.
		// Use /api/v1/allocation/warehouses instead (allocation module).
		// These routes are kept for backward compatibility only.
		// TODO: Remove these routes after frontend migrates to allocation/warehouses.
		group.GET("/warehouses", h.ListWarehouses)
		group.POST("/warehouses", h.CreateWarehouse)
		group.GET("/warehouses/:id", h.GetWarehouse)
		group.PUT("/warehouses/:id", h.UpdateWarehouse)
		group.DELETE("/warehouses/:id", h.DeleteWarehouse)

		// Per-SKU warehouse inventory
		group.GET("/sku/:sku_id/warehouses", h.ListInventoryBySku)

		// Bin locations
		group.GET("/locations", h.ListLocations)

		// Inventory transfers
		group.GET("/transfers", h.ListTransfers)

		// Cross-platform inventory sync (oversell prevention)
		group.POST("/sync-cross-platform/:productId", h.SyncCrossPlatform)
		group.GET("/oversell-report", h.OversellReport)
	}
}
