package demandcase

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	h := NewHandler(NewService(db, logger))
	g := rg.Group("/demand-cases")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.GET("/:id/decision-card", h.DecisionCard)
	g.GET("/:id/permission-requests", h.PermissionRequests)
	write := g.Group("", middleware.RequirePermission(db, "ai.action"))
	write.POST("", h.Create)
	write.POST("/:id/evidence", h.AddEvidence)
	write.POST("/:id/falsifications", h.AddFalsification)
	write.POST("/:id/evaluate", h.Evaluate)
	write.POST("/research/import", h.ImportResearch)
	write.POST("/research/reviewed-market-permission-batch", h.ImportReviewedBatch)
	pg := rg.Group("/problem-cases")
	pg.GET("", h.ListProblems)
	pg.GET("/:id", h.GetProblem)
	pw := pg.Group("", middleware.RequirePermission(db, "ai.action"))
	pw.POST("", h.CreateProblem)
	pw.POST("/research/reviewed-problem-batch", h.ImportReviewedProblemBatch)
	pw.POST("/:id/evidence", h.AddProblemEvidence)
	pw.POST("/:id/evaluate", h.EvaluateProblem)
	pw.POST("/:id/promote", h.PromoteProblem)
}
