package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers agent routes on the given router group.
// The orchestrator wires agent run endpoints through the AI runtime; pass nil
// to leave them disabled.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, orchestrator *ai.Orchestrator) {
	svc := NewService(db, logger)
	h := NewHandler(svc, orchestrator)

	agents := rg.Group("/agents")
	{
		// Static routes first.
		agents.GET("", h.ListAgents)
		agents.GET("/evolution", h.Evolution)
		agents.GET("/entropy", h.Entropy)
		agents.POST("", h.CreateAgent)

		// Parameterized.
		agents.GET("/:id", h.GetAgent)
		agents.POST("/:id/actions", h.ExecuteAction)
	}
}
