package evolution

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	evo := rg.Group("/evolution")
	{
		evo.GET("/nudges", h.ListNudges)
		evo.POST("/nudges/evaluate", h.EvaluateNudges)
		evo.POST("/nudges/:id/accept", h.AcceptNudge)
		evo.POST("/nudges/:id/dismiss", h.DismissNudge)
	}
}
