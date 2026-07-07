package approval

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers approval request routes under the given group.
// Routes are mounted at /approval.
// svc must already have WithBus(bus) if event publishing is desired.
func RegisterRoutes(rg *gin.RouterGroup, svc *Service) {
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
