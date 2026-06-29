package completeness

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers completeness check routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	r := rg.Group("/completeness")
	{
		r.GET("/checks", h.ListChecks)
		r.POST("/check/:productId", h.Check)
	}
}
