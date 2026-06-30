package agentlearning

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers agent-learning HTTP routes under the given group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	al := rg.Group("/agent-learning")
	{
		al.GET("/accuracy", h.GetAllAccuracy)
		al.GET("/accuracy/:agentId", h.GetAccuracyByAgent)
		al.GET("/evaluations", h.ListEvaluations)
		al.POST("/evaluate", h.EvaluateDecision)
		al.POST("/recalculate", h.RecalculateAccuracy)
	}
}
