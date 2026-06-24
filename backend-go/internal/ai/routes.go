package ai

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers AI routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, hub *realtime.Hub) {
	svc := NewService(db, logger)
	orch := NewOrchestrator(db, logger)
	streamer := NewStreamer(hub, logger)
	h := NewHandler(svc, orch, streamer)

	ai := rg.Group("/ai")
	{
		// Static routes first.
		ai.POST("/chat", h.Chat)
		ai.POST("/run", h.RunAgent)
		ai.GET("/traces", h.ListTraces)
		ai.GET("/actions", h.ListActions)
		ai.GET("/agents", h.Roster)
		ai.GET("/agents/specs", h.AgentSpecs)
		ai.POST("/actions", h.CreateAction)

		// Parameterized routes after.
		ai.GET("/traces/:trace_id", h.GetTrace)
		ai.GET("/actions/:id", h.GetAction)
		ai.POST("/actions/:id/approve", h.ApproveAction)
		ai.POST("/actions/:id/reject", h.RejectAction)
		ai.POST("/actions/:id/execute", h.ExecuteAction)
		ai.POST("/actions/:id/review", h.ReviewAction)
	}
}
