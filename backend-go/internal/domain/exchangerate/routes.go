package exchangerate

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers exchange-rate routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/exchange-rates")
	{
		// Static paths first.
		group.GET("", h.List)
		group.POST("", h.Create)

		// Parameterized.
		group.DELETE("/:id", h.Delete)
		group.PUT("/:from_currency/:to_currency", h.UpdateByPair)
		group.GET("/:from_currency/:to_currency/latest", h.GetLatest)
	}
}
