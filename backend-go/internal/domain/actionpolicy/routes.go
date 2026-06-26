package actionpolicy

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)
	p := rg.Group("/policy")
	{
		// PolicyRule CRUD (existing)
		p.GET("/rules", h.ListRules)
		p.GET("/rules/:id", h.GetRule)
		p.POST("/rules", h.CreateRule)
		p.PUT("/rules/:id", h.UpdateRule)
		p.DELETE("/rules/:id", h.DeleteRule)
		p.POST("/rules/:id/toggle", h.HandleToggleRule)
		p.POST("/evaluate", h.Evaluate)

		// ApprovalPolicy CRUD (new)
		p.GET("/approval-policies", h.ListPolicies)
		p.GET("/approval-policies/:id", h.GetPolicy)
		p.POST("/approval-policies", h.CreatePolicy)
		p.PUT("/approval-policies/:id", h.UpdatePolicy)
		p.DELETE("/approval-policies/:id", h.DeletePolicy)

		// ApprovalRequest (new)
		p.GET("/approval-requests", h.ListRequests)
		p.GET("/approval-requests/:id", h.GetRequest)
		p.POST("/approval-requests/:id/review", h.HandleReview)
	}
}
