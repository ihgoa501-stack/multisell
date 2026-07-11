package demandcase

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	h := NewHandler(NewService(db, logger))
	g := rg.Group("/demand-cases")
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
	g.POST("/:id/evidence", h.AddEvidence)
	g.POST("/:id/falsifications", h.AddFalsification)
	g.POST("/:id/evaluate", h.Evaluate)
	g.GET("/:id/decision-card", h.DecisionCard)
}
