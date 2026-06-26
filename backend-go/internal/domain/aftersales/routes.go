package aftersales

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers aftersales routes on the given router group.
// The events parameter is optional; pass nil if no event publishing is desired.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, events interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}) {
	invAdapter := NewInventoryRestockAdapter(db)
	orderAdapter := NewOrderWriterAdapter(db)
	svc := NewService(db, logger, invAdapter, orderAdapter, events)
	h := NewHandler(svc)

	group := rg.Group("/aftersales")
	{
		group.GET("", h.List)
		group.GET("/summary", h.Summary)
		group.GET("/:id", h.Get)
		group.POST("", h.Create)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.POST("/:id/approve", h.Approve)
		group.POST("/:id/reject", h.Reject)
		group.POST("/:id/receive", h.Receive)
		group.POST("/:id/refund", h.Refund)
	}
}
