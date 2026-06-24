package notification

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers notification routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/notification")
	{
		group.GET("", h.List)
		group.GET("/unread-count", h.UnreadCount)
		group.PUT("/read-all", h.MarkAllRead)
		group.GET("/:id", h.Get)
		group.POST("", h.Create)
		group.PUT("/:id/read", h.MarkAsRead)
		group.DELETE("/:id", h.Delete)

		// Alert rules
		group.GET("/alert-rules", h.ListAlertRules)
		group.POST("/alert-rules", h.CreateAlertRule)
		group.PUT("/alert-rules/:id", h.UpdateAlertRule)
		group.DELETE("/alert-rules/:id", h.DeleteAlertRule)
	}
}
