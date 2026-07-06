package report

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers report routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/report")
	{
		group.GET("/sales", h.Sales)
		group.GET("/profit", h.Profit)
		group.GET("/inventory", h.Inventory)
		group.GET("/settlement", h.Settlement)
		group.GET("/platform-fee", h.PlatformFee)
		group.GET("/daily", h.DailyReport)
		group.GET("/weekly", h.WeeklyReport)
	}
}
