package productimage

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes attaches the Owner-facing API to an already JWT-protected group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, client *imageservice.Client, executionTokenKeys ...string) {
	service := NewService(db, logger, client, executionTokenKeys...)
	h := NewHandler(service)
	g := rg.Group("/product-images")
	g.GET("/capabilities", h.Capabilities)
	g.POST("/assets", h.UploadAsset)
	g.POST("/tasks", h.CreateTask)
	g.GET("/tasks", h.ListTasks)
	g.GET("/tasks/:id", h.GetTask)
	g.POST("/tasks/:id/executions", h.Execute)
	g.POST("/tasks/:id/execution-approvals", h.ApproveExecution)
	g.GET("/tasks/:id/attempts", h.Attempts)
	g.GET("/tasks/:id/output/content", h.OutputContent)
	governance := NewGovernanceHandler(service)
	g.POST("/rights-grants", governance.CreateRights)
	g.GET("/rights-grants", governance.ListRights)
	g.POST("/rights-grants/:grant_id/revocations", governance.RevokeRights)
	g.POST("/tasks/:id/reviews", governance.CreateReview)
	g.GET("/tasks/:id/reviews", governance.ListReviews)
	g.POST("/tasks/:id/costs", governance.CreateCost)
	g.GET("/tasks/:id/costs", governance.ListCosts)
	imageSetHandler := NewImageSetHandler(db, client)
	g.POST("/image-sets", imageSetHandler.Create)
	g.GET("/image-sets/:set_id", imageSetHandler.Get)
	g.POST("/image-sets/:set_id/freeze", imageSetHandler.Freeze)
	RegisterMCP(rg, service)
}
