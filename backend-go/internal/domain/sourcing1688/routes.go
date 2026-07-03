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
		group.POST("", h.Create)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.POST("/:id/import", h.Import)
		group.POST("/:id/reject", h.Reject)
	}
}
