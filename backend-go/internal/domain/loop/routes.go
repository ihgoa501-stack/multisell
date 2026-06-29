package loop

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers evaluation loop routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	r := rg.Group("/loop")
	{
		r.GET("/recommendations", h.GetRecommendations)
		r.POST("/evaluate/:productId", h.Evaluate)
	}
}
