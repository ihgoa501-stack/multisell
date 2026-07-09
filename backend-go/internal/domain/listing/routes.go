package listing

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers listing routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, bus *eventbus.Bus, approvalSvc *approval.Service) {
	svc := NewService(db, logger, bus, NewSKUProvider(db), NewDecisionReader(db), NewCandidateReader(db), NewProfitReader(db))
	h := NewHandler(svc, approvalSvc)

	group := rg.Group("/listings")
	{
		group.GET("", h.List)
		group.GET("/:id", h.Get)
		group.POST("", h.Create)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.POST("/:id/publish", h.Publish)
		group.POST("/:id/sync", h.Sync)
			group.POST("/suggest", h.Suggest)
	}

	// Listing publish chain — uses /listing (singular) prefix.
	chain := rg.Group("/listing")
	{
		// POST /v1/listing — alias for POST /v1/listings (frontend compatibility)
		chain.POST("", h.Create)

		chain.POST("/products/:product_id/publish/:platform_id", h.PublishProduct)
		chain.GET("/products/:product_id/listings", h.ListByProduct)
		chain.GET("/products/:product_id/platform-comparison", h.GetPlatformComparison)

		// Static path from-decisions must be registered before :task_id.
		chain.POST("/listing-tasks/from-decisions", h.CreateTasksFromDecisions)
		chain.POST("/listing-tasks/:task_id/recheck", h.RecheckTask)
		chain.POST("/listing-tasks/:task_id/cancel", h.CancelTask)
		chain.POST("/listing-tasks/:task_id/publish", h.PublishTask)
	}
}
