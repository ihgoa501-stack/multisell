package dashboard

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers dashboard routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/dashboard")
	{
		group.GET("/overview", h.Overview)
		group.GET("/orders", h.Orders)
		group.GET("/inventory", h.Inventory)
		group.GET("/exceptions", h.Exceptions)
		group.GET("/rejection-reasons", h.RejectionReasons)
		group.GET("/brief", h.DailyBrief)
	}
}
