package sourcing1688

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers sourcing1688 routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/sourcing-1688")
	{
		group.GET("", h.List)
		group.GET("/summary", h.Summary)
		group.GET("/:id", h.Get)
		group.GET("/:id/snapshot", h.Snapshot)
		group.GET("/:id/draft", h.Draft)
		group.POST("/capture", h.Capture)
		group.POST("/:id/review", h.Review)
		group.POST("/:id/convert-to-draft", h.ConvertToDraft)
	}
}
