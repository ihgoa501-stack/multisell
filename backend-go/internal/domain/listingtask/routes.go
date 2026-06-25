package listingtask

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers listing task routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	tasks := rg.Group("/listing-tasks")
	{
		tasks.GET("", h.List)
		tasks.GET("/:id", h.Get)
		tasks.POST("", h.Create)
		tasks.PUT("/:id", h.Update)
		tasks.DELETE("/:id", h.Delete)

		// Items nested under a task — use :id for task, :item_id for item
		tasks.GET("/:id/items", h.ListItems)
		tasks.POST("/:id/items", h.CreateItem)
		tasks.PUT("/:id/items/:item_id", h.UpdateItem)
		tasks.DELETE("/:id/items/:item_id", h.DeleteItem)
	}

	// Listing publish chain — uses /listing-task (singular) prefix.
	chain := rg.Group("/listing-task")
	{
		chain.GET("/stats", h.ListStats)
		chain.POST("/retry-all", h.RetryAll)
		chain.POST("/:task_id/execute", h.Execute)
		chain.POST("/:task_id/retry-failed", h.RetryFailed)
		chain.POST("/:task_id/items/:item_id/retry", h.RetryItem)
	}
}
