package sentiment

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers sentiment routes on the given router group.
// Must be called under a JWT-protected group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/sentiment")
	{
		group.GET("/negative", h.ListNegativeSentiment)
		group.GET("/:productId", h.GetProductSentiment)
		group.POST("/:productId/refresh", h.RefreshSentiment)
	}
}
