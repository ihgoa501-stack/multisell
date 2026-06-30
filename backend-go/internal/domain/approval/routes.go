package approval

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers approval request routes under the given group.
// Routes are mounted at /approval.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	g := rg.Group("/approval")
	{
		g.GET("", h.ListApprovals)
		g.GET("/:id", h.GetApproval)
		g.POST("", h.CreateApproval)
		g.PUT("/:id/review", h.ReviewApproval)
		g.GET("/my", h.MyPending)
		g.GET("/stats", h.ApprovalStats)
	}
}
