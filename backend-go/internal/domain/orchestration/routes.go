package orchestration

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers orchestration routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, bus *eventbus.Bus, aiOrch *ai.Orchestrator, logger *zap.Logger) {
	orch := NewPipelineOrchestrator(db, bus, aiOrch, logger.Named("orchestration"))
	h := NewHandler(orch)

	group := rg.Group("/orchestration")
	{
		// Product pipeline endpoints.
		group.GET("/products/:id/pipeline", h.GetPipelineStatus)
		group.POST("/products/:id/pipeline/start", h.StartPipeline)
		group.POST("/products/:id/pipeline/step/:step/retry", h.RetryStep)

		// Config endpoints.
		group.GET("/pipeline/config", h.ListConfigs)
		group.POST("/pipeline/config", h.CreateConfig)
	}

	// Register event bus subscribers for pipeline advancement.
	RegisterSubscribers(bus, orch)
}
