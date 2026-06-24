package importbatch

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers importbatch routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/importbatch")
	{
		group.GET("", h.ListBatches)
		group.GET("/:id", h.GetBatch)
		group.POST("", h.CreateBatch)
		group.PUT("/:id", h.UpdateBatch)
		group.DELETE("/:id", h.DeleteBatch)
		group.GET("/:id/rows", h.ListRows)
	}
}
