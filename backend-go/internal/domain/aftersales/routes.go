package aftersales

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers aftersales routes on the given router group.
// The events parameter is optional; pass nil if no event publishing is desired.
// The deliveryChecker parameter is optional; pass nil to use a no-op checker.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, events interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}, deliveryChecker ...DeliveryChecker) {
	orderAdapter := NewOrderWriterAdapter(db)
	svc := NewService(db, logger, orderAdapter, events)

	// Resolve delivery checker (optional variadic arg).
	var dc DeliveryChecker
	if len(deliveryChecker) > 0 && deliveryChecker[0] != nil {
		dc = deliveryChecker[0]
	}
	disputeSvc := NewDisputeService(db, logger, dc)
	h := NewHandler(svc, disputeSvc)

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

		// Dispute endpoints
		dispute := group.Group("/disputes")
		{
			dispute.GET("", h.ListDisputes)
			dispute.POST("", h.CreateDispute)
			dispute.GET("/:id", h.GetDispute)
			dispute.POST("/:id/evaluate", h.EvaluateDispute)
			dispute.POST("/:id/auto-decide", h.AutoDecideDispute)
			dispute.PUT("/:id/status", h.UpdateDisputeStatus)
		}
	}
}
