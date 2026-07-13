package productanalysis

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers product analysis routes on the given router group.
// Must be called under a JWT-protected group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/product-analysis")
	{
		group.POST("/analyze", h.Analyze)
		group.GET("/analyses", h.ListAnalyses)
		group.GET("/analyses/:id", h.GetAnalysis)
		group.POST("/analyses/:id/feedback", h.RecordFeedback)

	}
}
