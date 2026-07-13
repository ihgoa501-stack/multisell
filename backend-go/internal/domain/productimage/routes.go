package productimage

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes attaches the Owner-facing API to an already JWT-protected group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, client *imageservice.Client, executionTokenKeys ...string) {
	service := NewService(db, logger, client, executionTokenKeys...)
	h := NewHandler(service)
	g := rg.Group("/product-images", middleware.RequirePermission(db, "product_image.owner"))
	g.GET("/capabilities", h.Capabilities)
	g.POST("/assets", h.UploadAsset)
	g.POST("/manual-imports", h.CreateManualImport)
	g.GET("/manual-imports", h.ListManualImports)
	g.POST("/tasks", h.CreateTask)
	g.GET("/tasks", h.ListTasks)
	g.GET("/tasks/:id", h.GetTask)
	g.POST("/tasks/:id/executions", h.Execute)
	g.POST("/tasks/:id/execution-approvals", h.ApproveExecution)
	budget := NewBudgetHandler(service)
	g.POST("/budget-policies", budget.CreatePolicy)
	g.GET("/budget-policies", budget.ListPolicies)
	g.GET("/budget-reservations", budget.ListReservations)
	g.POST("/budget-reservations/:reservation_id/cancel", budget.Cancel)
	g.POST("/budget-reservations/:reservation_id/charges", budget.Reconcile)
	g.POST("/budget-reservations/:reservation_id/no-charge-reconciliations", budget.ReconcileNoCharge)
	g.GET("/tasks/:id/attempts", h.Attempts)
	g.GET("/tasks/:id/output/content", h.OutputContent)
	governance := NewGovernanceHandler(service)
	g.POST("/rights-grants", governance.CreateRights)
	g.GET("/rights-grants", governance.ListRights)
	g.POST("/rights-grants/:grant_id/revocations", governance.RevokeRights)
	g.POST("/tasks/:id/reviews", governance.CreateReview)
	g.GET("/tasks/:id/reviews", governance.ListReviews)
	g.POST("/tasks/:id/feedback", governance.CreateFeedback)
	g.GET("/recipes/:recipe_key/summary", governance.RecipeSummary)
	g.POST("/tasks/:id/costs", governance.CreateCost)
	g.GET("/tasks/:id/costs", governance.ListCosts)
	imageSetHandler := NewImageSetHandler(db, client)
	g.POST("/image-sets", imageSetHandler.Create)
	g.GET("/image-sets/:set_id", imageSetHandler.Get)
	g.POST("/image-sets/:set_id/freeze", imageSetHandler.Freeze)
	var releaseKey, releaseKeyID string
	if len(executionTokenKeys) > 1 {
		releaseKey = executionTokenKeys[1]
	}
	if len(executionTokenKeys) > 2 {
		releaseKeyID = executionTokenKeys[2]
	}
	releaseService := NewReleaseService(db, client, releaseKey, releaseKeyID)
	release := NewReleaseHandler(releaseService)
	g.POST("/rule-snapshots", release.CreateRule)
	g.POST("/image-sets/:set_id/decisions", release.DecideSet)
	g.POST("/release-attestations", release.Issue)
	g.GET("/release-attestations/:attestation_id", release.Get)
	// Empty by default: existing e-commerce adapters accept arbitrary URLs and
	// intentionally do not satisfy the controlled-byte publication contract.
	publish := NewPublishHandler(NewPublishService(db, client, releaseService, NewPublisherRegistry()))
	g.POST("/release-attestations/:attestation_id/publish-attempts", publish.Execute)
	g.GET("/publish-attempts/:attempt_id", publish.Get)
	g.POST("/publish-attempts/:attempt_id/reconcile", publish.Reconcile)
	mcpHandler := NewMCPHandler(service)
	g.POST("/mcp", mcpHandler.ServeHTTP)
}
