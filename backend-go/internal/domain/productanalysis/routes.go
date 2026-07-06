package productanalysis

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/prismadapter"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers product analysis routes on the given router group.
// Must be called under a JWT-protected group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	RegisterRoutesWithPrism(rg, db, logger, nil)
}

// RegisterRoutesWithPrism registers product analysis routes with an optional Prism service.
func RegisterRoutesWithPrism(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, prismSvc prismadapter.PrismService) {
	svc := NewService(db, logger)
	h := NewHandler(svc)
	if prismSvc != nil {
		h.WithPrism(prismSvc)
	}

	group := rg.Group("/product-analysis")
	{
		group.POST("/analyze", h.Analyze)
		group.GET("/analyses", h.ListAnalyses)
		group.GET("/analyses/:id", h.GetAnalysis)
		group.POST("/analyses/:id/feedback", h.RecordFeedback)

		// P3: Prism image generation trigger (#193)
		if prismSvc != nil {
			group.POST("/trigger-prism", h.TriggerPrism)
		}
	}
}
