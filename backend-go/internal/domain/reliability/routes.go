package reliability

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers reliability routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/reliability")
	{
		group.GET("/agent-status", h.GetAgentStatus)
		group.GET("/llm-cost", h.GetLLMCost)
		group.GET("/failures", h.GetFailures)
		group.POST("/failures/:id/resolve", h.ResolveFailure)
	}
}
