package metabolism

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers the metabolism API routes under the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, adapter ScoringAdapter, scorer SemanticScorer) {
	h := NewHandler(NewService(db, logger, adapter, scorer))

	metabolism := rg.Group("/metabolism")
	{
		metabolism.GET("", h.ListLogs)
		metabolism.GET("/:id", h.GetLog)
		metabolism.POST("/dry-run", h.DryRun)
	}
}
