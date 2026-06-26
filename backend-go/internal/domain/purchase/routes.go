package purchase

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers purchase routes on the given router group.
// The events parameter is optional; pass nil for no event publishing.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, events interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}) {
	svc := NewService(db, logger, events)
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
	}
}
