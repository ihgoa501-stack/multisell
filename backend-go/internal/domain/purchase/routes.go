package purchase

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers purchase routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/purchase")
	{
		// Order routes
		group.GET("/orders", h.ListOrders)
		group.GET("/orders/:id", h.GetOrder)
		group.POST("/orders", h.CreateOrder)
		group.POST("/orders/:id/approve", h.ApproveOrder)
		group.POST("/orders/:id/receive", h.ReceiveOrder)
		group.POST("/orders/:id/cancel", h.CancelOrder)

		// Suggestion routes
		group.GET("/suggestions", h.ListSuggestions)
		group.POST("/suggestions/generate", h.GenerateSuggestions)

		// Supplier routes
		group.GET("/suppliers", h.ListSuppliers)
		group.GET("/suppliers/:id", h.GetSupplier)
		group.POST("/suppliers", h.CreateSupplier)
		group.PUT("/suppliers/:id", h.UpdateSupplier)
		group.DELETE("/suppliers/:id", h.DeleteSupplier)
		group.GET("/suppliers/:id/kpi", h.GetSupplierKPI)
	}
}
