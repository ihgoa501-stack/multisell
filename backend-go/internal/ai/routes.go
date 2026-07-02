package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/realtime"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers AI routes on the given router group.
// moaCoord can be nil; if set, MOA routes are registered.
// cmd can be nil; if set, action execution dispatches through the command handlers.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, hub *realtime.Hub, moaCoord *MOACoordinator, cmd *command.Dispatcher) {
	svc := NewService(db, logger).WithDispatcher(cmd)
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

		// MOA multi-agent orchestration (optional).
		if moaCoord != nil {
			ai.POST("/moa", func(c *gin.Context) {
				var req MOARequest
				if err := c.ShouldBindJSON(&req); err != nil {
					response.Error(c, http.StatusBadRequest, err.Error())
					return
				}
				if req.Mode == "" {
					req.Mode = "suggestion"
				}
				result, err := moaCoord.Run(c.Request.Context(), &req)
				if err != nil {
					response.InternalError(c, err)
					return
				}
				response.Success(c, result)
			})
		}
	}
}
