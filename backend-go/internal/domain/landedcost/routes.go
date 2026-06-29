package landedcost

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers landedcost routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/landed-cost")
	{
		// Static routes first.
		group.POST("/calculate", h.Calculate)

		// Parameterized: GET /landed-cost/:productId
		// ?platform=1 (optional — filter by platform)
		group.GET("/:productId", h.GetLandedCost)

		// Compare across platforms: GET /landed-cost/:productId/compare
		group.GET("/:productId/compare", h.CompareAcrossPlatforms)
	}
}
