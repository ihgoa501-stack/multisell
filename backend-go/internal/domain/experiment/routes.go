package experiment

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	h := NewHandler(NewService(db, logger))
	g := rg.Group("/experiments")
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:experimentId", h.Get)
	g.PUT("/:experimentId", h.Update)
	g.POST("/:experimentId/evidence", h.AddEvidence)
	g.POST("/:experimentId/evidence/:evidenceId/verify", h.VerifyEvidence)
	g.POST("/:experimentId/links", h.AddObjectLink)
	g.POST("/:experimentId/gates/evaluate", h.EvaluateGate)
	g.GET("/:experimentId/owner-summary", h.OwnerSummary)
}
