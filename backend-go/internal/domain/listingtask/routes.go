package listingtask

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/rbac"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers listing task routes on the given router group.
// approvalSvc, oplogSvc, rbacSvc, loopRec, publishHook may be nil (feature disabled).
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger,
	approvalSvc *approval.Service, oplogSvc *operationlog.Service, rbacSvc *rbac.Service,
	loopRec LoopRecorder, publishHook PublishHook) *Service {
	svc := NewService(db, logger, approvalSvc, oplogSvc, loopRec)
	svc.publishHook = publishHook
	h := NewHandler(svc, rbacSvc)

	tasks := rg.Group("/listing-tasks")
	{
		tasks.GET("", h.List)
		tasks.GET("/:id", h.Get)
		tasks.POST("", h.Create)
		tasks.POST("/from-suggestion", h.CreateFromSuggestion)
		tasks.PUT("/:id", h.Update)
		tasks.DELETE("/:id", h.Delete)

		// Items nested under a task — use :id for task, :item_id for item
		tasks.GET("/:id/items", h.ListItems)
		tasks.POST("/:id/items", h.CreateItem)
		tasks.PUT("/:id/items/:item_id", h.UpdateItem)
		tasks.DELETE("/:id/items/:item_id", h.DeleteItem)
		tasks.GET("/:id/review", h.Review)
	}

	// Listing publish chain — uses /listing-task (singular) prefix.
	chain := rg.Group("/listing-task")
	{
		chain.GET("/stats", h.ListStats)
		chain.POST("/retry-all", h.RetryAll)
		chain.POST("/:task_id/execute", h.Execute)
		chain.POST("/:task_id/retry-failed", h.RetryFailed)
		chain.POST("/:task_id/feedback", h.Feedback)
		chain.POST("/:task_id/items/:item_id/retry", h.RetryItem)
	}
	return svc
}
