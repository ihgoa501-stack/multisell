package order

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers order routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, approvalSvc *approval.Service) {
	svc := NewService(db, logger)
	h := NewHandler(svc, approvalSvc)

	group := rg.Group("/order")
	{
		group.GET("", h.List)
		group.GET("/summary", h.Summary)
		group.GET("/:id", h.Get)
		group.POST("", h.Create)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.POST("/:id/status", h.UpdateStatus)
	}
}
