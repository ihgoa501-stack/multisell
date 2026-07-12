package sourcing1688

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers sourcing1688 routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, bridge toolbridge.Bridge) {
	svc := NewService(db, logger)
	h := NewHandler(svc)
	fetchHandler := NewControlledFetchHandler(svc, bridge)

	group := rg.Group("/sourcing-1688", middleware.RequirePermission(db, "product.read"))
	{
		group.GET("", h.List)
		group.GET("/summary", h.Summary)
		group.GET("/:id", h.Get)
		group.GET("/:id/snapshot", h.Snapshot)
		group.GET("/:id/draft", h.Draft)
		group.GET("/:id/identity-history", h.IdentityHistory)
		group.GET("/:id/lifecycle", h.Lifecycle)
		group.GET("/:id/acceptance-report", h.AcceptanceReport)
		group.GET("/:id/publish-requests", h.ListPublishRequests)
		group.GET("/processed-images/:id/content", h.ProcessedImageContent)
		group.GET("/capture-failures", h.ListCaptureFailures)
	}

	write := rg.Group("/sourcing-1688", middleware.RequirePermission(db, "product.write"))
	write.POST("/capture", h.Capture)
	write.POST("/fetch", fetchHandler.Fetch)
	write.POST("/:id/review", h.Review)
	write.POST("/:id/review-decision", h.ReviewDecision)
	write.POST("/:id/capture-failed", h.CaptureFailed)
	write.POST("/:id/convert-to-draft", h.ConvertToDraft)
	write.PUT("/:id/draft", h.UpdateDraft)
	write.POST("/:id/submit-draft-approval", h.SubmitDraftApproval)
	write.POST("/:id/approvals/:approvalId/decision", h.DecideDraftApproval)
	write.POST("/duplicates/:id/resolve", h.ResolveDuplicate)
	write.POST("/processed-images", h.ProcessImage)
	write.POST("/capture-failures", h.RecordCaptureFailure)

	publish := rg.Group("/sourcing-1688", middleware.RequirePermission(db, "listing.publish"))
	publish.POST("/:id/publish-requests", h.RequestPublish)
	publish.POST("/:id/publish-requests/:attemptId/decision", h.DecidePublish)
	publish.POST("/:id/publish-requests/:attemptId/execute", h.ExecutePublish)
	publish.POST("/:id/publish-requests/:attemptId/reconcile", h.ReconcilePublish)
}
