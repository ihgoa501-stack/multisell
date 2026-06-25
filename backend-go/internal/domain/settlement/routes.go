package settlement

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers settlement routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/settlement")
	{
		// Static routes first to avoid conflict with /:id
		group.GET("/summary", h.Summary)

		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/:id", h.Get)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.POST("/:id/reconcile", h.Reconcile)

		// Settlement items (sub-resource)
		group.POST("/:id/items", h.AddItem)
		group.GET("/:id/items", h.ListItems)

		// Item reconciliation (uses item_id, not settlement id)
		itemsGroup := rg.Group("/settlement/items")
		itemsGroup.PUT("/:item_id/reconciliation", h.UpdateItemReconciliation)
	}
}
