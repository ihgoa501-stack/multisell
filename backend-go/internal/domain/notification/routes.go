package notification

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers notification routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, hubOpt ...*realtime.Hub) {
	var h *realtime.Hub
	if len(hubOpt) > 0 {
		h = hubOpt[0]
	}
	svc := NewService(db, logger, h)
	handler := NewHandler(svc)

	group := rg.Group("/notification")
	{
		group.GET("", handler.List)
		group.GET("/unread-count", handler.UnreadCount)
		group.PUT("/read-all", handler.MarkAllRead)
		group.GET("/:id", handler.Get)
		group.POST("", handler.Create)
		group.PUT("/:id/read", handler.MarkAsRead)
		group.DELETE("/:id", handler.Delete)

		// Alert rules
		group.GET("/alert-rules", handler.ListAlertRules)
		group.POST("/alert-rules", handler.CreateAlertRule)
		group.PUT("/alert-rules/:id", handler.UpdateAlertRule)
		group.DELETE("/alert-rules/:id", handler.DeleteAlertRule)
	}
}
