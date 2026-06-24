package allocation

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers allocation routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/allocation")
	{
		// Warehouse management
		group.GET("/warehouses", h.ListWarehouses)
		group.POST("/warehouses", h.CreateWarehouse)
		group.PUT("/warehouses/:id", h.UpdateWarehouse)
		group.DELETE("/warehouses/:id", h.DeleteWarehouse)

		// Allocation rule management
		group.GET("/rules", h.ListRules)
		group.POST("/rules", h.CreateRule)
		group.PUT("/rules/:id", h.UpdateRule)
		group.DELETE("/rules/:id", h.DeleteRule)

		// Cost allocation batches
		group.GET("/cost/batches", h.ListBatches)
		group.POST("/cost/batches", h.CreateBatch)
		group.GET("/cost/batches/:id", h.GetBatch)
	}
}
