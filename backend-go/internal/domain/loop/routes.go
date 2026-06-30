package loop

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/prismadapter"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers evaluation loop routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, prismSvc prismadapter.PrismService, prismStrict bool) {
	svc := NewService(db, logger, prismSvc, prismStrict)
	h := NewHandler(svc)

	r := rg.Group("/loop")
	{
		r.GET("/recommendations", h.GetRecommendations)
		r.POST("/evaluate/:productId", h.Evaluate)
	}
}
