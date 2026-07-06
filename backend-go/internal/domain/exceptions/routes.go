package exceptions

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers exceptions routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/exceptions")
	{
		group.GET("", h.List)
		group.GET("/:id", h.Get)
		group.POST("", h.Create)
		group.PUT("/:id", h.Update)
		group.PUT("/:id/resolve", h.Resolve)
		group.PUT("/:id/assign", h.Assign)
		group.POST("/auto-detect", h.AutoDetect)
		group.DELETE("/:id", h.Delete)
	}
}
