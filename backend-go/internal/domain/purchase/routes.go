package purchase

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
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
		// Legacy order writes used float amounts and internal status as external
		// truth. Keep the paths fail-closed so old clients cannot silently write.
		group.POST("/orders", h.LegacyWriteFrozen)
		group.POST("/orders/:id/approve", h.LegacyWriteFrozen)
		group.POST("/orders/:id/receive", h.LegacyWriteFrozen)
		group.POST("/orders/:id/cancel", h.LegacyWriteFrozen)

		authority := group.Group("/authorities", middleware.RequirePermission(db, "purchase.owner"))
		authority.POST("", h.CreateAuthority)
		authority.GET("", h.ListAuthorities)
		authority.GET("/:id", h.GetAuthority)
		authority.POST("/:id/owner-approval", h.ApproveAuthority)
		authority.POST("/:id/external-submissions", h.RecordSubmission)
		authority.POST("/:id/order-receipts", h.RecordOrderReceipt)
		authority.POST("/:id/failure-receipts", h.RecordFailureReceipt)
		authority.POST("/:id/receiving-events", h.RecordReceiving)

		// Suggestion routes
		group.GET("/suggestions", h.ListSuggestions)
		group.POST("/suggestions/generate", h.GenerateSuggestions)
	}
}
