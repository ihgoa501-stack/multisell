package sourcing1688

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterExtensionRoutes exposes only the narrow Owner-private collection
// seam to device-bound extension credentials. It deliberately excludes draft,
// approval, publish and task-link mutations.
func RegisterExtensionRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg *config.Config, logger *zap.Logger) {
	h := NewHandler(NewService(db, logger))
	group := rg.Group("/extension/sourcing-1688", middleware.ExtensionAuth(cfg, db, "sourcing1688.collect"))
	group.POST("/private-collections", h.CollectPrivate)
	group.POST("/private-collections/failures", h.RecordPrivateCaptureFailure)
	group.GET("/private-collections/requests/:requestId", h.GetPrivateCollectionRequest)
}

// RegisterRoutes registers sourcing1688 routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, bridge toolbridge.Bridge) {
	svc := NewService(db, logger)
	h := NewHandler(svc)
	fetchHandler := NewControlledFetchHandler(svc, bridge)

	group := rg.Group("/sourcing-1688", middleware.RequirePermission(db, "product.read"))
	{
		group.GET("", h.List)
		group.GET("/summary", h.Summary)
		group.GET("/eligible-tasks", h.ListEligibleTasks)
		group.GET("/private-collections/failures", h.ListPrivateCaptureFailures)
		group.GET("/private-collections/requests/:requestId", h.GetPrivateCollectionRequest)
		group.GET("/:id", h.Get)
		group.GET("/:id/snapshot", h.Snapshot)
		group.GET("/:id/draft", h.Draft)
		group.GET("/:id/identity-history", h.IdentityHistory)
		group.GET("/:id/lifecycle", h.Lifecycle)
		group.GET("/:id/acceptance-report", h.AcceptanceReport)
		group.GET("/:id/publish-requests", h.ListPublishRequests)
		group.GET("/:id/task-links", h.ListPrivateTaskLinks)
		group.GET("/:id/task-links/:linkId/draft", h.TaskDraft)
		group.GET("/:id/task-links/:linkId/publish-requests", h.ListTaskPublishRequests)
		group.GET("/:id/samples", h.ListSourcingSamples)
		group.GET("/:id/task-links/:linkId/compliance-evidence", h.ListComplianceEvidence)
		group.GET("/:id/cost-versions", h.ListSourcingCostVersions)
		group.GET("/processed-images/:id/content", h.ProcessedImageContent)
		group.GET("/capture-failures", h.ListCaptureFailures)
	}

	write := rg.Group("/sourcing-1688", middleware.RequirePermission(db, "product.write"))
	write.POST("/private-collections", h.CollectPrivate)
	write.POST("/:id/task-links", h.LinkPrivateTask)
	write.POST("/:id/samples", h.CreateSourcingSample)
	write.POST("/:id/samples/:sampleId/transitions", h.TransitionSourcingSample)
	write.POST("/:id/task-links/:linkId/sample-waiver", h.WaiveSourcingSample)
	write.POST("/:id/task-links/:linkId/compliance-evidence", h.CreateComplianceEvidence)
	write.POST("/:id/task-links/:linkId/compliance-evidence/:evidenceId/review", h.ReviewComplianceEvidence)
	write.POST("/:id/task-links/:linkId/compliance-evidence/:evidenceId/revoke", h.RevokeComplianceEvidence)
	write.POST("/:id/cost-versions", h.CreateSourcingCostVersion)
	write.POST("/:id/task-links/:linkId/convert-to-draft", h.ConvertTaskToDraft)
	write.PUT("/:id/task-links/:linkId/draft", h.UpdateTaskDraft)
	write.POST("/:id/task-links/:linkId/submit-draft-approval", h.SubmitTaskDraftApproval)
	write.POST("/:id/task-links/:linkId/approvals/:approvalId/decision", h.DecideTaskDraftApproval)
	write.PATCH("/:id/private-workcopy", h.UpdatePrivateWorkcopy)
	write.POST("/:id/private-archive", h.ArchivePrivateCollection)
	write.POST("/:id/private-restore", h.RestorePrivateCollection)
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
	publish.POST("/:id/task-links/:linkId/publish-requests", h.RequestTaskPublish)
	publish.POST("/:id/task-links/:linkId/publish-requests/:attemptId/decision", h.DecideTaskPublish)
	publish.POST("/:id/task-links/:linkId/publish-requests/:attemptId/execute", h.ExecuteTaskPublish)
	publish.POST("/:id/task-links/:linkId/publish-requests/:attemptId/reconcile", h.ReconcileTaskPublish)
	publish.POST("/:id/task-links/:linkId/publish-requests/:attemptId/terminal-observations", h.ObserveTaskPublishTerminal)
	publish.POST("/:id/publish-requests/:attemptId/decision", h.DecidePublish)
	publish.POST("/:id/publish-requests/:attemptId/execute", h.ExecutePublish)
	publish.POST("/:id/publish-requests/:attemptId/reconcile", h.ReconcilePublish)
}
