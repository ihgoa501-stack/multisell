package cost

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers cost dashboard routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, budget float64) {
	svc := NewService(db, logger)
	h := NewHandler(svc, budget)

	group := rg.Group("/cost")
	{
		group.GET("/dashboard", h.Dashboard)
	}
}
