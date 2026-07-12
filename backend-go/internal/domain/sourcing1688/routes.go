package sourcing1688

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers sourcing1688 routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, bridge toolbridge.Bridge) {
	svc := NewService(db, logger)
	h := NewHandler(svc)
	fetchHandler := NewControlledFetchHandler(svc, bridge)

	group := rg.Group("/sourcing-1688")
	{
		group.GET("", h.List)
		group.GET("/summary", h.Summary)
		group.GET("/:id", h.Get)
		group.GET("/:id/snapshot", h.Snapshot)
		group.GET("/:id/draft", h.Draft)
		group.GET("/:id/identity-history", h.IdentityHistory)
		group.GET("/:id/lifecycle", h.Lifecycle)
		group.GET("/:id/acceptance-report", h.AcceptanceReport)
		group.POST("/capture", h.Capture)
		group.POST("/fetch", fetchHandler.Fetch)
		group.POST("/:id/review", h.Review)
		group.POST("/:id/review-decision", h.ReviewDecision)
		group.POST("/:id/capture-failed", h.CaptureFailed)
		group.POST("/:id/convert-to-draft", h.ConvertToDraft)
		group.PUT("/:id/draft", h.UpdateDraft)
		group.POST("/:id/submit-draft-approval", h.SubmitDraftApproval)
		group.POST("/:id/approvals/:approvalId/decision", h.DecideDraftApproval)
		group.GET("/:id/publish-requests", h.ListPublishRequests)
		group.POST("/:id/publish-requests", h.RequestPublish)
		group.POST("/:id/publish-requests/:attemptId/decision", h.DecidePublish)
		group.POST("/:id/publish-requests/:attemptId/execute", h.ExecutePublish)
		group.POST("/:id/publish-requests/:attemptId/reconcile", h.ReconcilePublish)
		group.POST("/duplicates/:id/resolve", h.ResolveDuplicate)
		group.POST("/processed-images", h.ProcessImage)
		group.GET("/processed-images/:id/content", h.ProcessedImageContent)
		group.POST("/capture-failures", h.RecordCaptureFailure)
		group.GET("/capture-failures", h.ListCaptureFailures)
	}
}
