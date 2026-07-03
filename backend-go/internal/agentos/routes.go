package agentos

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers AgentOS routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	agentos := rg.Group("/agentos")
	{
		agentos.GET("", h.Overview)
		agentos.GET("/status", h.Status)
		agentos.GET("/work-items", h.WorkItems)
		agentos.GET("/autonomy", h.Autonomy)
		agentos.GET("/work-items/:id", h.WorkItemDetail)
		agentos.GET("/agent-timeline", h.AgentTimeline)
		agentos.GET("/failures", h.FailedRuns)
		agentos.GET("/traffic-summary", h.TrafficSummary)
		agentos.GET("/intercepted-actions", h.InterceptedActions)
		agentos.GET("/audit-replay/:correlation_id", h.AuditReplay)
	}
}
