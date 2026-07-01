package approval

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers approval request routes under the given group.
// Routes are mounted at /approval.
// oplogSvc may be nil (audit logging disabled).
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, oplogSvc *operationlog.Service) {
	svc := NewService(db, logger, oplogSvc)
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
