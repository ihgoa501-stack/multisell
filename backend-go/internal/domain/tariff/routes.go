package tariff

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers tariff routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/tariff")
	{
		// Static route first to avoid conflict with /:id
		group.POST("/decide", h.Decide)

		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/:id", h.Get)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
}
