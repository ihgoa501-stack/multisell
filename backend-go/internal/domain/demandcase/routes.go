package demandcase

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	h := NewHandler(NewService(db, logger))
	g := rg.Group("/demand-cases", middleware.RequirePermission(db, "market.read"))
	g.GET("", h.List)
	g.GET("/comparison", h.Compare)
	g.GET("/:id", h.Get)
	g.GET("/:id/decision-card", h.DecisionCard)
	g.GET("/:id/owner-decision", h.LatestMarketDecision)
	write := rg.Group("/demand-cases", middleware.RequirePermission(db, "market.write"))
	write.POST("", h.Create)
	write.POST("/:id/evidence", h.AddEvidence)
	write.POST("/:id/falsifications", h.AddFalsification)
	write.POST("/:id/evaluate", h.Evaluate)
	write.POST("/research/import", h.ImportResearch)
	write.POST("/research/first-public-batch", h.RunFirstBatch)
	decide := rg.Group("/demand-cases", middleware.RequirePermission(db, "market.decide"))
	decide.POST("/:id/owner-decisions", h.DecideMarket)
	opportunities := rg.Group("/product-opportunities", middleware.RequirePermission(db, "market.read"))
	opportunities.GET("", h.ListProductOpportunities)
	opportunities.GET("/:id", h.GetProductOpportunity)
	opportunityWrite := rg.Group("/product-opportunities", middleware.RequirePermission(db, "market.write"))
	opportunityWrite.POST("", h.CreateProductOpportunity)
	opportunityWrite.POST("/:id/evaluate", h.EvaluateProductOpportunity)
	opportunityDecide := rg.Group("/product-opportunities", middleware.RequirePermission(db, "market.decide"))
	opportunityDecide.POST("/:id/owner-decisions", h.DecideProductOpportunity)
}
